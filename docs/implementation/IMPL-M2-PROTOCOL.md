# Implementation Guide: Milestone 2 - Protocol Implementation

**Milestone:** M2 - Protocol Implementation
**Goal:** Implement NuGet v3 and v2 protocol support with HTTP client infrastructure
**Chunks:** 18 (M2.1 - M2.18)
**Estimated Time:** 48 hours
**Dependencies:** M1 (Foundation) must be complete

## Overview

This milestone implements the HTTP client infrastructure and both NuGet v3 (JSON/REST) and v2 (OData XML) protocols. You'll build:

- HTTP client with retry logic and Retry-After header parsing
- NuGet v3 protocol (service index, search, metadata, download, autocomplete)
- NuGet v2 protocol (legacy OData support)
- Authentication mechanisms (API key, bearer token, basic auth)
- Resource provider system for modular service discovery
- Source repository abstraction
- NuGet client core operations

---

## [M2.1] HTTP Client - Basic Configuration

**Time Estimate:** 1 hour
**Dependencies:** M1.1 (Go module)
**Status:** Not started

### What You'll Build

Create the foundational HTTP client with configurable timeouts, user agent, and HTTP/2 support. This will be used by all protocol implementations.

### Step-by-Step Instructions

**Step 1: Create HTTP client types**

Create `http/client.go`:

```go
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	DefaultTimeout        = 30 * time.Second
	DefaultDialTimeout    = 10 * time.Second
	DefaultUserAgent      = "gonuget/0.1.0"
)

// Client wraps http.Client with NuGet-specific configuration
type Client struct {
	httpClient *http.Client
	userAgent  string
	timeout    time.Duration
}

// Config holds HTTP client configuration
type Config struct {
	Timeout       time.Duration
	DialTimeout   time.Duration
	UserAgent     string
	TLSConfig     *tls.Config
	MaxIdleConns  int
	EnableHTTP2   bool
}

// DefaultConfig returns a client configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Timeout:      DefaultTimeout,
		DialTimeout:  DefaultDialTimeout,
		UserAgent:    DefaultUserAgent,
		MaxIdleConns: 100,
		EnableHTTP2:  true,
	}
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     cfg.TLSConfig,
		ForceAttemptHTTP2:   cfg.EnableHTTP2,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		userAgent: cfg.UserAgent,
		timeout:   cfg.Timeout,
	}
}

// Do executes an HTTP request with context and user agent
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	return c.httpClient.Do(req)
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return c.Do(ctx, req)
}

// SetUserAgent updates the client's user agent string
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
}
```

**Step 2: Create functional options pattern**

Add to `http/client.go`:

```go
// Option is a functional option for configuring the client
type Option func(*Config)

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithUserAgent sets the user agent string
func WithUserAgent(ua string) Option {
	return func(cfg *Config) {
		cfg.UserAgent = ua
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return func(cfg *Config) {
		cfg.TLSConfig = tlsCfg
	}
}

// WithMaxIdleConns sets the maximum idle connections
func WithMaxIdleConns(n int) Option {
	return func(cfg *Config) {
		cfg.MaxIdleConns = n
	}
}

// NewClientWithOptions creates a client with functional options
func NewClientWithOptions(opts ...Option) *Client {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return NewClient(cfg)
}
```

### Verification Steps

```bash
# Build
go build ./http

# Format check
gofmt -l http/
```

### Testing

Create `http/client_test.go`:

```go
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}
	if cfg.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, DefaultUserAgent)
	}
	if !cfg.EnableHTTP2 {
		t.Error("EnableHTTP2 = false, want true")
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "nil config uses defaults",
			cfg:  nil,
			want: DefaultUserAgent,
		},
		{
			name: "custom user agent",
			cfg:  &Config{UserAgent: "custom/1.0"},
			want: "custom/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			if client.userAgent != tt.want {
				t.Errorf("userAgent = %q, want %q", client.userAgent, tt.want)
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		ua := r.Header.Get("User-Agent")
		if ua != DefaultUserAgent {
			t.Errorf("User-Agent = %q, want %q", ua, DefaultUserAgent)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient(nil)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithOptions(WithTimeout(50 * time.Millisecond))
	ctx := context.Background()

	_, err := client.Get(ctx, server.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestFunctionalOptions(t *testing.T) {
	client := NewClientWithOptions(
		WithTimeout(5*time.Second),
		WithUserAgent("test/1.0"),
		WithMaxIdleConns(50),
	)

	if client.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", client.timeout)
	}
	if client.userAgent != "test/1.0" {
		t.Errorf("userAgent = %q, want test/1.0", client.userAgent)
	}
}
```

