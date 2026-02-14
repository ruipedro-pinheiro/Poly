package tui

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/mcp"
	"github.com/pedromelo/poly/internal/sandbox"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
	"github.com/pedromelo/poly/internal/tui/components/infopanel"
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

	case TableRondeStreamMsg:
		return m.handleTableRondeStreamMsg(msg)

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
		m.layout = ComputeLayout(m.width, m.height)
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
		m.infoPanelCmp.SetSize(infopanel.PanelWidth, m.height)
		m.updateViewport()
		m.syncStatusBar()
		m.syncInfoPanel()

		return m, nil
	}

	// Forward non-key messages to the add provider form (cursor blink, etc.)
	if m.state == viewAddProvider && m.addProviderForm != nil {
		cmd := m.addProviderForm.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
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
	m.headerBar.SetTokens(m.sessionInputTokens, m.sessionOutputTokens, m.sessionCost)
}

// syncInfoPanel pushes the current Model state into the info panel component
func (m *Model) syncInfoPanel() {
	m.infoPanelCmp.SetProvider(m.defaultProvider, theme.ProviderColor(m.defaultProvider))
	m.infoPanelCmp.SetThinkingMode(m.thinkingMode)
	m.infoPanelCmp.SetTokenInfo(
		m.sessionInputTokens, m.sessionOutputTokens,
		m.sessionCacheCreationTokens, m.sessionCacheReadTokens,
		m.sessionCost,
	)
	m.infoPanelCmp.SetYoloMode(tools.YoloMode)

	// Modified files
	files := make([]infopanel.ModifiedFile, len(m.modifiedFiles))
	for i, f := range m.modifiedFiles {
		files[i] = infopanel.ModifiedFile{Path: f}
	}
	m.infoPanelCmp.SetModifiedFiles(files)

	// Providers
	providerStatuses := make([]infopanel.ProviderStatus, 0, len(m.controlRoomProviders))
	for _, name := range m.controlRoomProviders {
		_, connected := m.providers[name]
		cost := m.providerCosts[name]
		hasPricing := false
		if p, ok := m.providers[name]; ok {
			hasPricing = llm.HasPricing(p.GetModel())
		}
		providerStatuses = append(providerStatuses, infopanel.ProviderStatus{
			Name:       name,
			Connected:  connected,
			Color:      theme.ProviderColor(name),
			Cost:       cost,
			HasPricing: hasPricing,
		})
	}
	m.infoPanelCmp.SetProviders(providerStatuses)

	// MCP servers
	if mcp.Global != nil {
		mcpStatuses := mcp.Global.Status()
		mcpServers := make([]infopanel.MCPServer, len(mcpStatuses))
		for i, s := range mcpStatuses {
			mcpServers[i] = infopanel.MCPServer{
				Name:      s.Name,
				Connected: s.Connected,
				ToolCount: s.ToolCount,
			}
		}
		m.infoPanelCmp.SetMCPServers(mcpServers)
	}

	// Sandbox mode
	m.infoPanelCmp.SetSandboxMode(sandbox.Enabled)
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
