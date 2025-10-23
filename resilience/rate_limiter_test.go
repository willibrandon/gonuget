package resilience

import (
	"context"
	"testing"
	"time"
)

func TestDefaultTokenBucketConfig(t *testing.T) {
	config := DefaultTokenBucketConfig()

	if config.Capacity != 100 {
		t.Errorf("Capacity = %d, want 100", config.Capacity)
	}
	if config.RefillRate != 50.0 {
		t.Errorf("RefillRate = %f, want 50.0", config.RefillRate)
	}
	if config.InitialTokens != 100 {
		t.Errorf("InitialTokens = %d, want 100", config.InitialTokens)
	}
}

func TestNewTokenBucket_InitialTokensCapped(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    5.0,
		InitialTokens: 20, // More than capacity
	}
	tb := NewTokenBucket(config)

	tokens := tb.Tokens()
	if tokens > 10.0 {
		t.Errorf("Tokens = %f, want <= 10 (capped at capacity)", tokens)
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    10.0, // 10 tokens/second
		InitialTokens: 10,
	}
	tb := NewTokenBucket(config)

	// Should allow first 10 requests immediately
	for i := range 10 {
		if !tb.Allow() {
			t.Errorf("Request %d denied, want allowed", i+1)
		}
	}

	// 11th request should be denied (bucket empty)
	if tb.Allow() {
		t.Error("Request 11 allowed, want denied (bucket empty)")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := &TokenBucket{
		capacity:     10,
		refillRate:   10.0,
		tokens:       0, // Start empty
		lastRefillAt: time.Now(),
	}

	// Should be denied initially
	if tb.Allow() {
		t.Error("Request allowed with empty bucket")
	}

	// Wait for 1 second (should refill 10 tokens)
	time.Sleep(1100 * time.Millisecond)

	// Should now allow 10 requests
	allowed := 0
	for range 10 {
		if tb.Allow() {
			allowed++
		}
	}

	if allowed < 10 {
		t.Errorf("Allowed %d requests after refill, want 10", allowed)
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      10,
		RefillRate:    10.0,
		InitialTokens: 10,
	}
	tb := NewTokenBucket(config)

	// Should allow batch of 5
	if !tb.AllowN(5) {
		t.Error("AllowN(5) denied, want allowed")
	}

	// Should allow another batch of 5
	if !tb.AllowN(5) {
		t.Error("AllowN(5) denied, want allowed")
	}

	// Should deny batch of 5 (bucket empty)
	if tb.AllowN(5) {
		t.Error("AllowN(5) allowed with empty bucket")
	}
}

func TestTokenBucket_Wait(t *testing.T) {
	tb := &TokenBucket{
		capacity:     10,
		refillRate:   100.0, // Fast refill for test
		tokens:       0,     // Start empty
		lastRefillAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := tb.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait() failed: %v", err)
	}

	// Should have waited for refill (at least 10ms for 1 token at 100/s rate)
	if elapsed < 5*time.Millisecond {
		t.Errorf("Wait elapsed %v, expected at least 5ms", elapsed)
	}
}

func TestTokenBucket_Wait_ContextCancelled(t *testing.T) {
	tb := &TokenBucket{
		capacity:     10,
		refillRate:   1.0, // Very slow refill
		tokens:       0,   // Start empty
		lastRefillAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := tb.Wait(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("Wait() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestTokenBucket_Tokens(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      100,
		RefillRate:    50.0,
		InitialTokens: 100,
	}
	tb := NewTokenBucket(config)

	tokens := tb.Tokens()
	if tokens < 99.0 || tokens > 101.0 {
		t.Errorf("Tokens() = %f, want ~100", tokens)
	}

	// Consume some tokens
	tb.AllowN(50)

	tokens = tb.Tokens()
	if tokens < 49.0 || tokens > 51.0 {
		t.Errorf("Tokens() after consuming 50 = %f, want ~50", tokens)
	}
}

func TestTokenBucket_Stats(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      100,
		RefillRate:    50.0,
		InitialTokens: 75,
	}
	tb := NewTokenBucket(config)

	stats := tb.Stats()

	if stats.Capacity != 100 {
		t.Errorf("Stats.Capacity = %d, want 100", stats.Capacity)
	}
	if stats.RefillRate != 50.0 {
		t.Errorf("Stats.RefillRate = %f, want 50.0", stats.RefillRate)
	}
	if stats.Tokens < 74.0 || stats.Tokens > 76.0 {
		t.Errorf("Stats.Tokens = %f, want ~75", stats.Tokens)
	}
}

func TestTokenBucket_BurstCapacity(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:      100,  // Allow burst of 100
		RefillRate:    10.0, // But sustained rate is 10/s
		InitialTokens: 100,
	}
	tb := NewTokenBucket(config)

	// Should allow burst of 100
	for i := range 100 {
		if !tb.Allow() {
			t.Fatalf("Burst request %d denied", i+1)
		}
	}

	// Now bucket is empty, should refill slowly
	time.Sleep(110 * time.Millisecond) // ~1 token

	// Should allow 1 request (1 token refilled in 0.1s at 10/s rate)
	if !tb.Allow() {
		t.Error("Request after refill denied")
	}

	// Immediate next request should be denied
	if tb.Allow() {
		t.Error("Immediate request allowed, want denied")
	}
}
