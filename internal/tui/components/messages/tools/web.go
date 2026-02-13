package tools

import (
	"fmt"
	"strings"

	"github.com/pedromelo/poly/internal/tui/layout"
)

type webRenderer struct{}

func init() {
	Register("web_fetch", func(opts *RenderOpts) ToolRenderer {
		return &webRenderer{}
	})
	Register("web_search", func(opts *RenderOpts) ToolRenderer {
		return &webRenderer{}
	})
}

func (w *webRenderer) Render(width int, opts *RenderOpts) string {
	desc := ""
	if url, ok := opts.Args["url"].(string); ok {
		desc = url
	} else if query, ok := opts.Args["query"].(string); ok {
		desc = query
	}

	suffix := ""
	if opts.Status == ToolStatusSuccess && opts.Result != "" {
		resultLines := strings.Split(strings.TrimSpace(opts.Result), "\n")
		count := len(resultLines)
		if count > 0 {
			if opts.Name == "web_search" {
				suffix = fmt.Sprintf("(%d results)", count)
			} else {
				suffix = fmt.Sprintf("(%d lines)", count)
			}
		}
	}

	line := FormatHeader(opts.Status, opts.Name, desc, suffix, width)

	// Show result preview for completed searches
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
