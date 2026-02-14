package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/tools"
)

// streamEventChan stores the current streaming channel (protected by streamMu)
var (
	streamEventChan <-chan llm.StreamEvent
	streamMu        sync.Mutex
)

// tableRondeStreamChans stores channels for @all Table Ronde streaming (protected by trChansMu)
var (
	tableRondeStreamChans   = make(map[string]<-chan llm.StreamEvent)
	trChansMu               sync.RWMutex
)

// getStreamChan safely reads a channel from the Table Ronde map
func getStreamChan(key string) (<-chan llm.StreamEvent, bool) {
	trChansMu.RLock()
	defer trChansMu.RUnlock()
	ch, ok := tableRondeStreamChans[key]
	return ch, ok
}

// setStreamChan safely writes a channel to the Table Ronde map
func setStreamChan(key string, ch <-chan llm.StreamEvent) {
	trChansMu.Lock()
	defer trChansMu.Unlock()
	tableRondeStreamChans[key] = ch
}

// clearStreamChans safely clears all Table Ronde channels
func clearStreamChans() {
	trChansMu.Lock()
	defer trChansMu.Unlock()
	for k := range tableRondeStreamChans {
		delete(tableRondeStreamChans, k)
	}
}

// getToolDefinitions converts tools to LLM format
func getToolDefinitions() []llm.ToolDefinition {
	toolDefs := tools.GetDefinitions()
	result := make([]llm.ToolDefinition, len(toolDefs))
	for i, t := range toolDefs {
		result[i] = llm.ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	return result
}

// convertImages loads image files and converts them to LLM format
func convertImages(paths []string) []llm.Image {
	if len(paths) == 0 {
		return nil
	}

	images := make([]llm.Image, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip unreadable images
		}

		// Detect media type from extension
		mediaType := "image/png" // default
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg":
			mediaType = "image/jpeg"
		case ".gif":
			mediaType = "image/gif"
		case ".webp":
			mediaType = "image/webp"
		}

		images = append(images, llm.Image{
			Data:      data,
			MediaType: mediaType,
			Path:      path,
		})
	}

	return images
}

// convertMessageImages converts both file paths and raw image data to LLM format
func convertMessageImages(msg Message) []llm.Image {
	var images []llm.Image

	// Add images from file paths
	images = append(images, convertImages(msg.Images)...)

	// Add images from raw data (pasted images)
	for i, data := range msg.ImageData {
		mediaType := "image/png"
		if i < len(msg.ImageTypes) {
			mediaType = msg.ImageTypes[i]
		}
		images = append(images, llm.Image{
			Data:      data,
			MediaType: mediaType,
			Path:      fmt.Sprintf("pasted-image-%d", i+1),
		})
	}

	return images
}

