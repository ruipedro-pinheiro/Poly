package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCustomProvider_Defaults(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{
		ID:   "test",
		Name: "Test Provider",
	})

	if p.config.AuthHeader != "Bearer" {
		t.Errorf("expected default AuthHeader=Bearer, got %q", p.config.AuthHeader)
	}
	if p.config.Color != "#888888" {
		t.Errorf("expected default Color=#888888, got %q", p.config.Color)
	}
}

func TestNewCustomProvider_PreservesValues(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{
		ID:         "custom",
		Name:       "Custom",
		AuthHeader: "x-api-key",
		Color:      "#FF0000",
		MaxTokens:  8192,
	})

	if p.config.AuthHeader != "x-api-key" {
		t.Errorf("expected AuthHeader=x-api-key, got %q", p.config.AuthHeader)
	}
	if p.config.Color != "#FF0000" {
		t.Errorf("expected Color=#FF0000, got %q", p.config.Color)
	}
	if p.config.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens=8192, got %d", p.config.MaxTokens)
	}
}

func TestCustomProvider_Name(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{ID: "mistral", Name: "Mistral"})
	if p.Name() != "mistral" {
		t.Errorf("expected Name()=mistral, got %q", p.Name())
	}
}

func TestCustomProvider_DisplayName(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{ID: "mistral", Name: "Mistral"})
	if p.DisplayName() != "Mistral" {
		t.Errorf("expected DisplayName()=Mistral, got %q", p.DisplayName())
	}
}

func TestCustomProvider_Color(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{ID: "test", Color: "#ABCDEF"})
	if p.Color() != "#ABCDEF" {
		t.Errorf("expected Color()=#ABCDEF, got %q", p.Color())
	}
}

func TestCustomProvider_GetSetModel(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{ID: "test", Model: "model-v1"})
	if p.GetModel() != "model-v1" {
		t.Errorf("expected model-v1, got %q", p.GetModel())
	}

	p.SetModel("model-v2")
	if p.GetModel() != "model-v2" {
		t.Errorf("expected model-v2 after SetModel, got %q", p.GetModel())
	}
}

func TestCustomProvider_SupportsTools(t *testing.T) {
	p := NewCustomProvider(CustomProviderConfig{ID: "test"})
	if !p.SupportsTools() {
		t.Error("expected SupportsTools()=true")
	}
}

func TestCustomProvider_IsConfigured(t *testing.T) {
	// Without API key
	p := NewCustomProvider(CustomProviderConfig{ID: "test"})
	if p.IsConfigured() {
		t.Error("expected IsConfigured()=false without API key")
	}

	// With API key
	p = NewCustomProvider(CustomProviderConfig{ID: "test", APIKey: "sk-test"})
	if !p.IsConfigured() {
		t.Error("expected IsConfigured()=true with API key")
	}
}

func TestCustomProvider_ToolFormat(t *testing.T) {
	tests := []struct {
		format string
		want   ToolFormat
	}{
		{"anthropic", ToolFormatAnthropic},
		{"openai", ToolFormatOpenAI},
		{"google", ToolFormatGoogle},
		{"", ToolFormatOpenAI},          // default
		{"unknown", ToolFormatOpenAI},   // fallback
	}

	for _, tc := range tests {
		p := NewCustomProvider(CustomProviderConfig{ID: "test", Format: tc.format})
		if p.ToolFormat() != tc.want {
			t.Errorf("Format %q: expected ToolFormat=%s, got %s", tc.format, tc.want, p.ToolFormat())
		}
	}
}

