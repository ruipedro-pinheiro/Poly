package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// allowedPrefixes lists extra directories outside cwd that are always accessible.
var (
	allowedPrefixes []string
	allowedMu       sync.RWMutex
)

func init() {
	home, err := os.UserHomeDir()
	if err == nil {
		// Only ~/.poly/ is allowed by default.
		// Removed /tmp/ from default for security, but tests can add it.
		AddAllowedPrefix(filepath.Join(home, ".poly") + string(filepath.Separator))
	}
}

// AddAllowedPrefix adds a path to the allowed prefixes (thread-safe).
// Used by tests to allow access to temporary test directories.
func AddAllowedPrefix(path string) {
	allowedMu.Lock()
	defer allowedMu.Unlock()
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path += string(filepath.Separator)
	}
	allowedPrefixes = append(allowedPrefixes, path)
}

// ValidatePath checks that a resolved path is inside the workspace (cwd)
// or an allowed prefix. Relative paths are resolved against cwd.
// All symlinks are resolved to prevent traversal attacks.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	cwd = filepath.Clean(cwd)

	// 1. Resolve to absolute path without evaluating symlinks yet
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(cwd, path))
	}

	// 2. Evaluate all symlinks to get the REAL target path
	// This is the core defense against traversal via links.
	resolved, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = resolved
	} else if os.IsNotExist(err) {
		// If file doesn't exist, validate the parent directory instead
		parent := filepath.Dir(absPath)
		resolvedParent, pErr := filepath.EvalSymlinks(parent)
		if pErr == nil {
			absPath = filepath.Join(resolvedParent, filepath.Base(absPath))
		}
	} else {
		return "", fmt.Errorf("invalid path (symlink resolution failed): %w", err)
	}

	// Final absolute path to check
	absPath = filepath.Clean(absPath)

	// 3. Check: is it inside cwd?
	// Note: using case-insensitive check for OS compatibility (Windows/macOS)
	lowerAbs := strings.ToLower(absPath)
	lowerCwd := strings.ToLower(cwd)
	cwdPrefix := lowerCwd + string(filepath.Separator)

	if lowerAbs == lowerCwd || strings.HasPrefix(lowerAbs, cwdPrefix) {
		return absPath, nil
	}

	// 4. Check: is it in an allowed prefix?
	allowedMu.RLock()
	defer allowedMu.RUnlock()
	for _, prefix := range allowedPrefixes {
		lowerPrefix := strings.ToLower(prefix)
		if strings.HasPrefix(lowerAbs, lowerPrefix) || lowerAbs == strings.TrimSuffix(lowerPrefix, string(filepath.Separator)) {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("Access denied: path '%s' is outside workspace (resolved to '%s')", path, absPath)
}
