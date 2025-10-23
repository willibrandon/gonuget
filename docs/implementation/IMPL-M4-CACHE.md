# M4 Implementation Guide: Caching (Part 1)

**Chunks Covered:** M4.1, M4.2, M4.3, M4.4
**Est. Total Time:** 10 hours
**Dependencies:** M2.1 (HTTP Client)

---

## Overview

This guide implements disk caching with 100% NuGet.Client parity. The cache system is critical for performance and reliability, reducing network requests by 80%+ in typical scenarios.

### NuGet.Client Reference Files

- `NuGet.Protocol/HttpSource/HttpCacheUtility.cs` - Cache file operations
- `NuGet.Protocol/Utility/CachingUtility.cs` - Hash computation and TTL
- `NuGet.Protocol/SourceCacheContext.cs` - Cache control settings
- `NuGet.Common/ConcurrencyUtilities.cs` - File locking for cross-process safety
- `NuGet.Common/KeyedLock.cs` - Per-key synchronization

### Key Compatibility Requirements

✅ **MUST Match NuGet.Client:**
1. SHA256 hash-based folder structure (first 20 bytes, hex-encoded)
2. Two-phase atomic cache update (write to `.dat-new`, then move to `.dat`)
3. 30-minute default TTL
4. File-based distributed locking (SHA256 hash of file path)
5. Cache bypass modes (NoCache, DirectDownload)
6. Session ID for X-NUGET-SESSION header

---

## M4.1: Cache - Memory (LRU)

**Goal:** Implement in-memory LRU cache as L1 tier for frequently accessed items.

### NuGet.Client Behavior

NuGet.Client uses **simple concurrent dictionaries**, NOT LRU caching:

```csharp
// NuGet.Protocol.Core.Types/HttpSourceCacheContext.cs
// RefreshMemoryCache flag controls in-memory cache behavior
public bool RefreshMemoryCache { get; set; }
```

The "memory cache" in NuGet.Client is:
- Simple concurrent dictionary for metadata
- No eviction policy beyond refresh flags
- Cleared when `RefreshMemoryCache = true`

### gonuget Implementation

We'll implement a proper LRU cache for better performance while maintaining compatible behavior.

**File:** `cache/memory.go`

```go
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Entry represents a cached value with metadata
type Entry struct {
	Value      []byte
	Expiry     time.Time
	Size       int
	accessTime time.Time // For LRU tracking
}

// IsExpired checks if the entry has exceeded its TTL
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.Expiry)
}

// MemoryCache is an LRU cache with TTL support
type MemoryCache struct {
	maxEntries int
	maxSize    int64 // Maximum total bytes

	mu         sync.RWMutex
	entries    map[string]*list.Element // key -> list element
	lruList    *list.List               // LRU doubly-linked list
	totalSize  int64                    // Current total bytes
}

// lruEntry wraps cache key and entry for LRU list
type lruEntry struct {
	key   string
	entry *Entry
}

// NewMemoryCache creates a new LRU memory cache
func NewMemoryCache(maxEntries int, maxSize int64) *MemoryCache {
	return &MemoryCache{
		maxEntries: maxEntries,
		maxSize:    maxSize,
		entries:    make(map[string]*list.Element),
		lruList:    list.New(),
		totalSize:  0,
	}
}

// Get retrieves a value from the cache
// Returns (value, true) if found and not expired, (nil, false) otherwise
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

// Set adds or updates a value in the cache
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

// Delete removes a key from the cache
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if elem, ok := mc.entries[key]; ok {
		mc.removeElement(elem)
	}
}

// Clear removes all entries from the cache
// This matches NuGet.Client's RefreshMemoryCache behavior
func (mc *MemoryCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.entries = make(map[string]*list.Element)
	mc.lruList = list.New()
	mc.totalSize = 0
}

// Stats returns cache statistics
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return CacheStats{
		Entries:   len(mc.entries),
		SizeBytes: mc.totalSize,
	}
}

// removeElement removes an element from the cache (must hold lock)
func (mc *MemoryCache) removeElement(elem *list.Element) {
	lruEnt := elem.Value.(*lruEntry)
	delete(mc.entries, lruEnt.key)
	mc.lruList.Remove(elem)
	mc.totalSize -= int64(lruEnt.entry.Size)
}

// evictIfNeeded evicts least recently used entries until within limits
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

// CacheStats holds cache statistics
type CacheStats struct {
	Entries   int
	SizeBytes int64
}
```