Run tests:

```bash
go test ./http -v
```

### Commit

```
feat: implement basic HTTP client with configuration

- Add HTTP client wrapper with configurable timeouts
- Support HTTP/2 by default
- Implement functional options pattern
- Add user agent header management
- Create comprehensive tests

Chunk: M2.1
Status: ✓ Complete
```

---

## [M2.2] HTTP Client - Retry Logic

**Time Estimate:** 3 hours
**Dependencies:** M2.1 (HTTP client)
**Status:** Not started

### What You'll Build

Implement exponential backoff retry logic with jitter for transient failures (429, 503, 504, network errors).

### Step-by-Step Instructions

**Step 1: Create retry configuration**

Create `http/retry.go`:

```go
package http

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"syscall"
	"time"
)

const (
	DefaultMaxRetries     = 3
	DefaultInitialBackoff = 1 * time.Second
	DefaultMaxBackoff     = 30 * time.Second
	DefaultBackoffFactor  = 2.0
	DefaultJitterFactor   = 0.1
)

// RetryConfig holds retry behavior configuration
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
	JitterFactor   float64
}

// DefaultRetryConfig returns retry configuration with sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     DefaultMaxRetries,
		InitialBackoff: DefaultInitialBackoff,
		MaxBackoff:     DefaultMaxBackoff,
		BackoffFactor:  DefaultBackoffFactor,
		JitterFactor:   DefaultJitterFactor,
	}
}

// IsRetriable determines if an error should be retried
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retriable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Connection reset, refused, timeout
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	return false
}

// IsRetriableStatus determines if an HTTP status code should be retried
func IsRetriableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,      // 429
		http.StatusServiceUnavailable,     // 503
		http.StatusGatewayTimeout:         // 504
		return true
	default:
		return false
	}
}

// CalculateBackoff computes exponential backoff with jitter
func (rc *RetryConfig) CalculateBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential backoff: initialBackoff * (factor ^ attempt)
	backoff := float64(rc.InitialBackoff) * math.Pow(rc.BackoffFactor, float64(attempt))

	// Cap at max backoff
	if backoff > float64(rc.MaxBackoff) {
		backoff = float64(rc.MaxBackoff)
	}

	// Add jitter: backoff * (1 ± jitterFactor)
	jitter := backoff * rc.JitterFactor * (2*rand.Float64() - 1)
	backoff += jitter

	// Ensure positive
	if backoff < 0 {
		backoff = float64(rc.InitialBackoff)
	}

	return time.Duration(backoff)
}
```

**Step 2: Add retry logic to client**

Add to `http/client.go`:

