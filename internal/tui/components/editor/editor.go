package editor

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/layout"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// Editor is the interface for the input component
type Editor interface {
	layout.Model
	layout.Sizeable
	layout.Focusable
	IsEmpty() bool
	Value() string
	Reset()
	SetPlaceholder(string)
	InsertText(string)
	Textarea() *textarea.Model
}

type editorCmp struct {
	width, height int
	textarea      textarea.Model
	focused       bool
	placeholder   string
	pendingImages int
	yoloMode      bool
	provider      string
}

// New creates a new editor component
func New() Editor {
	ta := textarea.New()
	ta.Placeholder = "Ready!"
	ta.ShowLineNumbers = false
	ta.SetStyles(textarea.Styles{
		Focused: textarea.StyleState{
			Base:        lipgloss.NewStyle(),
			Text:        lipgloss.NewStyle().Foreground(styles.Text),
			CursorLine:  lipgloss.NewStyle(),
			Placeholder: lipgloss.NewStyle().Foreground(styles.Overlay0).Italic(true),
			Prompt:      lipgloss.NewStyle().Foreground(styles.Mauve),
		},
		Blurred: textarea.StyleState{
			Base:        lipgloss.NewStyle(),
			Text:        lipgloss.NewStyle().Foreground(styles.Overlay0),
			CursorLine:  lipgloss.NewStyle(),
			Placeholder: lipgloss.NewStyle().Foreground(styles.Overlay0).Italic(true),
			Prompt:      lipgloss.NewStyle().Foreground(styles.Overlay0),
		},
	})
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.Focus()

	return &editorCmp{
		textarea:    ta,
		focused:     true,
		placeholder: "Ready!",
	}
}

func (e *editorCmp) Init() tea.Cmd {
	return textarea.Blink
}

func (e *editorCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

func (e *editorCmp) View() string {
	if e.width == 0 {
		return ""
	}

	innerWidth := e.width - 6

	// Prompt symbol: ❯ in Mauve, or Red for YOLO
	promptStyle := lipgloss.NewStyle().Bold(true)
	prompt := "❯"
	if e.yoloMode {
		promptStyle = promptStyle.Foreground(styles.Red)
	} else {
		promptStyle = promptStyle.Foreground(styles.Mauve)
	}

	// Provider indicator in Peach bold
	providerStr := ""
	if e.provider != "" {
		providerStr = lipgloss.NewStyle().
			Foreground(styles.Peach).
			Bold(true).
			Render("@" + e.provider + " ")
	}

	// Image count
	imageStr := ""
	if e.pendingImages > 0 {
		imageStr = lipgloss.NewStyle().
			Foreground(styles.Overlay1).
			Italic(true).
			Render(fmt.Sprintf(" [img:%d] ", e.pendingImages))
	}

	// Input area
	inputView := e.textarea.View()

	// Content inside the box (no hints here - they go below)
	content := promptStyle.Render(prompt) + " " + providerStr + imageStr + "\n" + inputView

	borderColor := styles.Surface2
	if e.focused {
		borderColor = styles.Mauve
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(innerWidth).
		Render(content)

	// Hints below the box with styled shortcuts
	keyStyle := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)
	sepStyle := lipgloss.NewStyle().Foreground(styles.Surface2)

	sep := sepStyle.Render(" · ")
	hints := "  " +
		keyStyle.Render("enter") + " " + descStyle.Render("envoyer") + sep +
		keyStyle.Render("ctrl+k") + " " + descStyle.Render("commandes") + sep +
		keyStyle.Render("@") + descStyle.Render("provider")

	// Pad hints to center under the box
	boxWidth := lipgloss.Width(box)
	hintsWidth := lipgloss.Width(hints)
	if hintsWidth < boxWidth {
		pad := (boxWidth - hintsWidth) / 2
		hints = strings.Repeat(" ", pad) + hints
	}

	return box + "\n" + hints
}

func (e *editorCmp) SetSize(width, height int) tea.Cmd {
	e.width = width
	e.height = height
	e.textarea.SetWidth(width - 8)
	if height > 3 {
		e.textarea.SetHeight(height - 2)
	}
	return nil
}

func (e *editorCmp) GetSize() (int, int) {
	return e.width, e.height
}

func (e *editorCmp) Focus() tea.Cmd {
	e.focused = true
	return e.textarea.Focus()
}

func (e *editorCmp) Blur() tea.Cmd {
	e.focused = false
	e.textarea.Blur()
	return nil
}

func (e *editorCmp) IsFocused() bool {
	return e.focused
}

func (e *editorCmp) IsEmpty() bool {
	return e.textarea.Value() == ""
}

func (e *editorCmp) Value() string {
	return e.textarea.Value()
}

func (e *editorCmp) Reset() {
	e.textarea.Reset()
}

func (e *editorCmp) SetPlaceholder(p string) {
	e.placeholder = p
	e.textarea.Placeholder = p
}

func (e *editorCmp) InsertText(text string) {
	e.textarea.InsertString(text)
}

func (e *editorCmp) Textarea() *textarea.Model {
	return &e.textarea
}

// SetPendingImages updates the image count display
func (e *editorCmp) SetPendingImages(count int) {
	e.pendingImages = count
}

// SetYoloMode updates the YOLO indicator
func (e *editorCmp) SetYoloMode(on bool) {
	e.yoloMode = on
}

// SetProvider updates the provider indicator
func (e *editorCmp) SetProvider(name string) {
	e.provider = name
}
