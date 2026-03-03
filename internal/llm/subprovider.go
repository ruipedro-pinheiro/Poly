package llm

import (
	"context"
	"errors"

	"github.com/pedromelo/poly/internal/tools"
)

// LazySubProvider implements tools.SubProvider by resolving the provider at call time.
// This avoids issues with providers not being configured at init time.
type LazySubProvider struct{}

// SubStream implements tools.SubProvider.
func (l *LazySubProvider) SubStream(systemPrompt string, messages []tools.SubMessage, toolDefs []ToolDefinition) <-chan tools.SubStreamEvent {
	out := make(chan tools.SubStreamEvent, 64)

	go func() {
		defer close(out)

		// Resolve provider at call time
		provider := resolveProvider()
		if provider == nil {
			out <- tools.SubStreamEvent{
				Type:  "error",
				Error: errors.New("no configured provider available for sub-task delegation"),
			}
			return
		}

		// Convert tools.SubMessage -> llm.Message (SubMessage is a subset of Message)
		llmMessages := make([]Message, 0, len(messages))
		for i, msg := range messages {
			content := msg.Content
			if i == 0 && msg.Role == "user" {
				content = "[System: " + systemPrompt + "]\n\n" + content
			}
			llmMessages = append(llmMessages, Message{
				Role:    msg.Role,
				Content: content,
			})
		}

		// ToolDefinition is now a shared type alias — no conversion needed.
		// tools.ToolCall and llm.ToolCall are the same type.

		events := provider.Stream(context.Background(), llmMessages, toolDefs)
		for event := range events {
			switch event.Type {
			case "content":
				out <- tools.SubStreamEvent{Type: "content", Content: event.Content}
			case "tool_use":
				if event.ToolCall != nil {
					// Same underlying type — direct pointer assignment
					out <- tools.SubStreamEvent{
						Type:     "tool_use",
						ToolCall: event.ToolCall,
					}
				}
			case "error":
				if event.Error != nil {
					out <- tools.SubStreamEvent{Type: "error", Error: event.Error}
					return
				}
			case "done":
				// don't forward done here, we send our own at the end
			}
		}

		out <- tools.SubStreamEvent{Type: "done"}
	}()

	return out
}

// resolveProvider finds the best configured provider for sub-tasks.
// Prefers cheapest provider to minimize costs.
func resolveProvider() Provider {
	providers := GetConfiguredProviders()
	if len(providers) == 0 {
		return nil
	}

	// Pick the cheapest configured provider
	best := providers[0]
	bestTier := GetProviderCostTier(best.Name())
	for _, p := range providers[1:] {
		tier := GetProviderCostTier(p.Name())
		if tier < bestTier {
			best = p
			bestTier = tier
		}
	}
	return best
}

// InitSubProvider registers a lazy sub-provider that resolves at call time.
func InitSubProvider() {
	tools.SetSubProvider(&LazySubProvider{})
}
