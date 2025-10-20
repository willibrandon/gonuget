# gonuget HTTP Client Design

**Component**: `pkg/gonuget/http/`
**Version**: 1.0.0
**Status**: Draft

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [HTTP Client](#http-client)
4. [Retry Logic](#retry-logic)
5. [Caching](#caching)
6. [Circuit Breaker](#circuit-breaker)
7. [Rate Limiting](#rate-limiting)
8. [Progress Tracking](#progress-tracking)
9. [Authentication](#authentication)
10. [Middleware Chain](#middleware-chain)
11. [Implementation Details](#implementation-details)

---

## Overview

The HTTP client is the foundation of gonuget, responsible for all network communication with NuGet feeds. It must be:

- **Resilient**: Automatic retry with backoff, circuit breakers
- **Fast**: HTTP/2 and HTTP/3 support, connection pooling, caching
- **Observable**: Logging, metrics, distributed tracing
- **Secure**: HTTPS enforcement, credential management
- **Respectful**: Rate limiting, Retry-After header support

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Code                          │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                         HTTPClient                               │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Middleware Chain                        │  │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ │  │
│  │  │ Logger │→│  Auth  │→│ Retry  │→│ Cache  │→│Circuit │ │  │
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘ │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                  net/http.Client (Go stdlib)                     │
│                  - HTTP/2 support                                │
│                  - Connection pooling                            │
│                  - TLS configuration                             │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
                            Network
```

### Component Relationships

```go
type HTTPClient struct {
    client        *http.Client       // Underlying Go HTTP client
    retry         *RetryHandler      // Retry logic
    cache         *ResponseCache     // HTTP response cache
    circuit       *CircuitBreaker    // Circuit breaker
    rateLimit     *RateLimiter       // Rate limiter
    auth          *AuthHandler       // Authentication
    logger        Logger             // Structured logging
    tracer        trace.Tracer       // Distributed tracing
    middleware    []Middleware       // Middleware chain
}
```

---

## HTTP Client

### Core Implementation

**File**: `pkg/gonuget/http/client.go`

```go
package http

import (
    "context"
    "crypto/tls"
    "net"
    "net/http"
    "time"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "golang.org/x/net/http2"
)

type HTTPClient struct {
    client        *http.Client
    retry         *RetryHandler
    cache         *ResponseCache
    circuit       *CircuitBreaker
    rateLimit     *RateLimiter
    auth          auth.Provider
    logger        Logger
    tracer        trace.Tracer
    userAgent     string
    timeout       time.Duration
}

type Config struct {
    // Connection
    MaxIdleConns        int           // Default: 100
    MaxIdleConnsPerHost int           // Default: 10
    IdleConnTimeout     time.Duration // Default: 90s
    Timeout             time.Duration // Default: 30s
    KeepAlive           time.Duration // Default: 30s

    // TLS
    TLSHandshakeTimeout time.Duration      // Default: 10s
    TLSConfig           *tls.Config        // Custom TLS config
    InsecureSkipVerify  bool               // Default: false (NEVER use in production)

    // HTTP/2 and HTTP/3
    EnableHTTP2         bool               // Default: true
    EnableHTTP3         bool               // Default: false (experimental)

    // Retry
    RetryConfig         *RetryConfig

    // Cache
    CacheConfig         *CacheConfig

    // Circuit Breaker
    CircuitConfig       *CircuitConfig

    // Rate Limiting
    RateLimitConfig     *RateLimitConfig

    // Auth
    Auth                auth.Provider

    // Observability
    Logger              Logger
    Tracer              trace.Tracer
    UserAgent           string
}

func NewHTTPClient(cfg *Config) *HTTPClient {
    // Create transport with optimized settings
    transport := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: cfg.KeepAlive,
        }).DialContext,
        MaxIdleConns:          cfg.MaxIdleConns,
        MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
        IdleConnTimeout:       cfg.IdleConnTimeout,
        TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
        ExpectContinueTimeout: 1 * time.Second,
        TLSClientConfig:       cfg.TLSConfig,
    }

    // Enable HTTP/2
    if cfg.EnableHTTP2 {
        if err := http2.ConfigureTransport(transport); err != nil {
            cfg.Logger.Warn("Failed to configure HTTP/2: {Error}", err)
        }
    }

    // Wrap with OpenTelemetry instrumentation
    otelTransport := otelhttp.NewTransport(transport)

    client := &http.Client{
        Transport: otelTransport,
        Timeout:   cfg.Timeout,
    }

    return &HTTPClient{
        client:    client,
        retry:     NewRetryHandler(cfg.RetryConfig),
        cache:     NewResponseCache(cfg.CacheConfig),
        circuit:   NewCircuitBreaker(cfg.CircuitConfig),
        rateLimit: NewRateLimiter(cfg.RateLimitConfig),
        auth:      cfg.Auth,
        logger:    cfg.Logger,
        tracer:    cfg.Tracer,
        userAgent: cfg.UserAgent,
        timeout:   cfg.Timeout,
    }
}

func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Set User-Agent
    if c.userAgent != "" {
        req.Header.Set("User-Agent", c.userAgent)
    }

    // Apply middleware chain
    handler := c.executeRequest
    for i := len(c.middleware) - 1; i >= 0; i-- {
        handler = c.middleware[i](handler)
    }

    return handler(ctx, req)
}

func (c *HTTPClient) executeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
    // 1. Check circuit breaker
    if !c.circuit.AllowRequest() {
        return nil, ErrCircuitOpen
    }

    // 2. Apply authentication
    if c.auth != nil {
        if err := c.auth.ApplyAuth(ctx, req); err != nil {
            return nil, fmt.Errorf("auth failed: %w", err)
        }
    }

    // 3. Check cache
    if req.Method == "GET" {
        if cached, ok := c.cache.Get(ctx, req); ok {
            c.logger.Debug("Cache hit for {URL}", req.URL)
            return cached, nil
        }
    }

    // 4. Apply rate limiting
    if err := c.rateLimit.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }

    // 5. Execute with retry
    resp, err := c.retry.Do(ctx, req, func(ctx context.Context, req *http.Request) (*http.Response, error) {
        // Create span for this request
        ctx, span := c.tracer.Start(ctx, "http.request",
            trace.WithSpanKind(trace.SpanKindClient),
            trace.WithAttributes(
                attribute.String("http.method", req.Method),
                attribute.String("http.url", req.URL.String()),
            ),
        )
        defer span.End()

        // Execute request
        start := time.Now()
        resp, err := c.client.Do(req.WithContext(ctx))
        duration := time.Since(start)

        // Log request
        if err != nil {
            c.logger.Error("HTTP request failed: {Method} {URL} ({Duration}ms): {Error}",
                req.Method, req.URL, duration.Milliseconds(), err)
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        } else {
            c.logger.Debug("HTTP request: {Method} {URL} {StatusCode} ({Duration}ms)",
                req.Method, req.URL, resp.StatusCode, duration.Milliseconds())
            span.SetAttributes(
                attribute.Int("http.status_code", resp.StatusCode),
                attribute.Int64("http.response_size", resp.ContentLength),
            )
        }

        return resp, err
    })

    // 6. Record circuit breaker result
    if err != nil {
        c.circuit.RecordFailure()
    } else {
        c.circuit.RecordSuccess()
    }

    // 7. Cache successful GET responses
    if err == nil && req.Method == "GET" && c.isCacheable(resp) {
        c.cache.Put(ctx, req, resp)
    }

    return resp, err
}

func (c *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    return c.Do(ctx, req)
}

func (c *HTTPClient) isCacheable(resp *http.Response) bool {
    // Only cache successful responses
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return false
    }

    // Respect Cache-Control headers
    cacheControl := resp.Header.Get("Cache-Control")
    if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
        return false
    }

    return true
}
```

---

## Retry Logic

### Retry Strategy

**File**: `pkg/gonuget/http/retry.go`

```go
package http

import (
    "context"
    "fmt"
    "math"
    "math/rand"
    "net/http"
    "strconv"
    "time"
)

type RetryConfig struct {
    MaxAttempts  int           // Default: 3
    InitialDelay time.Duration // Default: 100ms
    MaxDelay     time.Duration // Default: 10s
    Multiplier   float64       // Default: 2.0
    Jitter       float64       // Default: 0.1 (10%)
    RetryOn      []int         // HTTP status codes to retry on
}

var DefaultRetryConfig = &RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    Jitter:       0.1,
    RetryOn:      []int{408, 429, 500, 502, 503, 504},
}

