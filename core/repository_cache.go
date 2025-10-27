package core

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
)

var (
	// globalRepositoryCacheMu protects the global repository cache
	globalRepositoryCacheMu sync.RWMutex
	// globalRepositoryCache caches SourceRepository instances by source URL
	// This enables connection pooling and provider caching across restore operations
	// Matches NuGet.Client's SourceRepositoryProvider behavior
	globalRepositoryCache = make(map[string]*SourceRepository)

	// globalMultiTierCache is the global multi-tier cache (memory + disk)
	// Shared across all repositories for maximum reuse
	globalMultiTierCache *cache.MultiTierCache
	globalCacheOnce      sync.Once
)

// initGlobalCache initializes the global multi-tier cache (once)
func initGlobalCache() {
	globalCacheOnce.Do(func() {
		// Determine cache directory (matches NuGet.Client: ~/.nuget/http-cache or ~/.local/share/NuGet/http-cache)
		cacheDir := os.Getenv("GONUGET_HTTP_CACHE")
		if cacheDir == "" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				cacheDir = filepath.Join(homeDir, ".gonuget", "http-cache")
			} else {
				// Fallback to temp directory if home dir unavailable
				cacheDir = filepath.Join(os.TempDir(), "gonuget-cache")
			}
		}

		// Create cache directory if it doesn't exist
		_ = os.MkdirAll(cacheDir, 0755)

		// Create memory cache (L1) - 1000 entries max, 100MB max size
		memCache := cache.NewMemoryCache(1000, 100*1024*1024)

		// Create disk cache (L2) - 1GB max size
		diskCache, err := cache.NewDiskCache(cacheDir, 1*1024*1024*1024)
		if err != nil {
			// Fallback to memory-only cache if disk cache creation fails
			globalMultiTierCache = cache.NewMultiTierCache(memCache, nil)
			return
		}

		// Combine into multi-tier cache
		globalMultiTierCache = cache.NewMultiTierCache(memCache, diskCache)
	})
}

// GetOrCreateRepository returns a cached SourceRepository for the given source URL,
// or creates a new one if it doesn't exist.
// This is critical for performance - it reuses HTTP clients and protocol providers
// across multiple restore operations.
func GetOrCreateRepository(sourceURL string) *SourceRepository {
	// Fast path: check if repository already exists
	globalRepositoryCacheMu.RLock()
	if repo, exists := globalRepositoryCache[sourceURL]; exists {
		globalRepositoryCacheMu.RUnlock()
		return repo
	}
	globalRepositoryCacheMu.RUnlock()

	// Slow path: create new repository
	globalRepositoryCacheMu.Lock()
	defer globalRepositoryCacheMu.Unlock()

	// Double-check in case another goroutine created it
	if repo, exists := globalRepositoryCache[sourceURL]; exists {
		return repo
	}

	// Initialize global cache (once)
	initGlobalCache()

	// Create new repository with global HTTP client AND global cache
	repo := NewSourceRepository(RepositoryConfig{
		SourceURL:  sourceURL,
		HTTPClient: nugethttp.GetGlobalClient(), // Use global HTTP client
		Cache:      globalMultiTierCache,        // Use global multi-tier cache (critical for first-run performance!)
	})

	globalRepositoryCache[sourceURL] = repo
	return repo
}

// ResetGlobalRepositoryCache clears the global repository cache (for testing only).
// WARNING: This should only be used in tests.
func ResetGlobalRepositoryCache() {
	globalRepositoryCacheMu.Lock()
	defer globalRepositoryCacheMu.Unlock()
	globalRepositoryCache = make(map[string]*SourceRepository)
}
