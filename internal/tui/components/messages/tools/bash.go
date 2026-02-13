package tools

import (
	"fmt"
	"strings"

	"github.com/pedromelo/poly/internal/tui/layout"
)

type bashRenderer struct{}

func init() {
	Register("bash", func(opts *RenderOpts) ToolRenderer {
		return &bashRenderer{}
	})
}

func (b *bashRenderer) Render(width int, opts *RenderOpts) string {
	cmd := ""
	if c, ok := opts.Args["command"].(string); ok {
		cmd = c
	}
	// Clean up multiline commands - show first line only
	if idx := strings.Index(cmd, "\n"); idx >= 0 {
		cmd = cmd[:idx] + "~"
	}

	suffix := ""
	if opts.Status == ToolStatusSuccess && opts.Result != "" {
		resultLines := strings.Split(strings.TrimSpace(opts.Result), "\n")
		count := len(resultLines)
		if count == 1 && len(resultLines[0]) < 60 {
			// Short result: show inline
			suffix = resultLines[0]
		} else if count > 0 {
			suffix = fmt.Sprintf("(%d lines)", count)
		}
	}

	line := FormatHeader(opts.Status, "bash", cmd, suffix, width)

	// Show output preview for completed tools (only if multi-line result)
	if opts.Status == ToolStatusSuccess && opts.Result != "" {
		resultLines := strings.Split(strings.TrimSpace(opts.Result), "\n")
		if len(resultLines) > 1 {
			preview := FormatResultPreview(opts.Result, layout.ToolPreviewLines, width)
			if preview != "" {
				line += "\n" + preview
			}
		}
	}

	// Error result
	if opts.IsError && opts.Result != "" {
		errMsg := opts.Result
		// Truncate long errors to first meaningful line
		errLines := strings.Split(strings.TrimSpace(errMsg), "\n")
		if len(errLines) > 0 {
			line += "\n" + FormatError(errLines[0])
			if len(errLines) > 1 {
				preview := FormatResultPreview(errMsg, layout.ToolErrorPreviewLines, width)
				if preview != "" {
					line += "\n" + preview
				}
			}
		}
	}

	return line
}
