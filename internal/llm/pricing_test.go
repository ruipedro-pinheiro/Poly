package llm

import (
	"math"
	"testing"
)

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// --- CalculateCost ---

func TestCalculateCost_KnownModelExact(t *testing.T) {
	// claude-opus-4: input=15.0, output=75.0
	got := CalculateCost(1_000_000, 1_000_000, "claude-opus-4")
	want := 90.0
	if !almostEqual(got, want, 0.001) {
		t.Errorf("CalculateCost(1M, 1M, claude-opus-4) = %f, want %f", got, want)
	}
}

func TestCalculateCost_KnownModelPrefix(t *testing.T) {
	// claude-sonnet-4-5-20250929 should match claude-sonnet-4 prefix: input=3.0, output=15.0
	got := CalculateCost(1_000_000, 1_000_000, "claude-sonnet-4-5-20250929")
	want := 18.0
	if !almostEqual(got, want, 0.001) {
		t.Errorf("CalculateCost(1M, 1M, claude-sonnet-4-5-20250929) = %f, want %f", got, want)
	}
}

func TestCalculateCost_UnknownModelFallback(t *testing.T) {
	// Unknown model uses default {2.0, 8.0}
	got := CalculateCost(1_000_000, 1_000_000, "unknown-model")
	want := 10.0
	if !almostEqual(got, want, 0.001) {
		t.Errorf("CalculateCost(1M, 1M, unknown-model) = %f, want %f", got, want)
	}
}

func TestCalculateCost_ZeroTokens(t *testing.T) {
	got := CalculateCost(0, 0, "claude-opus-4")
	if got != 0.0 {
		t.Errorf("CalculateCost(0, 0, claude-opus-4) = %f, want 0.0", got)
	}
}

func TestCalculateCost_CheapModel(t *testing.T) {
	// gemini-2.5-flash: input=0.15, output=0.60
	got := CalculateCost(1_000_000, 1_000_000, "gemini-2.5-flash")
	want := 0.75
	if !almostEqual(got, want, 0.001) {
		t.Errorf("CalculateCost(1M, 1M, gemini-2.5-flash) = %f, want %f", got, want)
	}
}

// --- CalculateCostWithCache ---

func TestCalculateCostWithCache_NoCache(t *testing.T) {
	// With zero cache tokens, should equal CalculateCost
	got := CalculateCostWithCache(1000, 500, 0, 0, "claude-sonnet-4")
	want := CalculateCost(1000, 500, "claude-sonnet-4")
	if !almostEqual(got, want, 0.0001) {
		t.Errorf("CalculateCostWithCache(no cache) = %f, want %f", got, want)
	}
}

func TestCalculateCostWithCache_WithCache(t *testing.T) {
	// claude-sonnet-4: input=3.0, output=15.0
	// inputCost: 1M * 3.0/1M = 3.0
	// outputCost: 500k * 15.0/1M = 7.5
	// cacheWriteCost: 200k * 3.0 * 1.25/1M = 0.75
	// cacheReadCost: 800k * 3.0 * 0.1/1M = 0.24
	// total = 11.49
	got := CalculateCostWithCache(1_000_000, 500_000, 200_000, 800_000, "claude-sonnet-4")
	want := 11.49
	if !almostEqual(got, want, 0.001) {
		t.Errorf("CalculateCostWithCache = %f, want %f", got, want)
	}
}

// --- HasPricing ---

func TestHasPricing_ExactMatch(t *testing.T) {
	if !HasPricing("claude-opus-4") {
		t.Error("HasPricing(claude-opus-4) should be true")
	}
}

func TestHasPricing_PrefixMatch(t *testing.T) {
	if !HasPricing("claude-opus-4-20260101") {
		t.Error("HasPricing(claude-opus-4-20260101) should be true")
	}
}

func TestHasPricing_NoMatch(t *testing.T) {
	if HasPricing("llama-3") {
		t.Error("HasPricing(llama-3) should be false")
	}
}

func TestHasPricing_EmptyString(t *testing.T) {
	if HasPricing("") {
		t.Error("HasPricing('') should be false")
	}
}

// --- EstimateCascadeCost ---

func TestEstimateCascadeCost_SingleModel(t *testing.T) {
	got := EstimateCascadeCost(10000, []string{"claude-opus-4"})
	if got <= 0 {
		t.Errorf("EstimateCascadeCost(single model) = %f, want > 0", got)
	}
}

func TestEstimateCascadeCost_MultipleModels(t *testing.T) {
	single := EstimateCascadeCost(10000, []string{"claude-opus-4"})
	multi := EstimateCascadeCost(10000, []string{"claude-opus-4", "gpt-4.1"})
	if multi <= single {
		t.Errorf("multi-model cost (%f) should be > single-model cost (%f)", multi, single)
	}
}

func TestEstimateCascadeCost_EmptyModels(t *testing.T) {
	got := EstimateCascadeCost(10000, []string{})
	if got != 0 {
		t.Errorf("EstimateCascadeCost(empty) = %f, want 0", got)
	}
}

func TestEstimateCascadeCost_LowTokensMinOutput(t *testing.T) {
	// With 100 input tokens, estimatedOutput = max(100/2, 500) = 500
	// claude-opus-4: input=15.0, output=75.0
	// cost = 100*15/1M + 500*75/1M = 0.0015 + 0.0375 = 0.039
	got := EstimateCascadeCost(100, []string{"claude-opus-4"})
	want := 0.039
	if !almostEqual(got, want, 0.001) {
		t.Errorf("EstimateCascadeCost(low tokens) = %f, want %f", got, want)
	}
}
