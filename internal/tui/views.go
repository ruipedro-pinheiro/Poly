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
	tuiLayout "github.com/pedromelo/poly/internal/tui/layout"
)

const (
	minInputBoxWidth     = 16
	minInputContentWidth = 10
	minWelcomeCardWidth  = 40
	maxWelcomeCardWidth  = 88

	minUserBubbleInnerWidth      = 16
	maxUserBubbleInnerWidth      = 92
	minAssistantBubbleInnerWidth = 28
	maxAssistantBubbleInnerWidth = 124
	inputPlaceholderText         = "Type a message..."
)

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
	header := m.renderHeader()

	chatArea := lipgloss.NewStyle().
		Width(m.chatWidth()).
		Height(m.viewport.Height()).
		Padding(0, 1).
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

// overlayRight places panel on the right side of base, line by line.
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
			// Truncate base line if it would overlap the panel
			if baseW > availBase {
				baseLine = lipgloss.NewStyle().Width(availBase).Render(baseLine)
			} else {
				// Pad base to reach the panel position
				pad := availBase - baseW
				if pad > 0 {
					baseLine = baseLine + strings.Repeat(" ", pad)
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
	return m.headerBar.View()
}

func (m Model) renderInput() string {
	provColor := theme.ProviderColor(m.defaultProvider)
	if m.defaultProvider == "" {
		provColor = theme.Overlay1
	}
	inputWidth := m.width - tuiLayout.InputWidthOffset
	if inputWidth < minInputBoxWidth {
		inputWidth = minInputBoxWidth
	}
	contentWidth := m.width - tuiLayout.InputBoxPadding
	if contentWidth < minInputContentWidth {
		contentWidth = minInputContentWidth
	}

	inputBoxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Background(theme.Base).
		Padding(0, 1).
		Width(inputWidth)

	if m.focused == "input" {
		inputBoxStyle = inputBoxStyle.
			BorderForeground(provColor).
			Background(theme.Base)
	}

	// Image attachment indicator
	imageIndicator := ""
	if len(m.pendingImages) > 0 {
		imageIndicator = lipgloss.NewStyle().
			Foreground(theme.Green).
			Render(fmt.Sprintf(" [img:%d] ", len(m.pendingImages)))
	}

	inputValue := strings.TrimSpace(m.textarea.Value())
	content := m.textarea.View()
	if inputValue == "" {
		// Avoid textarea's virtual cursor reverse-video block when empty.
		// We render a custom caret via boxPrefix instead.
		content = ""
		placeholder := lipgloss.NewStyle().
			Foreground(theme.Subtext0).
			Render(inputPlaceholderText)
		if m.focused == "input" {
			content += placeholder
		} else {
			content = placeholder
		}
	}

	inputContent := lipgloss.NewStyle().
		Foreground(theme.Text).
		Width(contentWidth).
		Render(imageIndicator + content)

	// Hints below the box (always present to keep layout stable)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	divider := lipgloss.NewStyle().Foreground(theme.Surface2).Render(" · ")
	descStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	modeText := "fast"
	if m.thinkingMode {
		modeText = "think"
	}
	routeHint := descStyle.Render("route ") + lipgloss.NewStyle().Foreground(provColor).Render("@"+m.defaultProvider)
	hints := hintStyle.Render("enter") + descStyle.Render(" send") +
		divider + hintStyle.Render("shift+enter") + descStyle.Render(" newline") +
		divider + hintStyle.Render("ctrl+k") + descStyle.Render(" commands") +
		divider + hintStyle.Render("mode") + descStyle.Render(" "+modeText) +
		divider + routeHint
	if m.isStreaming {
		hints = hintStyle.Render("esc") + descStyle.Render(" stop")
	}

	boxPrefix := ""
	if m.focused == "input" {
		boxPrefix = lipgloss.NewStyle().Foreground(provColor).Render("▍")
	} else {
		boxPrefix = lipgloss.NewStyle().Foreground(theme.Surface2).Render("▏")
	}
	box := inputBoxStyle.Render(boxPrefix + " " + inputContent)

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
		content.WriteString(m.renderWelcomePanel())
	} else {
		for i, msg := range m.messages {
			content.WriteString(m.renderMessage(msg, i))
			// Add spacing between messages
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
		isExpanded := m.thinkingExpanded[index]
		isStreamingLast := m.isStreaming && index == len(m.messages)-1
		return m.renderAssistantMessage(msg, index, availWidth, isExpanded, isStreamingLast)
	}
}

// renderUserMessage renders a user message with subtle background + Mauve accent
func (m Model) renderUserMessage(msg Message, availWidth int) string {
	// Label
	label := lipgloss.NewStyle().
		Foreground(theme.Subtext0).
		Bold(true).
		Render("you")
	labelRow := label
	imageCount := len(msg.Images) + len(msg.ImageData)
	if imageCount > 0 {
		imgBadge := lipgloss.NewStyle().Foreground(theme.Green).Render(" img:" + fmt.Sprintf("%d", imageCount))
		labelRow += imgBadge
	}
	labelWidth := lipgloss.Width(labelRow)

	maxInner := max(1, availWidth-6)
	if maxInner > maxUserBubbleInnerWidth {
		maxInner = maxUserBubbleInnerWidth
	}
	minInner := minUserBubbleInnerWidth
	if maxInner < minInner {
		minInner = maxInner
	}

	contentInner := maxLineWidth(msg.Content)
	if contentInner < labelWidth {
		contentInner = labelWidth
	}
	innerWidth := clampInt(contentInner, minInner, maxInner)

	// Content
	content := renderTextBlock(msg.Content, innerWidth)
	body := labelRow + "\n" + content

	inner := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderTop(false).
		BorderRight(false).
		BorderBottom(false).
		BorderForeground(theme.Mauve).
		PaddingLeft(2).
		PaddingRight(1).
		Foreground(theme.Text).
		MaxWidth(availWidth).
		Render(body)

	return inner
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
func (m Model) renderAssistantMessage(msg Message, index int, availWidth int, isExpanded bool, isStreamingLast bool) string {
	providerColor := theme.ProviderColor(msg.Provider)

	// Header line: provider name
	providerName := msg.Provider
	if providerName == "" {
		providerName = "assistant"
	}
	header := lipgloss.NewStyle().
		Foreground(providerColor).
		Bold(true).
		Render(strings.ToUpper(providerName))

	// Inner width for content
	innerWidth := availWidth - 6
	if innerWidth > maxAssistantBubbleInnerWidth {
		innerWidth = maxAssistantBubbleInnerWidth
	}
	if innerWidth < minAssistantBubbleInnerWidth {
		innerWidth = minAssistantBubbleInnerWidth
	}

	var parts []string
	parts = append(parts, header)

	// Thinking block - collapsed by default, expanded during streaming or on toggle
	if msg.Thinking != "" {
		showExpanded := isExpanded || isStreamingLast
		if showExpanded {
			thinkLabel := lipgloss.NewStyle().
				Foreground(theme.Lavender).
				Bold(true).
				Render("▼ THINKING")

			thinkContent := lipgloss.NewStyle().
				Foreground(theme.Overlay1).
				Italic(true).
				MaxWidth(innerWidth).
				Render(msg.Thinking)

			thinkBlock := lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(theme.Surface1).
				Padding(0, 1).
				MaxWidth(innerWidth).
				Render(thinkLabel + "\n" + thinkContent)

			parts = append(parts, thinkBlock)
		} else {
			collapsed := lipgloss.NewStyle().
				Foreground(theme.Overlay1).
				Render(core.IconActive + " Thinking...")
			parts = append(parts, collapsed)
		}
	}

	// Render body: inline interleaved blocks if available, else legacy
	if len(msg.Blocks) > 0 {
		// Check if all tool calls are done so we can batch-summarize
		allDone := true
		hasErrors := false
		var successNames []string
		for _, tc := range msg.ToolCalls {
			status := toolCallStatus(tc.Status)
			if status == tools.ToolStatusRunning || status == tools.ToolStatusPending {
				allDone = false
				break
			}
			if status == tools.ToolStatusError {
				hasErrors = true
			} else {
				successNames = append(successNames, tc.Name)
			}
		}

		if allDone && len(msg.ToolCalls) > 1 {
			// All tools finished: render text blocks normally, replace tool blocks with batch summary
			for _, block := range msg.Blocks {
				if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
					parts = append(parts, renderTextBlock(block.Text, innerWidth))
				}
			}
			// Batch summary for successes
			if len(successNames) > 0 {
				parts = append(parts, tools.RenderBatchSummary(successNames))
			}
			// Show errors individually
			if hasErrors {
				for _, tc := range msg.ToolCalls {
					if toolCallStatus(tc.Status) == tools.ToolStatusError {
						parts = append(parts, renderInlineToolCall(tc, innerWidth))
					}
				}
			}
		} else {
			// Still running or single tool: render blocks individually
			for _, block := range msg.Blocks {
				switch block.Type {
				case "text":
					if strings.TrimSpace(block.Text) != "" {
						parts = append(parts, renderTextBlock(block.Text, innerWidth))
					}
				case "tool":
					if block.ToolIdx < len(msg.ToolCalls) {
						tc := msg.ToolCalls[block.ToolIdx]
						parts = append(parts, renderInlineToolCall(tc, innerWidth))
					}
				}
			}
		}
	} else {
		// Legacy fallback
		if strings.TrimSpace(msg.Content) != "" {
			parts = append(parts, renderTextBlock(msg.Content, innerWidth))
		}
		if len(msg.ToolCalls) > 0 {
			rendered := renderToolCallBlock(msg.ToolCalls, innerWidth)
			if rendered != "" {
				parts = append(parts, rendered)
			}
		}
	}

	// Per-message token/cost annotation (subtle, at the bottom)
	if msg.InputTokens > 0 || msg.OutputTokens > 0 {
		tokenInfo := fmt.Sprintf("%s/%s tokens",
			formatTokenCount(msg.InputTokens),
			formatTokenCount(msg.OutputTokens))
		if cost := calculateCost(msg.InputTokens, msg.OutputTokens, msg.Provider); cost > 0 {
			tokenInfo += fmt.Sprintf(" · $%.4f", cost)
		}
		tokenLine := lipgloss.NewStyle().
			Foreground(theme.Overlay0).
			Italic(true).
			Render("usage: " + tokenInfo)
		parts = append(parts, tokenLine)
	}

	body := strings.Join(parts, "\n")

	// Thick left bar in provider color + subtle background
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderTop(false).
		BorderRight(false).
		BorderBottom(false).
		BorderForeground(providerColor).
		PaddingLeft(2).
		PaddingRight(1).
		MaxWidth(availWidth).
		Render(body)
}

func (m Model) renderWelcomePanel() string {
	width := m.contentWidth()
	if width > maxWelcomeCardWidth {
		width = maxWelcomeCardWidth
	}
	if width < minWelcomeCardWidth {
		width = minWelcomeCardWidth
	}

	title := core.GradientText("POLY", theme.Mauve, theme.Blue, true)
	subtitle := lipgloss.NewStyle().Foreground(theme.Subtext0).Render("local multi-model cockpit")

	provider := m.defaultProvider
	if provider == "" {
		provider = "none"
	}

	tips := []string{
		"start typing to chat with @" + provider,
		"switch model inline with @claude @gpt @gemini @grok",
		"run /compare or use @all for side-by-side answers",
		"ctrl+k command palette  ctrl+d control room  ctrl+t think mode",
	}

	tipStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	var body []string
	for _, tip := range tips {
		body = append(body, tipStyle.Render("· "+tip))
	}

	card := lipgloss.NewStyle().
		Width(width).
		Background(theme.Mantle).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface2).
		Padding(1, 2).
		Render(title + "\n" + subtitle + "\n\n" + strings.Join(body, "\n"))

	return lipgloss.PlaceHorizontal(m.contentWidth(), lipgloss.Center, card)
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func maxLineWidth(s string) int {
	if s == "" {
		return 0
	}
	maxW := 0
	for _, line := range strings.Split(s, "\n") {
		w := lipgloss.Width(line)
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

func renderTextBlock(content string, width int) string {
	text := strings.Trim(content, "\n")
	if width > 1 {
		text = wrapTextHard(text, width)
	}
	text = renderMarkdown(text, width)

	return lipgloss.NewStyle().
		Foreground(theme.Text).
		Width(width).
		MaxWidth(width).
		Render(text)
}

func wrapTextHard(text string, width int) string {
	if width <= 1 || text == "" {
		return text
	}

	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		if strings.TrimSpace(paragraph) == "" {
			out = append(out, "")
			continue
		}

		var line strings.Builder
		for _, word := range strings.Fields(paragraph) {
			wordWidth := lipgloss.Width(word)
			lineWidth := lipgloss.Width(line.String())

			// New word fits current line.
			if lineWidth == 0 && wordWidth <= width {
				line.WriteString(word)
				continue
			}
			if lineWidth > 0 && lineWidth+1+wordWidth <= width {
				line.WriteByte(' ')
				line.WriteString(word)
				continue
			}

			// Flush current line before processing the word.
			if lineWidth > 0 {
				out = append(out, line.String())
				line.Reset()
			}

			// Break very long words hard so they never overflow right edge.
			if wordWidth > width {
				var chunk strings.Builder
				for _, r := range word {
					next := chunk.String() + string(r)
					if lipgloss.Width(next) > width && chunk.Len() > 0 {
						out = append(out, chunk.String())
						chunk.Reset()
					}
					chunk.WriteRune(r)
				}
				if chunk.Len() > 0 {
					line.WriteString(chunk.String())
				}
				continue
			}

			line.WriteString(word)
		}

		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}

	return strings.Join(out, "\n")
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
	_ = session.AddMessage(m.toSessionMsg(msg))
}

// saveLastMessage persists the last message (called when streaming completes)
func (m *Model) saveLastMessage() {
	if len(m.messages) == 0 {
		return
	}
	m.saveMessageAt(len(m.messages) - 1)
}

// formatTokenCount formats a token count for display (e.g. 1234 -> "1.2k")
func formatTokenCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// saveMessageAt persists a specific message by index
func (m *Model) saveMessageAt(idx int) {
	if idx < 0 || idx >= len(m.messages) {
		return
	}
	_ = session.AddMessage(m.toSessionMsg(m.messages[idx]))
}
