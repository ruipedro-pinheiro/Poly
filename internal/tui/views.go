package tui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/components/messages/tools"
	"github.com/pedromelo/poly/internal/tui/core"
	tuiLayout "github.com/pedromelo/poly/internal/tui/layout"
)

// sidebarWidth returns the sidebar width from the layout context.
func (m Model) sidebarWidth() int {
	return m.layout.SidebarWidth
}

// chatWidth returns the width available for the chat area from the layout context.
func (m Model) chatWidth() int {
	return m.layout.ChatWidth
}

// contentWidth returns the width available for message content from the layout context.
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
	// Header
	header := m.renderHeader()

	sbWidth := m.sidebarWidth()
	cWidth := m.chatWidth()

	// Chat area
	chatArea := lipgloss.NewStyle().
		Width(cWidth).
		Height(m.viewport.Height()).
		Padding(0, 1).
		Render(m.viewport.View())

	// Sidebar
	var sidebarArea string
	if sbWidth > 0 {
		sidebarArea = m.renderSidebar(sbWidth, m.viewport.Height())
	}

	// Combine chat + sidebar horizontally
	var mainArea string
	if sbWidth > 0 {
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, chatArea, sidebarArea)
	} else {
		mainArea = chatArea
	}

	// Input area
	inputArea := m.renderInput()

	// Status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		mainArea,
		inputArea,
		statusBar,
	)
}

func (m Model) renderHeader() string {
	return m.headerBar.View()
}

func (m Model) renderInput() string {
	inputBoxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Padding(0, 1).
		Width(m.width - tuiLayout.InputWidthOffset)

	if m.focused == "input" {
		inputBoxStyle = inputBoxStyle.BorderForeground(theme.Mauve)
	}

	// Provider indicator
	providerDot := lipgloss.NewStyle().
		Foreground(theme.ProviderColor(m.defaultProvider)).
		Render("● ")

	// Image attachment indicator
	imageIndicator := ""
	if len(m.pendingImages) > 0 {
		imageIndicator = lipgloss.NewStyle().
			Foreground(theme.Green).
			Render(fmt.Sprintf(" [img:%d] ", len(m.pendingImages)))
	}

	content := m.textarea.View()

	inputContent := lipgloss.NewStyle().
		Width(m.width - tuiLayout.InputBoxPadding).
		Render(providerDot + imageIndicator + content)

	// Hints below the box
	hintStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	var hints string
	if m.isStreaming {
		hints = hintStyle.Render("esc stop streaming")
	} else {
		hints = hintStyle.Render("enter send · ctrl+k commands · @provider or @all")
	}

	box := inputBoxStyle.Render(inputContent)

	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(box + "\n" + lipgloss.NewStyle().Padding(0, 2).Render(hints))
}

func (m Model) renderStatusBar() string {
	return m.statusBar.View()
}

func (m *Model) updateViewport() {
	var content strings.Builder

	if len(m.messages) == 0 {
		// Welcome message when no conversations yet
		welcomeStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay1).
			Italic(true).
			Width(m.contentWidth())

		tipStyle := lipgloss.NewStyle().
			Foreground(theme.Overlay0)

		welcome := welcomeStyle.Render("Start a conversation with any AI model.\n\n")
		tips := tipStyle.Render(
			"Tips:\n" +
				"  * Type a message to chat with @" + m.defaultProvider + "\n" +
				"  * Use @claude, @gpt, @gemini, or @grok to pick a model\n" +
				"  * Use @all to ask all connected models at once\n" +
				"  * Ctrl+D opens the dashboard to connect providers\n" +
				"  * Ctrl+K opens the command palette\n" +
				"  * Ctrl+T toggles thinking mode (ON by default)",
		)
		content.WriteString(welcome + tips)
	} else {
		for i, msg := range m.messages {
			content.WriteString(m.renderMessage(msg, i))
			// Add extra spacing between messages (not after the last one)
			if i < len(m.messages)-1 {
				content.WriteString("\n\n")
			} else {
				content.WriteString("\n")
			}
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m Model) renderMessage(msg Message, index int) string {
	availWidth := m.contentWidth()

	switch msg.Role {
	case "user":
		return m.renderUserMessage(msg, availWidth)
	case "system":
		return m.renderSystemMessage(msg, availWidth)
	default:
		return m.renderAssistantMessage(msg, index, availWidth)
	}
}

// renderUserMessage renders a user message with a thick left bar (Mauve)
func (m Model) renderUserMessage(msg Message, availWidth int) string {
	// Label
	label := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("You")
	imageCount := len(msg.Images) + len(msg.ImageData)
	if imageCount > 0 {
		imgBadge := lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf(" [img:%d]", imageCount))
		label += imgBadge
	}

	// Content
	contentStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Width(availWidth - 4)
	content := contentStyle.Render(msg.Content)

	// Wrap in thick left border
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(theme.Mauve).
		PaddingLeft(1).
		Width(availWidth).
		Render(label + "\n" + content)
}

// renderSystemMessage renders a system/info message dimmed
func (m Model) renderSystemMessage(msg Message, availWidth int) string {
	return lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Italic(true).
		Width(availWidth).
		Render(msg.Content)
}

