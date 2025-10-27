package http

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultRedirectCacheTTL is the default TTL for redirect cache (24 hours)
	DefaultRedirectCacheTTL = 24 * time.Hour
)

type redirectCacheEntry struct {
	TargetURL string    `json:"target_url"`
	Expires   time.Time `json:"expires"`
}

var (
	redirectDiskCacheMu   sync.RWMutex
	redirectDiskCachePath string
	redirectDiskCacheData map[string]redirectCacheEntry
	redirectDiskCacheInit sync.Once
)

// initRedirectDiskCache initializes the redirect disk cache from disk.
func initRedirectDiskCache() {
	redirectDiskCacheInit.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		redirectDiskCachePath = filepath.Join(home, ".gonuget", "redirects.json")

		// Load existing cache from disk
		data, err := os.ReadFile(redirectDiskCachePath)
		if err == nil {
			_ = json.Unmarshal(data, &redirectDiskCacheData)
		}

		if redirectDiskCacheData == nil {
			redirectDiskCacheData = make(map[string]redirectCacheEntry)
		}
	})
}

// GetCachedRedirect retrieves a cached redirect target.
// Returns final URL and true if found and not expired.
func GetCachedRedirect(sourceURL string) (string, bool) {
	initRedirectDiskCache()

	redirectDiskCacheMu.RLock()
	defer redirectDiskCacheMu.RUnlock()

	entry, exists := redirectDiskCacheData[sourceURL]
	if !exists {
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.Expires) {
		return "", false
	}

	return entry.TargetURL, true
}

// SetCachedRedirect stores a redirect target with 24h TTL.
func SetCachedRedirect(sourceURL, targetURL string) error {
	initRedirectDiskCache()

	redirectDiskCacheMu.Lock()
	defer redirectDiskCacheMu.Unlock()

	redirectDiskCacheData[sourceURL] = redirectCacheEntry{
		TargetURL: targetURL,
		Expires:   time.Now().Add(DefaultRedirectCacheTTL),
	}

	// Write to disk
	data, err := json.MarshalIndent(redirectDiskCacheData, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(redirectDiskCachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(redirectDiskCachePath, data, 0644)
}

// ClearRedirectDiskCache removes all cached redirects (for testing).
func ClearRedirectDiskCache() error {
	initRedirectDiskCache()

	redirectDiskCacheMu.Lock()
	defer redirectDiskCacheMu.Unlock()

	redirectDiskCacheData = make(map[string]redirectCacheEntry)
	return os.Remove(redirectDiskCachePath)
}
