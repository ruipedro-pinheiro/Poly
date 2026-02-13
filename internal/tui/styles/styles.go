package styles

import "charm.land/lipgloss/v2"

// BorderStyles defines the unified border system for the entire TUI.
// Design choice: RoundedBorder everywhere for a clean, modern look.
// Messages now use RoundedBorder (full box) for a chat-bubble design.
type BorderStyles struct {
	Message  lipgloss.Border // chat messages: RoundedBorder full box
	Dialog   lipgloss.Border // dialogs/overlays: RoundedBorder
	Panel    lipgloss.Border // sidebar panels: NormalBorder (left only)
	Input    lipgloss.Border // editor/input: RoundedBorder
	ToolCall lipgloss.Border // tool call blocks: RoundedBorder
}

// Borders is the canonical border set. Reference this instead of
// calling lipgloss.*Border() directly in components.
var Borders = BorderStyles{
	Message:  lipgloss.RoundedBorder(),
	Dialog:   lipgloss.RoundedBorder(),
	Panel:    lipgloss.NormalBorder(),
	Input:    lipgloss.RoundedBorder(),
	ToolCall: lipgloss.RoundedBorder(),
}

// ChatStyles holds styles for chat message rendering
type ChatStyles struct {
	UserBorder       lipgloss.Style
	AssistantBorder  lipgloss.Style
	UserLabel        lipgloss.Style
	AssistantLabel   lipgloss.Style
	ThinkingBg       lipgloss.Style
	ToolCallFocused  lipgloss.Style
	ToolCallBlurred  lipgloss.Style
}

// DialogStyles holds styles for dialog rendering
type DialogStyles struct {
	Border  lipgloss.Style
	Title   lipgloss.Style
	Content lipgloss.Style
}

// Styles is the master style collection for the TUI
type Styles struct {
	// Base
	Base     lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Text     lipgloss.Style
	Muted    lipgloss.Style
	Subtle   lipgloss.Style

	// Semantic
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// Chat
	Chat ChatStyles

	// Dialogs
	Dialog DialogStyles
}

// DefaultStyles returns the default Catppuccin Mocha style set
func DefaultStyles() Styles {
	return Styles{
		Base:     lipgloss.NewStyle().Background(Base),
		Title:    lipgloss.NewStyle().Foreground(Mauve).Bold(true),
		Subtitle: lipgloss.NewStyle().Foreground(Overlay1),
		Text:     lipgloss.NewStyle().Foreground(Text),
		Muted:    lipgloss.NewStyle().Foreground(Overlay0),
		Subtle:   lipgloss.NewStyle().Foreground(Surface2),

		Success: lipgloss.NewStyle().Foreground(Green),
		Error:   lipgloss.NewStyle().Foreground(Red),
		Warning: lipgloss.NewStyle().Foreground(Yellow),
		Info:    lipgloss.NewStyle().Foreground(Blue),

		Chat: ChatStyles{
			UserBorder: lipgloss.NewStyle().
				BorderStyle(Borders.Message).
				BorderForeground(Mauve).
				Padding(0, 1),
			AssistantBorder: lipgloss.NewStyle().
				BorderStyle(Borders.Message).
				Padding(0, 1),
			UserLabel: lipgloss.NewStyle().
				Foreground(Mauve).
				Bold(true),
			AssistantLabel: lipgloss.NewStyle().
				Bold(true),
			ThinkingBg: lipgloss.NewStyle().
				Background(Surface0).
				Foreground(Overlay1).
				Italic(true),
			ToolCallFocused: lipgloss.NewStyle().
				BorderStyle(Borders.ToolCall).
				BorderForeground(Mauve),
			ToolCallBlurred: lipgloss.NewStyle().
				BorderStyle(Borders.ToolCall).
				BorderForeground(Surface1),
		},

		Dialog: DialogStyles{
			Border: lipgloss.NewStyle().
				BorderStyle(Borders.Dialog).
				BorderForeground(Mauve).
				Background(Surface0).
				Padding(1, 2),
			Title: lipgloss.NewStyle().
				Foreground(Mauve).
				Bold(true),
			Content: lipgloss.NewStyle().
				Foreground(Text),
		},
	}
}
