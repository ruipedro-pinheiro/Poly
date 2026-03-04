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

	"github.com/pedromelo/poly/internal/auth"
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
// providers (GPT, Grok, Copilot).
type OAIBaseProvider struct {
	providerID  string
	displayName string
	color       string
	endpoint    string
	config      ProviderConfig
	httpClient  *http.Client

	// Hooks for provider-specific behavior
	setHeaders               func(req *http.Request, token string)
	handleStreamError        func(err error, currentToken string) (newToken string, retryErr error)
	hasReasoningContent      bool
	alwaysUseReasoningTokens bool
	authError                string
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
		defer func() {
			if r := recover(); r != nil {
				defer func() { recover() }() //nolint:errcheck // protect against closed channel
				eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("internal error (panic recovered)")}
			}
			close(eventChan)
		}()

		token, err := auth.GetStorage().GetAccessToken(p.providerID)
		if err != nil || token == "" {
			errMsg := p.authError
			if errMsg == "" {
				errMsg = "no API key configured"
			}
			eventChan <- StreamEvent{Type: "error", Error: errors.New(errMsg)}
			return
		}

		thinkingMode := GetThinkingMode(opts)
		role := GetRole(opts)

		// Define the request handler for the agentic loop
		handler := func(ctx context.Context, history []Message, internalChan chan<- StreamEvent) (*SingleTurnResult, error) {
			oaiHistory := p.buildOAIHistory(history, role)
			oaiTools := OAIToolDefsFromPoly(toolDefs)
			body := p.buildRequestBody(oaiHistory, oaiTools, thinkingMode)

			// Emit thinking indicator for reasoning models
			if thinkingMode && IsReasoningModel(p.providerID, p.config.Model) {
				internalChan <- StreamEvent{Type: "thinking", Thinking: "(reasoning...)"}
			}

			res, err := p.streamRequest(ctx, body, token, internalChan)
			if err != nil {
				// Allow provider-specific error recovery (e.g. Copilot 401 token refresh)
				if p.handleStreamError != nil {
					newToken, retryErr := p.handleStreamError(err, token)
					if retryErr == nil && newToken != "" {
						token = newToken
						res, err = p.streamRequest(ctx, body, token, internalChan)
					}
				}
				if err != nil {
					return nil, err
				}
			}

			// Convert oaiStreamResult to generic SingleTurnResult
			result := &SingleTurnResult{
				Content:      res.content,
				InputTokens:  res.inputTokens,
				OutputTokens: res.outputTokens,
			}

			for _, tc := range res.toolCalls {
				result.ToolCalls = append(result.ToolCalls, ToolCall{
					ID:        tc.id,
					Name:      tc.name,
					Arguments: tc.input,
				})
			}

			return result, nil
		}

		// Run the universal agentic loop
		RunAgenticLoop(ctx, p.providerID, p.config.Model, messages, toolDefs, eventChan, handler)
	}()

	return eventChan
}

func (p *OAIBaseProvider) buildOAIHistory(messages []Message, role string) []OAIMessage {
	history := make([]OAIMessage, 0, len(messages)+1)
	history = append(history, NewOAITextMessage("system", BuildSystemPrompt(p.providerID, role)))

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
	if p.endpoint == "" {
		return nil, fmt.Errorf("provider %q has no endpoint configured", p.providerID)
	}
	if !strings.HasPrefix(p.endpoint, "http://") && !strings.HasPrefix(p.endpoint, "https://") {
		return nil, fmt.Errorf("provider %q endpoint %q must start with http:// or https://", p.providerID, p.endpoint)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
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

	resp, err := DoWithRetry(ctx, p.httpClient, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseSSEStream(resp.Body, eventChan)
}

func (p *OAIBaseProvider) parseSSEStream(body io.Reader, eventChan chan<- StreamEvent) (*oaiStreamResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
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
		if p.hasReasoningContent && delta.ReasoningContent != "" {
			eventChan <- StreamEvent{Type: "thinking", Thinking: delta.ReasoningContent}
		}

		for _, tc := range delta.ToolCalls {
			if tc.ID != "" {
				toolCallsMap[tc.Index] = &oaiToolCall{id: tc.ID, name: tc.Function.Name}
			}
			if tc.Function.Arguments != "" && toolCallsMap[tc.Index] != nil {
				toolCallsMap[tc.Index].rawArgs += tc.Function.Arguments
			}
		}

		reason := strings.ToLower(event.Choices[0].FinishReason)
		if reason == "tool_calls" || reason == "stop" {
			break
		}
	}

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

	return result, scanner.Err()
}