// sendToProvider sends a message to the specified provider and starts streaming
func (m *Model) sendToProvider(providerName string, content string) tea.Cmd {
	// Handle @all - Table Ronde orchestration (all providers in parallel)
	if providerName == "all" {
		return m.sendTableRonde(content)
	}

	p, ok := m.providers[providerName]
	if !ok {
		return func() tea.Msg {
			return StreamMsg{
				Content:  "Provider not found: " + providerName + ". Connect it in Control Room (Ctrl+D).",
				Done:     true,
				Provider: providerName,
			}
		}
	}

	if !p.IsConfigured() {
		return func() tea.Msg {
			return StreamMsg{
				Content:  "Provider not configured. Connect it in Control Room (Ctrl+D).",
				Done:     true,
				Provider: providerName,
			}
		}
	}

	// Apply model variant
	variant := m.modelVariant
	if variant == "" {
		variant = "default"
	}
	if models, ok := llm.GetModelVariants()[providerName]; ok {
		if model, ok := models[variant]; ok {
			p.SetModel(model)
		}
	}

	// Convert messages to LLM format, filtering out empty content
	// Only include images if provider supports them (auto-detected)
	supportsImages := llm.SupportsImages(providerName)
	llmMessages := make([]llm.Message, 0, len(m.messages)-1)
	for _, msg := range m.messages[:len(m.messages)-1] { // Exclude the empty assistant message
		// Skip messages with empty content and no images (API rejects them)
		if strings.TrimSpace(msg.Content) == "" && len(msg.Images) == 0 && len(msg.ImageData) == 0 {
			continue
		}
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		llmMsg := llm.Message{
			Role:    role,
			Content: msg.Content,
		}
		// Only include images if provider supports them
		if supportsImages {
			llmMsg.Images = convertMessageImages(msg)
		}
		llmMessages = append(llmMessages, llmMsg)
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCtx = cancel

	// Get tools if provider supports them
	var toolDefs []llm.ToolDefinition
	if p.SupportsTools() {
		toolDefs = getToolDefinitions()
	}

	// Start streaming
	streamMu.Lock()
	streamEventChan = p.Stream(ctx, llmMessages, toolDefs, llm.StreamOptions{ThinkingMode: m.thinkingMode})
	streamMu.Unlock()

	// Return command to read first event
	return readStreamEvent(providerName)
}

// sendTableRonde implements the @all Table Ronde: all providers respond in parallel
func (m *Model) sendTableRonde(content string) tea.Cmd {
	// Find all configured providers
	configuredProviders := []string{}
	for name, p := range m.providers {
		if p.IsConfigured() {
			configuredProviders = append(configuredProviders, name)
		}
	}

	if len(configuredProviders) == 0 {
		return func() tea.Msg {
			return StreamMsg{
				Content:  "No providers configured. Connect them in Control Room (Ctrl+D).",
				Done:     true,
				Provider: "all",
			}
		}
	}

	// If only 1 provider, fallback to normal single-provider streaming
	if len(configuredProviders) == 1 {
		return m.sendToProvider(configuredProviders[0], content)
	}

	// Estimate cost
	approxTokens := 0
	for _, msg := range m.messages {
		approxTokens += (len(msg.Content) + 3) / 4
	}

	models := make([]string, 0, len(configuredProviders))
	for _, name := range configuredProviders {
		if p, ok := m.providers[name]; ok {
			models = append(models, p.GetModel())
		}
	}

	estimatedCost := llm.EstimateCascadeCost(approxTokens, models)

	// Remove the empty assistant placeholder
	m.messages = m.messages[:len(m.messages)-1]

	// Add cost estimate as system message
	if estimatedCost > 0 {
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Table Ronde to %d providers — estimated ~$%.2f", len(configuredProviders), estimatedCost),
		})
	}

	// Capture original user images for later rounds
	var userImages []llm.Image
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			userImages = convertMessageImages(m.messages[i])
			break
		}
	}

	// Initialize Table Ronde state
	m.tableRonde = &tableRondeState{
		participants:    configuredProviders,
		activeProviders: make(map[string]bool),
		messageIndices:  make(map[string]int),
		round:           1,
		maxRounds:       llm.GetMaxTableRounds(),
		userQuestion:    content,
		userImages:      userImages,
	}

	// Add system message for round 1
	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: fmt.Sprintf("Table Ronde — Round 1 with %d providers", len(configuredProviders)),
	})

	// Create message slots for ALL providers
	for _, name := range configuredProviders {
		m.tableRonde.activeProviders[name] = true
		m.tableRonde.messageIndices[name] = len(m.messages)
		m.messages = append(m.messages, Message{
			Role:     "assistant",
			Content:  "",
			Provider: name,
		})
	}

	// Build LLM messages from full chat history
	llmMessages := m.buildLLMMessages()

	// Create ONE shared context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCtx = cancel

	// Start ALL providers streaming in parallel
	cmds := make([]tea.Cmd, 0, len(configuredProviders))
	for _, name := range configuredProviders {
		p := m.providers[name]
		if variants, ok := llm.GetModelVariants()[name]; ok {
			if model, ok := variants["default"]; ok {
				p.SetModel(model)
			}
		}
		var toolDefs []llm.ToolDefinition
		if p.SupportsTools() {
			toolDefs = getToolDefinitions()
		}

		providerMessages := filterMessagesForProvider(llmMessages, name)
		setStreamChan(name, p.Stream(ctx, providerMessages, toolDefs, llm.StreamOptions{Role: "participant", ThinkingMode: m.thinkingMode}))

		n := name // capture
		cmds = append(cmds, readTableRondeEvent(n, 1))
	}

	m.status = fmt.Sprintf("Table Ronde — %d providers responding...", len(configuredProviders))

	return tea.Batch(cmds...)
}

