package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/session"
	"github.com/pedromelo/poly/internal/tui/components/status"
)

// handleKeyMsg dispatches keyboard input to the appropriate view handler
func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 1. Handle special inputs like paste and raw text
	if cmd, done := m.handleRawInput(msg); done {
		return m, cmd
	}

	// 2. Handle view-specific modal keys
	if newM, cmd, done := m.handleViewSpecificKeys(msg); done {
		return newM, cmd
	}

	// 3. Streaming guard: don't process keys while streaming (except quit/cancel)
	if m.isStreaming && !key.Matches(msg, m.keys.Quit) && !key.Matches(msg, m.keys.Cancel) {
		return m, nil
	}

	// 4. Handle global shortcuts
	if newM, cmd, done := m.handleGlobalKeys(msg); done {
		return newM, cmd
	}

	// 5. Handle chat-specific navigation and actions
	return m.handleChatInputKeys(msg)
}

func (m *Model) handleRawInput(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	// Handle pasted text via msg.Text
	if msg.Text != "" && len(msg.Text) > 1 {
		if m.state == viewControlRoom && (m.oauthPending != "" || m.apiKeyPending != "") {
			m.authInput = strings.TrimSpace(msg.Text)
			return nil, true
		}
		if m.state == viewChat {
			m.textarea.InsertString(msg.Text)
			m.syncTextareaHeight()
			return nil, true
		}
		return nil, true
	}

	// Handle Ctrl+V
	if key.Matches(msg, m.keys.Paste) && m.state == viewChat && !m.isStreaming {
		if imgData, mediaType, ok := getClipboardImage(); ok {
			m.pendingImages = append(m.pendingImages, imgData)
			m.pendingImageTypes = append(m.pendingImageTypes, mediaType)
			m.status = fmt.Sprintf("[img] %d image(s) attached", len(m.pendingImages))
			return nil, true
		}
		text := getClipboardContent()
		if text != "" {
			m.textarea.InsertString(text)
			m.syncTextareaHeight()
		}
		return nil, true
	}
	return nil, false
}

func (m Model) handleViewSpecificKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch m.state {
	case viewSplash:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit, true
		}
		m.state = viewChat
		return m, nil, true

	case viewApproval:
		newM, cmd := m.handleApprovalKey(msg)
		return newM, cmd, true

	case viewSessionList:
		newM, _ := m.handleSessionListKey(msg.String())
		return newM, nil, true

	case viewCommandPalette:
		newM, cmd := m.handlePaletteKey(msg)
		return newM, cmd, true

	case viewModelPicker:
		newM, cmd := m.handleModelPickerKey(msg)
		return newM, cmd, true

	case viewAddProvider:
		newM, cmd := m.handleAddProviderKey(msg)
		return newM, cmd, true
	}

	// Sub-state in Control Room
	if m.state == viewControlRoom && (m.oauthPending != "" || m.apiKeyPending != "") {
		newM, cmd := m.handleAuthInputKey(msg)
		return newM, cmd, true
	}

	return m, nil, false
}