func TestSaveAndLoadCustomProviders(t *testing.T) {
	// Use temp dir
	tmpDir := t.TempDir()
	oldFile := customProvidersFile
	customProvidersFile = filepath.Join(tmpDir, "providers.json")
	defer func() { customProvidersFile = oldFile }()

	cfg := CustomProviderConfig{
		ID:      "test-save",
		Name:    "Test Save",
		BaseURL: "https://api.test.com/v1",
		APIKey:  "sk-test",
		Model:   "test-model",
		Format:  "openai",
		Color:   "#123456",
	}

	// Save
	err := SaveCustomProvider(cfg)
	if err != nil {
		t.Fatalf("SaveCustomProvider() error: %v", err)
	}

	// Load
	configs := GetCustomProviders()
	if len(configs) != 1 {
		t.Fatalf("expected 1 custom provider, got %d", len(configs))
	}
	if configs[0].ID != "test-save" {
		t.Errorf("expected ID=test-save, got %q", configs[0].ID)
	}
	if configs[0].BaseURL != "https://api.test.com/v1" {
		t.Errorf("expected BaseURL, got %q", configs[0].BaseURL)
	}
}

func TestSaveCustomProvider_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	oldFile := customProvidersFile
	customProvidersFile = filepath.Join(tmpDir, "providers.json")
	defer func() { customProvidersFile = oldFile }()

	// Save initial
	SaveCustomProvider(CustomProviderConfig{ID: "test", Name: "V1", Format: "openai"})

	// Update
	SaveCustomProvider(CustomProviderConfig{ID: "test", Name: "V2", Format: "openai"})

	configs := GetCustomProviders()
	if len(configs) != 1 {
		t.Fatalf("expected 1 provider after update, got %d", len(configs))
	}
	if configs[0].Name != "V2" {
		t.Errorf("expected Name=V2 after update, got %q", configs[0].Name)
	}
}

func TestDeleteCustomProvider(t *testing.T) {
	tmpDir := t.TempDir()
	oldFile := customProvidersFile
	customProvidersFile = filepath.Join(tmpDir, "providers.json")
	defer func() { customProvidersFile = oldFile }()

	SaveCustomProvider(CustomProviderConfig{ID: "keep", Name: "Keep", Format: "openai"})
	SaveCustomProvider(CustomProviderConfig{ID: "delete", Name: "Delete", Format: "openai"})

	err := DeleteCustomProvider("delete")
	if err != nil {
		t.Fatalf("DeleteCustomProvider() error: %v", err)
	}

	configs := GetCustomProviders()
	if len(configs) != 1 {
		t.Fatalf("expected 1 provider after delete, got %d", len(configs))
	}
	if configs[0].ID != "keep" {
		t.Errorf("expected remaining provider ID=keep, got %q", configs[0].ID)
	}
}

func TestGetCustomProviders_NoFile(t *testing.T) {
	oldFile := customProvidersFile
	customProvidersFile = "/nonexistent/path/providers.json"
	defer func() { customProvidersFile = oldFile }()

	configs := GetCustomProviders()
	if configs != nil {
		t.Errorf("expected nil for no file, got %v", configs)
	}
}

func TestLoadCustomProviders_NoFile(t *testing.T) {
	oldFile := customProvidersFile
	customProvidersFile = "/nonexistent/path/providers.json"
	defer func() { customProvidersFile = oldFile }()

	err := LoadCustomProviders()
	if err != nil {
		t.Errorf("LoadCustomProviders() should not error for missing file: %v", err)
	}
}

func TestSaveCustomProvider_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	oldFile := customProvidersFile
	customProvidersFile = filepath.Join(tmpDir, "providers.json")
	defer func() { customProvidersFile = oldFile }()

	SaveCustomProvider(CustomProviderConfig{ID: "test", Format: "openai"})

	info, err := os.Stat(customProvidersFile)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}
}

func TestCustomProviderConfig_JSON(t *testing.T) {
	cfg := CustomProviderConfig{
		ID:      "test",
		Name:    "Test",
		BaseURL: "https://api.test.com",
		Model:   "model-1",
		Format:  "openai",
		Color:   "#FF0000",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded CustomProviderConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ID != cfg.ID || decoded.Name != cfg.Name || decoded.Format != cfg.Format {
		t.Errorf("JSON roundtrip mismatch: got %+v", decoded)
	}
}