// startNextRound launches the next Table Ronde round for mentioned providers
func (m *Model) startNextRound(mentions []pendingMention) tea.Cmd {
	if m.tableRonde == nil || m.tableRonde.round >= m.tableRonde.maxRounds || len(mentions) == 0 {
		m.finishTableRonde()
		return nil
	}

	m.tableRonde.round++
	round := m.tableRonde.round

	// Build "mentioned by" description
	mentionedBy := make([]string, 0, len(mentions))
	for _, mention := range mentions {
		mentionedBy = append(mentionedBy, mention.target+" (by "+mention.by+")")
	}

	// Add system message for this round
	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: fmt.Sprintf("Table Ronde — Round %d (mentioned: %s)", round, strings.Join(mentionedBy, ", ")),
	})

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCtx = cancel

	cmds := make([]tea.Cmd, 0, len(mentions))
	for _, mention := range mentions {
		name := mention.target
		p, ok := m.providers[name]
		if !ok || !p.IsConfigured() {
			continue
		}

		// Create message slot
		m.tableRonde.activeProviders[name] = true
		m.tableRonde.messageIndices[name] = len(m.messages)
		m.messages = append(m.messages, Message{
			Role:     "assistant",
			Content:  "",
			Provider: name,
		})

		// Apply model variant
		if variants, ok := llm.GetModelVariants()[name]; ok {
			if model, ok := variants["default"]; ok {
				p.SetModel(model)
			}
		}
		var toolDefs []llm.ToolDefinition
		if p.SupportsTools() {
			toolDefs = getToolDefinitions()
		}

		// Build full conversation history (ALL messages including previous rounds)
		llmMessages := m.buildLLMMessages()
		providerMessages := filterMessagesForProvider(llmMessages, name)
		setStreamChan(name, p.Stream(ctx, providerMessages, toolDefs, llm.StreamOptions{Role: "participant", ThinkingMode: m.thinkingMode}))

		n := name // capture
		cmds = append(cmds, readTableRondeEvent(n, round))
	}

	if len(cmds) == 0 {
		m.finishTableRonde()
		return nil
	}

	m.status = fmt.Sprintf("Table Ronde — Round %d (%d providers)...", round, len(cmds))

	return tea.Batch(cmds...)
}

// extractMentions scans completed messages for @provider mentions
func (m *Model) extractMentions() []pendingMention {
	if m.tableRonde == nil {
		return nil
	}
	var mentions []pendingMention
	seen := make(map[string]bool)
	providerNames := config.GetProviderNames()
	for provider, idx := range m.tableRonde.messageIndices {
		if idx >= len(m.messages) {
			continue
		}
		content := strings.ToLower(m.messages[idx].Content)
		for _, name := range providerNames {
			if name == provider {
				continue // skip self-mentions
			}
			if strings.Contains(content, "@"+strings.ToLower(name)) && !seen[name] {
				seen[name] = true
				mentions = append(mentions, pendingMention{target: name, by: provider})
			}
		}
	}
	return mentions
}

// finishTableRonde cleans up Table Ronde state
func (m *Model) finishTableRonde() {
	m.status = "Ready"
	m.isStreaming = false
	m.tableRonde = nil
	// Clean up stream channels
	clearStreamChans()
}

// buildLLMMessages converts chat history to LLM messages (excluding empty slots)
func (m *Model) buildLLMMessages() []llm.Message {
	llmMessages := make([]llm.Message, 0)
	for _, msg := range m.messages {
		if msg.Role == "assistant" && msg.Content == "" {
			continue
		}
		if strings.TrimSpace(msg.Content) == "" && len(msg.Images) == 0 && len(msg.ImageData) == 0 {
			continue
		}
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		llmMessages = append(llmMessages, llm.Message{
			Role:    role,
			Content: msg.Content,
			Images:  convertMessageImages(msg),
		})
	}
	return llmMessages
}

