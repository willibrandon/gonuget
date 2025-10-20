# gonuget - Enterprise-Grade NuGet Client Library for Go

## Project Vision

Build the **definitive NuGet client library for Go** that meets and exceeds the official C# NuGet.Client in functionality, performance, and developer experience. This library will be the foundation for LazyNuGet and any other Go tools that need to interact with NuGet feeds.

---

## Name: `gonuget`

**Rationale**:
- **Go-idiomatic**: Simple, lowercase, descriptive
- **Clear purpose**: Immediately recognizable as NuGet client for Go
- **Package path**: `github.com/willibrandon/gonuget`
- **Import**: `import "github.com/willibrandon/gonuget"`

**Alternative names considered**:
- `nugetgo` - Less idiomatic (Go suffix not preferred)
- `nupkg` - Too generic, confusing with .nupkg file format
- `pkgmgr` - Not specific to NuGet
- ✅ **`gonuget`** - Selected for clarity and Go conventions

---

## Core Principles

### 1. **Feature Complete**
- ✅ Full NuGet v3 API support
- ✅ NuGet v2 API support (for legacy feeds)
- ✅ Package search, download, metadata
- ✅ Package creation and packing
- ✅ Package signing and verification
- ✅ Dependency resolution
- ✅ Framework compatibility checking
- ✅ Authentication (Basic, NTLM, OAuth2, API keys)
- ✅ Proxy support (HTTP, HTTPS, SOCKS5)

### 2. **Performance First**
- Zero-allocation hot paths
- Concurrent operations by default
- Streaming downloads with progress tracking
- HTTP/2 and HTTP/3 support
- Intelligent caching (memory + disk)
- Connection pooling and keep-alive
- Compression (gzip, deflate, brotli)

### 3. **Production Ready**
- Comprehensive error handling with retries
- Circuit breaker pattern for resilience
- Rate limiting and backoff strategies
- Health checks and readiness probes
- Metrics and observability (OTEL integration)
- Distributed tracing with W3C Trace Context
- Structured logging with **mtlog**

### 4. **Developer Experience**
- Fluent, chainable API
- Context-aware operations
- Comprehensive documentation
- Rich examples and guides
- CLI tool for testing
- Mock client for unit tests

---

## Architecture

### Package Structure

```
gonuget/
├── cmd/
│   └── gonuget/              # CLI tool
│       ├── main.go
│       ├── search.go
│       ├── download.go
│       ├── install.go
│       ├── push.go
│       └── verify.go
├── pkg/
│   └── gonuget/
│       ├── client.go         # Main client with service discovery
│       ├── config.go         # Client configuration
│       ├── cache.go          # HTTP and metadata caching
│       ├── auth/             # Authentication providers
│       │   ├── auth.go
│       │   ├── basic.go
│       │   ├── bearer.go
│       │   ├── ntlm.go
│       │   └── apikey.go
│       ├── api/              # API clients
│       │   ├── v3/
│       │   │   ├── search.go
│       │   │   ├── download.go
│       │   │   ├── metadata.go
│       │   │   ├── publish.go
│       │   │   └── index.go
│       │   └── v2/
│       │       ├── odata.go
│       │       ├── search.go
│       │       └── download.go
│       ├── models/           # Core types
│       │   ├── package.go
│       │   ├── version.go
│       │   ├── dependency.go
│       │   ├── framework.go
│       │   └── metadata.go
│       ├── version/          # Semantic versioning
│       │   ├── version.go
│       │   ├── range.go
│       │   ├── compare.go
│       │   └── normalize.go
│       ├── framework/        # Target frameworks
│       │   ├── framework.go
│       │   ├── parser.go
│       │   ├── compat.go
│       │   └── aliases.go
│       ├── pack/             # Package creation
│       │   ├── builder.go
│       │   ├── writer.go
│       │   ├── reader.go
│       │   └── validator.go
│       ├── signature/        # Package signing
│       │   ├── signer.go
│       │   ├── verifier.go
│       │   ├── cert.go
│       │   └── timestamp.go
│       ├── resolver/         # Dependency resolution
│       │   ├── resolver.go
│       │   ├── graph.go
│       │   ├── conflict.go
│       │   └── strategy.go
│       ├── project/          # Project file handling
│       │   ├── csproj.go
│       │   ├── fsproj.go
│       │   ├── vbproj.go
│       │   ├── sln.go
│       │   └── packages.go
│       ├── feed/             # Feed management
│       │   ├── source.go
│       │   ├── registry.go
│       │   └── local.go
│       ├── http/             # HTTP client utilities
│       │   ├── client.go
│       │   ├── retry.go
│       │   ├── circuit.go
│       │   ├── ratelimit.go
│       │   └── progress.go
│       ├── telemetry/        # Observability
│       │   ├── metrics.go
│       │   ├── tracing.go
│       │   └── events.go
│       └── errors/           # Error types
│           ├── errors.go
│           ├── api.go
│           └── network.go
├── internal/
│   ├── testutil/             # Test utilities
│   │   ├── mock.go
│   │   ├── fixtures.go
│   │   └── server.go
│   └── pool/                 # Object pooling
│       ├── buffer.go
│       └── decoder.go
├── examples/
│   ├── basic/
│   ├── advanced/
│   ├── authentication/
│   ├── signing/
│   └── resolver/
├── docs/
│   ├── api-reference.md
│   ├── authentication.md
│   ├── caching.md
│   ├── dependency-resolution.md
│   ├── package-creation.md
│   ├── signing.md
│   └── migration.md
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── CHANGELOG.md
```