```go
// Add to Client struct
type Client struct {
	httpClient  *http.Client
	userAgent   string
	timeout     time.Duration
	retryConfig *RetryConfig
}

// Add to Config struct
type Config struct {
	Timeout       time.Duration
	DialTimeout   time.Duration
	UserAgent     string
	TLSConfig     *tls.Config
	MaxIdleConns  int
	EnableHTTP2   bool
	RetryConfig   *RetryConfig
}

// Update DefaultConfig
func DefaultConfig() *Config {
	return &Config{
		Timeout:      DefaultTimeout,
		DialTimeout:  DefaultDialTimeout,
		UserAgent:    DefaultUserAgent,
		MaxIdleConns: 100,
		EnableHTTP2:  true,
		RetryConfig:  DefaultRetryConfig(),
	}
}

// Update NewClient
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.RetryConfig == nil {
		cfg.RetryConfig = DefaultRetryConfig()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     cfg.TLSConfig,
		ForceAttemptHTTP2:   cfg.EnableHTTP2,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		userAgent:   cfg.UserAgent,
		timeout:     cfg.Timeout,
		retryConfig: cfg.RetryConfig,
	}
}

// DoWithRetry executes an HTTP request with retry logic
func (c *Client) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Clone request for retry (body may have been consumed)
		reqClone := req.Clone(ctx)
		if req.Header.Get("User-Agent") == "" {
			reqClone.Header.Set("User-Agent", c.userAgent)
		}

		resp, lastErr = c.httpClient.Do(reqClone)

		// Success
		if lastErr == nil && !IsRetriableStatus(resp.StatusCode) {
			return resp, nil
		}

		// Check if error is retriable
		if lastErr != nil && !IsRetriable(lastErr) {
			return nil, lastErr
		}

		// Check if status is retriable
		if resp != nil && !IsRetriableStatus(resp.StatusCode) {
			return resp, nil
		}

		// Don't sleep after last attempt
		if attempt < c.retryConfig.MaxRetries {
			// Close response body before retry
			if resp != nil {
				resp.Body.Close()
			}

			backoff := c.retryConfig.CalculateBackoff(attempt)

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
	}

	return resp, nil
}
```

**Step 3: Add retry options**

Add to `http/client.go`:

```go
// WithRetryConfig sets custom retry configuration
func WithRetryConfig(retryCfg *RetryConfig) Option {
	return func(cfg *Config) {
		cfg.RetryConfig = retryCfg
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(n int) Option {
	return func(cfg *Config) {
		if cfg.RetryConfig == nil {
			cfg.RetryConfig = DefaultRetryConfig()
		}
		cfg.RetryConfig.MaxRetries = n
	}
}
```

### Verification Steps

```bash
# Build
go build ./http

# Format check
gofmt -l http/
```

### Testing

Create `http/retry_test.go`:

```go
package http

import (
	"context"
	"errors"
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
```

Add import to test file:

```go
import (
	"fmt"
	// ... other imports
)
```

Run tests:

```bash
go test ./http -v -run TestRetry
go test ./http -v
```

### Commit

```
feat: add HTTP retry logic with exponential backoff

- Implement exponential backoff with jitter
- Retry on 429, 503, 504 status codes
- Retry on network errors (timeout, connection reset)
- Configurable max retries and backoff parameters
- Add comprehensive retry tests

Chunk: M2.2
Status: ✓ Complete
```

---

## [M2.3] HTTP Client - Retry-After Header

**Time Estimate:** 2 hours
**Dependencies:** M2.2 (Retry logic)
**Status:** Not started

### What You'll Build

Parse and respect the Retry-After header for rate limiting (429) and service unavailable (503) responses.

### Step-by-Step Instructions

**Step 1: Create Retry-After parser**

Add to `http/retry.go`:

```go
import (
	"strconv"
	"strings"
	"time"
)

// ParseRetryAfter parses the Retry-After header value
// Returns duration to wait, or 0 if header is invalid/missing
// Supports both delay-seconds (int) and HTTP-date formats
func ParseRetryAfter(headerValue string) time.Duration {
	if headerValue == "" {
		return 0
	}

	// Try parsing as delay-seconds (integer)
	if seconds, err := strconv.Atoi(strings.TrimSpace(headerValue)); err == nil {
		if seconds < 0 {
			return 0
		}
		// Cap at 5 minutes for safety
		if seconds > 300 {
			seconds = 300
		}
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC1123, RFC850, ANSI C)
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC850,
		time.ANSIC,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, strings.TrimSpace(headerValue)); err == nil {
			duration := time.Until(t)
			if duration < 0 {
				return 0
			}
			// Cap at 5 minutes for safety
			if duration > 5*time.Minute {
				duration = 5 * time.Minute
			}
			return duration
		}
	}

	return 0
}
```

**Step 2: Update retry logic to use Retry-After**

Update `DoWithRetry` in `http/client.go`:

