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
		want    time.Duration
	}{
		{"attempt 0 - 1s", 0, 1 * time.Second},
		{"attempt 1 - 2s", 1, 2 * time.Second},
		{"attempt 2 - 4s", 2, 4 * time.Second},
		{"attempt 3 - 8s", 3, 8 * time.Second},
		{"attempt 5 - capped at 30s", 5, 30 * time.Second},
		{"attempt 10 - capped at 30s", 10, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RetryDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("RetryDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestRetryDelayMonotonicallyIncreases(t *testing.T) {
	prev := RetryDelay(0)
	for i := 1; i < 5; i++ {
		curr := RetryDelay(i)
		if curr < prev {
			t.Errorf("RetryDelay(%d) = %v < RetryDelay(%d) = %v, expected monotonic increase", i, curr, i-1, prev)
		}
		prev = curr
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
