package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/security"
	"github.com/pedromelo/poly/internal/tools"
)

// CustomProviderConfig defines a user-added provider
type CustomProviderConfig struct {
	Name       string `json:"name"`     // Display name (e.g., "Mistral")
	ID         string `json:"id"`       // Unique ID (e.g., "mistral")
	BaseURL    string `json:"base_url"` // API endpoint (e.g., "https://api.mistral.ai/v1")
	APIKey     string `json:"api_key"`  // API key
	Model      string `json:"model"`    // Default model
	Format     string `json:"format"`   // "openai", "anthropic", "google"
	Color      string `json:"color"`    // Hex color for UI
	MaxTokens  int    `json:"max_tokens"`
	AuthHeader string `json:"auth_header"` // "Bearer" (default) or "x-api-key" etc.
}

// CustomProvider implements Provider for user-defined APIs
type CustomProvider struct {
	config     CustomProviderConfig
	httpClient *http.Client
}

// NewCustomProvider creates a custom provider from config
func NewCustomProvider(cfg CustomProviderConfig) *CustomProvider {
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens(cfg.ID)
		if cfg.MaxTokens == 0 {
			cfg.MaxTokens = 4096 // Fallback for truly unknown providers
		}
	}
	if cfg.AuthHeader == "" {
		cfg.AuthHeader = "Bearer"
	}
	if cfg.Color == "" {
		cfg.Color = "#888888"
	}
	return &CustomProvider{config: cfg, httpClient: newStreamHTTPClient()}
}

func (p *CustomProvider) Name() string        { return p.config.ID }
func (p *CustomProvider) DisplayName() string { return p.config.Name }
func (p *CustomProvider) Color() string       { return p.config.Color }
func (p *CustomProvider) GetModel() string    { return p.config.Model }
func (p *CustomProvider) SetModel(m string)   { p.config.Model = m }
func (p *CustomProvider) SupportsTools() bool { return true }
func (p *CustomProvider) IsConfigured() bool  { return p.config.APIKey != "" }

func (p *CustomProvider) ToolFormat() ToolFormat {
	switch p.config.Format {
	case "anthropic":
		return ToolFormatAnthropic
	case "google":
		return ToolFormatGoogle
	default:
		return ToolFormatOpenAI
	}
}

func (p *CustomProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				defer func() { recover() }() //nolint:errcheck // protect against closed channel
				eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("internal error (panic recovered)")}
			}
			close(eventChan)
		}()

		role := GetRole(opts)
		thinkingMode := GetThinkingMode(opts)
		p.agenticLoop(ctx, messages, toolDefs, eventChan, role, thinkingMode)
	}()

	return eventChan
}

// customStreamResult holds the parsed result of a single streaming request
type customStreamResult struct {
	content   string
	toolCalls []customToolCall
}

type customToolCall struct {
	id    string
	name  string
	input map[string]interface{}
}

// agenticLoop runs the tool-use loop for custom providers
func (p *CustomProvider) agenticLoop(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, eventChan chan<- StreamEvent, role string, thinkingMode bool) {
	var fullContent strings.Builder

	switch p.config.Format {
	case "anthropic":
		p.agenticLoopAnthropic(ctx, initialMessages, toolDefs, eventChan, role, thinkingMode, &fullContent)
	case "google":
		p.agenticLoopGoogle(ctx, initialMessages, toolDefs, eventChan, role, thinkingMode, &fullContent)
	default:
		p.agenticLoopOpenAI(ctx, initialMessages, toolDefs, eventChan, role, thinkingMode, &fullContent)
	}
}

// --- OpenAI format agentic loop ---