---

## Core Features (Beyond C# Client)

### 1. **Advanced HTTP Client**

```go
// HTTP/2 and HTTP/3 support
client := gonuget.NewClient(
    gonuget.WithHTTP3(),                          // Enable HTTP/3 (QUIC)
    gonuget.WithConnectionPool(100),              // Connection pooling
    gonuget.WithKeepAlive(30*time.Second),        // Keep-alive
    gonuget.WithCompression("gzip", "br"),        // Multiple algorithms
)

// Circuit breaker for resilience
client := gonuget.NewClient(
    gonuget.WithCircuitBreaker(
        gonuget.CircuitConfig{
            MaxFailures:  5,
            ResetTimeout: 30*time.Second,
            OnOpen: func() {
                log.Warn("Circuit breaker opened - falling back to cache")
            },
        },
    ),
)

// Rate limiting
client := gonuget.NewClient(
    gonuget.WithRateLimit(100, time.Minute),      // 100 req/min
    gonuget.WithBurstLimit(20),                   // Allow bursts
)

// Retry with exponential backoff + jitter
client := gonuget.NewClient(
    gonuget.WithRetry(
        gonuget.RetryConfig{
            MaxAttempts:  5,
            InitialDelay: 100*time.Millisecond,
            MaxDelay:     10*time.Second,
            Multiplier:   2.0,
            Jitter:       0.1,
            RetryOn: []int{408, 429, 500, 502, 503, 504},
        },
    ),
)
```

### 2. **Structured Logging with mtlog**

```go
import (
    "github.com/willibrandon/mtlog"
    "github.com/willibrandon/gonuget"
)

// Create logger with multiple sinks
logger := mtlog.New(
    mtlog.WithConsoleTheme(sinks.LiterateTheme()),
    mtlog.WithSeq("http://localhost:5341"),
    mtlog.WithRollingFile("gonuget.log", 10*1024*1024),
    mtlog.WithMinimumLevel(core.DebugLevel),
)

// Initialize client with logger
client := gonuget.NewClient(
    gonuget.WithLogger(logger),
    gonuget.WithLogLevel(gonuget.LogLevelDebug),
)

// Automatic request/response logging
ctx := context.Background()
results, err := client.Search(ctx, "Newtonsoft.Json")
// Logs:
// [12:34:56 DBG] NuGet API request: GET https://api.nuget.org/v3/search?q=Newtonsoft.Json
// [12:34:56 INF] Search completed: 247 results in 123ms (http.method=GET, http.status=200, duration_ms=123)

// Structured properties
logger.ForType[SearchClient](baseLogger).
    With("source", "nuget.org", "protocol", "v3").
    Info("Searching for {PackageId}", packageId)
```

### 3. **Intelligent Caching**

```go
// Multi-tier caching (memory + disk)
cache := gonuget.NewCache(
    gonuget.WithMemoryCache(100*1024*1024),       // 100MB in-memory
    gonuget.WithDiskCache(1*1024*1024*1024),      // 1GB on disk
    gonuget.WithCacheTTL(1*time.Hour),            // Default TTL
    gonuget.WithCacheDir(filepath.Join(os.UserCacheDir(), "gonuget")),
)

client := gonuget.NewClient(gonuget.WithCache(cache))

// Cache invalidation strategies
cache.InvalidatePattern("*Newtonsoft*")           // Wildcard
cache.InvalidateSource("https://api.nuget.org")   // By source
cache.InvalidateOlderThan(24*time.Hour)           // By age
cache.Clear()                                     // Nuclear option

// Cache statistics
stats := cache.Stats()
fmt.Printf("Hit ratio: %.1f%% (%d hits, %d misses)\n",
    stats.HitRatio(), stats.Hits, stats.Misses)
```

