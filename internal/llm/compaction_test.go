package llm

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     int
	}{
		{
			name:     "empty messages",
			messages: nil,
			want:     0,
		},
		{
			name:     "empty slice",
			messages: []Message{},
			want:     0,
		},
		{
			name: "single short message",
			messages: []Message{
				{Role: "user", Content: "hello"},
			},
			want: 1, // 5 chars / 4 = 1
		},
		{
			name: "single empty content",
			messages: []Message{
				{Role: "user", Content: ""},
			},
			want: 0,
		},
		{
			name: "message with tool calls",
			messages: []Message{
				{
					Role:    "assistant",
					Content: "Let me check",
					ToolCalls: []ToolCall{
						{
							ID:   "call_1",
							Name: "read_file",
							Arguments: map[string]interface{}{
								"path": "/tmp/test.txt",
							},
						},
					},
				},
			},
			// "Let me check" = 12 chars / 4 = 3
			// tool call overhead = 50
			// "/tmp/test.txt" = 13 chars / 4 = 3
			want: 56,
		},
		{
			name: "message with tool result",
			messages: []Message{
				{
					Role:       "user",
					Content:    "",
					ToolResult: &ToolResult{Content: "file contents here"},
				},
			},
			// "" = 0
			// "file contents here" = 18 / 4 = 4
			want: 4,
		},
		{
			name: "long message proportional",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("a", 400)},
			},
			want: 100, // 400 / 4
		},
		{
			name: "multiple messages",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("x", 40)},
				{Role: "assistant", Content: strings.Repeat("y", 80)},
			},
			want: 30, // 40/4 + 80/4 = 10 + 20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.messages)
			if got != tt.want {
				t.Errorf("EstimateTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNeedsCompaction(t *testing.T) {
	tests := []struct {
		name          string
		messages      []Message
		contextWindow int
		want          bool
	}{
		{
			name:          "empty messages - no compaction",
			messages:      nil,
			contextWindow: 1000,
			want:          false,
		},
		{
			name: "below threshold - no compaction",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("a", 100)},
			},
			contextWindow: 1000,
			// 100/4 = 25 tokens, threshold = 800, 25 < 800
			want: false,
		},
		{
			name: "above threshold - needs compaction",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("a", 4000)},
			},
			contextWindow: 1000,
			// 4000/4 = 1000 tokens, threshold = 800, 1000 > 800
			want: true,
		},
		{
			name: "exactly at threshold - no compaction",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("a", 3200)},
			},
			contextWindow: 1000,
			// 3200/4 = 800 tokens, threshold = 800, 800 > 800 is false
			want: false,
		},
		{
			name: "zero context window uses default",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("a", 4)},
			},
			contextWindow: 0,
			// 4/4 = 1 token, default window 200000, threshold = 160000
			want: false,
		},
		{
			name: "negative context window uses default",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("a", 4)},
			},
			contextWindow: -1,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsCompaction(tt.messages, tt.contextWindow)
			if got != tt.want {
				t.Errorf("NeedsCompaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompactionConstants(t *testing.T) {
	if DefaultContextWindow != 200000 {
		t.Errorf("DefaultContextWindow = %d, want 200000", DefaultContextWindow)
	}
	if CompactionThreshold != 0.80 {
		t.Errorf("CompactionThreshold = %f, want 0.80", CompactionThreshold)
	}
	if MinMessagesToKeep != 6 {
		t.Errorf("MinMessagesToKeep = %d, want 6", MinMessagesToKeep)
	}
	if CharsPerToken != 4 {
		t.Errorf("CharsPerToken = %d, want 4", CharsPerToken)
	}
}