func (m Model) handleGlobalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.cancelCtx != nil {
			m.cancelCtx()
		}
		return m, tea.Quit, true

	case key.Matches(msg, m.keys.Help):
		if m.state == viewHelp {
			m.state = viewChat
		} else {
			m.state = viewHelp
		}
		return m, nil, true

	case key.Matches(msg, m.keys.ModelPicker):
		m.state = viewModelPicker
		m.modelPickerIndex = 0
		m.modelPickerFilter = ""
		return m, nil, true

	case key.Matches(msg, m.keys.ControlRoom):
		m.state = viewControlRoom
		m.controlRoomIndex = 0
		m.authStatusMsg = ""
		return m, nil, true

	case key.Matches(msg, m.keys.CommandPalette):
		m.state = viewCommandPalette
		m.paletteFilter = ""
		m.paletteIndex = 0
		return m, nil, true

	case key.Matches(msg, m.keys.SessionList):
		m.state = viewSessionList
		m.sessionListIndex = 0
		return m, nil, true

	case key.Matches(msg, m.keys.InfoPanel):
		m.infoPanelCmp.Toggle()
		m.syncInfoPanel()
		return m, nil, true

	case key.Matches(msg, m.keys.ThinkingToggle):
		m.thinkingMode = !m.thinkingMode
		m.modelVariant = "default"
		if m.thinkingMode {
			m.modelVariant = "think"
		}
		m.status = fmt.Sprintf("Thinking mode %s", map[bool]string{true: "ON", false: "OFF"}[m.thinkingMode])
		return m, nil, true

	case key.Matches(msg, m.keys.Clear), key.Matches(msg, m.keys.NewSession):
		m.messages = []Message{}
		_ = session.Clear()
		if key.Matches(msg, m.keys.NewSession) {
			m.sessionInputTokens, m.sessionOutputTokens = 0, 0
			m.sessionCacheCreationTokens, m.sessionCacheReadTokens = 0, 0
			m.sessionCost = 0
			m.modifiedFiles = nil
			m.status = "New session"
		} else {
			m.status = "Chat cleared"
		}
		m.updateViewport()
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) handleChatInputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// History and Control Room navigation
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.state == viewChat && m.textarea.Line() == 0 && len(m.inputHistory) > 0 {
			if m.inputHistoryIdx == -1 {
				m.inputHistoryDraft = m.textarea.Value()
				m.inputHistoryIdx = len(m.inputHistory) - 1
			} else if m.inputHistoryIdx > 0 {
				m.inputHistoryIdx--
			}
			m.textarea.SetValue(m.inputHistory[m.inputHistoryIdx])
			m.textarea.CursorEnd()
			m.syncTextareaHeight()
			return m, nil
		}
		if m.state == viewControlRoom && m.controlRoomIndex > 0 {
			m.controlRoomIndex--
			return m, nil
		}

	case key.Matches(msg, m.keys.Down):
		if m.state == viewChat && m.inputHistoryIdx != -1 {
			if m.textarea.Line() == m.textarea.LineCount()-1 {
				if m.inputHistoryIdx < len(m.inputHistory)-1 {
					m.inputHistoryIdx++
					m.textarea.SetValue(m.inputHistory[m.inputHistoryIdx])
				} else {
					m.inputHistoryIdx = -1
					m.textarea.SetValue(m.inputHistoryDraft)
					m.inputHistoryDraft = ""
				}
				m.textarea.CursorEnd()
				m.syncTextareaHeight()
				return m, nil
			}
		}
		if m.state == viewControlRoom && m.controlRoomIndex < len(m.controlRoomProviders)-1 {
			m.controlRoomIndex++
			return m, nil
		}

	case key.Matches(msg, m.keys.Disconnect):
		if m.state == viewControlRoom {
			provider := m.controlRoomProviders[m.controlRoomIndex]
			if auth.GetStorage().IsConnected(provider) {
				_ = auth.GetStorage().RemoveAuth(provider)
				m.status = provider + " disconnected"
			}
			return m, nil
		}
	}

	// Add new provider in Control Room
	if m.state == viewControlRoom && (msg.String() == "n" || msg.String() == "N") {
		m.state = viewAddProvider
		f := newAddProviderForm(dialogWidth(46, m.width, 36))
		m.addProviderForm = &f
		return m, m.addProviderForm.Init()
	}

	switch {
	case key.Matches(msg, m.keys.Cancel):
		return m.handleCancelAction()

	case key.Matches(msg, m.keys.Send):
		return m.handleSendKey()

	case key.Matches(msg, m.keys.Tab):
		if m.state == viewChat && !m.isStreaming {
			if m.focused == "input" {
				input := m.textarea.Value()
				if m.completion.active || strings.HasPrefix(input, "/") || strings.Contains(input, "@") {
					return m.handleTabCompletion()
				}
				m.focused = "messages"
				m.textarea.Blur()
				return m, nil
			}
			m.focused = "input"
			m.textarea.Focus()
			return m, nil
		}
	}

	// Viewport navigation
	if m.state == viewChat && m.focused == "messages" {
		switch msg.String() {
		case "j", "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k", "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "t":
			if len(m.messages) > 0 {
				lastIdx := len(m.messages) - 1
				if m.messages[lastIdx].Thinking != "" {
					m.thinkingExpanded[lastIdx] = !m.thinkingExpanded[lastIdx]
					m.updateViewport()
				}
			}
			return m, nil
		case "enter":
			m.focused = "input"
			m.textarea.Focus()
			return m, nil
		}
	}

	// Reset completion state
	if m.state == viewChat && !key.Matches(msg, m.keys.Tab) {
		m.completion.active = false
	}

	// Fallback: update textarea
	if m.state == viewChat && m.focused == "input" {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.syncTextareaHeight()
		return m, cmd
	}

	return m, nil
}