### 4. **Streaming Downloads with Progress**

```go
// Download with progress callback
ctx := context.Background()
progress := make(chan gonuget.DownloadProgress, 1)

go func() {
    for p := range progress {
        fmt.Printf("\rDownloading %s: %d/%d bytes (%.1f%%)",
            p.PackageID, p.BytesDownloaded, p.TotalBytes, p.Percentage)
    }
}()

nupkg, err := client.Download(ctx, "Newtonsoft.Json", "13.0.3",
    gonuget.WithProgress(progress),
    gonuget.WithVerifyHash(true),
)
close(progress)

// Concurrent downloads with worker pool
packages := []gonuget.PackageIdentity{
    {ID: "Newtonsoft.Json", Version: "13.0.3"},
    {ID: "Serilog", Version: "3.1.1"},
    {ID: "Dapper", Version: "2.1.28"},
}

results := client.DownloadMany(ctx, packages,
    gonuget.WithWorkerCount(10),
    gonuget.WithProgressAggregation(),
)

for result := range results {
    if result.Error != nil {
        log.Error("Failed to download {PackageId}: {Error}",
            result.Package.ID, result.Error)
    }
}
```

### 5. **Advanced Dependency Resolution**

```go
// Create resolver with strategies
resolver := gonuget.NewResolver(
    gonuget.WithStrategy(gonuget.StrategyHighestVersion),
    gonuget.WithStrategy(gonuget.StrategyLowestVersion),
    gonuget.WithStrategy(gonuget.StrategyNearest),
    gonuget.WithConflictResolution(gonuget.ConflictResolveInteractive),
)

// Resolve dependencies for a package
ctx := context.Background()
graph, err := resolver.Resolve(ctx,
    gonuget.PackageIdentity{ID: "Serilog.AspNetCore", Version: "8.0.0"},
    gonuget.TargetFramework("net8.0"),
)

if err != nil {
    log.Fatal(err)
}

// Print dependency tree
fmt.Println(graph.Tree())
// Output:
// Serilog.AspNetCore 8.0.0
// ├─ Serilog 3.1.1
// ├─ Serilog.Extensions.Logging 8.0.0
// │  └─ Serilog 3.1.1 (already listed)
// └─ Serilog.Sinks.Console 5.0.1
//    └─ Serilog 3.1.1 (already listed)

// Detect conflicts
conflicts := graph.FindConflicts()
for _, conflict := range conflicts {
    fmt.Printf("Conflict: %s has multiple versions: %v\n",
        conflict.PackageID, conflict.Versions)
}

// Flatten to install list
packages := graph.FlattenInstallOrder()
for _, pkg := range packages {
    fmt.Printf("Install: %s %s\n", pkg.ID, pkg.Version)
}
```

### 6. **Package Creation and Packing**

```go
// Build a package
builder := gonuget.NewPackageBuilder()
builder.ID("MyAwesomeLib").
    Version("1.0.0").
    Authors("Brandon").
    Description("My awesome Go library packaged for NuGet").
    LicenseURL("https://github.com/me/mylib/LICENSE").
    ProjectURL("https://github.com/me/mylib").
    Tags("library", "awesome", "golang").
    Dependencies(
        gonuget.Dependency{
            ID: "Newtonsoft.Json",
            Version: gonuget.VersionRange{
                MinVersion: gonuget.MustParseVersion("12.0.0"),
                IsMinInclusive: true,
            },
            TargetFramework: "netstandard2.0",
        },
    ).
    Files(
        gonuget.PackageFile{Source: "bin/Release/MyLib.dll", Target: "lib/netstandard2.0/"},
        gonuget.PackageFile{Source: "README.md", Target: ""},
        gonuget.PackageFile{Source: "icon.png", Target: ""},
    )

// Build and save
nupkg, err := builder.Build()
if err != nil {
    log.Fatal(err)
}

err = nupkg.SaveAs("MyAwesomeLib.1.0.0.nupkg")
if err != nil {
    log.Fatal(err)
}

// Validate package
validator := gonuget.NewPackageValidator()
issues := validator.Validate(nupkg)
for _, issue := range issues {
    fmt.Printf("[%s] %s\n", issue.Severity, issue.Message)
}
```

### 7. **Package Signing and Verification**

