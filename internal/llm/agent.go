package llm

import (
	"context"
	"strings"

	"github.com/pedromelo/poly/internal/tools"
)

// SingleTurnResult represents the output of a single LLM request (text + tool calls)
type SingleTurnResult struct {
	Content             string
	Thinking            string
	ToolCalls           []ToolCall
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
}

// RequestHandler is a function that performs a single LLM request and streams events.
// It returns the final accumulated result of the turn.
type RequestHandler func(ctx context.Context, history []Message, eventChan chan<- StreamEvent) (*SingleTurnResult, error)

// RunAgenticLoop is the universal brain of Poly. It handles the multi-turn 
// tool execution loop for ANY provider.
func RunAgenticLoop(ctx context.Context, providerName string, model string, initialMessages []Message, toolDefs []ToolDefinition, eventChan chan<- StreamEvent, requestHandler RequestHandler) {
	history := make([]Message, 0, len(initialMessages)+8)
	history = append(history, initialMessages...)

	var fullContent strings.Builder

	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		// Execute the turn via the provider-specific handler
		result, err := requestHandler(ctx, history, eventChan)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		fullContent.WriteString(result.Content)

		// If no tool calls, we are done
		if len(result.ToolCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:             fullContent.String(),
					Provider:            providerName,
					Model:               model,
					InputTokens:         result.InputTokens,
					OutputTokens:        result.OutputTokens,
					CacheCreationTokens: result.CacheCreationTokens,
					CacheReadTokens:     result.CacheReadTokens,
				},
			}
			return
		}

		// Add assistant's response (including tool calls) to history
		history = append(history, Message{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		// Execute tools and add results to history
		for _, tc := range result.ToolCalls {
			// Notify UI that a tool is starting
			eventChan <- StreamEvent{
				Type:     "tool_use",
				ToolCall: &tc,
			}

			// Execute the tool
			toolRes := tools.Execute(tc.Name, tc.Arguments)

			// Notify UI of the result
			eventChan <- StreamEvent{
				Type:       "tool_result",
				ToolCall:   &tc,
				ToolResult: &ToolResult{ToolUseID: tc.ID, Content: toolRes.Content, IsError: toolRes.IsError},
			}

			// Add result to history
			history = append(history, Message{
				Role: "user",
				ToolResult: &ToolResult{
					ToolUseID: tc.ID,
					Content:   toolRes.Content,
					IsError:   toolRes.IsError,
				},
			})
		}
	}

	// Max turns reached
	eventChan <- StreamEvent{Type: "content", Content: "\n⚠️ Max tool turns reached\n"}
	eventChan <- StreamEvent{
		Type: "done",
		Response: &Response{
			Content:  fullContent.String(),
			Provider: providerName,
			Model:    model,
		},
	}
}
