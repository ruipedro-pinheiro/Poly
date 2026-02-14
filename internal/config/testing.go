package config

// SetConfigDirForTest allows external test packages to override the config directory.
func SetConfigDirForTest(dir string) {
	configDir = dir
}

// SetForTest allows external test packages to set the global config directly.
func SetForTest(cfg *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	current = cfg
}
