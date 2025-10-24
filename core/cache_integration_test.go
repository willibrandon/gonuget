package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
)

func TestV3Provider_GetMetadata_CacheHit(t *testing.T) {
	var metadataCallCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			// Service index
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl/3.6.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/registration/testpkg/index.json":
			metadataCallCount++
			// Return full registration index with inline items
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/testpkg/page.json",
						"lower": "1.0.0",
						"upper": "1.0.0",
						"count": 1,
						"items": []map[string]any{
							{
								"@id": "http://" + r.Host + "/registration/testpkg/1.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/testpkg/1.0.0.json",
									"id":      "TestPkg",
									"version": "1.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	// Create repository with cache
	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()
	cacheCtx := cache.NewSourceCacheContext()

	// First call - should miss cache and hit server
	metadata1, err := repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() first call error = %v", err)
	}
	if metadata1 == nil {
		t.Fatal("GetMetadata() returned nil metadata")
	}
	if metadata1.ID != "TestPkg" {
		t.Errorf("GetMetadata() ID = %s, want TestPkg", metadata1.ID)
	}

	initialMetadataCallCount := metadataCallCount

	// Second call - should hit cache, no server call to metadata endpoint
	metadata2, err := repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() second call error = %v", err)
	}
	if metadata2 == nil {
		t.Fatal("GetMetadata() returned nil metadata on cache hit")
	}
	if metadata2.ID != metadata1.ID {
		t.Errorf("Cached metadata ID = %s, want %s", metadata2.ID, metadata1.ID)
	}

	// Verify no additional metadata endpoint calls (cache hit)
	if metadataCallCount != initialMetadataCallCount {
		t.Errorf("Cache miss detected: metadataCallCount increased from %d to %d", initialMetadataCallCount, metadataCallCount)
	}
}

