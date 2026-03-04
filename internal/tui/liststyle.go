package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
)

func listCursor(selected bool) string {
	if selected {
		return core.IconArrow + " "
	}
	return "  "
}

func renderListRow(innerWidth int, row string, selected bool) string {
	row = strings.TrimRight(row, " ")
	base := lipgloss.NewStyle().
		Width(innerWidth).
		MaxWidth(innerWidth)
	if selected {
		base = base.
			Background(theme.Surface0).
			Foreground(theme.Lavender).
			Bold(true)
	}
	return base.Render(row)
}

func renderDialogHints(innerWidth int, lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(false).
		BorderForeground(theme.Surface1).
		Foreground(theme.Overlay0).
		Width(innerWidth).
		Render(content)
}

func truncateToWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 2 {
		return strings.Repeat(".", max)
	}

	runes := []rune(s)
	for i := len(runes); i >= 0; i-- {
		candidate := string(runes[:i]) + ".."
		if lipgloss.Width(candidate) <= max {
			return candidate
		}
	}
	return strings.Repeat(".", max)
}
