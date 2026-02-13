package modelpicker

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// ModelOption represents a selectable model
type ModelOption struct {
	Provider string
	Variant  string
	Display  string
}

// SelectionResult is sent when a model is selected
type SelectionResult struct {
	Provider string
	Variant  string
	Display  string
}

type pickerDialog struct {
	width, height int
	index         int
	filter        string
	models        []ModelOption
	recentModels  []ModelOption
	currentProv   string
	currentVar    string
	providerColor func(string) color.Color
}

// New creates a new model picker dialog
func New(models []ModelOption, recentModels []ModelOption, currentProvider, currentVariant string, providerColor func(string) color.Color) dialogs.DialogModel {
	return &pickerDialog{
		models:        models,
		recentModels:  recentModels,
		currentProv:   currentProvider,
		currentVar:    currentVariant,
		providerColor: providerColor,
	}
}

func (p *pickerDialog) ID() dialogs.DialogID { return dialogs.DialogModelPicker }

func (p *pickerDialog) Init() tea.Cmd { return nil }

func (p *pickerDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		keyStr := msg.String()
		switch {
		case keyStr == "esc":
			return p, func() tea.Msg { return dialogs.CloseDialogMsg{} }

		case keyStr == "enter":
			filtered := p.filteredModels()
			if p.index < len(filtered) {
				selected := filtered[p.index]
				return p, func() tea.Msg {
					return dialogs.DialogClosedMsg{
						ID: dialogs.DialogModelPicker,
						Result: SelectionResult{
							Provider: selected.Provider,
							Variant:  selected.Variant,
							Display:  selected.Display,
						},
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
			filtered := p.filteredModels()
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

func (p *pickerDialog) View() string {
	titleStr := core.Title("Select Model", 48, styles.Mauve, styles.Surface2)

	var content strings.Builder
	content.WriteString(titleStr + "\n\n")

	// Filter input
	if p.filter != "" {
		filterStyle := lipgloss.NewStyle().
			Background(styles.Surface1).
			Foreground(styles.Text).
			Padding(0, 1).
			Width(38)
		content.WriteString(filterStyle.Render("> "+p.filter) + "\n\n")
	}

	models := p.filteredModels()

	// Recently used
	if len(p.recentModels) > 0 && p.filter == "" {
		sectionStyle := lipgloss.NewStyle().
			Foreground(styles.Overlay0).
			Bold(true)
		content.WriteString(sectionStyle.Render("  Recently Used") + "\n")
		for _, model := range p.recentModels {
			content.WriteString(p.renderRow(model, -1, false))
		}
		content.WriteString("\n")
	}

	// Group by provider
	currentProvider := ""
	globalIndex := 0
	for _, model := range models {
		if model.Provider != currentProvider {
			currentProvider = model.Provider
			color := styles.Overlay1
			if p.providerColor != nil {
				color = p.providerColor(model.Provider)
			}
			providerStyle := lipgloss.NewStyle().
				Foreground(color).
				Bold(true)
			content.WriteString("\n" + providerStyle.Render("  "+strings.Title(model.Provider)) + "\n")
		}

		isCurrent := p.currentProv == model.Provider && p.currentVar == model.Variant
		content.WriteString(p.renderRow(model, globalIndex, isCurrent))
		globalIndex++
	}

	content.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)
	content.WriteString(hintStyle.Render("  up/dn select · enter confirm · type to filter · esc cancel"))

	dialog := lipgloss.NewStyle().
		Background(styles.Surface0).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Padding(1, 2).
		Width(54).
		Render(content.String())

	return lipgloss.Place(
		p.width, p.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.Base)),
	)
}

func (p *pickerDialog) renderRow(model ModelOption, index int, isCurrent bool) string {
	isSelected := index == p.index && index >= 0

	selector := "    "
	if isSelected {
		selector = lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render("  > ")
	}

	nameStyle := lipgloss.NewStyle().Foreground(styles.Text)
	displayName := model.Display
	if len(displayName) > len(model.Provider)+1 {
		displayName = displayName[len(model.Provider)+1:]
	}

	color := styles.Overlay1
	if p.providerColor != nil {
		color = p.providerColor(model.Provider)
	}
	providerBadge := lipgloss.NewStyle().
		Foreground(color).
		Render(model.Provider)

	currentMark := ""
	if isCurrent {
		currentMark = lipgloss.NewStyle().Foreground(styles.Green).Render(" " + core.IconPending)
	}

	nameWidth := 28
	row := selector + nameStyle.Width(nameWidth).Render(displayName) + providerBadge + currentMark

	if isSelected {
		row = lipgloss.NewStyle().Background(styles.Surface1).Width(50).Render(row)
	}

	return row + "\n"
}

func (p *pickerDialog) filteredModels() []ModelOption {
	if p.filter == "" {
		return p.models
	}
	filter := strings.ToLower(p.filter)
	var result []ModelOption
	for _, m := range p.models {
		if strings.Contains(strings.ToLower(m.Display), filter) ||
			strings.Contains(strings.ToLower(m.Provider), filter) {
			result = append(result, m)
		}
	}
	return result
}

func (p *pickerDialog) SetSize(width, height int) {
	p.width = width
	p.height = height
}