func TestV3Provider_GetMetadata_NoCache(t *testing.T) {
	var metadataCallCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl/3.6.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/registration/testpkg/index.json":
			metadataCallCount++
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/testpkg/page.json",
						"lower": "1.0.0",
						"upper": "1.0.0",
						"count": 1,
						"items": []map[string]any{
							{
								"@id": "http://" + r.Host + "/registration/testpkg/1.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/testpkg/1.0.0.json",
									"id":      "TestPkg",
									"version": "1.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()

	// Create cache context with NoCache = true
	cacheCtx := cache.NewSourceCacheContext()
	cacheCtx.NoCache = true

	// First call with NoCache
	_, err = repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	firstMetadataCallCount := metadataCallCount

	// Second call with NoCache - should bypass cache and hit server again
	_, err = repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() second call error = %v", err)
	}

	// Verify server was called again (cache bypassed)
	if metadataCallCount == firstMetadataCallCount {
		t.Error("NoCache flag not respected: no additional server call made")
	}
}

func TestV3Provider_GetMetadata_NilCacheContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl/3.6.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/registration/testpkg/index.json":
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/testpkg/page.json",
						"lower": "1.0.0",
						"upper": "1.0.0",
						"count": 1,
						"items": []map[string]any{
							{
								"@id": "http://" + r.Host + "/registration/testpkg/1.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/testpkg/1.0.0.json",
									"id":      "TestPkg",
									"version": "1.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()

	// Pass nil for SourceCacheContext - should use default (caching enabled)
	metadata, err := repo.GetMetadata(ctx, nil, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() with nil cache context error = %v", err)
	}
	if metadata == nil {
		t.Fatal("GetMetadata() returned nil metadata")
	}
}

func TestV3Provider_DownloadPackage_CacheHit(t *testing.T) {
	var downloadCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/packages/",
						"@type": "PackageBaseAddress/3.0.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/packages/testpkg/1.0.0/testpkg.1.0.0.nupkg":
			downloadCount++
			w.Header().Set("Content-Type", "application/zip")
			// Write valid ZIP header (PK signature)
			_, _ = w.Write([]byte{0x50, 0x4B, 0x03, 0x04})
			_, _ = w.Write([]byte("fake package content"))
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()
	cacheCtx := cache.NewSourceCacheContext()

	// First download - should hit server
	reader1, err := repo.DownloadPackage(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() first call error = %v", err)
	}
	defer func() { _ = reader1.Close() }()

	if downloadCount != 1 {
		t.Errorf("Expected 1 download, got %d", downloadCount)
	}

	// Second download - should hit cache
	reader2, err := repo.DownloadPackage(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() second call error = %v", err)
	}
	defer func() { _ = reader2.Close() }()

	// Verify no additional download (cache hit)
	if downloadCount != 1 {
		t.Errorf("Cache miss: downloadCount = %d, want 1", downloadCount)
	}
}

func TestV3Provider_DownloadPackage_DirectDownload(t *testing.T) {
	var downloadCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/packages/",
						"@type": "PackageBaseAddress/3.0.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/packages/testpkg/1.0.0/testpkg.1.0.0.nupkg":
			downloadCount++
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write([]byte{0x50, 0x4B, 0x03, 0x04})
			_, _ = w.Write([]byte("fake package content"))
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()

	// Create cache context with DirectDownload = true (read from cache, don't write)
	cacheCtx := cache.NewSourceCacheContext()
	cacheCtx.DirectDownload = true

	// First download with DirectDownload - should hit server but not cache result
	reader1, err := repo.DownloadPackage(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer func() { _ = reader1.Close() }()

	// Second download with DirectDownload - should hit server again (not cached)
	reader2, err := repo.DownloadPackage(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() second call error = %v", err)
	}
	defer func() { _ = reader2.Close() }()

	// Verify server was hit both times (no caching with DirectDownload)
	if downloadCount != 2 {
		t.Errorf("DirectDownload not respected: downloadCount = %d, want 2", downloadCount)
	}
}

func TestV3Provider_ListVersions_CacheHit(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl/3.6.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/registration/testpkg/index.json":
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"items": []map[string]any{
							{"catalogEntry": map[string]any{"version": "1.0.0"}},
							{"catalogEntry": map[string]any{"version": "2.0.0"}},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()
	cacheCtx := cache.NewSourceCacheContext()

	// First call - cache miss
	versions1, err := repo.ListVersions(ctx, cacheCtx, "TestPkg")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if len(versions1) != 2 {
		t.Errorf("ListVersions() returned %d versions, want 2", len(versions1))
	}

	initialCallCount := callCount

	// Second call - cache hit
	versions2, err := repo.ListVersions(ctx, cacheCtx, "TestPkg")
	if err != nil {
		t.Fatalf("ListVersions() second call error = %v", err)
	}
	if len(versions2) != len(versions1) {
		t.Errorf("Cached versions count = %d, want %d", len(versions2), len(versions1))
	}

	// Verify no additional server calls (cache hit)
	if callCount != initialCallCount {
		t.Errorf("Cache miss: callCount increased from %d to %d", initialCallCount, callCount)
	}
}

func TestV3Provider_NoCacheConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl/3.6.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/registration/testpkg/index.json":
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/testpkg/page.json",
						"lower": "1.0.0",
						"upper": "1.0.0",
						"count": 1,
						"items": []map[string]any{
							{
								"@id": "http://" + r.Host + "/registration/testpkg/1.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/testpkg/1.0.0.json",
									"id":      "TestPkg",
									"version": "1.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create repository WITHOUT cache (nil)
	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      nil, // No cache
	})

	ctx := context.Background()

	// Should work without cache
	metadata, err := repo.GetMetadata(ctx, nil, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() without cache error = %v", err)
	}
	if metadata == nil {
		t.Fatal("GetMetadata() returned nil metadata")
	}
}

func TestV3Provider_CacheMaxAge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl/3.6.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
		case "/registration/testpkg/index.json":
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/testpkg/page.json",
						"lower": "1.0.0",
						"upper": "1.0.0",
						"count": 1,
						"items": []map[string]any{
							{
								"@id": "http://" + r.Host + "/registration/testpkg/1.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/testpkg/1.0.0.json",
									"id":      "TestPkg",
									"version": "1.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()

	// Use very short MaxAge
	cacheCtx := cache.NewSourceCacheContext()
	cacheCtx.MaxAge = 1 * time.Millisecond

	// First call
	_, err = repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	// Second call - cache should be expired
	// This test just verifies the MaxAge parameter is accepted and used
	_, err = repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() after expiry error = %v", err)
	}
}

// V2 Provider Cache Integration Tests

func TestV2Provider_GetMetadata_CacheHit(t *testing.T) {
	var metadataCallCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			// Return 404 for V3 index check - triggers V2 detection
			http.NotFound(w, r)
		case "/", "":
			// V2 service document
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<service xmlns="http://www.w3.org/2007/app">
  <workspace>
    <collection href="Packages">
      <title>Packages</title>
    </collection>
  </workspace>
</service>`))
		case "/Packages(Id='TestPkg',Version='1.0.0')":
			metadataCallCount++
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<entry xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <id>http://` + r.Host + `/Packages(Id='TestPkg',Version='1.0.0')</id>
  <title>TestPkg</title>
  <content type="application/zip" src="http://` + r.Host + `/package/TestPkg/1.0.0"/>
  <m:properties>
    <d:Id>TestPkg</d:Id>
    <d:Version>1.0.0</d:Version>
  </m:properties>
</entry>`))
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()
	cacheCtx := cache.NewSourceCacheContext()

	// First call - should miss cache and hit server
	metadata1, err := repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() first call error = %v", err)
	}
	if metadata1 == nil {
		t.Fatal("GetMetadata() returned nil metadata")
	}

	initialMetadataCallCount := metadataCallCount

	// Second call - should hit cache
	metadata2, err := repo.GetMetadata(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() second call error = %v", err)
	}
	if metadata2 == nil {
		t.Fatal("GetMetadata() returned nil metadata on cache hit")
	}

	// Verify no additional server calls (cache hit)
	if metadataCallCount != initialMetadataCallCount {
		t.Errorf("Cache miss detected: metadataCallCount increased from %d to %d", initialMetadataCallCount, metadataCallCount)
	}
}

