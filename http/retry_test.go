package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"network timeout", &net.DNSError{IsTimeout: true}, true},
		{"connection reset", syscall.ECONNRESET, true},
		{"connection refused", syscall.ECONNREFUSED, true},
		{"context deadline", context.DeadlineExceeded, true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetriable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetriable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsRetriableStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{404, false},
		{429, true},
		{500, false},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			got := IsRetriableStatus(tt.code)
			if got != tt.want {
				t.Errorf("IsRetriableStatus(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := DefaultRetryConfig()

	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{0, 900 * time.Millisecond, 1100 * time.Millisecond},
		{1, 1800 * time.Millisecond, 2200 * time.Millisecond},
		{2, 3600 * time.Millisecond, 4400 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			backoff := cfg.CalculateBackoff(tt.attempt)
			if backoff < tt.wantMin || backoff > tt.wantMax {
				t.Errorf("CalculateBackoff(%d) = %v, want between %v and %v",
					tt.attempt, backoff, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestClient_DoWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(nil)
	ctx := context.Background()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestClient_DoWithRetry_EventualSuccess(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.RetryConfig = &RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		JitterFactor:   0.1,
	}
	client := NewClient(cfg)
	ctx := context.Background()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_DoWithRetry_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.RetryConfig = &RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  2.0,
		JitterFactor:   0.1,
	}
	client := NewClient(cfg)
	ctx := context.Background()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	// Should return last response even after max retries
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want 503", resp.StatusCode)
	}
}

func TestClient_DoWithRetry_NonRetriableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(nil)
	ctx := context.Background()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	// Should not retry 404
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}
