package approval

import (
	"encoding/json"
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/layout"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// ApprovalAction represents the user's decision
type ApprovalAction int

const (
	ActionAllow       ApprovalAction = 0
	ActionAllowAlways ApprovalAction = 1
	ActionDeny        ApprovalAction = 2
)

// ToolApproval holds the tool request info
type ToolApproval struct {
	Name    string
	Args    map[string]interface{}
	Summary string
}

// ApprovalResult is sent when the user makes a decision
type ApprovalResult struct {
	Action ApprovalAction
	Tool   ToolApproval
}

type approvalDialog struct {
	width, height int
	index         int // 0=Allow, 1=Allow Always, 2=Deny
	tool          ToolApproval
}

// New creates a new approval dialog
func New(tool ToolApproval) dialogs.DialogModel {
	return &approvalDialog{
		tool: tool,
	}
}

func (a *approvalDialog) ID() dialogs.DialogID { return dialogs.DialogApproval }

func (a *approvalDialog) Init() tea.Cmd { return nil }

func (a *approvalDialog) Update(msg tea.Msg) (dialogs.DialogModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "tab", "l"))):
			a.index = (a.index + 1) % 3
			return a, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "shift+tab", "h"))):
			a.index = (a.index + 2) % 3
			return a, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return a, a.makeResult(ApprovalAction(a.index))

		case key.Matches(msg, key.NewBinding(key.WithKeys("a", "A"))):
			return a, a.makeResult(ActionAllow)

		case key.Matches(msg, key.NewBinding(key.WithKeys("s", "S"))):
			return a, a.makeResult(ActionAllowAlways)

		case key.Matches(msg, key.NewBinding(key.WithKeys("d", "D", "esc", "n", "N"))):
			return a, a.makeResult(ActionDeny)
		}
	}
	return a, nil
}

func (a *approvalDialog) makeResult(action ApprovalAction) tea.Cmd {
	tool := a.tool
	return func() tea.Msg {
		return dialogs.DialogClosedMsg{
			ID: dialogs.DialogApproval,
			Result: ApprovalResult{
				Action: action,
				Tool:   tool,
			},
		}
	}
}

func (a *approvalDialog) View() string {
	width := int(float64(a.width) * layout.DialogWidthRatio)
	if width > layout.DialogMaxWidth {
		width = layout.DialogMaxWidth
	}
	if width < layout.DialogMinWidth {
		width = layout.DialogMinWidth
	}
	innerWidth := width - 6

	// Title with diamond icon
	titleIcon := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render("\u25C6")
	titleText := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render(" Permission Required")
	titleStr := titleIcon + titleText +
		" " + lipgloss.NewStyle().Foreground(styles.Surface2).
		Render(strings.Repeat(core.IconSep, innerWidth-22))

	// Header
	toolStyle := lipgloss.NewStyle().
		Foreground(styles.Peach).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.Subtext0)

	header := toolStyle.Render(a.tool.Name) +
		descStyle.Render(" wants to execute:")

	// Content with pipe prefix
	content := a.renderContent(innerWidth)

	// Buttons with more spacing
	buttons := a.renderButtons(innerWidth)

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.Overlay0).
		Width(innerWidth).
		AlignHorizontal(lipgloss.Center)
	helpLine := helpStyle.Render("a allow \u00B7 s session \u00B7 d deny \u00B7 \u2190\u2192 navigate")

	parts := []string{
		titleStr,
		"",
		header,
		"",
		content,
		"",
		buttons,
		"",
		helpLine,
	}
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Mauve).
		Padding(1, 2).
		Width(width).
		Render(body)

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
	)
}

func (a *approvalDialog) renderContent(width int) string {
	// Pipe prefix style for command preview
	pipeStyle := lipgloss.NewStyle().Foreground(styles.Mauve)
	pipe := pipeStyle.Render("\u2503")

	switch a.tool.Name {
	case "bash":
		if cmd, ok := a.tool.Args["command"].(string); ok {
			preview := cmd
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			lines := strings.Split(preview, "\n")
			var formatted []string
			for i, line := range lines {
				prefix := "  "
				if i == 0 {
					prefix = "$ "
				}
				formatted = append(formatted, "  "+pipe+" "+lipgloss.NewStyle().
					Foreground(styles.Text).Render(prefix+line))
			}
			return strings.Join(formatted, "\n")
		}

	case "write_file":
		if path, ok := a.tool.Args["path"].(string); ok {
			pathStyle := lipgloss.NewStyle().Foreground(styles.Blue).Bold(true)
			info := "  " + pipe + " " + pathStyle.Render(path)
			if content, ok := a.tool.Args["content"].(string); ok {
				lines := strings.Count(content, "\n") + 1
				info += lipgloss.NewStyle().Foreground(styles.Subtext0).
					Render(fmt.Sprintf(" (%d lines)", lines))
			}
			return info
		}

	case "edit_file":
		if path, ok := a.tool.Args["path"].(string); ok {
			return "  " + pipe + " " + lipgloss.NewStyle().Foreground(styles.Blue).Bold(true).Render(path)
		}

	case "multiedit":
		if editsRaw, ok := a.tool.Args["edits"]; ok {
			data, _ := json.Marshal(editsRaw)
			var edits []map[string]interface{}
			json.Unmarshal(data, &edits)
			info := fmt.Sprintf("%d file(s)", len(edits))
			result := "  " + pipe + " " + lipgloss.NewStyle().Foreground(styles.Blue).Render(info)
			for _, e := range edits {
				if p, ok := e["path"].(string); ok {
					result += "\n  " + pipe + "   " + lipgloss.NewStyle().Foreground(styles.Subtext0).Render(p)
				}
			}
			return result
		}
	}

	if a.tool.Summary != "" {
		return "  " + pipe + " " + lipgloss.NewStyle().
			Foreground(styles.Text).
			Render(a.tool.Summary)
	}

	return ""
}

func (a *approvalDialog) renderButtons(width int) string {
	type buttonDef struct {
		label string
		key   string
	}

	buttons := []buttonDef{
		{"Allow", "a"},
		{"Allow Session", "s"},
		{"Deny", "d"},
	}

	var rendered []string
	for i, btn := range buttons {
		rendered = append(rendered, renderButton(btn.label, btn.key, i == a.index))
	}

	buttonsStr := strings.Join(rendered, "   ")
	if lipgloss.Width(buttonsStr) > width {
		buttonsStr = lipgloss.JoinVertical(lipgloss.Center, rendered...)
	}

	return lipgloss.NewStyle().
		Width(width).
		AlignHorizontal(lipgloss.Center).
		Render(buttonsStr)
}

func renderButton(label, hotkey string, selected bool) string {
	var fg, bg color.Color
	if selected {
		fg = styles.Base
		bg = styles.Mauve
	} else {
		fg = styles.Text
		bg = styles.Surface1
	}

	btnStyle := lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Padding(0, 2)

	return btnStyle.Render(label)
}

func (a *approvalDialog) SetSize(width, height int) {
	a.width = width
	a.height = height
}
