package shell

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/pedromelo/poly/internal/llm"
)

// Shell represents the hybrid interactive shell
type Shell struct {
	rl         *readline.Instance
	history    []string
	variables  map[string]string
	lastOutput string
	prompt     string
}

// New creates a new hybrid shell instance
func New() (*Shell, error) {
	// Create readline instance with completion
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[36mpoly>\033[0m ",
		HistoryFile:     os.ExpandEnv("$HOME/.poly/shell_history"),
		AutoComplete:    newCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create readline: %w", err)
	}

	s := &Shell{
		rl:        rl,
		history:   make([]string, 0),
		variables: make(map[string]string),
		prompt:    "\033[36mpoly>\033[0m ",
	}

	return s, nil
}

// Run starts the interactive shell loop
func (s *Shell) Run() error {
	defer s.rl.Close()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nBye! 👋")
		os.Exit(0)
	}()

	// Welcome message
	s.printWelcome()

	// Main loop
	for {
		line, err := s.rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Add to history
		s.history = append(s.history, line)

		// Process command
		if err := s.processCommand(line); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
		}
	}

	return nil
}

// processCommand parses and executes a command
func (s *Shell) processCommand(line string) error {
	// Check for built-in commands
	if strings.HasPrefix(line, "!") {
		return s.handleBuiltin(line[1:])
	}

	// Parse the command
	cmd, err := parseCommand(line)
	if err != nil {
		return err
	}

	// Execute based on type
	switch cmd.Type {
	case CommandTypeAI:
		return s.executeAICommand(cmd)
	case CommandTypeShell:
		return s.executeShellCommand(cmd)
	case CommandTypePipe:
		return s.executePipeline(cmd)
	case CommandTypeVariable:
		return s.setVariable(cmd)
	default:
		return fmt.Errorf("unknown command type")
	}
}

// handleBuiltin handles built-in shell commands
func (s *Shell) handleBuiltin(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "history":
		s.printHistory()
	case "clear":
		fmt.Print("\033[H\033[2J")
	case "help":
		s.printHelp()
	case "vars":
		s.printVariables()
	case "providers":
		s.printProviders()
	case "exit", "quit":
		fmt.Println("Bye! 👋")
		os.Exit(0)
	default:
		return fmt.Errorf("unknown built-in command: %s", parts[0])
	}
	return nil
}

// printWelcome prints the welcome message
func (s *Shell) printWelcome() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║  🚀 POLY HYBRID SHELL v2.0 - Quantum Edition 🚀          ║
║                                                           ║
║  Multi-AI Interactive Shell with superpowers!            ║
║  Type 'help' for commands or start chatting with AIs     ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	
	// List available providers
	providers := llm.GetConfiguredProviders()
	fmt.Printf("Available AIs: ")
	for i, p := range providers {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("\033[35m@%s\033[0m", p.Name())
	}
	fmt.Println()
}

// printHelp prints help information
func (s *Shell) printHelp() {
	help := `
🎯 POLY HYBRID SHELL - Commands:

AI COMMANDS:
  @claude hello              → Chat with Claude
  @all what is 2+2          → Ask all AIs
  @gemini explain this code  → Ask Gemini
  
SHELL COMMANDS:
  ls -la                     → Normal shell command
  cat file.txt               → Execute any shell command
  
PIPES (🔥 POWERFUL):
  ls | @claude explain       → Pipe shell output to AI
  git diff | @all review     → Send diff to all AIs
  cat code.go | @gpt optimize → AI code review
  
VARIABLES:
  $output = ls               → Save command output
  $result = @claude test     → Save AI response
  echo $output               → Use variables
  
BUILT-IN COMMANDS:
  !history                   → Show command history
  !clear                     → Clear screen
  !vars                      → Show variables
  !providers                 → List AI providers
  !help                      → Show this help
  !exit / !quit              → Exit shell

TIPS:
  • Use Tab for completion
  • Use ↑↓ for history
  • Ctrl+C to interrupt
  • Ctrl+D to exit
`
	fmt.Println(help)
}

// printHistory prints command history
func (s *Shell) printHistory() {
	fmt.Println("\n📜 Command History:")
	for i, cmd := range s.history {
		fmt.Printf("  %3d  %s\n", i+1, cmd)
	}
	fmt.Println()
}

// printVariables prints all variables
func (s *Shell) printVariables() {
	if len(s.variables) == 0 {
		fmt.Println("No variables set.")
		return
	}
	fmt.Println("\n📦 Variables:")
	for k, v := range s.variables {
		preview := v
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		fmt.Printf("  \033[33m$%s\033[0m = %s\n", k, preview)
	}
	fmt.Println()
}

// printProviders prints available AI providers
func (s *Shell) printProviders() {
	providers := llm.GetConfiguredProviders()
	fmt.Println("\n🤖 Available AI Providers:")
	for _, p := range providers {
		fmt.Printf("  \033[35m@%s\033[0m\n", p.Name())
	}
	fmt.Println()
}

// setVariable sets a shell variable
func (s *Shell) setVariable(cmd *Command) error {
	// Extract variable name and value
	// Format: $var = command
	parts := strings.SplitN(cmd.Raw, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid variable assignment")
	}
	
	varName := strings.TrimSpace(strings.TrimPrefix(parts[0], "$"))
	valueCmd := strings.TrimSpace(parts[1])
	
	// Execute the command to get the value
	valueParts := strings.Fields(valueCmd)
	if len(valueParts) == 0 {
		return fmt.Errorf("no command to execute for variable")
	}
	
	// Execute and capture output
	output, err := executeShellCommandCapture(valueCmd)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	
	s.variables[varName] = output
	fmt.Printf("\033[33m$%s\033[0m set (%d bytes)\n", varName, len(output))
	
	return nil
}

// Close closes the shell
func (s *Shell) Close() error {
	return s.rl.Close()
}
