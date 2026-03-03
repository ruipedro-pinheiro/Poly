package llm

import (
	"bufio"
	"bytes"
	"context"
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

// GPTProvider implements Provider for OpenAI's GPT models
type GPTProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewGPTProvider creates a new GPT provider
func NewGPTProvider(cfg ProviderConfig) *GPTProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("gpt")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("gpt")
	}

	return &GPTProvider{
		config:     cfg,
		httpClient: newStreamHTTPClient(),
	}
}

func (p *GPTProvider) Name() string {
	return "gpt"
}

func (p *GPTProvider) DisplayName() string {
	return "GPT"
}

func (p *GPTProvider) Color() string {
	return "#10A37F" // OpenAI Green
}

func (p *GPTProvider) ToolFormat() ToolFormat {
	return ToolFormatOpenAI
}

func (p *GPTProvider) SetModel(model string) {
	p.config.Model = model
}

func (p *GPTProvider) GetModel() string {
	return p.config.Model
}

func (p *GPTProvider) SupportsTools() bool {
	return true
}

func (p *GPTProvider) IsConfigured() bool {
	return auth.GetStorage().IsConnected("gpt")
}

func (p *GPTProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer close(eventChan)

		apiKey, err := auth.GetStorage().GetAccessToken("gpt")
		if err != nil || apiKey == "" {
			eventChan <- StreamEvent{Type: "error", Error: errors.New("no API key configured")}
			return
		}

		p.agenticLoop(ctx, messages, toolDefs, apiKey, eventChan, GetRole(opts), GetThinkingMode(opts))
	}()

	return eventChan
}

func (p *GPTProvider) agenticLoop(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, apiKey string, eventChan chan<- StreamEvent, role string, thinkingMode bool) {
	// Build conversation history with typed structs
	history := make([]OAIMessage, 0, len(initialMessages)+1)

	// System message with dynamic prompt
	history = append(history, NewOAITextMessage("system", BuildSystemPrompt("gpt", role)))

	// Add initial messages with image support
	for _, msg := range initialMessages {
		history = append(history, buildGPTMessage(msg))
	}

	// Convert tool definitions to OpenAI format
	oaiTools := OAIToolDefsFromPoly(toolDefs)

	// Agentic loop
	var fullContent strings.Builder

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := OAIRequestBody{
			Model:         p.config.Model,
			Stream:        true,
			Messages:      history,
			StreamOptions: &OAIStreamOptions{IncludeUsage: true},
		}

		if thinkingMode && isReasoningModel(p.config.Model) {
			body.ReasoningEffort = "high"
			body.MaxCompletionTokens = p.config.MaxTokens
		} else {
			body.MaxTokens = p.config.MaxTokens
		}

		if len(oaiTools) > 0 {
			body.Tools = oaiTools
		}

		if thinkingMode && isReasoningModel(p.config.Model) {
			eventChan <- StreamEvent{Type: "thinking", Thinking: "(reasoning...)"}
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
					Provider:     "gpt",
					Model:        p.config.Model,
					InputTokens:  result.inputTokens,
					OutputTokens: result.outputTokens,
				},
			}
			return
		}

		// Build assistant message with tool_calls
		oaiToolCalls := make([]OAIToolCallMsg, len(result.toolCalls))
		for i, tc := range result.toolCalls {
			argsJSON, _ := json.Marshal(tc.input)
			oaiToolCalls[i] = OAIToolCallMsg{
				ID:   tc.id,
				Type: "function",
				Function: OAIToolCallFunc{
					Name:      tc.name,
					Arguments: string(argsJSON),
				},
			}
		}
		history = append(history, NewOAIAssistantMessage(result.content, oaiToolCalls))

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

			history = append(history, NewOAIToolResultMessage(tc.id, toolResult.Content))
		}
	}

	eventChan <- StreamEvent{
		Type:    "content",
		Content: "\n⚠️ Max tool turns reached\n",
	}
	eventChan <- StreamEvent{
		Type: "done",
		Response: &Response{
			Content:  "",
			Provider: "gpt",
			Model:    p.config.Model,
		},
	}
}

// buildGPTMessage creates a message with optional images (OpenAI format)
func buildGPTMessage(msg Message) OAIMessage {
	return OAIMessage{
		Role:    msg.Role,
		Content: BuildOAIImageParts(msg),
	}
}

type gptStreamResult struct {
	content      string
	toolCalls    []gptToolCall
	inputTokens  int
	outputTokens int
}

type gptToolCall struct {
	id      string
	name    string
	input   map[string]interface{}
	rawArgs string
}

func isReasoningModel(model string) bool {
	return strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4")
}

func (p *GPTProvider) streamRequest(ctx context.Context, body interface{}, apiKey string, eventChan chan<- StreamEvent) (*gptStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://api.openai.com/v1/chat/completions"

	var resp *http.Response
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
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err = p.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))

		if !ShouldRetry(resp.StatusCode) {
			return nil, lastErr
		}
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large SSE events
	result := &gptStreamResult{}

	// Track tool calls being built
	toolCallsMap := make(map[int]*gptToolCall)

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
				toolCallsMap[tc.Index] = &gptToolCall{
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
