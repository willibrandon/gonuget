package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/gonuget/resilience"
)

// TestClient_CircuitBreaker_OpensAfterFailures verifies circuit breaker opens after N failures
func TestClient_CircuitBreaker_OpensAfterFailures(t *testing.T) {
	// Create failing server
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with circuit breaker (opens after 3 failures)
	client := NewClient(&Config{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         3,
			Timeout:             1 * time.Second,
			MaxHalfOpenRequests: 1,
		},
	})

	ctx := context.Background()

	// Make requests until circuit opens
	for i := range 5 {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(ctx, req)

		// First 3 requests should fail with 500
		if i < 3 {
			if resp == nil {
				t.Errorf("Request %d: expected response, got nil", i+1)
				continue
			}
			if resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("Request %d: expected 500, got %d", i+1, resp.StatusCode)
			}
			_ = resp.Body.Close()
		} else {
			// After 3 failures, circuit should be open
			if err == nil {
				t.Errorf("Request %d: expected circuit breaker error, got nil", i+1)
				if resp != nil {
					_ = resp.Body.Close()
				}
				continue
			}
			if !strings.Contains(err.Error(), "circuit breaker is open") {
				t.Errorf("Request %d: expected 'circuit breaker is open', got: %v", i+1, err)
			}
			// Circuit is open - test passed
			break
		}
	}

	// Verify circuit opened (should be exactly 3 failures before opening)
	if failCount != 3 {
		t.Errorf("Expected exactly 3 failures before circuit opened, got %d", failCount)
	}
}

// TestClient_CircuitBreaker_HalfOpenTransition verifies circuit breaker transitions to half-open
func TestClient_CircuitBreaker_HalfOpenTransition(t *testing.T) {
	// Create server that fails first 3 requests, then succeeds
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Create client with circuit breaker (short timeout for faster test)
	client := NewClient(&Config{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         3,
			Timeout:             100 * time.Millisecond, // Short timeout
			MaxHalfOpenRequests: 1,
		},
	})

	ctx := context.Background()

	// Make 3 failing requests to open circuit
	for range 3 {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(ctx, req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}

	// Verify circuit is open
	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := client.Do(ctx, req)
	if err == nil || !strings.Contains(err.Error(), "circuit breaker is open") {
		t.Fatal("Circuit breaker should be open after 3 failures")
	}

	// Wait for timeout (circuit should transition to half-open)
	time.Sleep(150 * time.Millisecond)

	// Next request should succeed (server now returns 200)
	req, _ = http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Expected successful request in half-open state, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// Circuit should now be closed - verify next request succeeds
	req, _ = http.NewRequest("GET", server.URL, nil)
	resp, err = client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Expected successful request after circuit closed, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK after circuit closed, got %d", resp.StatusCode)
	}
}

// TestClient_CircuitBreaker_PerHost verifies circuit breaker is per-host
func TestClient_CircuitBreaker_PerHost(t *testing.T) {
	// Create two servers - one failing, one working
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	workingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer workingServer.Close()

	// Create client with circuit breaker
	client := NewClient(&Config{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         3,
			Timeout:             1 * time.Second,
			MaxHalfOpenRequests: 1,
		},
	})

	ctx := context.Background()

	// Open circuit for failing server
	for range 4 {
		req, _ := http.NewRequest("GET", failingServer.URL, nil)
		resp, err := client.Do(ctx, req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}

	// Verify circuit is open for failing server
	req, _ := http.NewRequest("GET", failingServer.URL, nil)
	_, err := client.Do(ctx, req)
	if err == nil || !strings.Contains(err.Error(), "circuit breaker is open") {
		t.Fatal("Circuit breaker should be open for failing server")
	}

	// Verify working server is unaffected (circuit per-host)
	req, _ = http.NewRequest("GET", workingServer.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Working server should be accessible, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK from working server, got %d", resp.StatusCode)
	}
}

// TestClient_RateLimiter_DelaysRequests verifies rate limiter delays requests
func TestClient_RateLimiter_DelaysRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with rate limiter (2 tokens, 1 per second)
	client := NewClient(&Config{
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   2,
			RefillRate: 1.0, // 1 token per second
		},
	})

	ctx := context.Background()

	// First 2 requests should succeed immediately
	start := time.Now()
	for i := range 2 {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(ctx, req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		_ = resp.Body.Close()
	}

	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Errorf("First 2 requests should be immediate, took %v", elapsed)
	}

	// Third request should be delayed ~1 second (waiting for token)
	start = time.Now()
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Third request failed: %v", err)
	}
	_ = resp.Body.Close()

	elapsed = time.Since(start)
	if elapsed < 900*time.Millisecond {
		t.Errorf("Third request should be delayed ~1s, was only %v", elapsed)
	}
	if elapsed > 1200*time.Millisecond {
		t.Errorf("Third request delayed too long: %v", elapsed)
	}
}

