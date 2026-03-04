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
	"os"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/tools"
)

// oaiStreamResult holds the parsed result from an OpenAI-compatible SSE stream.
type oaiStreamResult struct {
	content      string
	toolCalls    []oaiToolCall
	inputTokens  int
	outputTokens int
}

// oaiToolCall represents a tool call accumulated during SSE streaming.
type oaiToolCall struct {
	id      string
	name    string
	input   map[string]interface{}
	rawArgs string
}

// OAIBaseProvider implements the shared Provider logic for all OpenAI-compatible
// providers (GPT, Grok, Copilot). Each concrete provider embeds this struct and
// configures its behavior via constructor fields and optional hooks.
//
// Provider-specific differences are expressed through:
//   - endpoint: the API URL
//   - setHeaders: custom HTTP headers (nil = default "Authorization: Bearer <token>")
//   - handleStreamError: error recovery hook (e.g. Copilot 401 token refresh)
//   - hasReasoningContent: parse reasoning_content from SSE deltas (Grok)
//   - alwaysUseReasoningTokens: use MaxCompletionTokens even without thinkingMode (Grok)
//   - authError: custom error message when no token is available
type OAIBaseProvider struct {
	providerID  string
	displayName string
	color       string
	endpoint    string
	config      ProviderConfig
	httpClient  *http.Client

	// setHeaders adds provider-specific HTTP headers to each request.
	// When nil, sets "Authorization: Bearer <token>" only.
	setHeaders func(req *http.Request, token string)

	// handleStreamError is called when streamRequest returns an error.
	// It may return a new token to trigger a retry (e.g. Copilot 401 refresh).
	// When nil, all errors propagate directly.
	handleStreamError func(err error, currentToken string) (newToken string, retryErr error)

	// hasReasoningContent — when true, parses the reasoning_content field from
	// SSE deltas and emits thinking events (Grok-specific).
	hasReasoningContent bool

	// alwaysUseReasoningTokens — when true, reasoning models always use
	// MaxCompletionTokens regardless of thinkingMode (Grok behavior: models
	// like grok-4 reason automatically). When false, MaxCompletionTokens is
	// only used when thinkingMode is explicitly requested (GPT/Copilot).
	alwaysUseReasoningTokens bool

	// authError overrides the default "no API key configured" error message.
	// Used by Copilot for a more specific auth error.
	authError string
}

// --- Provider interface implementation ---

func (p *OAIBaseProvider) Name() string           { return p.providerID }
func (p *OAIBaseProvider) DisplayName() string    { return p.displayName }
func (p *OAIBaseProvider) Color() string          { return p.color }
func (p *OAIBaseProvider) ToolFormat() ToolFormat { return ToolFormatOpenAI }
func (p *OAIBaseProvider) SetModel(m string)      { p.config.Model = m }
func (p *OAIBaseProvider) GetModel() string       { return p.config.Model }
func (p *OAIBaseProvider) SupportsTools() bool    { return true }

func (p *OAIBaseProvider) IsConfigured() bool {
	return auth.GetStorage().IsConnected(p.providerID)
}

func (p *OAIBaseProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer close(eventChan)

		token, err := auth.GetStorage().GetAccessToken(p.providerID)
		if err != nil || token == "" {
			errMsg := p.authError
			if errMsg == "" {
				errMsg = "no API key configured"
			}
			eventChan <- StreamEvent{Type: "error", Error: errors.New(errMsg)}
			return
		}

		p.agenticLoop(ctx, messages, toolDefs, token, eventChan, GetRole(opts), GetThinkingMode(opts))
	}()

	return eventChan
}

// --- Agentic loop ---

func (p *OAIBaseProvider) agenticLoop(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, token string, eventChan chan<- StreamEvent, role string, thinkingMode bool) {
	// Build conversation history
	history := make([]OAIMessage, 0, len(initialMessages)+1)
	history = append(history, NewOAITextMessage("system", BuildSystemPrompt(p.providerID, role)))

	for _, msg := range initialMessages {
		history = append(history, OAIMessage{
			Role:    msg.Role,
			Content: BuildOAIImageParts(msg),
		})
	}

	oaiTools := OAIToolDefsFromPoly(toolDefs)

	var fullContent strings.Builder

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		body := p.buildRequestBody(history, oaiTools, thinkingMode)

		// Emit thinking indicator for reasoning models
		if thinkingMode && IsReasoningModel(p.providerID, p.config.Model) {
			eventChan <- StreamEvent{Type: "thinking", Thinking: "(reasoning...)"}
		}

		result, err := p.streamRequest(ctx, body, token, eventChan)
		if err != nil {
			// Allow provider-specific error recovery (e.g. Copilot 401 token refresh)
			if p.handleStreamError != nil {
				newToken, retryErr := p.handleStreamError(err, token)
				if retryErr == nil && newToken != "" {
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

		// No tool calls → done
		if len(result.toolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:      fullContent.String(),
					Provider:     p.providerID,
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
		Content: "\n\u26a0\ufe0f Max tool turns reached\n",
	}
	eventChan <- StreamEvent{
		Type: "done",
		Response: &Response{
			Content:  fullContent.String(),
			Provider: p.providerID,
			Model:    p.config.Model,
		},
	}
}

// buildRequestBody creates the OAI request body with provider-specific reasoning logic.
func (p *OAIBaseProvider) buildRequestBody(history []OAIMessage, oaiTools []OAIToolDef, thinkingMode bool) OAIRequestBody {
	body := OAIRequestBody{
		Model:         p.config.Model,
		Stream:        true,
		Messages:      history,
		StreamOptions: &OAIStreamOptions{IncludeUsage: true},
	}

	isReasoning := IsReasoningModel(p.providerID, p.config.Model)
	useReasoningTokens := isReasoning && (p.alwaysUseReasoningTokens || thinkingMode)

	if useReasoningTokens {
		body.MaxCompletionTokens = p.config.MaxTokens
		if thinkingMode && SupportsReasoningEffort(p.providerID, p.config.Model) {
			body.ReasoningEffort = "high"
		}
	} else {
		body.MaxTokens = p.config.MaxTokens
	}

	if len(oaiTools) > 0 {
		body.Tools = oaiTools
	}

	return body
}

// --- HTTP + SSE streaming ---

func (p *OAIBaseProvider) streamRequest(ctx context.Context, body interface{}, token string, eventChan chan<- StreamEvent) (*oaiStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

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

		req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		if p.setHeaders != nil {
			p.setHeaders(req, token)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}

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

	return p.parseSSEStream(resp.Body, eventChan)
}

// parseSSEStream parses the OpenAI-compatible SSE stream, handling content,
// tool calls, usage, and optionally reasoning_content (Grok).
func (p *OAIBaseProvider) parseSSEStream(body io.Reader, eventChan chan<- StreamEvent) (*oaiStreamResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large SSE events
	result := &oaiStreamResult{}

	toolCallsMap := make(map[int]*oaiToolCall)

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
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ToolCalls        []struct {
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

		// Handle reasoning_content (Grok-specific; harmless no-op for providers
		// where hasReasoningContent is false since the field will be empty)
		if p.hasReasoningContent && delta.ReasoningContent != "" {
			eventChan <- StreamEvent{Type: "thinking", Thinking: delta.ReasoningContent}
		}

		// Handle tool calls
		for _, tc := range delta.ToolCalls {
			if tc.ID != "" {
				toolCallsMap[tc.Index] = &oaiToolCall{
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
			} else {
				fmt.Fprintf(os.Stderr, "warning: failed to parse tool call args for %q: %v\n", tc.name, err)
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