```go
// DoWithRetry executes an HTTP request with retry logic
func (c *Client) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Clone request for retry (body may have been consumed)
		reqClone := req.Clone(ctx)
		if req.Header.Get("User-Agent") == "" {
			reqClone.Header.Set("User-Agent", c.userAgent)
		}

		resp, lastErr = c.httpClient.Do(reqClone)

		// Success
		if lastErr == nil && !IsRetriableStatus(resp.StatusCode) {
			return resp, nil
		}

		// Check if error is retriable
		if lastErr != nil && !IsRetriable(lastErr) {
			return nil, lastErr
		}

		// Check if status is retriable
		if resp != nil && !IsRetriableStatus(resp.StatusCode) {
			return resp, nil
		}

		// Don't sleep after last attempt
		if attempt < c.retryConfig.MaxRetries {
			var backoff time.Duration

			// Check for Retry-After header
			if resp != nil {
				retryAfter := ParseRetryAfter(resp.Header.Get("Retry-After"))
				if retryAfter > 0 {
					backoff = retryAfter
				}
			}

			// Fall back to exponential backoff if no Retry-After
			if backoff == 0 {
				backoff = c.retryConfig.CalculateBackoff(attempt)
			}

			// Close response body before retry
			if resp != nil {
				resp.Body.Close()
			}

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
	}

	return resp, nil
}
```

### Verification Steps

```bash
# Build
go build ./http

# Format check
gofmt -l http/
```

### Testing

Add to `http/retry_test.go`:

```go
func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		wantMin     time.Duration
		wantMax     time.Duration
	}{
		{
			name:        "empty",
			headerValue: "",
			wantMin:     0,
			wantMax:     0,
		},
		{
			name:        "delay-seconds",
			headerValue: "120",
			wantMin:     120 * time.Second,
			wantMax:     120 * time.Second,
		},
		{
			name:        "delay-seconds with whitespace",
			headerValue: "  60  ",
			wantMin:     60 * time.Second,
			wantMax:     60 * time.Second,
		},
		{
			name:        "negative seconds",
			headerValue: "-10",
			wantMin:     0,
			wantMax:     0,
		},
		{
			name:        "capped at 5 minutes",
			headerValue: "600",
			wantMin:     300 * time.Second,
			wantMax:     300 * time.Second,
		},
		{
			name:        "HTTP-date RFC1123",
			headerValue: time.Now().Add(30 * time.Second).Format(time.RFC1123),
			wantMin:     29 * time.Second,
			wantMax:     31 * time.Second,
		},
		{
			name:        "HTTP-date in past",
			headerValue: time.Now().Add(-30 * time.Second).Format(time.RFC1123),
			wantMin:     0,
			wantMax:     0,
		},
		{
			name:        "invalid format",
			headerValue: "invalid",
			wantMin:     0,
			wantMax:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRetryAfter(tt.headerValue)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("ParseRetryAfter(%q) = %v, want between %v and %v",
					tt.headerValue, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestClient_DoWithRetry_RetryAfterHeader(t *testing.T) {
	var attempts int32
	start := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
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

	elapsed := time.Since(start)

	// Should have waited at least 1 second due to Retry-After
	if elapsed < 1*time.Second {
		t.Errorf("elapsed = %v, want >= 1s", elapsed)
	}

	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestClient_DoWithRetry_RetryAfterHTTPDate(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 2 {
			retryTime := time.Now().Add(500 * time.Millisecond)
			w.Header().Set("Retry-After", retryTime.Format(time.RFC1123))
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewClient(nil)
	ctx := context.Background()

	req, _ := http.NewRequest("GET", server.URL, nil)
	start := time.Now()
	resp, err := client.DoWithRetry(ctx, req)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)

	// Should have waited at least 500ms
	if elapsed < 400*time.Millisecond {
		t.Errorf("elapsed = %v, want >= 400ms", elapsed)
	}
}
```

Run tests:

```bash
go test ./http -v -run TestRetryAfter
go test ./http -v
```

### Commit

```
feat: add Retry-After header parsing and support

- Parse Retry-After in delay-seconds and HTTP-date formats
- Use Retry-After duration before falling back to exponential backoff
- Cap Retry-After at 5 minutes for safety
- Add comprehensive tests for header parsing

Chunk: M2.3
Status: ✓ Complete
```

---

## [M2.4] Protocol v3 - Service Index

**Time Estimate:** 3 hours
**Dependencies:** M2.3 (HTTP client), M1.12 (Package identity)
**Status:** Not started