// TestClient_RateLimiter_ContextCancellation verifies rate limiter respects context
func TestClient_RateLimiter_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with rate limiter (1 token, slow refill)
	client := NewClient(&Config{
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   1,
			RefillRate: 0.1, // Very slow - 0.1 token per second (10s for 1 token)
		},
	})

	// Consume the initial token
	ctx := context.Background()
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	_ = resp.Body.Close()

	// Second request should block waiting for token, but we'll cancel context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ = http.NewRequest("GET", server.URL, nil)
	_, err = client.Do(ctx, req)

	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}

	if !strings.Contains(err.Error(), "rate limit wait failed") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected rate limit or context error, got: %v", err)
	}
}

// TestClient_RateLimiter_PerHost verifies rate limiter is per-host
func TestClient_RateLimiter_PerHost(t *testing.T) {
	// Create two servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Create client with rate limiter (1 token, slow refill)
	client := NewClient(&Config{
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   1,
			RefillRate: 0.5, // 0.5 token per second
		},
	})

	ctx := context.Background()

	// Consume token for server1
	req, _ := http.NewRequest("GET", server1.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Server1 request failed: %v", err)
	}
	_ = resp.Body.Close()

	// Server2 should have its own token bucket (independent)
	req, _ = http.NewRequest("GET", server2.URL, nil)
	start := time.Now()
	resp, err = client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Server2 request failed: %v", err)
	}
	_ = resp.Body.Close()

	elapsed := time.Since(start)
	// Server2 should be immediate (has its own tokens)
	if elapsed > 500*time.Millisecond {
		t.Errorf("Server2 should have independent rate limit, took %v", elapsed)
	}
}

// TestClient_CircuitBreaker_And_RateLimiter verifies both work together
func TestClient_CircuitBreaker_And_RateLimiter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with both circuit breaker and rate limiter
	client := NewClient(&Config{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         5,
			Timeout:             1 * time.Second,
			MaxHalfOpenRequests: 1,
		},
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   2,
			RefillRate: 10.0, // Fast refill for test
		},
	})

	ctx := context.Background()

	// Make requests - should be rate limited but not circuit broken
	for i := range 5 {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(ctx, req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		_ = resp.Body.Close()
	}

	// All requests should succeed (circuit not opened, rate limiter just delays)
}

// TestClient_DoWithRetry_CircuitBreaker verifies circuit breaker wraps retry sequence
func TestClient_DoWithRetry_CircuitBreaker(t *testing.T) {
	// Create server that always fails
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with circuit breaker and retry
	client := NewClient(&Config{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         2, // Opens after 2 failures
			Timeout:             1 * time.Second,
			MaxHalfOpenRequests: 1,
		},
		RetryConfig: &RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 10,
			MaxBackoff:     100,
			BackoffFactor:  2,
		},
	})

	ctx := context.Background()

	// First DoWithRetry call - will make 4 attempts (initial + 3 retries)
	// This counts as 1 failure for circuit breaker (returns 500 response, not error)
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Second DoWithRetry call - another failure, circuit should open
	req, _ = http.NewRequest("GET", server.URL, nil)
	resp, err = client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error on second call: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Third attempt should fail immediately with circuit breaker error
	req, _ = http.NewRequest("GET", server.URL, nil)
	_, err = client.DoWithRetry(ctx, req)
	if err == nil {
		t.Fatal("Expected circuit breaker error")
	}

	if !strings.Contains(err.Error(), "circuit breaker is open") {
		t.Errorf("Expected 'circuit breaker is open', got: %v", err)
	}

	// Verify circuit breaker prevented retry attempts
	// Circuit breaker wraps the ENTIRE retry sequence, so:
	// - First DoWithRetry: Circuit allows, but the operation itself fails (counts as 1 circuit failure)
	// - Second DoWithRetry: Circuit allows, operation fails again (counts as 2nd circuit failure, opens circuit)
	// - Third DoWithRetry: Circuit is open, blocks immediately (no attempts)
	// However, we only see 2 HTTP attempts because circuit opened after 2 failures at the circuit level
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts before circuit opened, got %d", attemptCount)
	}
}

// TestClient_NoResilience verifies client works without resilience configured
func TestClient_NoResilience(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client without circuit breaker or rate limiter (nil configs)
	client := NewClient(&Config{
		CircuitBreakerConfig: nil,
		RateLimiterConfig:    nil,
	})

	ctx := context.Background()

	// Should work normally without resilience
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// DoWithRetry should also work
	req, _ = http.NewRequest("GET", server.URL, nil)
	resp, err = client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("DoWithRetry failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK from DoWithRetry, got %d", resp.StatusCode)
	}
}
