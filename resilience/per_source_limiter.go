package resilience

import (
	"context"
	"sync"
)

// PerSourceLimiter manages separate rate limiters for each source.
type PerSourceLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*TokenBucket
	config   TokenBucketConfig
}

// NewPerSourceLimiter creates a per-source rate limiter.
func NewPerSourceLimiter(config TokenBucketConfig) *PerSourceLimiter {
	return &PerSourceLimiter{
		limiters: make(map[string]*TokenBucket),
		config:   config,
	}
}

// NewPerSourceLimiterWithDefaults creates limiter with default config.
func NewPerSourceLimiterWithDefaults() *PerSourceLimiter {
	return NewPerSourceLimiter(DefaultTokenBucketConfig())
}

// getLimiter returns rate limiter for a source, creating if needed.
func (psl *PerSourceLimiter) getLimiter(source string) *TokenBucket {
	psl.mu.RLock()
	limiter, exists := psl.limiters[source]
	psl.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	psl.mu.Lock()
	defer psl.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = psl.limiters[source]
	if exists {
		return limiter
	}

	limiter = NewTokenBucket(psl.config)
	psl.limiters[source] = limiter
	return limiter
}

// Allow checks if a request to source can proceed.
func (psl *PerSourceLimiter) Allow(source string) bool {
	limiter := psl.getLimiter(source)
	return limiter.Allow()
}

// AllowN checks if N requests to source can proceed.
func (psl *PerSourceLimiter) AllowN(source string, n int) bool {
	limiter := psl.getLimiter(source)
	return limiter.AllowN(n)
}

// Wait blocks until a token is available for source.
func (psl *PerSourceLimiter) Wait(ctx context.Context, source string) error {
	limiter := psl.getLimiter(source)
	return limiter.Wait(ctx)
}

// WaitN blocks until N tokens are available for source.
func (psl *PerSourceLimiter) WaitN(ctx context.Context, source string, n int) error {
	limiter := psl.getLimiter(source)
	return limiter.WaitN(ctx, n)
}

// GetStats returns statistics for a specific source.
func (psl *PerSourceLimiter) GetStats(source string) *TokenBucketStats {
	psl.mu.RLock()
	limiter, exists := psl.limiters[source]
	psl.mu.RUnlock()

	if !exists {
		return nil
	}

	stats := limiter.Stats()
	return &stats
}

// GetAllStats returns statistics for all sources.
func (psl *PerSourceLimiter) GetAllStats() map[string]TokenBucketStats {
	psl.mu.RLock()
	defer psl.mu.RUnlock()

	stats := make(map[string]TokenBucketStats, len(psl.limiters))
	for source, limiter := range psl.limiters {
		stats[source] = limiter.Stats()
	}

	return stats
}
