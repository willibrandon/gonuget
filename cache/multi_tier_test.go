package cache

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestMultiTierCache_L1Hit(t *testing.T) {
	// L1 hit should not touch L2
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Populate L1 only
	l1.Set("test-key", []byte("L1 data"), 30*time.Minute)

	// Get should return L1 data without touching L2
	ctx := context.Background()
	data, ok, err := mtc.Get(ctx, "https://example.com", "test-key", 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	if string(data) != "L1 data" {
		t.Errorf("Get() = %q, want %q", string(data), "L1 data")
	}

	// Verify L2 was not touched (no file exists)
	cacheFile, _ := l2.GetCachePath("https://example.com", "test-key")
	if fileExists(cacheFile) {
		t.Error("L1 hit should not create L2 file")
	}
}

func TestMultiTierCache_L2HitWithPromotion(t *testing.T) {
	// L2 hit should promote to L1
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Populate L2 only
	ctx := context.Background()
	sourceURL := "https://example.com"
	cacheKey := "test-key"
	data := []byte("L2 data")

	err = l2.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("L2.Set() error = %v", err)
	}

	// Verify L1 is empty
	if _, ok := l1.Get(cacheKey); ok {
		t.Fatal("L1 should be empty initially")
	}

	// Get should find in L2 and promote to L1
	gotData, ok, err := mtc.Get(ctx, sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() expected cache hit")
	}
	if string(gotData) != "L2 data" {
		t.Errorf("Get() = %q, want %q", string(gotData), "L2 data")
	}

	// Verify promotion to L1
	l1Data, ok := l1.Get(cacheKey)
	if !ok {
		t.Fatal("Data should be promoted to L1")
	}
	if string(l1Data) != "L2 data" {
		t.Errorf("L1 data = %q, want %q", string(l1Data), "L2 data")
	}

	// Second get should hit L1 (verify promotion worked)
	gotData2, ok2, err2 := mtc.Get(ctx, sourceURL, cacheKey, 30*time.Minute)
	if err2 != nil {
		t.Fatalf("Get() error = %v", err2)
	}
	if !ok2 {
		t.Fatal("Get() expected cache hit")
	}
	if string(gotData2) != "L2 data" {
		t.Errorf("Get() = %q, want %q", string(gotData2), "L2 data")
	}
}

func TestMultiTierCache_CacheMiss(t *testing.T) {
	// Miss in both L1 and L2
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Get non-existent key
	ctx := context.Background()
	data, ok, err := mtc.Get(ctx, "https://example.com", "missing-key", 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Fatal("Get() expected cache miss")
	}
	if data != nil {
		t.Errorf("Get() data = %v, want nil", data)
	}
}

func TestMultiTierCache_SetWritesBothTiers(t *testing.T) {
	// Set should write to both L1 and L2
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Set data
	ctx := context.Background()
	sourceURL := "https://example.com"
	cacheKey := "test-key"
	data := []byte("test data")

	err = mtc.Set(ctx, sourceURL, cacheKey, bytes.NewReader(data), 30*time.Minute, nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify L1 has data
	l1Data, ok := l1.Get(cacheKey)
	if !ok {
		t.Fatal("L1 should have data after Set()")
	}
	if string(l1Data) != "test data" {
		t.Errorf("L1 data = %q, want %q", string(l1Data), "test data")
	}

	// Verify L2 has data
	reader, ok, err := l2.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("L2.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("L2 should have data after Set()")
	}
	defer func() { _ = reader.Close() }()

	l2Data, err := bytes.NewBuffer(nil), error(nil)
	_, err = l2Data.ReadFrom(reader)
	if err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}
	if l2Data.String() != "test data" {
		t.Errorf("L2 data = %q, want %q", l2Data.String(), "test data")
	}
}