type RetryHandler struct {
    config *RetryConfig
    logger Logger
}

func NewRetryHandler(cfg *RetryConfig) *RetryHandler {
    if cfg == nil {
        cfg = DefaultRetryConfig
    }
    return &RetryHandler{config: cfg}
}

func (h *RetryHandler) Do(ctx context.Context, req *http.Request, fn func(context.Context, *http.Request) (*http.Response, error)) (*http.Response, error) {
    var lastErr error
    var resp *http.Response

    for attempt := 0; attempt < h.config.MaxAttempts; attempt++ {
        // Execute request
        resp, lastErr = fn(ctx, req)

        // Success - return immediately
        if lastErr == nil && !h.shouldRetry(resp.StatusCode) {
            return resp, nil
        }

        // Last attempt - don't delay, just return
        if attempt == h.config.MaxAttempts-1 {
            break
        }

        // Calculate delay
        delay := h.calculateDelay(attempt, resp)

        // Log retry
        if lastErr != nil {
            h.logger.Warn("Request failed, retrying in {Delay}ms (attempt {Attempt}/{MaxAttempts}): {Error}",
                delay.Milliseconds(), attempt+1, h.config.MaxAttempts, lastErr)
        } else {
            h.logger.Warn("Request returned {StatusCode}, retrying in {Delay}ms (attempt {Attempt}/{MaxAttempts})",
                resp.StatusCode, delay.Milliseconds(), attempt+1, h.config.MaxAttempts)
        }

        // Wait with cancellation support
        select {
        case <-time.After(delay):
            // Continue to next attempt
        case <-ctx.Done():
            return nil, ctx.Err()
        }

        // Drain and close response body to allow connection reuse
        if resp != nil && resp.Body != nil {
            io.Copy(io.Discard, resp.Body)
            resp.Body.Close()
        }
    }

    // All attempts failed
    if lastErr != nil {
        return nil, fmt.Errorf("request failed after %d attempts: %w", h.config.MaxAttempts, lastErr)
    }
    return resp, nil
}

