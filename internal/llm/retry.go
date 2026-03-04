package llm

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/pedromelo/poly/internal/security"
)

const (
	MaxRetries = 3
	BaseDelay  = 1 * time.Second
	MaxDelay   = 30 * time.Second
)

// DoWithRetry executes an HTTP request with exponential backoff and jitter.
// It handles status code checking, error body sanitization, and body reset for retries.
func DoWithRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RetryDelay(attempt - 1)):
			}

			// If the request has a body, we need to reset it for the next attempt.
			// http.NewRequestWithContext sets req.GetBody for bytes.Reader, which we use.
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("failed to reset request body: %w", err)
				}
				req.Body = body
			}
		}

		resp, lastErr = client.Do(req)
		if lastErr != nil {
			// Network errors are retriable
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		// Read and sanitize error body for the error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, security.SanitizeResponseBody(bodyBytes))

		if !ShouldRetry(resp.StatusCode) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

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
	// Add 0-50% random jitter before capping to ensure result never exceeds MaxDelay
	jitter := delay * 0.5 * rand.Float64()
	delay += jitter
	if delay > float64(MaxDelay) {
		delay = float64(MaxDelay)
	}
	return time.Duration(delay)
}
