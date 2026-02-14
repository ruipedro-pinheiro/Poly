package shell

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/security"
)

// executeAICommand executes an AI command
func (s *Shell) executeAICommand(cmd *Command) error {
	// Substitute variables in message
	message := s.substituteVariables(cmd.Message)

	// Check if it's @all
	if cmd.Provider == "all" {
		return s.executeAllAI(message)
	}

	// Single AI
	provider, ok := llm.GetProvider(cmd.Provider)
	if !ok {
		return fmt.Errorf("unknown provider @%s", cmd.Provider)
	}

	fmt.Printf("\n\033[35m@%s\033[0m is thinking...\n", provider.Name())

	// Create messages
	messages := []llm.Message{
		{Role: "user", Content: message},
	}

	// Stream response
	start := time.Now()
	var responseBuffer strings.Builder
	
	eventCh := provider.Stream(context.Background(), messages, nil)
	for event := range eventCh {
		if event.Error != nil {
			return fmt.Errorf("AI error: %w", event.Error)
		}
		if event.Content != "" {
			fmt.Print(event.Content)
			responseBuffer.WriteString(event.Content)
		}
	}

	duration := time.Since(start)
	fmt.Printf("\n\n\033[90m(%.2fs)\033[0m\n\n", duration.Seconds())

	// Save last output
	s.lastOutput = responseBuffer.String()
	s.variables["last"] = s.lastOutput

	return nil
}

// executeAllAI executes command for all AIs
func (s *Shell) executeAllAI(message string) error {
	providers := llm.GetConfiguredProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no AI providers available")
	}

	fmt.Println()
	for i, provider := range providers {
		if i > 0 {
			fmt.Println("\n---")
		}

		fmt.Printf("\033[35m@%s\033[0m: ", provider.Name())
		
		messages := []llm.Message{
			{Role: "user", Content: message},
		}

		start := time.Now()
		eventCh := provider.Stream(context.Background(), messages, nil)
		for event := range eventCh {
			if event.Error != nil {
				fmt.Printf("\n\033[31mError: %v\033[0m", event.Error)
				break
			}
			if event.Content != "" {
				fmt.Print(event.Content)
			}
		}

		duration := time.Since(start)
		fmt.Printf(" \033[90m(%.2fs)\033[0m", duration.Seconds())
	}
	
	fmt.Println()
	return nil
}

// executeShellCommand executes a shell command
func (s *Shell) executeShellCommand(cmd *Command) error {
	if len(cmd.Parts) == 0 {
		return fmt.Errorf("no command to execute")
	}

	// Substitute variables
	for i, part := range cmd.Parts {
		cmd.Parts[i] = s.substituteVariables(part)
	}

	// Check for blocked patterns
	fullCmd := strings.Join(cmd.Parts, " ")
	if security.IsBlocked(fullCmd) {
		return fmt.Errorf("blocked: dangerous command pattern detected")
	}

	// Create command
	shellCmd := exec.Command(cmd.Parts[0], cmd.Parts[1:]...)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	// Execute
	err := shellCmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// executeShellCommandCapture executes a shell command and captures output
func executeShellCommandCapture(cmdLine string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdLine)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// executePipeline executes a piped command
func (s *Shell) executePipeline(cmd *Command) error {
	if len(cmd.Pipeline) == 0 {
		return fmt.Errorf("empty pipeline")
	}

	var input string
	var err error

	for i, stage := range cmd.Pipeline {
		if stage.IsAI {
			// AI stage - send input as prompt
			input, err = s.executeAIPipe(stage, input)
			if err != nil {
				return fmt.Errorf("pipeline stage %d failed: %w", i+1, err)
			}
		} else {
			// Shell stage
			input, err = s.executeShellPipe(stage, input)
			if err != nil {
				return fmt.Errorf("pipeline stage %d failed: %w", i+1, err)
			}
		}
	}

	// Print final output if not already printed
	if !cmd.Pipeline[len(cmd.Pipeline)-1].IsAI {
		fmt.Print(input)
	}

	// Save to $last
	s.lastOutput = input
	s.variables["last"] = input

	return nil
}

// executeShellPipe executes a shell stage in a pipeline
func (s *Shell) executeShellPipe(stage PipeStage, input string) (string, error) {
	cmd := exec.Command("sh", "-c", stage.Command)
	
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// executeAIPipe executes an AI stage in a pipeline
func (s *Shell) executeAIPipe(stage PipeStage, input string) (string, error) {
	// Get provider
	var provider llm.Provider
	var ok bool

	if stage.Provider == "all" {
		// For @all in pipe, use first provider (or implement multi-response)
		providers := llm.GetConfiguredProviders()
		if len(providers) == 0 {
			return "", fmt.Errorf("no AI providers available")
		}
		provider = providers[0]
	} else {
		provider, ok = llm.GetProvider(stage.Provider)
		if !ok {
			return "", fmt.Errorf("unknown provider @%s", stage.Provider)
		}
	}

	// Build prompt
	prompt := input
	if stage.Command != "" {
		prompt = fmt.Sprintf("%s\n\nInput:\n%s", stage.Command, input)
	}

	fmt.Printf("\n\033[35m@%s\033[0m is processing...\n\n", provider.Name())

	// Stream response
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	var responseBuffer strings.Builder
	start := time.Now()

	eventCh := provider.Stream(context.Background(), messages, nil)
	for event := range eventCh {
		if event.Error != nil {
			return "", fmt.Errorf("AI error: %w", event.Error)
		}
		if event.Content != "" {
			fmt.Print(event.Content)
			responseBuffer.WriteString(event.Content)
		}
	}

	duration := time.Since(start)
	fmt.Printf("\n\n\033[90m(%.2fs)\033[0m\n\n", duration.Seconds())

	return responseBuffer.String(), nil
}

// substituteVariables replaces $var with variable values
func (s *Shell) substituteVariables(text string) string {
	result := text
	for k, v := range s.variables {
		result = strings.ReplaceAll(result, "$"+k, v)
	}
	return result
}
