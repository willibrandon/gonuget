# gonuget - Product Requirements Document: Infrastructure

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-10-19
**Owner:** Engineering

---

## Table of Contents

1. [Overview](#overview)
2. [HTTP Client](#http-client)
3. [Retry Logic](#retry-logic)
4. [Caching](#caching)
5. [Circuit Breaker](#circuit-breaker)
6. [Rate Limiting](#rate-limiting)
7. [Observability](#observability)
8. [Acceptance Criteria](#acceptance-criteria)

---

## Overview

This document specifies infrastructure requirements including HTTP client configuration, retry logic, caching strategies, circuit breakers, rate limiting, and observability.

**Related Design Documents:**
- DESIGN-HTTP.md - HTTP client and resilience patterns

---

## HTTP Client

### Requirement HTTP-001: HTTP Client Configuration

**Priority:** P0 (Critical)
**Component:** `http` package

**Description:**
Configurable HTTP client with modern protocol support.

**Functional Requirements:**

1. **Protocol support:**
   - HTTP/1.1 (baseline)
   - HTTP/2 (multiplexing)
   - HTTP/3 (optional, via QUIC)

2. **Connection pooling:**
   - Persistent connections
   - Configurable pool size
   - Idle connection timeout

3. **Timeouts:**
   - Connection timeout (default 10s)
   - Request timeout (default 30s)
   - Response header timeout (default 10s)

4. **TLS:**
   - TLS 1.2+ required
   - Certificate verification enabled
   - Custom CA certificates support

**API:**
```go
type HTTPClientConfig struct {
    Timeout time.Duration
    ConnectionTimeout time.Duration
    MaxIdleConns int
    MaxConnsPerHost int
    TLSConfig *tls.Config
    HTTP2Enabled bool
    HTTP3Enabled bool
}

func NewHTTPClient(config *HTTPClientConfig) *http.Client
```

**Functional Options:**
```go
func WithTimeout(d time.Duration) HTTPClientOption
func WithHTTP2() HTTPClientOption
func WithHTTP3() HTTPClientOption
func WithTLSConfig(config *tls.Config) HTTPClientOption
```

**Acceptance Criteria:**
- ✅ Supports HTTP/1.1, HTTP/2
- ✅ HTTP/3 support optional
- ✅ Connection pooling configured
- ✅ TLS verification enforced

---

### Requirement HTTP-002: Request Building

**Priority:** P0 (Critical)
**Component:** `http` package

**Description:**
Build HTTP requests with headers and authentication.

**Functional Requirements:**

1. **Headers:**
   - User-Agent: `gonuget/{version} (+https://github.com/...)`
   - Accept: application/json (v3), application/xml (v2)
   - Accept-Encoding: gzip, deflate

2. **Authentication:**
   - API key header
   - Bearer token
   - Basic auth

3. **Context propagation:**
   - Attach context.Context to requests
   - Cancel on context cancellation

**API:**
```go
type RequestBuilder struct {
    method string
    url string
    headers map[string]string
    body io.Reader
}

func NewRequest(method, url string) *RequestBuilder
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder
func (rb *RequestBuilder) WithAuth(creds Credentials) *RequestBuilder
func (rb *RequestBuilder) WithBody(body io.Reader) *RequestBuilder
func (rb *RequestBuilder) Build(ctx context.Context) (*http.Request, error)
```

**Acceptance Criteria:**
- ✅ Builds requests with headers
- ✅ Authentication applied
- ✅ Context attached

---

## Retry Logic

### Requirement RETRY-001: Exponential Backoff

**Priority:** P0 (Critical)
**Component:** `http/retry` package

**Description:**
Retry failed requests with exponential backoff.

**Retry Strategy:**

1. **Retryable conditions:**
   - Network errors (connection refused, timeout)
   - HTTP 429 (Too Many Requests)
   - HTTP 500, 502, 503, 504 (Server errors)
   - NOT retryable: 4xx (except 429), 2xx, 3xx

2. **Backoff algorithm:**
   - Initial delay: 1s
   - Multiplier: 2
   - Max delay: 30s
   - Jitter: ±25% randomization

3. **Max retries:**
   - Default: 3 retries
   - Configurable: 0-10 retries

**Backoff Calculation:**
```
delay = min(initial_delay * (multiplier ^ attempt), max_delay)
actual_delay = delay * (1 + jitter * rand(-1, 1))
```

**API:**
```go
type RetryConfig struct {
    MaxRetries int
    InitialDelay time.Duration
    MaxDelay time.Duration
    Multiplier float64
    Jitter float64
}

type RetryHandler struct {
    config *RetryConfig
}

func NewRetryHandler(config *RetryConfig) *RetryHandler
func (rh *RetryHandler) Do(ctx context.Context, req *http.Request) (*http.Response, error)
```

**Acceptance Criteria:**
- ✅ Retries on network errors
- ✅ Retries on 429, 5xx
- ✅ Exponential backoff works
- ✅ Jitter applied
- ✅ Max retries honored

---

### Requirement RETRY-002: Retry-After Header

**Priority:** P0 (Critical)
**Component:** `http/retry` package

**Description:**
Parse and honor Retry-After header from HTTP 429/503 responses.

**Retry-After Formats:**

1. **Seconds:**
   ```
   Retry-After: 120
   ```
   Wait 120 seconds before retry.

2. **HTTP Date:**
   ```
   Retry-After: Wed, 21 Oct 2015 07:28:00 GMT
   ```
   Wait until specified time.

**Functional Requirements:**

1. **Parse header:**
   - Try parsing as integer (seconds)
   - Try parsing as HTTP date (RFC 7231)
   - Return duration to wait

2. **Apply wait:**
   - Override calculated backoff
   - Use Retry-After value
   - Respect max delay cap

3. **Context cancellation:**
   - Check context during wait
   - Cancel immediately if context done

**API:**
```go
func parseRetryAfter(header string) time.Duration
func (rh *RetryHandler) waitRetryAfter(ctx context.Context, duration time.Duration) error
```

**Acceptance Criteria:**
- ✅ Parses seconds format
- ✅ Parses HTTP date format
- ✅ Honors Retry-After over backoff
- ✅ Context cancellation during wait

---

## Caching

### Requirement CACHE-001: Multi-Tier Cache

**Priority:** P0 (Critical)
**Component:** `cache` package

**Description:**
Multi-tier caching (memory + disk) with TTL.

**Cache Tiers:**

1. **Memory cache (L1):**
   - LRU eviction
   - Configurable size (default 10,000 entries)
   - Fast access (<5ms)

2. **Disk cache (L2):**
   - Persistent across restarts
   - Larger capacity (default 1GB)
   - Slower access (<50ms)

**Cache Keys:**
- Service index: `service-index:{feed-url}`
- Metadata: `metadata:{feed-url}:{id}:{version}`
- Search: `search:{feed-url}:{query}:{options-hash}`

**Functional Requirements:**

1. **Get:**
   - Check L1 (memory)
   - On miss, check L2 (disk)
   - On hit in L2, promote to L1

2. **Set:**
   - Write to L1
   - Async write to L2
   - Apply TTL

3. **Eviction:**
   - LRU for memory cache
   - Size-based for disk cache
   - TTL expiration

4. **Invalidation:**
   - By key
   - By prefix (e.g., all metadata for feed)
   - Clear all

**API:**
```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
}

type MultiTierCache struct {
    l1 *MemoryCache
    l2 *DiskCache
}

func NewMultiTierCache(l1 *MemoryCache, l2 *DiskCache) *MultiTierCache
```

**Cache Configuration:**
```go
type CacheConfig struct {
    MemorySize int // Max entries in L1
    DiskSize int64 // Max bytes in L2
    DefaultTTL time.Duration
}
```

**Acceptance Criteria:**
- ✅ Two-tier caching works
- ✅ LRU eviction for memory
- ✅ TTL expiration
- ✅ Persistence across restarts (L2)

---

### Requirement CACHE-002: Cache Validation

**Priority:** P1 (High)
**Component:** `cache` package

**Description:**
Validate cached entries before use.

**Validation Methods:**

1. **TTL-based:**
   - Check expiry time
   - Evict if expired

2. **ETag-based:**
   - Store ETag with cached entry
   - Conditional GET: `If-None-Match: {etag}`
   - 304 Not Modified → cache valid

3. **Last-Modified:**
   - Store timestamp
   - Conditional GET: `If-Modified-Since: {timestamp}`
   - 304 Not Modified → cache valid

**API:**
```go
type CacheEntry struct {
    Value []byte
    Expiry time.Time
    ETag string
    LastModified time.Time
}

func (c *Cache) GetWithValidation(ctx context.Context, key string, validator func() (bool, error)) (*CacheEntry, error)
```

**Acceptance Criteria:**
- ✅ TTL validation
- ✅ ETag revalidation
- ✅ Conditional GET support

---

## Circuit Breaker

### Requirement CB-001: Circuit Breaker Pattern

**Priority:** P1 (High)
**Component:** `circuitbreaker` package

**Description:**
Implement circuit breaker to prevent cascading failures.

**States:**

1. **Closed** (normal):
   - All requests pass through
   - Count failures
   - Transition to Open on threshold

2. **Open** (failing):
   - Reject requests immediately
   - Return error without attempting
   - Transition to Half-Open after timeout

3. **Half-Open** (testing):
   - Allow limited requests through
   - If success → Closed
   - If failure → Open

**Configuration:**
- Failure threshold: 5 failures
- Open timeout: 30s
- Half-open max requests: 3

**Functional Requirements:**

1. **Failure detection:**
   - Count consecutive failures
   - Trigger on network errors and 5xx responses

2. **State transitions:**
   - Closed → Open: After N failures
   - Open → Half-Open: After timeout
   - Half-Open → Closed: After successful request
   - Half-Open → Open: On failure

3. **Request handling:**
   - Closed: Execute normally
   - Open: Return error immediately (fast fail)
   - Half-Open: Execute limited requests

**API:**
```go
type CircuitBreaker struct {
    state State
    failures int
    threshold int
    timeout time.Duration
}

type State int
const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error
func (cb *CircuitBreaker) State() State
```

**Acceptance Criteria:**
- ✅ State transitions work correctly
- ✅ Fast-fail in Open state
- ✅ Recovery in Half-Open state
- ✅ Prevents cascading failures

---

## Rate Limiting

### Requirement RL-001: Token Bucket Rate Limiting

**Priority:** P1 (High)
**Component:** `ratelimit` package

**Description:**
Rate limit requests per source using token bucket algorithm.

**Algorithm:**

1. **Bucket:**
   - Capacity: Maximum burst size
   - Refill rate: Tokens per second

2. **Consumption:**
   - Each request consumes 1 token
   - If tokens available → Allow
   - If no tokens → Wait or reject

3. **Refill:**
   - Continuous refill at rate
   - Cap at capacity

**Configuration:**
- Default rate: 100 requests/second
- Default burst: 200 requests
- Per-source limits

**Functional Requirements:**

1. **Acquire token:**
   - Check available tokens
   - Consume token if available
   - Wait if unavailable (up to timeout)

2. **Per-source limits:**
   - Separate bucket per NuGet source
   - Different limits per source

3. **Graceful degradation:**
   - Don't fail requests, just delay
   - Respect context cancellation

**API:**
```go
type RateLimiter struct {
    rate float64 // Tokens per second
    burst int // Bucket capacity
}

func NewRateLimiter(rate float64, burst int) *RateLimiter
func (rl *RateLimiter) Wait(ctx context.Context) error
func (rl *RateLimiter) Allow() bool
```

**Acceptance Criteria:**
- ✅ Limits requests to configured rate
- ✅ Allows bursts up to capacity
- ✅ Per-source limiting
- ✅ Context cancellation honored

---

## Observability

### Requirement OBS-001: Structured Logging (mtlog)

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Integrate mtlog for structured logging throughout library.

**Log Levels:**
- Verbose: Detailed diagnostics
- Debug: Development information
- Information: Important events
- Warning: Potential issues
- Error: Failures
- Fatal: Critical failures

**Logging Requirements:**

1. **Structured properties:**
   ```go
   log.Information("Fetching package {PackageId} version {Version} from {Source}",
       packageId, version, sourceUrl)
   ```

2. **Contextual logging:**
   - Request ID
   - Source URL
   - Package identity
   - Operation name

3. **Performance:**
   - No logging overhead when disabled
   - Lazy evaluation of expensive operations

**API:**
```go
type Logger interface {
    Verbose(template string, args ...any)
    Debug(template string, args ...any)
    Information(template string, args ...any)
    Warning(template string, args ...any)
    Error(template string, args ...any)
    Fatal(template string, args ...any)
}

func SetLogger(logger Logger)
func GetLogger() Logger
```

**Standard Properties:**
- `PackageId` - Package ID
- `Version` - Package version
- `SourceUrl` - Feed URL
- `OperationId` - Unique operation ID
- `Duration` - Operation duration

**Acceptance Criteria:**
- ✅ mtlog integrated throughout
- ✅ All operations logged
- ✅ Structured properties used
- ✅ No performance impact when disabled

---

### Requirement OBS-002: OpenTelemetry Tracing

**Priority:** P1 (High)
**Component:** `observability` package

**Description:**
Integrate OpenTelemetry for distributed tracing.

**Spans:**

1. **HTTP requests:**
   - Span per HTTP call
   - Include URL, method, status code
   - Duration timing

2. **Operations:**
   - Search operation span
   - Download operation span
   - Dependency resolution span

3. **Context propagation:**
   - W3C Trace Context headers
   - Propagate across HTTP calls

**Functional Requirements:**

1. **Span creation:**
   ```go
   ctx, span := tracer.Start(ctx, "FetchPackageMetadata",
       trace.WithAttributes(
           attribute.String("package.id", packageId),
           attribute.String("package.version", version.String()),
       ))
   defer span.End()
   ```

2. **Automatic instrumentation:**
   - HTTP client instrumented
   - Spans created for operations

3. **Error recording:**
   - Record errors in spans
   - Set span status

**API:**
```go
func InitTracing(serviceName string) (*trace.TracerProvider, error)
func GetTracer() trace.Tracer
```

**Span Attributes:**
- `package.id`
- `package.version`
- `source.url`
- `http.method`
- `http.status_code`
- `error.message`

**Acceptance Criteria:**
- ✅ OTEL tracing integrated
- ✅ Spans created for operations
- ✅ Context propagated
- ✅ Exportable to Jaeger, Zipkin, etc.

---

### Requirement OBS-003: Prometheus Metrics

**Priority:** P1 (High)
**Component:** `observability` package

**Description:**
Expose Prometheus metrics for monitoring.

**Metrics:**

1. **Counters:**
   - `gonuget_requests_total{source, operation, status}`
   - `gonuget_errors_total{source, operation, error_type}`
   - `gonuget_cache_hits_total{cache_tier}`
   - `gonuget_cache_misses_total{cache_tier}`

2. **Histograms:**
   - `gonuget_request_duration_seconds{source, operation}`
   - `gonuget_download_size_bytes{source}`

3. **Gauges:**
   - `gonuget_circuit_breaker_state{source}` (0=closed, 1=open, 2=half-open)
   - `gonuget_rate_limiter_available_tokens{source}`

**Functional Requirements:**

1. **Metric collection:**
   - Increment counters on events
   - Record durations in histograms
   - Update gauges on state changes

2. **Labels:**
   - Source URL
   - Operation type
   - HTTP status code
   - Error type

3. **Exposition:**
   - `/metrics` endpoint (optional)
   - Prometheus format
   - OpenMetrics format

**API:**
```go
func InitMetrics() *prometheus.Registry
func RecordRequest(source, operation string, duration time.Duration, status int)
func RecordError(source, operation, errorType string)
func RecordCacheHit(tier string)
```

**Acceptance Criteria:**
- ✅ Metrics collected
- ✅ Prometheus format
- ✅ Labels applied correctly
- ✅ Scrapable by Prometheus

---

### Requirement OBS-004: Health Checks

**Priority:** P1 (High)
**Component:** `observability` package

**Description:**
Provide health check endpoints for monitoring.

**Health Checks:**

1. **Liveness:**
   - Library loaded
   - No panics
   - Basic functionality

2. **Readiness:**
   - Can reach NuGet sources
   - Cache accessible
   - Dependencies available

**Health Status:**
- `Healthy` - All checks pass
- `Degraded` - Some non-critical failures
- `Unhealthy` - Critical failures

**Functional Requirements:**

1. **Check sources:**
   - Ping each configured source
   - Report individual status

2. **Check cache:**
   - Verify cache accessible
   - Check disk space

3. **Check circuit breakers:**
   - Report open circuit breakers

**API:**
```go
type HealthCheck struct {
    Name string
    Status HealthStatus
    Message string
}

type HealthStatus int
const (
    Healthy HealthStatus = iota
    Degraded
    Unhealthy
)

func GetHealth(ctx context.Context) ([]*HealthCheck, error)
```

**Acceptance Criteria:**
- ✅ Health checks implemented
- ✅ Source connectivity checked
- ✅ Cache status checked
- ✅ Circuit breaker status included

---

## Acceptance Criteria

### HTTP Client

**Functional:**
- ✅ HTTP/1.1, HTTP/2 support
- ✅ Connection pooling configured
- ✅ TLS verification enforced
- ✅ Timeouts configured

**Performance:**
- ✅ Connection reuse works
- ✅ HTTP/2 multiplexing enabled
- ✅ No connection leaks

### Retry Logic

**Functional:**
- ✅ Retries on transient failures
- ✅ Exponential backoff applied
- ✅ Retry-After header honored
- ✅ Max retries enforced

**Robustness:**
- ✅ Context cancellation during retry
- ✅ No infinite retries
- ✅ Jitter prevents thundering herd

### Caching

**Functional:**
- ✅ Two-tier cache works
- ✅ LRU eviction
- ✅ TTL expiration
- ✅ Disk persistence

**Performance:**
- ✅ Cache hit <5ms (memory)
- ✅ Cache hit <50ms (disk)
- ✅ Reduces network requests 80%+

### Circuit Breaker

**Functional:**
- ✅ State transitions correct
- ✅ Fast-fail in Open state
- ✅ Recovery mechanism works

**Robustness:**
- ✅ Prevents cascading failures
- ✅ Automatic recovery
- ✅ Configurable thresholds

### Observability

**Functional:**
- ✅ mtlog logging integrated
- ✅ OTEL tracing works
- ✅ Prometheus metrics exposed
- ✅ Health checks implemented

**Usability:**
- ✅ Logs structured and searchable
- ✅ Traces visualizable in Jaeger
- ✅ Metrics scrapable by Prometheus
- ✅ Health checks useful for monitoring

---

## Related Documents

- PRD-OVERVIEW.md - Product vision and goals
- PRD-CORE.md - Core library requirements
- PRD-PROTOCOL.md - Protocol implementation
- PRD-PACKAGING.md - Package operations
- PRD-TESTING.md - Testing requirements
- PRD-RELEASE.md - Release criteria

---

**END OF PRD-INFRASTRUCTURE.md**