func (m Model) handleCancelAction() (tea.Model, tea.Cmd) {
	if m.isStreaming && m.cancelCtx != nil {
		m.cancelCtx()
		m.isStreaming = false
		m.streamTokenCount = 0
		m.tableRonde = nil
		for k := range tableRondeStreamChans {
			delete(tableRondeStreamChans, k)
		}
		m.statusBar.Update(status.SetStreamingMsg{Active: false})

		if len(m.messages) > 0 {
			lastMsg := m.messages[len(m.messages)-1]
			if lastMsg.Role == "assistant" {
				tokenCount := (len(lastMsg.Content) + 3) / 4
				m.status = fmt.Sprintf("Cancelled after ~%d tokens", tokenCount)
			} else {
				m.status = "Cancelled"
			}
		} else {
			m.status = "Cancelled"
		}
	}

	if m.infoPanelCmp.IsVisible() && m.state == viewChat {
		m.infoPanelCmp.Toggle()
		return m, nil
	}
	if m.focused == "messages" {
		m.focused = "input"
		m.textarea.Focus()
		return m, nil
	}
	m.state = viewChat
	return m, nil
}

// handleSendKey processes Enter key - sends message or executes control room action
func (m Model) handleSendKey() (tea.Model, tea.Cmd) {
	// Handle Control Room actions
	if m.state == viewControlRoom {
		return m.handleControlRoomEnter()
	}

	if m.state == viewChat && (strings.TrimSpace(m.textarea.Value()) != "" || len(m.pendingImages) > 0) && !m.isStreaming {
		content := strings.TrimSpace(m.textarea.Value())

		// Handle commands (start with /)
		if strings.HasPrefix(content, "/") {
			// Save command to input history
			m.inputHistory = append(m.inputHistory, content)
			if len(m.inputHistory) > 100 {
				m.inputHistory = m.inputHistory[len(m.inputHistory)-100:]
			}
			m.inputHistoryIdx = -1
			m.inputHistoryDraft = ""
			config.AddHistory(content)

			m.textarea.Reset()
			m.handleCommand(content)
			// If compact was requested, trigger it
			if m.isCompacting {
				return m, func() tea.Msg { return CompactMsg{} }
			}
			// If retry was requested, re-send the saved content
			if m.retryContent != "" {
				m.textarea.SetValue(m.retryContent)
				m.retryContent = ""
				return m.handleSendKey()
			}
			// If a skill was triggered, send its content as a user message
			if m.skillContent != "" {
				m.textarea.SetValue(m.skillContent)
				m.skillContent = ""
				return m.handleSendKey()
			}
			// If compare was requested, launch parallel provider queries
			if len(m.comparePending) > 0 {
				cmds := m.comparePending
				m.comparePending = nil
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

		// Parse @file mentions and enrich content with file contents
		enrichedContent, includedFiles := ParseFileMentions(content)

		// Prepend persistent context files (re-read fresh each time)
		contextPrefix := m.buildContextPrefix()
		if contextPrefix != "" {
			enrichedContent = contextPrefix + enrichedContent
		}

		// Create user message with any pending images
		// Display shows original content, LLM gets enriched content with file data
		userMsg := Message{
			Role:       "user",
			Content:    enrichedContent,
			ImageData:  m.pendingImages,
			ImageTypes: m.pendingImageTypes,
		}
		m.addMessage(userMsg)

		// Clear pending images
		m.pendingImages = nil
		m.pendingImageTypes = nil

		provider := m.parseProvider(content)

		// Add empty assistant message that will be filled by streaming
		m.messages = append(m.messages, Message{
			Role:     "assistant",
			Content:  "",
			Provider: provider,
		})

		// Save to input history
		if content != "" {
			m.inputHistory = append(m.inputHistory, content)
			if len(m.inputHistory) > 100 {
				m.inputHistory = m.inputHistory[len(m.inputHistory)-100:]
			}
			m.inputHistoryIdx = -1
			m.inputHistoryDraft = ""
			config.AddHistory(content)
		}

		m.textarea.Reset()
		m.syncTextareaHeight()
		m.updateViewport()

		// Start streaming
		m.isStreaming = true
		m.streamStartTime = time.Now()
		m.streamTokenCount = 0
		if len(includedFiles) > 0 {
			m.status = fmt.Sprintf("Working... (%d file(s) attached)", len(includedFiles))
		} else {
			m.status = "Working..."
		}
		// Start the streaming tick for live elapsed/speed display
		tickCmd := tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
			return StreamTickMsg{}
		})
		return m, tea.Batch(m.sendToProvider(provider, enrichedContent), tickCmd)
	}
	return m, nil
}