// renderAssistantMessage renders an assistant message with provider-colored left bar
func (m Model) renderAssistantMessage(msg Message, index int, availWidth int) string {
	providerColor := theme.ProviderColor(msg.Provider)

	// Header line: ◇ provider · time ─────────
	headerParts := []string{core.IconModel + " " + msg.Provider}
	headerText := lipgloss.NewStyle().Foreground(providerColor).Bold(true).Render(strings.Join(headerParts, ""))

	// Separator line after header
	headerLen := lipgloss.Width(headerText)
	sepLen := availWidth - headerLen - 5
	if sepLen < 3 {
		sepLen = 3
	}
	separator := lipgloss.NewStyle().Foreground(theme.Surface2).Render(" " + strings.Repeat("─", sepLen))
	header := headerText + separator

	// Inner width for content
	innerWidth := availWidth - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	var parts []string
	parts = append(parts, header)

	// Thinking block - simple background, NO nested border
	if msg.Thinking != "" {
		thinkLabel := lipgloss.NewStyle().
			Foreground(theme.Lavender).
			Bold(true).
			Render(core.IconLoading + " Thinking")

		thinkContent := lipgloss.NewStyle().
			Foreground(theme.Overlay1).
			Italic(true).
			Width(innerWidth).
			Render(msg.Thinking)

		// Simple background block, no border
		thinkBlock := lipgloss.NewStyle().
			Background(theme.Surface0).
			Padding(0, 1).
			Width(innerWidth).
			Render(thinkLabel + "\n" + thinkContent)

		parts = append(parts, thinkBlock)
	}

	// Render body: inline interleaved blocks if available, else legacy
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
	} else {
		// Legacy fallback
		if strings.TrimSpace(msg.Content) != "" {
			parts = append(parts, renderMarkdown(msg.Content, innerWidth))
		}
		if len(msg.ToolCalls) > 0 {
			rendered := renderToolCallBlock(msg.ToolCalls, innerWidth)
			if rendered != "" {
				parts = append(parts, rendered)
			}
		}
	}

	body := strings.Join(parts, "\n")

	// Thick left bar in provider color - clean and simple
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(providerColor).
		PaddingLeft(1).
		Width(availWidth).
		Render(body)
}

// renderInlineToolCall renders a single tool call inline
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

	switch status {
	case tools.ToolStatusRunning, tools.ToolStatusPending:
		// Active: box with visible border
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Surface2).
			Padding(0, 1).
			Width(width).
			Render(rendered)

	case tools.ToolStatusError:
		// Error: red box (renderer already includes error details)
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Red).
			Padding(0, 1).
			Width(width).
			Render(rendered)

	default:
		// Success: dim compact line, no box
		return lipgloss.NewStyle().Foreground(theme.Overlay0).Render(rendered)
	}
}

// toolCallStatus converts int status to ToolStatus
func toolCallStatus(s int) tools.ToolStatus {
	switch s {
	case 0:
		return tools.ToolStatusPending
	case 1:
		return tools.ToolStatusRunning
	case 2:
		return tools.ToolStatusSuccess
	case 3:
		return tools.ToolStatusError
	default:
		return tools.ToolStatusPending
	}
}

// renderToolCallBlock renders tool calls, collapsing completed ones
func renderToolCallBlock(calls []ToolCallData, width int) string {
	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Padding(0, 1).
		Width(width)

	// Count by status
	success, errors, active := 0, 0, 0
	for _, tc := range calls {
		switch toolCallStatus(tc.Status) {
		case tools.ToolStatusSuccess:
			success++
		case tools.ToolStatusError:
			errors++
		default:
			active++ // pending or running
		}
	}

	allDone := active == 0
	var lines []string

	if allDone && errors == 0 {
		// All success: dim one-liner summary
		label := "tool"
		if success > 1 {
			label = "tools"
		}
		return lipgloss.NewStyle().Foreground(theme.Overlay0).Render(
			fmt.Sprintf("%s %d %s completed", core.IconCheck, success, label),
		)
	}

	if allDone {
		// All done but has errors: show errors only
		for _, tc := range calls {
			if toolCallStatus(tc.Status) == tools.ToolStatusError {
				rendered := tools.RenderToolCall(width-4, &tools.RenderOpts{
					Name:    tc.Name,
					Args:    tc.Args,
					Result:  tc.Result,
					IsError: tc.IsError,
					Status:  tools.ToolStatusError,
					Compact: true,
				})
				lines = append(lines, rendered)
			}
		}
	} else {
		// Still running: show active/error calls, collapse completed
		if success > 0 {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Overlay0).Render(
				fmt.Sprintf("%s %d completed", core.IconCheck, success),
			))
		}
		for _, tc := range calls {
			status := toolCallStatus(tc.Status)
			if status == tools.ToolStatusSuccess {
				continue // collapsed above
			}
			rendered := tools.RenderToolCall(width-4, &tools.RenderOpts{
				Name:    tc.Name,
				Args:    tc.Args,
				Result:  tc.Result,
				IsError: tc.IsError,
				Status:  status,
				Compact: true,
			})
			lines = append(lines, rendered)
		}
	}

	content := strings.Join(lines, "\n")
	return containerStyle.Render(content)
}

// addMessage adds a message and persists it to the session
func (m *Model) addMessage(msg Message) {
	m.messages = append(m.messages, msg)
	// Persist to session
	session.AddMessage(session.Message{
		Role:     msg.Role,
		Content:  msg.Content,
		Provider: msg.Provider,
		Thinking: msg.Thinking,
		Images:   msg.Images,
	})
}

// saveSession saves the current state to the session
func (m *Model) saveSession() {
	session.SetProvider(m.defaultProvider)
}

// saveLastMessage persists the last message (called when streaming completes)
func (m *Model) saveLastMessage() {
	if len(m.messages) == 0 {
		return
	}
	m.saveMessageAt(len(m.messages) - 1)
}

// saveMessageAt persists a specific message by index
func (m *Model) saveMessageAt(idx int) {
	if idx < 0 || idx >= len(m.messages) {
		return
	}
	msg := m.messages[idx]
	session.AddMessage(session.Message{
		Role:     msg.Role,
		Content:  msg.Content,
		Provider: msg.Provider,
		Thinking: msg.Thinking,
		Images:   msg.Images,
	})
}
