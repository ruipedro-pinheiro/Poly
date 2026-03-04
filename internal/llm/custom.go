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

	"github.com/pedromelo/poly/internal/config"
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

		// Define the request handler for the agentic loop
		handler := func(ctx context.Context, history []Message, internalChan chan<- StreamEvent) (*SingleTurnResult, error) {
			var body interface{}
			var res *customStreamResult
			var err error

			switch p.config.Format {
			case "anthropic":
				antHistory := p.buildAnthropicHistory(history)
				antTools := AntToolDefsFromPoly(toolDefs, false)
				body = AntRequestBody{
					Model:     p.config.Model,
					MaxTokens: p.config.MaxTokens,
					Stream:    true,
					Messages:  antHistory,
					System:    BuildSystemPrompt(p.config.ID, role),
					Tools:     antTools,
				}

			case "google":
				gemContents := p.buildGoogleHistory(history, role)
				gemTools := GemToolDefsFromPoly(toolDefs)
				body = GemRequestBody{
					Contents:         gemContents,
					GenerationConfig: &GemGenerationConfig{MaxOutputTokens: p.config.MaxTokens},
					Tools:            gemTools,
				}

			default: // OpenAI
				oaiHistory := p.buildOpenAIHistory(history, role)
				oaiTools := OAIToolDefsFromPoly(toolDefs)
				body = OAIRequestBody{
					Model:    p.config.Model,
					Stream:   true,
					Messages: oaiHistory,
					Tools:    oaiTools,
				}
				// OpenAI specific reasoning handling...
			}

			resp, err := p.doRequest(ctx, body, thinkingMode)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			switch p.config.Format {
			case "anthropic":
				res = p.parseAnthropicStreamResult(resp.Body, internalChan)
			case "google":
				res = p.parseGoogleStreamResult(resp.Body, internalChan)
			default:
				res = p.parseOpenAIStreamResult(resp.Body, internalChan)
			}

			// Convert customStreamResult to generic SingleTurnResult
			return &SingleTurnResult{
				Content:   res.content,
				ToolCalls: res.toolCalls,
			}, nil
		}

		// Run the universal agentic loop
		RunAgenticLoop(ctx, p.config.ID, p.config.Model, messages, toolDefs, eventChan, handler)
	}()

	return eventChan
}

func (p *CustomProvider) buildOpenAIHistory(messages []Message, role string) []OAIMessage {
	history := make([]OAIMessage, 0, len(messages)+1)
	history = append(history, NewOAITextMessage("system", BuildSystemPrompt(p.config.ID, role)))
	for _, msg := range messages {
		content := BuildOAIImageParts(msg)
		if content != nil {
			history = append(history, OAIMessage{
				Role:    msg.Role,
				Content: content,
			})
		}
	}
	return history
}

func (p *CustomProvider) buildAnthropicHistory(messages []Message) []AntMessage {
	var history []AntMessage
	for _, msg := range messages {
		content := BuildAntImageContent(msg)
		if content != nil {
			history = append(history, AntMessage{
				Role:    msg.Role,
				Content: content,
			})
		}
	}
	return history
}

func (p *CustomProvider) buildGoogleHistory(messages []Message, role string) []GemContent {
	contents := make([]GemContent, 0, len(messages)+2)
	contents = append(contents,
		GemContent{Role: "user", Parts: []GemPart{NewGemTextPart(BuildSystemPrompt(p.config.ID, role))}},
		GemContent{Role: "model", Parts: []GemPart{NewGemTextPart("Understood.")}},
	)
	for _, msg := range messages {
		r := "user"
		if msg.Role == "assistant" {
			r = "model"
		}
		contents = append(contents, GemContent{Role: r, Parts: []GemPart{NewGemTextPart(msg.Content)}})
	}
	return contents
}

// customStreamResult holds the parsed result of a single streaming request
type customStreamResult struct {
	content   string
	toolCalls []ToolCall
}

// --- HTTP request with retry ---

func (p *CustomProvider) doRequest(ctx context.Context, body interface{}, thinkingMode bool) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("provider %q has no base_url configured", p.config.ID)
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, fmt.Errorf("provider %q base_url must start with http:// or https://", p.config.ID)
	}

	url := baseURL
	switch p.config.Format {
	case "anthropic":
		url += "/messages"
	case "google":
		url += "/models/" + p.config.Model + ":streamGenerateContent?alt=sse"
	default:
		url += "/chat/completions"
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
			// Google uses URL param or env
		default:
			if p.config.AuthHeader == "x-api-key" {
				req.Header.Set("x-api-key", p.config.APIKey)
			} else {
				req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
			}
		}
	}

	return DoWithRetry(ctx, p.httpClient, req)
}

