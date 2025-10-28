package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/willibrandon/gonuget/cache"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/restore"
)

// Helper functions for cache file parity tests

// loadProject loads a .NET project file.
func loadProject(projectPath string) (*project.Project, error) {
	return project.LoadProject(projectPath)
}

// calculateDgSpecHash calculates the dgSpecHash for a project.
func calculateDgSpecHash(proj *project.Project) (string, error) {
	return restore.CalculateDgSpecHash(proj)
}

// isCacheValid checks if a cache file is valid.
func isCacheValid(cachePath, currentHash string) (bool, *restore.CacheFile, error) {
	cache, err := restore.LoadCacheFile(cachePath)
	if err != nil {
		return false, nil, err
	}

	valid := cache.IsValid() && cache.DgSpecHash == currentHash
	return valid, cache, nil
}

// ComputeCacheHashHandler computes a cache hash for a given value.
// This validates that gonuget's hash algorithm matches NuGet.Client's CachingUtility.ComputeHash().
type ComputeCacheHashHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ComputeCacheHashHandler) ErrorCode() string { return "CACHE_HASH_001" }

// Handle processes the request.
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

// ErrorCode returns the error code for this handler.
func (h *SanitizeCacheFilenameHandler) ErrorCode() string { return "CACHE_SANITIZE_001" }

// Handle processes the request.
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

// ErrorCode returns the error code for this handler.
func (h *GenerateCachePathsHandler) ErrorCode() string { return "CACHE_PATHS_001" }

// Handle processes the request.
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

// ValidateCacheFileHandler validates TTL expiration logic for a cache file.
// This validates that gonuget matches NuGet.Client's CachingUtility.ReadCacheFile() TTL behavior.
type ValidateCacheFileHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ValidateCacheFileHandler) ErrorCode() string { return "CACHE_VALIDATE_001" }

// Handle processes the request.
func (h *ValidateCacheFileHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ValidateCacheFileRequest
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

	// Try to get the file with the specified max age
	maxAge := time.Duration(req.MaxAgeSeconds) * time.Second
	reader, valid, err := dc.Get(req.SourceURL, req.CacheKey, maxAge)
	if err != nil {
		return nil, fmt.Errorf("validate cache file: %w", err)
	}

	// Close reader if we got one
	if reader != nil {
		_ = reader.Close()
	}

	return ValidateCacheFileResponse{
		Valid: valid,
	}, nil
}

// CalculateDgSpecHashHandler calculates the dgSpecHash for a project.
// This validates that gonuget's dgSpecHash matches NuGet.Client's DependencyGraphSpec.GetHash().
type CalculateDgSpecHashHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *CalculateDgSpecHashHandler) ErrorCode() string { return "CACHE_DGSPEC_001" }

// Handle processes the request.
func (h *CalculateDgSpecHashHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req CalculateDgSpecHashRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.ProjectPath == "" {
		return nil, fmt.Errorf("projectPath is required")
	}

	// Load project file
	proj, err := loadProject(req.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}

	// Calculate dgSpecHash
	hash, err := calculateDgSpecHash(proj)
	if err != nil {
		return nil, fmt.Errorf("calculate dgSpecHash: %w", err)
	}

	return CalculateDgSpecHashResponse{
		Hash: hash,
	}, nil
}

// VerifyProjectCacheFileHandler verifies a project.nuget.cache file.
// This validates that gonuget can read and validate cache files created by dotnet.
type VerifyProjectCacheFileHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *VerifyProjectCacheFileHandler) ErrorCode() string { return "CACHE_VERIFY_001" }

// Handle processes the request.
func (h *VerifyProjectCacheFileHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req VerifyProjectCacheFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.CachePath == "" {
		return nil, fmt.Errorf("cachePath is required")
	}
	if req.CurrentHash == "" {
		return nil, fmt.Errorf("currentHash is required")
	}

	// Load and validate cache file
	valid, cacheFile, err := isCacheValid(req.CachePath, req.CurrentHash)
	if err != nil {
		return nil, fmt.Errorf("validate cache file: %w", err)
	}

	// Build response
	resp := VerifyProjectCacheFileResponse{
		Valid:   valid,
		Version: 0,
		Success: false,
	}

	if cacheFile != nil {
		resp.Version = cacheFile.Version
		resp.DgSpecHash = cacheFile.DgSpecHash
		resp.Success = cacheFile.Success
		resp.ProjectFilePath = cacheFile.ProjectFilePath
		resp.ExpectedPackageFilesCount = len(cacheFile.ExpectedPackageFiles)
	}

	return resp, nil
}
