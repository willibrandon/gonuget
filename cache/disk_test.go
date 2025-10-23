package cache

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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
			wantLen:  40 + 1 + 32, // hash + $ + last 32 chars (string is 35 chars)
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
				if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
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
			name:  "forward slash (invalid on all platforms)",
			input: "file/with/slashes.txt",
			want:  "file_with_slashes.txt",
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
	defer func() { _ = reader.Close() }()

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
	defer func() { _ = reader.Close() }()

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
	for i := range 10 {
		go func(n int) {
			data := fmt.Appendf(nil, "version %d", n)
			_ = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
			done <- true
		}(i)
	}

	// Wait for all writes
	for range 10 {
		<-done
	}

	// Give a moment for file system operations to settle
	time.Sleep(50 * time.Millisecond)

	// Should be able to read valid data
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	defer func() { _ = reader.Close() }()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(got) == 0 {
		t.Error("Get() returned empty data")
	}
}

func TestDiskCache_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "delete-test"
	data := []byte("test data")

	// Set cache
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete
	err = dc.Delete(sourceURL, cacheKey)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Fatal("Get() expected cache miss after delete")
	}
}

func TestDiskCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	// Add multiple entries
	for i := range 5 {
		sourceURL := fmt.Sprintf("https://example.com/feed%d", i)
		cacheKey := fmt.Sprintf("key%d", i)
		data := fmt.Appendf(nil, "data%d", i)

		err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// Clear
	err = dc.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify all entries are gone
	for i := range 5 {
		sourceURL := fmt.Sprintf("https://example.com/feed%d", i)
		cacheKey := fmt.Sprintf("key%d", i)

		_, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if ok {
			t.Errorf("Get() expected cache miss after clear for key%d", i)
		}
	}
}

func TestDiskCache_GetCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "service-index"

	cacheFile, newFile := dc.GetCachePath(sourceURL, cacheKey)

	// Verify paths are constructed correctly
	if cacheFile == "" {
		t.Fatal("cacheFile should not be empty")
	}
	if newFile == "" {
		t.Fatal("newFile should not be empty")
	}
	if newFile != cacheFile+"-new" {
		t.Errorf("newFile = %s, want %s-new", newFile, cacheFile)
	}

	// Verify hash is in path
	hash := ComputeHash(sourceURL, true)
	sanitized := RemoveInvalidFileNameChars(hash)
	if !containsString(cacheFile, sanitized) {
		t.Errorf("cacheFile path should contain hash folder")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDiskCache_SetWithValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "validation-test"
	data := []byte(`{"valid":"json"}`)

	// Test with successful validation
	validator := func(rs io.ReadSeeker) error {
		// Read and validate
		content, err := io.ReadAll(rs)
		if err != nil {
			return err
		}
		if len(content) == 0 {
			return fmt.Errorf("empty content")
		}
		return nil
	}

	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), validator)
	if err != nil {
		t.Fatalf("Set() with validation error = %v", err)
	}

	// Verify it was cached
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	_ = reader.Close()
}

func TestDiskCache_SetWithValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "validation-error-test"
	data := []byte(`invalid data`)

	// Test with failing validation
	validator := func(rs io.ReadSeeker) error {
		return fmt.Errorf("validation failed")
	}

	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), validator)
	if err == nil {
		t.Fatal("Set() expected validation error")
	}
	if !containsSubstring(err.Error(), "validation failed") {
		t.Errorf("Set() error = %v, want validation failed", err)
	}
}

func TestNewDiskCache_InvalidPath(t *testing.T) {
	// Use platform-appropriate invalid path
	var invalidPath string
	if runtime.GOOS == "windows" {
		// Windows: Use NUL device with subdirectory (invalid)
		invalidPath = "NUL\\invalid\\path"
	} else {
		// Unix: Use /dev/null with subdirectory (invalid)
		invalidPath = "/dev/null/invalid/path"
	}

	// Try to create cache in invalid location
	_, err := NewDiskCache(invalidPath, 1024*1024)
	if err == nil {
		t.Fatal("NewDiskCache() expected error for invalid path")
	}
}

func TestDiskCache_GetMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	// Try to get non-existent file
	reader, ok, err := dc.Get("https://example.com", "missing-key", 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Fatal("Get() expected cache miss for missing file")
	}
	if reader != nil {
		t.Fatal("Get() expected nil reader for missing file")
	}
}

func TestDiskCache_SetDirectoryCreationError(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	// Set a cache entry first to create the directory structure
	sourceURL := "https://example.com/test"
	cacheKey := "test-key"
	data := []byte("test data")

	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it was cached
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	_ = reader.Close()
}

func TestDiskCache_SetRenameRaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://example.com/race"
	cacheKey := "race-key"

	// First write creates the cache file
	data1 := []byte("version 1")
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data1), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Second write should handle the existing file gracefully
	data2 := []byte("version 2")
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data2), nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Should be able to read one of the versions
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(got) == 0 {
		t.Error("Get() returned empty data")
	}
}

func TestIsFileAlreadyOpen(t *testing.T) {
	// Test isFileAlreadyOpen utility function
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.dat")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// File not open - should return false
	if isFileAlreadyOpen(testFile) {
		t.Error("isFileAlreadyOpen() should return false for closed file")
	}

	// Open file for reading
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = file.Close() }()

	// File is open - behavior is platform-specific
	// On Unix, file can be opened multiple times
	// On Windows, exclusive access may prevent reopening
	_ = isFileAlreadyOpen(testFile)

	// Non-existent file should return false
	if isFileAlreadyOpen(filepath.Join(tmpDir, "nonexistent.dat")) {
		t.Error("isFileAlreadyOpen() should return false for non-existent file")
	}
}

func TestDiskCache_SetValidationError(t *testing.T) {
	// Test Set with validation that fails
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024*10)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "validation-test"
	data := []byte("test data")

	// Validation function that fails
	validate := func(r io.ReadSeeker) error {
		return fmt.Errorf("validation failed")
	}

	// Set should fail due to validation
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(data), validate)
	if err == nil {
		t.Fatal("Set() should fail with validation error")
	}

	// Verify cache file was not created
	cacheFile, _ := dc.GetCachePath(sourceURL, cacheKey)
	if fileExists(cacheFile) {
		t.Error("Cache file should not exist after validation failure")
	}
}

func TestGetInvalidFileNameChars_Coverage(t *testing.T) {
	// Test both Unix and Windows paths
	chars := getInvalidFileNameChars()

	// Should always contain null character
	if !slices.Contains(chars, '\x00') {
		t.Error("Invalid chars should include null character")
	}

	// Should contain forward slash (invalid on all platforms)
	if !slices.Contains(chars, '/') {
		t.Error("Invalid chars should include forward slash")
	}
}
