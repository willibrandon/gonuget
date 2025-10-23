package cache

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"
)

func TestDiskCache_TTL_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "test-resource"
	testData := []byte("test content")

	// Write to cache
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(testData), nil)
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
	_ = reader.Close()

	// Test 2: Read with zero TTL (should fail - expired)
	reader, ok, err = dc.Get(sourceURL, cacheKey, 0*time.Second)
	if err != nil {
		t.Fatalf("Get() with 0s TTL failed: %v", err)
	}
	if ok {
		_ = reader.Close()
		t.Fatal("Get() with 0s TTL should return cache miss (expired)")
	}

	// Test 3: Manually modify file time to simulate old cache
	cachePath, _ := dc.GetCachePath(sourceURL, cacheKey)
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
		_ = reader.Close()
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
	_ = reader.Close()
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}
	if !bytes.Equal(data, testData) {
		t.Errorf("Data mismatch: got %s, want %s", data, testData)
	}
}

func TestDiskCache_NoCache_Bypass(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache("", 1024*1024) // Empty rootDir = cache disabled
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "test-resource"
	testData := []byte("test content")

	// Write should succeed but do nothing
	err = dc.Set(sourceURL, cacheKey, bytes.NewReader(testData), nil)
	if err != nil {
		t.Fatalf("Set() with NoCache failed: %v", err)
	}

	// Read should return cache miss
	reader, ok, err := dc.Get(sourceURL, cacheKey, 1*time.Hour)
	if err != nil {
		t.Fatalf("Get() with NoCache failed: %v", err)
	}
	if ok {
		_ = reader.Close()
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

func TestDiskCache_TTL_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "edge-case-test"
	testData := []byte("edge case data")

	tests := []struct {
		name      string
		ttl       time.Duration
		fileAge   time.Duration
		wantHit   bool
		wantError bool
	}{
		{
			name:    "Fresh file with default 30min TTL",
			ttl:     30 * time.Minute,
			fileAge: 0,
			wantHit: true,
		},
		{
			name:    "File at exact TTL boundary",
			ttl:     30 * time.Minute,
			fileAge: 30 * time.Minute,
			wantHit: false, // age >= maxAge means expired
		},
		{
			name:    "File just under TTL boundary",
			ttl:     30 * time.Minute,
			fileAge: 29*time.Minute + 59*time.Second,
			wantHit: true,
		},
		{
			name:    "Very large TTL (24 hours)",
			ttl:     24 * time.Hour,
			fileAge: 1 * time.Hour,
			wantHit: true,
		},
		{
			name:    "Zero TTL (immediate expiration)",
			ttl:     0,
			fileAge: 0,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write fresh data
			err := dc.Set(sourceURL, cacheKey, bytes.NewReader(testData), nil)
			if err != nil {
				t.Fatalf("Set() failed: %v", err)
			}

			// Modify file time if needed
			if tt.fileAge > 0 {
				cachePath, _ := dc.GetCachePath(sourceURL, cacheKey)
				oldTime := time.Now().Add(-tt.fileAge)
				err = os.Chtimes(cachePath, oldTime, oldTime)
				if err != nil {
					t.Fatalf("Chtimes() failed: %v", err)
				}
			}

			// Try to get with specified TTL
			reader, ok, err := dc.Get(sourceURL, cacheKey, tt.ttl)
			if (err != nil) != tt.wantError {
				t.Fatalf("Get() error = %v, wantError %v", err, tt.wantError)
			}

			if ok != tt.wantHit {
				t.Errorf("Get() hit = %v, want %v", ok, tt.wantHit)
			}

			if ok && reader != nil {
				_ = reader.Close()
			}
		})
	}
}

func TestDiskCache_TTL_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	dc, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	sourceURL := "https://api.nuget.org/v3/index.json"
	cacheKey := "nonexistent"

	// Get should return miss for missing file
	reader, ok, err := dc.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() for missing file failed: %v", err)
	}
	if ok {
		_ = reader.Close()
		t.Error("Get() for missing file should return miss")
	}
	if reader != nil {
		t.Error("Get() for missing file should return nil reader")
	}
}