func (h *RetryHandler) shouldRetry(statusCode int) bool {
    for _, code := range h.config.RetryOn {
        if statusCode == code {
            return true
        }
    }
    return false
}

func (h *RetryHandler) calculateDelay(attempt int, resp *http.Response) time.Duration {
    // Check for Retry-After header
    if resp != nil {
        if delay := h.parseRetryAfter(resp.Header.Get("Retry-After")); delay > 0 {
            return delay
        }
    }

    // Exponential backoff: initialDelay * multiplier^attempt
    delay := float64(h.config.InitialDelay) * math.Pow(h.config.Multiplier, float64(attempt))

    // Add jitter: delay * (1 ± jitter)
    jitter := delay * h.config.Jitter
    delay = delay + (rand.Float64()*2-1)*jitter

    // Cap at max delay
    if delay > float64(h.config.MaxDelay) {
        delay = float64(h.config.MaxDelay)
    }

    return time.Duration(delay)
}

func (h *RetryHandler) parseRetryAfter(header string) time.Duration {
    if header == "" {
        return 0
    }

    // Try parsing as seconds
    if seconds, err := strconv.Atoi(header); err == nil {
        return time.Duration(seconds) * time.Second
    }

    // Try parsing as HTTP date
    if t, err := http.ParseTime(header); err == nil {
        duration := time.Until(t)
        if duration > 0 {
            return duration
        }
    }

    return 0
}
```

### Retry Decision Tree

```
Request Failed
      │
      ├─ Network Error (timeout, connection refused, etc.)
      │  └─ Retry: YES (transient error)
      │
      ├─ HTTP 408 (Request Timeout)
      │  └─ Retry: YES
      │
      ├─ HTTP 429 (Too Many Requests)
      │  ├─ Has Retry-After header
      │  │  └─ Wait for Retry-After duration, then retry
      │  └─ No Retry-After header
      │     └─ Exponential backoff, then retry
      │
      ├─ HTTP 500 (Internal Server Error)
      │  └─ Retry: YES (server error, might be transient)
      │
      ├─ HTTP 502 (Bad Gateway)
      │  └─ Retry: YES (gateway error, might be transient)
      │
      ├─ HTTP 503 (Service Unavailable)
      │  ├─ Has Retry-After header
      │  │  └─ Wait for Retry-After duration, then retry
      │  └─ No Retry-After header
      │     └─ Exponential backoff, then retry
      │
      ├─ HTTP 504 (Gateway Timeout)
      │  └─ Retry: YES (timeout, might succeed on retry)
      │
      ├─ HTTP 401 (Unauthorized)
      │  └─ Retry: NO (auth required, retry won't help)
      │
      ├─ HTTP 403 (Forbidden)
      │  └─ Retry: NO (access denied, retry won't help)
      │
      ├─ HTTP 404 (Not Found)
      │  └─ Retry: NO (resource doesn't exist, retry won't help)
      │
      └─ Other status codes
         └─ Retry: NO
```

---

## Caching

### Cache Architecture

**File**: `pkg/gonuget/http/cache.go`

```go
package http

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

type CacheConfig struct {
    Enabled        bool          // Default: true
    MemorySizeMB   int           // Default: 100MB
    DiskSizeMB     int           // Default: 1GB
    TTL            time.Duration // Default: 1 hour
    CacheDir       string        // Default: ~/.cache/gonuget/http
}

type ResponseCache struct {
    config      *CacheConfig
    memoryCache *MemoryCache
    diskCache   *DiskCache
    stats       *CacheStats
    mu          sync.RWMutex
}

type CachedResponse struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
    Timestamp  time.Time
    ETag       string
    Expiry     time.Time
}

type CacheStats struct {
    Hits        int64
    Misses      int64
    Evictions   int64
    BytesServed int64
}

func NewResponseCache(cfg *CacheConfig) *ResponseCache {
    return &ResponseCache{
        config:      cfg,
        memoryCache: NewMemoryCache(cfg.MemorySizeMB * 1024 * 1024),
        diskCache:   NewDiskCache(cfg.CacheDir, cfg.DiskSizeMB*1024*1024),
        stats:       &CacheStats{},
    }
}

func (c *ResponseCache) Get(ctx context.Context, req *http.Request) (*http.Response, bool) {
    if !c.config.Enabled || req.Method != "GET" {
        return nil, false
    }

    key := c.cacheKey(req)

    // Try memory cache first (fast)
    c.mu.RLock()
    if cached, ok := c.memoryCache.Get(key); ok {
        if !cached.Expired() {
            c.mu.RUnlock()
            atomic.AddInt64(&c.stats.Hits, 1)
            atomic.AddInt64(&c.stats.BytesServed, int64(len(cached.Body)))
            return cached.ToHTTPResponse(), true
        }
    }
    c.mu.RUnlock()

    // Try disk cache (slower)
    if cached, ok := c.diskCache.Get(key); ok {
        if !cached.Expired() {
            // Promote to memory cache
            c.mu.Lock()
            c.memoryCache.Put(key, cached)
            c.mu.Unlock()

            atomic.AddInt64(&c.stats.Hits, 1)
            atomic.AddInt64(&c.stats.BytesServed, int64(len(cached.Body)))
            return cached.ToHTTPResponse(), true
        }
    }

    // Cache miss
    atomic.AddInt64(&c.stats.Misses, 1)
    return nil, false
}

func (c *ResponseCache) Put(ctx context.Context, req *http.Request, resp *http.Response) {
    if !c.config.Enabled || req.Method != "GET" {
        return
    }

    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return
    }

    // Restore response body for caller
    resp.Body = io.NopCloser(bytes.NewReader(body))

    // Create cached response
    cached := &CachedResponse{
        StatusCode: resp.StatusCode,
        Headers:    resp.Header.Clone(),
        Body:       body,
        Timestamp:  time.Now(),
        ETag:       resp.Header.Get("ETag"),
        Expiry:     c.calculateExpiry(resp),
    }

    key := c.cacheKey(req)

    // Store in memory cache
    c.mu.Lock()
    evicted := c.memoryCache.Put(key, cached)
    c.mu.Unlock()

    if evicted {
        atomic.AddInt64(&c.stats.Evictions, 1)
    }

    // Store in disk cache (async)
    go c.diskCache.Put(key, cached)
}

