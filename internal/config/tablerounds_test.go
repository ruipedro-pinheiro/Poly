package config

import (
	"testing"
)

func TestSetMaxTableRounds(t *testing.T) {
	tmpDir := t.TempDir()
	configDir = tmpDir

	// Load defaults first
	Load()

	// Default value should be 0 (meaning "use default 5" in llm.GetMaxTableRounds)
	cfg := Get()
	if cfg.Settings.MaxTableRounds != 0 {
		t.Errorf("expected default MaxTableRounds=0, got %d", cfg.Settings.MaxTableRounds)
	}

	// Set custom value
	SetMaxTableRounds(3)
	cfg = Get()
	if cfg.Settings.MaxTableRounds != 3 {
		t.Errorf("expected MaxTableRounds=3 after SetMaxTableRounds(3), got %d", cfg.Settings.MaxTableRounds)
	}

	// Set back to 0
	SetMaxTableRounds(0)
	cfg = Get()
	if cfg.Settings.MaxTableRounds != 0 {
		t.Errorf("expected MaxTableRounds=0 after SetMaxTableRounds(0), got %d", cfg.Settings.MaxTableRounds)
	}
}

func TestMergeConfig_MaxTableRounds(t *testing.T) {
	base := DefaultConfig()
	user := &Config{
		Providers: map[string]ProviderConfig{},
		Theme:     ThemeConfig{ProviderColors: map[string]string{}},
		Settings: SettingsConfig{
			MaxTableRounds: 7,
		},
	}

	merged := mergeConfig(base, user)
	if merged.Settings.MaxTableRounds != 7 {
		t.Errorf("expected merged MaxTableRounds=7, got %d", merged.Settings.MaxTableRounds)
	}
}

func TestMergeConfig_MaxTableRoundsZeroNoOverride(t *testing.T) {
	base := DefaultConfig()
	base.Settings.MaxTableRounds = 10

	user := &Config{
		Providers: map[string]ProviderConfig{},
		Theme:     ThemeConfig{ProviderColors: map[string]string{}},
		Settings: SettingsConfig{
			MaxTableRounds: 0, // zero should NOT override base
		},
	}

	merged := mergeConfig(base, user)
	if merged.Settings.MaxTableRounds != 10 {
		t.Errorf("expected merged MaxTableRounds=10 (zero should not override), got %d", merged.Settings.MaxTableRounds)
	}
}
