package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/tools"
)

// streamEventChan stores the current streaming channel
var streamEventChan <-chan llm.StreamEvent

// cascadeStreamChans stores channels for @all cascade streaming
var cascadeStreamChans = make(map[string]<-chan llm.StreamEvent)

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
	// Handle @all - cascade orchestration (cheapest first, then reviewers)
	if providerName == "all" {
		return m.sendCascade(content)
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
	streamEventChan = p.Stream(ctx, llmMessages, toolDefs, llm.StreamOptions{ThinkingMode: m.thinkingMode})

	// Return command to read first event
	return readStreamEvent(providerName)
}

// sendCascade implements the @all cascade: cheapest provider responds first, others review
func (m *Model) sendCascade(content string) tea.Cmd {
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

	// Sort by cost tier (ascending - cheapest first)
	sort.Slice(configuredProviders, func(i, j int) bool {
		return llm.GetProviderCostTier(configuredProviders[i]) < llm.GetProviderCostTier(configuredProviders[j])
	})

	// First provider = responder (cheapest), rest = reviewers
	responder := configuredProviders[0]
	reviewers := configuredProviders[1:]

	// Remove the empty assistant placeholder
	m.messages = m.messages[:len(m.messages)-1]

	// Capture original user images for reviewers
	var userImages []llm.Image
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			userImages = convertMessageImages(m.messages[i])
			break
		}
	}

	// Initialize cascade state
	m.cascade = &cascadeState{
		responder:       responder,
		reviewers:       reviewers,
		activeReviewers: make(map[string]bool),
		messageIndices:  make(map[string]int),
		phase:           CascadeResponder,
		userQuestion:    content,
		userImages:      userImages,
	}

	// Add message slot for responder
	m.cascade.messageIndices[responder] = len(m.messages)
	m.messages = append(m.messages, Message{
		Role:     "assistant",
		Content:  "",
		Provider: responder,
	})

	// Add message slots for reviewers (they'll be filled in phase 2)
	for _, rev := range reviewers {
		m.cascade.activeReviewers[rev] = true
		m.cascade.messageIndices[rev] = len(m.messages)
		m.messages = append(m.messages, Message{
			Role:     "assistant",
			Content:  "",
			Provider: rev,
		})
	}

	// Build LLM messages from history
	llmMessages := m.buildLLMMessages()

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCtx = cancel

	// Start streaming the responder with "responder" role
	p := m.providers[responder]
	if models, ok := llm.GetModelVariants()[responder]; ok {
		if model, ok := models["default"]; ok {
			p.SetModel(model)
		}
	}
	var toolDefs []llm.ToolDefinition
	if p.SupportsTools() {
		toolDefs = getToolDefinitions()
	}

	// Filter images if not supported
	providerMessages := filterMessagesForProvider(llmMessages, responder)

	cascadeStreamChans[responder] = p.Stream(ctx, providerMessages, toolDefs, llm.StreamOptions{Role: "responder", ThinkingMode: m.thinkingMode})
	m.status = ">> " + responder + " responding..."

	return readCascadeEvent(responder, CascadeResponder)
}