func (c *ResponseCache) cacheKey(req *http.Request) string {
    // Cache key: hash of URL + relevant headers
    h := sha256.New()
    h.Write([]byte(req.URL.String()))

    // Include headers that affect response (e.g., Accept-Encoding)
    for _, header := range []string{"Accept", "Accept-Encoding"} {
        if value := req.Header.Get(header); value != "" {
            h.Write([]byte(header))
            h.Write([]byte(value))
        }
    }

    return hex.EncodeToString(h.Sum(nil))
}

func (c *ResponseCache) calculateExpiry(resp *http.Response) time.Time {
    // Check Cache-Control max-age
    if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
        if maxAge := parseCacheControlMaxAge(cacheControl); maxAge > 0 {
            return time.Now().Add(maxAge)
        }
    }

    // Check Expires header
    if expires := resp.Header.Get("Expires"); expires != "" {
        if t, err := http.ParseTime(expires); err == nil {
            return t
        }
    }

    // Default TTL
    return time.Now().Add(c.config.TTL)
}

func (c *CachedResponse) Expired() bool {
    return time.Now().After(c.Expiry)
}

func (c *CachedResponse) ToHTTPResponse() *http.Response {
    return &http.Response{
        StatusCode: c.StatusCode,
        Header:     c.Headers.Clone(),
        Body:       io.NopCloser(bytes.NewReader(c.Body)),
        ContentLength: int64(len(c.Body)),
    }
}

