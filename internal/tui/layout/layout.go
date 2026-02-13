package layout

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Model is the base interface for all TUI components
type Model interface {
	Init() tea.Cmd
	Update(tea.Msg) (Model, tea.Cmd)
	View() string
}

// Focusable components can receive and lose focus
type Focusable interface {
	Focus() tea.Cmd
	Blur() tea.Cmd
	IsFocused() bool
}

// Sizeable components can be resized
type Sizeable interface {
	SetSize(width, height int) tea.Cmd
	GetSize() (int, int)
}

// Help components expose their keybindings
type Help interface {
	Bindings() []key.Binding
}