### What You'll Build

Implement NuGet v3 service index discovery with caching. The service index provides resource URLs for search, metadata, download, etc.

### Step-by-Step Instructions

**Step 1: Create v3 protocol types**

Create `protocol/v3/types.go`:

```go
package v3

import (
	"time"
)

// ServiceIndex represents the NuGet v3 service index
// See: https://docs.microsoft.com/en-us/nuget/api/service-index
type ServiceIndex struct {
	Version   string      `json:"version"`
	Resources []Resource  `json:"resources"`
	Context   interface{} `json:"@context,omitempty"`
}

// Resource represents a service resource in the service index
type Resource struct {
	ID      string `json:"@id"`
	Type    string `json:"@type"`
	Comment string `json:"comment,omitempty"`
}

// Well-known resource types
const (
	// Search
	ResourceTypeSearchQueryService      = "SearchQueryService"
	ResourceTypeSearchAutocompleteService = "SearchAutocompleteService"

	// Registration (metadata)
	ResourceTypeRegistrationsBaseUrl = "RegistrationsBaseUrl"

	// Package download
	ResourceTypePackageBaseAddress = "PackageBaseAddress"

	// Package publish
	ResourceTypePackagePublish = "PackagePublish"

	// Catalog
	ResourceTypeCatalog = "Catalog/3.0.0"
)

// Default service index cache TTL (40 minutes as per NuGet spec)
const ServiceIndexCacheTTL = 40 * time.Minute
```

**Step 2: Create service index client**

Create `protocol/v3/service_index.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	nugethttp "github.com/yourusername/gonuget/http"
)

// ServiceIndexClient provides access to NuGet v3 service index
type ServiceIndexClient struct {
	httpClient *nugethttp.Client

	mu    sync.RWMutex
	cache map[string]*cachedServiceIndex
}

type cachedServiceIndex struct {
	index     *ServiceIndex
	expiresAt time.Time
}

// NewServiceIndexClient creates a new service index client
func NewServiceIndexClient(httpClient *nugethttp.Client) *ServiceIndexClient {
	return &ServiceIndexClient{
		httpClient: httpClient,
		cache:      make(map[string]*cachedServiceIndex),
	}
}

// GetServiceIndex retrieves the service index for a given source URL
// Caches the result for ServiceIndexCacheTTL
func (c *ServiceIndexClient) GetServiceIndex(ctx context.Context, sourceURL string) (*ServiceIndex, error) {
	// Check cache
	c.mu.RLock()
	cached, ok := c.cache[sourceURL]
	c.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached.index, nil
	}

	// Fetch from server
	index, err := c.fetchServiceIndex(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.cache[sourceURL] = &cachedServiceIndex{
		index:     index,
		expiresAt: time.Now().Add(ServiceIndexCacheTTL),
	}
	c.mu.Unlock()

	return index, nil
}

func (c *ServiceIndexClient) fetchServiceIndex(ctx context.Context, sourceURL string) (*ServiceIndex, error) {
	// Ensure URL ends with /index.json
	indexURL := sourceURL
	if len(indexURL) > 0 && indexURL[len(indexURL)-1] != '/' {
		indexURL += "/"
	}
	indexURL += "index.json"

	resp, err := c.httpClient.DoWithRetry(ctx, mustNewRequest("GET", indexURL, nil))
	if err != nil {
		return nil, fmt.Errorf("fetch service index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("service index returned %d: %s", resp.StatusCode, body)
	}

	var index ServiceIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("decode service index: %w", err)
	}

	return &index, nil
}

// GetResourceURL finds the first resource of the given type
func (c *ServiceIndexClient) GetResourceURL(ctx context.Context, sourceURL, resourceType string) (string, error) {
	index, err := c.GetServiceIndex(ctx, sourceURL)
	if err != nil {
		return "", err
	}

	for _, resource := range index.Resources {
		if resource.Type == resourceType {
			return resource.ID, nil
		}
	}

	return "", fmt.Errorf("resource type %q not found in service index", resourceType)
}

// GetAllResourceURLs finds all resources of the given type
func (c *ServiceIndexClient) GetAllResourceURLs(ctx context.Context, sourceURL, resourceType string) ([]string, error) {
	index, err := c.GetServiceIndex(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, resource := range index.Resources {
		if resource.Type == resourceType {
			urls = append(urls, resource.ID)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("resource type %q not found in service index", resourceType)
	}

	return urls, nil
}

// ClearCache removes all cached service indexes
func (c *ServiceIndexClient) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]*cachedServiceIndex)
	c.mu.Unlock()
}

// Helper to create request (panics on error for cleaner code)
func mustNewRequest(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("invalid request: %v", err))
	}
	return req
}
```

