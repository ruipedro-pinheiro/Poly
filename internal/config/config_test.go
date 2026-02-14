package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pedromelo/poly/internal/hooks"
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

func TestSaveLoadRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	cfg := DefaultConfig()
	cfg.DefaultProvider = "gpt"
	cfg.Settings.MaxToolTurns = 25
	cfg.Settings.MaxTableRounds = 7
	cfg.Settings.StreamingBuffer = 8192
	cfg.Settings.SaveSessions = false
	enabled := true
	cfg.Settings.Notifications = &enabled
	cfg.Theme.ProviderColors["custom"] = "#FF0000"

	configMu.Lock()
	current = cfg
	configMu.Unlock()

	if err := Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Reload from disk
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.DefaultProvider != "gpt" {
		t.Errorf("expected DefaultProvider=gpt, got %s", loaded.DefaultProvider)
	}
	if loaded.Settings.MaxToolTurns != 25 {
		t.Errorf("expected MaxToolTurns=25, got %d", loaded.Settings.MaxToolTurns)
	}
	if loaded.Settings.MaxTableRounds != 7 {
		t.Errorf("expected MaxTableRounds=7, got %d", loaded.Settings.MaxTableRounds)
	}
	if loaded.Settings.StreamingBuffer != 8192 {
		t.Errorf("expected StreamingBuffer=8192, got %d", loaded.Settings.StreamingBuffer)
	}
	if loaded.Theme.ProviderColors["custom"] != "#FF0000" {
		t.Errorf("expected custom color=#FF0000, got %s", loaded.Theme.ProviderColors["custom"])
	}
}

func TestSetMaxTableRounds_Persists(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	SetMaxTableRounds(10)

	cfg := Get()
	if cfg.Settings.MaxTableRounds != 10 {
		t.Errorf("expected MaxTableRounds=10, got %d", cfg.Settings.MaxTableRounds)
	}

	// Verify it persists on disk
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Settings.MaxTableRounds != 10 {
		t.Errorf("expected persisted MaxTableRounds=10, got %d", loaded.Settings.MaxTableRounds)
	}
}

func TestSetColorTheme(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	SetColorTheme("dracula")

	if GetColorTheme() != "dracula" {
		t.Errorf("expected color theme 'dracula', got %q", GetColorTheme())
	}
}

func TestSetNotifications(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Default is true
	if !NotificationsEnabled() {
		t.Error("expected notifications enabled by default")
	}

	SetNotifications(false)
	if NotificationsEnabled() {
		t.Error("expected notifications disabled after SetNotifications(false)")
	}

	SetNotifications(true)
	if !NotificationsEnabled() {
		t.Error("expected notifications enabled after SetNotifications(true)")
	}
}

func TestSetSandbox(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	if SandboxEnabled() {
		t.Error("sandbox should be disabled by default")
	}

	SetSandbox(true)
	if !SandboxEnabled() {
		t.Error("expected sandbox enabled after SetSandbox(true)")
	}
}

func TestGetSandboxImage(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Default
	if GetSandboxImage() != "alpine:latest" {
		t.Errorf("expected default sandbox image 'alpine:latest', got %q", GetSandboxImage())
	}
}

func TestGetProviderNames(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	names := GetProviderNames()
	if len(names) == 0 {
		t.Fatal("expected at least one provider name")
	}

	// Should be sorted
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)
	for i := range names {
		if names[i] != sorted[i] {
			t.Errorf("provider names not sorted: got %v", names)
			break
		}
	}
}

func TestGetProviderColor(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Existing provider
	color := GetProviderColor("claude")
	if color == "" || color == "#888888" {
		t.Errorf("expected claude to have a non-default color, got %q", color)
	}

	// Non-existent provider falls back
	color = GetProviderColor("nonexistent")
	if color != "#888888" {
		t.Errorf("expected fallback color #888888 for nonexistent, got %q", color)
	}
}

