package tools

import (
	"fmt"
	"strings"
)

type readRenderer struct{}

func init() {
	Register("read_file", func(opts *RenderOpts) ToolRenderer {
		return &readRenderer{}
	})
	Register("glob", func(opts *RenderOpts) ToolRenderer {
		return &readRenderer{}
	})
	Register("grep", func(opts *RenderOpts) ToolRenderer {
		return &readRenderer{}
	})
}

func (r *readRenderer) Render(width int, opts *RenderOpts) string {
	arg := ""
	switch opts.Name {
	case "read_file":
		if p, ok := opts.Args["file_path"].(string); ok {
			arg = ShortenPath(p)
		} else if p, ok := opts.Args["path"].(string); ok {
			arg = ShortenPath(p)
		}
	case "glob":
		if p, ok := opts.Args["pattern"].(string); ok {
			arg = p
		}
	case "grep":
		if p, ok := opts.Args["pattern"].(string); ok {
			arg = p
		}
	}

	suffix := ""
	if opts.Status == ToolStatusSuccess && opts.Result != "" {
		resultLines := strings.Split(strings.TrimSpace(opts.Result), "\n")
		count := len(resultLines)
		if count > 0 {
			switch opts.Name {
			case "grep":
				suffix = fmt.Sprintf("(%d matches)", count)
			case "glob":
				suffix = fmt.Sprintf("(%d files)", count)
			default:
				suffix = fmt.Sprintf("(%d lines)", count)
			}
		}
	}

	line := FormatHeader(opts.Status, opts.Name, arg, suffix, width)

	if opts.IsError && opts.Result != "" {
		line += "\n" + FormatError(opts.Result)
	}

	return line
}