func (p *CustomProvider) agenticLoopOpenAI(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, eventChan chan<- StreamEvent, role string, thinkingMode bool, fullContent *strings.Builder) {
	// Build conversation history
	msgs := make([]OAIMessage, 0, len(initialMessages)+1)
	msgs = append(msgs, NewOAITextMessage("system", BuildSystemPrompt(p.config.ID, role)))
	for _, msg := range initialMessages {
		msgs = append(msgs, NewOAITextMessage(msg.Role, msg.Content))
	}

	oaiTools := OAIToolDefsFromPoly(toolDefs)

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := OAIRequestBody{
			Model:    p.config.Model,
			Stream:   true,
			Messages: msgs,
			Tools:    oaiTools,
		}
		if thinkingMode && IsReasoningModel(p.config.ID, p.config.Model) {
			body.ReasoningEffort = "high"
			body.MaxCompletionTokens = p.config.MaxTokens
		} else {
			body.MaxTokens = p.config.MaxTokens
		}

		if thinkingMode && IsReasoningModel(p.config.ID, p.config.Model) {
			eventChan <- StreamEvent{Type: "thinking", Thinking: "(reasoning...)"}
		}

		resp, err := p.doRequest(ctx, body, thinkingMode)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		result := p.parseOpenAIStreamResult(resp.Body, eventChan)
		resp.Body.Close()

		fullContent.WriteString(result.content)

		if len(result.toolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:  fullContent.String(),
					Provider: p.config.ID,
					Model:    p.config.Model,
				},
			}
			return
		}

		// Build assistant message with tool_calls
		tcMsgs := make([]OAIToolCallMsg, len(result.toolCalls))
		for i, tc := range result.toolCalls {
			argsJSON, _ := json.Marshal(tc.input)
			tcMsgs[i] = OAIToolCallMsg{
				ID:   tc.id,
				Type: "function",
				Function: OAIToolCallFunc{
					Name:      tc.name,
					Arguments: string(argsJSON),
				},
			}
		}
		msgs = append(msgs, NewOAIAssistantMessage(result.content, tcMsgs))

		// Execute tools
		for _, tc := range result.toolCalls {
			eventChan <- StreamEvent{
				Type:     "tool_use",
				ToolCall: &ToolCall{ID: tc.id, Name: tc.name, Arguments: tc.input},
			}
			toolResult := tools.Execute(tc.name, tc.input)
			eventChan <- StreamEvent{
				Type:       "tool_result",
				ToolCall:   &ToolCall{ID: tc.id, Name: tc.name, Arguments: tc.input},
				ToolResult: &ToolResult{ToolUseID: tc.id, Content: toolResult.Content, IsError: toolResult.IsError},
			}
			msgs = append(msgs, NewOAIToolResultMessage(tc.id, toolResult.Content))
		}
	}

	eventChan <- StreamEvent{Type: "content", Content: "\n⚠️ Max tool turns reached\n"}
	eventChan <- StreamEvent{
		Type:     "done",
		Response: &Response{Content: fullContent.String(), Provider: p.config.ID, Model: p.config.Model},
	}
}

// --- Anthropic format agentic loop ---

func (p *CustomProvider) agenticLoopAnthropic(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, eventChan chan<- StreamEvent, role string, thinkingMode bool, fullContent *strings.Builder) {
	msgs := make([]AntMessage, 0, len(initialMessages))
	for _, msg := range initialMessages {
		msgs = append(msgs, NewAntTextMessage(msg.Role, msg.Content))
	}

	antTools := AntToolDefsFromPoly(toolDefs, false)

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := AntRequestBody{
			Model:     p.config.Model,
			MaxTokens: p.config.MaxTokens,
			Stream:    true,
			Messages:  msgs,
			System:    BuildSystemPrompt(p.config.ID, role),
			Tools:     antTools,
		}
		if thinkingMode {
			budgetTokens := 10000
			if body.MaxTokens <= budgetTokens {
				body.MaxTokens = budgetTokens + 4096
			}
			body.Thinking = &AntThinkingConfig{
				Type:         "enabled",
				BudgetTokens: budgetTokens,
			}
		}

		resp, err := p.doRequest(ctx, body, thinkingMode)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		result := p.parseAnthropicStreamResult(resp.Body, eventChan)
		resp.Body.Close()

		fullContent.WriteString(result.content)

		if len(result.toolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:  fullContent.String(),
					Provider: p.config.ID,
					Model:    p.config.Model,
				},
			}
			return
		}

		// Build assistant content blocks
		assistantBlocks := make([]AntContentBlock, 0, len(result.toolCalls)+1)
		if result.content != "" {
			assistantBlocks = append(assistantBlocks, AntContentBlock{Type: "text", Text: result.content})
		}
		for _, tc := range result.toolCalls {
			assistantBlocks = append(assistantBlocks, AntContentBlock{
				Type:  "tool_use",
				ID:    tc.id,
				Name:  tc.name,
				Input: tc.input,
			})
		}
		msgs = append(msgs, AntMessage{Role: "assistant", Content: assistantBlocks})

		// Execute tools
		toolResultBlocks := make([]AntContentBlock, 0, len(result.toolCalls))
		for _, tc := range result.toolCalls {
			eventChan <- StreamEvent{
				Type:     "tool_use",
				ToolCall: &ToolCall{ID: tc.id, Name: tc.name, Arguments: tc.input},
			}
			toolResult := tools.Execute(tc.name, tc.input)
			eventChan <- StreamEvent{
				Type:       "tool_result",
				ToolCall:   &ToolCall{ID: tc.id, Name: tc.name, Arguments: tc.input},
				ToolResult: &ToolResult{ToolUseID: tc.id, Content: toolResult.Content, IsError: toolResult.IsError},
			}
			toolResultBlocks = append(toolResultBlocks, NewAntToolResultBlock(tc.id, toolResult.Content, toolResult.IsError))
		}
		msgs = append(msgs, AntMessage{Role: "user", Content: toolResultBlocks})
	}

	eventChan <- StreamEvent{Type: "content", Content: "\n⚠️ Max tool turns reached\n"}
	eventChan <- StreamEvent{
		Type:     "done",
		Response: &Response{Content: fullContent.String(), Provider: p.config.ID, Model: p.config.Model},
	}
}

