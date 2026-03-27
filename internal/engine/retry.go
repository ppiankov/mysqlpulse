package engine

import (
	"context"
	"time"
)

// Pinger abstracts the PingContext method for testing.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// PingWithRetry attempts to ping with exponential backoff.
// Returns nil on success, or the last error after maxRetries attempts.
// Does not retry if ctx is cancelled.
func PingWithRetry(ctx context.Context, p Pinger, maxRetries int) error {
	backoff := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = p.PingContext(ctx)
		if lastErr == nil {
			return nil
		}

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}
	return lastErr
}
