package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
)

// jsonEvent is the NDJSON output format for streaming events.
type jsonEvent struct {
	Type        string                 `json:"type"`
	Content     string                 `json:"content,omitempty"`
	Thinking    string                 `json:"thinking,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      string                 `json:"output,omitempty"`
	IsError     bool                   `json:"is_error,omitempty"`
	Model       string                 `json:"model,omitempty"`
	InputTokens int                    `json:"input_tokens,omitempty"`
	OutputTokens int                   `json:"output_tokens,omitempty"`
	Message     string                 `json:"message,omitempty"`
}

// RunJSON handles --json mode: sends prompt to default provider,
// streams response as NDJSON to stdout, returns exit code.
func RunJSON(prompt string) int {
	cfg := config.Get()

	providerName := cfg.DefaultProvider
	if providerName == "" {
		providerName = "claude"
	}

	provider, ok := llm.GetProvider(providerName)
	if !ok {
		writeJSONError("provider %q not found", providerName)
		return 1
	}

	if !provider.IsConfigured() {
		writeJSONError("provider %q is not configured", providerName)
		return 1
	}

	if model := llm.GetDefaultModel(providerName); model != "" {
		provider.SetModel(model)
	}

	systemPrompt := llm.BuildSystemPrompt(providerName, "default")

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	enc := json.NewEncoder(os.Stdout)

	ctx := context.Background()
	ch := provider.Stream(ctx, messages, nil)

	for event := range ch {
		switch event.Type {
		case "content":
			enc.Encode(jsonEvent{Type: "content", Content: event.Content})
		case "thinking":
			enc.Encode(jsonEvent{Type: "thinking", Thinking: event.Thinking})
		case "tool_use":
			evt := jsonEvent{Type: "tool_use"}
			if event.ToolCall != nil {
				evt.Name = event.ToolCall.Name
				evt.Input = event.ToolCall.Arguments
			}
			enc.Encode(evt)
		case "tool_result":
			evt := jsonEvent{Type: "tool_result"}
			if event.ToolResult != nil {
				evt.Output = event.ToolResult.Content
				evt.IsError = event.ToolResult.IsError
			}
			if event.ToolCall != nil {
				evt.Name = event.ToolCall.Name
			}
			enc.Encode(evt)
		case "done":
			evt := jsonEvent{Type: "done"}
			if event.Response != nil {
				evt.Model = event.Response.Model
				evt.InputTokens = event.Response.InputTokens
				evt.OutputTokens = event.Response.OutputTokens
			}
			enc.Encode(evt)
			return 0
		case "error":
			msg := "unknown error"
			if event.Error != nil {
				msg = event.Error.Error()
			}
			enc.Encode(jsonEvent{Type: "error", Message: msg})
			return 1
		}
	}

	return 0
}

func writeJSONError(format string, args ...interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(jsonEvent{Type: "error", Message: fmt.Sprintf(format, args...)})
}