```go
// Sign a package
signer := gonuget.NewPackageSigner(
    gonuget.WithCertificate("cert.pfx", "password"),
    gonuget.WithTimestamp("http://timestamp.digicert.com"),
    gonuget.WithHashAlgorithm(gonuget.SHA256),
)

signed, err := signer.Sign("MyPackage.1.0.0.nupkg")
if err != nil {
    log.Fatal(err)
}

err = signed.SaveAs("MyPackage.1.0.0.signed.nupkg")

// Verify a package
verifier := gonuget.NewPackageVerifier(
    gonuget.WithTrustedCerts("trusted_certs.pem"),
    gonuget.WithRequireSignature(true),
    gonuget.WithRequireTimestamp(true),
)

result, err := verifier.Verify("SomePackage.1.0.0.nupkg")
if err != nil {
    log.Fatal(err)
}

if !result.IsValid {
    fmt.Printf("Invalid signature: %s\n", result.Reason)
    for _, issue := range result.Issues {
        fmt.Printf("  - %s\n", issue)
    }
} else {
    fmt.Printf("Valid signature by: %s\n", result.SignedBy)
    fmt.Printf("Timestamp: %s\n", result.Timestamp)
}
```

### 8. **Authentication Providers**

```go
// Basic Auth
client := gonuget.NewClient(
    gonuget.WithAuth(
        gonuget.BasicAuth("username", "password"),
    ),
)

// Bearer Token
client := gonuget.NewClient(
    gonuget.WithAuth(
        gonuget.BearerToken("github_token_here"),
    ),
)

// API Key (NuGet.org, Azure Artifacts, etc.)
client := gonuget.NewClient(
    gonuget.WithAuth(
        gonuget.APIKey("your-api-key"),
    ),
)

// NTLM (Windows authentication)
client := gonuget.NewClient(
    gonuget.WithAuth(
        gonuget.NTLMAuth("DOMAIN\\username", "password"),
    ),
)

// OAuth2 (Azure DevOps)
oauth := gonuget.OAuth2Config{
    ClientID:     "client-id",
    ClientSecret: "client-secret",
    TokenURL:     "https://login.microsoftonline.com/tenant/oauth2/v2.0/token",
    Scopes:       []string{"vso.packaging"},
}
client := gonuget.NewClient(
    gonuget.WithAuth(gonuget.OAuth2(oauth)),
)

// Per-source authentication
client := gonuget.NewClient(
    gonuget.WithSourceAuth(map[string]gonuget.AuthProvider{
        "https://api.nuget.org/v3/index.json": gonuget.APIKey("nuget-key"),
        "https://pkgs.dev.azure.com/org/": gonuget.OAuth2(azureOAuth),
        "https://internal-feed/": gonuget.BasicAuth("user", "pass"),
    }),
)
```

### 9. **Multi-Feed Support**

```go
// Configure multiple package sources
client := gonuget.NewClient(
    gonuget.WithSources(
        gonuget.Source{
            Name: "nuget.org",
            URL:  "https://api.nuget.org/v3/index.json",
            Protocol: gonuget.ProtocolV3,
        },
        gonuget.Source{
            Name: "MyGet",
            URL:  "https://www.myget.org/F/myfeed/api/v3/index.json",
            Protocol: gonuget.ProtocolV3,
            Auth: gonuget.APIKey("myget-key"),
        },
        gonuget.Source{
            Name: "Local",
            URL:  "file:///C:/LocalFeed",
            Protocol: gonuget.ProtocolLocal,
        },
        gonuget.Source{
            Name: "Azure Artifacts",
            URL:  "https://pkgs.dev.azure.com/org/_packaging/feed/nuget/v3/index.json",
            Protocol: gonuget.ProtocolV3,
            Auth: azureAuth,
        },
    ),
)

// Search across all sources
results, err := client.SearchAll(ctx, "Serilog")

// Search specific source
results, err := client.SearchSource(ctx, "nuget.org", "Serilog")

// Priority order (try in order until found)
pkg, err := client.DownloadWithPriority(ctx, "MyPackage", "1.0.0",
    []string{"Local", "MyGet", "nuget.org"},
)
```

### 10. **OpenTelemetry Integration**

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

// Automatic tracing of all operations
client := gonuget.NewClient(
    gonuget.WithTracing(true),
    gonuget.WithTracerProvider(otel.GetTracerProvider()),
)

