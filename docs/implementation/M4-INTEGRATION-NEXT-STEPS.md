# M4 Integration Guide: Next Steps

**Status:** Implementation Complete - Integration Required
**Last Updated:** 2025-10-24
**Owner:** Engineering

---

## Table of Contents

1. [Overview](#overview)
2. [What's Been Implemented](#whats-been-implemented)
3. [Integration Task 1: Wire Cache into Repository](#integration-task-1-wire-cache-into-repository)
4. [Integration Task 2: Wire Resilience into HTTP Client](#integration-task-2-wire-resilience-into-http-client)
5. [Integration Task 3: Implement Observability (mtlog)](#integration-task-3-implement-observability-mtlog)
6. [Integration Task 4: Wire Observability Throughout](#integration-task-4-wire-observability-throughout)
7. [Testing Strategy](#testing-strategy)
8. [Success Criteria](#success-criteria)

---

## Overview

All M4 infrastructure components have been **implemented and tested** but are **NOT YET INTEGRATED** into the core library. This guide outlines the integration work needed to wire everything together.

### Implementation Status

| Component | Implementation | Tests | Integration | Location |
|-----------|---------------|-------|-------------|----------|
| Memory Cache (LRU) | ✅ Complete | ✅ Passing | ❌ Not Integrated | `cache/memory.go` |
| Disk Cache | ✅ Complete | ✅ Passing | ❌ Not Integrated | `cache/disk.go` |
| Multi-Tier Cache | ✅ Complete | ✅ Passing | ❌ Not Integrated | `cache/multi_tier.go` |
| Cache Context | ✅ Complete | ✅ Passing | ❌ Not Integrated | `cache/context.go` |
| Circuit Breaker | ✅ Complete | ✅ Passing | ❌ Not Integrated | `resilience/circuit_breaker.go` |
| HTTP Circuit Breaker | ✅ Complete | ✅ Passing | ❌ Not Integrated | `resilience/http_breaker.go` |
| Token Bucket Rate Limiter | ✅ Complete | ✅ Passing | ❌ Not Integrated | `resilience/rate_limiter.go` |
| Per-Source Rate Limiter | ✅ Complete | ✅ Passing | ❌ Not Integrated | `resilience/per_source_limiter.go` |
| HTTP Retry | ✅ Complete | ✅ Passing | ✅ **Integrated** | `http/retry.go` |
| mtlog Integration | ❌ Not Implemented | ❌ N/A | ❌ Not Integrated | - |
| OpenTelemetry Tracing | ❌ Not Implemented | ❌ N/A | ❌ Not Integrated | - |
| Prometheus Metrics | ❌ Not Implemented | ❌ N/A | ❌ Not Integrated | - |

---

## What's Been Implemented

### Cache Components (Fully Tested)

All cache components are production-ready with 100% NuGet.Client parity:

**Memory Cache API:**
```go
mc := cache.NewMemoryCache(maxEntries, maxBytes)
mc.Set(key, data, ttl)           // Store with TTL
data, ok := mc.Get(key)           // Retrieve
mc.Delete(key)                    // Remove
mc.Clear()                        // Clear all
stats := mc.Stats()               // Get statistics
```

**Disk Cache API:**
```go
dc, _ := cache.NewDiskCache(rootDir, maxSize)
dc.Set(sourceURL, cacheKey, reader, validator)  // Atomic write
reader, hit, _ := dc.Get(sourceURL, cacheKey, maxAge)
dc.Delete(sourceURL, cacheKey)
dc.Clear(sourceURL)
path := dc.GetCachePath(sourceURL, cacheKey)
```

**Multi-Tier Cache API:**
```go
mtc := cache.NewMultiTierCache(memCache, diskCache)
data, hit, _ := mtc.Get(ctx, sourceURL, cacheKey, maxAge)
mtc.Set(ctx, sourceURL, cacheKey, reader, maxAge, validator)
mtc.Delete(ctx, sourceURL, cacheKey)
mtc.Clear(ctx, sourceURL)
```

**Cache Context API:**
```go
cacheCtx := cache.NewSourceCacheContext()
cacheCtx.MaxAge = 30 * time.Minute
cacheCtx.NoCache = false
cacheCtx.DirectDownload = false
cacheCtx.RefreshMemoryCache = false
sessionID := cacheCtx.GenerateSessionID()
```

### Resilience Components (Fully Tested)

**Circuit Breaker API:**
```go
cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig())
err := cb.CanExecute()            // Check if allowed
cb.RecordSuccess()                // Record success
cb.RecordFailure()                // Record failure
cb.Reset()                        // Manual reset
state := cb.State()               // Get current state
stats := cb.Stats()               // Get statistics
```

**HTTP Circuit Breaker API:**
```go
httpCB := resilience.NewHTTPCircuitBreaker(config)
resp, err := httpCB.Execute(ctx, host, func() (*http.Response, error) {
    return http.Get(url)
})
state := httpCB.GetState(host)
allStats := httpCB.GetAllStats()
```

**Rate Limiter API:**
```go
tb := resilience.NewTokenBucket(resilience.DefaultTokenBucketConfig())
if tb.Allow() { /* proceed */ }
tb.Wait(ctx)                      // Block until token available
tb.WaitN(ctx, n)                  // Wait for N tokens
tokens := tb.Tokens()             // Current tokens
```

**Per-Source Rate Limiter API:**
```go
psl := resilience.NewPerSourceLimiter(config)
if psl.Allow(sourceURL) { /* proceed */ }
psl.Wait(ctx, sourceURL)
stats := psl.GetStats(sourceURL)
allStats := psl.GetAllStats()
```

### HTTP Retry (Already Integrated)

```go
// Already wired into http/client.go
client := nugethttp.NewClient(&nugethttp.ClientConfig{
    RetryConfig: &nugethttp.RetryConfig{
        MaxRetries:   3,
        InitialDelay: time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
        Jitter:       0.25,
    },
})
resp, err := client.DoWithRetry(ctx, req)
```

---

## Integration Task 1: Wire Cache into Repository

**Goal:** Integrate multi-tier cache into `core.SourceRepository` for metadata and package caching.

**Priority:** P0 (Critical)
**Estimated Time:** 2 hours

### Changes Required

#### 1. Update `core/repository.go`

**Add fields to SourceRepository:**
```go
type SourceRepository struct {
	name            string
	sourceURL       string
	authenticator   auth.Authenticator
	httpClient      *nugethttp.Client
	providerFactory *ProviderFactory

	// Cache integration (NEW)
	cache           *cache.MultiTierCache

	mu       sync.RWMutex
	provider ResourceProvider
}
```

**Add WithCache builder method:**
```go
// WithCache configures the repository cache
func (r *SourceRepository) WithCache(cache *cache.MultiTierCache) *SourceRepository {
	r.cache = cache
	return r
}
```

**Update GetMetadata with caching:**
```go
func (r *SourceRepository) GetMetadata(ctx context.Context, packageID, version string) (*ProtocolMetadata, error) {
	// Check cache first
	if r.cache != nil {
		cacheKey := fmt.Sprintf("metadata:%s:%s", packageID, version)
		cached, hit, err := r.cache.Get(ctx, r.sourceURL, cacheKey, 30*time.Minute)
		if err == nil && hit {
			var metadata ProtocolMetadata
			if err := json.Unmarshal(cached, &metadata); err == nil {
				return &metadata, nil
			}
		}
	}

	// Get provider and fetch
	provider, err := r.GetProvider(ctx)
	if err != nil {
		return nil, err
	}

	metadata, err := provider.GetMetadata(ctx, packageID, version)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		cacheKey := fmt.Sprintf("metadata:%s:%s", packageID, version)
		jsonData, err := json.Marshal(metadata)
		if err == nil {
			r.cache.Set(ctx, r.sourceURL, cacheKey, bytes.NewReader(jsonData), 30*time.Minute, nil)
		}
	}

	return metadata, nil
}
```

**Update DownloadPackage with caching:**
```go
func (r *SourceRepository) DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error) {
	// Check cache first
	if r.cache != nil {
		cacheKey := fmt.Sprintf("package:%s.%s.nupkg", packageID, version)
		cached, hit, err := r.cache.Get(ctx, r.sourceURL, cacheKey, 24*time.Hour) // Packages are immutable
		if err == nil && hit {
			return io.NopCloser(bytes.NewReader(cached)), nil
		}
	}

	// Get provider and download
	provider, err := r.GetProvider(ctx)
	if err != nil {
		return nil, err
	}

	reader, err := provider.DownloadPackage(ctx, packageID, version)
	if err != nil {
		return nil, err
	}

	// Read package data for caching
	packageData, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		return nil, err
	}

	// Cache the downloaded package with ZIP validation
	if r.cache != nil {
		cacheKey := fmt.Sprintf("package:%s.%s.nupkg", packageID, version)
		validator := func(rs io.ReadSeeker) error {
			// Basic ZIP validation - check for PK signature
			var sig [2]byte
			if _, err := rs.Read(sig[:]); err != nil {
				return fmt.Errorf("failed to read signature: %w", err)
			}
			if sig[0] != 0x50 || sig[1] != 0x4B { // PK signature
				return fmt.Errorf("invalid ZIP signature")
			}
			return nil
		}
		r.cache.Set(ctx, r.sourceURL, cacheKey, bytes.NewReader(packageData), 24*time.Hour, validator)
	}

	return io.NopCloser(bytes.NewReader(packageData)), nil
}
```

#### 2. Update `core/client.go`

**Add cache configuration:**
```go
type ClientConfig struct {
	Sources         []SourceRepository
	UserAgent       string
	HTTPClient      *nugethttp.Client
	CacheDir        string        // NEW: Cache root directory
	CacheMaxSize    int64         // NEW: Max disk cache size
	MemoryCacheSize int          // NEW: Max memory cache entries
}

func NewClient(cfg ClientConfig) (*Client, error) {
	// Create multi-tier cache if configured
	var mtCache *cache.MultiTierCache
	if cfg.CacheDir != "" {
		memCache := cache.NewMemoryCache(cfg.MemoryCacheSize, 100*1024*1024) // 100MB
		diskCache, err := cache.NewDiskCache(cfg.CacheDir, cfg.CacheMaxSize)
		if err != nil {
			return nil, fmt.Errorf("failed to create disk cache: %w", err)
		}
		mtCache = cache.NewMultiTierCache(memCache, diskCache)
	}

	// Configure repositories with cache
	for i := range cfg.Sources {
		if mtCache != nil {
			cfg.Sources[i].WithCache(mtCache)
		}
	}

	// ... rest of client creation
}
```

### Testing

**Test file:** `core/repository_cache_test.go`

```go
func TestSourceRepository_GetMetadata_WithCache(t *testing.T) {
	// Create test cache
	memCache := cache.NewMemoryCache(100, 10*1024*1024)
	diskCache, _ := cache.NewDiskCache(t.TempDir(), 100*1024*1024)
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	// Create repository with cache
	repo := core.NewSourceRepository(core.RepositoryConfig{
		Name:      "test",
		SourceURL: "https://api.nuget.org/v3/index.json",
	}).WithCache(mtCache)

	ctx := context.Background()

	// First call should miss cache and fetch
	metadata1, err := repo.GetMetadata(ctx, "Newtonsoft.Json", "13.0.1")
	require.NoError(t, err)
	require.NotNil(t, metadata1)

	// Second call should hit cache (verify by checking metrics or timing)
	metadata2, err := repo.GetMetadata(ctx, "Newtonsoft.Json", "13.0.1")
	require.NoError(t, err)
	require.Equal(t, metadata1.ID, metadata2.ID)
}
```

---

## Integration Task 2: Wire Resilience into HTTP Client

**Goal:** Wrap HTTP client with circuit breaker and rate limiter.

**Priority:** P0 (Critical)
**Estimated Time:** 3 hours

### Changes Required

#### 1. Update `http/client.go`

**Add resilience fields:**
```go
type Client struct {
	httpClient     *http.Client
	retryHandler   *RetryHandler
	circuitBreaker *resilience.HTTPCircuitBreaker  // NEW
	rateLimiter    *resilience.PerSourceLimiter    // NEW
	userAgent      string
}

type ClientConfig struct {
	Timeout            time.Duration
	ConnectionTimeout  time.Duration
	MaxIdleConns       int
	MaxConnsPerHost    int
	TLSConfig          *tls.Config
	HTTP2Enabled       bool
	RetryConfig        *RetryConfig
	CircuitBreakerConfig *resilience.CircuitBreakerConfig  // NEW
	RateLimiterConfig    *resilience.TokenBucketConfig     // NEW
}
```

**Update NewClient:**
```go
func NewClient(cfg *ClientConfig) *Client {
	// ... existing HTTP client setup ...

	client := &Client{
		httpClient:   httpClient,
		retryHandler: newRetryHandler(cfg.RetryConfig),
		userAgent:    userAgent,
	}

	// Add circuit breaker if configured
	if cfg.CircuitBreakerConfig != nil {
		client.circuitBreaker = resilience.NewHTTPCircuitBreaker(*cfg.CircuitBreakerConfig)
	}

	// Add rate limiter if configured
	if cfg.RateLimiterConfig != nil {
		client.rateLimiter = resilience.NewPerSourceLimiter(*cfg.RateLimiterConfig)
	}

	return client
}
```

**Update Do method to use resilience:**
```go
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Set User-Agent
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// Extract host for circuit breaker/rate limiter
	host := req.URL.Host

	// Apply rate limiting
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, host); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	// Apply circuit breaker
	if c.circuitBreaker != nil {
		return c.circuitBreaker.Execute(ctx, host, func() (*http.Response, error) {
			return c.httpClient.Do(req)
		})
	}

	return c.httpClient.Do(req)
}
```

**Update DoWithRetry to use resilience:**
```go
func (c *Client) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	host := req.URL.Host

	// Apply rate limiting
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, host); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	// Combine circuit breaker + retry
	if c.circuitBreaker != nil {
		return c.circuitBreaker.Execute(ctx, host, func() (*http.Response, error) {
			return c.retryHandler.DoWithRetry(ctx, req, c.httpClient)
		})
	}

	return c.retryHandler.DoWithRetry(ctx, req, c.httpClient)
}
```

### Testing

**Test file:** `http/client_resilience_test.go`

```go
func TestClient_WithCircuitBreaker(t *testing.T) {
	// Create failing server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with circuit breaker
	client := NewClient(&ClientConfig{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:        3,
			Timeout:            1 * time.Second,
			MaxHalfOpenRequests: 1,
		},
	})

	// Make requests until circuit opens
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		_, err := client.Do(context.Background(), req)
		if err != nil && strings.Contains(err.Error(), "circuit breaker is open") {
			return // Success - circuit opened
		}
	}
	t.Fatal("Circuit breaker did not open")
}

func TestClient_WithRateLimiter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with rate limiter (low rate for testing)
	client := NewClient(&ClientConfig{
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   2,
			RefillRate: 1.0, // 1 token/second
		},
	})

	// First 2 requests should succeed immediately
	start := time.Now()
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		_, err := client.Do(context.Background(), req)
		require.NoError(t, err)
	}

	// Third request should be delayed ~1 second
	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := client.Do(context.Background(), req)
	require.NoError(t, err)

	elapsed := time.Since(start)
	assert.Greater(t, elapsed, 900*time.Millisecond, "Rate limiter should delay request")
}
```

---

## Integration Task 3: Implement Observability (mtlog)

**Goal:** Create observability infrastructure using mtlog for logging, OpenTelemetry for tracing, and Prometheus for metrics.

**Priority:** P0 (Critical)
**Estimated Time:** 4 hours

### Part A: mtlog Logger Wrapper

**File:** `observability/logger.go`

```go
package observability

import (
	"context"

	"github.com/willibrandon/mtlog/core"
)

// Logger is the gonuget logging interface wrapping mtlog.
// This provides a simplified interface tailored to gonuget's needs.
type Logger interface {
	// Verbose logs verbose diagnostic messages
	Verbose(messageTemplate string, args ...any)
	VerboseContext(ctx context.Context, messageTemplate string, args ...any)

	// Debug logs debug messages
	Debug(messageTemplate string, args ...any)
	DebugContext(ctx context.Context, messageTemplate string, args ...any)

	// Information logs informational messages
	Information(messageTemplate string, args ...any)
	InfoContext(ctx context.Context, messageTemplate string, args ...any)

	// Warning logs warning messages
	Warning(messageTemplate string, args ...any)
	WarnContext(ctx context.Context, messageTemplate string, args ...any)

	// Error logs error messages
	Error(messageTemplate string, args ...any)
	ErrorContext(ctx context.Context, messageTemplate string, args ...any)

	// Fatal logs fatal error messages
	Fatal(messageTemplate string, args ...any)
	FatalContext(ctx context.Context, messageTemplate string, args ...any)

	// With returns a logger enriched with properties
	With(args ...any) Logger

	// ForContext returns a logger with SourceContext property
	ForContext(propertyName string, value any) Logger
}

// mtlogLogger wraps core.Logger from mtlog
type mtlogLogger struct {
	logger core.Logger
}

// NewLogger creates a Logger wrapping an mtlog core.Logger
func NewLogger(mtlog core.Logger) Logger {
	return &mtlogLogger{logger: mtlog}
}

func (l *mtlogLogger) Verbose(messageTemplate string, args ...any) {
	l.logger.Verbose(messageTemplate, args...)
}

func (l *mtlogLogger) VerboseContext(ctx context.Context, messageTemplate string, args ...any) {
	l.logger.VerboseContext(ctx, messageTemplate, args...)
}

func (l *mtlogLogger) Debug(messageTemplate string, args ...any) {
	l.logger.Debug(messageTemplate, args...)
}

func (l *mtlogLogger) DebugContext(ctx context.Context, messageTemplate string, args ...any) {
	l.logger.DebugContext(ctx, messageTemplate, args...)
}

func (l *mtlogLogger) Information(messageTemplate string, args ...any) {
	l.logger.Information(messageTemplate, args...)
}

func (l *mtlogLogger) InfoContext(ctx context.Context, messageTemplate string, args ...any) {
	l.logger.InfoContext(ctx, messageTemplate, args...)
}

func (l *mtlogLogger) Warning(messageTemplate string, args ...any) {
	l.logger.Warning(messageTemplate, args...)
}

func (l *mtlogLogger) WarnContext(ctx context.Context, messageTemplate string, args ...any) {
	l.logger.WarnContext(ctx, messageTemplate, args...)
}

func (l *mtlogLogger) Error(messageTemplate string, args ...any) {
	l.logger.Error(messageTemplate, args...)
}

func (l *mtlogLogger) ErrorContext(ctx context.Context, messageTemplate string, args ...any) {
	l.logger.ErrorContext(ctx, messageTemplate, args...)
}

func (l *mtlogLogger) Fatal(messageTemplate string, args ...any) {
	l.logger.Fatal(messageTemplate, args...)
}

func (l *mtlogLogger) FatalContext(ctx context.Context, messageTemplate string, args ...any) {
	l.logger.FatalContext(ctx, messageTemplate, args...)
}

func (l *mtlogLogger) With(args ...any) Logger {
	return &mtlogLogger{logger: l.logger.With(args...)}
}

func (l *mtlogLogger) ForContext(propertyName string, value any) Logger {
	return &mtlogLogger{logger: l.logger.ForContext(propertyName, value)}
}

// NullLogger is a no-op logger for when logging is disabled
type nullLogger struct{}

func NewNullLogger() Logger {
	return &nullLogger{}
}

func (n *nullLogger) Verbose(messageTemplate string, args ...any)                            {}
func (n *nullLogger) VerboseContext(ctx context.Context, messageTemplate string, args ...any) {}
func (n *nullLogger) Debug(messageTemplate string, args ...any)                              {}
func (n *nullLogger) DebugContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) Information(messageTemplate string, args ...any)                        {}
func (n *nullLogger) InfoContext(ctx context.Context, messageTemplate string, args ...any)    {}
func (n *nullLogger) Warning(messageTemplate string, args ...any)                            {}
func (n *nullLogger) WarnContext(ctx context.Context, messageTemplate string, args ...any)    {}
func (n *nullLogger) Error(messageTemplate string, args ...any)                              {}
func (n *nullLogger) ErrorContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) Fatal(messageTemplate string, args ...any)                              {}
func (n *nullLogger) FatalContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) With(args ...any) Logger                                                 { return n }
func (n *nullLogger) ForContext(propertyName string, value any) Logger                        { return n }
```

### Part B: OpenTelemetry Tracing

**File:** `observability/tracing.go`

```go
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// InitTracing initializes OpenTelemetry tracing
func InitTracing(serviceName string) {
	tracer = otel.Tracer(serviceName)
}

// StartPackageDownloadSpan starts a trace span for package downloads
func StartPackageDownloadSpan(ctx context.Context, packageID, version, sourceURL string) (context.Context, trace.Span) {
	return tracer.Start(ctx, "DownloadPackage",
		trace.WithAttributes(
			attribute.String("package.id", packageID),
			attribute.String("package.version", version),
			attribute.String("source.url", sourceURL),
		))
}

// StartMetadataFetchSpan starts a trace span for metadata fetching
func StartMetadataFetchSpan(ctx context.Context, packageID, version, sourceURL string) (context.Context, trace.Span) {
	return tracer.Start(ctx, "FetchMetadata",
		trace.WithAttributes(
			attribute.String("package.id", packageID),
			attribute.String("package.version", version),
			attribute.String("source.url", sourceURL),
		))
}

// RecordError records an error in the current span
func RecordError(span trace.Span, err error) {
	if err != nil && span != nil {
		span.RecordError(err)
	}
}

// SetSpanStatus sets the span status based on error
func SetSpanStatus(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeError, Description: err.Error()})
	} else {
		span.SetStatus(trace.Status{Code: trace.StatusCodeOk})
	}
}
```

### Part C: Prometheus Metrics

**File:** `observability/metrics.go`

```go
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Cache metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"}, // metadata, package
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	// Package download metrics
	PackageDownloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_package_downloads_total",
			Help: "Total number of package download attempts",
		},
		[]string{"status"}, // success, failure
	)

	PackageDownloadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gonuget_package_download_duration_seconds",
			Help:    "Package download duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"package_id"},
	)

	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"source", "method", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gonuget_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source", "method"},
	)

	// Circuit breaker metrics
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"host"},
	)

	// Rate limiter metrics
	RateLimiterTokens = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_rate_limiter_tokens",
			Help: "Available tokens in rate limiter",
		},
		[]string{"source"},
	)
)
```

### Testing

**Test file:** `observability/logger_test.go`

```go
func TestLogger_WithMtlog(t *testing.T) {
	var buf bytes.Buffer
	mtlogger := mtlog.New(
		mtlog.WithConsole(),
		mtlog.WithMinimumLevel(core.VerboseLevel),
	)

	logger := NewLogger(mtlogger)

	logger.Information("Test message {Property}", "value")
	logger.With("RequestID", "12345").Information("Scoped message")
	logger.ForContext("SourceContext", "test").Information("Context message")
}

func TestNullLogger(t *testing.T) {
	logger := NewNullLogger()

	// Should not panic
	logger.Information("Test")
	logger.Error("Error")
	logger.With("key", "value").Information("Scoped")
}
```

---

## Integration Task 4: Wire Observability Throughout

**Goal:** Add logging, tracing, and metrics throughout the codebase.

**Priority:** P1 (High)
**Estimated Time:** 3 hours

### Changes Required

#### 1. Update `core/repository.go` with logging

```go
type SourceRepository struct {
	name            string
	sourceURL       string
	authenticator   auth.Authenticator
	httpClient      *nugethttp.Client
	providerFactory *ProviderFactory
	cache           *cache.MultiTierCache
	logger          observability.Logger  // NEW

	mu       sync.RWMutex
	provider ResourceProvider
}

func NewSourceRepository(cfg RepositoryConfig) *SourceRepository {
	logger := cfg.Logger
	if logger == nil {
		logger = observability.NewNullLogger()
	}

	return &SourceRepository{
		// ... existing fields ...
		logger: logger,
	}
}

func (r *SourceRepository) GetMetadata(ctx context.Context, packageID, version string) (*ProtocolMetadata, error) {
	r.logger.InfoContext(ctx, "Fetching metadata {PackageID} {Version} from {Source}",
		packageID, version, r.sourceURL)

	// Check cache first
	if r.cache != nil {
		cacheKey := fmt.Sprintf("metadata:%s:%s", packageID, version)
		cached, hit, err := r.cache.Get(ctx, r.sourceURL, cacheKey, 30*time.Minute)
		if err != nil {
			r.logger.WarnContext(ctx, "Cache error for {PackageID} {Version}: {Error}",
				packageID, version, err)
		}
		if hit {
			observability.CacheHitsTotal.WithLabelValues("metadata").Inc()
			r.logger.DebugContext(ctx, "Cache hit for metadata {PackageID} {Version}",
				packageID, version)

			var metadata ProtocolMetadata
			if err := json.Unmarshal(cached, &metadata); err == nil {
				return &metadata, nil
			}
		}
		observability.CacheMissesTotal.WithLabelValues("metadata").Inc()
	}

	// Fetch from source
	provider, err := r.GetProvider(ctx)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to get provider for {Source}: {Error}",
			r.sourceURL, err)
		return nil, err
	}

	metadata, err := provider.GetMetadata(ctx, packageID, version)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to fetch metadata {PackageID} {Version}: {Error}",
			packageID, version, err)
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		cacheKey := fmt.Sprintf("metadata:%s:%s", packageID, version)
		jsonData, _ := json.Marshal(metadata)
		r.cache.Set(ctx, r.sourceURL, cacheKey, bytes.NewReader(jsonData), 30*time.Minute, nil)
		r.logger.DebugContext(ctx, "Cached metadata {PackageID} {Version}",
			packageID, version)
	}

	r.logger.InfoContext(ctx, "Successfully fetched metadata {PackageID} {Version}",
		packageID, version)

	return metadata, nil
}

func (r *SourceRepository) DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error) {
	// Start trace span
	ctx, span := observability.StartPackageDownloadSpan(ctx, packageID, version, r.sourceURL)
	defer span.End()

	r.logger.InfoContext(ctx, "Downloading package {PackageID} {Version} from {Source}",
		packageID, version, r.sourceURL)

	start := time.Now()

	// Check cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("package:%s.%s.nupkg", packageID, version)
		cached, hit, err := r.cache.Get(ctx, r.sourceURL, cacheKey, 24*time.Hour)
		if err == nil && hit {
			duration := time.Since(start)
			observability.CacheHitsTotal.WithLabelValues("package").Inc()
			observability.PackageDownloadsTotal.WithLabelValues("success").Inc()
			observability.PackageDownloadDuration.WithLabelValues(packageID).Observe(duration.Seconds())

			r.logger.InfoContext(ctx, "Downloaded package {PackageID} {Version} from cache in {Duration}",
				packageID, version, duration)

			observability.SetSpanStatus(span, nil)
			return io.NopCloser(bytes.NewReader(cached)), nil
		}
		observability.CacheMissesTotal.WithLabelValues("package").Inc()
	}

	// Download from source
	provider, err := r.GetProvider(ctx)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to get provider: {Error}", err)
		observability.RecordError(span, err)
		observability.SetSpanStatus(span, err)
		return nil, err
	}

	reader, err := provider.DownloadPackage(ctx, packageID, version)
	if err != nil {
		duration := time.Since(start)
		observability.PackageDownloadsTotal.WithLabelValues("failure").Inc()

		r.logger.ErrorContext(ctx, "Failed to download package {PackageID} {Version} after {Duration}: {Error}",
			packageID, version, duration, err)

		observability.RecordError(span, err)
		observability.SetSpanStatus(span, err)
		return nil, err
	}

	// Read and cache
	packageData, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to read package data: {Error}", err)
		observability.RecordError(span, err)
		observability.SetSpanStatus(span, err)
		return nil, err
	}

	// Cache with validation
	if r.cache != nil {
		cacheKey := fmt.Sprintf("package:%s.%s.nupkg", packageID, version)
		validator := func(rs io.ReadSeeker) error {
			var sig [2]byte
			if _, err := rs.Read(sig[:]); err != nil {
				return fmt.Errorf("failed to read signature: %w", err)
			}
			if sig[0] != 0x50 || sig[1] != 0x4B {
				return fmt.Errorf("invalid ZIP signature")
			}
			return nil
		}
		r.cache.Set(ctx, r.sourceURL, cacheKey, bytes.NewReader(packageData), 24*time.Hour, validator)
	}

	duration := time.Since(start)
	observability.PackageDownloadsTotal.WithLabelValues("success").Inc()
	observability.PackageDownloadDuration.WithLabelValues(packageID).Observe(duration.Seconds())

	r.logger.InfoContext(ctx, "Successfully downloaded package {PackageID} {Version} in {Duration}",
		packageID, version, duration)

	observability.SetSpanStatus(span, nil)
	return io.NopCloser(bytes.NewReader(packageData)), nil
}
```

#### 2. Add logging to HTTP client

```go
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.logger != nil {
		c.logger.DebugContext(ctx, "HTTP {Method} {URL}", req.Method, req.URL.String())
	}

	start := time.Now()

	// ... existing Do logic with resilience ...

	resp, err := /* ... */

	duration := time.Since(start)

	if c.logger != nil {
		if err != nil {
			c.logger.WarnContext(ctx, "HTTP {Method} {URL} failed after {Duration}: {Error}",
				req.Method, req.URL.String(), duration, err)
		} else {
			c.logger.DebugContext(ctx, "HTTP {Method} {URL} returned {StatusCode} in {Duration}",
				req.Method, req.URL.String(), resp.StatusCode, duration)
		}
	}

	observability.HTTPRequestsTotal.WithLabelValues(
		req.URL.Host, req.Method, fmt.Sprintf("%d", resp.StatusCode),
	).Inc()
	observability.HTTPRequestDuration.WithLabelValues(
		req.URL.Host, req.Method,
	).Observe(duration.Seconds())

	return resp, err
}
```

---

## Testing Strategy

### Unit Tests

**Coverage Requirement:** 80%+ for all integrated code

1. **Cache Integration Tests:**
   - Test cache hit/miss paths
   - Test TTL expiration
   - Test cache bypass modes

2. **Resilience Integration Tests:**
   - Test circuit breaker state transitions
   - Test rate limiting behavior
   - Test retry + circuit breaker interaction

3. **Observability Tests:**
   - Test logging output
   - Test metric collection
   - Test span creation

### Integration Tests

**Test file:** `integration_test.go`

```go
func TestFullStackIntegration(t *testing.T) {
	// Create mtlog logger
	mtlogger := mtlog.New(
		mtlog.WithConsole(),
		mtlog.WithMinimumLevel(core.InformationLevel),
	)
	logger := observability.NewLogger(mtlogger)

	// Create cache
	memCache := cache.NewMemoryCache(1000, 100*1024*1024)
	diskCache, _ := cache.NewDiskCache(t.TempDir(), 1*1024*1024*1024)
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	// Create HTTP client with resilience
	httpClient := nugethttp.NewClient(&nugethttp.ClientConfig{
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures: 5,
			Timeout:     30 * time.Second,
		},
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   100,
			RefillRate: 50.0,
		},
	})

	// Create repository with everything
	repo := core.NewSourceRepository(core.RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  "https://api.nuget.org/v3/index.json",
		HTTPClient: httpClient,
		Logger:     logger,
	}).WithCache(mtCache)

	ctx := context.Background()

	// Test metadata fetch (should log, cache, trace)
	metadata, err := repo.GetMetadata(ctx, "Newtonsoft.Json", "13.0.1")
	require.NoError(t, err)
	require.NotNil(t, metadata)

	// Second fetch should hit cache
	metadata2, err := repo.GetMetadata(ctx, "Newtonsoft.Json", "13.0.1")
	require.NoError(t, err)
	require.Equal(t, metadata.ID, metadata2.ID)

	// Test package download (should log, cache, trace, metric)
	reader, err := repo.DownloadPackage(ctx, "Newtonsoft.Json", "13.0.1")
	require.NoError(t, err)
	defer reader.Close()

	data, _ := io.ReadAll(reader)
	require.Greater(t, len(data), 0)

	// Second download should hit cache
	reader2, err := repo.DownloadPackage(ctx, "Newtonsoft.Json", "13.0.1")
	require.NoError(t, err)
	defer reader2.Close()
}
```

---

## Success Criteria

### Functional Requirements

- ✅ Cache integrated into repository (metadata + packages)
- ✅ Circuit breaker protects HTTP requests
- ✅ Rate limiter controls request rate per source
- ✅ Logging via mtlog throughout
- ✅ Tracing via OpenTelemetry for expensive operations
- ✅ Metrics via Prometheus for monitoring

### Performance Requirements

- ✅ Cache hit latency <5ms (memory), <50ms (disk)
- ✅ Logging overhead <100ns per log call when disabled
- ✅ Circuit breaker decision time <1μs
- ✅ Rate limiter check time <1μs
- ✅ Overall throughput degradation <5% with all features enabled

### Quality Requirements

- ✅ Unit test coverage ≥80%
- ✅ Integration tests pass
- ✅ No race conditions (verified with `-race`)
- ✅ No memory leaks
- ✅ All linters pass

---

## Related Documents

- PRD-INFRASTRUCTURE.md - Infrastructure requirements
- IMPL-M4-CACHE.md - Cache implementation guide
- IMPL-M4-RESILIENCE.md - Resilience implementation guide
- IMPL-M4-OBSERVABILITY.md - Observability implementation guide

---

**END OF M4-INTEGRATION-NEXT-STEPS.md**
