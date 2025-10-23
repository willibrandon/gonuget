package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	// HashLength is the number of bytes to use from the SHA256 hash.
	// Set to 20 to maintain backwards compatibility with SHA-1 length.
	HashLength = 20

	// BufferSize for file I/O operations (matches NuGet.Client).
	BufferSize = 8192

	// CacheFileExtension for final cache files.
	CacheFileExtension = ".dat"

	// NewFileExtension for temporary files during atomic write.
	NewFileExtension = ".dat-new"
)

// DiskCache provides persistent caching to disk.
type DiskCache struct {
	rootDir string
	maxSize int64
}

// NewDiskCache creates a new disk cache.
func NewDiskCache(rootDir string, maxSize int64) (*DiskCache, error) {
	// Create root directory if it doesn't exist
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	return &DiskCache{
		rootDir: rootDir,
		maxSize: maxSize,
	}, nil
}

// ComputeHash computes a hash for the given value.
// Matches NuGet.Client's CachingUtility.ComputeHash exactly.
func ComputeHash(value string, addIdentifiableCharacters bool) string {
	// Get trailing portion for readability
	trailing := value
	if len(value) > 32 {
		trailing = value[len(value)-32:]
	}

	// Compute SHA256 hash
	h := sha256.New()
	h.Write([]byte(value))
	hash := h.Sum(nil)

	// Truncate to first 20 bytes (SHA-1 length) for backwards compatibility
	hexHash := hex.EncodeToString(hash[:HashLength])

	if addIdentifiableCharacters {
		return hexHash + "$" + trailing
	}
	return hexHash
}

// RemoveInvalidFileNameChars replaces invalid filename characters with underscores.
// Matches NuGet.Client's CachingUtility.RemoveInvalidFileNameChars.
func RemoveInvalidFileNameChars(value string) string {
	// Get invalid characters for the current OS
	invalid := getInvalidFileNameChars()

	// Replace invalid chars with underscore
	var sb strings.Builder
	sb.Grow(len(value))

	for _, ch := range value {
		if containsRune(invalid, ch) {
			sb.WriteRune('_')
		} else {
			sb.WriteRune(ch)
		}
	}

	result := sb.String()

	// Collapse double underscores (run twice like NuGet.Client)
	result = strings.ReplaceAll(result, "__", "_")
	result = strings.ReplaceAll(result, "__", "_")

	return result
}

// getInvalidFileNameChars returns OS-specific invalid filename characters.
// This matches .NET's Path.GetInvalidFileNameChars() behavior exactly.
func getInvalidFileNameChars() []rune {
	// Platform-specific invalid characters to match NuGet.Client exactly
	switch filepath.Separator {
	case '/': // Unix-like systems (Linux, macOS)
		// On Unix, only / and \0 are invalid in filenames
		return []rune{'/', '\x00'}
	case '\\': // Windows
		// Windows invalid filename characters
		return []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00'}
	default:
		// Fallback to conservative set
		return []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00'}
	}
}

// containsRune checks if a rune slice contains a specific rune.
func containsRune(runes []rune, r rune) bool {
	return slices.Contains(runes, r)
}

// GetCachePath computes the cache file path for a source URL and cache key.
// Matches NuGet.Client's HttpCacheUtility.InitializeHttpCacheResult.
func (dc *DiskCache) GetCachePath(sourceURL string, cacheKey string) (cacheFile string, newFile string) {
	// Compute hash for source URL to create folder name
	baseFolderName := RemoveInvalidFileNameChars(ComputeHash(sourceURL, true))

	// Create file name from cache key
	baseFileName := RemoveInvalidFileNameChars(cacheKey) + CacheFileExtension

	// Build paths
	cacheFolder := filepath.Join(dc.rootDir, baseFolderName)
	cacheFile = filepath.Join(cacheFolder, baseFileName)
	newFile = cacheFile + "-new" // Matches NuGet.Client's "-new" suffix

	return cacheFile, newFile
}

// Get retrieves a cached file if it exists and is not expired.
// Returns (reader, true) if found and valid, (nil, false) otherwise.
func (dc *DiskCache) Get(sourceURL string, cacheKey string, maxAge time.Duration) (io.ReadCloser, bool, error) {
	cacheFile, _ := dc.GetCachePath(sourceURL, cacheKey)

	// Check if file exists and is not expired
	reader, valid := readCacheFile(maxAge, cacheFile)
	if reader == nil {
		return nil, false, nil
	}

	return reader, valid, nil
}

