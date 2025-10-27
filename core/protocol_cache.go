package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultProtocolCacheTTL is the default TTL for protocol detection cache (24 hours)
	DefaultProtocolCacheTTL = 24 * time.Hour
)

type protocolCacheEntry struct {
	Protocol string    `json:"protocol"`
	Expires  time.Time `json:"expires"`
}

var (
	protocolCacheMu   sync.RWMutex
	protocolCachePath string
	protocolCacheData map[string]protocolCacheEntry
	protocolCacheInit sync.Once
)

// initProtocolCache initializes the protocol cache from disk.
func initProtocolCache() {
	protocolCacheInit.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		protocolCachePath = filepath.Join(home, ".gonuget", "protocol_cache.json")

		// Load existing cache from disk
		data, err := os.ReadFile(protocolCachePath)
		if err == nil {
			_ = json.Unmarshal(data, &protocolCacheData)
		}

		if protocolCacheData == nil {
			protocolCacheData = make(map[string]protocolCacheEntry)
		}
	})
}

// GetCachedProtocol retrieves a cached protocol detection result.
// Returns protocol version ("v2" or "v3") and true if found and not expired.
func GetCachedProtocol(sourceURL string) (string, bool) {
	initProtocolCache()

	protocolCacheMu.RLock()
	defer protocolCacheMu.RUnlock()

	entry, exists := protocolCacheData[sourceURL]
	if !exists {
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.Expires) {
		return "", false
	}

	return entry.Protocol, true
}

// SetCachedProtocol stores a protocol detection result with 24h TTL.
func SetCachedProtocol(sourceURL, protocol string) error {
	initProtocolCache()

	protocolCacheMu.Lock()
	defer protocolCacheMu.Unlock()

	protocolCacheData[sourceURL] = protocolCacheEntry{
		Protocol: protocol,
		Expires:  time.Now().Add(DefaultProtocolCacheTTL),
	}

	// Write to disk
	data, err := json.MarshalIndent(protocolCacheData, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(protocolCachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(protocolCachePath, data, 0644)
}

// ClearProtocolCache removes all cached protocol detection results (for testing).
func ClearProtocolCache() error {
	initProtocolCache()

	protocolCacheMu.Lock()
	defer protocolCacheMu.Unlock()

	protocolCacheData = make(map[string]protocolCacheEntry)
	return os.Remove(protocolCachePath)
}
