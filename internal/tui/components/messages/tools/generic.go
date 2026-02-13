package tools

import (
	"fmt"

	"github.com/pedromelo/poly/internal/tui/layout"
)

type genericRenderer struct{}

func init() {
	Register("_generic", func(opts *RenderOpts) ToolRenderer {
		return &genericRenderer{}
	})
}

func (g *genericRenderer) Render(width int, opts *RenderOpts) string {
	desc := ""
	for _, key := range []string{"file_path", "path", "command", "url", "query", "pattern", "file"} {
		if v, ok := opts.Args[key].(string); ok {
			desc = v
			break
		}
	}

	if desc == "" && len(opts.Args) > 0 {
		for _, v := range opts.Args {
			if s, ok := v.(string); ok {
				desc = s
				break
			}
		}
		if desc == "" {
			desc = fmt.Sprintf("%d args", len(opts.Args))
		}
	}

	line := FormatHeader(opts.Status, opts.Name, desc, "", width)

	if opts.Status == ToolStatusSuccess && opts.Result != "" {
		preview := FormatResultPreview(opts.Result, layout.ToolPreviewLines, width)
		if preview != "" {
			line += "\n" + preview
		}
	}

	if opts.IsError && opts.Result != "" {
		line += "\n" + FormatError(opts.Result)
	}

	return line
}