// --- Google format agentic loop ---

func (p *CustomProvider) agenticLoopGoogle(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, eventChan chan<- StreamEvent, role string, thinkingMode bool, fullContent *strings.Builder) {
	contents := make([]GemContent, 0, len(initialMessages)+2)
	contents = append(contents,
		GemContent{Role: "user", Parts: []GemPart{NewGemTextPart(BuildSystemPrompt(p.config.ID, role))}},
		GemContent{Role: "model", Parts: []GemPart{NewGemTextPart("Understood.")}},
	)
	for _, msg := range initialMessages {
		r := "user"
		if msg.Role == "assistant" {
			r = "model"
		}
		contents = append(contents, GemContent{Role: r, Parts: []GemPart{NewGemTextPart(msg.Content)}})
	}

	genConfig := &GemGenerationConfig{MaxOutputTokens: p.config.MaxTokens}
	if thinkingMode {
		genConfig.ThinkingConfig = &GemThinkingConfig{ThinkingBudget: 8192}
	}

	gemTools := GemToolDefsFromPoly(toolDefs)

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := GemRequestBody{
			Contents:         contents,
			GenerationConfig: genConfig,
			Tools:            gemTools,
		}

		resp, err := p.doRequest(ctx, body, thinkingMode)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		result := p.parseGoogleStreamResult(resp.Body, eventChan)
		resp.Body.Close()

		fullContent.WriteString(result.content)

		if len(result.toolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:  fullContent.String(),
					Provider: p.config.ID,
					Model:    p.config.Model,
				},
			}
			return
		}

		// Build model response with function calls
		modelParts := make([]GemPart, 0, len(result.toolCalls)+1)
		if result.content != "" {
			modelParts = append(modelParts, NewGemTextPart(result.content))
		}
		for _, tc := range result.toolCalls {
			modelParts = append(modelParts, NewGemFunctionCallPart(tc.name, tc.input))
		}
		contents = append(contents, GemContent{Role: "model", Parts: modelParts})

		// Execute tools and build function responses
		responseParts := make([]GemPart, 0, len(result.toolCalls))
		for _, tc := range result.toolCalls {
			eventChan <- StreamEvent{
				Type:     "tool_use",
				ToolCall: &ToolCall{ID: tc.id, Name: tc.name, Arguments: tc.input},
			}
			toolResult := tools.Execute(tc.name, tc.input)
			eventChan <- StreamEvent{
				Type:       "tool_result",
				ToolCall:   &ToolCall{ID: tc.id, Name: tc.name, Arguments: tc.input},
				ToolResult: &ToolResult{ToolUseID: tc.id, Content: toolResult.Content, IsError: toolResult.IsError},
			}
			responseParts = append(responseParts, NewGemFunctionResponsePart(tc.name, toolResult.Content))
		}
		contents = append(contents, GemContent{Role: "user", Parts: responseParts})
	}

	eventChan <- StreamEvent{Type: "content", Content: "\n⚠️ Max tool turns reached\n"}
	eventChan <- StreamEvent{
		Type:     "done",
		Response: &Response{Content: fullContent.String(), Provider: p.config.ID, Model: p.config.Model},
	}
}