Add import:

```go
import (
	"net/http"
)
```

### Verification Steps

```bash
# Build
go build ./protocol/v3

# Format check
gofmt -l protocol/
```

### Testing

Create `protocol/v3/service_index_test.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	nugethttp "github.com/yourusername/gonuget/http"
)

var testServiceIndex = &ServiceIndex{
	Version: "3.0.0",
	Resources: []Resource{
		{
			ID:   "https://api.nuget.org/v3/registration5-gz-semver2/",
			Type: ResourceTypeRegistrationsBaseUrl,
		},
		{
			ID:   "https://api.nuget.org/v3-flatcontainer/",
			Type: ResourceTypePackageBaseAddress,
		},
		{
			ID:   "https://azuresearch-usnc.nuget.org/query",
			Type: ResourceTypeSearchQueryService,
		},
		{
			ID:   "https://azuresearch-usnc.nuget.org/autocomplete",
			Type: ResourceTypeSearchAutocompleteService,
		},
	},
}

func TestServiceIndexClient_GetServiceIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			t.Errorf("Path = %q, want /index.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	index, err := client.GetServiceIndex(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if index.Version != "3.0.0" {
		t.Errorf("Version = %q, want 3.0.0", index.Version)
	}

	if len(index.Resources) != 4 {
		t.Errorf("Resources count = %d, want 4", len(index.Resources))
	}
}

func TestServiceIndexClient_Cache(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	// First call - should hit server
	_, err := client.GetServiceIndex(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call - should use cache
	_, err = client.GetServiceIndex(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cache should be used)", callCount)
	}
}

func TestServiceIndexClient_CacheExpiration(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	// First call
	_, err := client.GetServiceIndex(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	// Manually expire cache
	client.mu.Lock()
	for k := range client.cache {
		client.cache[k].expiresAt = time.Now().Add(-1 * time.Second)
	}
	client.mu.Unlock()

	// Second call - cache expired, should hit server again
	_, err = client.GetServiceIndex(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (cache expired)", callCount)
	}
}

func TestServiceIndexClient_GetResourceURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	tests := []struct {
		resourceType string
		want         string
		wantErr      bool
	}{
		{
			resourceType: ResourceTypeSearchQueryService,
			want:         "https://azuresearch-usnc.nuget.org/query",
			wantErr:      false,
		},
		{
			resourceType: ResourceTypePackageBaseAddress,
			want:         "https://api.nuget.org/v3-flatcontainer/",
			wantErr:      false,
		},
		{
			resourceType: "NonExistentType",
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got, err := client.GetResourceURL(ctx, server.URL, tt.resourceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetResourceURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetResourceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceIndexClient_GetAllResourceURLs(t *testing.T) {
	multiResourceIndex := &ServiceIndex{
		Version: "3.0.0",
		Resources: []Resource{
			{
				ID:   "https://search1.nuget.org/query",
				Type: ResourceTypeSearchQueryService,
			},
			{
				ID:   "https://search2.nuget.org/query",
				Type: ResourceTypeSearchQueryService,
			},
			{
				ID:   "https://api.nuget.org/v3-flatcontainer/",
				Type: ResourceTypePackageBaseAddress,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(multiResourceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	urls, err := client.GetAllResourceURLs(ctx, server.URL, ResourceTypeSearchQueryService)
	if err != nil {
		t.Fatalf("GetAllResourceURLs() error = %v", err)
	}

	if len(urls) != 2 {
		t.Errorf("len(urls) = %d, want 2", len(urls))
	}

	expected := map[string]bool{
		"https://search1.nuget.org/query": true,
		"https://search2.nuget.org/query": true,
	}

	for _, url := range urls {
		if !expected[url] {
			t.Errorf("unexpected URL: %q", url)
		}
	}
}

func TestServiceIndexClient_ClearCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	// Populate cache
	_, err := client.GetServiceIndex(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if len(client.cache) == 0 {
		t.Error("cache should not be empty")
	}

	// Clear cache
	client.ClearCache()

	if len(client.cache) != 0 {
		t.Errorf("cache size = %d, want 0 after clear", len(client.cache))
	}
}
```

