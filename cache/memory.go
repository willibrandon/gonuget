// Package cache provides in-memory and disk caching for NuGet operations.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Entry represents a cached value with metadata.
type Entry struct {
	Value      []byte
	Expiry     time.Time
	Size       int
	accessTime time.Time // For LRU tracking
}

// IsExpired checks if the entry has exceeded its TTL.
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.Expiry)
}

// MemoryCache is an LRU cache with TTL support.
type MemoryCache struct {
	maxEntries int
	maxSize    int64 // Maximum total bytes

	mu        sync.RWMutex
	entries   map[string]*list.Element // key -> list element
	lruList   *list.List               // LRU doubly-linked list
	totalSize int64                    // Current total bytes
}

// lruEntry wraps cache key and entry for LRU list.
type lruEntry struct {
	key   string
	entry *Entry
}

// NewMemoryCache creates a new LRU memory cache.
func NewMemoryCache(maxEntries int, maxSize int64) *MemoryCache {
	return &MemoryCache{
		maxEntries: maxEntries,
		maxSize:    maxSize,
		entries:    make(map[string]*list.Element),
		lruList:    list.New(),
		totalSize:  0,
	}
}

// Get retrieves a value from the cache.
// Returns (value, true) if found and not expired, (nil, false) otherwise.
func (mc *MemoryCache) Get(key string) ([]byte, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	elem, ok := mc.entries[key]
	if !ok {
		return nil, false
	}

	lruEnt := elem.Value.(*lruEntry)

	// Check expiration
	if lruEnt.entry.IsExpired() {
		mc.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	mc.lruList.MoveToFront(elem)
	lruEnt.entry.accessTime = time.Now()

	// Return copy to prevent external modification
	value := make([]byte, len(lruEnt.entry.Value))
	copy(value, lruEnt.entry.Value)

	return value, true
}

// Set adds or updates a value in the cache.
func (mc *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	expiry := now.Add(ttl)

	// Check if key already exists
	if elem, ok := mc.entries[key]; ok {
		// Update existing entry
		lruEnt := elem.Value.(*lruEntry)
		oldSize := lruEnt.entry.Size

		lruEnt.entry.Value = value
		lruEnt.entry.Expiry = expiry
		lruEnt.entry.Size = len(value)
		lruEnt.entry.accessTime = now

		mc.totalSize = mc.totalSize - int64(oldSize) + int64(len(value))
		mc.lruList.MoveToFront(elem)
	} else {
		// Add new entry
		entry := &Entry{
			Value:      value,
			Expiry:     expiry,
			Size:       len(value),
			accessTime: now,
		}

		lruEnt := &lruEntry{
			key:   key,
			entry: entry,
		}

		elem := mc.lruList.PushFront(lruEnt)
		mc.entries[key] = elem
		mc.totalSize += int64(len(value))
	}

	// Evict if necessary
	mc.evictIfNeeded()
}

// Delete removes a key from the cache.
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if elem, ok := mc.entries[key]; ok {
		mc.removeElement(elem)
	}
}

// Clear removes all entries from the cache.
// This matches NuGet.Client's RefreshMemoryCache behavior.
func (mc *MemoryCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.entries = make(map[string]*list.Element)
	mc.lruList = list.New()
	mc.totalSize = 0
}

// Stats returns cache statistics.
func (mc *MemoryCache) Stats() Stats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return Stats{
		Entries:   len(mc.entries),
		SizeBytes: mc.totalSize,
	}
}

// removeElement removes an element from the cache (must hold lock).
func (mc *MemoryCache) removeElement(elem *list.Element) {
	lruEnt := elem.Value.(*lruEntry)
	delete(mc.entries, lruEnt.key)
	mc.lruList.Remove(elem)
	mc.totalSize -= int64(lruEnt.entry.Size)
}

// evictIfNeeded evicts least recently used entries until within limits.
func (mc *MemoryCache) evictIfNeeded() {
	// Evict by entry count
	for mc.lruList.Len() > mc.maxEntries {
		elem := mc.lruList.Back()
		if elem != nil {
			mc.removeElement(elem)
		}
	}

	// Evict by size
	for mc.totalSize > mc.maxSize && mc.lruList.Len() > 0 {
		elem := mc.lruList.Back()
		if elem != nil {
			mc.removeElement(elem)
		}
	}
}

// Stats holds cache statistics.
type Stats struct {
	Entries   int
	SizeBytes int64
}
