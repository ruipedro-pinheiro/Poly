package tui

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
)

// renderCommandPalette renders the command palette overlay.
// It dynamically lists ALL commands from the CommandRegistry, not just the
// hardcoded paletteCommands list.
func (m Model) renderCommandPalette() string {
	w := dialogWidth(46, m.width, 36)
	innerWidth := w - 6 // padding + border

	var content strings.Builder

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

	// Build command list from registry
	filtered := m.filteredRegistryCommands()

	// Compute visible range for scrolling
	maxVisible := m.height - 14 // leave room for frame + filter + hints
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if m.paletteIndex >= maxVisible {
		startIdx = m.paletteIndex - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	// Scroll indicator top
	if startIdx > 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true).
			Render("   ... "+strings.Repeat("^", 3)) + "\n")
	}

	// Render commands
	for i := startIdx; i < endIdx; i++ {
		cmd := filtered[i]
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
		row.WriteString(nameStyle.Width(20).Render("/" + cmd.Name))

		// Category (right side)
		if cmd.Category != "" {
			catStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
			row.WriteString(catStyle.Render(cmd.Category))
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

	// Scroll indicator bottom
	if endIdx < len(filtered) {
		content.WriteString(lipgloss.NewStyle().Foreground(theme.Overlay0).Italic(true).
			Render("   ... "+strings.Repeat("v", 3)) + "\n")
	}

	content.WriteString("\n")
	hintKey := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintDesc := lipgloss.NewStyle().Foreground(theme.Overlay0)
	content.WriteString(hintKey.Render("↑↓") + hintDesc.Render(" choose · "))
	content.WriteString(hintKey.Render("Enter") + hintDesc.Render(" confirm · "))
	content.WriteString(hintKey.Render("Esc") + hintDesc.Render(" close"))

	return m.renderDialogFrame("Commands", content.String(), 46)
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

// filteredPaletteCommands returns CommandEntry items matching the current filter.
// This bridges the old palette system (used by handlePaletteKey in update_keys.go)
// to the new registry-based command list.
func (m Model) filteredPaletteCommands() []CommandEntry {
	cmds := m.filteredRegistryCommands()
	entries := make([]CommandEntry, len(cmds))
	for i, cmd := range cmds {
		c := cmd // capture
		entries[i] = CommandEntry{
			Name: "/" + c.Name,
			Action: func(m *Model) {
				c.Handler(m, nil)
			},
		}
	}
	return entries
}

type scoredRegistryCmd struct {
	cmd   *Command
	score int
}

// filteredRegistryCommands returns commands from the registry matching the
// current filter using fuzzy matching. When no filter is active, returns all
// commands from the registry in registration order.
func (m Model) filteredRegistryCommands() []*Command {
	if m.commands == nil {
		return nil
	}

	// Get all commands in registration order
	_, catMap := m.commands.ByCategory()
	var allCmds []*Command
	// Use ordered list from registry
	catOrder, _ := m.commands.ByCategory()
	for _, cat := range catOrder {
		allCmds = append(allCmds, catMap[cat]...)
	}

	if m.paletteFilter == "" {
		return allCmds
	}

	var scored []scoredRegistryCmd
	for _, cmd := range allCmds {
		// Match against name, description, and category
		nameOk, nameScore := fuzzyMatch(m.paletteFilter, cmd.Name)
		descOk, descScore := fuzzyMatch(m.paletteFilter, cmd.Description)
		catOk, catScore := fuzzyMatch(m.paletteFilter, cmd.Category)

		if nameOk || descOk || catOk {
			bestScore := nameScore
			if descScore > bestScore {
				bestScore = descScore
			}
			if catScore > bestScore {
				bestScore = catScore
			}
			// Name matches get a big bonus
			if nameOk {
				bestScore += 10
			}
			scored = append(scored, scoredRegistryCmd{cmd: cmd, score: bestScore})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]*Command, len(scored))
	for i, s := range scored {
		result[i] = s.cmd
	}
	return result
}
