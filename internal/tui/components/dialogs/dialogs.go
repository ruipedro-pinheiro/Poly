package dialogs

import (
	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/tui/layout"
)

// DialogCmp manages a LIFO stack of dialogs
type DialogCmp interface {
	layout.Model
	HasDialogs() bool
	ActiveDialogID() DialogID
	SetSize(width, height int)
}

type dialogCmp struct {
	width, height int
	dialogs       []DialogModel // LIFO stack: last element is topmost
}

// New creates a new dialog stack manager
func New() DialogCmp {
	return &dialogCmp{}
}

func (d *dialogCmp) Init() tea.Cmd {
	return nil
}

func (d *dialogCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case OpenDialogMsg:
		msg.Model.SetSize(d.width, d.height)
		d.dialogs = append(d.dialogs, msg.Model)
		return d, msg.Model.Init()

	case CloseDialogMsg:
		if len(d.dialogs) > 0 {
			closed := d.dialogs[len(d.dialogs)-1]
			d.dialogs = d.dialogs[:len(d.dialogs)-1]
			return d, func() tea.Msg {
				return DialogClosedMsg{ID: closed.ID()}
			}
		}
		return d, nil

	case CloseAllDialogsMsg:
		d.dialogs = nil
		return d, nil
	}

	// Forward messages to the topmost dialog
	if len(d.dialogs) > 0 {
		top := d.dialogs[len(d.dialogs)-1]
		updated, cmd := top.Update(msg)
		d.dialogs[len(d.dialogs)-1] = updated
		return d, cmd
	}

	return d, nil
}

func (d *dialogCmp) View() string {
	if len(d.dialogs) == 0 {
		return ""
	}
	// Render only the topmost dialog
	return d.dialogs[len(d.dialogs)-1].View()
}

func (d *dialogCmp) HasDialogs() bool {
	return len(d.dialogs) > 0
}

func (d *dialogCmp) ActiveDialogID() DialogID {
	if len(d.dialogs) == 0 {
		return ""
	}
	return d.dialogs[len(d.dialogs)-1].ID()
}

func (d *dialogCmp) SetSize(width, height int) {
	d.width = width
	d.height = height
	for _, dialog := range d.dialogs {
		dialog.SetSize(width, height)
	}
}