// handleControlRoomEnter processes Enter key in the Control Room
func (m Model) handleControlRoomEnter() (tea.Model, tea.Cmd) {
	// Handle OAuth code submission (async — don't block the TUI)
	if m.oauthPending != "" {
		code := m.authInput
		if code == "" {
			code = getClipboardContent()
		}
		if code != "" {
			provider := m.oauthPending
			m.authStatusMsg = "Exchanging code..."
			return m, startAnthropicCodeExchange(provider, code)
		}
		m.authStatusMsg = "No code provided"
		return m, nil
	}

	// Handle API key submission
	if m.apiKeyPending != "" {
		apiKey := m.authInput
		if apiKey == "" {
			apiKey = getClipboardContent()
		}
		if apiKey != "" {
			_ = auth.GetStorage().SetAPIKey(m.apiKeyPending, apiKey)
			m.status = m.apiKeyPending + " connected!"
			m.apiKeyPending = ""
			m.authInput = ""
			m.authStatusMsg = ""
		} else {
			m.authStatusMsg = "No API key provided"
		}
		return m, nil
	}

	// Start auth for selected provider (config-driven)
	provider := m.controlRoomProviders[m.controlRoomIndex]
	if !auth.GetStorage().IsConnected(provider) {
		cfg := config.Get()
		provCfg, ok := cfg.Providers[provider]
		if !ok {
			m.status = "Unknown provider: " + provider
			return m, nil
		}
		switch provCfg.AuthType {
		case "oauth":
			// Claude uses PKCE code-paste flow, others use callback-based OAuth
			if provider == "claude" {
				_, err := auth.StartAnthropicOAuth("max")
				if err != nil {
					m.status = "OAuth error: " + err.Error()
				} else {
					m.oauthPending = provider
					m.authInput = ""
					m.status = "Browser opened - copy code"
				}
			} else {
				m.oauthPending = provider
				m.status = "Opening browser for " + provCfg.Name + "..."
				return m, startOAuthForProvider(provider)
			}
		case "device_flow":
			m.oauthPending = provider
			m.status = "Starting GitHub device flow..."
			return m, startDeviceFlow(provider)
		case "api_key":
			m.apiKeyPending = provider
			m.authInput = ""
			m.status = "Enter API key for " + provCfg.Name
		case "none":
			// Local providers (e.g. Ollama) need no authentication — just set as default
			m.defaultProvider = provider
			m.status = provider + " set as default (no auth needed)"
		default:
			m.apiKeyPending = provider
			m.authInput = ""
			m.status = "Enter credentials for " + provCfg.Name
		}
	} else {
		m.defaultProvider = provider
		m.status = provider + " set as default"
	}
	return m, nil
}

