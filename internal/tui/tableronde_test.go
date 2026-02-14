package tui

import (
	"testing"

	"github.com/pedromelo/poly/internal/config"
)

// initTestConfig forces a default config load so config.GetProviderNames()
// returns the standard providers (claude, gemini, gpt, grok).
func initTestConfig(t *testing.T) {
	t.Helper()
	// Get() will load DefaultConfig if nothing was loaded yet.
	// This gives us claude, gpt, gemini, grok as provider names.
	cfg := config.Get()
	if cfg == nil {
		t.Fatal("config.Get() returned nil")
	}
}

func TestExtractMentions_NilTableRonde(t *testing.T) {
	m := &Model{}
	mentions := m.extractMentions()
	if mentions != nil {
		t.Errorf("expected nil mentions when tableRonde is nil, got %v", mentions)
	}
}

func TestExtractMentions_NoMentions(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "This is a normal response without any mentions.", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 0 {
		t.Errorf("expected 0 mentions, got %d: %v", len(mentions), mentions)
	}
}

func TestExtractMentions_SelfMention(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "I am @claude and I think this is good.", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 0 {
		t.Errorf("expected 0 mentions (self-mention should be skipped), got %d", len(mentions))
	}
}

func TestExtractMentions_ValidMention(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "I think @gpt should weigh in on this.", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}
	if mentions[0].target != "gpt" {
		t.Errorf("expected target=gpt, got %s", mentions[0].target)
	}
	if mentions[0].by != "claude" {
		t.Errorf("expected by=claude, got %s", mentions[0].by)
	}
}

func TestExtractMentions_MultipleMentionsFromMultipleProviders(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "Let me ask @gpt about this.", Provider: "claude"},
			{Role: "assistant", Content: "I agree, and @gemini should confirm.", Provider: "gpt"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0, "gpt": 1},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 2 {
		t.Fatalf("expected 2 mentions, got %d: %v", len(mentions), mentions)
	}

	targets := map[string]bool{}
	for _, mention := range mentions {
		targets[mention.target] = true
	}
	if !targets["gpt"] {
		t.Error("expected gpt to be mentioned")
	}
	if !targets["gemini"] {
		t.Error("expected gemini to be mentioned")
	}
}

func TestExtractMentions_CaseInsensitive(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "Hey @GPT, what do you think?", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention (case-insensitive), got %d", len(mentions))
	}
	if mentions[0].target != "gpt" {
		t.Errorf("expected target=gpt, got %s", mentions[0].target)
	}
}

func TestExtractMentions_Dedup(t *testing.T) {
	initTestConfig(t)

	// Both claude and gemini mention gpt -- gpt should appear only once via the seen map
	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "Ask @gpt for details.", Provider: "claude"},
			{Role: "assistant", Content: "I also want @gpt to respond.", Provider: "gemini"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0, "gemini": 1},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention (dedup), got %d: %v", len(mentions), mentions)
	}
	if mentions[0].target != "gpt" {
		t.Errorf("expected target=gpt, got %s", mentions[0].target)
	}
}

func TestExtractMentions_IndexOutOfBounds(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "@gpt help", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{
				"claude": 0,
				"gpt":    99, // out of bounds - should be skipped
			},
		},
	}

	mentions := m.extractMentions()
	// claude at index 0 mentions gpt (valid)
	// gpt at index 99 is out of bounds (skipped)
	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}
	if mentions[0].target != "gpt" {
		t.Errorf("expected target=gpt, got %s", mentions[0].target)
	}
}

func TestExtractMentions_MultipleMentionsInSingleMessage(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "Both @gpt and @gemini should look at this.", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 2 {
		t.Fatalf("expected 2 mentions, got %d", len(mentions))
	}

	targets := map[string]bool{}
	for _, mention := range mentions {
		targets[mention.target] = true
	}
	if !targets["gpt"] {
		t.Error("expected gpt in mentions")
	}
	if !targets["gemini"] {
		t.Error("expected gemini in mentions")
	}
}

func TestExtractMentions_EmptyMessages(t *testing.T) {
	initTestConfig(t)

	m := &Model{
		messages: []Message{
			{Role: "assistant", Content: "", Provider: "claude"},
		},
		tableRonde: &tableRondeState{
			messageIndices: map[string]int{"claude": 0},
		},
	}

	mentions := m.extractMentions()
	if len(mentions) != 0 {
		t.Errorf("expected 0 mentions for empty content, got %d", len(mentions))
	}
}
