package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// TokenBucketConfig holds token bucket configuration.
type TokenBucketConfig struct {
	// Capacity is the maximum number of tokens in bucket.
	Capacity int

	// RefillRate is tokens added per second.
	RefillRate float64

	// InitialTokens is the number of tokens at startup (default: Capacity).
	InitialTokens int
}

// DefaultTokenBucketConfig returns default configuration.
// Default: 100 req/s burst, 50 req/s sustained.
func DefaultTokenBucketConfig() TokenBucketConfig {
	return TokenBucketConfig{
		Capacity:      100,
		RefillRate:    50.0,
		InitialTokens: 100,
	}
}

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket struct {
	mu sync.Mutex

	capacity     int
	refillRate   float64
	tokens       float64
	lastRefillAt time.Time
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(config TokenBucketConfig) *TokenBucket {
	initialTokens := config.InitialTokens
	if initialTokens == 0 {
		initialTokens = config.Capacity
	}
	if initialTokens > config.Capacity {
		initialTokens = config.Capacity
	}

	return &TokenBucket{
		capacity:     config.Capacity,
		refillRate:   config.RefillRate,
		tokens:       float64(initialTokens),
		lastRefillAt: time.Now(),
	}
}

// refill adds tokens based on elapsed time.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillAt).Seconds()
	tb.lastRefillAt = now

	// Add tokens based on refill rate and elapsed time
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
}

// Allow checks if a request can proceed (non-blocking).
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// AllowN checks if N requests can proceed (non-blocking).
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	needed := float64(n)
	if tb.tokens >= needed {
		tb.tokens -= needed
		return true
	}

	return false
}

// Wait blocks until a token is available or context is cancelled.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}

		// Calculate wait time until next token
		waitTime := tb.calculateWaitTime(1)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Retry after wait
		}
	}
}

// WaitN blocks until N tokens are available or context is cancelled.
func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
	for {
		if tb.AllowN(n) {
			return nil
		}

		waitTime := tb.calculateWaitTime(n)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Retry after wait
		}
	}
}

// calculateWaitTime calculates how long to wait for n tokens.
func (tb *TokenBucket) calculateWaitTime(n int) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	deficit := float64(n) - tb.tokens
	if deficit <= 0 {
		return 0
	}

	// Calculate time needed to accumulate deficit tokens
	seconds := deficit / tb.refillRate
	return time.Duration(seconds * float64(time.Second))
}

// Tokens returns the current number of available tokens.
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// Stats returns current rate limiter statistics.
func (tb *TokenBucket) Stats() TokenBucketStats {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	return TokenBucketStats{
		Capacity:   tb.capacity,
		RefillRate: tb.refillRate,
		Tokens:     tb.tokens,
	}
}

// TokenBucketStats holds token bucket statistics.
type TokenBucketStats struct {
	Capacity   int
	RefillRate float64
	Tokens     float64
}
