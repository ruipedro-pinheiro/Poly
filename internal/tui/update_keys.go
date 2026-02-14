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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// handleKeyMsg dispatches keyboard input to the appropriate view handler
func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Handle pasted text via msg.Text
	if msg.Text != "" && len(msg.Text) > 1 {
		if m.state == viewControlRoom && (m.oauthPending != "" || m.apiKeyPending != "") {
			m.authInput = strings.TrimSpace(msg.Text)
			return m, nil
		}
		if m.state == viewChat {
			m.textarea.InsertString(msg.Text)
			m.syncTextareaHeight()
			return m, nil
		}
		return m, nil
	}

	// Handle Ctrl+V
	if key.Matches(msg, m.keys.Paste) && m.state == viewChat && !m.isStreaming {
		if imgData, mediaType, ok := getClipboardImage(); ok {
			m.pendingImages = append(m.pendingImages, imgData)
			m.pendingImageTypes = append(m.pendingImageTypes, mediaType)
			m.status = fmt.Sprintf("[img] %d image(s) attached", len(m.pendingImages))
			return m, nil
		}
		text := getClipboardContent()
		if text != "" {
			m.textarea.InsertString(text)
			m.syncTextareaHeight()
		}
		return m, nil
	}

	// Splash screen
	if m.state == viewSplash {
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		m.state = viewChat
		return m, nil
	}

	// Approval dialog handling - MUST be before streaming guard
	// because tools request approval during streaming
	if m.state == viewApproval {
		return m.handleApprovalKey(msg)
	}

	// Don't process keys while streaming (except quit/cancel)
	if m.isStreaming && !key.Matches(msg, m.keys.Quit) && !key.Matches(msg, m.keys.Cancel) {
		return m, nil
	}

	// Session list handling
	if m.state == viewSessionList {
		newM, done := m.handleSessionListKey(msg.String())
		if done {
			return newM, nil
		}
		return newM, nil
	}

	// Command Palette input handling
	if m.state == viewCommandPalette {
		return m.handlePaletteKey(msg)
	}

	// Model Picker filter handling
	if m.state == viewModelPicker {
		return m.handleModelPickerKey(msg)
	}

	// Global keys
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.cancelCtx != nil {
			m.cancelCtx()
		}
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		if m.state == viewHelp {
			m.state = viewChat
		} else {
			m.state = viewHelp
		}
		return m, nil

	case key.Matches(msg, m.keys.ModelPicker):
		if m.state == viewModelPicker {
			m.state = viewChat
		} else {
			m.state = viewModelPicker
			m.modelPickerIndex = 0
			m.modelPickerFilter = ""
		}
		return m, nil

	case key.Matches(msg, m.keys.ControlRoom):
		if m.state == viewControlRoom {
			m.state = viewChat
			m.oauthPending = ""
			m.apiKeyPending = ""
			m.authInput = ""
			m.authStatusMsg = ""
		} else {
			m.state = viewControlRoom
			m.controlRoomIndex = 0
			m.authStatusMsg = ""
		}
		return m, nil

	case key.Matches(msg, m.keys.CommandPalette):
		if m.state == viewCommandPalette {
			m.state = viewChat
		} else {
			m.state = viewCommandPalette
			m.paletteFilter = ""
			m.paletteIndex = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.SessionList):
		if m.state == viewSessionList {
			m.state = viewChat
		} else {
			m.state = viewSessionList
			m.sessionListIndex = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.InfoPanel):
		m.infoPanelCmp.Toggle()
		m.syncInfoPanel()
		return m, nil

	case key.Matches(msg, m.keys.ThinkingToggle):
		m.thinkingMode = !m.thinkingMode
		if m.thinkingMode {
			m.modelVariant = "think"
			m.status = "Thinking mode ON"
		} else {
			m.modelVariant = "default"
			m.status = "Thinking mode OFF"
		}
		return m, nil

	case key.Matches(msg, m.keys.Clear):
		m.messages = []Message{}
		session.Clear()
		m.updateViewport()
		m.status = "Chat cleared"
		return m, nil

	case key.Matches(msg, m.keys.NewSession):
		m.messages = []Message{}
		session.Clear()
		m.sessionInputTokens = 0
		m.sessionOutputTokens = 0
		m.sessionCacheCreationTokens = 0
		m.sessionCacheReadTokens = 0
		m.sessionCost = 0
		m.modifiedFiles = nil
		m.updateViewport()
		m.status = "New session"
		return m, nil

	// Control Room navigation + Input history
	case key.Matches(msg, m.keys.Up):
		if m.state == viewChat && m.textarea.Line() == 0 && len(m.inputHistory) > 0 {
			if m.inputHistoryIdx == -1 {
				// Starting to browse: save current draft
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
		if m.state == viewControlRoom && m.oauthPending == "" && m.apiKeyPending == "" {
			if m.controlRoomIndex > 0 {
				m.controlRoomIndex--
			}
			return m, nil
		}

	case key.Matches(msg, m.keys.Down):
		if m.state == viewChat && m.inputHistoryIdx != -1 {
			lastLine := m.textarea.LineCount() - 1
			if lastLine < 0 {
				lastLine = 0
			}
			if m.textarea.Line() == lastLine {
				if m.inputHistoryIdx < len(m.inputHistory)-1 {
					m.inputHistoryIdx++
					m.textarea.SetValue(m.inputHistory[m.inputHistoryIdx])
				} else {
					// Back to draft
					m.inputHistoryIdx = -1
					m.textarea.SetValue(m.inputHistoryDraft)
					m.inputHistoryDraft = ""
				}
				m.textarea.CursorEnd()
				m.syncTextareaHeight()
				return m, nil
			}
		}
		if m.state == viewControlRoom && m.oauthPending == "" && m.apiKeyPending == "" {
			if m.controlRoomIndex < len(m.controlRoomProviders)-1 {
				m.controlRoomIndex++
			}
			return m, nil
		}

	case key.Matches(msg, m.keys.Disconnect):
		if m.state == viewControlRoom && m.oauthPending == "" && m.apiKeyPending == "" {
			provider := m.controlRoomProviders[m.controlRoomIndex]
			if auth.GetStorage().IsConnected(provider) {
				auth.GetStorage().RemoveAuth(provider)
				m.status = provider + " disconnected"
			}
			return m, nil
		}
	}

	// 'n' to add new provider in Control Room
	if m.state == viewControlRoom && m.oauthPending == "" && m.apiKeyPending == "" {
		if msg.String() == "n" || msg.String() == "N" {
			m.state = viewAddProvider
			m.addProviderField = 0
			m.addProviderValues = []string{"", "", "", ""}
			m.addProviderFormat = 0
			return m, nil
		}
	}

	// Add Provider form handling
	if m.state == viewAddProvider {
		return m.handleAddProviderKey(msg)
	}

	// Handle OAuth code or API key input
	if m.state == viewControlRoom && (m.oauthPending != "" || m.apiKeyPending != "") {
		return m.handleAuthInputKey(msg)
	}

	// Tab key in chat mode: completion or focus toggle
	if m.state == viewChat && key.Matches(msg, m.keys.Tab) && !m.isStreaming {
		if m.focused == "input" {
			// Try tab completion first
			input := m.textarea.Value()
			if m.completion.active || strings.HasPrefix(input, "/") || strings.Contains(input, "@") {
				return m.handleTabCompletion()
			}
			// No completion context: switch focus to messages
			m.focused = "messages"
			m.textarea.Blur()
			return m, nil
		}
		// In messages focus: switch back to input
		m.focused = "input"
		m.textarea.Focus()
		return m, nil
	}

	// Reset completion state on any non-Tab key in chat mode
	if m.state == viewChat && !key.Matches(msg, m.keys.Tab) {
		m.completion.active = false
	}

	// Message viewport navigation when focused on messages
	if m.state == viewChat && m.focused == "messages" {
		keyStr := msg.String()
		switch keyStr {
		case "j", "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k", "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "t":
			// Toggle thinking block expand/collapse for the last assistant message
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

	switch {
	case key.Matches(msg, m.keys.Cancel):
		if m.isStreaming && m.cancelCtx != nil {
			m.cancelCtx()
			m.isStreaming = false
			m.streamTokenCount = 0
			m.tableRonde = nil
			// Clean up Table Ronde stream channels
			for k := range tableRondeStreamChans {
				delete(tableRondeStreamChans, k)
			}
			m.statusBar.Update(status.SetStreamingMsg{Active: false})

			// Add cancel summary
			if len(m.messages) > 0 {
				lastIdx := len(m.messages) - 1
				lastMsg := m.messages[lastIdx]
				if lastMsg.Role == "assistant" {
					// Count approximate tokens generated
					tokenCount := (len(lastMsg.Content) + 3) / 4
					toolCount := 0
					for _, tc := range lastMsg.ToolCalls {
						if tc.Status == 2 { // ToolStatusSuccess
							toolCount++
						}
					}

					summary := fmt.Sprintf("Cancelled after ~%d tokens", tokenCount)
					if toolCount > 0 {
						summary += fmt.Sprintf(", %d tool(s) completed", toolCount)
					}
					m.status = summary
				} else {
					m.status = "Cancelled"
				}
			} else {
				m.status = "Cancelled"
			}
		}
		// Close info panel if visible (before other dismiss logic)
		if m.infoPanelCmp.IsVisible() && m.state == viewChat {
			m.infoPanelCmp.Toggle()
			return m, nil
		}
		if m.focused == "messages" {
			m.focused = "input"
			m.textarea.Focus()
			return m, nil
		}
		if m.oauthPending != "" || m.apiKeyPending != "" {
			m.oauthPending = ""
			m.apiKeyPending = ""
			m.authInput = ""
			return m, nil
		}
		m.state = viewChat
		return m, nil

	case key.Matches(msg, m.keys.Send):
		return m.handleSendKey()
	}

	// Default: update textarea in chat mode
	if m.state == viewChat && m.focused == "input" {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.syncTextareaHeight()
		return m, cmd
	}

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
			auth.GetStorage().SetAPIKey(m.apiKeyPending, apiKey)
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
		case "api_key":
			m.apiKeyPending = provider
			m.authInput = ""
			m.status = "Enter API key for " + provCfg.Name
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
	keyStr := msg.String()
	switch keyStr {
	case "tab", "down":
		m.addProviderField = (m.addProviderField + 1) % 5
		return m, nil
	case "shift+tab", "up":
		m.addProviderField = (m.addProviderField + 4) % 5
		return m, nil
	case "left":
		if m.addProviderField == 4 {
			m.addProviderFormat = (m.addProviderFormat + 2) % 3
		}
		return m, nil
	case "right":
		if m.addProviderField == 4 {
			m.addProviderFormat = (m.addProviderFormat + 1) % 3
		}
		return m, nil
	case "enter":
		if m.addProviderValues[0] != "" && m.addProviderValues[1] != "" && m.addProviderValues[3] != "" {
			formats := []string{"openai", "anthropic", "google"}
			apiKey := m.addProviderValues[2]
			cfg := llm.CustomProviderConfig{
				ID:        m.addProviderValues[0],
				Name:      cases.Title(language.English).String(m.addProviderValues[0]),
				BaseURL:   m.addProviderValues[1],
				APIKey:    apiKey,
				Model:     m.addProviderValues[3],
				Format:    formats[m.addProviderFormat],
				MaxTokens: 4096,
				Color:     "#888888",
			}
			if err := llm.SaveCustomProvider(cfg); err != nil {
				m.status = "Error: " + err.Error()
			} else {
				m.providers = llm.GetAllProviders()
				m.controlRoomProviders = llm.GetProviderNames()
				m.status = "Added @" + m.addProviderValues[0]
			}
		}
		m.state = viewControlRoom
		return m, nil
	case "esc":
		m.state = viewControlRoom
		return m, nil
	case "backspace":
		if m.addProviderField < 4 && len(m.addProviderValues[m.addProviderField]) > 0 {
			m.addProviderValues[m.addProviderField] = m.addProviderValues[m.addProviderField][:len(m.addProviderValues[m.addProviderField])-1]
		}
		return m, nil
	default:
		if m.addProviderField < 4 && len(keyStr) == 1 {
			m.addProviderValues[m.addProviderField] += keyStr
		}
		return m, nil
	}
}

// handleAuthInputKey handles key presses when entering OAuth code or API key
func (m Model) handleAuthInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	if keyStr == "enter" {
		return m.handleControlRoomEnter()
	}
	if keyStr == "esc" {
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
