package shell

import (
	"fmt"
	"strings"
)

// CommandType represents the type of command
type CommandType int

const (
	CommandTypeUnknown CommandType = iota
	CommandTypeAI      // @provider message
	CommandTypeShell   // regular shell command
	CommandTypePipe    // command | @provider
	CommandTypeVariable // $var = command
)

// Command represents a parsed command
type Command struct {
	Type     CommandType
	Raw      string
	Provider string   // For AI commands
	Message  string   // For AI commands
	Parts    []string // For shell commands
	Pipeline []PipeStage // For piped commands
}

// PipeStage represents a stage in a pipeline
type PipeStage struct {
	IsAI     bool
	Provider string
	Command  string
}

// parseCommand parses a command line into a Command struct
func parseCommand(line string) (*Command, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty command")
	}

	cmd := &Command{Raw: line}

	// Check for variable assignment
	if strings.Contains(line, "=") && strings.HasPrefix(line, "$") {
		cmd.Type = CommandTypeVariable
		return cmd, nil
	}

	// Check for pipe
	if strings.Contains(line, "|") {
		return parsePipeline(line)
	}

	// Check for AI command (starts with @)
	if strings.HasPrefix(line, "@") {
		return parseAICommand(line)
	}

	// Default: shell command
	cmd.Type = CommandTypeShell
	cmd.Parts = splitCommand(line)
	return cmd, nil
}

// parseAICommand parses an AI command (@provider message)
func parseAICommand(line string) (*Command, error) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid AI command")
	}

	provider := strings.TrimPrefix(parts[0], "@")
	message := ""
	if len(parts) == 2 {
		message = parts[1]
	}

	return &Command{
		Type:     CommandTypeAI,
		Raw:      line,
		Provider: provider,
		Message:  message,
	}, nil
}

// parsePipeline parses a piped command (cmd1 | cmd2 | @ai)
func parsePipeline(line string) (*Command, error) {
	stages := strings.Split(line, "|")
	if len(stages) < 2 {
		return nil, fmt.Errorf("invalid pipeline")
	}

	cmd := &Command{
		Type:     CommandTypePipe,
		Raw:      line,
		Pipeline: make([]PipeStage, 0, len(stages)),
	}

	for _, stage := range stages {
		stage = strings.TrimSpace(stage)
		if stage == "" {
			continue
		}

		pipeStage := PipeStage{}
		
		// Check if this stage is an AI command
		if strings.HasPrefix(stage, "@") {
			parts := strings.SplitN(stage, " ", 2)
			pipeStage.IsAI = true
			pipeStage.Provider = strings.TrimPrefix(parts[0], "@")
			if len(parts) == 2 {
				pipeStage.Command = parts[1]
			}
		} else {
			pipeStage.IsAI = false
			pipeStage.Command = stage
		}

		cmd.Pipeline = append(cmd.Pipeline, pipeStage)
	}

	return cmd, nil
}

// splitCommand splits a command line into parts, respecting quotes
func splitCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range line {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