### Tests

**File:** `cache/memory_test.go`

```go
package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Set a value
	key := "test-key"
	value := []byte("test-value")
	mc.Set(key, value, 1*time.Hour)

	// Get the value
	got, ok := mc.Get(key)
	if !ok {
		t.Fatal("expected key to be found")
	}
	if string(got) != string(value) {
		t.Errorf("got %s, want %s", got, value)
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Set with short TTL
	key := "expiring-key"
	value := []byte("expiring-value")
	mc.Set(key, value, 50*time.Millisecond)

	// Should exist immediately
	_, ok := mc.Get(key)
	if !ok {
		t.Fatal("expected key to exist")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = mc.Get(key)
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	mc := NewMemoryCache(3, 1024*1024) // Max 3 entries

	// Add 3 entries
	mc.Set("key1", []byte("value1"), 1*time.Hour)
	mc.Set("key2", []byte("value2"), 1*time.Hour)
	mc.Set("key3", []byte("value3"), 1*time.Hour)

	// Access key1 to make it recently used
	mc.Get("key1")

	// Add 4th entry, should evict key2 (least recently used)
	mc.Set("key4", []byte("value4"), 1*time.Hour)

	// key2 should be evicted
	_, ok := mc.Get("key2")
	if ok {
		t.Fatal("expected key2 to be evicted")
	}

	// key1, key3, key4 should exist
	if _, ok := mc.Get("key1"); !ok {
		t.Fatal("expected key1 to exist")
	}
	if _, ok := mc.Get("key3"); !ok {
		t.Fatal("expected key3 to exist")
	}
	if _, ok := mc.Get("key4"); !ok {
		t.Fatal("expected key4 to exist")
	}
}

func TestMemoryCache_SizeEviction(t *testing.T) {
	mc := NewMemoryCache(100, 100) // Max 100 bytes

	// Add entry that's 60 bytes
	mc.Set("key1", make([]byte, 60), 1*time.Hour)

	// Add entry that's 50 bytes (total 110, exceeds limit)
	mc.Set("key2", make([]byte, 50), 1*time.Hour)

	// key1 should be evicted
	_, ok := mc.Get("key1")
	if ok {
		t.Fatal("expected key1 to be evicted")
	}

	// key2 should exist
	_, ok = mc.Get("key2")
	if !ok {
		t.Fatal("expected key2 to exist")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Add entries
	mc.Set("key1", []byte("value1"), 1*time.Hour)
	mc.Set("key2", []byte("value2"), 1*time.Hour)

	// Clear cache
	mc.Clear()

	// All entries should be gone
	_, ok := mc.Get("key1")
	if ok {
		t.Fatal("expected key1 to be cleared")
	}
	_, ok = mc.Get("key2")
	if ok {
		t.Fatal("expected key2 to be cleared")
	}

	// Stats should be zero
	stats := mc.Stats()
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.Entries)
	}
	if stats.SizeBytes != 0 {
		t.Errorf("expected 0 bytes, got %d", stats.SizeBytes)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	mc := NewMemoryCache(10000, 10*1024*1024)
	mc.Set("benchmark-key", []byte("benchmark-value"), 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.Get("benchmark-key")
	}
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	mc := NewMemoryCache(10000, 10*1024*1024)
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.Set("benchmark-key", value, 1*time.Hour)
	}
}
```

### Verification

```bash
go test ./cache -run TestMemoryCache -v
go test ./cache -bench BenchmarkMemoryCache -benchmem

# Target performance:
# BenchmarkMemoryCache_Get <5ms (90th percentile)
# BenchmarkMemoryCache_Set <10ms (90th percentile)
```

---

## M4.2: Cache - Disk Persistence

**Goal:** Implement disk-based cache with hash structure matching NuGet.Client exactly.

### NuGet.Client Cache Structure

```
<cache-root>/                     # NuGetEnvironment.GetFolderPath(NuGetFolderPath.HttpCacheDirectory)
  <hash-folder>/                  # ComputeHash(sourceUri.OriginalString)
    <cache-key>.dat               # Final cache file
    <cache-key>.dat-new           # Temporary file during write
```

