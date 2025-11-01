# gonuget

NuGet client library and CLI for Go with protocol parity to the official .NET NuGet.Client.

## Status

**Library**: 77% complete (68/88 milestones)

**CLI**: Command restructure complete (noun-first hierarchy with `package`, `source`, `config`, `restore`, and `version` namespaces)

Interop tests passing against NuGet.Client for feature parity validation.

## Features

**Core Operations**
- Package search, metadata retrieval, version resolution
- Package download with content verification
- Full dependency graph resolution with conflict detection
- Transitive dependency resolution with parallel processing

**Version System**
- SemVer 2.0 parsing and comparison (<20ns/op)
- Legacy 4-part version support (Major.Minor.Build.Revision)
- Version range evaluation with floating version support

**Framework Support**
- Target Framework Moniker (TFM) parsing and compatibility checking
- Framework-specific dependency selection
- Portable Class Library (PCL) profile mapping

**Protocol Implementation**
- NuGet V3 (service index, registration, search, download, autocomplete)
- NuGet V2 (OData feeds with XML/Atom parsing)
- Automatic protocol detection
- Multi-source repository management

**Package Operations**
- Package reading from .nupkg files (ZIP + nuspec parsing)
- Package creation with OPC (Open Packaging Conventions) compliance
- PKCS#7 signature creation, verification, and RFC 3161 timestamping
- Asset selection with Runtime Identifier (RID) resolution

**Infrastructure**
- Multi-tier caching (LRU memory + disk persistence with ETag validation)
- HTTP/2 and HTTP/3 support with automatic fallback
- Retry logic with exponential backoff and Retry-After header support
- Circuit breaker pattern for fault tolerance
- Per-source rate limiting with token bucket algorithm

**Observability**
- OpenTelemetry tracing with distributed context propagation
- Prometheus metrics for operation monitoring
- Structured logging via mtlog integration

**Authentication**
- API key authentication (X-NuGet-ApiKey header)
- Bearer token authentication
- HTTP basic authentication

## Installation

### Library

```bash
go get github.com/willibrandon/gonuget
```

### CLI

```bash
git clone https://github.com/willibrandon/gonuget
cd gonuget
make build
./gonuget --version
```

See [cmd/gonuget/README.md](cmd/gonuget/README.md) for CLI documentation.

## CLI Quick Start

```bash
# Add a package source
gonuget source add https://api.nuget.org/v3/index.json --name "NuGet.org"

# List configured sources
gonuget source list

# Add a package to a project
gonuget package add Newtonsoft.Json --project MyProject.csproj --version 13.0.3

# Search for packages
gonuget package search Newtonsoft --format json

# Get configuration value
gonuget config get repositoryPath

# Enable shell completion (bash, zsh, powershell)
gonuget completion bash > /etc/bash_completion.d/gonuget
```

**Command Structure**: Noun-first hierarchy matching `dotnet` CLI (e.g., `gonuget package add`, `gonuget source list`)

**Performance**: 15-17x faster than dotnet nuget for CLI operations

## Library Usage

### Basic Package Search

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/willibrandon/gonuget/core"
    "github.com/willibrandon/gonuget/http"
)

func main() {
    // Create HTTP client
    httpClient := http.NewClient(nil)

    // Create repository manager
    repoManager := core.NewRepositoryManager()

    // Add NuGet.org as source
    repo := core.NewSourceRepository(core.RepositoryConfig{
        Name:       "nuget.org",
        SourceURL:  "https://api.nuget.org/v3/index.json",
        HTTPClient: httpClient,
    })
    repoManager.AddRepository(repo)

    // Create client
    client := core.NewClient(core.ClientConfig{
        RepositoryManager: repoManager,
    })

    // Search for packages
    ctx := context.Background()
    results, err := client.SearchPackages(ctx, "newtonsoft", core.SearchOptions{
        Take:              10,
        IncludePrerelease: false,
    })
    if err != nil {
        log.Fatal(err)
    }

    for repoName, pkgs := range results {
        fmt.Printf("Repository: %s\n", repoName)
        for _, pkg := range pkgs {
            fmt.Printf("  %s %s\n", pkg.ID, pkg.Version)
        }
    }
}
```

### Version Parsing and Comparison

```go
import "github.com/willibrandon/gonuget/version"

v1, _ := version.Parse("1.2.3-beta.1")
v2, _ := version.Parse("1.2.3")

if v1.Compare(v2) < 0 {
    fmt.Println("v1 is less than v2")
}

