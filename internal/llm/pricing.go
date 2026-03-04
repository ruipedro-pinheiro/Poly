package llm

import (
	"fmt"
	"os"
	"sync"
)

// ModelPricing holds input/output cost per 1M tokens
type ModelPricing struct {
	Input  float64 // $ per 1M input tokens
	Output float64 // $ per 1M output tokens
}

// pricingWarnOnce tracks models we've already warned about to avoid log spam
var (
	pricingWarnedModels   = make(map[string]bool)
	pricingWarnedModelsMu sync.Mutex
)

// pricingTable maps model prefixes to pricing
// Source: provider pricing pages + OpenRouter API, March 2026
var pricingTable = map[string]ModelPricing{
	// Anthropic (claude-4.6 = latest)
	"claude-opus-4-6":   {5.0, 25.0},
	"claude-sonnet-4-6": {3.0, 15.0},
	"claude-opus-4-5":   {5.0, 25.0},
	"claude-sonnet-4-5": {3.0, 15.0},
	"claude-haiku-4-5":  {1.0, 5.0},
	"claude-opus-4-1":   {15.0, 75.0},
	"claude-sonnet-4":   {3.0, 15.0},
	"claude-opus-4":     {15.0, 75.0},

	// OpenAI — GPT-5 series (reasoning)
	"gpt-5.2-pro": {21.0, 168.0},
	"gpt-5.2":     {1.75, 14.0},
	"gpt-5.1":     {1.25, 10.0},
	"gpt-5-pro":   {15.0, 120.0},
	"gpt-5-mini":  {0.25, 2.0},
	"gpt-5-nano":  {0.05, 0.40},
	"gpt-5":       {1.25, 10.0},

	// OpenAI — GPT-4 series (non-reasoning)
	"gpt-4.1":      {2.0, 8.0},
	"gpt-4.1-mini": {0.40, 1.60},
	"gpt-4.1-nano": {0.10, 0.40},
	"gpt-4o":       {2.50, 10.0},
	"gpt-4o-mini":  {0.15, 0.60},

	// OpenAI — o-series (reasoning)
	"o3-pro":  {20.0, 80.0},
	"o3-mini": {1.10, 4.40},
	"o3":      {2.0, 8.0},
	"o4-mini": {1.10, 4.40},

	// Google Gemini
	"gemini-3.1-pro":       {2.0, 12.0},
	"gemini-3-flash":       {0.50, 3.0},
	"gemini-3.1-flash-lit": {0.25, 1.50},
	"gemini-2.5-pro":       {1.25, 10.0},
	"gemini-2.5-flash-lit": {0.10, 0.40},
	"gemini-2.5-flash":     {0.30, 2.50},
	"gemini-2.0-flash-lit": {0.075, 0.30},
	"gemini-2.0-flash":     {0.10, 0.40},

	// xAI Grok
	"grok-4-1-fast":  {0.20, 0.50},
	"grok-4-fast":    {0.20, 0.50},
	"grok-4-0":       {3.0, 15.0},
	"grok-code-fast": {0.20, 1.50},
	"grok-3-mini":    {0.30, 0.50},
	"grok-3":         {3.0, 15.0},
}

// CalculateCost calculates cost in USD from token counts and model name
func CalculateCost(inputTokens, outputTokens int, model string) float64 {
	pricing := lookupPricing(model)
	return (float64(inputTokens) * pricing.Input / 1_000_000) +
		(float64(outputTokens) * pricing.Output / 1_000_000)
}

// CalculateCostWithCache calculates cost accounting for Anthropic prompt caching.
// Cached read tokens cost 10% of input price, cache creation costs 25% more than input price.
func CalculateCostWithCache(inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int, model string) float64 {
	pricing := lookupPricing(model)
	inputCost := float64(inputTokens) * pricing.Input / 1_000_000
	outputCost := float64(outputTokens) * pricing.Output / 1_000_000
	cacheWriteCost := float64(cacheCreationTokens) * pricing.Input * 1.25 / 1_000_000
	cacheReadCost := float64(cacheReadTokens) * pricing.Input * 0.1 / 1_000_000
	return inputCost + outputCost + cacheWriteCost + cacheReadCost
}

// HasPricing returns true if the model has known pricing in the table
func HasPricing(model string) bool {
	if _, ok := pricingTable[model]; ok {
		return true
	}
	for prefix := range pricingTable {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// EstimateCascadeCost estimates the cost of sending approxTokens to multiple models
func EstimateCascadeCost(approxInputTokens int, models []string) float64 {
	var total float64
	estimatedOutput := approxInputTokens / 2
	if estimatedOutput < 500 {
		estimatedOutput = 500
	}
	for _, model := range models {
		total += CalculateCost(approxInputTokens, estimatedOutput, model)
	}
	return total
}

// lookupPricing finds pricing by matching model name prefix.
// Uses longest-prefix-match to avoid ambiguity when multiple prefixes overlap
// (e.g. "gpt-5" and "gpt-5-mini" both match "gpt-5-mini-20260301").
func lookupPricing(model string) ModelPricing {
	// Try exact match first
	if p, ok := pricingTable[model]; ok {
		return p
	}

	// Try longest prefix match (handles versioned model names like claude-sonnet-4-6-20250929)
	var bestPricing ModelPricing
	bestLen := 0
	for prefix, p := range pricingTable {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix && len(prefix) > bestLen {
			bestPricing = p
			bestLen = len(prefix)
		}
	}
	if bestLen > 0 {
		return bestPricing
	}

	// Default fallback — warn once per model so misconfiguration is visible
	pricingWarnedModelsMu.Lock()
	if !pricingWarnedModels[model] {
		pricingWarnedModels[model] = true
		fmt.Fprintf(os.Stderr, "warning: no pricing data for model %q, using fallback ($2.0/$8.0 per 1M tokens)\n", model)
	}
	pricingWarnedModelsMu.Unlock()
	return ModelPricing{2.0, 8.0}
}