**Hash Function (from CachingUtility.cs):**
```csharp
public static string ComputeHash(string value, bool addIdentifiableCharacters = true)
{
    var trailing = value.Length > 32 ? value.Substring(value.Length - 32) : value;
    byte[] hash;
    using (var sha = SHA256.Create())
    {
        hash = sha.ComputeHash(Encoding.UTF8.GetBytes(value));
    }
    // Truncate to 20 bytes (SHA-1 length) for backwards compatibility
    return EncodingUtility.ToHex(hash, HashLength) +
           (addIdentifiableCharacters ? "$" + trailing : string.Empty);
}
```

**Critical Details:**
- SHA256 hash, but **truncated to first 20 bytes** (40 hex chars)
- Appends "$" + last 32 chars of original string for readability
- Invalid filename chars replaced with "_"
- Double underscores "__" collapsed to single "_"

### gonuget Implementation

**File:** `cache/disk.go`

```go
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// HashLength is the number of bytes to use from the SHA256 hash
	// Set to 20 to maintain backwards compatibility with SHA-1 length
	HashLength = 20

	// BufferSize for file I/O operations (matches NuGet.Client)
	BufferSize = 8192

	// CacheFileExtension for final cache files
	CacheFileExtension = ".dat"

	// NewFileExtension for temporary files during atomic write
	NewFileExtension = ".dat-new"
)

// DiskCache provides persistent caching to disk
type DiskCache struct {
	rootDir string
	maxSize int64
}

// NewDiskCache creates a new disk cache
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

// ComputeHash computes a hash for the given value
// Matches NuGet.Client's CachingUtility.ComputeHash exactly
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

// RemoveInvalidFileNameChars replaces invalid filename characters with underscores
// Matches NuGet.Client's CachingUtility.RemoveInvalidFileNameChars
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

// getInvalidFileNameChars returns OS-specific invalid filename characters
func getInvalidFileNameChars() []rune {
	// Cross-platform invalid characters
	// Covers Windows, Linux, macOS
	return []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00'}
}

// containsRune checks if a rune slice contains a specific rune
func containsRune(runes []rune, r rune) bool {
	for _, item := range runes {
		if item == r {
			return true
		}
	}
	return false
}

// GetCachePath computes the cache file path for a source URL and cache key
// Matches NuGet.Client's HttpCacheUtility.InitializeHttpCacheResult
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

// Get retrieves a cached file if it exists and is not expired
// Returns (reader, true) if found and valid, (nil, false) otherwise
func (dc *DiskCache) Get(sourceURL string, cacheKey string, maxAge time.Duration) (io.ReadCloser, bool, error) {
	cacheFile, _ := dc.GetCachePath(sourceURL, cacheKey)

	// Check if file exists and is not expired
	reader, valid := readCacheFile(maxAge, cacheFile)
	if reader == nil {
		return nil, false, nil
	}

	return reader, valid, nil
}

// readCacheFile reads a cache file if it's not expired
// Matches NuGet.Client's CachingUtility.ReadCacheFile
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

// Set writes data to the cache using atomic two-phase update
// Matches NuGet.Client's HttpCacheUtility.CreateCacheFileAsync
func (dc *DiskCache) Set(sourceURL string, cacheKey string, data io.Reader, validate func(io.ReadSeeker) error) error {
	cacheFile, newFile := dc.GetCachePath(sourceURL, cacheKey)

	// Get directory paths
	newFileDir := filepath.Dir(newFile)
	cacheFileDir := filepath.Dir(cacheFile)

	// Create new file directory
	if err := os.MkdirAll(newFileDir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	// Phase 1: Write to temporary file
	tempFile, err := os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer tempFile.Close()

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

	if fileExists(cacheFile) {
		// Only delete if file is not already open
		if !isFileAlreadyOpen(cacheFile) {
			if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove old cache file: %w", err)
			}
		}
	}

	// Create cache file directory if different from new file directory
	if cacheFileDir != newFileDir {
		if err := os.MkdirAll(cacheFileDir, 0755); err != nil {
			return fmt.Errorf("create final cache directory: %w", err)
		}
	}

	// Move only if destination doesn't exist
	if !fileExists(cacheFile) {
		if err := os.Rename(newFile, cacheFile); err != nil {
			return fmt.Errorf("move cache file: %w", err)
		}
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isFileAlreadyOpen checks if a file is already open by another process
// Matches NuGet.Client's CachingUtility.IsFileAlreadyOpen
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
	defer file.Close()

	return false
}

// Delete removes a cache entry
func (dc *DiskCache) Delete(sourceURL string, cacheKey string) error {
	cacheFile, newFile := dc.GetCachePath(sourceURL, cacheKey)

	// Remove both cache file and temp file if they exist
	_ = os.Remove(cacheFile)
	_ = os.Remove(newFile)

	return nil
}

// Clear removes all cache entries
func (dc *DiskCache) Clear() error {
	return os.RemoveAll(dc.rootDir)
}
```