// handleAddProviderKey handles key presses in the add provider form
func (m Model) handleAddProviderKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.addProviderForm == nil {
		m.state = viewControlRoom
		return m, nil
	}

	keyStr := msg.String()
	switch keyStr {
	case "esc":
		m.addProviderForm = nil
		m.state = viewControlRoom
		return m, nil

	case "tab", "down":
		cmd := m.addProviderForm.NextField()
		return m, cmd

	case "shift+tab", "up":
		cmd := m.addProviderForm.PrevField()
		return m, cmd

	case "left", "h":
		if m.addProviderForm.focusIndex == apFieldFormat {
			m.addProviderForm.CycleFormat(-1)
			return m, nil
		}
		// Fall through to forward to textinput
		cmd := m.addProviderForm.Update(msg)
		return m, cmd

	case "right", "l":
		if m.addProviderForm.focusIndex == apFieldFormat {
			m.addProviderForm.CycleFormat(1)
			return m, nil
		}
		cmd := m.addProviderForm.Update(msg)
		return m, cmd

	case "enter":
		err := m.addProviderForm.SaveProvider()
		if err != nil {
			m.status = "Error: " + err.Error()
		} else if m.addProviderForm.Completed() {
			m.providers = llm.GetAllProviders()
			m.controlRoomProviders = llm.GetProviderNames()
			m.status = "Added @" + m.addProviderForm.ProviderID()
		}
		m.addProviderForm = nil
		m.state = viewControlRoom
		return m, nil

	default:
		// Forward to the focused textinput
		cmd := m.addProviderForm.Update(msg)
		return m, cmd
	}
}

// handleAuthInputKey handles key presses when entering OAuth code or API key
func (m Model) handleAuthInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	if keyStr == "enter" {
		return m.handleControlRoomEnter()
	}
	if keyStr == "esc" {
		cancelDeviceFlow() // cancel any in-progress device flow polling
		m.oauthPending = ""
		m.apiKeyPending = ""
		m.authInput = ""
		return m, nil
	}
	if len(keyStr) == 1 {
		m.authInput += keyStr
		return m, nil
	}
	if keyStr == "backspace" && len(m.authInput) > 0 {
		m.authInput = m.authInput[:len(m.authInput)-1]
		return m, nil
	}
	if keyStr == "ctrl+v" || keyStr == "ctrl+p" {
		m.authInput = getClipboardContent()
		return m, nil
	}
	return m, nil
}

