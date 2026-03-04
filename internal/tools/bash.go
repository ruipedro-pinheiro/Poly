package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/sandbox"
	"github.com/pedromelo/poly/internal/security"
)

// NOTE: Safe command classification is handled by permission.ClassifyBashCommand().
// Blocked patterns are in internal/security and shared with shell/executor.go.

const (
	maxOutput      = 30000
	defaultTimeout = 60 * time.Second
	maxTimeout     = 10 * time.Minute
)

// BashTool executes shell commands
type BashTool struct{}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return "Execute a shell command. Supports configurable timeout (default 60s, max 10min). Returns stdout, stderr, and exit code."
}

func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Timeout in milliseconds (default: 60000, max: 600000)",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(args map[string]interface{}) ToolResult {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return ToolResult{Content: "Error: command is required", IsError: true}
	}

	// Check for blocked patterns
	if security.IsBlocked(command) {
		return ToolResult{Content: "Blocked: dangerous command pattern detected", IsError: true}
	}

	// Determine timeout
	timeout := defaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Millisecond
		if timeout > maxTimeout {
			timeout = maxTimeout
		}
		if timeout < time.Second {
			timeout = time.Second
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cwd, _ := os.Getwd()
	startTime := time.Now()

	// Route to sandbox if enabled
	if sandbox.Enabled {
		output, exitCode, err := sandbox.Run(ctx, command, cwd)
		elapsed := time.Since(startTime)

		if ctx.Err() == context.DeadlineExceeded {
			return ToolResult{
				Content: fmt.Sprintf("[sandbox] Command timed out after %s\n%s", timeout, truncateOutput(output)),
				IsError: true,
			}
		}

		if err != nil {
			return ToolResult{
				Content: fmt.Sprintf("[sandbox] Error: %v", err),
				IsError: true,
			}
		}

		output = strings.TrimSpace(output)
		if output == "" {
			output = "(no output)"
		}
		output = truncateOutput(output)

		timing := ""
		if elapsed > time.Second {
			timing = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
		}

		if exitCode != 0 {
			return ToolResult{
				Content: fmt.Sprintf("[sandbox] Exit code %d%s\n%s", exitCode, timing, output),
				IsError: true,
			}
		}

		return ToolResult{Content: "[sandbox] " + output + timing}
	}

	// Execute command locally
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = cwd
	cmd.Env = security.SafeEnv(os.Environ())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	elapsed := time.Since(startTime)

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		partial := truncateOutput(stdout.String())
		return ToolResult{
			Content: fmt.Sprintf("Command timed out after %s\n%s", timeout, partial),
			IsError: true,
		}
	}

	// Build output
	var parts []string
	if stdout.Len() > 0 {
		parts = append(parts, truncateOutput(strings.TrimSpace(stdout.String())))
	}
	if stderr.Len() > 0 {
		parts = append(parts, "STDERR:\n"+truncateOutput(strings.TrimSpace(stderr.String())))
	}

	output := strings.Join(parts, "\n\n")
	if output == "" {
		output = "(no output)"
	}

	timing := ""
	if elapsed > time.Second {
		timing = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return ToolResult{
				Content: fmt.Sprintf("Exit code %d%s\n%s", exitErr.ExitCode(), timing, output),
				IsError: true,
			}
		}
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	return ToolResult{Content: output + timing}
}

// truncateOutput keeps start + end if output is too long
func truncateOutput(output string) string {
	if len(output) <= maxOutput {
		return output
	}

	half := maxOutput / 2
	lines := strings.Split(output, "\n")

	var headSize, headEnd int
	for i := 0; i < len(lines); i++ {
		headSize += len(lines[i]) + 1
		if headSize > half {
			headEnd = i
			break
		}
	}

	var tailSize int
	tailStart := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		tailSize += len(lines[i]) + 1
		if tailSize > half {
			tailStart = i + 1
			break
		}
	}

	skipped := tailStart - headEnd - 1
	return strings.Join(lines[:headEnd+1], "\n") +
		fmt.Sprintf("\n\n... (%d lines truncated) ...\n\n", skipped) +
		strings.Join(lines[tailStart:], "\n")
}
