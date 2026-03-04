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

// CopilotProvider implements Provider for GitHub Copilot.
// Copilot exposes an OpenAI-compatible chat completions API
// with custom headers and a device-flow-based auth token.
type CopilotProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewCopilotProvider creates a new Copilot provider
func NewCopilotProvider(cfg ProviderConfig) *CopilotProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("copilot")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("copilot")
	}

	return &CopilotProvider{
		config:     cfg,
		httpClient: newStreamHTTPClient(),
	}
}

func (p *CopilotProvider) Name() string {
	return "copilot"
}

func (p *CopilotProvider) DisplayName() string {
	return "Copilot"
}

func (p *CopilotProvider) Color() string {
	return "#6e40c9" // GitHub purple
}

func (p *CopilotProvider) ToolFormat() ToolFormat {
	return ToolFormatOpenAI
}

func (p *CopilotProvider) SetModel(model string) {
	p.config.Model = model
}

func (p *CopilotProvider) GetModel() string {
	return p.config.Model
}

func (p *CopilotProvider) SupportsTools() bool {
	return true
}

func (p *CopilotProvider) IsConfigured() bool {
	return auth.GetStorage().IsConnected("copilot")
}

func (p *CopilotProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer close(eventChan)

		token, err := auth.GetStorage().GetAccessToken("copilot")
		if err != nil || token == "" {
			eventChan <- StreamEvent{Type: "error", Error: errors.New("not authenticated — connect via Control Room")}
			return
		}

		p.agenticLoop(ctx, messages, toolDefs, token, eventChan, GetRole(opts), GetThinkingMode(opts))
	}()

	return eventChan
}

func (p *CopilotProvider) agenticLoop(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, token string, eventChan chan<- StreamEvent, role string, thinkingMode bool) {
	// Build conversation history with OAI typed structs
	history := make([]OAIMessage, 0, len(initialMessages)+1)

	// System message
	history = append(history, NewOAITextMessage("system", BuildSystemPrompt("copilot", role)))

	// Convert initial messages
	for _, msg := range initialMessages {
		history = append(history, buildCopilotMessage(msg))
	}

	// Convert tool definitions to OpenAI format
	oaiTools := OAIToolDefsFromPoly(toolDefs)

	var fullContent strings.Builder

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := OAIRequestBody{
			Model:         p.config.Model,
			Stream:        true,
			Messages:      history,
			StreamOptions: &OAIStreamOptions{IncludeUsage: true},
			MaxTokens:     p.config.MaxTokens,
		}

		if len(oaiTools) > 0 {
			body.Tools = oaiTools
		}

		result, err := p.streamRequest(ctx, body, token, eventChan)
		if err != nil {
			// On 401, the session token may have expired mid-request.
			// The storage layer auto-refreshes on GetAccessToken, but if it expired
			// between getting the token and making the request, retry once.
			if strings.Contains(err.Error(), "401") {
				newToken, refreshErr := auth.GetStorage().GetAccessToken("copilot")
				if refreshErr == nil && newToken != "" && newToken != token {
					token = newToken
					result, err = p.streamRequest(ctx, body, token, eventChan)
				}
			}
			if err != nil {
				eventChan <- StreamEvent{Type: "error", Error: err}
				return
			}
		}

		fullContent.WriteString(result.content)

		// No tool calls? Done
		if len(result.toolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:      fullContent.String(),
					Provider:     "copilot",
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
			eventChan <- StreamEvent{
				Type: "tool_use",
				ToolCall: &ToolCall{
					ID:        tc.id,
					Name:      tc.name,
					Arguments: tc.input,
				},
			}

			toolResult := tools.Execute(tc.name, tc.input)

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
			Provider: "copilot",
			Model:    p.config.Model,
		},
	}
}

// buildCopilotMessage creates a message with optional images (OpenAI format)
func buildCopilotMessage(msg Message) OAIMessage {
	return OAIMessage{
		Role:    msg.Role,
		Content: BuildOAIImageParts(msg),
	}
}

// copilotStreamResult holds the parsed result from a Copilot SSE stream
type copilotStreamResult struct {
	content      string
	toolCalls    []copilotToolCall
	inputTokens  int
	outputTokens int
}

type copilotToolCall struct {
	id      string
	name    string
	input   map[string]interface{}
	rawArgs string
}

func (p *CopilotProvider) streamRequest(ctx context.Context, body interface{}, token string, eventChan chan<- StreamEvent) (*copilotStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://api.githubcopilot.com/chat/completions"

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
		req.Header.Set("Authorization", "Bearer "+token)
		// Copilot-specific headers
		req.Header.Set("Copilot-Integration-Id", "vscode-chat")
		req.Header.Set("Editor-Version", "vscode/1.96.0")
		req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.24.0")
		req.Header.Set("Openai-Intent", "conversation-panel")

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
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	result := &copilotStreamResult{}

	toolCallsMap := make(map[int]*copilotToolCall)

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

		if event.Usage != nil {
			result.inputTokens = event.Usage.PromptTokens
			result.outputTokens = event.Usage.CompletionTokens
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
				toolCallsMap[tc.Index] = &copilotToolCall{
					id:   tc.ID,
					name: tc.Function.Name,
				}
			}
			if tc.Function.Arguments != "" && toolCallsMap[tc.Index] != nil {
				toolCallsMap[tc.Index].rawArgs += tc.Function.Arguments
			}
		}

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
