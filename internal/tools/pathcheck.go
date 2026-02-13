package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// allowedPrefixes lists extra directories outside cwd that are always accessible.
var allowedPrefixes []string

func init() {
	home, err := os.UserHomeDir()
	if err == nil {
		allowedPrefixes = append(allowedPrefixes, filepath.Join(home, ".poly")+string(filepath.Separator))
	}
	allowedPrefixes = append(allowedPrefixes, "/tmp"+string(filepath.Separator))
}

// ValidatePath checks that a resolved path is inside the workspace (cwd),
// ~/.poly/, or /tmp/. Relative paths are resolved against cwd.
// Symlinks are resolved when the target exists.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Resolve to absolute
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(cwd, path))
	}

	// If the target exists, resolve symlinks to get the real path
	resolved, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = resolved
	}
	// If file doesn't exist yet (e.g. write_file creating a new file),
	// we resolve the parent directory's symlinks instead.
	if os.IsNotExist(err) {
		parentResolved, pErr := filepath.EvalSymlinks(filepath.Dir(absPath))
		if pErr == nil {
			absPath = filepath.Join(parentResolved, filepath.Base(absPath))
		}
	}

	// Check: is it inside cwd?
	cwdPrefix := cwd + string(filepath.Separator)
	if absPath == cwd || strings.HasPrefix(absPath, cwdPrefix) {
		return absPath, nil
	}

	// Check: is it in an allowed prefix (~/.poly/, /tmp/)?
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(absPath, prefix) || absPath == strings.TrimSuffix(prefix, string(filepath.Separator)) {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("Access denied: path '%s' is outside workspace", path)
}
