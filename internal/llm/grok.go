package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/tools"
)


func init() {
	RegisterProvider(NewGrokProvider(ProviderConfig{}))
}

// GrokProvider implements Provider for xAI's Grok
type GrokProvider struct {
	config ProviderConfig
}

// NewGrokProvider creates a new Grok provider
func NewGrokProvider(cfg ProviderConfig) *GrokProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("grok")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("grok")
	}

	return &GrokProvider{
		config: cfg,
	}
}

func (p *GrokProvider) Name() string {
	return "grok"
}

func (p *GrokProvider) DisplayName() string {
	return "Grok"
}

func (p *GrokProvider) Color() string {
	return "#1DA1F2" // xAI Blue
}

func (p *GrokProvider) ToolFormat() ToolFormat {
	return ToolFormatOpenAI
}

func (p *GrokProvider) SetModel(model string) {
	p.config.Model = model
}

func (p *GrokProvider) GetModel() string {
	return p.config.Model
}

func (p *GrokProvider) SupportsTools() bool {
	return true
}

func (p *GrokProvider) IsConfigured() bool {
	return auth.GetStorage().IsConnected("grok")
}

func (p *GrokProvider) Send(ctx context.Context, messages []Message, toolDefs []ToolDefinition) (*Response, error) {
	return nil, errors.New("not implemented")
}

func (p *GrokProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer close(eventChan)

		apiKey, err := auth.GetStorage().GetAccessToken("grok")
		if err != nil || apiKey == "" {
			eventChan <- StreamEvent{Type: "error", Error: errors.New("no API key configured")}
			return
		}

		p.agenticLoop(ctx, messages, toolDefs, apiKey, eventChan, GetRole(opts))
	}()

	return eventChan
}

func (p *GrokProvider) agenticLoop(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, apiKey string, eventChan chan<- StreamEvent, role string) {
	// Build conversation history
	conversationHistory := make([]map[string]interface{}, 0, len(initialMessages)+1)

	// System message with dynamic prompt
	conversationHistory = append(conversationHistory, map[string]interface{}{
		"role":    "system",
		"content": BuildSystemPrompt("grok", role),
	})

	// Add initial messages with image support
	for _, msg := range initialMessages {
		conversationHistory = append(conversationHistory, buildGrokMessage(msg))
	}

	// Build tools array (OpenAI format)
	var openaiTools []map[string]interface{}
	if len(toolDefs) > 0 {
		openaiTools = make([]map[string]interface{}, len(toolDefs))
		for i, tool := range toolDefs {
			openaiTools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.InputSchema,
				},
			}
		}
	}

	// Agentic loop
	var fullContent strings.Builder

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := map[string]interface{}{
			"model":          p.config.Model,
			"max_tokens":     p.config.MaxTokens,
			"stream":         true,
			"messages":       conversationHistory,
			"stream_options": map[string]interface{}{"include_usage": true},
		}

		if len(openaiTools) > 0 {
			body["tools"] = openaiTools
		}

		result, err := p.streamRequest(ctx, body, apiKey, eventChan)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		fullContent.WriteString(result.content)

		// No tool calls? Done
		if len(result.toolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:      fullContent.String(),
					Provider:     "grok",
					Model:        p.config.Model,
					InputTokens:  result.inputTokens,
					OutputTokens: result.outputTokens,
				},
			}
			return
		}

		// Build assistant message with tool_calls
		assistantMsg := map[string]interface{}{
			"role": "assistant",
		}
		if result.content != "" {
			assistantMsg["content"] = result.content
		}

		// OpenAI format for tool_calls
		toolCallsForMsg := make([]map[string]interface{}, len(result.toolCalls))
		for i, tc := range result.toolCalls {
			argsJSON, _ := json.Marshal(tc.input)
			toolCallsForMsg[i] = map[string]interface{}{
				"id":   tc.id,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.name,
					"arguments": string(argsJSON),
				},
			}
		}
		assistantMsg["tool_calls"] = toolCallsForMsg
		conversationHistory = append(conversationHistory, assistantMsg)

		// Execute tools and add results
		for _, tc := range result.toolCalls {
			// Emit tool_use event before execution
			eventChan <- StreamEvent{
				Type: "tool_use",
				ToolCall: &ToolCall{
					ID:        tc.id,
					Name:      tc.name,
					Arguments: tc.input,
				},
			}

			toolResult := tools.Execute(tc.name, tc.input)

			// Emit tool_result event after execution
			eventChan <- StreamEvent{
				Type: "tool_result",
				ToolCall: &ToolCall{
					ID:        tc.id,
					Name:      tc.name,
					Arguments: tc.input,
				},
				ToolResult: &ToolResult{
					ToolUseID: tc.id,
					Content:   toolResult.Content,
					IsError:   toolResult.IsError,
				},
			}

			// OpenAI format for tool results
			conversationHistory = append(conversationHistory, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": tc.id,
				"content":      toolResult.Content,
			})
		}
	}

	eventChan <- StreamEvent{
		Type:    "content",
		Content: "\n⚠️ Max tool turns reached\n",
	}
	eventChan <- StreamEvent{
		Type: "done",
		Response: &Response{
			Content:  fullContent.String(),
			Provider: "grok",
			Model:    p.config.Model,
		},
	}
}

