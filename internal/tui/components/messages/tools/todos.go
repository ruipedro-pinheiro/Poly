package tools

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
)

type todosRenderer struct{}

func init() {
	Register("todos", func(opts *RenderOpts) ToolRenderer {
		return &todosRenderer{}
	})
}

func (t *todosRenderer) Render(width int, opts *RenderOpts) string {
	// Count todos by status from the args
	pending, active, done := 0, 0, 0
	if todosI, ok := opts.Args["todos"].([]interface{}); ok {
		for _, ti := range todosI {
			if todo, ok := ti.(map[string]interface{}); ok {
				switch todo["status"] {
				case "pending":
					pending++
				case "in_progress":
					active++
				case "completed":
					done++
				}
			}
		}
	}

	// Build compact description: "2 pending / 1 active / 3 done"
	var parts []string
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if active > 0 {
		parts = append(parts, fmt.Sprintf("%d active", active))
	}
	if done > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Overlay0).Render(fmt.Sprintf("%d done", done)))
	}

	desc := ""
	for i, p := range parts {
		if i > 0 {
			desc += " / "
		}
		desc += p
	}
	if desc == "" {
		desc = "empty"
	}

	return FormatSummary(opts.Status, "todos", desc)
}