// Version range evaluation
vr, _ := version.ParseVersionRange("[1.0.0,2.0.0)")
if vr.IsSatisfiedBy(v2) {
    fmt.Println("v2 satisfies range")
}
```

### Dependency Resolution

```go
import (
    "github.com/willibrandon/gonuget/core/resolver"
    "github.com/willibrandon/gonuget/protocol/v3"
)

// Create metadata client
httpClient := http.NewClient(nil)
serviceIndexClient := v3.NewServiceIndexClient(httpClient)
metadataClient := v3.NewMetadataClient(httpClient, serviceIndexClient)

// Create resolver
res := resolver.NewResolver(
    metadataClient,
    []string{"https://api.nuget.org/v3/index.json"},
    "net8.0",
)

// Resolve dependencies
result, err := res.Resolve(ctx, "Newtonsoft.Json", "[13.0.1]")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Resolved %d packages\n", len(result.Packages))
for _, pkg := range result.Packages {
    fmt.Printf("  %s %s\n", pkg.ID, pkg.Version)
}
```

### Package Reading

```go
import "github.com/willibrandon/gonuget/packaging"

reader, err := packaging.OpenPackageReader("package.nupkg")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

nuspec := reader.GetNuspec()
fmt.Printf("Package: %s %s\n", nuspec.Metadata.ID, nuspec.Metadata.Version)

// List files
files := reader.GetFiles()
for _, f := range files {
    fmt.Printf("  %s (%d bytes)\n", f, reader.GetEntry(f).UncompressedSize64)
}
```

### Package Creation

```go
import (
    "github.com/willibrandon/gonuget/packaging"
    "github.com/willibrandon/gonuget/version"
)

builder := packaging.NewPackageBuilder()
ver, _ := version.Parse("1.0.0")

builder.
    SetID("MyPackage").
    SetVersion(ver).
    SetDescription("My package description").
    SetAuthors("Author Name").
    AddFile("lib/net8.0/MyLibrary.dll", "MyLibrary.dll")

if err := builder.Save("MyPackage.1.0.0.nupkg"); err != nil {
    log.Fatal(err)
}
```

### Framework Compatibility

```go
import "github.com/willibrandon/gonuget/frameworks"

net80, _ := frameworks.Parse("net8.0")
net70, _ := frameworks.Parse("net7.0")
netstandard20, _ := frameworks.Parse("netstandard2.0")

// Check compatibility
if net80.IsCompatibleWith(netstandard20) {
    fmt.Println("net8.0 apps can use netstandard2.0 packages")
}

// Get portable framework name
fmt.Println(net80.GetShortFolderName()) // "net8.0"
```

### Caching

```go
import (
    "github.com/willibrandon/gonuget/cache"
    "time"
)

// Create memory cache
memCache := cache.NewMemoryCache(100 * 1024 * 1024) // 100MB

// Create disk cache
diskCache, _ := cache.NewDiskCache("/tmp/nuget-cache", 1*1024*1024*1024) // 1GB

// Create multi-tier cache
mtCache := cache.NewMultiTierCache(memCache, diskCache)

// Use with client
repo := core.NewSourceRepository(core.RepositoryConfig{
    Name:       "nuget.org",
    SourceURL:  "https://api.nuget.org/v3/index.json",
    HTTPClient: httpClient,
    Cache:      mtCache,
})
```

## Testing

### Library Tests

```bash
# All tests
make test

# Go unit tests only (skip integration)
make test-go-unit

# Interop tests (validate parity with NuGet.Client)
make test-interop

# Specific package
go test ./version
go test ./core/resolver
```

The project includes C# interop tests that validate exact behavioral parity with NuGet.Client by running identical operations in both implementations and comparing results.

### CLI Tests

```bash
# Run CLI tests
go test ./cmd/gonuget/... -v

# Run CLI benchmarks
go test -tags=benchmark -bench=. ./cmd/gonuget
```

## Performance

### Library

- Version comparison: <20ns/op (zero allocations)
- Framework compatibility checks: optimized for hot path operations
- HTTP/2 connection pooling and multiplexing
- HTTP/3 support for reduced latency
- Multi-tier caching with efficient TTL and ETag validation

### CLI

- **15-17x faster** than dotnet nuget for common operations
- **30-35% less memory** per command invocation
- Startup time: ~6-7ms (vs ~100-120ms for dotnet nuget)
- Zero runtime overhead (native binary)

See [cmd/gonuget/benchmarks/README.md](cmd/gonuget/benchmarks/README.md) for detailed benchmarks.

## Requirements

- Go 1.25.2 or later
- For interop tests: .NET 9.0 SDK

## Development

```bash
# Build
make build

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

## License

MIT License - See [LICENSE](LICENSE) file