// --- HTTP request with retry ---

func (p *CustomProvider) doRequest(ctx context.Context, body interface{}, thinkingMode bool) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := strings.TrimSuffix(p.config.BaseURL, "/")
	switch p.config.Format {
	case "anthropic":
		url += "/messages"
	case "google":
		url += "/models/" + p.config.Model + ":streamGenerateContent?alt=sse"
	default:
		url += "/chat/completions"
	}

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RetryDelay(attempt - 1)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")

		if p.config.APIKey != "" {
			switch p.config.Format {
			case "anthropic":
				req.Header.Set("x-api-key", p.config.APIKey)
				req.Header.Set("anthropic-version", "2023-06-01")
				if thinkingMode {
					req.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")
				}
			case "google":
				// Google uses URL param
			default:
				if p.config.AuthHeader == "x-api-key" {
					req.Header.Set("x-api-key", p.config.APIKey)
				} else {
					req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
				}
			}
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, security.SanitizeResponseBody(bodyBytes))

		if !ShouldRetry(resp.StatusCode) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// --- Stream parsers that return results with tool calls ---

func (p *CustomProvider) parseOpenAIStreamResult(body io.Reader, eventChan chan<- StreamEvent) *customStreamResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	result := &customStreamResult{}
	toolCallsMap := make(map[int]*customToolCall)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		if delta.Content != "" {
			result.content += delta.Content
			eventChan <- StreamEvent{Type: "content", Content: delta.Content}
		}

		for _, tc := range delta.ToolCalls {
			if tc.ID != "" {
				toolCallsMap[tc.Index] = &customToolCall{id: tc.ID, name: tc.Function.Name}
			}
			if tc.Function.Arguments != "" && toolCallsMap[tc.Index] != nil {
				// Accumulate raw args - we'll parse at the end
				if toolCallsMap[tc.Index].input == nil {
					toolCallsMap[tc.Index].input = map[string]interface{}{"_raw": ""}
				}
				toolCallsMap[tc.Index].input["_raw"] = toolCallsMap[tc.Index].input["_raw"].(string) + tc.Function.Arguments
			}
		}

		if event.Choices[0].FinishReason == "tool_calls" || event.Choices[0].FinishReason == "stop" {
			break
		}
	}

	// Parse accumulated tool call arguments
	for _, tc := range toolCallsMap {
		if raw, ok := tc.input["_raw"].(string); ok && raw != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &args); err == nil {
				tc.input = args
			} else {
				tc.input = make(map[string]interface{})
			}
		}
		if tc.input == nil {
			tc.input = make(map[string]interface{})
		}
		result.toolCalls = append(result.toolCalls, *tc)
	}

	return result
}

func (p *CustomProvider) parseAnthropicStreamResult(body io.Reader, eventChan chan<- StreamEvent) *customStreamResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	result := &customStreamResult{}
	var currentToolCall *customToolCall
	var currentToolInput strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "content_block_start":
			if contentBlock, ok := event["content_block"].(map[string]interface{}); ok {
				blockType, _ := contentBlock["type"].(string)
				if blockType == "tool_use" {
					id, _ := contentBlock["id"].(string)
					name, _ := contentBlock["name"].(string)
					currentToolCall = &customToolCall{id: id, name: name}
					currentToolInput.Reset()
				}
			}

		case "content_block_delta":
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				deltaType, _ := delta["type"].(string)
				if deltaType == "text_delta" {
					if text, ok := delta["text"].(string); ok && text != "" {
						result.content += text
						eventChan <- StreamEvent{Type: "content", Content: text}
					}
				} else if deltaType == "thinking_delta" {
					if thinking, ok := delta["thinking"].(string); ok && thinking != "" {
						eventChan <- StreamEvent{Type: "thinking", Thinking: thinking}
					}
				} else if deltaType == "input_json_delta" {
					if partial, ok := delta["partial_json"].(string); ok {
						currentToolInput.WriteString(partial)
					}
				}
			}

		case "content_block_stop":
			if currentToolCall != nil {
				inputStr := currentToolInput.String()
				var input map[string]interface{}
				if inputStr != "" {
					if err := json.Unmarshal([]byte(inputStr), &input); err != nil {
						input = make(map[string]interface{})
					}
				}
				if input == nil {
					input = make(map[string]interface{})
				}
				currentToolCall.input = input
				result.toolCalls = append(result.toolCalls, *currentToolCall)
				currentToolCall = nil
			}

		case "message_stop":
			return result
		}
	}

	return result
}

