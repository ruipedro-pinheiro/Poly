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
	header := m.renderHeader()

	// Chat area with consistent padding
	chatArea := lipgloss.NewStyle().
		Width(m.chatWidth()).
		Height(m.viewport.Height()).
		Padding(1, 2).
		Render(m.viewport.View())

	inputArea := m.renderInput()
	statusBar := m.renderStatusBar()

	result := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		chatArea,
		inputArea,
		statusBar,
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
	if panelW == 0 {
		return base
	}

	out := make([]string, len(baseLines))
	for i, baseLine := range baseLines {
		if i < len(panelLines) && panelLines[i] != "" {
			baseW := lipgloss.Width(baseLine)
			availBase := totalWidth - panelW
			if availBase < 0 {
				availBase = 0
			}
			if baseW > availBase {
				baseLine = lipgloss.NewStyle().Width(availBase).Render(baseLine)
			} else {
				pad := availBase - baseW
				if pad > 0 {
					baseLine += strings.Repeat(" ", pad)
				}
			}
			out[i] = baseLine + panelLines[i]
		} else {
			out[i] = baseLine
		}
	}

	return strings.Join(out, "\n")
}

func (m Model) renderHeader() string {
	return theme.HeaderStyle.Width(m.width).Render(m.headerBar.View())
}

func (m Model) renderInput() string {
	style := theme.InputBoxStyle
	if m.focused == "input" {
		style = theme.InputFocusedStyle
	}

	style = style.Width(m.width - 4).MarginLeft(2)

	providerDot := lipgloss.NewStyle().
		Foreground(theme.ProviderColor(m.defaultProvider)).
		Render("● ")

	imageIndicator := ""
	if len(m.pendingImages) > 0 {
		imageIndicator = lipgloss.NewStyle().
			Foreground(theme.Green).
			Render(fmt.Sprintf(" [img:%d] ", len(m.pendingImages)))
	}

	textArea := m.textarea.View()
	inputContent := providerDot + imageIndicator + textArea

	hintStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).MarginLeft(4)
	var hints string
	if m.isStreaming {
		hints = hintStyle.Render("esc stop")
	} else if strings.TrimSpace(m.textarea.Value()) == "" {
		hints = hintStyle.Render("enter send · @provider")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		style.Render(inputContent),
		hints,
		"", // Extra spacer
	)
}

func (m Model) renderStatusBar() string {
	return theme.StatusBarStyle.Width(m.width).Render(m.statusBar.View())
}

func (m *Model) updateViewport() {
	var content strings.Builder

	if len(m.messages) == 0 {
		welcomeStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay1).
			Italic(true).
			Width(m.contentWidth() - 4).
			Padding(2, 4)

		welcome := welcomeStyle.Render("Start a conversation with any AI model.\n\n" +
			"Tips:\n" +
			"  * Type a message to chat with @" + m.defaultProvider + "\n" +
			"  * Use @claude, @gpt, @gemini, or @grok to pick a model\n" +
			"  * Use @all to ask all connected models at once\n" +
			"  * Ctrl+D opens the dashboard\n" +
			"  * Ctrl+K opens the command palette")
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
	availWidth := m.contentWidth() - 4

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
	label := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("YOU")
	if count := len(msg.Images) + len(msg.ImageData); count > 0 {
		label += lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf(" [img:%d]", count))
	}

	content := lipgloss.NewStyle().
		Foreground(theme.Text).
		Width(availWidth - 2).
		Render(msg.Content)

	return theme.UserBubbleStyle.Width(availWidth).Render(label + "\n" + content)
}

func (m Model) renderSystemMessage(msg Message, availWidth int) string {
	return lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true).
		Padding(0, 2).
		Width(availWidth).
		Render("— " + msg.Content)
}

func (m Model) renderAssistantMessage(msg Message, index int, availWidth int, isExpanded bool, isStreamingLast bool) string {
	providerColor := theme.ProviderColor(msg.Provider)
	
	// Elegant header
	name := strings.ToUpper(msg.Provider)
	header := lipgloss.NewStyle().Foreground(providerColor).Bold(true).Render(core.IconModel + " " + name)
	
	innerWidth := availWidth - 2
	var parts []string
	parts = append(parts, header)

	if msg.Thinking != "" {
		if isExpanded || isStreamingLast {
			thinkContent := theme.ThinkingStyle.Width(innerWidth).Render(msg.Thinking)
			parts = append(parts, thinkContent)
		} else {
			collapsed := lipgloss.NewStyle().Foreground(theme.Overlay1).PaddingLeft(2).Render("• Thinking...")
			parts = append(parts, collapsed)
		}
	}

	// Render blocks or fallback content
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

	// Stats line
	if msg.InputTokens > 0 || msg.OutputTokens > 0 {
		stats := fmt.Sprintf("%s/%s tokens", formatTokenCount(msg.InputTokens), formatTokenCount(msg.OutputTokens))
		if cost := calculateCost(msg.InputTokens, msg.OutputTokens, msg.Provider); cost > 0 {
			stats += fmt.Sprintf(" · $%.4f", cost)
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Surface2).Italic(true).Render(stats))
	}

	body := strings.Join(parts, "\n")
	return lipgloss.NewStyle().PaddingLeft(1).Render(body)
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
		Padding(0, 1).
		Width(width).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface1)

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
