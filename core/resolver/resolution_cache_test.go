package resolver

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOperationCache_DeduplicatesConcurrent(t *testing.T) {
	cache := NewOperationCache(5 * time.Minute)

	callCount := int32(0)
	operation := func(ctx context.Context) (*PackageDependencyInfo, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(100 * time.Millisecond) // Simulate work
		return &PackageDependencyInfo{ID: "TestPackage", Version: "1.0.0"}, nil
	}

	// Start 10 concurrent requests for same key
	var wg sync.WaitGroup
	results := make([]*PackageDependencyInfo, 10)

	for i := range 10 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := cache.GetOrStart(context.Background(), "test-key", operation)
			if err != nil {
				t.Errorf("GetOrStart failed: %v", err)
				return
			}
			results[index] = result
		}(i)
	}

	wg.Wait()

	// Should have called operation exactly once
	if callCount != 1 {
		t.Errorf("Expected operation to be called once, called %d times", callCount)
	}

	// All results should be identical
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		if result.ID != "TestPackage" || result.Version != "1.0.0" {
			t.Errorf("Result %d has wrong value: %v", i, result)
		}
	}
}

func TestOperationCache_TTL(t *testing.T) {
	cache := NewOperationCache(100 * time.Millisecond)

	operation := func(ctx context.Context) (*PackageDependencyInfo, error) {
		return &PackageDependencyInfo{ID: "Test", Version: "1.0.0"}, nil
	}

	// First call
	_, err := cache.GetOrStart(context.Background(), "test-key", operation)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Second call should start new operation (TTL expired)
	callCount := 0
	operation2 := func(ctx context.Context) (*PackageDependencyInfo, error) {
		callCount++
		return &PackageDependencyInfo{ID: "Test", Version: "2.0.0"}, nil
	}

	result, err := cache.GetOrStart(context.Background(), "test-key", operation2)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected new operation after TTL, but cached value was used")
	}

	if result.Version != "2.0.0" {
		t.Errorf("Expected new version 2.0.0, got %s", result.Version)
	}
}

func TestOperationCache_ContextCancellation(t *testing.T) {
	cache := NewOperationCache(5 * time.Minute)

	started := make(chan struct{})
	operation := func(ctx context.Context) (*PackageDependencyInfo, error) {
		close(started)
		time.Sleep(500 * time.Millisecond) // Simulate long operation
		return &PackageDependencyInfo{ID: "Test", Version: "1.0.0"}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start operation in background
	go func() {
		_, _ = cache.GetOrStart(ctx, "test-key", operation)
	}()

	// Wait for operation to start
	<-started

	// Now cancel
	cancel()

	// Try to get with canceled context
	_, err := cache.GetOrStart(ctx, "test-key", operation)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestOperationCache_Clear(t *testing.T) {
	cache := NewOperationCache(5 * time.Minute)

	operation := func(ctx context.Context) (*PackageDependencyInfo, error) {
		return &PackageDependencyInfo{ID: "Test", Version: "1.0.0"}, nil
	}

	// Add some operations
	_, err := cache.GetOrStart(context.Background(), "key1", operation)
	if err != nil {
		t.Fatalf("GetOrStart key1 failed: %v", err)
	}
	_, err = cache.GetOrStart(context.Background(), "key2", operation)
	if err != nil {
		t.Fatalf("GetOrStart key2 failed: %v", err)
	}

	// Clear cache
	cache.Clear()

	// Verify operations are cleared (check internal state)
	count := 0
	cache.operations.Range(func(key, value any) bool {
		count++
		return true
	})

	if count != 0 {
		t.Errorf("Expected 0 operations after clear, got %d", count)
	}
}

func TestOperationCache_NoTTL(t *testing.T) {
	cache := NewOperationCache(0) // No expiration

	callCount := int32(0)
	started := make(chan struct{})
	operation := func(ctx context.Context) (*PackageDependencyInfo, error) {
		atomic.AddInt32(&callCount, 1)
		close(started)
		time.Sleep(200 * time.Millisecond) // Slow operation
		return &PackageDependencyInfo{ID: "Test", Version: "1.0.0"}, nil
	}

	// Start first call
	go func() {
		_, _ = cache.GetOrStart(context.Background(), "test-key", operation)
	}()

	// Wait for operation to start
	<-started

	// Second call while first is still running - should share same operation
	_, err := cache.GetOrStart(context.Background(), "test-key", operation)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	// With no TTL, both calls should use the same operation
	if callCount != 1 {
		t.Errorf("Expected operation called once with no TTL, got %d", callCount)
	}
}

func TestOperationCache_Error(t *testing.T) {
	cache := NewOperationCache(5 * time.Minute)

	operation := func(ctx context.Context) (*PackageDependencyInfo, error) {
		return nil, context.DeadlineExceeded
	}

	result, err := cache.GetOrStart(context.Background(), "test-key", operation)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got %v", result)
	}
}
