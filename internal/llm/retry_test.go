package llm

import (
	"net/http"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"200 OK - no retry", http.StatusOK, false},
		{"201 Created - no retry", http.StatusCreated, false},
		{"400 Bad Request - no retry", http.StatusBadRequest, false},
		{"401 Unauthorized - no retry", http.StatusUnauthorized, false},
		{"403 Forbidden - no retry", http.StatusForbidden, false},
		{"404 Not Found - no retry", http.StatusNotFound, false},
		{"429 Too Many Requests - retry", http.StatusTooManyRequests, true},
		{"500 Internal Server Error - retry", http.StatusInternalServerError, true},
		{"502 Bad Gateway - retry", http.StatusBadGateway, true},
		{"503 Service Unavailable - retry", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout - retry", http.StatusGatewayTimeout, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRetry(tt.statusCode)
			if got != tt.want {
				t.Errorf("ShouldRetry(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		minWant time.Duration // base delay (without jitter)
		maxWant time.Duration // base + 50% jitter, capped at MaxDelay
	}{
		{"attempt 0 - ~1s", 0, 1 * time.Second, 1500 * time.Millisecond},
		{"attempt 1 - ~2s", 1, 2 * time.Second, 3 * time.Second},
		{"attempt 2 - ~4s", 2, 4 * time.Second, 6 * time.Second},
		{"attempt 3 - ~8s", 3, 8 * time.Second, 12 * time.Second},
		{"attempt 5 - capped at 30s", 5, 20 * time.Second, 30 * time.Second},
		{"attempt 10 - capped at 30s", 10, 20 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RetryDelay(tt.attempt)
			if got < tt.minWant || got > tt.maxWant {
				t.Errorf("RetryDelay(%d) = %v, want between %v and %v", tt.attempt, got, tt.minWant, tt.maxWant)
			}
		})
	}
}

func TestRetryDelayGenerallyIncreases(t *testing.T) {
	// With jitter, individual calls may not be strictly monotonic.
	// Instead, verify that the average over multiple samples increases.
	const samples = 50
	averages := make([]float64, 6)
	for attempt := 0; attempt < 6; attempt++ {
		var total time.Duration
		for i := 0; i < samples; i++ {
			total += RetryDelay(attempt)
		}
		averages[attempt] = float64(total) / float64(samples)
	}
	for i := 1; i < 5; i++ {
		if averages[i] < averages[i-1] {
			t.Errorf("Average RetryDelay(%d) = %.0fns < average RetryDelay(%d) = %.0fns", i, averages[i], i-1, averages[i-1])
		}
	}
}

func TestRetryDelayNeverExceedsMax(t *testing.T) {
	for i := 0; i <= 20; i++ {
		d := RetryDelay(i)
		if d > MaxDelay {
			t.Errorf("RetryDelay(%d) = %v, exceeds MaxDelay %v", i, d, MaxDelay)
		}
	}
}

func TestConstants(t *testing.T) {
	if MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", MaxRetries)
	}
	if BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", BaseDelay)
	}
	if MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", MaxDelay)
	}
}