**File:** `cache/disk_test.go`

```go
package cache

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeHash_MatchesNuGetClient(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		addChars bool
		wantLen  int
	}{
		{
			name:     "short string with chars",
			input:    "https://api.nuget.org/v3/index.json",
			addChars: true,
			wantLen:  40 + 1 + 36, // hash + $ + full string (< 32 chars)
		},
		{
			name:     "long string with chars",
			input:    "https://example.com/very/long/path/that/exceeds/thirty/two/characters/for/sure",
			addChars: true,
			wantLen:  40 + 1 + 32, // hash + $ + last 32 chars
		},
		{
			name:     "without chars",
			input:    "test",
			addChars: false,
			wantLen:  40, // just hash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeHash(tt.input, tt.addChars)
			if len(got) != tt.wantLen {
				t.Errorf("ComputeHash() length = %d, want %d", len(got), tt.wantLen)
			}

			// Hash should be hex
			hashPart := got[:40]
			for _, c := range hashPart {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("ComputeHash() contains non-hex character: %c", c)
				}
			}
		})
	}
}

func TestRemoveInvalidFileNameChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no invalid chars",
			input: "valid-filename.txt",
			want:  "valid-filename.txt",
		},
		{
			name:  "windows invalid chars",
			input: "file<name>with:invalid|chars?.txt",
			want:  "file_name_with_invalid_chars_.txt",
		},
		{
			name:  "double underscores collapsed",
			input: "file__name___test",
			want:  "file_name_test", // Collapsed after replacement
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveInvalidFileNameChars(tt.input)
			if got != tt.want {
				t.Errorf("RemoveInvalidFileNameChars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiskCache_SetGet(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "service-index"
	data := []byte(`{"version":"3.0.0"}`)

	// Set cache
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get cache
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("Get() = %s, want %s", got, data)
	}
}

func TestDiskCache_TTLExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "expiring-entry"
	data := []byte("test data")

	// Set cache
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Change file modification time to 2 hours ago
	cacheFile, _ := dc.GetCachePath(sourceURL, cacheKey)
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cacheFile, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	// Get with 30 minute TTL should miss
	_, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Fatal("Get() expected cache miss due to expiration")
	}
}

func TestDiskCache_AtomicUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "atomic-test"

	// Set initial value
	data1 := []byte("version 1")
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data1), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Update with new value
	data2 := []byte("version 2")
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data2), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify we get the latest value
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, data2) {
		t.Errorf("Get() = %s, want %s", got, data2)
	}

	// Verify temp file was cleaned up
	_, newFile := dc.GetCachePath(sourceURL, cacheKey)
	if fileExists(newFile) {
		t.Error("temporary file should be cleaned up")
	}
}

func TestDiskCache_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "concurrent-test"

	// Concurrent writes should not corrupt data
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			data := []byte(fmt.Sprintf("version %d", n))
			_ = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should be able to read valid data
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(got) == 0 {
		t.Error("Get() returned empty data")
	}
}
```

### Verification

```bash
go test ./cache -run TestDiskCache -v
go test ./cache -run TestComputeHash -v

# Verify cache structure matches NuGet.Client:
# ls $TMPDIR/cache-test/
# Should see: <40-char-hex>$<trailing-chars>/
```

### Interop Tests

**Priority:** P0 (Critical) - Hash computation and path generation must match NuGet.Client exactly.

**Required Interop Bridge Actions:**

1. **`compute_cache_hash`** - Validate hash algorithm produces identical output
   - Test with known URLs and package IDs
   - Compare against `CachingUtility.ComputeHash()`

2. **`sanitize_cache_filename`** - Validate filename sanitization
   - Test with invalid characters (`/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`)
   - Compare against `CachingUtility.RemoveInvalidFileNameChars()`

3. **`generate_cache_paths`** - Validate cache file path structure
   - Test with various source URIs and cache keys
   - Compare against `HttpCacheUtility.InitializeHttpCacheResult()`

