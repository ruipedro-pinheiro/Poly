package config

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectInfo holds detected project metadata
type ProjectInfo struct {
	Type    string   // "go", "node", "rust", "python", "unknown"
	Name    string   // project name from config file
	Details []string // key details (version, module, etc.)
}

// DetectProject scans cwd for project markers and returns info
func DetectProject() *ProjectInfo {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	// Try each detector in order of priority
	if info := detectGo(cwd); info != nil {
		addGitInfo(cwd, info)
		return info
	}
	if info := detectNode(cwd); info != nil {
		addGitInfo(cwd, info)
		return info
	}
	if info := detectRust(cwd); info != nil {
		addGitInfo(cwd, info)
		return info
	}
	if info := detectPython(cwd); info != nil {
		addGitInfo(cwd, info)
		return info
	}

	// Check for Makefile/Dockerfile even without a language
	info := &ProjectInfo{Type: "unknown", Name: filepath.Base(cwd)}
	if targets := detectMakeTargets(cwd); len(targets) > 0 {
		info.Details = append(info.Details, "Make targets: "+strings.Join(targets, ", "))
	}
	if base := detectDockerBase(cwd); base != "" {
		info.Details = append(info.Details, "Docker: "+base)
	}
	addGitInfo(cwd, info)

	if len(info.Details) == 0 {
		return nil // nothing detected at all
	}
	return info
}

// FormatProjectInfo returns a one-line summary for the system prompt
func FormatProjectInfo(info *ProjectInfo) string {
	if info == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("Project: %s (%s)", info.Name, info.Type)}
	parts = append(parts, info.Details...)
	return strings.Join(parts, " | ")
}

// detectGo checks for go.mod
func detectGo(dir string) *ProjectInfo {
	path := filepath.Join(dir, "go.mod")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	info := &ProjectInfo{Type: "go"}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimPrefix(line, "module ")
			info.Name = mod
			// Use last path segment as short name
			if idx := strings.LastIndex(mod, "/"); idx >= 0 {
				info.Details = append(info.Details, "Module: "+mod)
				info.Name = mod[idx+1:]
			}
		}
		if strings.HasPrefix(line, "go ") {
			info.Details = append(info.Details, "Go "+strings.TrimPrefix(line, "go "))
		}
	}
	if info.Name == "" {
		info.Name = filepath.Base(dir)
	}

	if targets := detectMakeTargets(dir); len(targets) > 0 {
		info.Details = append(info.Details, "Make targets: "+strings.Join(targets, ", "))
	}
	return info
}

// detectNode checks for package.json
func detectNode(dir string) *ProjectInfo {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)
	info := &ProjectInfo{Type: "node", Name: filepath.Base(dir)}

	// Simple extraction without full JSON parsing
	if name := extractJSONField(content, "name"); name != "" {
		info.Name = name
	}
	if ver := extractJSONField(content, "version"); ver != "" {
		info.Details = append(info.Details, "v"+ver)
	}

	// Count deps
	deps := countJSONKeys(content, "dependencies")
	devDeps := countJSONKeys(content, "devDependencies")
	if deps > 0 || devDeps > 0 {
		info.Details = append(info.Details, fmt.Sprintf("%d deps, %d devDeps", deps, devDeps))
	}

	return info
}

// detectRust checks for Cargo.toml
func detectRust(dir string) *ProjectInfo {
	path := filepath.Join(dir, "Cargo.toml")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	info := &ProjectInfo{Type: "rust", Name: filepath.Base(dir)}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "name") && strings.Contains(line, "=") {
			val := extractTOMLValue(line)
			if val != "" {
				info.Name = val
			}
		}
		if strings.HasPrefix(line, "edition") && strings.Contains(line, "=") {
			val := extractTOMLValue(line)
			if val != "" {
				info.Details = append(info.Details, "Edition "+val)
			}
		}
	}
	return info
}

// detectPython checks for pyproject.toml or requirements.txt
func detectPython(dir string) *ProjectInfo {
	// Try pyproject.toml first
	path := filepath.Join(dir, "pyproject.toml")
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		info := &ProjectInfo{Type: "python", Name: filepath.Base(dir)}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "name") && strings.Contains(line, "=") {
				val := extractTOMLValue(line)
				if val != "" {
					info.Name = val
				}
			}
		}
		return info
	}

	// Fall back to requirements.txt
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		return &ProjectInfo{
			Type: "python",
			Name: filepath.Base(dir),
		}
	}

	return nil
}

// addGitInfo adds git branch and remote info
func addGitInfo(dir string, info *ProjectInfo) {
	// Check if .git exists
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return
	}

	// Get current branch
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	if out, err := cmd.Output(); err == nil {
		branch := strings.TrimSpace(string(out))
		if branch != "" {
			info.Details = append(info.Details, "Branch: "+branch)
		}
	}

	// Get remote URL
	cmd = exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	if out, err := cmd.Output(); err == nil {
		remote := strings.TrimSpace(string(out))
		if remote != "" {
			// Shorten GitHub URLs
			remote = strings.TrimSuffix(remote, ".git")
			if strings.Contains(remote, "github.com") {
				parts := strings.Split(remote, "github.com")
				if len(parts) > 1 {
					remote = "github.com" + parts[len(parts)-1]
				}
			}
			info.Details = append(info.Details, "Remote: "+remote)
		}
	}
}

// detectMakeTargets returns the first N target names from a Makefile
func detectMakeTargets(dir string) []string {
	path := filepath.Join(dir, "Makefile")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var targets []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(targets) < 10 {
		line := scanner.Text()
		// Match lines like "target:" at the start (not indented, not comments)
		if len(line) > 0 && line[0] != '\t' && line[0] != '#' && line[0] != '.' {
			if idx := strings.Index(line, ":"); idx > 0 {
				target := strings.TrimSpace(line[:idx])
				// Skip variable assignments and includes
				if !strings.ContainsAny(target, " =+?$") {
					targets = append(targets, target)
				}
			}
		}
	}
	return targets
}

// detectDockerBase extracts the base image from a Dockerfile
func detectDockerBase(dir string) string {
	path := filepath.Join(dir, "Dockerfile")
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// extractJSONField does a naive extraction of a top-level string field from JSON
func extractJSONField(content, field string) string {
	key := fmt.Sprintf(`"%s"`, field)
	idx := strings.Index(content, key)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(key):]
	// Find the colon then the value
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return ""
	}
	rest = rest[colonIdx+1:]
	// Find the opening quote
	openIdx := strings.Index(rest, `"`)
	if openIdx < 0 {
		return ""
	}
	rest = rest[openIdx+1:]
	closeIdx := strings.Index(rest, `"`)
	if closeIdx < 0 {
		return ""
	}
	return rest[:closeIdx]
}

// countJSONKeys counts the number of keys in a JSON object field (naive)
func countJSONKeys(content, field string) int {
	key := fmt.Sprintf(`"%s"`, field)
	idx := strings.Index(content, key)
	if idx < 0 {
		return 0
	}
	rest := content[idx+len(key):]
	openBrace := strings.Index(rest, "{")
	if openBrace < 0 {
		return 0
	}
	closeBrace := strings.Index(rest, "}")
	if closeBrace < 0 || closeBrace <= openBrace {
		return 0
	}
	block := rest[openBrace+1 : closeBrace]
	// Count occurrences of ": (key-value separator)
	return strings.Count(block, `":`)
}

// extractTOMLValue extracts the value from a simple TOML key = "value" line
func extractTOMLValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) < 2 {
		return ""
	}
	val := strings.TrimSpace(parts[1])
	val = strings.Trim(val, `"'`)
	return val
}
