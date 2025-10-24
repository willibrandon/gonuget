package cache

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/willibrandon/gonuget/observability"
)

// MultiTierCache combines memory (L1) and disk (L2) caching with automatic promotion.
// When data is found in L2, it's promoted to L1 for faster subsequent access.
type MultiTierCache struct {
	l1 *MemoryCache
	l2 *DiskCache
}

// NewMultiTierCache creates a new multi-tier cache combining memory and disk layers.
func NewMultiTierCache(l1 *MemoryCache, l2 *DiskCache) *MultiTierCache {
	return &MultiTierCache{
		l1: l1,
		l2: l2,
	}
}

// Get retrieves from L1 first, then L2, promoting to L1 on L2 hit.
func (mtc *MultiTierCache) Get(ctx context.Context, sourceURL string, cacheKey string, maxAge time.Duration) ([]byte, bool, error) {
	// Check L1 (memory cache)
	if data, ok := mtc.l1.Get(cacheKey); ok {
		observability.CacheHitsTotal.WithLabelValues("memory").Inc()
		return data, true, nil
	}

	// Check L2 (disk cache)
	reader, ok, err := mtc.l2.Get(sourceURL, cacheKey, maxAge)
	if err != nil {
		observability.CacheMissesTotal.WithLabelValues("disk").Inc()
		return nil, false, err
	}
	if !ok {
		observability.CacheMissesTotal.WithLabelValues("memory").Inc()
		observability.CacheMissesTotal.WithLabelValues("disk").Inc()
		return nil, false, nil
	}
	defer func() { _ = reader.Close() }()

	// Read data from disk
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, err
	}

	// L2 hit - record metric
	observability.CacheHitsTotal.WithLabelValues("disk").Inc()
	observability.CacheMissesTotal.WithLabelValues("memory").Inc()

	// Promote to L1
	mtc.l1.Set(cacheKey, data, maxAge)

	return data, true, nil
}

// Set writes to both L1 and L2.
func (mtc *MultiTierCache) Set(ctx context.Context, sourceURL string, cacheKey string, data io.Reader, maxAge time.Duration, validate func(io.ReadSeeker) error) error {
	// Read data into memory
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	// Write to L1 (memory)
	mtc.l1.Set(cacheKey, dataBytes, maxAge)

	// Write to L2 (disk) - use bytes.NewReader for validation
	return mtc.l2.Set(sourceURL, cacheKey, bytes.NewReader(dataBytes), validate)
}

// Clear clears both caches.
func (mtc *MultiTierCache) Clear() error {
	mtc.l1.Clear()
	return mtc.l2.Clear()
}