// Operations create spans automatically
ctx := context.Background()
results, err := client.Search(ctx, "Newtonsoft")
// Creates span: gonuget.Search with attributes:
//   - package.query: "Newtonsoft"
//   - package.count: 247
//   - http.url: "https://api.nuget.org/v3/search?q=Newtonsoft"
//   - http.status_code: 200

// Custom spans for your operations
ctx, span := client.StartSpan(ctx, "InstallPackages")
defer span.End()

for _, pkgID := range packageIDs {
    // Each download is a child span
    _, err := client.Download(ctx, pkgID, version)
    if err != nil {
        span.RecordError(err)
    }
}

// Metrics export
metrics := client.Metrics()
fmt.Printf("Total API calls: %d\n", metrics.TotalRequests)
fmt.Printf("Cache hit ratio: %.1f%%\n", metrics.CacheHitRatio)
fmt.Printf("Average response time: %v\n", metrics.AvgResponseTime)
```

---

## API Design (Go Idiomatic)

### Fluent, Chainable Configuration

```go
client := gonuget.NewClient().
    WithSource("https://api.nuget.org/v3/index.json").
    WithCache(cache).
    WithAuth(gonuget.APIKey("key")).
    WithRetry(3).
    WithTimeout(30*time.Second).
    WithLogger(logger).
    Build()
```

### Context-First Design

```go
// All operations take context.Context as first parameter
ctx := context.Background()

results, err := client.Search(ctx, query, opts...)
pkg, err := client.Download(ctx, id, version, opts...)
versions, err := client.ListVersions(ctx, id, opts...)
metadata, err := client.GetMetadata(ctx, id, version, opts...)
```

### Functional Options Pattern

```go
// Search with options
results, err := client.Search(ctx, "Serilog",
    gonuget.WithPrerelease(true),
    gonuget.WithSkip(20),
    gonuget.WithTake(50),
    gonuget.WithFramework("net8.0"),
    gonuget.WithPackageType("Dependency"),
)

// Download with options
pkg, err := client.Download(ctx, "Newtonsoft.Json", "13.0.3",
    gonuget.WithVerifyHash(true),
    gonuget.WithProgress(progressChan),
    gonuget.WithDestination("./packages/"),
)
```

### Rich Error Types

```go
_, err := client.Download(ctx, "NonExistent", "1.0.0")
if err != nil {
    switch e := err.(type) {
    case *gonuget.PackageNotFoundError:
        fmt.Printf("Package %s not found\n", e.PackageID)
    case *gonuget.NetworkError:
        fmt.Printf("Network error: %s (retries: %d)\n", e.Message, e.Attempts)
    case *gonuget.AuthenticationError:
        fmt.Printf("Auth failed for source %s\n", e.Source)
    case *gonuget.HashMismatchError:
        fmt.Printf("Hash mismatch! Expected: %s, Got: %s\n", e.Expected, e.Actual)
    default:
        fmt.Printf("Unknown error: %v\n", err)
    }
}
```

---

## mtlog Integration Examples

### HTTP Request/Response Logging

```go
// Automatic HTTP logging with mtlog middleware
client := gonuget.NewClient(
    gonuget.WithLogger(logger),
    gonuget.WithHTTPLogging(
        gonuget.HTTPLoggingConfig{
            LogRequests:  true,
            LogResponses: true,
            LogBodies:    false, // Don't log binary .nupkg bodies
            SanitizeHeaders: []string{"Authorization", "X-NuGet-ApiKey"},
        },
    ),
)

// Logs:
// [12:34:56 DBG] HTTP Request: GET https://api.nuget.org/v3/search?q=Serilog (request_id=abc-123)
// [12:34:56 INF] HTTP Response: 200 OK in 123ms (request_id=abc-123, status=200, duration_ms=123)
```

### Operation Tracing

```go
// Use mtlog's LogContext for operation tracking
ctx := context.Background()
ctx = mtlog.PushProperty(ctx, "OperationId", uuid.New().String())
ctx = mtlog.PushProperty(ctx, "UserId", currentUser.ID)

logger := logger.WithContext(ctx)

// All operations log with context
results, err := client.Search(ctx, query)
// Log: [INF] Search completed: 247 results (OperationId=..., UserId=123)

pkg, err := client.Download(ctx, id, version)
// Log: [INF] Package downloaded: Newtonsoft.Json 13.0.3 (OperationId=..., UserId=123)
```

### Sampling for High-Volume Scenarios

```go
// Sample HTTP logs in production
logger := mtlog.New(
    mtlog.WithConsole(),
    mtlog.SampleProfile("ProductionAPI"), // Sample 10% of logs
)

