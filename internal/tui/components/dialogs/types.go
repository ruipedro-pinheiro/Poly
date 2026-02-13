package dialogs

import (
	tea "charm.land/bubbletea/v2"
)

// DialogID uniquely identifies a dialog type
type DialogID string

const (
	DialogHelp           DialogID = "help"
	DialogModelPicker    DialogID = "model_picker"
	DialogControlRoom    DialogID = "control_room"
	DialogAddProvider    DialogID = "add_provider"
	DialogCommandPalette DialogID = "command_palette"
	DialogSessionList    DialogID = "session_list"
	DialogApproval       DialogID = "approval"
)

// DialogModel is the interface for all dialog components
type DialogModel interface {
	ID() DialogID
	Init() tea.Cmd
	Update(tea.Msg) (DialogModel, tea.Cmd)
	View() string
	SetSize(width, height int)
}

// OpenDialogMsg opens a new dialog on the stack
type OpenDialogMsg struct {
	Model DialogModel
}

// CloseDialogMsg closes the topmost dialog
type CloseDialogMsg struct{}

// CloseAllDialogsMsg closes all dialogs
type CloseAllDialogsMsg struct{}

// DialogClosedMsg is sent when a dialog closes, carrying any result
type DialogClosedMsg struct {
	ID     DialogID
	Result interface{}
}