Run tests:

```bash
go test ./protocol/v3 -v
```

### Commit

```
feat: implement NuGet v3 service index client

- Add ServiceIndex and Resource types
- Implement service index discovery with HTTP client
- Cache service index for 40 minutes (per NuGet spec)
- Add GetResourceURL and GetAllResourceURLs helpers
- Support cache clearing
- Add comprehensive tests

Chunk: M2.4
Status: ✓ Complete
```

---

*Due to the length constraints, I'll create the remaining chunks M2.5 through M2.18 in a continuation file. Would you like me to proceed with creating IMPL-M2-PROTOCOL-CONTINUED.md?*

Actually, let me check the file size first and continue in the same file if possible.

Let me continue with the remaining chunks in this file.

---

## [M2.5] Protocol v3 - Search

**Time Estimate:** 4 hours
**Dependencies:** M2.4 (Service index), M1.2 (Version types)
**Status:** Not started

### What You'll Build

Implement NuGet v3 package search with pagination and filtering.

### Step-by-Step Instructions

**Step 1: Create search types**

Add to `protocol/v3/types.go`:

```go
// SearchResponse represents the response from the search API
type SearchResponse struct {
	TotalHits int             `json:"totalHits"`
	Data      []SearchResult  `json:"data"`
	Context   interface{}     `json:"@context,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ID             string          `json:"@id"`
	Type           string          `json:"@type"`
	Registration   string          `json:"registration,omitempty"`
	PackageID      string          `json:"id"`
	Version        string          `json:"version"`
	Description    string          `json:"description"`
	Summary        string          `json:"summary,omitempty"`
	Title          string          `json:"title,omitempty"`
	IconURL        string          `json:"iconUrl,omitempty"`
	LicenseURL     string          `json:"licenseUrl,omitempty"`
	ProjectURL     string          `json:"projectUrl,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Authors        []string        `json:"authors,omitempty"`
	TotalDownloads int64           `json:"totalDownloads"`
	Verified       bool            `json:"verified"`
	Versions       []SearchVersion `json:"versions,omitempty"`
}

// SearchVersion represents a version in search results
type SearchVersion struct {
	Version   string `json:"version"`
	Downloads int64  `json:"downloads"`
	ID        string `json:"@id"`
}
```

**Step 2: Create search client**

Create `protocol/v3/search.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	nugethttp "github.com/yourusername/gonuget/http"
)

// SearchClient provides package search functionality
type SearchClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// SearchOptions holds search parameters
type SearchOptions struct {
	Query          string
	Skip           int
	Take           int
	Prerelease     bool
	SemVerLevel    string // "2.0.0" for SemVer 2.0 support
}

// NewSearchClient creates a new search client
func NewSearchClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *SearchClient {
	return &SearchClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// Search searches for packages matching the query
func (c *SearchClient) Search(ctx context.Context, sourceURL string, opts SearchOptions) (*SearchResponse, error) {
	// Get search endpoint from service index
	searchURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeSearchQueryService)
	if err != nil {
		return nil, fmt.Errorf("get search URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	if opts.Query != "" {
		params.Set("q", opts.Query)
	}
	if opts.Skip > 0 {
		params.Set("skip", strconv.Itoa(opts.Skip))
	}
	if opts.Take > 0 {
		params.Set("take", strconv.Itoa(opts.Take))
	} else {
		params.Set("take", "20") // Default
	}
	params.Set("prerelease", strconv.FormatBool(opts.Prerelease))
	if opts.SemVerLevel != "" {
		params.Set("semVerLevel", opts.SemVerLevel)
	}

	// Build full URL
	fullURL := searchURL
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	// Execute request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, body)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	return &searchResp, nil
}

// SearchSimple performs a simple search with default options
func (c *SearchClient) SearchSimple(ctx context.Context, sourceURL, query string) (*SearchResponse, error) {
	return c.Search(ctx, sourceURL, SearchOptions{
		Query:      query,
		Take:       20,
		Prerelease: true,
	})
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v3

# Format check
gofmt -l protocol/v3/
```

