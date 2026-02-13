package config

import (
	"os"
	"path/filepath"
)

// MemoryPath returns the path to ~/.poly/MEMORY.md
func MemoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".poly", "MEMORY.md")
}

// LoadMemoryMD loads ~/.poly/MEMORY.md if it exists
func LoadMemoryMD() string {
	path := MemoryPath()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// ClearMemoryMD deletes ~/.poly/MEMORY.md
func ClearMemoryMD() error {
	path := MemoryPath()
	if path == "" {
		return os.ErrNotExist
	}
	return os.Remove(path)
}