func (p *CustomProvider) parseGoogleStreamResult(body io.Reader, eventChan chan<- StreamEvent) *customStreamResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	result := &customStreamResult{}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text         string `json:"text"`
						Thought      bool   `json:"thought,omitempty"`
						FunctionCall *struct {
							Name string                 `json:"name"`
							Args map[string]interface{} `json:"args"`
						} `json:"functionCall,omitempty"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Candidates) == 0 || len(event.Candidates[0].Content.Parts) == 0 {
			continue
		}

		for _, part := range event.Candidates[0].Content.Parts {
			if part.Thought && part.Text != "" {
				eventChan <- StreamEvent{Type: "thinking", Thinking: part.Text}
			} else if part.FunctionCall != nil {
				args := part.FunctionCall.Args
				if args == nil {
					args = make(map[string]interface{})
				}
				tc := customToolCall{
					id:    fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, len(result.toolCalls)),
					name:  part.FunctionCall.Name,
					input: args,
				}
				result.toolCalls = append(result.toolCalls, tc)
			} else if part.Text != "" {
				result.content += part.Text
				eventChan <- StreamEvent{Type: "content", Content: part.Text}
			}
		}
	}

	return result
}

// --- Storage for custom providers ---

var customProvidersFile string

func init() {
	home, _ := os.UserHomeDir()
	customProvidersFile = filepath.Join(home, ".poly", "providers.json")
}

// LoadCustomProviders loads and registers all custom providers
func LoadCustomProviders() error {
	data, err := os.ReadFile(customProvidersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No custom providers yet
		}
		return err
	}

	var configs []CustomProviderConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	for _, cfg := range configs {
		RegisterProvider(NewCustomProvider(cfg))
	}

	return nil
}

// SaveCustomProvider adds a new custom provider and saves
func SaveCustomProvider(cfg CustomProviderConfig) error {
	// Load existing
	var configs []CustomProviderConfig
	data, err := os.ReadFile(customProvidersFile)
	if err == nil {
		if err := json.Unmarshal(data, &configs); err != nil {
			configs = nil
		}
	}

	// Update or add
	found := false
	for i, c := range configs {
		if c.ID == cfg.ID {
			configs[i] = cfg
			found = true
			break
		}
	}
	if !found {
		configs = append(configs, cfg)
	}

	// Save
	if err := os.MkdirAll(filepath.Dir(customProvidersFile), 0700); err != nil {
		return err
	}

	data, err = json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(customProvidersFile, data, 0600); err != nil {
		return err
	}

	// Register the provider
	RegisterProvider(NewCustomProvider(cfg))

	return nil
}

// DeleteCustomProvider removes a custom provider
func DeleteCustomProvider(id string) error {
	var configs []CustomProviderConfig
	data, err := os.ReadFile(customProvidersFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	// Filter out the provider
	newConfigs := make([]CustomProviderConfig, 0, len(configs))
	for _, c := range configs {
		if c.ID != id {
			newConfigs = append(newConfigs, c)
		}
	}

	data, err = json.MarshalIndent(newConfigs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(customProvidersFile, data, 0600)
}

// GetCustomProviders returns all custom provider configs
func GetCustomProviders() []CustomProviderConfig {
	var configs []CustomProviderConfig
	data, err := os.ReadFile(customProvidersFile)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil
	}
	return configs
}
