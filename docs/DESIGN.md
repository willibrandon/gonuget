# gonuget Design Document

**Version**: 1.0.0
**Date**: 2025-01-19
**Status**: Draft
**Author**: Brandon Williams

---

## Table of Contents

1. [Overview](#overview)
2. [Design Principles](#design-principles)
3. [Architecture](#architecture)
4. [Package Structure](#package-structure)
5. [Core Abstractions](#core-abstractions)
6. [Component Overview](#component-overview)
7. [Cross-Cutting Concerns](#cross-cutting-concerns)
8. [Performance Targets](#performance-targets)
9. [Security Considerations](#security-considerations)
10. [Migration from C# Client](#migration-from-c-client)

---

## Overview

### Mission Statement

Build a **comprehensive, enterprise-grade NuGet client library for Go** that meets and exceeds the functionality, performance, and developer experience of the official C# NuGet.Client.

### Goals

1. **100% Feature Parity**: Support all core NuGet operations (search, download, install, pack, sign, push)
2. **Superior Performance**: 2-3x faster than C# client for common operations
3. **Production Ready**: Battle-tested reliability with comprehensive error handling
4. **Developer Experience**: Go-idiomatic API, extensive documentation, easy testing
5. **Modern Architecture**: HTTP/3, OpenTelemetry, structured logging, circuit breakers

### Non-Goals

- **GUI Client**: gonuget is a library and CLI tool, not a graphical application
- **Visual Studio Integration**: No IDE plugin support (library can be used by IDE plugins)
- **MSBuild Engine**: No full MSBuild execution (only .csproj XML manipulation)
- **Source Repository Hosting**: No package feed hosting capabilities

---

## Design Principles

### 1. Go-Idiomatic Design

```go
// Good: Context-first, functional options
client := gonuget.NewClient(
    gonuget.WithSource("https://api.nuget.org/v3/index.json"),
    gonuget.WithCache(cache),
    gonuget.WithLogger(logger),
)
results, err := client.Search(ctx, "newtonsoft", gonuget.WithPrerelease(true))

// Bad: Builder pattern, C#-style fluent API
client := gonuget.NewClientBuilder().
    SetSource("https://api.nuget.org/v3/index.json").
    SetCache(cache).
    Build()
```

### 2. Context Propagation

All operations accept `context.Context` as the first parameter for:
- Cancellation support
- Deadline enforcement
- Distributed tracing
- Request-scoped values

```go
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOption) (*SearchResult, error)
func (c *Client) Download(ctx context.Context, id, version string, opts ...DownloadOption) (*Package, error)
```

### 3. Explicit Error Handling

Rich error types with actionable information:

```go
type PackageNotFoundError struct {
    PackageID string
    Version   string
    Source    string
}

type NetworkError struct {
    Operation string
    URL       string
    Attempts  int
    Err       error
}

type AuthenticationError struct {
    Source string
    Err    error
}
```

### 4. Zero-Allocation Hot Paths

Performance-critical paths avoid allocations:

```go
// Version comparison: zero allocations
func (v *NuGetVersion) Compare(other *NuGetVersion) int {
    // No string allocations, direct integer comparison
    if v.Major != other.Major {
        return v.Major - other.Major
    }
    // ... continue comparison
}
```

### 5. Safe Concurrency

```go
// Thread-safe caching with double-check pattern
type SafeCache struct {
    mu    sync.RWMutex
    cache map[string]*CachedItem
    sem   chan struct{} // Limit concurrent fetches
}

func (c *SafeCache) GetOrFetch(ctx context.Context, key string) (*Item, error) {
    // Fast read path
    c.mu.RLock()
    item, ok := c.cache[key]
    c.mu.RUnlock()
    if ok && item.Valid() {
        return item.Value, nil
    }

    // Acquire semaphore for fetch
    select {
    case c.sem <- struct{}{}:
        defer func() { <-c.sem }()
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    // Double-check inside write lock
    c.mu.Lock()
    defer c.mu.Unlock()
    if item, ok := c.cache[key]; ok && item.Valid() {
        return item.Value, nil
    }

    // Fetch and cache
    value, err := c.fetch(ctx, key)
    if err != nil {
        return nil, err
    }
    c.cache[key] = &CachedItem{Value: value, Timestamp: time.Now()}
    return value, nil
}
```

### 6. Dependency Injection

Use interfaces for testability:

```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

type Client struct {
    http    HTTPClient      // Injectable for testing
    cache   Cache           // Injectable for testing
    logger  Logger          // Injectable for testing
}

// Production
client := gonuget.NewClient(gonuget.WithHTTPClient(httpClient))

// Testing
mock := &MockHTTPClient{responses: testResponses}
client := gonuget.NewClient(gonuget.WithHTTPClient(mock))
```

### 7. Progressive Disclosure

Simple things should be simple, complex things should be possible:

```go
// Simple: Single source, default options
client := gonuget.NewClient()
results, err := client.Search(ctx, "serilog")

// Advanced: Multiple sources, custom auth, caching, retry
client := gonuget.NewClient(
    gonuget.WithSources(
        gonuget.Source{Name: "nuget.org", URL: nugetOrgURL, Protocol: ProtocolV3},
        gonuget.Source{Name: "myget", URL: mygetURL, Auth: apiKeyAuth},
    ),
    gonuget.WithCache(customCache),
    gonuget.WithRetry(retryConfig),
    gonuget.WithCircuitBreaker(circuitConfig),
    gonuget.WithLogger(logger),
)
```

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application Layer                        │
│                    (CLI, LazyNuGet TUI, etc.)                   │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      gonuget Public API                          │
│  Client, Search, Download, Install, Pack, Sign, Push, Verify   │
└─────────────────────────────────────────────────────────────────┘
                                 │
                ┌────────────────┼────────────────┐
                ▼                ▼                ▼
      ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
      │   Protocol   │  │  Versioning  │  │  Frameworks  │
      │   (V2/V3)    │  │   (SemVer)   │  │    (TFM)     │
      └──────────────┘  └──────────────┘  └──────────────┘
                ▼                ▼                ▼
      ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
      │   Packaging  │  │     HTTP     │  │Configuration │
      │ (ZIP/.nuspec)│  │ (Retry/Cache)│  │ (nuget.config)│
      └──────────────┘  └──────────────┘  └──────────────┘
                                 │
                ┌────────────────┼────────────────┐
                ▼                ▼                ▼
      ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
      │     Auth     │  │  Telemetry   │  │    Errors    │
      │   (Plugins)  │  │(Logs/Metrics)│  │  (Rich Types)│
      └──────────────┘  └──────────────┘  └──────────────┘
```

### Layered Architecture

#### Layer 1: Core Primitives
- **version/**: Version parsing, comparison, ranges (NuGet SemVer)
- **framework/**: TFM parsing, compatibility checking
- **models/**: Core types (PackageIdentity, PackageMetadata, Dependency, etc.)

#### Layer 2: Infrastructure
- **http/**: HTTP client with retry, caching, circuit breaker, rate limiting
- **auth/**: Authentication providers (Basic, Bearer, NTLM, OAuth2, API Key)
- **cache/**: Multi-tier caching (memory + disk)
- **telemetry/**: Logging (mtlog), metrics, distributed tracing (OpenTelemetry)
- **errors/**: Rich error types with context

#### Layer 3: Protocol Implementation
- **api/v3/**: NuGet v3 API (service discovery, search, download, metadata, publish)
- **api/v2/**: NuGet v2 API (OData protocol, legacy feeds)
- **feed/**: Source management, feed registry, local repositories

#### Layer 4: Package Operations
- **pack/**: Package creation, building, validation
- **signature/**: Package signing, verification, certificate handling
- **resolver/**: Dependency resolution, conflict detection, graph building
- **project/**: MSBuild project file manipulation (.csproj, .fsproj, .vbproj, .sln)

#### Layer 5: Public API
- **client.go**: Main client with high-level operations
- **search.go**: Search operations
- **download.go**: Download and install operations
- **publish.go**: Pack, sign, push operations
- **config.go**: Configuration management

### Resource Provider Pattern

Inspired by C# NuGet.Client's resource provider architecture:

```go
// Resource represents a capability (search, download, metadata, etc.)
type Resource interface {
    Type() ResourceType
}

// ResourceProvider creates resources for a given source
type ResourceProvider interface {
    TryCreate(ctx context.Context, source *Source) (Resource, error)
    ResourceType() ResourceType
    Priority() int
}

// SourceRepository aggregates resources for a package source
type SourceRepository struct {
    source    *Source
    providers []ResourceProvider
    resources sync.Map // Cache created resources
}

func (r *SourceRepository) GetResource(ctx context.Context, typ ResourceType) (Resource, error) {
    // Check cache
    if res, ok := r.resources.Load(typ); ok {
        return res.(Resource), nil
    }

    // Find provider for this resource type
    for _, provider := range r.providers {
        if provider.ResourceType() == typ {
            res, err := provider.TryCreate(ctx, r.source)
            if err != nil {
                continue // Try next provider
            }
            r.resources.Store(typ, res)
            return res, nil
        }
    }

    return nil, fmt.Errorf("no provider for resource type %s", typ)
}
```

**Built-in Resource Types**:
- `ServiceIndexResource` - Service discovery for v3 feeds
- `SearchResource` - Package search
- `DownloadResource` - Package download
- `MetadataResource` - Package metadata retrieval
- `PublishResource` - Package publishing
- `DependencyInfoResource` - Dependency information
- `AutoCompleteResource` - Package ID/version autocomplete

**Provider Priority**:
```go
const (
    PriorityFirst   = 1000  // Executes first
    PriorityDefault = 0     // Default priority
    PriorityLast    = -1000 // Executes last (fallback)
)

// ServiceIndexResourceV3Provider runs last (requires network)
// LocalPackageSearchResourceProvider runs first (local cache)
```

---

## Package Structure

```
github.com/willibrandon/gonuget/
├── cmd/
│   └── gonuget/                    # CLI tool
│       ├── main.go
│       ├── search.go
│       ├── download.go
│       ├── install.go
│       ├── pack.go
│       ├── sign.go
│       ├── push.go
│       └── verify.go
│
├── pkg/gonuget/                    # Public API
│   ├── client.go                   # Main client
│   ├── search.go                   # Search operations
│   ├── download.go                 # Download/install operations
│   ├── publish.go                  # Pack/sign/push operations
│   ├── config.go                   # Configuration
│   ├── options.go                  # Functional options
│   │
│   ├── models/                     # Core types
│   │   ├── package.go              # PackageIdentity, PackageMetadata
│   │   ├── dependency.go           # Dependency, DependencyGroup
│   │   ├── source.go               # Source, SourceRepository
│   │   └── search.go               # SearchResult, SearchMetadata
│   │
│   ├── version/                    # Semantic versioning
│   │   ├── version.go              # NuGetVersion
│   │   ├── range.go                # VersionRange, FloatRange
│   │   ├── compare.go              # VersionComparer
│   │   ├── parser.go               # Version parsing
│   │   └── normalize.go            # Version normalization
│   │
│   ├── framework/                  # Target frameworks
│   │   ├── framework.go            # NuGetFramework
│   │   ├── parser.go               # TFM parsing
│   │   ├── compat.go               # CompatibilityProvider
│   │   ├── mappings.go             # Framework identifier mappings
│   │   └── mappings_generated.go   # Generated from C# DefaultFrameworkMappings
│   │
│   ├── http/                       # HTTP infrastructure
│   │   ├── client.go               # HTTPClient with pooling
│   │   ├── retry.go                # RetryHandler with backoff
│   │   ├── cache.go                # HTTP response caching
│   │   ├── circuit.go              # CircuitBreaker for resilience
│   │   ├── ratelimit.go            # RateLimiter
│   │   ├── progress.go             # Progress tracking
│   │   └── middleware.go           # HTTP middleware chain
│   │
│   ├── auth/                       # Authentication
│   │   ├── auth.go                 # AuthProvider interface
│   │   ├── basic.go                # BasicAuthProvider
│   │   ├── bearer.go               # BearerTokenProvider
│   │   ├── apikey.go               # APIKeyProvider
│   │   ├── ntlm.go                 # NTLMAuthProvider (Windows)
│   │   ├── oauth2.go               # OAuth2Provider (Azure DevOps)
│   │   ├── store.go                # TokenStore (caching)
│   │   └── plugin.go               # CredentialPlugin (future)
│   │
│   ├── cache/                      # Caching infrastructure
│   │   ├── cache.go                # Cache interface
│   │   ├── memory.go               # MemoryCache (LRU)
│   │   ├── disk.go                 # DiskCache (file-based)
│   │   ├── multi.go                # MultiTierCache (memory + disk)
│   │   └── stats.go                # Cache statistics
│   │
│   ├── api/                        # Protocol implementations
│   │   ├── v3/
│   │   │   ├── index.go            # Service index discovery
│   │   │   ├── search.go           # SearchQueryService
│   │   │   ├── download.go         # PackageBaseAddress
│   │   │   ├── metadata.go         # RegistrationsBaseUrl
│   │   │   ├── publish.go          # PackagePublish
│   │   │   ├── autocomplete.go     # SearchAutocompleteService
│   │   │   ├── types.go            # V3 data structures
│   │   │   └── converters.go       # JSON unmarshaling (string/array)
│   │   └── v2/
│   │       ├── odata.go            # OData protocol
│   │       ├── search.go           # V2 search (Packages())
│   │       ├── download.go         # V2 download
│   │       └── types.go            # V2 data structures
│   │
│   ├── feed/                       # Feed management
│   │   ├── source.go               # Source configuration
│   │   ├── registry.go             # SourceRegistry
│   │   ├── local.go                # LocalPackageRepository
│   │   └── aggregator.go           # AggregateRepository (multi-source)
│   │
│   ├── pack/                       # Package creation
│   │   ├── builder.go              # PackageBuilder
│   │   ├── writer.go               # NupkgWriter (ZIP creation)
│   │   ├── reader.go               # NupkgReader (ZIP reading)
│   │   ├── nuspec.go               # NuspecWriter/NuspecReader (XML)
│   │   ├── validator.go            # PackageValidator
│   │   └── manifest.go             # ManifestMetadata
│   │
│   ├── signature/                  # Package signing
│   │   ├── signer.go               # PackageSigner
│   │   ├── verifier.go             # SignatureVerifier
│   │   ├── cert.go                 # Certificate handling
│   │   ├── timestamp.go            # RFC 3161 timestamping
│   │   └── trust.go                # TrustProvider
│   │
│   ├── resolver/                   # Dependency resolution
│   │   ├── resolver.go             # DependencyResolver
│   │   ├── graph.go                # DependencyGraph
│   │   ├── conflict.go             # ConflictDetector
│   │   ├── strategy.go             # ResolutionStrategy
│   │   └── nearest.go              # NearestWinsStrategy
│   │
│   ├── project/                    # MSBuild project files
│   │   ├── csproj.go               # CSharpProject
│   │   ├── fsproj.go               # FSharpProject
│   │   ├── vbproj.go               # VisualBasicProject
│   │   ├── sln.go                  # Solution
│   │   ├── packages.go             # PackagesConfig (legacy)
│   │   └── msbuild.go              # MSBuild XML manipulation
│   │
│   ├── telemetry/                  # Observability
│   │   ├── logger.go               # Logger interface (wraps mtlog)
│   │   ├── metrics.go              # Metrics collection
│   │   ├── tracing.go              # OpenTelemetry tracing
│   │   └── events.go               # Event definitions
│   │
│   └── errors/                     # Error types
│       ├── errors.go               # Base error types
│       ├── api.go                  # API errors (404, 401, etc.)
│       ├── network.go              # Network errors (timeout, retry)
│       ├── package.go              # Package errors (invalid, corrupt)
│       └── auth.go                 # Authentication errors
│
├── internal/                       # Internal utilities
│   ├── resource/                   # Resource provider implementation
│   │   ├── provider.go             # ResourceProvider interface
│   │   ├── registry.go             # ProviderRegistry
│   │   └── repository.go           # SourceRepository
│   ├── pool/                       # Object pooling
│   │   ├── buffer.go               # Buffer pool
│   │   └── decoder.go              # JSON decoder pool
│   ├── testutil/                   # Test utilities
│   │   ├── mock.go                 # Mock HTTP client
│   │   ├── fixtures.go             # Test fixtures
│   │   └── server.go               # Test HTTP server
│   └── util/                       # Shared utilities
│       ├── normalize.go            # String normalization
│       ├── hash.go                 # Hash calculation
│       └── path.go                 # Path manipulation
│
├── examples/                       # Example code
│   ├── basic/
│   ├── advanced/
│   ├── authentication/
│   └── testing/
│
├── docs/                          # Documentation
│   ├── DESIGN.md                  # This file
│   ├── DESIGN-HTTP.md             # HTTP client design
│   ├── DESIGN-VERSIONING.md       # Versioning design
│   ├── DESIGN-FRAMEWORKS.md       # Frameworks design
│   ├── DESIGN-PROTOCOL.md         # Protocol design
│   ├── DESIGN-PACKAGING.md        # Packaging design
│   ├── DESIGN-TESTING.md          # Testing design
│   ├── API-REFERENCE.md           # API documentation
│   ├── MIGRATION.md               # Migration from C# client
│   └── guides/                    # User guides
│
├── testdata/                      # Test data
│   ├── fixtures/                  # JSON/XML fixtures
│   ├── packages/                  # Test .nupkg files
│   └── certificates/              # Test certificates
│
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── CHANGELOG.md
└── Makefile
```

---

## Core Abstractions

### Client

The main entry point for all operations:

```go
type Client struct {
    sources   []*SourceRepository
    http      *http.HTTPClient
    cache     cache.Cache
    logger    telemetry.Logger
    tracer    trace.Tracer
    resolver  *resolver.DependencyResolver
    config    *Config
}

func NewClient(opts ...ClientOption) *Client
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOption) (*SearchResult, error)
func (c *Client) Download(ctx context.Context, id, version string, opts ...DownloadOption) (*Package, error)
func (c *Client) Install(ctx context.Context, projectPath, id, version string, opts ...InstallOption) error
func (c *Client) Pack(ctx context.Context, nuspecPath string, opts ...PackOption) (*Package, error)
func (c *Client) Sign(ctx context.Context, pkgPath string, opts ...SignOption) error
func (c *Client) Push(ctx context.Context, pkgPath string, opts ...PushOption) error
func (c *Client) Verify(ctx context.Context, pkgPath string, opts ...VerifyOption) (*VerificationResult, error)
```

### Source

Represents a package source (feed):

```go
type Source struct {
    Name     string
    URL      string
    Protocol Protocol  // ProtocolV2, ProtocolV3, ProtocolLocal
    Enabled  bool
    Auth     auth.Provider
}

type SourceRepository struct {
    Source    *Source
    providers []resource.Provider
    resources sync.Map
}

func (r *SourceRepository) GetResource(ctx context.Context, typ resource.Type) (resource.Resource, error)
```

### PackageIdentity

Uniquely identifies a package:

```go
type PackageIdentity struct {
    ID      string              // Case-insensitive
    Version *version.NuGetVersion
}

func (p *PackageIdentity) String() string {
    return fmt.Sprintf("%s %s", p.ID, p.Version)
}

func (p *PackageIdentity) Equals(other *PackageIdentity) bool {
    return strings.EqualFold(p.ID, other.ID) && p.Version.Equals(other.Version)
}
```

### PackageMetadata

Extended package information:

```go
type PackageMetadata struct {
    Identity      *PackageIdentity
    Title         string
    Description   string
    Authors       []string
    Owners        []string
    ProjectURL    string
    LicenseURL    string
    IconURL       string
    Tags          []string
    Dependencies  []*DependencyGroup
    Published     time.Time
    DownloadCount int64
    IsListed      bool
    IsPrerelease  bool
    RequireLicense bool
}
```

### Dependency

Package dependency specification:

```go
type Dependency struct {
    ID           string
    VersionRange *version.VersionRange
    Include      []string  // Assets to include
    Exclude      []string  // Assets to exclude
}

type DependencyGroup struct {
    TargetFramework *framework.NuGetFramework
    Dependencies    []*Dependency
}
```

---

## Component Overview

### 1. HTTP Client (pkg/gonuget/http/)

See [DESIGN-HTTP.md](./DESIGN-HTTP.md) for full details.

**Features**:
- Retry with exponential backoff + jitter
- HTTP response caching (memory + disk)
- Circuit breaker for fault tolerance
- Rate limiting (per-source)
- Retry-After header support
- Connection pooling
- Progress tracking
- OpenTelemetry integration

### 2. Versioning (pkg/gonuget/version/)

See [DESIGN-VERSIONING.md](./DESIGN-VERSIONING.md) for full details.

**Features**:
- NuGet-flavored SemVer 2.0 parsing
- 4-part version support (legacy)
- Version range parsing (`[1.0, 2.0)`)
- Floating version ranges (`1.0.*`, `1.0.0-*`)
- Version comparison (multiple modes)
- Version normalization

### 3. Frameworks (pkg/gonuget/framework/)

See [DESIGN-FRAMEWORKS.md](./DESIGN-FRAMEWORKS.md) for full details.

**Features**:
- TFM parsing (net8.0, netstandard2.1, etc.)
- Framework compatibility checking
- PCL (Portable Class Library) support
- Framework precedence and fallback
- Identifier mapping (short → full name)

### 4. Protocol (pkg/gonuget/api/)

See [DESIGN-PROTOCOL.md](./DESIGN-PROTOCOL.md) for full details.

**V3 API**:
- Service index discovery
- SearchQueryService
- PackageBaseAddress (download)
- RegistrationsBaseUrl (metadata)
- SearchAutocompleteService
- PackagePublish

**V2 API**:
- OData protocol
- Packages() search
- Download from /packages/{id}/{version}

### 5. Packaging (pkg/gonuget/pack/)

See [DESIGN-PACKAGING.md](./DESIGN-PACKAGING.md) for full details.

**Features**:
- Package creation (.nupkg ZIP format)
- .nuspec XML generation
- Package validation
- Package extraction
- Framework-specific content
- Package signing and verification

### 6. Testing (internal/testutil/, testdata/)

See [DESIGN-TESTING.md](./DESIGN-TESTING.md) for full details.

**Features**:
- Mock HTTP client with fixture support
- Golden file tests for JSON parsing
- Integration tests against real feeds
- Benchmark tests for performance
- Comprehensive test coverage (>90%)

---

## Cross-Cutting Concerns

### Logging (mtlog Integration)

```go
import "github.com/willibrandon/mtlog"

// Create logger
logger := mtlog.New(
    mtlog.WithConsoleTheme(sinks.LiterateTheme()),
    mtlog.WithSeq("http://localhost:5341"),
    mtlog.WithRollingFile("gonuget.log", 10*1024*1024),
)

// Inject into client
client := gonuget.NewClient(
    gonuget.WithLogger(logger),
    gonuget.WithLogLevel(gonuget.LogLevelDebug),
)

// Structured logging
logger.Info("Searching for {PackageId} in {Source}", pkgID, source)
// [12:34:56 INF] Searching for Newtonsoft.Json in nuget.org
```

**Log Levels**:
- `Verbose`: Detailed diagnostic information
- `Debug`: Internal state changes
- `Information`: General informational messages
- `Warning`: Potentially harmful situations
- `Error`: Error events that might allow operation to continue
- `Fatal`: Severe errors that cause termination

**Structured Properties**:
- `PackageId`: Package identifier
- `Version`: Package version
- `Source`: Feed URL
- `Operation`: Operation name (Search, Download, etc.)
- `Duration`: Operation duration
- `StatusCode`: HTTP status code
- `RequestId`: Correlation ID

### Metrics (OpenTelemetry)

```go
import "go.opentelemetry.io/otel/metric"

// Metrics
var (
    searchCounter     metric.Int64Counter   // Total searches
    downloadCounter   metric.Int64Counter   // Total downloads
    cacheHitRatio     metric.Float64Gauge   // Cache hit ratio
    requestDuration   metric.Int64Histogram // Request duration
    downloadSize      metric.Int64Histogram // Package size
)

// Record metrics
searchCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("source", source),
    attribute.String("query", query),
))

requestDuration.Record(ctx, duration.Milliseconds(), metric.WithAttributes(
    attribute.String("operation", "search"),
    attribute.Int("status_code", statusCode),
))
```

### Distributed Tracing (OpenTelemetry)

```go
import "go.opentelemetry.io/otel/trace"

// Create span
ctx, span := tracer.Start(ctx, "gonuget.Search",
    trace.WithSpanKind(trace.SpanKindClient),
    trace.WithAttributes(
        attribute.String("package.query", query),
        attribute.String("package.source", source),
    ),
)
defer span.End()

// Execute operation
results, err := c.searchResource.Search(ctx, query)
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return nil, err
}

// Add result metadata
span.SetAttributes(
    attribute.Int("package.count", len(results.Data)),
    attribute.Int64("duration_ms", duration.Milliseconds()),
)
```

**Trace Propagation**:
- W3C Trace Context headers (`traceparent`, `tracestate`)
- Automatic span creation for HTTP requests
- Parent-child span relationships
- Context propagation across goroutines

### Error Handling

Rich error types with stack traces and context:

```go
// Define error types
type PackageNotFoundError struct {
    PackageID string
    Version   string
    Source    string
}

func (e *PackageNotFoundError) Error() string {
    return fmt.Sprintf("package %s %s not found in %s", e.PackageID, e.Version, e.Source)
}

// Use errors.Is and errors.As
_, err := client.Download(ctx, "NonExistent", "1.0.0")
if err != nil {
    var notFound *PackageNotFoundError
    if errors.As(err, &notFound) {
        fmt.Printf("Package %s not found\n", notFound.PackageID)
    }
}
```

**Error Wrapping**:
```go
// Wrap errors with context
func (c *Client) downloadPackage(ctx context.Context, url string) error {
    resp, err := c.http.Get(ctx, url)
    if err != nil {
        return fmt.Errorf("failed to download from %s: %w", url, err)
    }
    // ...
}

// Unwrap to get root cause
rootErr := errors.Unwrap(err)
```

### Configuration

Configuration from multiple sources (precedence order):

1. Programmatic configuration (highest priority)
2. Environment variables
3. `~/.gonuget/config.json`
4. `nuget.config` (NuGet standard)
5. Default values (lowest priority)

```go
// Load configuration
config, err := gonuget.LoadConfig(
    gonuget.WithConfigFile("~/.gonuget/config.json"),
    gonuget.WithNuGetConfig("."), // Search for nuget.config
    gonuget.WithEnvironment(),    // Load from env vars
)

// Configuration structure
type Config struct {
    Sources   []*Source
    Cache     CacheConfig
    HTTP      HTTPConfig
    Logging   LoggingConfig
    Telemetry TelemetryConfig
}
```

### Cancellation and Timeouts

All operations respect context cancellation:

```go
// Timeout for search
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := client.Search(ctx, "serilog")
if errors.Is(err, context.DeadlineExceeded) {
    fmt.Println("Search timed out")
}

// User cancellation
ctx, cancel := context.WithCancel(context.Background())

// Cancel from another goroutine
go func() {
    time.Sleep(5 * time.Second)
    cancel()
}()

results, err := client.Search(ctx, "newtonsoft")
if errors.Is(err, context.Canceled) {
    fmt.Println("Search canceled by user")
}
```

---

## Performance Targets

### Latency (vs C# NuGet.Client)

| Operation | C# Baseline | gonuget Target | Improvement |
|-----------|-------------|----------------|-------------|
| Service discovery | 200ms | <50ms | 4x faster |
| Search (20 results) | 300ms | <100ms | 3x faster |
| Download 1MB package | 150ms | <100ms | 1.5x faster |
| Parse .nuspec | 5ms | <1ms | 5x faster |
| Version comparison | 100ns | <50ns | 2x faster |
| Cache hit (memory) | 10µs | <5µs | 2x faster |
| Cache hit (disk) | 1ms | <500µs | 2x faster |

### Throughput

- **Concurrent downloads**: 1000+ simultaneous downloads
- **Search throughput**: 1000+ searches/second (cached)
- **Version comparisons**: 10M+ comparisons/second

### Memory Usage

- **Client memory**: <50MB for typical usage
- **Cache size**: Configurable (default 100MB memory + 1GB disk)
- **Zero allocations**: Hot paths (version comparison, framework compat)

### Network Efficiency

- **HTTP/2 multiplexing**: Multiple requests over single connection
- **Connection reuse**: Keep-alive connections
- **Compression**: gzip, deflate, brotli support
- **Range requests**: Resume interrupted downloads

---

## Security Considerations

### 1. Package Signature Verification

All downloaded packages should be verified:

```go
verifier := gonuget.NewPackageVerifier(
    gonuget.WithTrustedCerts("trusted_certs.pem"),
    gonuget.WithRequireSignature(true),
    gonuget.WithRequireTimestamp(true),
)

result, err := verifier.Verify("package.nupkg")
if !result.IsValid {
    return fmt.Errorf("invalid signature: %s", result.Reason)
}
```

### 2. HTTPS by Default

All HTTP sources upgraded to HTTPS:

```go
source := gonuget.Source{
    URL: "http://api.nuget.org/v3/index.json", // HTTP
}
// Automatically upgraded to https://api.nuget.org/v3/index.json
```

### 3. Path Traversal Prevention

ZIP extraction validates paths:

```go
func extractPackage(zipPath, dest string) error {
    for _, file := range zipReader.File {
        // Prevent path traversal
        if strings.Contains(file.Name, "..") {
            return fmt.Errorf("invalid path: %s", file.Name)
        }

        // Ensure within destination
        fullPath := filepath.Join(dest, file.Name)
        if !strings.HasPrefix(fullPath, dest) {
            return fmt.Errorf("path outside destination: %s", file.Name)
        }
    }
}
```

### 4. Credential Storage

Credentials encrypted at rest:

```go
// Never log credentials
logger.Info("Authenticating to {Source}", source) // OK
logger.Debug("Using API key: {ApiKey}", apiKey)    // BAD - leaks credential

// Sanitize headers in logs
sanitize := []string{"Authorization", "X-NuGet-ApiKey"}
```

### 5. Dependency Confusion Prevention

Warn when package comes from unexpected source:

```go
// Corporate policy: internal packages must come from internal feed
if strings.HasPrefix(pkg.ID, "MyCompany.") {
    if pkg.Source != "https://internal-feed/" {
        return fmt.Errorf("package %s must come from internal feed, got %s", pkg.ID, pkg.Source)
    }
}
```

---

## Migration from C# Client

### API Mapping

| C# NuGet.Client | gonuget | Notes |
|-----------------|---------|-------|
| `NuGetVersion.Parse()` | `version.Parse()` | Similar API |
| `SourceRepository` | `SourceRepository` | Same concept, different implementation |
| `PackageSearchResource` | `SearchResource` | Interface-based |
| `FindPackageByIdResource` | `DownloadResource` | Simplified API |
| `ISettings` | `Config` | Configuration management |
| `ILogger` | `Logger` (mtlog) | Structured logging |

### Example Migration

**C# Code**:
```csharp
var cache = new SourceCacheContext();
var repository = Repository.Factory.GetCoreV3("https://api.nuget.org/v3/index.json");
var resource = await repository.GetResourceAsync<PackageSearchResource>();

var searchFilter = new SearchFilter(includePrerelease: true);
var results = await resource.SearchAsync("Newtonsoft.Json", searchFilter,
    skip: 0, take: 20, logger, CancellationToken.None);

foreach (var result in results) {
    Console.WriteLine($"{result.Identity.Id} {result.Identity.Version}");
}
```

**Go Code**:
```go
client := gonuget.NewClient(
    gonuget.WithSource("https://api.nuget.org/v3/index.json"),
)

ctx := context.Background()
results, err := client.Search(ctx, "Newtonsoft.Json",
    gonuget.WithPrerelease(true),
    gonuget.WithTake(20),
)
if err != nil {
    log.Fatal(err)
}

for _, pkg := range results.Data {
    fmt.Printf("%s %s\n", pkg.ID, pkg.Version)
}
```

### Key Differences

1. **Error Handling**: Go uses explicit error returns instead of exceptions
2. **Context**: Go uses `context.Context` instead of `CancellationToken`
3. **Options**: Go uses functional options instead of parameter objects
4. **Resources**: Go uses interfaces with dependency injection
5. **Async**: Go uses goroutines instead of `async/await`

---

## Next Steps

1. Read component-specific design documents:
   - [DESIGN-HTTP.md](./DESIGN-HTTP.md) - HTTP client details
   - [DESIGN-VERSIONING.md](./DESIGN-VERSIONING.md) - Versioning details
   - [DESIGN-FRAMEWORKS.md](./DESIGN-FRAMEWORKS.md) - Framework compatibility
   - [DESIGN-PROTOCOL.md](./DESIGN-PROTOCOL.md) - NuGet protocol implementation
   - [DESIGN-PACKAGING.md](./DESIGN-PACKAGING.md) - Package operations
   - [DESIGN-TESTING.md](./DESIGN-TESTING.md) - Testing strategy

2. Review implementation plan in `ROADMAP.md`

3. Begin Phase 1 implementation:
   - Bootstrap repository structure
   - Implement HTTP client with retry/cache
   - Extract framework mappings from C# client
   - Implement resource provider pattern

---

**Document Status**: Draft v1.0
**Last Updated**: 2025-01-19
**Next Review**: After Phase 1 completion