### Testing

Create `protocol/v3/search_test.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/yourusername/gonuget/http"
)

var testSearchResponse = &SearchResponse{
	TotalHits: 2,
	Data: []SearchResult{
		{
			PackageID:      "Newtonsoft.Json",
			Version:        "13.0.3",
			Description:    "Json.NET is a popular high-performance JSON framework for .NET",
			Authors:        []string{"James Newton-King"},
			TotalDownloads: 1000000000,
			Verified:       true,
			Tags:           []string{"json", "serialization"},
			Versions: []SearchVersion{
				{Version: "13.0.3", Downloads: 50000000},
				{Version: "13.0.2", Downloads: 45000000},
			},
		},
		{
			PackageID:      "Newtonsoft.Json.Bson",
			Version:        "1.0.2",
			Description:    "Json.NET BSON adds support for reading and writing BSON",
			Authors:        []string{"James Newton-King"},
			TotalDownloads: 10000000,
			Verified:       false,
		},
	},
}

func setupSearchServer(t *testing.T) (*httptest.Server, *SearchClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/search",
					Type: ResourceTypeSearchQueryService,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})

	// Search endpoint
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// Validate query parameters
		if q := query.Get("q"); q == "" {
			t.Error("expected 'q' parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testSearchResponse)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	searchClient := NewSearchClient(httpClient, serviceIndexClient)

	return server, searchClient
}

func TestSearchClient_Search(t *testing.T) {
	server, client := setupSearchServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.Search(ctx, server.URL, SearchOptions{
		Query:      "newtonsoft",
		Take:       20,
		Prerelease: true,
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if resp.TotalHits != 2 {
		t.Errorf("TotalHits = %d, want 2", resp.TotalHits)
	}

	if len(resp.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2", len(resp.Data))
	}

	first := resp.Data[0]
	if first.PackageID != "Newtonsoft.Json" {
		t.Errorf("PackageID = %q, want Newtonsoft.Json", first.PackageID)
	}

	if first.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", first.Version)
	}

	if !first.Verified {
		t.Error("Verified = false, want true")
	}

	if len(first.Versions) != 2 {
		t.Errorf("len(Versions) = %d, want 2", len(first.Versions))
	}
}

func TestSearchClient_SearchWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			index := &ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/search",
						Type: ResourceTypeSearchQueryService,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
			return
		}

		query := r.URL.Query()

		// Validate pagination parameters
		skip := query.Get("skip")
		if skip != "10" {
			t.Errorf("skip = %q, want 10", skip)
		}

		take := query.Get("take")
		if take != "5" {
			t.Errorf("take = %q, want 5", take)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&SearchResponse{TotalHits: 100, Data: []SearchResult{}})
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewSearchClient(httpClient, serviceIndexClient)

	ctx := context.Background()

	_, err := client.Search(ctx, server.URL, SearchOptions{
		Query: "test",
		Skip:  10,
		Take:  5,
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}

func TestSearchClient_SearchSimple(t *testing.T) {
	server, client := setupSearchServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.SearchSimple(ctx, server.URL, "newtonsoft")
	if err != nil {
		t.Fatalf("SearchSimple() error = %v", err)
	}

	if resp.TotalHits != 2 {
		t.Errorf("TotalHits = %d, want 2", resp.TotalHits)
	}
}
```

Run tests:

```bash
go test ./protocol/v3 -v -run TestSearch
```

### Commit

```
feat: implement NuGet v3 package search

- Add SearchResponse and SearchResult types
- Implement search with pagination (skip/take)
- Support prerelease and SemVer level filtering
- Add SearchSimple helper for common use case
- Create comprehensive search tests

Chunk: M2.5
Status: ✓ Complete
```

---
