package persistence

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/metrics"

	"github.com/navidrome/navidrome/log"
)

// RetryWithBackoff attempts an operation with exponential backoff
// maxAttempts: maximum number of attempts (minimum 1)
// initialDelay: delay before first retry
// maxDelay: maximum delay between retries
func RetryWithBackoff(ctx context.Context, operation string, op func() error, maxAttempts int, initialDelay, maxDelay time.Duration) error {
	var lastErr error
	delay := initialDelay
	startTime := time.Now()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := op()
		if err == nil {
			if attempt > 1 {
				// Record successful retry
				metrics.IncrementSQLiteRetry(operation)
			}
			return nil
		}

		lastErr = err
		if !isRetryableError(err) {
			// Record non-retryable error
			metrics.IncrementSQLiteLockError(operation, "non_retryable")
			return err
		}

		metrics.IncrementSQLiteLockError(operation, "retryable")
		if attempt == maxAttempts {
			break
		}

		// Use exponential backoff with jitter
		jitter := time.Duration(float64(delay) * (0.5 + rand.Float64())) // 50-150% of base delay
		if jitter > maxDelay {
			jitter = maxDelay
		}

		log.Debug(ctx, "Retrying operation after error",
			"operation", operation,
			"attempt", attempt,
			"maxAttempts", maxAttempts,
			"delay", jitter,
			"error", err)

		select {
		case <-time.After(jitter):
			metrics.ObserveSQLiteLockWait(operation, jitter.Seconds())
		case <-ctx.Done():
			return ctx.Err()
		}

		delay *= 2
	}

	return lastErr
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "database is locked") ||
		strings.Contains(errStr, "busy") ||
		errors.Is(err, sql.ErrConnDone)
}
