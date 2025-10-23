package resilience

import (
	"context"
	"testing"
)

func TestPerSourceLimiter_Isolation(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    10.0,
		InitialTokens: 10,
	}
	psl := NewPerSourceLimiter(config)

	source1 := "https://api.nuget.org/v3/index.json"
	source2 := "https://pkgs.dev.azure.com/example/index.json"

	// Exhaust source1
	for i := range 10 {
		if !psl.Allow(source1) {
			t.Fatalf("source1 request %d denied", i+1)
		}
	}

	// source1 should be rate limited
	if psl.Allow(source1) {
		t.Error("source1 allowed after exhaustion")
	}

	// source2 should still have tokens
	for i := range 10 {
		if !psl.Allow(source2) {
			t.Errorf("source2 request %d denied (should be isolated)", i+1)
		}
	}
}

func TestPerSourceLimiter_GetAllStats(t *testing.T) {
	psl := NewPerSourceLimiterWithDefaults()

	sources := []string{
		"https://api.nuget.org/v3/index.json",
		"https://pkgs.dev.azure.com/example/index.json",
		"https://github.com/example/index.json",
	}

	// Make requests to each source
	for _, source := range sources {
		psl.Allow(source)
	}

	stats := psl.GetAllStats()

	if len(stats) != len(sources) {
		t.Errorf("Stats count = %d, want %d", len(stats), len(sources))
	}

	for _, source := range sources {
		if _, exists := stats[source]; !exists {
			t.Errorf("Stats missing for source: %s", source)
		}
	}
}

func TestPerSourceLimiter_LazyCreation(t *testing.T) {
	psl := NewPerSourceLimiterWithDefaults()

	// Initially no limiters
	stats := psl.GetAllStats()
	if len(stats) != 0 {
		t.Errorf("Initial stats count = %d, want 0", len(stats))
	}

	// First request creates limiter
	source := "https://api.nuget.org/v3/index.json"
	if !psl.Allow(source) {
		t.Error("First request denied")
	}

	// Now should have 1 limiter
	stats = psl.GetAllStats()
	if len(stats) != 1 {
		t.Errorf("Stats count after request = %d, want 1", len(stats))
	}
}

func TestPerSourceLimiter_AllowN(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    10.0,
		InitialTokens: 10,
	}
	psl := NewPerSourceLimiter(config)

	source := "https://api.nuget.org/v3/index.json"

	// Should allow batch of 5
	if !psl.AllowN(source, 5) {
		t.Error("AllowN(5) denied, want allowed")
	}

	// Should allow another batch of 5
	if !psl.AllowN(source, 5) {
		t.Error("AllowN(5) denied, want allowed")
	}

	// Should deny batch of 5 (bucket empty)
	if psl.AllowN(source, 5) {
		t.Error("AllowN(5) allowed with empty bucket")
	}
}

func TestPerSourceLimiter_Wait(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    100.0, // Fast refill for test
		InitialTokens: 10,
	}
	psl := NewPerSourceLimiter(config)

	source := "https://api.nuget.org/v3/index.json"

	// Exhaust tokens
	for range 10 {
		psl.Allow(source)
	}

	// Wait should succeed after refill
	ctx := context.Background()
	err := psl.Wait(ctx, source)
	if err != nil {
		t.Fatalf("Wait() failed: %v", err)
	}
}

func TestPerSourceLimiter_WaitN(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    100.0, // Fast refill for test
		InitialTokens: 10,
	}
	psl := NewPerSourceLimiter(config)

	source := "https://api.nuget.org/v3/index.json"

	// Exhaust tokens
	for range 10 {
		psl.Allow(source)
	}

	// WaitN should succeed after refill
	ctx := context.Background()
	err := psl.WaitN(ctx, source, 5)
	if err != nil {
		t.Fatalf("WaitN() failed: %v", err)
	}
}

func TestPerSourceLimiter_GetStats(t *testing.T) {
	psl := NewPerSourceLimiterWithDefaults()

	source := "https://api.nuget.org/v3/index.json"

	// Stats for non-existent source should be nil
	stats := psl.GetStats(source)
	if stats != nil {
		t.Error("GetStats() for non-existent source should return nil")
	}

	// Make a request to create limiter
	psl.Allow(source)

	// Now stats should exist
	stats = psl.GetStats(source)
	if stats == nil {
		t.Fatal("GetStats() for existing source should not be nil")
	}

	if stats.Capacity != 100 {
		t.Errorf("Stats.Capacity = %d, want 100", stats.Capacity)
	}
}