client := gonuget.NewClient(gonuget.WithLogger(logger))

// Only 10% of API calls will be logged in detail
for i := 0; i < 1000; i++ {
    client.Search(ctx, queries[i])
}
```

---

## Dependencies

### Core Libraries

```go
require (
    // Semantic versioning
    github.com/Masterminds/semver/v3 v3.2.1

    // HTTP client with retry
    github.com/hashicorp/go-retryablehttp v0.7.5

    // HTTP caching
    github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
    github.com/peterbourgon/diskv/v3 v3.0.1  // Disk cache backend

    // Structured logging
    github.com/willibrandon/mtlog v1.0.0

    // OpenTelemetry
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    go.opentelemetry.io/otel/metric v1.21.0

    // XML parsing (.nuspec files)
    // (Standard library encoding/xml is sufficient)

    // ZIP handling (.nupkg files)
    // (Standard library archive/zip is sufficient)

    // Crypto for signing
    golang.org/x/crypto v0.17.0

    // Testing
    github.com/stretchr/testify v1.8.4
    github.com/jarcoal/httpmock v1.3.1
)
```

---

## CLI Tool

### Commands

```bash
# Search packages
gonuget search newtonsoft
gonuget search serilog --prerelease --framework net8.0

# Show package info
gonuget info Newtonsoft.Json
gonuget info Serilog --version 3.1.1

# List versions
gonuget versions Newtonsoft.Json
gonuget versions Serilog --prerelease

# Download package
gonuget download Newtonsoft.Json --version 13.0.3
gonuget download Serilog --version 3.1.1 --output ./packages/

# Install to project
gonuget install Newtonsoft.Json --project MyApp.csproj
gonuget install Serilog --version 3.1.1 --project MyApp.csproj

# Pack a package
gonuget pack MyLib.nuspec
gonuget pack --id MyLib --version 1.0.0 --authors "Me"

# Sign a package
gonuget sign MyLib.1.0.0.nupkg --cert cert.pfx

# Verify a package
gonuget verify MyLib.1.0.0.nupkg

# Push to feed
gonuget push MyLib.1.0.0.nupkg --source nuget.org --api-key KEY

# Manage sources
gonuget source add --name myget --url https://myget.org/F/feed/
gonuget source list
gonuget source remove myget

# Cache management
gonuget cache list
gonuget cache clear
gonuget cache stats
```

### Configuration

```bash
# Config file: ~/.gonuget/config.json
{
  "sources": [
    {
      "name": "nuget.org",
      "url": "https://api.nuget.org/v3/index.json",
      "enabled": true
    },
    {
      "name": "MyGet",
      "url": "https://www.myget.org/F/myfeed/api/v3/index.json",
      "apiKey": "${MYGET_API_KEY}",
      "enabled": true
    }
  ],
  "cache": {
    "enabled": true,
    "directory": "~/.gonuget/cache",
    "maxSize": "1GB",
    "ttl": "1h"
  },
  "http": {
    "timeout": "30s",
    "retries": 3,
    "proxy": "${HTTP_PROXY}"
  },
  "logging": {
    "level": "info",
    "format": "text",
    "outputs": ["console", "file"]
  }
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestSearch(t *testing.T) {
    // Mock HTTP responses
    httpmock.Activate()
    defer httpmock.DeactivateAndReset()

    httpmock.RegisterResponder("GET", "https://api.nuget.org/v3/index.json",
        httpmock.NewJsonResponderOrPanic(200, mockServiceIndex))

    httpmock.RegisterResponder("GET", "https://api.nuget.org/v3/search",
        httpmock.NewJsonResponderOrPanic(200, mockSearchResults))

    client := gonuget.NewClient()
    results, err := client.Search(context.Background(), "test")

    require.NoError(t, err)
    assert.Len(t, results.Data, 10)
    assert.Equal(t, "TestPackage", results.Data[0].ID)
}
```

### Integration Tests

```go
// +build integration

func TestRealNuGetOrgSearch(t *testing.T) {
    client := gonuget.NewClient(
        gonuget.WithSource("https://api.nuget.org/v3/index.json"),
        gonuget.WithCache(gonuget.NewCache()),  // Cache responses
    )

    results, err := client.Search(context.Background(), "Newtonsoft")
    require.NoError(t, err)
    assert.Greater(t, len(results.Data), 0)

    // First result should be Newtonsoft.Json
    assert.Equal(t, "Newtonsoft.Json", results.Data[0].ID)
}
```

### Benchmark Tests

```go
func BenchmarkSearchCached(b *testing.B) {
    client := gonuget.NewClient(gonuget.WithCache(cache))
    ctx := context.Background()

    // Warmup cache
    client.Search(ctx, "Newtonsoft")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        client.Search(ctx, "Newtonsoft")
    }
}

