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
	// If streaming was cancelled, drain remaining events without processing
	if !m.isStreaming {
		streamEventChan = nil
		return m, nil
	}
	if len(m.messages) > 0 {
		lastIdx := len(m.messages) - 1
		if msg.Error != nil {
			// Check if it's an image support error
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
		} else if msg.Done {
			// Track tokens
			m.updateTokenStats(msg)

			// Show response time + streaming stats
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
				m.statusBar.Update(status.SetStreamingMsg{
					Active: false,
				})
			}
			m.isStreaming = false
			m.streamTokenCount = 0
			m.saveLastMessage()
			m.updateViewport()
			m.syncStatusBar()

			// Desktop notification
			if m.notificationsOn {
				notify.Send("Poly", msg.Provider+" finished responding")
			}

			// Auto-compaction: check if context is getting too large
			llmMessages := m.buildLLMMessagesForCompaction()
			if llm.NeedsCompaction(llmMessages, 0) && !m.isCompacting {
				m.isCompacting = true
				m.status = "Auto-compacting context..."
				return m, func() tea.Msg { return CompactMsg{} }
			}

			return m, nil
		} else {
			// Handle tool_use: tool starting
			if msg.ToolCall != nil && msg.ToolResult == nil {
				toolIdx := len(m.messages[lastIdx].ToolCalls)
				m.messages[lastIdx].ToolCalls = append(m.messages[lastIdx].ToolCalls, ToolCallData{
					Name:   msg.ToolCall.Name,
					Args:   msg.ToolCall.Arguments,
					Status: 1, // running
				})
				m.messages[lastIdx].Blocks = append(m.messages[lastIdx].Blocks, ContentBlock{
					Type:    "tool",
					ToolIdx: toolIdx,
				})
				m.updateViewport()
				return m, readStreamEvent(msg.Provider)
			}

			// Handle tool_result: tool finished
			if msg.ToolCall != nil && msg.ToolResult != nil {
				for i := len(m.messages[lastIdx].ToolCalls) - 1; i >= 0; i-- {
					tc := &m.messages[lastIdx].ToolCalls[i]
					if tc.Name == msg.ToolCall.Name && tc.Status == 1 {
						tc.Result = msg.ToolResult.Content
						tc.IsError = msg.ToolResult.IsError
						if msg.ToolResult.IsError {
							tc.Status = 3 // error
						} else {
							tc.Status = 2 // success
						}
						break
					}
				}
				m.updateViewport()
				return m, readStreamEvent(msg.Provider)
			}

			if msg.Thinking != "" && m.thinkingMode {
				m.messages[lastIdx].Thinking += msg.Thinking
			}
			// Accumulate content (for persistence)
			m.messages[lastIdx].Content += msg.Content
			// Approximate token count (1 token ~ 4 chars)
			if msg.Content != "" {
				m.streamTokenCount += (len(msg.Content) + 3) / 4
			}
			// Track in ordered blocks (for inline rendering)
			if msg.Content != "" {
				blocks := m.messages[lastIdx].Blocks
				if len(blocks) > 0 && blocks[len(blocks)-1].Type == "text" {
					blocks[len(blocks)-1].Text += msg.Content
					m.messages[lastIdx].Blocks = blocks
				} else {
					m.messages[lastIdx].Blocks = append(m.messages[lastIdx].Blocks, ContentBlock{
						Type: "text",
						Text: msg.Content,
					})
				}
			}
			m.updateViewport()
			return m, readStreamEvent(msg.Provider)
		}
	}
	return m, nil
}

