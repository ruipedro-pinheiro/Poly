package llm

import (
	"strings"
	"testing"

	"github.com/pedromelo/poly/internal/config"
)

func TestGetMaxTableRounds_Default(t *testing.T) {
	// Ensure config is loaded (default has MaxTableRounds=0)
	cfg := config.Get()
	if cfg == nil {
		t.Fatal("config.Get() returned nil")
	}
	// Default config has MaxTableRounds = 0, so GetMaxTableRounds should return 5
	got := GetMaxTableRounds()
	if got != 5 {
		t.Errorf("GetMaxTableRounds() default = %d, want 5", got)
	}
}

func TestGetMaxTableRounds_CustomValue(t *testing.T) {
	// Set a custom value
	config.SetMaxTableRounds(3)
	defer config.SetMaxTableRounds(0) // reset to default after test

	got := GetMaxTableRounds()
	if got != 3 {
		t.Errorf("GetMaxTableRounds() with config=3 = %d, want 3", got)
	}
}

func TestGetMaxTableRounds_ZeroFallsBackToDefault(t *testing.T) {
	config.SetMaxTableRounds(0)

	got := GetMaxTableRounds()
	if got != 5 {
		t.Errorf("GetMaxTableRounds() with config=0 = %d, want 5 (default)", got)
	}
}

func TestBuildSystemPrompt_ContainsParticipantRole(t *testing.T) {
	prompt := BuildSystemPrompt("claude", "participant")
	if !strings.Contains(prompt, "TABLE RONDE PARTICIPANT") {
		t.Error("participant role should contain TABLE RONDE PARTICIPANT section")
	}
	if !strings.Contains(prompt, "@mention") {
		t.Error("participant role should mention @mention capability")
	}
}

func TestBuildSystemPrompt_ContainsGroundTruth(t *testing.T) {
	prompt := BuildSystemPrompt("claude", "default")
	if !strings.Contains(prompt, "GROUND TRUTH") {
		t.Error("system prompt should contain GROUND TRUTH section")
	}
	if !strings.Contains(prompt, "FACT 1") {
		t.Error("system prompt should contain FACT 1")
	}
}

func TestBuildSystemPrompt_ProviderIdentity(t *testing.T) {
	prompt := BuildSystemPrompt("claude", "default")
	if !strings.Contains(prompt, "Claude") {
		t.Error("system prompt for claude should contain Claude display name")
	}
}

func TestBuildSystemPrompt_EmptyProvider(t *testing.T) {
	prompt := BuildSystemPrompt("", "default")
	// Should not contain FACT 1 (provider identity) when no provider specified
	if strings.Contains(prompt, "FACT 1:") {
		t.Error("system prompt with empty provider should not contain FACT 1")
	}
	// Should still have GROUND TRUTH section
	if !strings.Contains(prompt, "GROUND TRUTH") {
		t.Error("system prompt should still have GROUND TRUTH even without provider")
	}
}

func TestBuildSystemPrompt_ReviewerRole(t *testing.T) {
	prompt := BuildSystemPrompt("gpt", "reviewer")
	if !strings.Contains(prompt, "REVIEWER") {
		t.Error("reviewer role should contain REVIEWER section")
	}
}

func TestBuildSystemPrompt_ResponderRole(t *testing.T) {
	prompt := BuildSystemPrompt("gpt", "responder")
	if !strings.Contains(prompt, "FIRST RESPONDER") {
		t.Error("responder role should contain FIRST RESPONDER section")
	}
}

func TestBuildSystemPrompt_NilConfig(t *testing.T) {
	// This test verifies the nil config fallback.
	// Since config.Get() auto-loads defaults, this path is hard to trigger
	// in normal circumstances, but we verify the function doesn't panic.
	prompt := BuildSystemPrompt("test", "default")
	if prompt == "" {
		t.Error("system prompt should never be empty")
	}
}