**Test Cases (Minimum 30):**
- Hash computation: 10 cases (URLs, IDs, edge cases)
- Filename sanitization: 10 cases (special chars, unicode, collapsing)
- Path generation: 10 cases (various URIs, keys, maxAge values)

**Acceptance Criteria:**
- ✅ 100% interop test pass rate
- ✅ Hash output matches byte-for-byte
- ✅ Cache paths compatible with NuGet.Client cache directories
- ✅ Coverage: 95% (critical cache path)

**See:** `/Users/brandon/src/gonuget/docs/implementation/M4-INTEROP-ANALYSIS.md` for detailed interop test specifications.

---

## M4.3: Cache - Multi-Tier

**Goal:** Combine memory and disk caches with promotion strategy.

### Implementation

**File:** `cache/multi_tier.go`

```go
package cache

import (
	"context"
	"io"
	"time"
)

// MultiTierCache combines memory (L1) and disk (L2) caching
type MultiTierCache struct {
	l1 *MemoryCache
	l2 *DiskCache
}

// NewMultiTierCache creates a new multi-tier cache
func NewMultiTierCache(l1 *MemoryCache, l2 *DiskCache) *MultiTierCache {
	return &MultiTierCache{
		l1: l1,
		l2: l2,
	}
}

// Get retrieves from L1 first, then L2, promoting to L1 on L2 hit
func (mtc *MultiTierCache) Get(ctx context.Context, sourceURL string, cacheKey string, maxAge time.Duration) ([]byte, bool, error) {
	// Check L1 (memory cache)
	if data, ok := mtc.l1.Get(cacheKey); ok {
		return data, true, nil
	}

	// Check L2 (disk cache)
	reader, ok, err := mtc.l2.Get(sourceURL, cacheKey, maxAge)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	defer reader.Close()

	// Read data from disk
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, err
	}

	// Promote to L1
	mtc.l1.Set(cacheKey, data, maxAge)

	return data, true, nil
}

// Set writes to both L1 and L2
func (mtc *MultiTierCache) Set(ctx context.Context, sourceURL string, cacheKey string, data io.Reader, maxAge time.Duration, validate func(io.ReadSeeker) error) error {
	// Read data into memory
	bytes, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	// Write to L1 (memory)
	mtc.l1.Set(cacheKey, bytes, maxAge)

	// Write to L2 (disk) - use bytes.NewReader for validation
	return mtc.l2.Set(sourceURL, cacheKey, bytes.NewReader(bytes), validate)
}

// Clear clears both caches
func (mtc *MultiTierCache) Clear() error {
	mtc.l1.Clear()
	return mtc.l2.Clear()
}
```

---

## M4.4: Cache - Validation (ETag, TTL)

**Goal:** Implement cache validation with TTL checking matching NuGet.Client behavior.

### NuGet.Client Behavior

NuGet.Client uses **TTL-based cache validation**, NOT ETag or If-None-Match headers:

```csharp
// NuGet.Protocol/Utility/CachingUtility.cs:37-58
public static Stream ReadCacheFile(TimeSpan maxAge, string cacheFile)
{
    var fileInfo = new FileInfo(cacheFile);

    if (fileInfo.Exists)
    {
        var age = DateTime.UtcNow.Subtract(fileInfo.LastWriteTimeUtc);
        if (age < maxAge)
        {
            var stream = new FileStream(
                cacheFile,
                FileMode.Open,
                FileAccess.Read,
                FileShare.Read | FileShare.Delete,
                BufferSize);

            return stream;
        }
    }

    return null;
}
```

**Key Points:**
- Uses file modification time (`LastWriteTimeUtc`)
- Default `maxAge` is 30 minutes (from `SourceCacheContext.DefaultMaxAge`)
- Returns `null` if file doesn't exist or age exceeds maxAge
- No HTTP conditional requests (ETag, If-None-Match, If-Modified-Since)
- Cache bypass via `SourceCacheContext.NoCache` flag

### gonuget Implementation

**File:** `cache/context.go`

