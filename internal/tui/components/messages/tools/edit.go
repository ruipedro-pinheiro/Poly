package tools

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/styles"
)

type editRenderer struct{}

func init() {
	Register("edit_file", func(opts *RenderOpts) ToolRenderer {
		return &editRenderer{}
	})
}

func (e *editRenderer) Render(width int, opts *RenderOpts) string {
	path := ""
	if p, ok := opts.Args["file_path"].(string); ok {
		path = ShortenPath(p)
	} else if p, ok := opts.Args["path"].(string); ok {
		path = ShortenPath(p)
	}

	// Count additions/deletions from old_string/new_string args
	oldStr, _ := opts.Args["old_string"].(string)
	newStr, _ := opts.Args["new_string"].(string)
	oldLines := 0
	newLines := 0
	if oldStr != "" {
		oldLines = strings.Count(oldStr, "\n") + 1
	}
	if newStr != "" {
		newLines = strings.Count(newStr, "\n") + 1
	}

	suffix := ""
	if oldLines > 0 || newLines > 0 {
		addStyle := lipgloss.NewStyle().Foreground(styles.Green)
		delStyle := lipgloss.NewStyle().Foreground(styles.Red)
		parts := []string{}
		if newLines > 0 {
			parts = append(parts, addStyle.Render(fmt.Sprintf("+%d", newLines)))
		}
		if oldLines > 0 {
			parts = append(parts, delStyle.Render(fmt.Sprintf("-%d", oldLines)))
		}
		suffix = "(" + strings.Join(parts, " ") + ")"
	}

	line := FormatHeader(opts.Status, "edit_file", path, suffix, width)

	// Show diff preview for completed edits
	if opts.Status == ToolStatusSuccess && oldStr != "" && newStr != "" {
		diffPreview := formatDiffPreview(oldStr, newStr, width)
		if diffPreview != "" {
			line += "\n" + diffPreview
		}
	}

	if opts.IsError && opts.Result != "" {
		line += "\n" + FormatError(opts.Result)
	}

	return line
}

// formatDiffPreview shows a mini diff with - and + lines
func formatDiffPreview(oldStr, newStr string, width int) string {
	pipeStyle := lipgloss.NewStyle().Foreground(styles.Surface2)
	delStyle := lipgloss.NewStyle().Foreground(styles.Red)
	addStyle := lipgloss.NewStyle().Foreground(styles.Green)

	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")

	maxLen := width - 8
	if maxLen < 20 {
		maxLen = 20
	}

	var output []string
	// Show max 2 deleted lines
	for i, l := range oldLines {
		if i >= 2 {
			break
		}
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if len(l) > maxLen {
			l = l[:maxLen-1] + "~"
		}
		output = append(output, "  "+pipeStyle.Render("┃ ")+delStyle.Render("- "+l))
	}
	// Show max 2 added lines
	for i, l := range newLines {
		if i >= 2 {
			break
		}
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if len(l) > maxLen {
			l = l[:maxLen-1] + "~"
		}
		output = append(output, "  "+pipeStyle.Render("┃ ")+addStyle.Render("+ "+l))
	}

	if len(output) == 0 {
		return ""
	}
	return strings.Join(output, "\n")
}