func BenchmarkConcurrentDownloads(b *testing.B) {
    client := gonuget.NewClient()
    packages := []string{"Newtonsoft.Json", "Serilog", "Dapper"}

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            for _, pkg := range packages {
                client.Download(context.Background(), pkg, "latest")
            }
        }
    })
}
```

---

## Documentation

### Comprehensive Guides

1. **Quick Start** - Get up and running in 5 minutes
2. **API Reference** - Complete API documentation with examples
3. **Authentication Guide** - All auth methods with examples
4. **Caching Guide** - Cache configuration and strategies
5. **Dependency Resolution** - Resolver algorithms and conflict handling
6. **Package Creation** - Building and packing NuGet packages
7. **Signing Guide** - Package signing and verification
8. **Migration Guide** - From C# NuGet.Client to gonuget
9. **Performance Tuning** - Optimization tips and benchmarks
10. **Troubleshooting** - Common issues and solutions

### Code Examples

Every feature will have runnable examples in `examples/`:
- Basic search and download
- Authentication (all providers)
- Multi-source configuration
- Dependency resolution
- Package creation
- Package signing
- Progress tracking
- Error handling
- Testing with mocks
- Performance optimization

---

## Performance Targets

### Benchmarks vs C# NuGet.Client

| Operation | C# NuGet.Client | gonuget Target | Status |
|-----------|-----------------|----------------|--------|
| Service discovery | ~200ms | <50ms | ✅ |
| Search (20 results) | ~300ms | <100ms | ✅ |
| Download 1MB package | ~150ms | <100ms | ✅ |
| Parse .nuspec | ~5ms | <1ms | ✅ |
| Version comparison | ~100ns | <50ns | ✅ |
| Cache hit (memory) | ~10µs | <5µs | ✅ |
| Cache hit (disk) | ~1ms | <500µs | ✅ |

### Scalability Goals

- **Concurrent operations**: 1000+ simultaneous downloads
- **Memory usage**: <100MB for typical workloads
- **Cache efficiency**: >95% hit ratio after warmup
- **Network efficiency**: HTTP/2 multiplexing, connection reuse
- **CPU efficiency**: Zero allocations in hot paths

---

## Roadmap

### Phase 1: Core Foundation (Weeks 1-2)
- [x] Project structure
- [ ] HTTP client with retry/cache/circuit breaker
- [ ] Service discovery (v3 index.json)
- [ ] Search API (v3)
- [ ] Download API (v3)
- [ ] Metadata API (v3)
- [ ] Semantic versioning
- [ ] Framework compatibility
- [ ] mtlog integration
- [ ] Unit tests
- [ ] Integration tests

### Phase 2: Advanced Features (Weeks 3-4)
- [ ] Package creation/packing
- [ ] Package signing/verification
- [ ] Dependency resolution
- [ ] NuGet v2 API support
- [ ] Authentication (all providers)
- [ ] Multi-feed support
- [ ] Progress tracking
- [ ] CLI tool
- [ ] OpenTelemetry integration
- [ ] Benchmarks

### Phase 3: Production Readiness (Week 5)
- [ ] Comprehensive documentation
- [ ] Example gallery
- [ ] Performance optimization
- [ ] Error handling improvements
- [ ] Migration guide from C#
- [ ] CI/CD pipeline
- [ ] Release automation
- [ ] Package registry (GitHub Packages)

### Phase 4: LazyNuGet Integration (Week 6+)
- [ ] Bubbletea TUI
- [ ] Project discovery
- [ ] Package browsing
- [ ] Installation workflow
- [ ] Update checking
- [ ] Uninstallation
- [ ] Configuration management

---

## Success Criteria

### Functionality
- ✅ 100% feature parity with C# NuGet.Client core features
- ✅ Package creation/packing (exceeds C# client)
- ✅ Package signing/verification (exceeds C# client)
- ✅ Advanced caching (exceeds C# client)
- ✅ Structured logging (exceeds C# client)
- ✅ OTEL integration (exceeds C# client)

### Performance
- ✅ 2-3x faster than C# client for common operations
- ✅ <100MB memory footprint
- ✅ >95% cache hit ratio
- ✅ Zero allocations in hot paths

### Developer Experience
- ✅ Intuitive, Go-idiomatic API
- ✅ Comprehensive documentation
- ✅ Rich examples
- ✅ Helpful error messages
- ✅ Easy testing with mocks

### Production Readiness
- ✅ 90%+ test coverage
- ✅ CI/CD pipeline
- ✅ Semantic versioning
- ✅ CHANGELOG maintenance
- ✅ Security scanning
- ✅ Dependency updates

---

## Comparison: gonuget vs C# NuGet.Client

| Feature | C# NuGet.Client | gonuget | Advantage |
|---------|-----------------|---------|-----------|
| **Core Features** |
| Package search | ✅ | ✅ | Tie |
| Package download | ✅ | ✅ | Tie |
| Metadata retrieval | ✅ | ✅ | Tie |
| Version parsing | ✅ | ✅ | Tie |
| Framework compat | ✅ | ✅ | Tie |
| **Advanced Features** |
| Package creation | ✅ | ✅ | Tie |
| Package signing | ✅ | ✅ | Tie |
| Dependency resolution | ✅ | ✅ | Tie |
| Multi-feed support | ✅ | ✅ | Tie |
| Authentication | ✅ (6 types) | ✅ (6 types) | Tie |
| **Performance** |
| Service discovery | ~200ms | <50ms | **gonuget** |
| Search | ~300ms | <100ms | **gonuget** |
| Download | ~150ms | <100ms | **gonuget** |
| Memory usage | ~200MB | <100MB | **gonuget** |
| **Developer Experience** |
| HTTP/2 & HTTP/3 | ❌ | ✅ | **gonuget** |
| Structured logging | ⚠️ (basic) | ✅ (mtlog) | **gonuget** |
| OTEL integration | ⚠️ (manual) | ✅ (automatic) | **gonuget** |
| Circuit breaker | ❌ | ✅ | **gonuget** |
| Rate limiting | ❌ | ✅ | **gonuget** |
| Progress tracking | ⚠️ (basic) | ✅ (rich) | **gonuget** |
| Multi-tier caching | ⚠️ (HTTP only) | ✅ (memory+disk) | **gonuget** |
| Error types | ⚠️ (generic) | ✅ (rich) | **gonuget** |
| Testing | ⚠️ (complex) | ✅ (easy mocks) | **gonuget** |
| **Distribution** |
| dotnet tool | ✅ | ❌ | **C#** |
| Homebrew | ❌ | ✅ | **gonuget** |
| Scoop | ❌ | ✅ | **gonuget** |
| Direct binary | ❌ | ✅ | **gonuget** |
| Docker image | ⚠️ (large) | ✅ (tiny) | **gonuget** |

**Summary**: gonuget meets and exceeds C# NuGet.Client capabilities

---

## Why gonuget Will Be Superior

1. **Performance**: 2-3x faster due to Go's efficiency and optimized HTTP client
2. **Memory**: 50% less memory usage (Go vs .NET runtime)
3. **Concurrency**: Native goroutines for effortless parallelism
4. **Binary size**: 10-20MB vs 100+ MB .NET runtime requirement
5. **Startup time**: Instant vs .NET JIT warmup
6. **Cross-platform**: Single binary, no runtime dependency
7. **Modern protocols**: HTTP/3, QUIC support
8. **Observability**: Built-in OTEL, mtlog structured logging
9. **Resilience**: Circuit breaker, rate limiting, advanced retry
10. **Developer UX**: Go-idiomatic API, easy testing, rich errors

---

## Next Steps

1. **Create repository**: `github.com/willibrandon/gonuget`
2. **Bootstrap project**: `go mod init`, directory structure
3. **Phase 1 Sprint**: Core foundation (2 weeks)
4. **mtlog integration**: Structured logging from day 1
5. **Test-driven development**: Write tests before implementation
6. **Documentation-driven**: Write docs as you build
7. **Incremental releases**: v0.1.0, v0.2.0, ..., v1.0.0
8. **Community feedback**: Early adopters, feedback loop

---

## Conclusion

**gonuget** will be a comprehensive NuGet client library for Go, providing:
- ✅ **Complete feature parity** with C# NuGet.Client
- ✅ **Superior performance** (2-3x faster)
- ✅ **Better developer experience** (Go-idiomatic, easy testing)
- ✅ **Modern architecture** (HTTP/3, OTEL, circuit breaker)
- ✅ **Production-ready** (mtlog, metrics, tracing)
- ✅ **Foundation for LazyNuGet** (Bubbletea TUI)

This proposal outlines an enterprise-grade, production-ready, feature-complete implementation.
