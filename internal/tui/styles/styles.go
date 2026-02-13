package styles

import "charm.land/lipgloss/v2"

// BorderStyles defines the unified border system for the entire TUI.
// Design choice: RoundedBorder everywhere for a clean, modern look.
// Messages now use RoundedBorder (full box) for a chat-bubble design.
type BorderStyles struct {
	Message  lipgloss.Border // chat messages: RoundedBorder full box
	Dialog   lipgloss.Border // dialogs/overlays: RoundedBorder
	Panel    lipgloss.Border // side panels: NormalBorder (left only)
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

