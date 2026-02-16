package httpclient

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig holds retry configuration
// Implements retry with exponential backoff and jitter
type RetryConfig struct {
	MaxAttempts int
	WaitTime    time.Duration
	MaxWaitTime time.Duration
	// ShouldRetry is an optional function to determine if a request should be retried
	ShouldRetry func(*http.Response, error) bool
}

// Default retry configuration
var (
	DefaultRetryWaitTime    = 200 * time.Millisecond
	DefaultRetryMaxWaitTime = 10 * time.Second
	DefaultRetryAttempts    = 3
)

// executeWithRetry executes an HTTP request with exponential backoff retry
// Implements exponential backoff with jitter based on AWS best practices
// Reference: https://amazonaws-china.com/cn/blogs/architecture/exponential-backoff-and-jitter/
func executeWithRetry(ctx context.Context, client Doer, req *http.Request, config *RetryConfig) (*http.Response, error) {
	applyRetryDefaults(config)

	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		resp, lastErr = client.Do(req)

		shouldRetry := defaultShouldRetry(resp, lastErr)
		if config.ShouldRetry != nil {
			shouldRetry = config.ShouldRetry(resp, lastErr)
		}

		if !shouldRetry {
			return resp, lastErr
		}

		if resp != nil {
			_ = resp.Body.Close()
		}

		if attempt < config.MaxAttempts-1 {
			if err := waitWithBackoff(ctx, attempt, config); err != nil {
				return nil, err
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", config.MaxAttempts, lastErr)
	}
	return resp, nil
}

// applyRetryDefaults applies default values to retry configuration
func applyRetryDefaults(config *RetryConfig) {
	if config.MaxAttempts == 0 {
		config.MaxAttempts = DefaultRetryAttempts
	}
	if config.WaitTime == 0 {
		config.WaitTime = DefaultRetryWaitTime
	}
	if config.MaxWaitTime == 0 {
		config.MaxWaitTime = DefaultRetryMaxWaitTime
	}
}

// waitWithBackoff waits for the calculated backoff duration with context support
func waitWithBackoff(ctx context.Context, attempt int, config *RetryConfig) error {
	backoff := calculateBackoff(attempt, config.WaitTime, config.MaxWaitTime)
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// defaultShouldRetry determines if a request should be retried
// Retry on network errors or 5xx server errors
func defaultShouldRetry(resp *http.Response, err error) bool {
	// Network error - retry
	if err != nil {
		return true
	}

	// No response - retry
	if resp == nil {
		return true
	}

	// Server errors (5xx) - retry
	if resp.StatusCode >= 500 {
		return true
	}

	// Too Many Requests (429) - retry
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}

	// Success or client error - don't retry
	return false
}

// calculateBackoff calculates exponential backoff with jitter
// Formula: min(maxWaitTime, waitTime * 2^attempt) + random jitter
func calculateBackoff(attempt int, waitTime, maxWaitTime time.Duration) time.Duration {
	// Calculate exponential backoff
	backoff := waitTime * time.Duration(math.Pow(2, float64(attempt)))

	// Cap at max wait time
	if backoff > maxWaitTime {
		backoff = maxWaitTime
	}

	// Add jitter (50% to 100% of backoff)
	jitter := backoff / 2
	if jitter > 0 {
		jitter = time.Duration(rand.Int63n(int64(jitter)))
	}

	return backoff/2 + jitter
}
