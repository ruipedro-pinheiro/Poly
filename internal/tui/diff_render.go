package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
)

// renderContentWithHighlights applies diff highlighting and tool output styling
func renderContentWithHighlights(content string, width int) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inDiff := false

	for i, line := range lines {
		styled := styleLine(line, &inDiff)
		result.WriteString(styled)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Render(result.String())
}

// styleLine applies appropriate styling to a single line
func styleLine(line string, inDiff *bool) string {
	// Detect diff header (start of diff)
	if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "+++ b/") ||
		strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
		*inDiff = true
		return lipgloss.NewStyle().Foreground(theme.Blue).Bold(true).Render(line)
	}

	// Diff hunk header (also starts a diff if we somehow missed the header)
	if strings.HasPrefix(line, "@@ ") {
		*inDiff = true
		return lipgloss.NewStyle().Foreground(theme.Mauve).Render(line)
	}

	if *inDiff {
		// Diff added line
		if strings.HasPrefix(line, "+") {
			return lipgloss.NewStyle().Foreground(theme.Green).Render(line)
		}

		// Diff removed line
		if strings.HasPrefix(line, "-") {
			return lipgloss.NewStyle().Foreground(theme.Red).Render(line)
		}

		// Diff context line (starts with space)
		if strings.HasPrefix(line, " ") {
			return lipgloss.NewStyle().Foreground(theme.Overlay1).Render(line)
		}

		// "No newline at end of file" marker
		if strings.HasPrefix(line, `\`) {
			return lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true).Render(line)
		}

		// Empty line within diff = context separator, keep inDiff
		if line == "" {
			return ""
		}

		// Any other non-diff line = end of diff block
		*inDiff = false
	}

	// Tool status indicators
	if strings.HasPrefix(line, "Edited ") || strings.HasPrefix(line, "Created ") || strings.HasPrefix(line, "Updated ") {
		return lipgloss.NewStyle().Foreground(theme.Green).Render(line)
	}

	if strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "ERROR:") {
		return lipgloss.NewStyle().Foreground(theme.Red).Render(line)
	}

	// Default: standard text color
	return lipgloss.NewStyle().Foreground(theme.Text).Render(line)
}