// buildGrokMessage creates a message with optional images (OpenAI format)
func buildGrokMessage(msg Message) map[string]interface{} {
	if len(msg.Images) == 0 {
		return map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	content := make([]map[string]interface{}, 0, len(msg.Images)+1)

	for _, img := range msg.Images {
		dataURL := fmt.Sprintf("data:%s;base64,%s", img.MediaType, base64.StdEncoding.EncodeToString(img.Data))
		content = append(content, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": dataURL,
			},
		})
	}

	if msg.Content != "" {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": msg.Content,
		})
	}

	return map[string]interface{}{
		"role":    msg.Role,
		"content": content,
	}
}

type grokStreamResult struct {
	content      string
	toolCalls    []grokToolCall
	inputTokens  int
	outputTokens int
}

type grokToolCall struct {
	id      string
	name    string
	input   map[string]interface{}
	rawArgs string
}

func (p *GrokProvider) streamRequest(ctx context.Context, body map[string]interface{}, apiKey string, eventChan chan<- StreamEvent) (*grokStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.x.ai/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	result := &grokStreamResult{}

	// Track tool calls being built
	toolCallsMap := make(map[int]*grokToolCall)

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
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Parse usage from final chunk
		if event.Usage != nil {
			result.inputTokens = event.Usage.PromptTokens
			result.outputTokens = event.Usage.CompletionTokens
		}

		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		// Handle content
		if delta.Content != "" {
			result.content += delta.Content
			eventChan <- StreamEvent{Type: "content", Content: delta.Content}
		}

		// Handle tool calls
		for _, tc := range delta.ToolCalls {
			if tc.ID != "" {
				// New tool call
				toolCallsMap[tc.Index] = &grokToolCall{
					id:   tc.ID,
					name: tc.Function.Name,
				}
			}
			if tc.Function.Arguments != "" && toolCallsMap[tc.Index] != nil {
				toolCallsMap[tc.Index].rawArgs += tc.Function.Arguments
			}
		}

		// Check finish reason
		if event.Choices[0].FinishReason == "tool_calls" || event.Choices[0].FinishReason == "stop" {
			break
		}
	}

	// Parse accumulated tool call arguments
	for _, tc := range toolCallsMap {
		if tc.rawArgs != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.rawArgs), &args); err == nil {
				tc.input = args
			}
		}
		if tc.input == nil {
			tc.input = make(map[string]interface{})
		}
		result.toolCalls = append(result.toolCalls, *tc)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
