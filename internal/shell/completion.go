package shell

import (
	"strings"

	"github.com/pedromelo/poly/internal/llm"
)

// completer implements readline.AutoCompleter
type completer struct {
	providers []string
}

func newCompleter() *completer {
	providers := llm.GetConfiguredProviders()
	providerIDs := make([]string, len(providers))
	for i, p := range providers {
		providerIDs[i] = "@" + p.Name()
	}
	return &completer{providers: providerIDs}
}

func (c *completer) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line)
	words := strings.Fields(lineStr)
	
	if len(words) == 0 {
		return c.completeEmpty()
	}

	lastWord := words[len(words)-1]

	// Complete AI providers
	if strings.HasPrefix(lastWord, "@") {
		return c.completeProvider(lastWord), len(lastWord)
	}

	// Complete built-in commands
	if strings.HasPrefix(lastWord, "!") {
		return c.completeBuiltin(lastWord), len(lastWord)
	}

	// Complete variables
	if strings.HasPrefix(lastWord, "$") {
		return c.completeVariable(lastWord), len(lastWord)
	}

	return nil, 0
}

func (c *completer) completeEmpty() ([][]rune, int) {
	suggestions := []string{
		"@all ",
		"@claude ",
		"@gemini ",
		"@gpt ",
		"@grok ",
		"!help",
		"!history",
		"!clear",
	}

	result := make([][]rune, len(suggestions))
	for i, s := range suggestions {
		result[i] = []rune(s)
	}
	return result, 0
}

func (c *completer) completeProvider(prefix string) [][]rune {
	var matches []string
	
	// Always include @all
	if strings.HasPrefix("@all", prefix) {
		matches = append(matches, "@all ")
	}

	// Add providers
	for _, p := range c.providers {
		if strings.HasPrefix(p, prefix) {
			matches = append(matches, p+" ")
		}
	}

	result := make([][]rune, len(matches))
	for i, m := range matches {
		result[i] = []rune(m)
	}
	return result
}

func (c *completer) completeBuiltin(prefix string) [][]rune {
	builtins := []string{
		"!help",
		"!history",
		"!clear",
		"!vars",
		"!providers",
		"!exit",
		"!quit",
	}

	var matches []string
	for _, b := range builtins {
		if strings.HasPrefix(b, prefix) {
			matches = append(matches, b)
		}
	}

	result := make([][]rune, len(matches))
	for i, m := range matches {
		result[i] = []rune(m)
	}
	return result
}

func (c *completer) completeVariable(prefix string) [][]rune {
	vars := []string{
		"$last",
		"$output",
		"$result",
	}

	var matches []string
	for _, v := range vars {
		if strings.HasPrefix(v, prefix) {
			matches = append(matches, v)
		}
	}

	result := make([][]rune, len(matches))
	for i, m := range matches {
		result[i] = []rune(m)
	}
	return result
}
