package help

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

type helpDialog struct {
	width, height int
}

// New creates a new help dialog
func New() dialogs.DialogModel {
	return &helpDialog{}
}

func (h *helpDialog) ID() dialogs.DialogID { return dialogs.DialogHelp }

func (h *helpDialog) Init() tea.Cmd { return nil }

func (h *helpDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if msg.String() == "esc" || msg.String() == "ctrl+h" {
			return h, func() tea.Msg { return dialogs.CloseDialogMsg{} }
		}
	}
	return h, nil
}

func (h *helpDialog) View() string {
	titleStr := core.Title("Help", 44, styles.Mauve, styles.Surface2)

	sectionStyle := lipgloss.NewStyle().
		Foreground(styles.Subtext0).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Mauve).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(styles.Text)

	var content strings.Builder
	content.WriteString(titleStr + "\n\n")

	// Navigation
	content.WriteString(sectionStyle.Render("  NAVIGATION") + "\n")
	navKeys := [][2]string{
		{"Ctrl+D", "Control Room (connect providers)"},
		{"Ctrl+O", "Model Picker"},
		{"Ctrl+K", "Command Palette"},
		{"Ctrl+H", "This help"},
	}
	for _, k := range navKeys {
		content.WriteString("  " + keyStyle.Width(10).Render(k[0]) + descStyle.Render(k[1]) + "\n")
	}
	content.WriteString("\n")

	// Chat
	content.WriteString(sectionStyle.Render("  CHAT") + "\n")
	chatKeys := [][2]string{
		{"Enter", "Send message"},
		{"Esc", "Cancel streaming / Close dialog"},
		{"Ctrl+L", "Clear chat"},
		{"Ctrl+N", "New session"},
	}
	for _, k := range chatKeys {
		content.WriteString("  " + keyStyle.Width(10).Render(k[0]) + descStyle.Render(k[1]) + "\n")
	}
	content.WriteString("\n")

	// Mentions
	content.WriteString(sectionStyle.Render("  MENTIONS") + "\n")
	mentions := [][2]string{
		{"@claude", "Send to Claude"},
		{"@gpt", "Send to GPT"},
		{"@gemini", "Send to Gemini"},
		{"@grok", "Send to Grok"},
		{"@all", "Send to all providers"},
	}
	for _, k := range mentions {
		content.WriteString("  " + keyStyle.Width(10).Render(k[0]) + descStyle.Render(k[1]) + "\n")
	}
	content.WriteString("\n")

	// Commands
	content.WriteString(sectionStyle.Render("  COMMANDS") + "\n")
	commands := [][2]string{
		{"/clear", "Clear chat history"},
		{"/model", "Change model variant"},
		{"/think", "Toggle thinking mode"},
		{"/sidebar", "Toggle sidebar"},
	}
	for _, k := range commands {
		content.WriteString("  " + keyStyle.Width(10).Render(k[0]) + descStyle.Render(k[1]) + "\n")
	}
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(styles.Overlay0).Render("  Press Esc to close"))

	dialog := lipgloss.NewStyle().
		Background(styles.Surface0).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Surface2).
		Padding(1, 2).
		Width(50).
		Render(content.String())

	return lipgloss.Place(
		h.width, h.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.Base)),
	)
}

func (h *helpDialog) SetSize(width, height int) {
	h.width = width
	h.height = height
}
