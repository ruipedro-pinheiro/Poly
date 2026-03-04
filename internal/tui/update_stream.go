package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/notify"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/tui/components/status"
)

// handleStreamMsg processes a streaming event from a provider
func (m Model) handleStreamMsg(msg StreamMsg) (tea.Model, tea.Cmd) {
	if !m.isStreaming {
		streamEventChan = nil
		return m, nil
	}
	if len(m.messages) == 0 {
		return m, nil
	}

	lastIdx := len(m.messages) - 1

	if msg.Error != nil {
		if llm.IsImageError(msg.Error) {
			llm.SetImageSupport(msg.Provider, false)
			m.messages[lastIdx].Content = ""
			m.status = "Retrying without images..."
			m.updateViewport()
			return m, m.sendToProvider(msg.Provider, "")
		}
		m.messages[lastIdx].Content = "Error: " + msg.Error.Error()
		m.status = "Error"
		m.isStreaming = false
		m.streamTokenCount = 0
		m.statusBar.Update(status.SetStreamingMsg{Active: false})
		m.saveLastMessage()
		m.updateViewport()
		return m, nil
	}

	if msg.Done {
		m.thinkingExpanded[lastIdx] = false
		m.updateTokenStats(msg)
		m.messages[lastIdx].InputTokens = msg.InputTokens
		m.messages[lastIdx].OutputTokens = msg.OutputTokens

		if !m.streamStartTime.IsZero() {
			elapsed := time.Since(m.streamStartTime)
			m.status = fmt.Sprintf("Ready (%ds)", int(elapsed.Seconds()))
			m.statusBar.Update(status.SetStreamingMsg{
				Active:  false,
				Elapsed: elapsed,
				Tokens:  m.streamTokenCount,
			})
			m.streamStartTime = time.Time{}
		} else {
			m.status = "Ready"
			m.statusBar.Update(status.SetStreamingMsg{Active: false})
		}
		m.isStreaming = false
		m.streamTokenCount = 0
		m.saveLastMessage()
		m.updateViewport()
		m.syncStatusBar()
		m.syncInfoPanel()

		if m.notificationsOn {
			notify.Send("Poly", msg.Provider+" finished responding")
		}

		llmMessages := m.buildLLMMessagesForCompaction()
		if llm.NeedsCompaction(llmMessages, 0) && !m.isCompacting {
			m.isCompacting = true
			m.status = "Auto-compacting context..."
			return m, func() tea.Msg { return CompactMsg{} }
		}
		return m, nil
	}

	// Shared logic for tool use, tool result, thinking and content
	m.applyStreamEvent(lastIdx, msg.Content, msg.Thinking, msg.ToolCall, msg.ToolResult)

	m.updateViewport()
	return m, readStreamEvent(msg.Provider)
}

// handleTableRondeStreamMsg processes a Table Ronde streaming event
func (m Model) handleTableRondeStreamMsg(msg TableRondeStreamMsg) (tea.Model, tea.Cmd) {
	if m.tableRonde == nil {
		return m, nil
	}

	idx, ok := m.tableRonde.messageIndices[msg.Provider]
	if !ok || idx < 0 || idx >= len(m.messages) {
		return m, nil
	}

	if msg.Error != nil {
		if llm.IsImageError(msg.Error) {
			llm.SetImageSupport(msg.Provider, false)
			m.messages[idx].Content = "(Images not supported)"
		} else {
			m.messages[idx].Content = "Error: " + msg.Error.Error()
		}
		m.tableRonde.activeProviders[msg.Provider] = false
		m.updateViewport()
		return m, m.checkRoundComplete()
	}

	if msg.Done {
		m.tableRonde.activeProviders[msg.Provider] = false
		m.messages[idx].InputTokens = msg.InputTokens
		m.messages[idx].OutputTokens = msg.OutputTokens
		m.saveMessageAt(idx)
		m.updateViewport()
		return m, m.checkRoundComplete()
	}

	// Shared logic for tool use, tool result, thinking and content
	m.applyStreamEvent(idx, msg.Content, msg.Thinking, msg.ToolCall, msg.ToolResult)

	m.updateViewport()
	return m, readTableRondeEvent(msg.Provider, msg.Round)
}

// applyStreamEvent centralizes the logic for updating a message during streaming
func (m *Model) applyStreamEvent(idx int, content, thinking string, toolCall *llm.ToolCall, toolResult *llm.ToolResult) {
	// Handle tool_use: tool starting
	if toolCall != nil && toolResult == nil {
		toolIdx := len(m.messages[idx].ToolCalls)
		m.messages[idx].ToolCalls = append(m.messages[idx].ToolCalls, ToolCallData{
			Name:   toolCall.Name,
			Args:   toolCall.Arguments,
			Status: 1, // running
		})
		m.messages[idx].Blocks = append(m.messages[idx].Blocks, ContentBlock{
			Type:    "tool",
			ToolIdx: toolIdx,
		})
		return
	}

	// Handle tool_result: tool finished
	if toolCall != nil && toolResult != nil {
		for i := len(m.messages[idx].ToolCalls) - 1; i >= 0; i-- {
			tc := &m.messages[idx].ToolCalls[i]
			if tc.Name == toolCall.Name && tc.Status == 1 {
				tc.Result = toolResult.Content
				tc.IsError = toolResult.IsError
				if toolResult.IsError {
					tc.Status = 3 // error
				} else {
					tc.Status = 2 // success
				}
				break
			}
		}
		return
	}

	// Handle thinking
	if thinking != "" && m.thinkingMode {
		m.messages[idx].Thinking += thinking
	}

	// Accumulate content
	if content != "" {
		m.messages[idx].Content += content
		// Update stream token count (only used for non-Table Ronde UI stats)
		if !m.isStreamingTableRonde() {
			m.streamTokenCount += (len(content) + 3) / 4
		}

		blocks := m.messages[idx].Blocks
		if len(blocks) > 0 && blocks[len(blocks)-1].Type == "text" {
			blocks[len(blocks)-1].Text += content
			m.messages[idx].Blocks = blocks
		} else {
			m.messages[idx].Blocks = append(m.messages[idx].Blocks, ContentBlock{
				Type: "text",
				Text: content,
			})
		}
	}
}