// --- Stream parsers ---

func (p *CustomProvider) parseOpenAIStreamResult(body io.Reader, eventChan chan<- StreamEvent) *customStreamResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	result := &customStreamResult{}
	toolCallsMap := make(map[int]*ToolCall)

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

		choice := event.Choices[0]
		delta := choice.Delta

		if delta.Content != "" {
			result.content += delta.Content
			eventChan <- StreamEvent{Type: "content", Content: delta.Content}
		}

		for _, tc := range delta.ToolCalls {
			if tc.ID != "" {
				toolCallsMap[tc.Index] = &ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: make(map[string]interface{})}
				toolCallsMap[tc.Index].Arguments["_raw"] = ""
			}
			if tc.Function.Arguments != "" && toolCallsMap[tc.Index] != nil {
				raw := toolCallsMap[tc.Index].Arguments["_raw"].(string)
				toolCallsMap[tc.Index].Arguments["_raw"] = raw + tc.Function.Arguments
			}
		}

		reason := strings.ToLower(choice.FinishReason)
		if reason == "tool_calls" || reason == "stop" || reason == "end_turn" {
			break
		}
	}

	for _, tc := range toolCallsMap {
		if raw, ok := tc.Arguments["_raw"].(string); ok && raw != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &args); err == nil {
				tc.Arguments = args
			} else {
				tc.Arguments = make(map[string]interface{})
			}
		}
		result.toolCalls = append(result.toolCalls, *tc)
	}

	return result
}

func (p *CustomProvider) parseAnthropicStreamResult(body io.Reader, eventChan chan<- StreamEvent) *customStreamResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	result := &customStreamResult{}
	var currentToolCall *ToolCall
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
				if blockType, _ := contentBlock["type"].(string); blockType == "tool_use" {
					id, _ := contentBlock["id"].(string)
					name, _ := contentBlock["name"].(string)
					currentToolCall = &ToolCall{ID: id, Name: name}
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
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(currentToolInput.String()), &input); err != nil {
					input = make(map[string]interface{})
				}
				currentToolCall.Arguments = input
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
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
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
				tc := ToolCall{
					ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, len(result.toolCalls)),
					Name:      part.FunctionCall.Name,
					Arguments: args,
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
			return nil
		}
		return err
	}

	var configs []CustomProviderConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	for _, cfg := range configs {
		p := NewCustomProvider(cfg)
		RegisterProvider(p)

		// Inject into global config
		gcfg := config.Get()
		if gcfg.Providers == nil {
			gcfg.Providers = make(map[string]config.ProviderConfig)
		}
		gcfg.Providers[cfg.ID] = config.ProviderConfig{
			ID:         cfg.ID,
			Name:       cfg.Name,
			Endpoint:   cfg.BaseURL,
			Format:     cfg.Format,
			Color:      cfg.Color,
			MaxTokens:  cfg.MaxTokens,
			AuthHeader: cfg.AuthHeader,
			AuthType:   "api_key",
			Models: map[string]string{
				"default": cfg.Model,
			},
		}
	}

	return nil
}

// SaveCustomProvider adds a new custom provider and saves
func SaveCustomProvider(cfg CustomProviderConfig) error {
	var configs []CustomProviderConfig
	data, err := os.ReadFile(customProvidersFile)
	if err == nil {
		_ = json.Unmarshal(data, &configs)
	}

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

	p := NewCustomProvider(cfg)
	RegisterProvider(p)

	gcfg := config.Get()
	if gcfg.Providers == nil {
		gcfg.Providers = make(map[string]config.ProviderConfig)
	}
	gcfg.Providers[cfg.ID] = config.ProviderConfig{
		ID:         cfg.ID,
		Name:       cfg.Name,
		Endpoint:   cfg.BaseURL,
		Format:     cfg.Format,
		Color:      cfg.Color,
		MaxTokens:  cfg.MaxTokens,
		AuthHeader: cfg.AuthHeader,
		AuthType:   "api_key",
		Models: map[string]string{
			"default": cfg.Model,
		},
	}

	return nil
}

// DeleteCustomProvider removes a custom provider
func DeleteCustomProvider(id string) error {
	var configs []CustomProviderConfig
	data, err := os.ReadFile(customProvidersFile)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(data, &configs)

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
	_ = json.Unmarshal(data, &configs)
	return configs
}