func TestV2Provider_DownloadPackage_CacheHit(t *testing.T) {
	var downloadCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			// Return 404 for V3 index check - triggers V2 detection
			http.NotFound(w, r)
		case "/", "":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<service xmlns="http://www.w3.org/2007/app">
  <workspace>
    <collection href="Packages">
      <title>Packages</title>
    </collection>
  </workspace>
</service>`))
		case "/package/TestPkg/1.0.0":
			downloadCount++
			w.Header().Set("Content-Type", "application/zip")
			// Write valid ZIP header
			_, _ = w.Write([]byte{0x50, 0x4B, 0x03, 0x04})
			_, _ = w.Write([]byte("fake package content"))
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()
	cacheCtx := cache.NewSourceCacheContext()

	// First download - should hit server
	reader1, err := repo.DownloadPackage(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() first call error = %v", err)
	}
	defer func() { _ = reader1.Close() }()

	if downloadCount != 1 {
		t.Errorf("Expected 1 download, got %d", downloadCount)
	}

	// Second download - should hit cache
	reader2, err := repo.DownloadPackage(ctx, cacheCtx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() second call error = %v", err)
	}
	defer func() { _ = reader2.Close() }()

	// Verify no additional download (cache hit)
	if downloadCount != 1 {
		t.Errorf("Cache miss: downloadCount = %d, want 1", downloadCount)
	}
}

func TestV2Provider_ListVersions_CacheHit(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			// Return 404 for V3 index check - triggers V2 detection
			http.NotFound(w, r)
			return
		case "/", "":
			callCount++
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<service xmlns="http://www.w3.org/2007/app">
  <workspace>
    <collection href="Packages">
      <title>Packages</title>
    </collection>
  </workspace>
</service>`))
		case "/FindPackagesById()":
			callCount++
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <m:properties>
      <d:Version>1.0.0</d:Version>
    </m:properties>
  </entry>
  <entry>
    <m:properties>
      <d:Version>2.0.0</d:Version>
    </m:properties>
  </entry>
</feed>`))
		}
	}))
	defer server.Close()

	// Create cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
	})

	ctx := context.Background()
	cacheCtx := cache.NewSourceCacheContext()

	// First call - cache miss
	versions1, err := repo.ListVersions(ctx, cacheCtx, "TestPkg")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if len(versions1) != 2 {
		t.Errorf("ListVersions() returned %d versions, want 2", len(versions1))
	}

	initialCallCount := callCount

	// Second call - cache hit
	versions2, err := repo.ListVersions(ctx, cacheCtx, "TestPkg")
	if err != nil {
		t.Fatalf("ListVersions() second call error = %v", err)
	}
	if len(versions2) != len(versions1) {
		t.Errorf("Cached versions count = %d, want %d", len(versions2), len(versions1))
	}

	// Verify no additional server calls (cache hit)
	if callCount != initialCallCount {
		t.Errorf("Cache miss: callCount increased from %d to %d", initialCallCount, callCount)
	}
}
