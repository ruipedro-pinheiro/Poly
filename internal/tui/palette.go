package tui

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
)

// renderCommandPalette renders the command palette overlay
func (m Model) renderCommandPalette() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Bold(true)

	title := titleStyle.Render("Commands")

	w := dialogWidth(46, m.width, 36)
	innerWidth := w - 6 // padding + border

	var content strings.Builder
	content.WriteString(title + "\n\n")

	// Filter input (bordered style)
	filterDisplay := m.paletteFilter
	if filterDisplay == "" {
		filterDisplay = lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true).Render("Type to filter...")
	}
	filterBox := lipgloss.NewStyle().
		Foreground(theme.Text).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Mauve).
		Padding(0, 1).
		Width(innerWidth - 2).
		Render("> " + filterDisplay)
	content.WriteString(filterBox + "\n\n")

	// Filter commands
	filtered := m.filteredPaletteCommands()

	// Render commands
	for i, cmd := range filtered {
		isSelected := i == m.paletteIndex

		var row strings.Builder

		// Selection cursor
		if isSelected {
			row.WriteString(lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render(" > "))
		} else {
			row.WriteString("   ")
		}

		// Command name
		nameStyle := lipgloss.NewStyle().Foreground(theme.Text)
		row.WriteString(nameStyle.Width(20).Render(cmd.Name))

		// Shortcut (right side)
		if cmd.Shortcut != "" {
			shortcutStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
			row.WriteString(shortcutStyle.Render(cmd.Shortcut))
		}

		// Row style: selected gets left border accent
		rowStr := row.String()
		if isSelected {
			rowStr = lipgloss.NewStyle().
				BorderStyle(lipgloss.ThickBorder()).
				BorderLeft(true).
				BorderRight(false).
				BorderTop(false).
				BorderBottom(false).
				BorderForeground(theme.Mauve).
				Width(innerWidth).
				Render(rowStr)
		}

		content.WriteString(rowStr + "\n")
	}

	content.WriteString("\n")
	hintKey := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintDesc := lipgloss.NewStyle().Foreground(theme.Overlay0)
	content.WriteString(hintKey.Render("↑↓") + hintDesc.Render(" choose · "))
	content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" confirm · "))
	content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" close"))

	dialog := dialogStyle(w).Render(content.String())
	return placeDialog(dialog, m.width, m.height)
}

// fuzzyMatch checks if query is a subsequence of target and returns a score.
// Higher score = better match. Consecutive matches and prefix matches score higher.
func fuzzyMatch(query, target string) (bool, int) {
	q := strings.ToLower(query)
	t := strings.ToLower(target)

	qi := 0
	score := 0
	prevMatch := -1

	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] == q[qi] {
			score++
			// Bonus for consecutive matches
			if prevMatch == ti-1 {
				score += 3
			}
			// Bonus for matching at start of target or after space
			if ti == 0 || (ti > 0 && t[ti-1] == ' ') {
				score += 5
			}
			prevMatch = ti
			qi++
		}
	}
	return qi == len(q), score
}

type scoredCommand struct {
	cmd   CommandEntry
	score int
}

// filteredPaletteCommands returns commands matching the current filter using fuzzy matching
func (m Model) filteredPaletteCommands() []CommandEntry {
	if m.paletteFilter == "" {
		return m.paletteCommands
	}

	var scored []scoredCommand
	for _, cmd := range m.paletteCommands {
		if ok, score := fuzzyMatch(m.paletteFilter, cmd.Name); ok {
			scored = append(scored, scoredCommand{cmd: cmd, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]CommandEntry, len(scored))
	for i, s := range scored {
		result[i] = s.cmd
	}
	return result
}
