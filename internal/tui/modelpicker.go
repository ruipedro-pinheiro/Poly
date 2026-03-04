package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// renderModelPicker renders the enhanced model picker grouped by provider
func (m Model) renderModelPicker() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Bold(true)

	title := titleStyle.Render("Select Model")

	w := dialogWidth(54, m.width, 40)
	innerWidth := w - 6 // padding + border

	var content strings.Builder
	content.WriteString(title + "\n\n")

	// Filter input (bordered style, only visible when filter active)
	if m.modelPickerFilter != "" {
		filterBox := lipgloss.NewStyle().
			Foreground(theme.Text).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(theme.Surface2).
			Padding(0, 1).
			Width(innerWidth - 2).
			Render("> " + m.modelPickerFilter)
		content.WriteString(filterBox + "\n\n")
	}

	// Get filtered models
	models := m.filteredModelPickerModels()

	// Recently used section
	if len(m.recentModels) > 0 && m.modelPickerFilter == "" {
		sectionStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Bold(true)
		content.WriteString(sectionStyle.Render("  Recently Used") + "\n")

		for _, model := range m.recentModels {
			content.WriteString(m.renderModelRow(model, -1, false, innerWidth))
		}
		content.WriteString("\n")
	}

	// Group by provider
	currentProvider := ""
	globalIndex := 0

	for _, model := range models {
		// Provider group header
		if model.provider != currentProvider {
			currentProvider = model.provider
			providerHeaderStyle := lipgloss.NewStyle().
				Foreground(theme.ProviderColor(model.provider)).
				Bold(true)
			content.WriteString("\n" + providerHeaderStyle.Render("  "+cases.Title(language.English).String(model.provider)) + "\n")
		}

		isCurrent := m.defaultProvider == model.provider && m.modelVariant == model.variant
		content.WriteString(m.renderModelRow(model, globalIndex, isCurrent, innerWidth))
		globalIndex++
	}

	content.WriteString("\n")
	content.WriteString(renderDialogHints(innerWidth,
		"↑↓ navigate · Enter confirm · type filter · Esc cancel",
	))

	dialog := dialogStyle(w).Render(content.String())
	return placeDialog(dialog, m.width, m.height)
}

// renderModelRow renders a single model option row
func (m Model) renderModelRow(model modelOption, index int, isCurrent bool, innerWidth int) string {
	isSelected := index == m.modelPickerIndex && index >= 0

	var row strings.Builder

	// Selection cursor
	cursor := listCursor(isSelected)
	row.WriteString(cursor)

	// Model name (variant part)
	nameStyle := lipgloss.NewStyle().Foreground(theme.Text)
	var displayName string
	if len(model.display) > len(model.provider)+1 {
		displayName = model.display[len(model.provider)+1:]
	} else {
		displayName = model.display
	}
	displayName = truncateToWidth(displayName, innerWidth-16)

	// Provider badge
	providerBadge := lipgloss.NewStyle().
		Foreground(theme.ProviderColor(model.provider)).
		Render(model.provider)

	// Current model marker
	currentMark := ""
	if isCurrent {
		currentMark = lipgloss.NewStyle().Foreground(theme.Green).Render(" " + core.IconCheck)
	}

	nameWidth := innerWidth - 16
	if nameWidth < 14 {
		nameWidth = 14
	}
	row.WriteString(nameStyle.Width(nameWidth).Render(displayName))
	row.WriteString(" ")
	row.WriteString(lipgloss.NewStyle().Width(10).AlignHorizontal(lipgloss.Right).Render(providerBadge + currentMark))

	// Row style: selected gets left border accent
	return renderListRow(innerWidth, row.String(), isSelected) + "\n"
}

// filteredModelPickerModels returns models matching the current filter
func (m Model) filteredModelPickerModels() []modelOption {
	if m.modelPickerFilter == "" {
		return m.modelPickerModels
	}

	filter := strings.ToLower(m.modelPickerFilter)
	var result []modelOption
	for _, model := range m.modelPickerModels {
		if strings.Contains(strings.ToLower(model.display), filter) ||
			strings.Contains(strings.ToLower(model.provider), filter) {
			result = append(result, model)
		}
	}
	return result
}