func (c *ResponseCache) Stats() *CacheStats {
    return &CacheStats{
        Hits:        atomic.LoadInt64(&c.stats.Hits),
        Misses:      atomic.LoadInt64(&c.stats.Misses),
        Evictions:   atomic.LoadInt64(&c.stats.Evictions),
        BytesServed: atomic.LoadInt64(&c.stats.BytesServed),
    }
}

func (s *CacheStats) HitRatio() float64 {
    total := s.Hits + s.Misses
    if total == 0 {
        return 0
    }
    return float64(s.Hits) / float64(total) * 100
}
```

### Memory Cache (LRU)

**File**: `pkg/gonuget/http/cache_memory.go`

```go
package http

import (
    "container/list"
    "sync"
)

type MemoryCache struct {
    maxSize int
    size    int
    items   map[string]*list.Element
    lru     *list.List
    mu      sync.RWMutex
}

type cacheEntry struct {
    key      string
    response *CachedResponse
    size     int
}

func NewMemoryCache(maxSize int) *MemoryCache {
    return &MemoryCache{
        maxSize: maxSize,
        items:   make(map[string]*list.Element),
        lru:     list.New(),
    }
}

func (c *MemoryCache) Get(key string) (*CachedResponse, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    elem, ok := c.items[key]
    if !ok {
        return nil, false
    }

    // Move to front (most recently used)
    c.lru.MoveToFront(elem)

    entry := elem.Value.(*cacheEntry)
    return entry.response, true
}

func (c *MemoryCache) Put(key string, response *CachedResponse) bool {
    c.mu.Lock()
    defer c.mu.Unlock()

    size := len(response.Body)

    // Update existing entry
    if elem, ok := c.items[key]; ok {
        c.lru.MoveToFront(elem)
        entry := elem.Value.(*cacheEntry)
        c.size = c.size - entry.size + size
        entry.response = response
        entry.size = size
        return false
    }

    // Evict if necessary
    evicted := false
    for c.size+size > c.maxSize && c.lru.Len() > 0 {
        c.evictOldest()
        evicted = true
    }

    // Add new entry
    entry := &cacheEntry{
        key:      key,
        response: response,
        size:     size,
    }
    elem := c.lru.PushFront(entry)
    c.items[key] = elem
    c.size += size

    return evicted
}

func (c *MemoryCache) evictOldest() {
    elem := c.lru.Back()
    if elem == nil {
        return
    }

    c.lru.Remove(elem)
    entry := elem.Value.(*cacheEntry)
    delete(c.items, entry.key)
    c.size -= entry.size
}
```

---

## Circuit Breaker

### Circuit Breaker Pattern

**File**: `pkg/gonuget/http/circuit.go`

```go
package http

import (
    "sync"
    "time"
)

type CircuitState int

const (
    StateClosed CircuitState = iota // Normal operation
    StateOpen                        // Circuit is open, rejecting requests
    StateHalfOpen                    // Testing if service recovered
)

type CircuitConfig struct {
    MaxFailures  int           // Open circuit after N consecutive failures
    ResetTimeout time.Duration // Try half-open after this duration
    OnOpen       func()        // Callback when circuit opens
    OnClose      func()        // Callback when circuit closes
}

type CircuitBreaker struct {
    config         *CircuitConfig
    state          CircuitState
    failures       int
    lastFailTime   time.Time
    mu             sync.RWMutex
}

func NewCircuitBreaker(cfg *CircuitConfig) *CircuitBreaker {
    if cfg == nil {
        cfg = &CircuitConfig{
            MaxFailures:  5,
            ResetTimeout: 30 * time.Second,
        }
    }
    return &CircuitBreaker{
        config: cfg,
        state:  StateClosed,
    }
}

