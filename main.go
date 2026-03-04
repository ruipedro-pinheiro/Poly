package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/pedromelo/poly/internal/cli"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
	"github.com/pedromelo/poly/internal/mcp"
	"github.com/pedromelo/poly/internal/sandbox"
	"github.com/pedromelo/poly/internal/shell"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tools"
	"github.com/pedromelo/poly/internal/tui"
	"golang.org/x/term"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	// Parse flags
	showVersion := flag.Bool("version", false, "Print version and exit")
	shellMode := flag.Bool("shell", false, "Start in interactive shell mode")
	interactiveMode := flag.Bool("i", false, "Start in interactive shell mode (alias)")
	printMode := flag.String("print", "", "Non-interactive mode: send prompt and print response")
	printAlias := flag.String("p", "", "Non-interactive mode (alias for --print)")
	jsonMode := flag.Bool("json", false, "Output streaming events as NDJSON")
	sandboxMode := flag.Bool("sandbox", false, "Run bash commands in a container (podman/docker)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("poly %s\n", version)
		os.Exit(0)
	}

	// Load config from ~/.poly/config.json (merges with defaults)
	if _, err := config.Load(); err != nil {
		log.Printf("warning: failed to load config: %v", err)
	}

	// Apply saved color theme
	if savedTheme := config.GetColorTheme(); savedTheme != "" {
		name := theme.ThemeName(savedTheme)
		if _, ok := theme.Palettes[name]; ok {
			theme.SetTheme(name)
		}
	}

	// Initialize sandbox from config or CLI flag
	if *sandboxMode || config.SandboxEnabled() {
		if sandbox.Available() {
			sandbox.Enabled = true
			sandbox.Image = config.GetSandboxImage()
		}
	}

	// Initialize tools
	tools.Init()

	// Register all built-in providers (after config.Load so they get actual user config)
	llm.RegisterAllProviders()

	// Load custom providers from ~/.poly/providers.json
	if err := llm.LoadCustomProviders(); err != nil {
		log.Printf("warning: failed to load custom providers: %v", err)
	}

	// Initialize sub-provider for delegate_task tool (lazy - resolves at call time)
	llm.InitSubProvider()

	// Initialize MCP servers (connects to configured servers, registers their tools)
	mcp.Init()
	if mcp.Global != nil {
		defer mcp.Global.Close()
	}

	// Determine print prompt (--print takes priority over -p)
	printPrompt := *printMode
	if printPrompt == "" {
		printPrompt = *printAlias
	}

	// Check for piped stdin
	stdinContent := ""
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		data, err := io.ReadAll(os.Stdin)
		if err == nil && len(data) > 0 {
			stdinContent = string(data)
		}
	}

	// If stdin has content, enable print mode
	if stdinContent != "" {
		if printPrompt != "" {
			// Combine: stdin content + prompt
			printPrompt = strings.TrimSpace(stdinContent) + "\n\n" + printPrompt
		} else {
			// Stdin only (no --print flag, but piped input)
			printPrompt = strings.TrimSpace(stdinContent)
		}
	}

	// Non-interactive JSON mode
	if *jsonMode {
		if printPrompt == "" {
			fmt.Fprintln(os.Stderr, "Error: --json requires a prompt via --print/-p or piped stdin")
			os.Exit(1)
		}
		os.Exit(cli.RunJSON(printPrompt))
	}

	// Non-interactive print mode
	if printPrompt != "" {
		os.Exit(cli.RunPrint(printPrompt))
	}

	// Check if shell mode is requested
	if *shellMode || *interactiveMode {
		if err := runShell(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running shell: %v\n", err)
			if mcp.Global != nil {
				mcp.Global.Close()
			}
			os.Exit(1)
		}
		return
	}

	// Default: run TUI
	if err := runTUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Poly: %v\n", err)
		if mcp.Global != nil {
			mcp.Global.Close()
		}
		os.Exit(1)
	}
}

func runShell() error {
	sh, err := shell.New()
	if err != nil {
		return err
	}
	defer sh.Close()

	return sh.Run()
}

func runTUI() error {
	p := tea.NewProgram(
		tui.New(),
		// AltScreen and MouseMode are now set via tea.View in View()
	)

	_, err := p.Run()
	return err
}