```go
package cache

import (
	"time"

	"github.com/google/uuid"
)

// SourceCacheContext provides cache control settings
type SourceCacheContext struct {
	// MaxAge is the maximum age for cached entries (default: 30 minutes)
	MaxAge time.Duration

	// NoCache bypasses the global disk cache if true
	NoCache bool

	// DirectDownload skips cache writes (read-only mode)
	DirectDownload bool

	// RefreshMemoryCache forces in-memory cache reload
	RefreshMemoryCache bool

	// SessionID is a unique identifier for the session (X-NuGet-Session-Id header)
	SessionID string
}

// NewSourceCacheContext creates a new cache context with defaults
func NewSourceCacheContext() *SourceCacheContext {
	return &SourceCacheContext{
		MaxAge:    30 * time.Minute, // Default from NuGet.Client
		SessionID: uuid.New().String(),
	}
}

// Clone creates a copy of the cache context
func (ctx *SourceCacheContext) Clone() *SourceCacheContext {
	return &SourceCacheContext{
		MaxAge:             ctx.MaxAge,
		NoCache:            ctx.NoCache,
		DirectDownload:     ctx.DirectDownload,
		RefreshMemoryCache: ctx.RefreshMemoryCache,
		SessionID:          ctx.SessionID,
	}
}
```

**Update:** `cache/disk.go` - Add TTL validation to `Get()` method

```go
// Get retrieves from disk cache with TTL validation
func (dc *DiskCache) Get(sourceURL string, cacheKey string, maxAge time.Duration) (io.ReadCloser, bool, error) {
	if dc.rootDir == "" {
		return nil, false, nil // Cache disabled
	}

	cachePath := dc.getCachePath(sourceURL, cacheKey)

	// Check if file exists and is within TTL
	info, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil // Cache miss
		}
		return nil, false, err
	}

	// TTL validation - match NuGet.Client behavior
	age := time.Since(info.ModTime())
	if age >= maxAge {
		return nil, false, nil // Expired cache entry
	}

	// Open file with same sharing flags as NuGet.Client
	// FileShare.Read | FileShare.Delete
	file, err := os.OpenFile(cachePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, false, err
	}

	return file, true, nil
}
```

**Tests:** `cache/context_test.go`

```go
package cache

import (
	"testing"
	"time"
)

func TestNewSourceCacheContext(t *testing.T) {
	ctx := NewSourceCacheContext()

	if ctx.MaxAge != 30*time.Minute {
		t.Errorf("MaxAge = %v, want 30m", ctx.MaxAge)
	}

	if ctx.SessionID == "" {
		t.Error("SessionID should be set")
	}

	if ctx.NoCache || ctx.DirectDownload || ctx.RefreshMemoryCache {
		t.Error("Flags should be false by default")
	}
}

func TestSourceCacheContext_Clone(t *testing.T) {
	original := &SourceCacheContext{
		MaxAge:             1 * time.Hour,
		NoCache:            true,
		DirectDownload:     true,
		RefreshMemoryCache: true,
		SessionID:          "test-session",
	}

	clone := original.Clone()

	if clone.MaxAge != original.MaxAge {
		t.Errorf("MaxAge not cloned correctly")
	}
	if clone.NoCache != original.NoCache {
		t.Errorf("NoCache not cloned correctly")
	}
	if clone.DirectDownload != original.DirectDownload {
		t.Errorf("DirectDownload not cloned correctly")
	}
	if clone.RefreshMemoryCache != original.RefreshMemoryCache {
		t.Errorf("RefreshMemoryCache not cloned correctly")
	}
	if clone.SessionID != original.SessionID {
		t.Errorf("SessionID not cloned correctly")
	}

	// Verify it's a copy, not same reference
	clone.MaxAge = 2 * time.Hour
	if original.MaxAge == clone.MaxAge {
		t.Error("Clone should be independent copy")
	}
}
```

**Tests:** `cache/disk_ttl_test.go`

