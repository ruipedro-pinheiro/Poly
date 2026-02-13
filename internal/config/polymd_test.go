package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolyMD(t *testing.T) {
	t.Run("no POLY.md files returns empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tmpDir)

		result := LoadPolyMD()
		if result != "" {
			t.Errorf("LoadPolyMD() = %q, want empty string", result)
		}
	})

	t.Run("POLY.md in current dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tmpDir)

		os.WriteFile(filepath.Join(tmpDir, "POLY.md"), []byte("project instructions"), 0644)

		result := LoadPolyMD()
		if result != "project instructions" {
			t.Errorf("LoadPolyMD() = %q, want %q", result, "project instructions")
		}
	})

	t.Run("poly.md lowercase variant", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tmpDir)

		os.WriteFile(filepath.Join(tmpDir, "poly.md"), []byte("lowercase variant"), 0644)

		result := LoadPolyMD()
		if result != "lowercase variant" {
			t.Errorf("LoadPolyMD() = %q, want %q", result, "lowercase variant")
		}
	})

	t.Run(".poly/POLY.md variant", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tmpDir)

		os.MkdirAll(filepath.Join(tmpDir, ".poly"), 0755)
		os.WriteFile(filepath.Join(tmpDir, ".poly", "POLY.md"), []byte("dotpoly variant"), 0644)

		result := LoadPolyMD()
		if result != "dotpoly variant" {
			t.Errorf("LoadPolyMD() = %q, want %q", result, "dotpoly variant")
		}
	})

	t.Run("parent and child POLY.md are concatenated", func(t *testing.T) {
		tmpDir := t.TempDir()
		childDir := filepath.Join(tmpDir, "child")
		os.MkdirAll(childDir, 0755)

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(childDir)

		os.WriteFile(filepath.Join(tmpDir, "POLY.md"), []byte("parent"), 0644)
		os.WriteFile(filepath.Join(childDir, "POLY.md"), []byte("child"), 0644)

		result := LoadPolyMD()
		// Root first, cwd last, separated by ---
		expected := "parent\n\n---\n\nchild"
		if result != expected {
			t.Errorf("LoadPolyMD() = %q, want %q", result, expected)
		}
	})

	t.Run("empty POLY.md is skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tmpDir)

		os.WriteFile(filepath.Join(tmpDir, "POLY.md"), []byte(""), 0644)

		result := LoadPolyMD()
		if result != "" {
			t.Errorf("LoadPolyMD() = %q, want empty string for empty file", result)
		}
	})
}
