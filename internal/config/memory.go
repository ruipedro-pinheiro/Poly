package config

import (
	"os"
	"path/filepath"
)

// LoadMemoryMD loads ~/.poly/MEMORY.md if it exists
func LoadMemoryMD() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".poly", "MEMORY.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
