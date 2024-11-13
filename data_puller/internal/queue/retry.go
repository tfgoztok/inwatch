package queue

import (
	"context"
	"time"
)

type RetryPolicy struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:  3,
		InitialWait: 1 * time.Second,
		MaxWait:     30 * time.Second,
	}
}

func (rp RetryPolicy) DoWithRetry(ctx context.Context, op func() error) error {
	var lastErr error
	wait := rp.InitialWait

	for attempt := 0; attempt <= rp.MaxRetries; attempt++ {
		// Execute the operation and check for errors
		if err := op(); err != nil {
			lastErr = err
			// If the maximum number of retries has been reached, exit the loop
			if attempt == rp.MaxRetries {
				break
			}

			// Wait for the context to be done or for the specified wait time
			select {
			case <-ctx.Done():
				return ctx.Err() // Return context error if done
			case <-time.After(wait):
			}

			// Exponential backoff: double the wait time for the next attempt
			wait *= 2
			if wait > rp.MaxWait {
				wait = rp.MaxWait // Cap the wait time at MaxWait
			}
			continue
		}
		return nil // Return nil if the operation was successful
	}

	return lastErr // Return the last error encountered
}