// handlePaletteKey handles key presses in the command palette
func (m Model) handlePaletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = viewChat
		return m, nil

	case key.Matches(msg, m.keys.Send):
		// Execute selected command
		filtered := m.filteredPaletteCommands()
		if m.paletteIndex < len(filtered) {
			cmd := filtered[m.paletteIndex]
			m.state = viewChat
			if cmd.Action != nil {
				cmd.Action(&m)
			} else if cmd.Name == "Quit" {
				return m, tea.Quit
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.paletteIndex > 0 {
			m.paletteIndex--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		filtered := m.filteredPaletteCommands()
		if m.paletteIndex < len(filtered)-1 {
			m.paletteIndex++
		}
		return m, nil

	case keyStr == "backspace":
		if len(m.paletteFilter) > 0 {
			m.paletteFilter = m.paletteFilter[:len(m.paletteFilter)-1]
			m.paletteIndex = 0
		}
		return m, nil

	default:
		if len(keyStr) == 1 {
			m.paletteFilter += keyStr
			m.paletteIndex = 0
		}
		return m, nil
	}
}

// handleModelPickerKey handles key presses in the model picker
func (m Model) handleModelPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.state = viewChat
		m.modelPickerFilter = ""
		return m, nil

	case key.Matches(msg, m.keys.Send):
		filtered := m.filteredModelPickerModels()
		if m.modelPickerIndex < len(filtered) {
			selected := filtered[m.modelPickerIndex]
			m.defaultProvider = selected.provider
			m.modelVariant = selected.variant
			if selected.variant == "think" {
				m.thinkingMode = true
			} else {
				m.thinkingMode = false
			}
			// Track recent
			m.addRecentModel(selected)
			m.status = "Model: " + selected.display
			m.state = viewChat
			m.modelPickerFilter = ""
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.modelPickerIndex > 0 {
			m.modelPickerIndex--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		filtered := m.filteredModelPickerModels()
		if m.modelPickerIndex < len(filtered)-1 {
			m.modelPickerIndex++
		}
		return m, nil

	case keyStr == "backspace":
		if len(m.modelPickerFilter) > 0 {
			m.modelPickerFilter = m.modelPickerFilter[:len(m.modelPickerFilter)-1]
			m.modelPickerIndex = 0
		}
		return m, nil

	default:
		if len(keyStr) == 1 {
			m.modelPickerFilter += keyStr
			m.modelPickerIndex = 0
		}
		return m, nil
	}
}

// addRecentModel adds a model to the recently used list
func (m *Model) addRecentModel(model modelOption) {
	// Remove if already in list
	filtered := make([]modelOption, 0, len(m.recentModels))
	for _, r := range m.recentModels {
		if r.provider != model.provider || r.variant != model.variant {
			filtered = append(filtered, r)
		}
	}
	// Prepend
	m.recentModels = append([]modelOption{model}, filtered...)
	// Keep max 3
	if len(m.recentModels) > 3 {
		m.recentModels = m.recentModels[:3]
	}
}

// handleTabCompletion processes Tab key for auto-completing slash commands and @mentions
func (m *Model) handleTabCompletion() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()
	cursorPos := m.textarea.Column()

	if m.completion.active {
		// Already in a completion cycle: cycle to next candidate
		m.completion.index = (m.completion.index + 1) % len(m.completion.candidates)
		candidate := m.completion.candidates[m.completion.index]

		// Replace the prefix in the input with the current candidate
		newInput := input[:m.completion.startPos] + candidate + input[m.completion.startPos+len(m.completion.prefix):]
		m.completion.prefix = candidate
		m.textarea.SetValue(newInput)
		m.textarea.SetCursorColumn(m.completion.startPos + len(candidate))

		if len(m.completion.candidates) > 1 {
			m.status = fmt.Sprintf("(%d/%d) %s",
				m.completion.index+1, len(m.completion.candidates),
				strings.Join(m.completion.candidates, "  "))
		}
		return m, nil
	}

	// Start a new completion
	candidates, prefix, startPos := m.getCompletions(input, cursorPos)
	if len(candidates) == 0 {
		return m, nil
	}

	if len(candidates) == 1 {
		// Single match: complete directly and add a trailing space
		completed := candidates[0] + " "
		newInput := input[:startPos] + completed + input[startPos+len(prefix):]
		m.textarea.SetValue(newInput)
		m.textarea.SetCursorColumn(startPos + len(completed))
		m.completion.active = false
		return m, nil
	}

	// Multiple candidates: start cycling
	m.completion = completionState{
		active:     true,
		candidates: candidates,
		prefix:     candidates[0],
		index:      0,
		startPos:   startPos,
	}

	// Replace prefix with first candidate
	newInput := input[:startPos] + candidates[0] + input[startPos+len(prefix):]
	m.textarea.SetValue(newInput)
	m.textarea.SetCursorColumn(startPos + len(candidates[0]))

	m.status = fmt.Sprintf("(%d/%d) %s",
		1, len(candidates),
		strings.Join(candidates, "  "))

	return m, nil
}
