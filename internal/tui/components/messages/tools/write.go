package tools

import (
	"fmt"
	"strings"
)

type writeRenderer struct{}

func init() {
	Register("write_file", func(opts *RenderOpts) ToolRenderer {
		return &writeRenderer{}
	})
}

func (w *writeRenderer) Render(width int, opts *RenderOpts) string {
	path := ""
	if p, ok := opts.Args["file_path"].(string); ok {
		path = ShortenPath(p)
	} else if p, ok := opts.Args["path"].(string); ok {
		path = ShortenPath(p)
	}

	suffix := ""
	if content, ok := opts.Args["content"].(string); ok {
		lineCount := strings.Count(content, "\n") + 1
		suffix = fmt.Sprintf("(%d lines)", lineCount)
	}

	line := FormatHeader(opts.Status, "write_file", path, suffix, width)

	if opts.IsError && opts.Result != "" {
		line += "\n" + FormatError(opts.Result)
	}

	return line
}