func TestSetProvider_AddNew(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	p := ProviderConfig{
		ID:       "custom-ai",
		Name:     "Custom AI",
		Format:   "openai",
		AuthType: "api_key",
		Endpoint: "https://api.custom.ai/v1",
	}
	SetProvider(p)

	got, ok := GetProvider("custom-ai")
	if !ok {
		t.Fatal("expected to find custom-ai provider")
	}
	if got.Name != "Custom AI" {
		t.Errorf("expected name 'Custom AI', got %q", got.Name)
	}
}

func TestDeleteProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Should have claude
	_, ok := GetProvider("claude")
	if !ok {
		t.Fatal("expected claude provider")
	}

	DeleteProvider("claude")
	_, ok = GetProvider("claude")
	if ok {
		t.Error("expected claude to be deleted")
	}
}

func TestGetProviderModel_FallbackDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	// Asking for a variant that doesn't exist falls back to "default"
	model := GetProviderModel("claude", "nonexistent-variant")
	defaultModel := GetProviderModel("claude", "default")
	if model != defaultModel {
		t.Errorf("expected fallback to default model %q, got %q", defaultModel, model)
	}
}

func TestLoadHistory(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	// Empty history
	history := LoadHistory()
	if history != nil {
		t.Errorf("expected nil for empty history, got %v", history)
	}

	// Add entries
	AddHistory("hello")
	AddHistory("world")

	history = LoadHistory()
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	if history[0] != "hello" || history[1] != "world" {
		t.Errorf("unexpected history: %v", history)
	}
}

func TestAddHistory_SkipsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	AddHistory("")
	AddHistory("   ")

	history := LoadHistory()
	if history != nil {
		t.Errorf("expected nil for empty-only history, got %v", history)
	}
}

func TestAddHistory_SkipsDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	AddHistory("same")
	AddHistory("same")
	AddHistory("same")

	history := LoadHistory()
	if len(history) != 1 {
		t.Errorf("expected 1 entry (consecutive dupes skipped), got %d", len(history))
	}
}

func TestAddHistory_NonConsecutiveDuplicatesAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	AddHistory("first")
	AddHistory("second")
	AddHistory("first") // non-consecutive, allowed

	history := LoadHistory()
	if len(history) != 3 {
		t.Errorf("expected 3 entries, got %d", len(history))
	}
}

func TestMergeConfig_MCPServers(t *testing.T) {
	base := DefaultConfig()
	user := &Config{
		MCPServers: map[string]MCPServerConfig{
			"test-mcp": {
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
	}

	result := mergeConfig(base, user)
	if result.MCPServers == nil {
		t.Fatal("expected MCPServers to be merged")
	}
	srv, ok := result.MCPServers["test-mcp"]
	if !ok {
		t.Fatal("expected test-mcp server")
	}
	if srv.Command != "node" {
		t.Errorf("expected command 'node', got %q", srv.Command)
	}
}

func TestMergeConfig_Hooks(t *testing.T) {
	base := DefaultConfig()
	user := &Config{
		Hooks: hooks.HooksConfig{
			PreTool: []hooks.Hook{
				{Tool: "bash", Command: "echo pre"},
			},
		},
	}

	result := mergeConfig(base, user)
	if len(result.Hooks.PreTool) != 1 {
		t.Errorf("expected 1 pre_tool hook, got %d", len(result.Hooks.PreTool))
	}
}

func TestConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	cfg := DefaultConfig()
	configMu.Lock()
	current = cfg
	configMu.Unlock()

	Save()

	info, err := os.Stat(filepath.Join(tmpDir, "config.json"))
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected config file permissions 0600, got %o", perm)
	}
}

func TestSave_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	configMu.Lock()
	current = nil
	configMu.Unlock()

	err := Save()
	if err != nil {
		t.Errorf("Save() with nil config should not error: %v", err)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	// Write invalid JSON
	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("not json"), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with invalid JSON should not error (uses defaults): %v", err)
	}
	// Should fall back to defaults
	if cfg.DefaultProvider != "claude" {
		t.Errorf("expected default provider 'claude' on invalid JSON, got %q", cfg.DefaultProvider)
	}
}

func TestGetMCPServers_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir
	Load()

	servers := GetMCPServers()
	if servers != nil {
		t.Errorf("expected nil MCP servers by default, got %v", servers)
	}
}

