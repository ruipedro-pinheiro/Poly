package tui

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
	"github.com/pedromelo/poly/internal/tui/components/dialogs"
	"github.com/pedromelo/poly/internal/tui/components/status"
	tuiLayout "github.com/pedromelo/poly/internal/tui/layout"
)

// Update is the main Bubble Tea update dispatcher.
// It routes messages to focused handlers in update_keys.go and update_stream.go.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// --- Status bar messages ---
	case status.InfoMsg, status.ClearStatusMsg:
		sm, cmd := m.statusBar.Update(msg)
		m.statusBar = sm.(status.StatusCmp)
		return m, cmd

	// --- Dialog messages ---
	case dialogs.OpenDialogMsg:
		dm, cmd := m.dialogMgr.Update(msg)
		m.dialogMgr = dm.(dialogs.DialogCmp)
		return m, cmd

	case dialogs.CloseDialogMsg:
		dm, cmd := m.dialogMgr.Update(msg)
		m.dialogMgr = dm.(dialogs.DialogCmp)
		return m, cmd

	case dialogs.CloseAllDialogsMsg:
		dm, cmd := m.dialogMgr.Update(msg)
		m.dialogMgr = dm.(dialogs.DialogCmp)
		return m, cmd

	case dialogs.DialogClosedMsg:
		return m.handleDialogClosed(msg)

	// --- Tool approval ---
	case ToolPendingMsg:
		if m.approvedTools[msg.Approval.Name] {
			tools.ApprovedChan <- true
			return m, watchForApprovals()
		}
		m.pendingApproval = msg.Approval
		m.state = viewApproval
		return m, nil

	// --- Streaming ---
	case StreamMsg:
		return m.handleStreamMsg(msg)

	case StreamTickMsg:
		if m.isStreaming && !m.streamStartTime.IsZero() {
			elapsed := time.Since(m.streamStartTime)
			m.statusBar.Update(status.SetStreamingMsg{
				Active:  true,
				Elapsed: elapsed,
				Tokens:  m.streamTokenCount,
			})
			return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return StreamTickMsg{}
			})
		}
		return m, nil

	case CascadeStreamMsg:
		return m.handleCascadeStreamMsg(msg)

	// --- Compaction ---
	case CompactMsg:
		return m.handleCompactMsg()

	case CompactDoneMsg:
		return m.handleCompactDoneMsg(msg)

	// --- Misc messages ---
	case UpdateAvailableMsg:
		if msg.Version != "" {
			sm, cmd := m.statusBar.Update(status.InfoMsg{
				Type: status.InfoTypeInfo,
				Msg:  "Update available: v" + msg.Version,
				TTL:  10 * time.Second,
			})
			m.statusBar = sm.(status.StatusCmp)
			return m, cmd
		}
		return m, nil

	case OAuthResultMsg:
		if msg.Success {
			m.status = msg.Provider + " connected!"
			if m.oauthPending == msg.Provider {
				m.oauthPending = ""
				m.authInput = ""
				m.authStatusMsg = ""
			}
		} else {
			m.status = "OAuth error: " + msg.Error
			m.authStatusMsg = "Error: " + msg.Error
		}
		return m, nil

	case CompareResultMsg:
		cmd := m.handleCompareResult(msg)
		return m, cmd

	// --- Keyboard input ---
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)

	// --- Window resize ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = ComputeLayout(m.width, m.height, m.sidebarVisible)
		initMarkdown(m.layout.ContentWidth)

		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(m.layout.ViewportWidth), viewport.WithHeight(m.layout.ChatHeight))
			m.viewport.KeyMap.Up.SetEnabled(false)
			m.viewport.KeyMap.Down.SetEnabled(false)
			m.viewport.KeyMap.PageUp.SetEnabled(false)
			m.viewport.KeyMap.PageDown.SetEnabled(false)
			m.viewport.KeyMap.HalfPageUp.SetEnabled(false)
			m.viewport.KeyMap.HalfPageDown.SetEnabled(false)
			m.ready = true
		} else {
			m.viewport.SetWidth(m.layout.ViewportWidth)
			m.viewport.SetHeight(m.layout.ChatHeight)
		}

		m.textarea.SetWidth(m.width - tuiLayout.InputBoxPadding)
		m.headerBar.SetSize(m.width, tuiLayout.HeaderHeight)
		m.headerBar.SetProvider(m.defaultProvider, theme.ProviderColor(m.defaultProvider))
		cwd, _ := os.Getwd()
		if home, _ := os.UserHomeDir(); home != "" && strings.HasPrefix(cwd, home) {
			cwd = "~" + cwd[len(home):]
		}
		m.headerBar.SetCwd(cwd)
		m.statusBar.SetWidth(m.width)
		m.dialogMgr.SetSize(m.width, m.height)
		m.updateViewport()
		m.syncStatusBar()

		return m, nil
	}

	// Default: forward to textarea/viewport in chat mode
	if m.state == viewChat {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
		m.syncTextareaHeight()

		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleDialogClosed processes results from dialog components
func (m Model) handleDialogClosed(msg dialogs.DialogClosedMsg) (tea.Model, tea.Cmd) {
	// Results from new dialog components will be handled here
	// as dialogs are migrated from the old viewState system.
	// For now, this is a no-op placeholder.
	return m, nil
}

// syncStatusBar pushes the current Model state into the status bar component
func (m *Model) syncStatusBar() {
	m.statusBar.Update(status.SetProviderMsg{
		Name:  m.defaultProvider,
		Color: theme.ProviderColor(m.defaultProvider),
	})
	m.statusBar.Update(status.SetTokensMsg{
		Input:         m.sessionInputTokens,
		Output:        m.sessionOutputTokens,
		CacheCreation: m.sessionCacheCreationTokens,
		CacheRead:     m.sessionCacheReadTokens,
		Cost:          m.sessionCost,
	})
}

// setStatus sets a status message on both the legacy field and the component
func (m *Model) setStatus(msg string) {
	m.status = msg
	m.statusBar.Update(status.InfoMsg{
		Type: status.InfoTypeInfo,
		Msg:  msg,
	})
}

// calculateCost calculates the session cost using the pricing table
func calculateCost(inputTokens, outputTokens int, provider string) float64 {
	p, ok := llm.GetProvider(provider)
	if ok {
		return llm.CalculateCost(inputTokens, outputTokens, p.GetModel())
	}
	return llm.CalculateCost(inputTokens, outputTokens, provider)
}

// calculateCostWithCache calculates session cost accounting for prompt caching
func calculateCostWithCache(inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int, provider string) float64 {
	if cacheCreationTokens == 0 && cacheReadTokens == 0 {
		return calculateCost(inputTokens, outputTokens, provider)
	}
	p, ok := llm.GetProvider(provider)
	if ok {
		return llm.CalculateCostWithCache(inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens, p.GetModel())
	}
	return llm.CalculateCostWithCache(inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens, provider)
}
