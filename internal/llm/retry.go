package llm

import (
	"math"
	"math/rand"
	"net/http"
	"time"
)

const (
	MaxRetries = 3
	BaseDelay  = 1 * time.Second
	MaxDelay   = 30 * time.Second
)

// ShouldRetry returns true if the HTTP status code warrants a retry
func ShouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// RetryDelay returns the delay before the next retry attempt using exponential backoff with jitter.
// Jitter prevents thundering herd when multiple streams are rate-limited simultaneously.
func RetryDelay(attempt int) time.Duration {
	delay := float64(BaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(MaxDelay) {
		delay = float64(MaxDelay)
	}
	// Add 0-50% random jitter
	jitter := delay * 0.5 * rand.Float64()
	return time.Duration(delay + jitter)
}
