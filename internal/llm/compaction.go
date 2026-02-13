package llm

import (
	"context"
	"fmt"
	"strings"
)

const (
	DefaultContextWindow = 200000 // tokens (Anthropic default)
	CompactionThreshold  = 0.80   // 80%
	MinMessagesToKeep    = 6      // keep at least the 6 most recent messages
	CharsPerToken        = 4      // rough approximation
)

// EstimateTokens estimates token count from messages (rough: ~4 chars per token)
func EstimateTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / CharsPerToken
		for _, tc := range m.ToolCalls {
			total += 50 // overhead per tool call
			// Count arguments size
			for _, v := range tc.Arguments {
				total += len(fmt.Sprintf("%v", v)) / CharsPerToken
			}
		}
		if m.ToolResult != nil {
			total += len(m.ToolResult.Content) / CharsPerToken
		}
	}
	return total
}

// NeedsCompaction returns true if messages exceed the threshold
func NeedsCompaction(messages []Message, contextWindow int) bool {
	if contextWindow <= 0 {
		contextWindow = DefaultContextWindow
	}
	estimated := EstimateTokens(messages)
	threshold := int(float64(contextWindow) * CompactionThreshold)
	return estimated > threshold
}

// CompactMessages summarizes old messages using the provider, keeping recent ones intact.
func CompactMessages(ctx context.Context, provider Provider, messages []Message, keepLast int) ([]Message, error) {
	if keepLast < MinMessagesToKeep {
		keepLast = MinMessagesToKeep
	}
	if len(messages) <= keepLast {
		return messages, nil
	}

	oldMessages := messages[:len(messages)-keepLast]
	recentMessages := messages[len(messages)-keepLast:]

	// Build the summary prompt from old messages
	var summaryContent strings.Builder
	summaryContent.WriteString("Summarize the following conversation concisely. ")
	summaryContent.WriteString("Preserve: key decisions, file paths, code changes, errors encountered, and user preferences. ")
	summaryContent.WriteString("Be factual and brief.\n\n")
	for _, m := range oldMessages {
		summaryContent.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, truncate(m.Content, 500)))
	}

	summaryMessages := []Message{
		{Role: "user", Content: summaryContent.String()},
	}

	// Use the provider to generate the summary
	events := provider.Stream(ctx, summaryMessages, nil)
	var summary strings.Builder
	for event := range events {
		if event.Type == "content" {
			summary.WriteString(event.Content)
		}
		if event.Error != nil {
			return messages, fmt.Errorf("compaction failed: %w", event.Error)
		}
	}

	// Build the compacted message list
	compacted := make([]Message, 0, len(recentMessages)+2)
	compacted = append(compacted, Message{
		Role:    "user",
		Content: "[Previous conversation summary]\n" + summary.String(),
	})
	compacted = append(compacted, Message{
		Role:    "assistant",
		Content: "Understood. I have the context from the previous conversation. Let's continue.",
	})
	compacted = append(compacted, recentMessages...)
	return compacted, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
