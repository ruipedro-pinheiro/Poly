package config

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadPolyMD walks from cwd up to root, collecting POLY.md / poly.md / .poly/POLY.md
// files. Returns them concatenated (root first, cwd last) separated by "---".
func LoadPolyMD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	var contents []string
	dir := cwd
	for {
		for _, name := range []string{"POLY.md", "poly.md", ".poly/POLY.md"} {
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if err == nil && len(data) > 0 {
				contents = append([]string{string(data)}, contents...)
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if len(contents) == 0 {
		return ""
	}
	return strings.Join(contents, "\n\n---\n\n")
}