// filterMessagesForProvider builds provider-specific messages (filter images if not supported)
func filterMessagesForProvider(messages []llm.Message, providerName string) []llm.Message {
	supportsImg := llm.SupportsImages(providerName)
	result := make([]llm.Message, len(messages))
	for i, msg := range messages {
		result[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if supportsImg {
			result[i].Images = msg.Images
		}
	}
	return result
}

// readStreamEvent reads the next event from the stream
func readStreamEvent(provider string) tea.Cmd {
	return func() tea.Msg {
		streamMu.Lock()
		ch := streamEventChan
		streamMu.Unlock()

		if ch == nil {
			return StreamMsg{Done: true, Provider: provider}
		}

		event, ok := <-ch
		if !ok {
			return StreamMsg{Done: true, Provider: provider}
		}

		switch event.Type {
		case "content":
			return StreamMsg{Content: event.Content, Provider: provider}
		case "thinking":
			return StreamMsg{Thinking: event.Thinking, Provider: provider}
		case "tool_use":
			return StreamMsg{
				ToolCall: event.ToolCall,
				Provider: provider,
			}
		case "tool_result":
			return StreamMsg{
				ToolCall:   event.ToolCall,
				ToolResult: event.ToolResult,
				Provider:   provider,
			}
		case "done":
			msg := StreamMsg{Done: true, Provider: provider}
			if event.Response != nil {
				msg.InputTokens = event.Response.InputTokens
				msg.OutputTokens = event.Response.OutputTokens
				msg.CacheCreationTokens = event.Response.CacheCreationTokens
				msg.CacheReadTokens = event.Response.CacheReadTokens
			}
			return msg
		case "error":
			return StreamMsg{Error: event.Error, Provider: provider}
		}

		return StreamMsg{Done: true, Provider: provider}
	}
}

// readTableRondeEvent reads from a specific provider's stream during Table Ronde
func readTableRondeEvent(provider string, round int) tea.Cmd {
	return func() tea.Msg {
		ch, ok := getStreamChan(provider)
		if !ok || ch == nil {
			return TableRondeStreamMsg{Done: true, Provider: provider, Round: round}
		}

		event, ok := <-ch
		if !ok {
			return TableRondeStreamMsg{Done: true, Provider: provider, Round: round}
		}

		switch event.Type {
		case "content":
			return TableRondeStreamMsg{Content: event.Content, Provider: provider, Round: round}
		case "thinking":
			return TableRondeStreamMsg{Thinking: event.Thinking, Provider: provider, Round: round}
		case "tool_use":
			return TableRondeStreamMsg{
				ToolCall: event.ToolCall,
				Provider: provider,
				Round:    round,
			}
		case "tool_result":
			return TableRondeStreamMsg{
				ToolCall:   event.ToolCall,
				ToolResult: event.ToolResult,
				Provider:   provider,
				Round:      round,
			}
		case "done":
			msg := TableRondeStreamMsg{Done: true, Provider: provider, Round: round}
			if event.Response != nil {
				msg.InputTokens = event.Response.InputTokens
				msg.OutputTokens = event.Response.OutputTokens
			}
			return msg
		case "error":
			return TableRondeStreamMsg{Error: event.Error, Provider: provider, Round: round}
		}

		return TableRondeStreamMsg{Done: true, Provider: provider, Round: round}
	}
}

// startAnthropicCodeExchange exchanges an Anthropic OAuth code in the background
func startAnthropicCodeExchange(provider, code string) tea.Cmd {
	return func() tea.Msg {
		tokens, err := auth.ExchangeAnthropicCode(code)
		if err != nil {
			return OAuthResultMsg{Provider: provider, Success: false, Error: err.Error()}
		}
		auth.GetStorage().SetOAuthTokens(provider, tokens)
		return OAuthResultMsg{Provider: provider, Success: true}
	}
}

// startOAuthForProvider starts the OAuth flow for any provider that supports it.
// Claude uses a PKCE code-paste flow (handled separately via startAnthropicCodeExchange).
func startOAuthForProvider(provider string) tea.Cmd {
	return func() tea.Msg {
		var tokens *auth.OAuthTokens
		var err error

		switch provider {
		case "gpt":
			tokens, err = auth.StartOpenAIOAuthWithCallback()
		case "gemini":
			tokens, err = auth.StartGoogleOAuthWithCallback()
		default:
			return OAuthResultMsg{Provider: provider, Success: false, Error: "OAuth not supported for " + provider}
		}

		if err != nil {
			return OAuthResultMsg{Provider: provider, Success: false, Error: err.Error()}
		}
		auth.GetStorage().SetOAuthTokens(provider, tokens)
		return OAuthResultMsg{Provider: provider, Success: true}
	}
}

