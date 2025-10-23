package main

import (
	"encoding/json"
	"fmt"

	"github.com/willibrandon/gonuget/cache"
)

// ComputeCacheHashHandler computes a cache hash for a given value.
// This validates that gonuget's hash algorithm matches NuGet.Client's CachingUtility.ComputeHash().
type ComputeCacheHashHandler struct{}

func (h *ComputeCacheHashHandler) ErrorCode() string { return "CACHE_HASH_001" }

func (h *ComputeCacheHashHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ComputeCacheHashRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.Value == "" {
		return nil, fmt.Errorf("value is required")
	}

	// Compute hash using gonuget implementation
	hash := cache.ComputeHash(req.Value, req.AddIdentifiableCharacters)

	return ComputeCacheHashResponse{
		Hash: hash,
	}, nil
}

// SanitizeCacheFilenameHandler sanitizes a filename by removing invalid characters.
// This validates that gonuget matches NuGet.Client's CachingUtility.RemoveInvalidFileNameChars().
type SanitizeCacheFilenameHandler struct{}

func (h *SanitizeCacheFilenameHandler) ErrorCode() string { return "CACHE_SANITIZE_001" }

func (h *SanitizeCacheFilenameHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req SanitizeCacheFilenameRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.Value == "" {
		return nil, fmt.Errorf("value is required")
	}

	// Sanitize filename using gonuget implementation
	sanitized := cache.RemoveInvalidFileNameChars(req.Value)

	return SanitizeCacheFilenameResponse{
		Sanitized: sanitized,
	}, nil
}

// GenerateCachePathsHandler generates cache file paths for a source URL and cache key.
// This validates that gonuget matches NuGet.Client's HttpCacheUtility.InitializeHttpCacheResult().
type GenerateCachePathsHandler struct{}

func (h *GenerateCachePathsHandler) ErrorCode() string { return "CACHE_PATHS_001" }

func (h *GenerateCachePathsHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req GenerateCachePathsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.CacheDirectory == "" {
		return nil, fmt.Errorf("cacheDirectory is required")
	}
	if req.SourceURL == "" {
		return nil, fmt.Errorf("sourceURL is required")
	}
	if req.CacheKey == "" {
		return nil, fmt.Errorf("cacheKey is required")
	}

	// Create disk cache instance
	dc, err := cache.NewDiskCache(req.CacheDirectory, 1024*1024*100) // 100MB max
	if err != nil {
		return nil, fmt.Errorf("create disk cache: %w", err)
	}

	// Generate cache paths
	cacheFile, newFile := dc.GetCachePath(req.SourceURL, req.CacheKey)

	// Compute base folder name for comparison
	baseFolderName := cache.RemoveInvalidFileNameChars(cache.ComputeHash(req.SourceURL, true))

	return GenerateCachePathsResponse{
		BaseFolderName: baseFolderName,
		CacheFile:      cacheFile,
		NewFile:        newFile,
	}, nil
}
