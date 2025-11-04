// Package resolver implements NuGet dependency resolution algorithms.
package resolver

import (
	"context"
	"sync"
	"time"
)

// WalkerCache caches package metadata lookups for the dependency walker.
// Uses two-tier caching: operation cache for in-flight operations, result cache for completed lookups.
type WalkerCache struct {
	operationCache *OperationCache
	resultCache    sync.Map // key -> *PackageDependencyInfo (fast path)
}

// NewWalkerCache creates a new walker cache
func NewWalkerCache() *WalkerCache {
	return &WalkerCache{
		operationCache: NewOperationCache(5 * time.Minute),
	}
}

// GetOrFetch gets cached result or fetches via operation
func (c *WalkerCache) GetOrFetch(
	ctx context.Context,
	key string,
	fetcher func(context.Context) (*PackageDependencyInfo, error),
) (*PackageDependencyInfo, error) {
	// Fast path: check result cache
	if cached, ok := c.resultCache.Load(key); ok {
		return cached.(*PackageDependencyInfo), nil
	}

	// Slow path: use operation cache
	result, err := c.operationCache.GetOrStart(ctx, key, fetcher)
	if err != nil {
		return nil, err
	}

	// Store in result cache for fast future lookups
	if result != nil {
		c.resultCache.Store(key, result)
	}

	return result, nil
}
