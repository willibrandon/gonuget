package resolver

import "sync"

// WalkerCache caches package metadata lookups for the dependency walker.
// This provides in-memory caching during a single resolution operation.
type WalkerCache struct {
	mu    sync.RWMutex
	cache map[string]*PackageDependencyInfo
}

// NewWalkerCache creates a new walker cache
func NewWalkerCache() *WalkerCache {
	return &WalkerCache{
		cache: make(map[string]*PackageDependencyInfo),
	}
}

// Get retrieves a cached package by key
func (c *WalkerCache) Get(key string) *PackageDependencyInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[key]
}

// Set caches a package by key
func (c *WalkerCache) Set(key string, info *PackageDependencyInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = info
}
