package hooks

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const hookTimeout = 10 * time.Second

// Hook defines a single hook entry
type Hook struct {
	Tool    string `json:"tool"`    // tool name to match, "*" = all tools
	Command string `json:"command"` // shell command template
}

// HooksConfig holds all configured hooks
type HooksConfig struct {
	PreTool   []Hook `json:"pre_tool"`
	PostTool  []Hook `json:"post_tool"`
	OnMessage []Hook `json:"on_message"`
}

// TemplateData is passed to hook command templates
type TemplateData struct {
	ToolName string
	Command  string // for bash tool: the command being run
	ExitCode int    // for post_tool: 0 = success, 1 = error
	Output   string // for post_tool: truncated result
}

var (
	current *HooksConfig
	mu      sync.RWMutex
)

// SetConfig sets the hooks configuration (called from config.Load)
func SetConfig(cfg *HooksConfig) {
	mu.Lock()
	defer mu.Unlock()
	if cfg == nil {
		current = &HooksConfig{}
	} else {
		current = cfg
	}
}

// getConfig returns the current hooks config
func getConfig() *HooksConfig {
	mu.RLock()
	defer mu.RUnlock()
	if current == nil {
		return &HooksConfig{}
	}
	return current
}

// RunPreToolHooks runs all matching pre_tool hooks in goroutines
func RunPreToolHooks(toolName string, args map[string]interface{}) {
	cfg := getConfig()
	if len(cfg.PreTool) == 0 {
		return
	}

	data := TemplateData{
		ToolName: toolName,
	}
	// Extract command for bash tool
	if toolName == "bash" {
		if cmd, ok := args["command"].(string); ok {
			data.Command = cmd
		}
	}

	runMatchingHooks(cfg.PreTool, toolName, data)
}

// RunPostToolHooks runs all matching post_tool hooks in goroutines
func RunPostToolHooks(toolName string, isError bool, output string) {
	cfg := getConfig()
	if len(cfg.PostTool) == 0 {
		return
	}

	exitCode := 0
	if isError {
		exitCode = 1
	}

	// Truncate output to prevent huge template expansions
	if len(output) > 1000 {
		output = output[:1000]
	}

	data := TemplateData{
		ToolName: toolName,
		ExitCode: exitCode,
		Output:   output,
	}

	runMatchingHooks(cfg.PostTool, toolName, data)
}

// RunOnMessageHooks runs all on_message hooks in goroutines
func RunOnMessageHooks(role string, content string) {
	cfg := getConfig()
	if len(cfg.OnMessage) == 0 {
		return
	}

	// Truncate content
	if len(content) > 500 {
		content = content[:500]
	}

	data := TemplateData{
		ToolName: role,
		Output:   content,
	}

	for _, hook := range cfg.OnMessage {
		go executeHook(hook.Command, data)
	}
}

// runMatchingHooks finds hooks matching the tool and runs them
func runMatchingHooks(hooks []Hook, toolName string, data TemplateData) {
	for _, hook := range hooks {
		if hook.Tool == "*" || hook.Tool == toolName {
			go executeHook(hook.Command, data)
		}
	}
}

// executeHook renders the command template and runs it with a timeout
func executeHook(cmdTemplate string, data TemplateData) {
	rendered, err := renderTemplate(cmdTemplate, data)
	if err != nil {
		// Silently skip broken templates
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), hookTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", rendered)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run()
	// Hooks are best-effort, don't propagate errors
}

// renderTemplate replaces placeholders in the command string with data values.
// Uses simple string replacement instead of text/template to prevent injection.
func renderTemplate(tmpl string, data TemplateData) (string, error) {
	// Quick path: no placeholders
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	result := tmpl
	result = strings.ReplaceAll(result, "{{.ToolName}}", data.ToolName)
	result = strings.ReplaceAll(result, "{{.Command}}", data.Command)
	result = strings.ReplaceAll(result, "{{.ExitCode}}", strconv.Itoa(data.ExitCode))
	result = strings.ReplaceAll(result, "{{.Output}}", data.Output)
	// Also support spaces around dot: {{ .ToolName }}
	result = strings.ReplaceAll(result, "{{ .ToolName }}", data.ToolName)
	result = strings.ReplaceAll(result, "{{ .Command }}", data.Command)
	result = strings.ReplaceAll(result, "{{ .ExitCode }}", strconv.Itoa(data.ExitCode))
	result = strings.ReplaceAll(result, "{{ .Output }}", data.Output)

	return result, nil
}
