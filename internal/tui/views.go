package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/components/messages/tools"
	"github.com/pedromelo/poly/internal/tui/core"
)

func (m Model) chatWidth() int {
	return m.layout.ChatWidth
}

func (m Model) contentWidth() int {
	return m.layout.ContentWidth
}

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	var content string
	switch m.state {
	case viewSplash:
		content = m.renderSplash()
	case viewHelp:
		content = m.renderHelp()
	case viewModelPicker:
		content = m.renderModelPicker()
	case viewControlRoom:
		content = m.renderControlRoom()
	case viewAddProvider:
		content = m.renderAddProvider()
	case viewCommandPalette:
		content = m.renderCommandPalette()
	case viewApproval:
		content = m.renderApproval()
	case viewSessionList:
		content = m.renderSessionList()
	default:
		content = m.renderChat()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) renderChat() string {
	header := m.headerBar.View()
	
	// Main viewport with minimal side padding
	chatArea := lipgloss.NewStyle().
		Width(m.chatWidth()).
		Height(m.viewport.Height()).
		Padding(0, 1).
		Render(m.viewport.View())

	inputArea := m.renderInput()
	statusBar := m.statusBar.View()

	result := lipgloss.JoinVertical(
		lipgloss.Left,
		theme.HeaderStyle.Width(m.width).Render(header),
		chatArea,
		theme.InputStyle.Width(m.width).Render(inputArea),
		theme.StatusStyle.Width(m.width).Render(statusBar),
	)

	if m.infoPanelCmp.IsVisible() && m.width >= 100 {
		panel := m.infoPanelCmp.View()
		result = overlayRight(result, panel, m.width)
	}

	return result
}

