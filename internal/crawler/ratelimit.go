package crawler

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const maxRetries = 5

// rateLimitTransport is an http.RoundTripper that handles 429 responses
// with exponential backoff and jitter.
type rateLimitTransport struct {
	base http.RoundTripper
}

// newRateLimitTransport wraps the given transport with rate-limit handling.
func newRateLimitTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &rateLimitTransport{base: base}
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := range maxRetries {
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, fmt.Errorf("round trip: %w", err)
		}

		// Log rate-limit headers at debug level.
		if remaining := resp.Header.Get("Ratelimit-Remaining"); remaining != "" {
			slog.Debug("rate-limit headers",
				"remaining", remaining,
				"reset", resp.Header.Get("Ratelimit-Reset"),
			)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Check context cancellation.
		if req.Context().Err() != nil {
			return resp, nil
		}

		waitDuration := calculateWait(resp, attempt)
		slog.Warn("rate limited, backing off",
			"attempt", attempt+1,
			"wait", waitDuration,
		)

		timer := time.NewTimer(waitDuration)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return resp, nil
		case <-timer.C:
		}

		// Close the 429 response body before retry.
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Debug("failed to close 429 response body", "error", closeErr)
		}
	}

	return resp, fmt.Errorf("exhausted %d retries after rate limiting: %w", maxRetries, err)
}

func calculateWait(resp *http.Response, attempt int) time.Duration {
	// Try Retry-After header first.
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}

	// Try RateLimit-Reset header (unix timestamp).
	if resetStr := resp.Header.Get("Ratelimit-Reset"); resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			wait := time.Until(time.Unix(resetUnix, 0))
			if wait > 0 {
				return wait
			}
		}
	}

	// Exponential backoff with jitter.
	base := math.Pow(2, float64(attempt)) //nolint:mnd // exponential base
	jitter := rand.Float64() * base       //nolint:gosec // jitter doesn't need crypto rand
	return time.Duration(base+jitter) * time.Second
}