// startReviewers launches all reviewer streams after the responder is done
func (m *Model) startReviewers() tea.Cmd {
	if m.cascade == nil || len(m.cascade.reviewers) == 0 {
		// No reviewers - we're done
		m.status = "Ready"
		m.isStreaming = false
		m.cascade = nil
		return nil
	}

	m.status = "Reviewers checking..."

	// IMPORTANT: Do NOT include full conversation history for reviewers.
	responderDisplayName := m.cascade.responder
	if p, ok := m.providers[m.cascade.responder]; ok {
		responderDisplayName = p.DisplayName()
	}

	reviewContext := fmt.Sprintf(
		"[REVIEW TASK]\n"+
			"A user asked the following question:\n\"%s\"\n\n"+
			"The AI named %s (provider: %s) responded with:\n---\n%s\n---\n\n"+
			"You are a DIFFERENT AI reviewing this response. This is NOT your response.\n"+
			"If the response is correct and complete: output only \"✓\".\n"+
			"If you find factual errors, security issues, or missing information: state the correction directly.\n\n"+
			"IMPORTANT: If the user's question asks each AI personally (e.g., \"your name\", \"who are you\",\n"+
			"\"introduce yourself\"), then give YOUR OWN answer instead of reviewing.",
		m.cascade.userQuestion,
		responderDisplayName,
		m.cascade.responder,
		m.cascade.responderContent,
	)
	reviewMessages := []llm.Message{{
		Role:    "user",
		Content: reviewContext,
		Images:  m.cascade.userImages,
	}}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCtx = cancel

	cmds := make([]tea.Cmd, 0, len(m.cascade.reviewers))
	for _, revName := range m.cascade.reviewers {
		p := m.providers[revName]
		if models, ok := llm.GetModelVariants()[revName]; ok {
			if model, ok := models["default"]; ok {
				p.SetModel(model)
			}
		}
		var toolDefs []llm.ToolDefinition
		if p.SupportsTools() {
			toolDefs = getToolDefinitions()
		}

		providerMessages := filterMessagesForProvider(reviewMessages, revName)
		cascadeStreamChans[revName] = p.Stream(ctx, providerMessages, toolDefs, llm.StreamOptions{Role: "reviewer", ThinkingMode: m.thinkingMode})

		rn := revName // capture
		cmds = append(cmds, readCascadeEvent(rn, CascadeReviewer))
	}

	return tea.Batch(cmds...)
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
		if streamEventChan == nil {
			return StreamMsg{Done: true, Provider: provider}
		}

		event, ok := <-streamEventChan
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

// readCascadeEvent reads from a specific provider's stream during cascade
func readCascadeEvent(provider string, phase CascadePhase) tea.Cmd {
	return func() tea.Msg {
		ch, ok := cascadeStreamChans[provider]
		if !ok || ch == nil {
			return CascadeStreamMsg{Done: true, Provider: provider, Phase: phase}
		}

		event, ok := <-ch
		if !ok {
			return CascadeStreamMsg{Done: true, Provider: provider, Phase: phase}
		}

		switch event.Type {
		case "content":
			return CascadeStreamMsg{Content: event.Content, Provider: provider, Phase: phase}
		case "thinking":
			return CascadeStreamMsg{Thinking: event.Thinking, Provider: provider, Phase: phase}
		case "tool_use":
			return CascadeStreamMsg{
				ToolCall: event.ToolCall,
				Provider: provider,
				Phase:    phase,
			}
		case "tool_result":
			return CascadeStreamMsg{
				ToolCall:   event.ToolCall,
				ToolResult: event.ToolResult,
				Provider:   provider,
				Phase:      phase,
			}
		case "done":
			return CascadeStreamMsg{Done: true, Provider: provider, Phase: phase}
		case "error":
			return CascadeStreamMsg{Error: event.Error, Provider: provider, Phase: phase}
		}

		return CascadeStreamMsg{Done: true, Provider: provider, Phase: phase}
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

// startOpenAIOAuth starts OpenAI OAuth flow in background
func startOpenAIOAuth() tea.Cmd {
	return func() tea.Msg {
		tokens, err := auth.StartOpenAIOAuthWithCallback()
		if err != nil {
			return OAuthResultMsg{Provider: "gpt", Success: false, Error: err.Error()}
		}
		auth.GetStorage().SetOAuthTokens("gpt", tokens)
		return OAuthResultMsg{Provider: "gpt", Success: true}
	}
}

// startGoogleOAuth starts Google OAuth flow in background
func startGoogleOAuth() tea.Cmd {
	return func() tea.Msg {
		tokens, err := auth.StartGoogleOAuthWithCallback()
		if err != nil {
			return OAuthResultMsg{Provider: "gemini", Success: false, Error: err.Error()}
		}
		auth.GetStorage().SetOAuthTokens("gemini", tokens)
		return OAuthResultMsg{Provider: "gemini", Success: true}
	}
}

