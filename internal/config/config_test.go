package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultProvider == "" {
		t.Error("DefaultProvider should not be empty")
	}

	if len(cfg.Providers) == 0 {
		t.Error("Providers should not be empty")
	}

	// Verify each provider has required fields
	for id, p := range cfg.Providers {
		if p.ID != id {
			t.Errorf("Provider %s: ID mismatch (got %s)", id, p.ID)
		}
		if p.Name == "" {
			t.Errorf("Provider %s: Name is empty", id)
		}
		if p.Format == "" {
			t.Errorf("Provider %s: Format is empty", id)
		}
		if p.AuthType == "" {
			t.Errorf("Provider %s: AuthType is empty", id)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temp config dir
	tmpDir := t.TempDir()
	configDir = tmpDir

	// Test 1: Load with no config file (should use defaults)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	if cfg.DefaultProvider == "" {
		t.Error("DefaultProvider should be set from defaults")
	}

	// Test 2: Load with custom config
	customCfg := Config{
		DefaultProvider: "test-provider",
		Providers: map[string]ProviderConfig{
			"test-provider": {
				ID:       "test-provider",
				Name:     "Test",
				Endpoint: "https://api.test.com",
				Format:   "openai",
				AuthType: "api_key",
			},
		},
		Settings: SettingsConfig{
			MaxToolTurns: 10,
		},
	}

	// Write custom config
	configPath := filepath.Join(tmpDir, "config.json")
	data, err := json.MarshalIndent(customCfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reload
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if cfg.DefaultProvider != "test-provider" {
		t.Errorf("Expected default_provider=test-provider, got %s", cfg.DefaultProvider)
	}

	if cfg.Settings.MaxToolTurns != 10 {
		t.Errorf("Expected MaxToolTurns=10, got %d", cfg.Settings.MaxToolTurns)
	}
}

func TestGetProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Test existing provider
	p, ok := GetProvider("claude")
	if !ok {
		t.Error("Expected to find claude provider")
	}
	if p.ID != "claude" {
		t.Errorf("Expected provider ID=claude, got %s", p.ID)
	}

	// Test non-existent provider
	_, ok = GetProvider("nonexistent")
	if ok {
		t.Error("Expected to not find nonexistent provider")
	}
}

func TestGetProviderModel(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Test getting default model
	model := GetProviderModel("claude", "default")
	if model == "" {
		t.Error("Expected claude to have a default model")
	}

	// Test non-existent provider
	model = GetProviderModel("nonexistent", "default")
	if model != "" {
		t.Errorf("Expected empty string for nonexistent provider, got %s", model)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	cfg := DefaultConfig()
	cfg.DefaultProvider = "test"

	configMu.Lock()
	current = cfg
	configMu.Unlock()

	err := Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist after Save()")
	}

	// Load and verify
	_, err = Load()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	loaded := Get()
	if loaded.DefaultProvider != "test" {
		t.Errorf("Expected reloaded default_provider=test, got %s", loaded.DefaultProvider)
	}
}