// readCacheFile reads a cache file if it's not expired.
// Matches NuGet.Client's CachingUtility.ReadCacheFile.
func readCacheFile(maxAge time.Duration, cacheFile string) (io.ReadCloser, bool) {
	fileInfo, err := os.Stat(cacheFile)
	if err != nil {
		return nil, false
	}

	// Check age
	age := time.Since(fileInfo.ModTime())
	if age >= maxAge {
		return nil, false
	}

	// Open file with appropriate sharing flags
	// FileShare.Read | FileShare.Delete matches NuGet.Client
	file, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, false
	}

	return file, true
}

// Set writes data to the cache using atomic two-phase update.
// Matches NuGet.Client's HttpCacheUtility.CreateCacheFileAsync.
func (dc *DiskCache) Set(sourceURL string, cacheKey string, data io.Reader, validate func(io.ReadSeeker) error) error {
	cacheFile, _ := dc.GetCachePath(sourceURL, cacheKey)

	// Create unique temp file for this operation to avoid collisions
	newFile := cacheFile + fmt.Sprintf("-new.%d", time.Now().UnixNano())

	// Get directory paths
	newFileDir := filepath.Dir(newFile)
	cacheFileDir := filepath.Dir(cacheFile)

	// Create new file directory
	if err := os.MkdirAll(newFileDir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	// Phase 1: Write to temporary file
	// Use O_RDWR to allow validation reading after writing
	tempFile, err := os.OpenFile(newFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = tempFile.Close() }()

	// Copy data to temp file
	if _, err := io.CopyBuffer(tempFile, data, make([]byte, BufferSize)); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Validate content if validator provided
	if validate != nil {
		if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek temp file: %w", err)
		}
		if err := validate(tempFile); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Close temp file before moving
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Phase 2: Atomic move to final location
	// This matches NuGet.Client's two-phase update pattern:
	// 1. Delete old file (if not already open)
	// 2. Move new file to cache location

	// Create cache file directory if different from new file directory
	if cacheFileDir != newFileDir {
		if err := os.MkdirAll(cacheFileDir, 0755); err != nil {
			return fmt.Errorf("create final cache directory: %w", err)
		}
	}

	// Try atomic rename first (works on Unix, may fail on Windows if destination exists)
	err = os.Rename(newFile, cacheFile)
	if err == nil {
		return nil // Success
	}

	// On Windows, rename fails if destination exists. Try remove then rename.
	// This is the NuGet.Client pattern but has a race condition window.
	// However, it's necessary for Windows compatibility.
	if fileExists(cacheFile) {
		// File exists - check if it's open
		if !isFileAlreadyOpen(cacheFile) {
			_ = os.Remove(cacheFile)
			// Retry rename after removal
			if err := os.Rename(newFile, cacheFile); err != nil {
				// If it still fails, check if another goroutine won the race
				if fileExists(cacheFile) {
					// Another goroutine completed successfully, clean up our temp file
					_ = os.Remove(newFile)
					return nil
				}
				// Neither us nor another goroutine succeeded - this is an error
				return fmt.Errorf("move cache file: %w", err)
			}
			return nil // Our retry succeeded
		}
		// File exists but is open - clean up temp and let the other writer finish
		_ = os.Remove(newFile)
		return nil
	}

	// File doesn't exist and rename failed - this is an error
	_ = os.Remove(newFile)
	return fmt.Errorf("rename failed and destination does not exist: %w", err)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isFileAlreadyOpen checks if a file is already open by another process.
// Matches NuGet.Client's CachingUtility.IsFileAlreadyOpen.
func isFileAlreadyOpen(filePath string) bool {
	// Try to open with exclusive access
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// Any other error means file is likely open
		return true
	}
	defer func() { _ = file.Close() }()

	return false
}

// Delete removes a cache entry.
func (dc *DiskCache) Delete(sourceURL string, cacheKey string) error {
	cacheFile, newFile := dc.GetCachePath(sourceURL, cacheKey)

	// Remove both cache file and temp file if they exist
	_ = os.Remove(cacheFile)
	_ = os.Remove(newFile)

	return nil
}

// Clear removes all cache entries.
func (dc *DiskCache) Clear() error {
	return os.RemoveAll(dc.rootDir)
}