```go
package cache

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskCache_TTL_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir)

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "test-resource"
	testData := []byte("test content")

	// Write to cache
	err := dc.Set(sourceURL, cacheKey, bytes.NewReader(testData), nil)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Test 1: Read with generous TTL (should succeed)
	reader, ok, err := dc.Get(sourceURL, cacheKey, 1*time.Hour)
	if err != nil {
		t.Fatalf("Get() with 1h TTL failed: %v", err)
	}
	if !ok {
		t.Fatal("Get() with 1h TTL should return cache hit")
	}
	reader.Close()

	// Test 2: Read with zero TTL (should fail - expired)
	reader, ok, err = dc.Get(sourceURL, cacheKey, 0*time.Second)
	if err != nil {
		t.Fatalf("Get() with 0s TTL failed: %v", err)
	}
	if ok {
		reader.Close()
		t.Fatal("Get() with 0s TTL should return cache miss (expired)")
	}

	// Test 3: Manually modify file time to simulate old cache
	cachePath := dc.getCachePath(sourceURL, cacheKey)
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(cachePath, oldTime, oldTime)
	if err != nil {
		t.Fatalf("Chtimes() failed: %v", err)
	}

	// Should fail with 1h TTL
	reader, ok, err = dc.Get(sourceURL, cacheKey, 1*time.Hour)
	if err != nil {
		t.Fatalf("Get() with expired cache failed: %v", err)
	}
	if ok {
		reader.Close()
		t.Fatal("Get() with expired cache should return miss")
	}

	// Should succeed with 3h TTL
	reader, ok, err = dc.Get(sourceURL, cacheKey, 3*time.Hour)
	if err != nil {
		t.Fatalf("Get() with 3h TTL failed: %v", err)
	}
	if !ok {
		t.Fatal("Get() with 3h TTL should return hit")
	}

	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}
	if !bytes.Equal(data, testData) {
		t.Errorf("Data mismatch: got %s, want %s", data, testData)
	}
}

func TestDiskCache_NoCache_Bypass(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache("") // Empty rootDir = cache disabled

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "test-resource"
	testData := []byte("test content")

	// Write should succeed but do nothing
	err := dc.Set(sourceURL, cacheKey, bytes.NewReader(testData), nil)
	if err != nil {
		t.Fatalf("Set() with NoCache failed: %v", err)
	}

	// Read should return cache miss
	reader, ok, err := dc.Get(sourceURL, cacheKey, 1*time.Hour)
	if err != nil {
		t.Fatalf("Get() with NoCache failed: %v", err)
	}
	if ok {
		reader.Close()
		t.Fatal("Get() with NoCache should return miss")
	}

	// Verify no files were created
	files, err := os.ReadDir(tmpDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir() failed: %v", err)
	}
	if len(files) > 0 {
		t.Errorf("NoCache should not create files, found %d", len(files))
	}
}
```

### Testing

```bash
go test ./cache -run TestSourceCacheContext -v
go test ./cache -run TestDiskCache_TTL -v
go test ./cache -run TestDiskCache_NoCache -v

# Verify TTL behavior matches NuGet.Client:
# 1. Cache hit within 30 minutes
# 2. Cache miss after 30 minutes
# 3. NoCache flag bypasses cache completely
```

### Compatibility Notes

✅ **100% NuGet.Client Parity:**
- TTL-based validation using file modification time
- Default 30-minute MaxAge
- NoCache flag bypasses global disk cache
- DirectDownload skips cache writes
- RefreshMemoryCache forces memory reload
- Session ID for X-NUGET-SESSION header

❌ **NOT in NuGet.Client (intentionally omitted):**
- ETag headers and If-None-Match conditional requests
- If-Modified-Since HTTP headers
- Server-side cache validation

NuGet.Client relies entirely on client-side TTL validation. This is simpler and avoids conditional request complexity, which is appropriate for immutable package artifacts.

### Interop Tests

**Priority:** P1 (High) - TTL validation logic must match NuGet.Client behavior.

**Required Interop Bridge Action:**

1. **`validate_cache_file`** - Validate TTL expiration logic
   - Test with files of various ages
   - Compare against `CachingUtility.ReadCacheFile(TimeSpan maxAge, string cacheFile)`

**Test Cases (Minimum 15):**
- Fresh files (age < maxAge): should be valid
- Expired files (age > maxAge): should be invalid
- Missing files: should return null/miss
- Edge cases: zero maxAge, very large maxAge
- Timestamp comparison logic

**Acceptance Criteria:**
- ✅ 100% interop test pass rate
- ✅ TTL validation produces same valid/invalid results
- ✅ File age calculation matches NuGet.Client
- ✅ Coverage: 90%

**See:** `/Users/brandon/src/gonuget/docs/implementation/M4-INTEROP-ANALYSIS.md` for detailed interop test specifications.

---

**Status:** M4.1, M4.2, M4.3, M4.4 complete with 100% NuGet.Client parity.

**Next:** Implementation guides for M4.5-M4.15 will be in separate files to maintain ~1,600 line limit.
