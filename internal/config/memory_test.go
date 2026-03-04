package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryPath(t *testing.T) {
	path := MemoryPath()
	if path == "" {
		t.Fatal("MemoryPath() returned empty string")
	}
	if !strings.HasSuffix(path, filepath.Join(".poly", "MEMORY.md")) {
		t.Errorf("MemoryPath() = %q, expected to end with .poly/MEMORY.md", path)
	}
}

func TestLoadMemoryMD(t *testing.T) {
	t.Run("returns empty when file does not exist", func(t *testing.T) {
		// Temporarily override HOME to isolate test
		origHome := os.Getenv("HOME")
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)

		result := LoadMemoryMD()
		if result != "" {
			t.Errorf("LoadMemoryMD() = %q, want empty when file missing", result)
		}
	})

	t.Run("reads existing memory file", func(t *testing.T) {
		origHome := os.Getenv("HOME")
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)

		polyDir := filepath.Join(tmpDir, ".poly")
		os.MkdirAll(polyDir, 0755)
		os.WriteFile(filepath.Join(polyDir, "MEMORY.md"), []byte("remembered stuff"), 0600)

		result := LoadMemoryMD()
		if result != "remembered stuff" {
			t.Errorf("LoadMemoryMD() = %q, want %q", result, "remembered stuff")
		}
	})
}

func TestClearMemoryMD(t *testing.T) {
	t.Run("removes existing memory file", func(t *testing.T) {
		origHome := os.Getenv("HOME")
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)

		polyDir := filepath.Join(tmpDir, ".poly")
		os.MkdirAll(polyDir, 0755)
		memPath := filepath.Join(polyDir, "MEMORY.md")
		os.WriteFile(memPath, []byte("to be cleared"), 0600)

		err := ClearMemoryMD()
		if err != nil {
			t.Fatalf("ClearMemoryMD() error = %v", err)
		}

		if _, err := os.Stat(memPath); !os.IsNotExist(err) {
			t.Error("MEMORY.md should be removed after ClearMemoryMD()")
		}
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		origHome := os.Getenv("HOME")
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)

		err := ClearMemoryMD()
		if err == nil {
			t.Error("ClearMemoryMD() should return error when file does not exist")
		}
	})
}
