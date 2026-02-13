package palette

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// Command represents a command palette entry
type Command struct {
	Name     string
	Shortcut string
}

// SelectionResult is sent when a command is selected
type SelectionResult struct {
	Name string
}

type paletteDialog struct {
	width, height int
	index         int
	filter        string
	commands      []Command
}

// New creates a new command palette dialog
func New(commands []Command) dialogs.DialogModel {
	return &paletteDialog{
		commands: commands,
	}
}

func (p *paletteDialog) ID() dialogs.DialogID { return dialogs.DialogCommandPalette }

func (p *paletteDialog) Init() tea.Cmd { return nil }

func (p *paletteDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		keyStr := msg.String()
		switch {
		case keyStr == "esc":
			return p, func() tea.Msg { return dialogs.CloseDialogMsg{} }

		case keyStr == "enter":
			filtered := p.filteredCommands()
			if p.index < len(filtered) {
				selected := filtered[p.index]
				return p, func() tea.Msg {
					return dialogs.DialogClosedMsg{
						ID:     dialogs.DialogCommandPalette,
						Result: SelectionResult{Name: selected.Name},
					}
				}
			}
			return p, nil

		case keyStr == "up":
			if p.index > 0 {
				p.index--
			}
			return p, nil

		case keyStr == "down":
			filtered := p.filteredCommands()
			if p.index < len(filtered)-1 {
				p.index++
			}
			return p, nil

		case keyStr == "backspace":
			if len(p.filter) > 0 {
				p.filter = p.filter[:len(p.filter)-1]
				p.index = 0
			}
			return p, nil

		default:
			if len(keyStr) == 1 {
				p.filter += keyStr
				p.index = 0
			}
			return p, nil
		}
	}
	return p, nil
}

func (p *paletteDialog) View() string {
	titleStr := core.Title("Commands", 40, styles.Mauve, styles.Surface2)

	var content strings.Builder
	content.WriteString(titleStr + "\n\n")

	// Filter input
	filterStyle := lipgloss.NewStyle().
		Background(styles.Surface1).
		Foreground(styles.Text).
		Padding(0, 1).
		Width(38)

	filterDisplay := p.filter
	if filterDisplay == "" {
		filterDisplay = lipgloss.NewStyle().Foreground(styles.Overlay0).Italic(true).Render("Type to filter...")
	}
	content.WriteString(filterStyle.Render("> "+filterDisplay) + "\n\n")

	filtered := p.filteredCommands()

	for i, cmd := range filtered {
		isSelected := i == p.index

		selector := "  "
		if isSelected {
			selector = lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render("> ")
		}

		nameStyle := lipgloss.NewStyle().Foreground(styles.Text)
		shortcutStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)

		name := nameStyle.Render(cmd.Name)
		shortcut := ""
		if cmd.Shortcut != "" {
			shortcut = shortcutStyle.Render(cmd.Shortcut)
		}

		nameWidth := 24
		row := selector + lipgloss.NewStyle().Width(nameWidth).Render(name) + shortcut

		if isSelected {
			row = lipgloss.NewStyle().Background(styles.Surface1).Width(40).Render(row)
		}

		content.WriteString(row + "\n")
	}

	content.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)
	content.WriteString(hintStyle.Render("  up/dn choose · enter confirm · esc close"))

	dialog := lipgloss.NewStyle().
		Background(styles.Surface0).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Padding(1, 2).
		Width(46).
		Render(content.String())

	return lipgloss.Place(
		p.width, p.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.Base)),
	)
}

func (p *paletteDialog) filteredCommands() []Command {
	if p.filter == "" {
		return p.commands
	}
	filter := strings.ToLower(p.filter)
	var result []Command
	for _, cmd := range p.commands {
		if strings.Contains(strings.ToLower(cmd.Name), filter) {
			result = append(result, cmd)
		}
	}
	return result
}

func (p *paletteDialog) SetSize(width, height int) {
	p.width = width
	p.height = height
}
