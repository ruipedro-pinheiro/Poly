package llm

import (
	"errors"
	"testing"

	"github.com/pedromelo/poly/internal/config"
)

func TestGetRole_Empty(t *testing.T) {
	role := GetRole(nil)
	if role != "default" {
		t.Errorf("expected 'default', got %q", role)
	}
}

func TestGetRole_EmptySlice(t *testing.T) {
	role := GetRole([]StreamOptions{})
	if role != "default" {
		t.Errorf("expected 'default', got %q", role)
	}
}

func TestGetRole_WithRole(t *testing.T) {
	role := GetRole([]StreamOptions{{Role: "reviewer"}})
	if role != "reviewer" {
		t.Errorf("expected 'reviewer', got %q", role)
	}
}

func TestGetRole_EmptyRole(t *testing.T) {
	role := GetRole([]StreamOptions{{Role: ""}})
	if role != "default" {
		t.Errorf("expected 'default' for empty role, got %q", role)
	}
}

func TestGetThinkingMode_Empty(t *testing.T) {
	mode := GetThinkingMode(nil)
	if mode {
		t.Error("expected false for nil opts")
	}
}

func TestGetThinkingMode_False(t *testing.T) {
	mode := GetThinkingMode([]StreamOptions{{ThinkingMode: false}})
	if mode {
		t.Error("expected false")
	}
}

func TestGetThinkingMode_True(t *testing.T) {
	mode := GetThinkingMode([]StreamOptions{{ThinkingMode: true}})
	if !mode {
		t.Error("expected true")
	}
}

func TestIsImageError_Nil(t *testing.T) {
	if IsImageError(nil) {
		t.Error("expected false for nil error")
	}
}

func TestIsImageError_Unrelated(t *testing.T) {
	if IsImageError(errors.New("connection refused")) {
		t.Error("expected false for unrelated error")
	}
}

func TestIsImageError_NotSupported(t *testing.T) {
	if !IsImageError(errors.New("image inputs are not supported")) {
		t.Error("expected true for 'image inputs are not supported'")
	}
}

func TestIsImageError_Unsupported(t *testing.T) {
	if !IsImageError(errors.New("unsupported image format")) {
		t.Error("expected true for 'unsupported image'")
	}
}

func TestIsImageError_Invalid(t *testing.T) {
	if !IsImageError(errors.New("invalid image data")) {
		t.Error("expected true for 'invalid image'")
	}
}

func TestGetProviderCostTier_Known(t *testing.T) {
	tier := GetProviderCostTier("claude")
	if tier != 3 {
		t.Errorf("expected claude cost tier=3, got %d", tier)
	}
}

func TestGetProviderCostTier_Unknown(t *testing.T) {
	tier := GetProviderCostTier("nonexistent")
	if tier != 2 {
		t.Errorf("expected default cost tier=2, got %d", tier)
	}
}

func TestGetDefaultModel(t *testing.T) {
	model := GetDefaultModel("claude")
	if model == "" {
		t.Error("expected non-empty default model for claude")
	}
}

func TestGetDefaultModel_Unknown(t *testing.T) {
	model := GetDefaultModel("nonexistent")
	if model != "" {
		t.Errorf("expected empty string for unknown provider, got %q", model)
	}
}

func TestGetProviderEndpoint(t *testing.T) {
	endpoint := GetProviderEndpoint("claude")
	if endpoint == "" {
		t.Error("expected non-empty endpoint for claude")
	}
}

func TestGetProviderEndpoint_Unknown(t *testing.T) {
	endpoint := GetProviderEndpoint("nonexistent")
	if endpoint != "" {
		t.Errorf("expected empty string for unknown provider, got %q", endpoint)
	}
}

func TestGetProviderMaxTokens(t *testing.T) {
	tokens := GetProviderMaxTokens("claude")
	if tokens <= 0 {
		t.Errorf("expected positive max tokens for claude, got %d", tokens)
	}
}

func TestGetProviderMaxTokens_Unknown(t *testing.T) {
	tokens := GetProviderMaxTokens("nonexistent")
	if tokens != 4096 {
		t.Errorf("expected default 4096 for unknown provider, got %d", tokens)
	}
}

func TestGetMaxToolTurns_Default(t *testing.T) {
	turns := GetMaxToolTurns()
	if turns <= 0 {
		t.Errorf("expected positive max tool turns, got %d", turns)
	}
}

func TestGetMaxToolTurns_Custom(t *testing.T) {
	config.SetForTest(&config.Config{
		Providers: map[string]config.ProviderConfig{},
		Settings:  config.SettingsConfig{MaxToolTurns: 25},
	})
	defer config.SetForTest(nil)

	// Need to reload since we bypassed the normal path
	turns := GetMaxToolTurns()
	if turns != 25 {
		t.Errorf("expected 25, got %d", turns)
	}
}

func TestGetModelVariants(t *testing.T) {
	variants := GetModelVariants()
	if len(variants) == 0 {
		t.Error("expected at least one provider with model variants")
	}
	// Claude should have models
	if _, ok := variants["claude"]; !ok {
		t.Error("expected claude in model variants")
	}
}

func TestToolFormatConstants(t *testing.T) {
	if ToolFormatAnthropic != "anthropic" {
		t.Errorf("expected ToolFormatAnthropic=anthropic, got %q", ToolFormatAnthropic)
	}
	if ToolFormatOpenAI != "openai" {
		t.Errorf("expected ToolFormatOpenAI=openai, got %q", ToolFormatOpenAI)
	}
	if ToolFormatGoogle != "google" {
		t.Errorf("expected ToolFormatGoogle=google, got %q", ToolFormatGoogle)
	}
}

func TestClaudeOAuthSystemPrompt(t *testing.T) {
	if ClaudeOAuthSystemPrompt == "" {
		t.Error("ClaudeOAuthSystemPrompt should not be empty")
	}
}
