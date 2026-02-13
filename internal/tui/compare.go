package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/llm"
)

// handleCompareCommand parses /compare args and starts the comparison.
// Usage:
//
//	/compare explain recursion             -> all configured providers
//	/compare claude,gemini explain recursion -> specific providers only
func (m *Model) handleCompareCommand(args []string) {
	if len(args) == 0 {
		m.status = "Usage: /compare [provider1,provider2] <prompt>"
		return
	}

	if m.isStreaming {
		m.status = "Cannot compare while streaming"
		return
	}

	// Parse: first arg might be a comma-separated provider list
	var targetProviders []string
	var prompt string

	firstArg := args[0]
	// Check if first arg is a provider list (comma-separated known providers)
	parts := strings.Split(firstArg, ",")
	allMatch := len(parts) > 0
	for _, p := range parts {
		if _, ok := m.providers[strings.TrimSpace(p)]; !ok {
			allMatch = false
			break
		}
	}
	if allMatch && len(args) > 1 {
		for _, p := range parts {
			name := strings.TrimSpace(p)
			if prov, ok := m.providers[name]; ok && prov.IsConfigured() {
				targetProviders = append(targetProviders, name)
			}
		}
		prompt = strings.Join(args[1:], " ")
	} else {
		prompt = strings.Join(args, " ")
	}

	// Strip surrounding quotes from prompt
	prompt = strings.Trim(prompt, "\"'")

	if prompt == "" {
		m.status = "Usage: /compare [provider1,provider2] <prompt>"
		return
	}

	// If no specific providers, use all configured
	if len(targetProviders) == 0 {
		for name, p := range m.providers {
			if p.IsConfigured() {
				targetProviders = append(targetProviders, name)
			}
		}
	}

	// Sort providers alphabetically for consistent ordering
	sort.Strings(targetProviders)

	if len(targetProviders) == 0 {
		m.status = "No configured providers. Use Control Room (Ctrl+D)."
		return
	}

	if len(targetProviders) < 2 {
		m.status = "Need 2+ configured providers to compare."
		return
	}

	// Add a system message showing the compare is starting
	m.messages = append(m.messages, Message{
		Role:    "system",
		Content: fmt.Sprintf("Comparing %d providers: %s\nPrompt: %s", len(targetProviders), strings.Join(targetProviders, ", "), prompt),
	})
	m.updateViewport()

	m.isStreaming = true
	m.streamStartTime = time.Now()
	m.status = fmt.Sprintf("Comparing %d providers...", len(targetProviders))

	// Build tea.Cmds that will run in parallel, each querying a provider
	cmds := make([]tea.Cmd, 0, len(targetProviders))
	for i, name := range targetProviders {
		provName := name
		idx := i
		total := len(targetProviders)
		p := m.providers[provName]

		// Set default model variant before launching
		if models, ok := llm.GetModelVariants()[provName]; ok {
			if model, ok := models["default"]; ok {
				p.SetModel(model)
			}
		}

		cmds = append(cmds, func() tea.Msg {
			messages := []llm.Message{{
				Role:    "user",
				Content: prompt,
			}}

			start := time.Now()
			events := p.Stream(context.Background(), messages, nil)

			var content strings.Builder
			var model string
			var streamErr error

			for event := range events {
				switch event.Type {
				case "content":
					content.WriteString(event.Content)
				case "done":
					if event.Response != nil {
						model = event.Response.Model
					}
				case "error":
					if event.Error != nil {
						streamErr = event.Error
					}
				}
			}
			elapsed := time.Since(start)

			return CompareResultMsg{
				Provider:  provName,
				Model:     model,
				Content:   content.String(),
				Error:     streamErr,
				ElapsedMs: elapsed.Milliseconds(),
				Index:     idx,
				Total:     total,
			}
		})
	}

	m.compareExpected = len(targetProviders)
	m.compareReceived = 0
	m.comparePending = cmds
}

// handleCompareResult processes a single provider's compare response
func (m *Model) handleCompareResult(msg CompareResultMsg) tea.Cmd {
	m.compareReceived++

	var content string
	if msg.Error != nil {
		content = fmt.Sprintf("=== %s (%dms) ===\nError: %s",
			msg.Provider, msg.ElapsedMs, msg.Error.Error())
	} else {
		modelInfo := msg.Provider
		if msg.Model != "" {
			modelInfo = fmt.Sprintf("%s (%s)", msg.Provider, msg.Model)
		}
		content = fmt.Sprintf("=== %s | %dms ===\n%s",
			modelInfo, msg.ElapsedMs, msg.Content)
	}

	m.messages = append(m.messages, Message{
		Role:     "assistant",
		Content:  content,
		Provider: msg.Provider,
	})
	m.saveMessageAt(len(m.messages) - 1)
	m.updateViewport()

	// Check if all providers have responded
	if m.compareReceived >= m.compareExpected {
		m.isStreaming = false
		m.compareExpected = 0
		m.compareReceived = 0
		elapsed := time.Since(m.streamStartTime)
		m.status = fmt.Sprintf("Compare done (%ds)", int(elapsed.Seconds()))
		m.streamStartTime = time.Time{}
	} else {
		m.status = fmt.Sprintf("Comparing... (%d/%d)", m.compareReceived, m.compareExpected)
	}

	return nil
}
