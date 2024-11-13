package queue

import (
	"context"
	"sync"
	"time"
)

// RateLimiter controls the rate of calls for different devices.
type RateLimiter struct {
	mu        sync.Mutex          // Mutex to protect access to lastCalls
	lastCalls map[int64]time.Time // deviceID -> last call time
	minWait   time.Duration       // Minimum wait time between calls
}

// NewRateLimiter creates a new RateLimiter with a specified minimum wait time.
func NewRateLimiter(minWait time.Duration) *RateLimiter {
	return &RateLimiter{
		lastCalls: make(map[int64]time.Time), // Initialize the lastCalls map
		minWait:   minWait,                   // Set the minimum wait time
	}
}

// Wait blocks until the device with the given deviceID can make a new call.
func (r *RateLimiter) Wait(ctx context.Context, deviceID int64) error {
	r.mu.Lock()                               // Lock to ensure thread-safe access to lastCalls
	lastCall, exists := r.lastCalls[deviceID] // Get the last call time for the device
	if exists {
		waitTime := r.minWait - time.Since(lastCall) // Calculate the wait time
		if waitTime > 0 {                            // If the wait time is positive, we need to wait
			r.mu.Unlock() // Unlock before waiting
			select {
			case <-ctx.Done(): // Check if the context is done
				return ctx.Err() // Return context error if done
			case <-time.After(waitTime): // Wait for the required time
			}
			r.mu.Lock() // Lock again after waiting
		}
	}
	r.lastCalls[deviceID] = time.Now() // Update the last call time for the device
	r.mu.Unlock()                      // Unlock after updating
	return nil                         // Return nil indicating success
}
