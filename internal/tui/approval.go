package tui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
)

// Approval options
const (
	approvalAllow       = 0
	approvalAllowAlways = 1
	approvalDeny        = 2
)

// ToolPendingMsg is sent when a tool needs user approval
type ToolPendingMsg struct {
	Approval tools.PendingApproval
}

// watchForApprovals returns a cmd that blocks until a tool needs approval
func watchForApprovals() tea.Cmd {
	return func() tea.Msg {
		pending := <-tools.PendingChan
		return ToolPendingMsg{Approval: pending}
	}
}

// renderApproval renders the permission dialog (Crush-style)
func (m Model) renderApproval() string {
	// Dialog sizing based on tool type
	width := int(float64(m.width) * 0.8)
	if width > 80 {
		width = 80
	}
	if width < 40 {
		width = 40
	}
	innerWidth := width - 6 // padding + border

	// --- Title ---
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Bold(true)
	title := titleStyle.Render("Permission Required")
	titleLine := lipgloss.NewStyle().
		Width(innerWidth).
		AlignHorizontal(lipgloss.Center).
		Render(title)

	// --- Header: tool name + description ---
	toolStyle := lipgloss.NewStyle().
		Foreground(theme.Peach).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(theme.Subtext0)

	header := toolStyle.Render(m.pendingApproval.Name) +
		descStyle.Render(" wants to execute:")

	// --- Content area (tool-specific) ---
	content := m.renderApprovalContent(innerWidth)

	// --- Buttons ---
	buttons := m.renderApprovalButtons(innerWidth)

	// --- Help line ---
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Width(innerWidth).
		AlignHorizontal(lipgloss.Center)
	help := helpStyle.Render("a allow · s always · d/esc deny · ←→ navigate · enter confirm")

	// --- Assemble dialog ---
	parts := []string{
		titleLine,
		"",
		header,
		"",
		content,
		"",
		buttons,
		"",
		help,
	}
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Padding(1, 2).
		Width(width).
		Render(body)

	// Center on screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// renderApprovalContent generates tool-specific content for the dialog
func (m Model) renderApprovalContent(width int) string {
	codeStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Background(theme.Surface0).
		Padding(0, 1).
		Width(width)

	switch m.pendingApproval.Name {
	case "bash":
		if cmd, ok := m.pendingApproval.Args["command"].(string); ok {
			preview := cmd
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			// Show as code block with $ prefix
			lines := strings.Split(preview, "\n")
			var formatted []string
			for i, line := range lines {
				if i == 0 {
					formatted = append(formatted, "$ "+line)
				} else {
					formatted = append(formatted, "  "+line)
				}
			}
			return codeStyle.Render(strings.Join(formatted, "\n"))
		}

	case "write_file":
		if path, ok := m.pendingApproval.Args["path"].(string); ok {
			pathStyle := lipgloss.NewStyle().Foreground(theme.Blue).Bold(true)
			info := pathStyle.Render(path)
			if content, ok := m.pendingApproval.Args["content"].(string); ok {
				lines := strings.Count(content, "\n") + 1
				info += lipgloss.NewStyle().Foreground(theme.Subtext0).
					Render(fmt.Sprintf(" (%d lines)", lines))
			}
			return info
		}

	case "edit_file":
		if path, ok := m.pendingApproval.Args["path"].(string); ok {
			pathStyle := lipgloss.NewStyle().Foreground(theme.Blue).Bold(true)
			return pathStyle.Render(path)
		}

	case "multiedit":
		if editsRaw, ok := m.pendingApproval.Args["edits"]; ok {
			data, _ := json.Marshal(editsRaw)
			var edits []map[string]interface{}
			if err := json.Unmarshal(data, &edits); err != nil {
				return ""
			}
			info := fmt.Sprintf("%d file(s)", len(edits))
			for _, e := range edits {
				if p, ok := e["path"].(string); ok {
					info += "\n  " + p
				}
			}
			return lipgloss.NewStyle().Foreground(theme.Blue).Render(info)
		}
	}

	// Default: show summary
	if m.pendingApproval.Summary != "" {
		return lipgloss.NewStyle().
			Foreground(theme.Text).
			Width(width).
			Render(m.pendingApproval.Summary)
	}

	return ""
}

// renderApprovalButtons renders the 3 selectable buttons (Crush-style)
func (m Model) renderApprovalButtons(width int) string {
	type buttonDef struct {
		text           string
		underlineIndex int // index of the hotkey letter to underline
	}

	buttons := []buttonDef{
		{"Allow", 0},              // underline 'A'
		{"Allow for Session", 10}, // underline 'S' in "Session"
		{"Deny", 0},               // underline 'D'
	}

	var rendered []string
	for i, btn := range buttons {
		rendered = append(rendered, renderButton(btn.text, btn.underlineIndex, i == m.approvalIndex))
	}

	buttonsStr := strings.Join(rendered, "  ")

	// If too wide, stack vertically
	if lipgloss.Width(buttonsStr) > width {
		buttonsStr = lipgloss.JoinVertical(lipgloss.Center, rendered...)
	}

	return lipgloss.NewStyle().
		Width(width).
		AlignHorizontal(lipgloss.Right).
		Render(buttonsStr)
}

// renderButton renders a single selectable button with underlined hotkey
func renderButton(text string, underlineIdx int, selected bool) string {
	var fg, bg color.Color
	if selected {
		fg = theme.Base
		bg = theme.Mauve
	} else {
		fg = theme.Text
		bg = theme.Surface1
	}

	// Inline styles for text parts (NO padding here)
	normal := lipgloss.NewStyle().
		Foreground(fg).
		Background(bg)

	underlinedStyle := lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Underline(true)

	// Build the inner text with underlined hotkey
	var inner string
	if underlineIdx >= 0 && underlineIdx < len(text) {
		before := text[:underlineIdx]
		hotkey := text[underlineIdx : underlineIdx+1]
		after := text[underlineIdx+1:]
		inner = normal.Render(before) + underlinedStyle.Render(hotkey) + normal.Render(after)
	} else {
		inner = normal.Render(text)
	}

	// Wrap with padding ONCE at the outer level only
	return lipgloss.NewStyle().
		Background(bg).
		Padding(0, 2).
		Render(inner)
}

// handleApprovalKey handles key presses in the approval dialog
func (m Model) handleApprovalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	// Navigation: left/right/tab to cycle through buttons
	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "tab", "l"))):
		m.approvalIndex = (m.approvalIndex + 1) % 3
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "shift+tab", "h"))):
		m.approvalIndex = (m.approvalIndex + 2) % 3
		return m, nil

	// Confirm selected option
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.executeApproval(m.approvalIndex)

	// Quick action keys (like Crush: a=allow, s=always, d=deny)
	case key.Matches(msg, key.NewBinding(key.WithKeys("a", "A"))):
		return m.executeApproval(approvalAllow)

	case key.Matches(msg, key.NewBinding(key.WithKeys("s", "S"))):
		return m.executeApproval(approvalAllowAlways)

	case key.Matches(msg, key.NewBinding(key.WithKeys("d", "D", "esc", "n", "N"))):
		return m.executeApproval(approvalDeny)
	}

	return m, nil
}

// executeApproval executes the selected approval action
func (m Model) executeApproval(action int) (tea.Model, tea.Cmd) {
	switch action {
	case approvalAllow:
		tools.ApprovedChan <- true
		m.state = viewChat
		m.status = fmt.Sprintf("Approved: %s", m.pendingApproval.Name)
		m.pendingApproval = tools.PendingApproval{}
		m.approvalIndex = 0
		return m, watchForApprovals()

	case approvalAllowAlways:
		m.approvedTools[m.pendingApproval.Name] = true
		tools.ApprovedChan <- true
		m.state = viewChat
		m.status = fmt.Sprintf("Auto-approved: %s", m.pendingApproval.Name)
		m.pendingApproval = tools.PendingApproval{}
		m.approvalIndex = 0
		return m, watchForApprovals()

	case approvalDeny:
		tools.ApprovedChan <- false
		m.state = viewChat
		m.status = "Tool denied"
		m.pendingApproval = tools.PendingApproval{}
		m.approvalIndex = 0
		return m, watchForApprovals()
	}

	return m, nil
}
