package llm

import (
	"testing"
)

func TestProviderRegistry(t *testing.T) {
	// Test getting all providers
	providers := GetAllProviders()
	
	if len(providers) == 0 {
		t.Error("Expected at least one provider registered")
	}

	// Verify standard providers exist (if configured)
	expectedNames := []string{"claude", "gpt", "gemini", "grok"}
	for _, name := range expectedNames {
		if p, exists := providers[name]; exists {
			if p.Name() != name {
				t.Errorf("Provider %s: Name() returned %s", name, p.Name())
			}
		}
	}
}

func TestGetProvider(t *testing.T) {
	// Test existing provider
	provider, ok := GetProvider("claude")
	if ok {
		if provider == nil {
			t.Fatal("Provider should not be nil when ok=true")
		}
		if provider.Name() != "claude" {
			t.Errorf("Expected provider Name()=claude, got %s", provider.Name())
		}
		if provider.DisplayName() == "" {
			t.Error("DisplayName should not be empty")
		}
		if provider.Color() == "" {
			t.Error("Color should not be empty")
		}
	}

	// Test non-existent provider
	_, ok = GetProvider("nonexistent")
	if ok {
		t.Error("Expected ok=false for non-existent provider")
	}
}

func TestGetConfiguredProviders(t *testing.T) {
	providers := GetConfiguredProviders()
	
	// All returned providers should be configured
	for _, p := range providers {
		if !p.IsConfigured() {
			t.Errorf("Provider %s returned by GetConfiguredProviders but IsConfigured()=false", p.Name())
		}
	}
}

func TestGetProviderNames(t *testing.T) {
	names := GetProviderNames()
	
	if len(names) == 0 {
		t.Error("Expected at least one provider name")
	}

	// Names should not be empty strings
	for _, name := range names {
		if name == "" {
			t.Error("Provider name should not be empty")
		}
	}
}

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		isValid bool
	}{
		{
			name:    "valid user message",
			msg:     Message{Role: "user", Content: "Hello"},
			isValid: true,
		},
		{
			name:    "valid assistant message",
			msg:     Message{Role: "assistant", Content: "Hi there"},
			isValid: true,
		},
		{
			name:    "valid system message",
			msg:     Message{Role: "system", Content: "You are helpful"},
			isValid: true,
		},
		{
			name:    "empty content",
			msg:     Message{Role: "user", Content: ""},
			isValid: false,
		},
		{
			name:    "invalid role",
			msg:     Message{Role: "invalid", Content: "test"},
			isValid: false,
		},
		{
			name:    "empty role",
			msg:     Message{Role: "", Content: "test"},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.msg.Role != "" && tt.msg.Content != "" && 
				(tt.msg.Role == "user" || tt.msg.Role == "assistant" || tt.msg.Role == "system")
			
			if valid != tt.isValid {
				t.Errorf("Expected valid=%v for message %+v, got %v", tt.isValid, tt.msg, valid)
			}
		})
	}
}

func TestToolCallStructure(t *testing.T) {
	tc := ToolCall{
		ID:   "call_123",
		Name: "read_file",
		Arguments: map[string]interface{}{
			"path": "/tmp/test.txt",
		},
	}

	if tc.ID != "call_123" {
		t.Errorf("Expected ID=call_123, got %s", tc.ID)
	}

	if tc.Name != "read_file" {
		t.Errorf("Expected Name=read_file, got %s", tc.Name)
	}

	path, ok := tc.Arguments["path"].(string)
	if !ok {
		t.Fatal("Expected path argument to be string")
	}
	if path != "/tmp/test.txt" {
		t.Errorf("Expected path=/tmp/test.txt, got %s", path)
	}
}

func TestImageSupport(t *testing.T) {
	// Test that image support can be queried
	_ = SupportsImages("claude")
	_ = SupportsImages("gpt")
	
	// Test setting image support
	SetImageSupport("test-provider", true)
	if !SupportsImages("test-provider") {
		t.Error("Expected test-provider to support images after SetImageSupport(true)")
	}
	
	SetImageSupport("test-provider", false)
	if SupportsImages("test-provider") {
		t.Error("Expected test-provider to not support images after SetImageSupport(false)")
	}
}