// isStreamingTableRonde returns true if @all Table Ronde is active
func (m *Model) isStreamingTableRonde() bool {
	return m.tableRonde != nil
}

// checkRoundComplete checks if all providers are done in the current round
func (m *Model) checkRoundComplete() tea.Cmd {
	if m.tableRonde == nil {
		return nil
	}
	for _, active := range m.tableRonde.activeProviders {
		if active {
			return nil
		}
	}
	mentions := m.extractMentions()
	if len(mentions) > 0 && m.tableRonde.round < m.tableRonde.maxRounds {
		return m.startNextRound(mentions)
	}
	m.finishTableRonde()
	return nil
}

// updateTokenStats updates token/cost tracking from a completed stream event
func (m *Model) updateTokenStats(msg StreamMsg) {
	if msg.InputTokens > 0 {
		m.sessionInputTokens += msg.InputTokens
	}
	if msg.OutputTokens > 0 {
		m.sessionOutputTokens += msg.OutputTokens
	}
	if msg.CacheCreationTokens > 0 {
		m.sessionCacheCreationTokens += msg.CacheCreationTokens
	}
	if msg.CacheReadTokens > 0 {
		m.sessionCacheReadTokens += msg.CacheReadTokens
	}
	m.sessionCost = calculateCostWithCache(m.sessionInputTokens, m.sessionOutputTokens, m.sessionCacheCreationTokens, m.sessionCacheReadTokens, m.defaultProvider)

	if msg.Provider != "" {
		provCost := calculateCost(msg.InputTokens, msg.OutputTokens, msg.Provider)
		m.providerCosts[msg.Provider] += provCost
	}
}

// handleCompactMsg starts the compaction process
func (m Model) handleCompactMsg() (tea.Model, tea.Cmd) {
	p, ok := m.providers[m.defaultProvider]
	if !ok || !p.IsConfigured() {
		m.isCompacting = false
		m.status = "Cannot compact: provider not configured"
		return m, nil
	}

	llmMessages := m.buildLLMMessagesForCompaction()

	return m, func() tea.Msg {
		ctx := context.Background()
		compacted, err := llm.CompactMessages(ctx, p, llmMessages, llm.MinMessagesToKeep)
		return CompactDoneMsg{Messages: compacted, Error: err}
	}
}

// handleCompactDoneMsg replaces messages with the compacted version
func (m Model) handleCompactDoneMsg(msg CompactDoneMsg) (tea.Model, tea.Cmd) {
	m.isCompacting = false
	if msg.Error != nil {
		m.status = "Compaction failed: " + msg.Error.Error()
		return m, nil
	}

	oldCount := len(m.messages)

	newMessages := make([]Message, 0, len(msg.Messages))
	for _, lm := range msg.Messages {
		role := lm.Role
		provider := ""
		if role == "assistant" {
			provider = m.defaultProvider
		}
		newMessages = append(newMessages, Message{
			Role:     role,
			Content:  lm.Content,
			Provider: provider,
		})
	}

	m.messages = newMessages
	_ = session.SetMessages(m.toSessionMsgs())

	m.updateViewport()
	m.status = fmt.Sprintf("Context compacted (%d -> %d messages)", oldCount, len(m.messages))
	return m, nil
}

// toSessionMsgs converts TUI messages to session messages for persistence
func (m *Model) toSessionMsgs() []session.Message {
	sessionMsgs := make([]session.Message, len(m.messages))
	for i, msg := range m.messages {
		sessionMsgs[i] = m.toSessionMsg(msg)
	}
	return sessionMsgs
}

// toSessionMsg converts a single TUI message to a session message
func (m *Model) toSessionMsg(msg Message) session.Message {
	return session.Message{
		Role:         msg.Role,
		Content:      msg.Content,
		Provider:     msg.Provider,
		Thinking:     msg.Thinking,
		Images:       msg.Images,
		InputTokens:  msg.InputTokens,
		OutputTokens: msg.OutputTokens,
	}
}

// buildLLMMessagesForCompaction converts TUI messages to LLM messages
func (m *Model) buildLLMMessagesForCompaction() []llm.Message {
	llmMessages := make([]llm.Message, 0, len(m.messages))
	for _, msg := range m.messages {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		llmMessages = append(llmMessages, llm.Message{
			Role:    role,
			Content: msg.Content,
		})
	}
	return llmMessages
}
