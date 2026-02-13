package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitStatusTool shows git repository status
type GitStatusTool struct{}

func (t *GitStatusTool) Name() string {
	return "git_status"
}

func (t *GitStatusTool) Description() string {
	return "Get git repository status (staged, unstaged, untracked files). Returns porcelain v2 format with branch info."
}

func (t *GitStatusTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Repository path. Defaults to current working directory.",
			},
		},
	}
}

func (t *GitStatusTool) Execute(args map[string]interface{}) ToolResult {
	dir := gitWorkDir(args)

	cmd := exec.Command("git", "status", "--porcelain=v2", "--branch")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %s\n%s", err, string(output)), IsError: true}
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return ToolResult{Content: "Nothing to report (clean working tree)"}
	}
	return ToolResult{Content: result}
}

// GitDiffTool shows git diff output
type GitDiffTool struct{}

func (t *GitDiffTool) Name() string {
	return "git_diff"
}

func (t *GitDiffTool) Description() string {
	return "Show git diff (staged or unstaged changes). Can target a specific file."
}

func (t *GitDiffTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"staged": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, show staged changes (git diff --staged). Default: false",
			},
			"file": map[string]interface{}{
				"type":        "string",
				"description": "Show diff for a specific file only.",
			},
		},
	}
}

func (t *GitDiffTool) Execute(args map[string]interface{}) ToolResult {
	dir := gitWorkDir(args)

	gitArgs := []string{"diff"}
	if staged, ok := args["staged"].(bool); ok && staged {
		gitArgs = append(gitArgs, "--staged")
	}
	if file, ok := args["file"].(string); ok && file != "" {
		gitArgs = append(gitArgs, "--", file)
	}

	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %s\n%s", err, string(output)), IsError: true}
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return ToolResult{Content: "No differences found."}
	}
	return ToolResult{Content: result}
}

// GitLogTool shows git commit log
type GitLogTool struct{}

func (t *GitLogTool) Name() string {
	return "git_log"
}

func (t *GitLogTool) Description() string {
	return "Show git commit log. Supports oneline format and limiting the number of commits."
}

func (t *GitLogTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"type":        "number",
				"description": "Number of commits to show. Default: 10",
			},
			"oneline": map[string]interface{}{
				"type":        "boolean",
				"description": "Use compact one-line format. Default: true",
			},
		},
	}
}

func (t *GitLogTool) Execute(args map[string]interface{}) ToolResult {
	dir := gitWorkDir(args)

	count := 10
	if c, ok := args["count"].(float64); ok && c > 0 {
		count = int(c)
		if count > 100 {
			count = 100
		}
	}

	oneline := true
	if o, ok := args["oneline"].(bool); ok {
		oneline = o
	}

	gitArgs := []string{"log", fmt.Sprintf("-%d", count)}
	if oneline {
		gitArgs = append(gitArgs, "--oneline")
	}

	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %s\n%s", err, string(output)), IsError: true}
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return ToolResult{Content: "No commits found."}
	}
	return ToolResult{Content: result}
}

// gitWorkDir returns the working directory from args or falls back to cwd
func gitWorkDir(args map[string]interface{}) string {
	if path, ok := args["path"].(string); ok && path != "" {
		return path
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