func overlayRight(base, panel string, totalWidth int) string {
	baseLines := strings.Split(base, "\n")
	panelLines := strings.Split(panel, "\n")
	panelW := lipgloss.Width(panel)
	if panelW == 0 { return base }

	out := make([]string, len(baseLines))
	for i, baseLine := range baseLines {
		if i < len(panelLines) && panelLines[i] != "" {
			baseW := lipgloss.Width(baseLine)
			availBase := totalWidth - panelW
			if availBase < 0 { availBase = 0 }
			if baseW > availBase {
				baseLine = lipgloss.NewStyle().Width(availBase).Render(baseLine)
			} else {
				pad := availBase - baseW
				if pad > 0 { baseLine += strings.Repeat(" ", pad) }
			}
			out[i] = baseLine + panelLines[i]
		} else {
			out[i] = baseLine
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) renderInput() string {
	providerDot := lipgloss.NewStyle().
		Foreground(theme.ProviderColor(m.defaultProvider)).
		Render("● ")

	imageIndicator := ""
	if len(m.pendingImages) > 0 {
		imageIndicator = lipgloss.NewStyle().
			Foreground(theme.Green).
			Render(fmt.Sprintf("[%d images] ", len(m.pendingImages)))
	}

	textArea := m.textarea.View()
	
	hintStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	hints := ""
	if m.isStreaming {
		hints = " (esc to stop)"
	} else if strings.TrimSpace(m.textarea.Value()) == "" {
		hints = " (type to chat, @ to route)"
	}

	return providerDot + imageIndicator + textArea + hintStyle.Render(hints)
}

func (m *Model) updateViewport() {
	var content strings.Builder

	if len(m.messages) == 0 {
		welcome := lipgloss.NewStyle().
			Foreground(theme.Overlay1).
			Padding(2, 2).
			Render("Ready to assist. @claude, @gpt, @gemini... or @all to compare.")
		content.WriteString(welcome)
	} else {
		for i, msg := range m.messages {
			content.WriteString(m.renderMessage(msg, i))
			if i < len(m.messages)-1 {
				content.WriteString("\n\n")
			}
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m Model) renderMessage(msg Message, index int) string {
	availWidth := m.contentWidth() - 2

	switch msg.Role {
	case "user":
		return m.renderUserMessage(msg, availWidth)
	case "system":
		return m.renderSystemMessage(msg, availWidth)
	default:
		isExpanded := m.thinkingExpanded[index]
		isStreamingLast := m.isStreaming && index == len(m.messages)-1
		return m.renderAssistantMessage(msg, index, availWidth, isExpanded, isStreamingLast)
	}
}

func (m Model) renderUserMessage(msg Message, availWidth int) string {
	prefix := theme.UserPrefixStyle.Render("YOU > ")
	
	content := lipgloss.NewStyle().
		Width(availWidth - lipgloss.Width("YOU > ")).
		Render(msg.Content)

	if count := len(msg.Images) + len(msg.ImageData); count > 0 {
		prefix += lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf("[%d images] ", count))
	}

	return prefix + content
}

func (m Model) renderSystemMessage(msg Message, availWidth int) string {
	return lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true).
		Width(availWidth).
		Render("! " + msg.Content)
}

func (m Model) renderAssistantMessage(msg Message, index int, availWidth int, isExpanded bool, isStreamingLast bool) string {
	providerColor := theme.ProviderColor(msg.Provider)
	prefixText := strings.ToUpper(msg.Provider) + " > "
	prefix := lipgloss.NewStyle().Foreground(providerColor).Bold(true).Render(prefixText)
	
	innerWidth := availWidth - lipgloss.Width(prefixText)
	if innerWidth < 20 { innerWidth = 20 }

	var parts []string

	// Thinking
	if msg.Thinking != "" {
		if isExpanded || isStreamingLast {
			parts = append(parts, theme.ThinkingStyle.Width(innerWidth).Render(msg.Thinking))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.Overlay0).PaddingLeft(2).Render("... thinking"))
		}
	}

	// Content blocks
	if len(msg.Blocks) > 0 {
		for _, block := range msg.Blocks {
			switch block.Type {
			case "text":
				if strings.TrimSpace(block.Text) != "" {
					parts = append(parts, renderMarkdown(block.Text, innerWidth))
				}
			case "tool":
				if block.ToolIdx < len(msg.ToolCalls) {
					tc := msg.ToolCalls[block.ToolIdx]
					parts = append(parts, renderInlineToolCall(tc, innerWidth))
				}
			}
		}
	} else if strings.TrimSpace(msg.Content) != "" {
		parts = append(parts, renderMarkdown(msg.Content, innerWidth))
	}

	// Stats
	if msg.InputTokens > 0 || msg.OutputTokens > 0 {
		stats := fmt.Sprintf("[%s/%s]", formatTokenCount(msg.InputTokens), formatTokenCount(msg.OutputTokens))
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Surface1).Render(stats))
	}

	body := strings.Join(parts, "\n\n")
	// Indent the body relative to the prefix
	indentedBody := lipgloss.NewStyle().PaddingLeft(lipgloss.Width(prefixText)).Render(body)
	
	// First line combines prefix and first part of body if possible, but simpler to just stack
	return prefix + "\n" + indentedBody
}

func renderInlineToolCall(tc ToolCallData, width int) string {
	status := toolCallStatus(tc.Status)
	rendered := tools.RenderToolCall(width-4, &tools.RenderOpts{
		Name:    tc.Name,
		Args:    tc.Args,
		Result:  tc.Result,
		IsError: tc.IsError,
		Status:  status,
		Compact: true,
	})

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(theme.Surface1).
		PaddingLeft(1).
		Width(width)

	if status == tools.ToolStatusError {
		style = style.BorderForeground(theme.Red)
	} else if status == tools.ToolStatusRunning {
		style = style.BorderForeground(theme.Yellow)
	}

	return style.Render(rendered)
}

func toolCallStatus(s int) tools.ToolStatus {
	switch s {
	case 1: return tools.ToolStatusRunning
	case 2: return tools.ToolStatusSuccess
	case 3: return tools.ToolStatusError
	default: return tools.ToolStatusPending
	}
}

func (m *Model) addMessage(msg Message) {
	m.messages = append(m.messages, msg)
	_ = session.AddMessage(m.toSessionMsg(msg))
}

func (m *Model) saveLastMessage() {
	if len(m.messages) > 0 {
		m.saveMessageAt(len(m.messages) - 1)
	}
}

func formatTokenCount(n int) string {
	if n >= 1000 { return fmt.Sprintf("%.1fk", float64(n)/1000) }
	return fmt.Sprintf("%d", n)
}

func (m *Model) saveMessageAt(idx int) {
	if idx >= 0 && idx < len(m.messages) {
		_ = session.AddMessage(m.toSessionMsg(m.messages[idx]))
	}
}