func (cb *CircuitBreaker) AllowRequest() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        // Check if we should transition to half-open
        if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
            cb.state = StateHalfOpen
            return true
        }
        return false
    case StateHalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if cb.state == StateHalfOpen {
        // Transition to closed
        cb.state = StateClosed
        cb.failures = 0
        if cb.config.OnClose != nil {
            go cb.config.OnClose()
        }
    } else if cb.state == StateClosed {
        cb.failures = 0
    }
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailTime = time.Now()

    if cb.state == StateHalfOpen {
        // Transition back to open
        cb.state = StateOpen
        if cb.config.OnOpen != nil {
            go cb.config.OnOpen()
        }
    } else if cb.state == StateClosed && cb.failures >= cb.config.MaxFailures {
        // Transition to open
        cb.state = StateOpen
        if cb.config.OnOpen != nil {
            go cb.config.OnOpen()
        }
    }
}

func (cb *CircuitBreaker) State() CircuitState {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}
```

### Circuit Breaker State Machine

```
     ┌──────────────┐
     │    Closed    │◄─────────────────┐
     │  (Normal)    │                  │
     └──────┬───────┘                  │
            │                          │
            │ N failures               │ Success
            ▼                          │
     ┌──────────────┐           ┌──────┴───────┐
     │     Open     │           │  Half-Open   │
     │  (Blocking)  │──────────►│  (Testing)   │
     └──────────────┘           └──────┬───────┘
      Timeout elapsed                  │
                                       │ Failure
                                       │
                                       ▼
                                  (Back to Open)
```

---

## Rate Limiting

### Token Bucket Rate Limiter

**File**: `pkg/gonuget/http/ratelimit.go`

```go
package http

import (
    "context"
    "sync"
    "time"
)

type RateLimitConfig struct {
    RequestsPerSecond float64 // Requests per second
    BurstSize         int     // Maximum burst size
}

type RateLimiter struct {
    rps        float64
    burstSize  int
    tokens     float64
    lastUpdate time.Time
    mu         sync.Mutex
}