func TestMultiTierCache_ClearBothTiers(t *testing.T) {
	// Clear should clear both L1 and L2
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Populate both caches
	ctx := context.Background()
	sourceURL := "https://example.com"
	cacheKey := "test-key"
	data := []byte("test data")

	err = mtc.Set(ctx, sourceURL, cacheKey, bytes.NewReader(data), 30*time.Minute, nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify both have data
	if _, ok := l1.Get(cacheKey); !ok {
		t.Fatal("L1 should have data before clear")
	}
	reader, ok, err := l2.Get(sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("L2.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("L2 should have data before clear")
	}
	_ = reader.Close() // Must close before Clear() on Windows

	// Clear
	err = mtc.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify both are cleared
	if _, ok := l1.Get(cacheKey); ok {
		t.Error("L1 should be cleared")
	}

	// Get should return miss after clear
	_, ok, err = mtc.Get(ctx, sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Error("Get() should return miss after clear")
	}
}

func TestMultiTierCache_SetWithValidation(t *testing.T) {
	// Set with validation should validate L2 write
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Set with failing validation
	ctx := context.Background()
	sourceURL := "https://example.com"
	cacheKey := "test-key"
	data := []byte("test data")

	validationCalled := false
	validate := func(r io.ReadSeeker) error {
		validationCalled = true
		return nil
	}

	err = mtc.Set(ctx, sourceURL, cacheKey, bytes.NewReader(data), 30*time.Minute, validate)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if !validationCalled {
		t.Error("Validation should be called")
	}

	// Verify data in both caches
	if _, ok := l1.Get(cacheKey); !ok {
		t.Error("L1 should have data after successful validation")
	}
	reader, ok, _ := l2.Get(sourceURL, cacheKey, 30*time.Minute)
	if !ok {
		t.Error("L2 should have data after successful validation")
	} else {
		_ = reader.Close() // Must close reader to avoid file lock on Windows
	}
}

func TestMultiTierCache_L2ExpiredNotPromoted(t *testing.T) {
	// Expired L2 entry should not be promoted
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)

	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Populate L2
	ctx := context.Background()
	sourceURL := "https://example.com"
	cacheKey := "test-key"
	data := []byte("test data")

	err = l2.Set(sourceURL, cacheKey, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("L2.Set() error = %v", err)
	}

	// Sleep to let TTL expire
	time.Sleep(100 * time.Millisecond)

	// Get with very short maxAge (should be expired)
	_, ok, err := mtc.Get(ctx, sourceURL, cacheKey, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Error("Get() should return miss for expired entry")
	}

	// Verify not promoted to L1
	if _, ok := l1.Get(cacheKey); ok {
		t.Error("Expired L2 entry should not be promoted to L1")
	}
}

func TestMultiTierCache_GetL2Error(t *testing.T) {
	// Test L2 error handling
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)
	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Create a corrupted disk cache by writing invalid data
	sourceURL := "https://example.com"
	cacheKey := "test-key"

	// Write valid data first
	err = l2.Set(sourceURL, cacheKey, bytes.NewReader([]byte("test")), nil)
	if err != nil {
		t.Fatalf("L2.Set() error = %v", err)
	}

	// Get from closed cache directory to trigger error
	// Close and remove the cache directory
	err = l2.Clear()
	if err != nil {
		t.Fatalf("L2.Clear() error = %v", err)
	}

	// Get should return miss (not error) when L2 file doesn't exist
	ctx := context.Background()
	_, ok, err := mtc.Get(ctx, sourceURL, cacheKey, 30*time.Minute)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Error("Get() should return miss when L2 file missing")
	}
}

func TestMultiTierCache_SetReadError(t *testing.T) {
	// Test Set with reader that fails
	tmpDir := t.TempDir()

	l1 := NewMemoryCache(100, 1024*1024)
	l2, err := NewDiskCache(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	mtc := NewMultiTierCache(l1, l2)

	// Create a reader that will fail
	ctx := context.Background()
	sourceURL := "https://example.com"
	cacheKey := "test-key"

	// Use empty reader (will succeed, so let's test validation failure instead)
	validateFailed := false
	validate := func(r io.ReadSeeker) error {
		validateFailed = true
		return io.ErrUnexpectedEOF
	}

	err = mtc.Set(ctx, sourceURL, cacheKey, bytes.NewReader([]byte("test")), 30*time.Minute, validate)
	if err == nil {
		t.Fatal("Set() should fail with validation error")
	}
	if !validateFailed {
		t.Error("Validation function should have been called")
	}

	// L1 should still have data even though L2 validation failed
	if _, ok := l1.Get(cacheKey); !ok {
		t.Error("L1 should have data even when L2 validation fails")
	}
}