// handleCascadeStreamMsg processes a cascade streaming event
func (m Model) handleCascadeStreamMsg(msg CascadeStreamMsg) (tea.Model, tea.Cmd) {
	if m.cascade != nil {
		if idx, ok := m.cascade.messageIndices[msg.Provider]; ok {
			// Bounds check to prevent panic
			if idx < 0 || idx >= len(m.messages) {
				return m, nil
			}
			if msg.Error != nil {
				if llm.IsImageError(msg.Error) {
					llm.SetImageSupport(msg.Provider, false)
					m.messages[idx].Content = "(Images not supported)"
				} else {
					m.messages[idx].Content = "Error: " + msg.Error.Error()
				}
				if msg.Phase == CascadeResponder {
					m.status = "Ready"
					m.isStreaming = false
					m.cascade = nil
				} else {
					m.cascade.activeReviewers[msg.Provider] = false
				}
			} else if msg.Done {
				if msg.Phase == CascadeResponder {
					m.cascade.responderContent = m.messages[idx].Content
					m.cascade.phase = CascadeReviewer
					m.saveMessageAt(idx)
					m.updateViewport()
					return m, m.startReviewers()
				} else {
					m.cascade.activeReviewers[msg.Provider] = false
					m.saveMessageAt(idx)
				}
			} else {
				// Handle tool_use: tool starting
				if msg.ToolCall != nil && msg.ToolResult == nil {
					m.messages[idx].ToolCalls = append(m.messages[idx].ToolCalls, ToolCallData{
						Name:   msg.ToolCall.Name,
						Args:   msg.ToolCall.Arguments,
						Status: 1, // running
					})
					m.updateViewport()
					return m, readCascadeEvent(msg.Provider, msg.Phase)
				}

				// Handle tool_result: tool finished
				if msg.ToolCall != nil && msg.ToolResult != nil {
					for i := len(m.messages[idx].ToolCalls) - 1; i >= 0; i-- {
						tc := &m.messages[idx].ToolCalls[i]
						if tc.Name == msg.ToolCall.Name && tc.Status == 1 {
							tc.Result = msg.ToolResult.Content
							tc.IsError = msg.ToolResult.IsError
							if msg.ToolResult.IsError {
								tc.Status = 3 // error
							} else {
								tc.Status = 2 // success
							}
							break
						}
					}
					m.updateViewport()
					return m, readCascadeEvent(msg.Provider, msg.Phase)
				}

				if msg.Thinking != "" && m.thinkingMode {
					m.messages[idx].Thinking += msg.Thinking
				}
				m.messages[idx].Content += msg.Content
				if msg.Phase == CascadeResponder {
					m.cascade.responderContent += msg.Content
				}
				m.updateViewport()
				return m, readCascadeEvent(msg.Provider, msg.Phase)
			}
			// Check if all reviewers are done
			if m.cascade != nil && m.cascade.phase == CascadeReviewer {
				allDone := true
				for _, active := range m.cascade.activeReviewers {
					if active {
						allDone = false
						break
					}
				}
				if allDone {
					m.status = "Ready"
					m.isStreaming = false
					m.cascade = nil
				}
			}
			m.updateViewport()
		}
	}
	return m, nil
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
	// Calculate cost (with cache discount for Anthropic)
	m.sessionCost = calculateCostWithCache(m.sessionInputTokens, m.sessionOutputTokens, m.sessionCacheCreationTokens, m.sessionCacheReadTokens, m.defaultProvider)
}

// handleCompactMsg starts the compaction process in a goroutine
func (m Model) handleCompactMsg() (tea.Model, tea.Cmd) {
	p, ok := m.providers[m.defaultProvider]
	if !ok || !p.IsConfigured() {
		m.isCompacting = false
		m.status = "Cannot compact: provider not configured"
		return m, nil
	}

	// Build LLM messages from current chat
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

	// Convert LLM messages back to TUI messages
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

	// Re-persist the compacted session (use SetMessages to keep same session)
	sessionMsgs := make([]session.Message, len(m.messages))
	for i, msg := range m.messages {
		sessionMsgs[i] = session.Message{
			Role:     msg.Role,
			Content:  msg.Content,
			Provider: msg.Provider,
		}
	}
	session.SetMessages(sessionMsgs)

	m.updateViewport()
	m.status = fmt.Sprintf("Context compacted (%d -> %d messages)", oldCount, len(m.messages))
	return m, nil
}

// buildLLMMessagesForCompaction converts TUI messages to LLM messages for token estimation
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