func NewRateLimiter(cfg *RateLimitConfig) *RateLimiter {
    if cfg == nil {
        return &RateLimiter{
            rps:        100,
            burstSize:  20,
            tokens:     20,
            lastUpdate: time.Now(),
        }
    }
    return &RateLimiter{
        rps:        cfg.RequestsPerSecond,
        burstSize:  cfg.BurstSize,
        tokens:     float64(cfg.BurstSize),
        lastUpdate: time.Now(),
    }
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    for {
        rl.mu.Lock()

        // Refill tokens based on elapsed time
        now := time.Now()
        elapsed := now.Sub(rl.lastUpdate).Seconds()
        rl.tokens = min(float64(rl.burstSize), rl.tokens+elapsed*rl.rps)
        rl.lastUpdate = now

        // Try to consume a token
        if rl.tokens >= 1 {
            rl.tokens--
            rl.mu.Unlock()
            return nil
        }

        // Calculate wait time
        waitTime := time.Duration((1-rl.tokens)/rl.rps*1000) * time.Millisecond
        rl.mu.Unlock()

        // Wait or check for cancellation
        select {
        case <-time.After(waitTime):
            // Continue loop
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

func min(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}
```

---

## Progress Tracking

### Download Progress

**File**: `pkg/gonuget/http/progress.go`

```go
package http

import (
    "io"
)

type DownloadProgress struct {
    URL             string
    BytesDownloaded int64
    TotalBytes      int64
    Percentage      float64
    Speed           int64 // Bytes per second
}

type ProgressReader struct {
    reader        io.Reader
    total         int64
    downloaded    int64
    progressChan  chan<- DownloadProgress
    lastReport    time.Time
    reportInterval time.Duration
    url           string
}

func NewProgressReader(reader io.Reader, total int64, url string, progressChan chan<- DownloadProgress) *ProgressReader {
    return &ProgressReader{
        reader:         reader,
        total:          total,
        progressChan:   progressChan,
        reportInterval: 100 * time.Millisecond,
        url:            url,
        lastReport:     time.Now(),
    }
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
    n, err := pr.reader.Read(p)
    pr.downloaded += int64(n)

    // Report progress
    if time.Since(pr.lastReport) >= pr.reportInterval || err != nil {
        pr.report()
        pr.lastReport = time.Now()
    }

    return n, err
}

func (pr *ProgressReader) report() {
    if pr.progressChan == nil {
        return
    }

    percentage := 0.0
    if pr.total > 0 {
        percentage = float64(pr.downloaded) / float64(pr.total) * 100
    }

    // Calculate speed
    elapsed := time.Since(pr.lastReport).Seconds()
    speed := int64(0)
    if elapsed > 0 {
        speed = int64(float64(pr.downloaded) / elapsed)
    }

    select {
    case pr.progressChan <- DownloadProgress{
        URL:             pr.url,
        BytesDownloaded: pr.downloaded,
        TotalBytes:      pr.total,
        Percentage:      percentage,
        Speed:           speed,
    }:
    default:
        // Don't block if channel is full
    }
}
```

---

## Authentication

See `pkg/gonuget/auth/` for full authentication implementation.

### Auth Provider Interface

**File**: `pkg/gonuget/auth/auth.go`

```go
package auth

import (
    "context"
    "net/http"
)

type Provider interface {
    ApplyAuth(ctx context.Context, req *http.Request) error
    Type() string
}

// Basic Auth
type BasicAuthProvider struct {
    username string
    password string
}

func (p *BasicAuthProvider) ApplyAuth(ctx context.Context, req *http.Request) error {
    req.SetBasicAuth(p.username, p.password)
    return nil
}

// Bearer Token
type BearerTokenProvider struct {
    token string
}

func (p *BearerTokenProvider) ApplyAuth(ctx context.Context, req *http.Request) error {
    req.Header.Set("Authorization", "Bearer "+p.token)
    return nil
}

// API Key (X-NuGet-ApiKey header)
type APIKeyProvider struct {
    apiKey string
}

func (p *APIKeyProvider) ApplyAuth(ctx context.Context, req *http.Request) error {
    req.Header.Set("X-NuGet-ApiKey", p.apiKey)
    return nil
}
```

---

## Middleware Chain

### Middleware Pattern

**File**: `pkg/gonuget/http/middleware.go`

```go
package http

import (
    "context"
    "net/http"
)

type HandlerFunc func(context.Context, *http.Request) (*http.Response, error)
type Middleware func(HandlerFunc) HandlerFunc

// Logging middleware
func LoggingMiddleware(logger Logger) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, req *http.Request) (*http.Response, error) {
            start := time.Now()
            logger.Debug("HTTP request: {Method} {URL}", req.Method, req.URL)

            resp, err := next(ctx, req)

            duration := time.Since(start)
            if err != nil {
                logger.Error("HTTP request failed: {Method} {URL} ({Duration}ms): {Error}",
                    req.Method, req.URL, duration.Milliseconds(), err)
            } else {
                logger.Debug("HTTP response: {StatusCode} ({Duration}ms)",
                    resp.StatusCode, duration.Milliseconds())
            }

            return resp, err
        }
    }
}

// Tracing middleware (OpenTelemetry)
func TracingMiddleware(tracer trace.Tracer) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, req *http.Request) (*http.Response, error) {
            ctx, span := tracer.Start(ctx, "http.request",
                trace.WithSpanKind(trace.SpanKindClient),
                trace.WithAttributes(
                    attribute.String("http.method", req.Method),
                    attribute.String("http.url", req.URL.String()),
                ),
            )
            defer span.End()

            resp, err := next(ctx, req)

            if err != nil {
                span.RecordError(err)
                span.SetStatus(codes.Error, err.Error())
            } else {
                span.SetAttributes(
                    attribute.Int("http.status_code", resp.StatusCode),
                )
            }

            return resp, err
        }
    }
}
```

---

## Implementation Details

### Package Dependencies

```go
require (
    // HTTP/2 support
    golang.org/x/net v0.17.0

    // OpenTelemetry
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0

    // Structured logging
    github.com/willibrandon/mtlog v1.0.0
)
```

### Performance Optimizations

1. **Connection Pooling**: Reuse HTTP connections
2. **Keep-Alive**: Persistent connections
3. **HTTP/2 Multiplexing**: Multiple requests over single connection
4. **Zero-Copy**: Use `io.Copy` to avoid unnecessary allocations
5. **Object Pooling**: Pool buffers and JSON decoders

### Testing Strategy

See [DESIGN-TESTING.md](./DESIGN-TESTING.md) for comprehensive testing strategy.

---

**Document Status**: Draft v1.0
**Last Updated**: 2025-01-19
**Next Review**: After implementation
