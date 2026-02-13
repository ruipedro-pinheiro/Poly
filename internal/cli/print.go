package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/llm"
)

// RunPrint handles non-interactive mode: sends prompt to default provider,
// streams response to stdout, returns exit code.
func RunPrint(prompt string) int {
	cfg := config.Get()

	// Get default provider
	providerName := cfg.DefaultProvider
	if providerName == "" {
		providerName = "claude"
	}

	provider, ok := llm.GetProvider(providerName)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: provider %q not found\n", providerName)
		return 1
	}

	if !provider.IsConfigured() {
		fmt.Fprintf(os.Stderr, "Error: provider %q is not configured. Run poly to set it up.\n", providerName)
		return 1
	}

	// Set default model
	if model := llm.GetDefaultModel(providerName); model != "" {
		provider.SetModel(model)
	}

	// Build system prompt
	systemPrompt := llm.BuildSystemPrompt(providerName, "default")

	// Build messages
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	// Stream response (no tools in print mode)
	ctx := context.Background()
	ch := provider.Stream(ctx, messages, nil)

	for event := range ch {
		switch event.Type {
		case "content":
			fmt.Print(event.Content)
		case "error":
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			return 1
		case "done":
			// Print final newline if needed
			fmt.Println()
			return 0
		}
	}

	fmt.Println()
	return 0
}
