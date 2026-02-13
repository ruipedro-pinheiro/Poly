package llm

// ModelPricing holds input/output cost per 1M tokens
type ModelPricing struct {
	Input  float64 // $ per 1M input tokens
	Output float64 // $ per 1M output tokens
}

// pricingTable maps model prefixes to pricing
// Source: provider pricing pages as of Jan 2026
var pricingTable = map[string]ModelPricing{
	// Anthropic
	"claude-opus-4":   {15.0, 75.0},
	"claude-sonnet-4": {3.0, 15.0},
	"claude-haiku-4":  {0.80, 4.0},

	// OpenAI
	"gpt-4.1":      {2.0, 8.0},
	"gpt-4.1-mini": {0.40, 1.60},
	"gpt-4.1-nano": {0.10, 0.40},
	"o3":           {10.0, 40.0},
	"o3-pro":       {20.0, 80.0},
	"o4-mini":      {1.10, 4.40},

	// Google
	"gemini-2.5-flash":      {0.15, 0.60},
	"gemini-2.5-flash-lite": {0.075, 0.30},
	"gemini-2.5-pro":        {1.25, 10.0},

	// xAI
	"grok-3":           {3.0, 15.0},
	"grok-3-fast":      {5.0, 25.0},
	"grok-3-mini-beta": {0.30, 0.50},
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

// lookupPricing finds pricing by matching model name prefix
func lookupPricing(model string) ModelPricing {
	// Try exact match first
	if p, ok := pricingTable[model]; ok {
		return p
	}

	// Try prefix match (handles versioned model names like claude-sonnet-4-5-20250929)
	for prefix, p := range pricingTable {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return p
		}
	}

	// Default fallback
	return ModelPricing{2.0, 8.0}
}
