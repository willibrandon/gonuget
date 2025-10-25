# NuGet.Client Implementation Review for Go Port

## Repository Overview

**Location**: https://github.com/NuGet/NuGet.Client
**License**: Apache 2.0
**Language**: C# (.NET 9.0)
**Size**: ~1,470 C# files in Core components
**Primary Use**: Official NuGet client libraries (used by Visual Studio, dotnet CLI, nuget.exe)

---

## Architecture

### Core Components (src/NuGet.Core/)

The NuGet client is split into modular, focused libraries:

#### 1. **NuGet.Protocol** (~40 C# files)
**Purpose**: HTTP client for NuGet v2 and v3 APIs

**Key Responsibilities**:
- Service discovery (index.json parsing)
- Package search
- Package download
- Metadata retrieval
- Authentication/proxy handling
- HTTP caching with retry logic

**Critical Files**:
- `ServiceIndexResourceV3.cs` - Service endpoint discovery
- `PackageSearchResourceV3.cs` - Search API client
- `RemoteV3FindPackageByIdResource.cs` - Download API client
- `ServiceTypes.cs` - API version constants

#### 2. **NuGet.Versioning** (~4,600 LOC)
**Purpose**: Semantic versioning parser and comparer

**Key Features**:
- Parses SemVer 2.0 versions (e.g., "1.2.3-beta+build")
- Version range parsing (e.g., "[1.0, 2.0)")
- Version comparison with different comparison modes
- Float ranges (e.g., "1.0.*")

**Critical Classes**:
- `NuGetVersion` - Main version type
- `VersionRange` - Dependency version constraints
- `VersionComparer` - Multiple comparison strategies

#### 3. **NuGet.Frameworks** (~5,700 LOC)
**Purpose**: Target framework parsing and compatibility

**Key Features**:
- Parses TFMs (e.g., "net8.0", "netstandard2.1")
- Framework compatibility checking
- Framework precedence/fallback chains
- Maps short names to full names

#### 4. **NuGet.Packaging**
**Purpose**: .nupkg file reading/writing

**Key Features**:
- Reads .nupkg (ZIP format)
- Parses .nuspec (XML manifest)
- Extracts metadata and dependencies
- Package creation/validation

#### 5. **NuGet.Configuration**
**Purpose**: NuGet.Config file parsing

**Key Features**:
- Package source management
- Authentication/credentials
- Settings hierarchy

---

## NuGet v3 API Specification

### Service Index (Entry Point)

**URL**: `https://api.nuget.org/v3/index.json`

**Structure**:
```json
{
  "version": "3.0.0",
  "resources": [
    {
      "@id": "https://api.nuget.org/v3/registration5-gz-semver2/",
      "@type": "RegistrationsBaseUrl/Versioned",
      "clientVersion": "4.3.0-alpha"
    },
    {
      "@id": "https://azuresearch-usnc.nuget.org/query",
      "@type": "SearchQueryService",
      "comment": "Query endpoint of NuGet Search service (primary)"
    },
    {
      "@id": "https://api.nuget.org/v3-flatcontainer/",
      "@type": "PackageBaseAddress/3.0.0"
    }
  ]
}
```

**Implementation Pattern** (ServiceIndexResourceV3.cs):
1. Fetch index.json
2. Parse "resources" array
3. Build lookup table by @type
4. Select best versioned endpoint for client version

**Key Resource Types**:
- `SearchQueryService` - Package search
- `RegistrationsBaseUrl` - Package metadata
- `PackageBaseAddress` - Package download
- `PackagePublish` - Push packages
- `SearchAutocompleteService` - Autocomplete API

### Search API

**Endpoint Type**: `SearchQueryService`

**Query Pattern**:
```
GET {SearchQueryService}?q={searchTerm}&skip={skip}&take={take}&prerelease={bool}&semVerLevel=2.0.0
```

**Example**:
```
GET https://azuresearch-usnc.nuget.org/query?q=newtonsoft&skip=0&take=20&prerelease=true&semVerLevel=2.0.0
```

**Response Structure** (from V3SearchResults.cs):
```json
{
  "totalHits": 1234,
  "data": [
    {
      "id": "Newtonsoft.Json",
      "version": "13.0.3",
      "description": "Json.NET is a popular high-performance JSON framework for .NET",
      "versions": [
        { "version": "13.0.3", "downloads": 123456789 },
        { "version": "13.0.2", "downloads": 98765432 }
      ],
      "authors": "James Newton-King",
      "iconUrl": "https://...",
      "licenseUrl": "https://...",
      "projectUrl": "https://...",
      "tags": ["json", "serialization"],
      "totalDownloads": 999999999
    }
  ]
}
```

**Key Fields** (from PackageSearchMetadata.cs):
- `id` (PackageId) - Package identifier
- `version` - Latest version
- `description` - Package description
- `authors` - Can be string or array
- `owners` - Can be string or array
- `downloadCount` / `totalDownloads` - Download count
- `iconUrl`, `licenseUrl`, `projectUrl` - URLs
- `tags` - Array of tags
- `versions` - Array of version objects with download counts
- `dependencyGroups` - Dependencies by target framework

**Optional Query Parameters**:
- `includeDelisted=true` - Include delisted packages
- `supportedFramework=net8.0` - Filter by framework
- `packageTypeFilter=Dependency` - Filter by package type

### Package Download API

**Endpoint Type**: `PackageBaseAddress`

**Download URL Pattern**:
```
GET {PackageBaseAddress}/{id-lowercase}/{version-lowercase}/{id-lowercase}.{version-lowercase}.nupkg
```

**Example**:
```
GET https://api.nuget.org/v3-flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg
```

**List Versions URL**:
```
GET {PackageBaseAddress}/{id-lowercase}/index.json
```

**Response**:
```json
{
  "versions": ["1.0.0", "1.0.1", "2.0.0", "13.0.3"]
}
```

### Package Metadata API (Registration)

**Endpoint Type**: `RegistrationsBaseUrl`

**URL Pattern**:
```
GET {RegistrationsBaseUrl}/{id-lowercase}/index.json
```

**Response**: Full package metadata including all versions, dependencies, deprecation info

---

## Key Implementation Patterns & Tricks

### 1. Service Discovery Pattern

**C# Implementation** (ServiceIndexResourceV3.cs):
```csharp
// Fetch index once, cache it
JObject index = await httpClient.GetAsync("https://api.nuget.org/v3/index.json");

// Build lookup by @type
Dictionary<string, List<ServiceIndexEntry>> _index;
foreach (var resource in index["resources"])
{
    var uri = resource["@id"];
    var types = resource["@type"]; // Can be string or array!
    var clientVersion = resource["clientVersion"]; // Optional

    // Add to lookup
    _index[type] = entries;
}

// Get best endpoint for a service type
var searchEndpoints = _index.GetServiceEntryUris("SearchQueryService");
```

**Go Translation Tips**:
```go
type ServiceIndex struct {
    Version   string            `json:"version"`
    Resources []ServiceResource `json:"resources"`
}

type ServiceResource struct {
    ID            string   `json:"@id"`
    Type          JsonType `json:"@type"` // Can be string or []string
    ClientVersion string   `json:"clientVersion,omitempty"`
}

// JsonType handles string or array
type JsonType []string

func (jt *JsonType) UnmarshalJSON(data []byte) error {
    // Try array first
    var arr []string
    if err := json.Unmarshal(data, &arr); err == nil {
        *jt = arr
        return nil
    }
    // Fall back to string
    var str string
    if err := json.Unmarshal(data, &str); err != nil {
        return err
    }
    *jt = []string{str}
    return nil
}
```

### 2. Version Parsing (Critical for Go)

**C# Pattern** (NuGetVersion.cs):
```csharp
// Format: Major.Minor.Patch[-Prerelease][+BuildMetadata]
// Examples: "1.0.0", "2.1.3-beta", "1.0.0+sha.abc123"

public class NuGetVersion {
    public int Major;
    public int Minor;
    public int Patch;
    public int Revision; // Optional 4th component
    public string ReleaseLabel; // "beta", "rc.1"
    public string Metadata; // Build metadata after +
}

// Parse logic handles:
// - Leading zeros (1.01.1 = 1.1.1)
// - Normalization (1.0 = 1.0.0.0)
// - Prerelease labels (case-insensitive)
// - Metadata (ignored for comparison)
```

**Go Implementation Tip**:
Use `github.com/Masterminds/semver` package - it already implements SemVer 2.0 correctly:
```go
import "github.com/Masterminds/semver/v3"

version, err := semver.NewVersion("1.2.3-beta+build")
if err != nil {
    return err
}

// Comparison works out of box
if version.LessThan(other) { ... }
```

### 3. HTTP Retry & Caching

**C# Pattern** (EnhancedHttpRetryHelper.cs, HttpSource):
```csharp
// Retry logic:
// - 3 retries with exponential backoff
// - Retry on: 408, 429, 500, 502, 503, 504
// - Don't retry on: 401, 403, 404
// - Add jitter to backoff

// Caching:
// - Cache GET requests based on URL
// - Respect Cache-Control headers
// - Store in %LOCALAPPDATA%/NuGet/v3-cache on Windows
```

**Go Translation**:
```go
// Use github.com/hashicorp/go-retryablehttp
import "github.com/hashicorp/go-retryablehttp"

client := retryablehttp.NewClient()
client.RetryMax = 3
client.RetryWaitMin = time.Second
client.RetryWaitMax = 30 * time.Second

// For caching, use github.com/gregjones/httpcache
import (
    "github.com/gregjones/httpcache"
    "github.com/gregjones/httpcache/diskcache"
)

cache := diskcache.New(filepath.Join(os.UserCacheDir(), "lazynuget", "http"))
transport := httpcache.NewTransport(cache)
client.HTTPClient.Transport = transport
```

### 4. Search Query Building

**C# Implementation** (PackageSearchResourceV3.cs:118-151):
```csharp
var queryUrl = new UriBuilder(searchEndpoint);
var queryString =
    "q=" + searchTerm +
    "&skip=" + skip +
    "&take=" + take +
    "&prerelease=" + includePrerelease.ToString().ToLowerInvariant();

if (includeDelisted) {
    queryString += "&includeDelisted=true";
}

if (supportedFrameworks != null && supportedFrameworks.Any()) {
    var frameworks = string.Join("&",
        supportedFrameworks.Select(fx =>
            "supportedFramework=" + fx.ToString()));
    queryString += "&" + frameworks;
}

queryString += "&semVerLevel=2.0.0";
queryUrl.Query = queryString;
```

**Go Translation**:
```go
import "net/url"

func buildSearchURL(baseURL, searchTerm string, opts SearchOptions) string {
    u, _ := url.Parse(baseURL)
    q := u.Query()

    q.Set("q", searchTerm)
    q.Set("skip", strconv.Itoa(opts.Skip))
    q.Set("take", strconv.Itoa(opts.Take))
    q.Set("prerelease", strconv.FormatBool(opts.IncludePrerelease))
    q.Set("semVerLevel", "2.0.0")

    if opts.IncludeDelisted {
        q.Set("includeDelisted", "true")
    }

    for _, fw := range opts.SupportedFrameworks {
        q.Add("supportedFramework", fw)
    }

    u.RawQuery = q.Encode()
    return u.String()
}
```

### 5. JSON Parsing with Flexible Types

**Problem**: NuGet JSON has fields that can be string OR array (authors, owners, @type)

**C# Solution** (MetadataStringOrArrayConverter.cs):
```csharp
[JsonConverter(typeof(MetadataStringOrArrayConverter))]
public IReadOnlyList<string> Authors { get; set; }

// Converter handles:
// "authors": "John Doe" -> ["John Doe"]
// "authors": ["John", "Jane"] -> ["John", "Jane"]
```

**Go Solution**:
```go
type FlexibleStringArray []string

func (fsa *FlexibleStringArray) UnmarshalJSON(data []byte) error {
    // Try array
    var arr []string
    if err := json.Unmarshal(data, &arr); err == nil {
        *fsa = arr
        return nil
    }

    // Try string
    var str string
    if err := json.Unmarshal(data, &str); err != nil {
        return err
    }
    *fsa = []string{str}
    return nil
}

type PackageMetadata struct {
    Authors FlexibleStringArray `json:"authors"`
    Owners  FlexibleStringArray `json:"owners"`
}
```

### 6. Package File Download

**C# Pattern** (RemoteV3FindPackageByIdResource.cs, FindPackagesByIdNupkgDownloader):
```csharp
// URL: {PackageBaseAddress}/{id-lower}/{version-lower}/{id-lower}.{version-lower}.nupkg

var id = packageId.ToLowerInvariant();
var version = nugetVersion.ToNormalizedString().ToLowerInvariant();
var downloadUrl = $"{baseAddress}{id}/{version}/{id}.{version}.nupkg";

using var stream = await httpClient.GetStreamAsync(downloadUrl);
// Stream is a .nupkg (ZIP file)
```

**Go Translation**:
```go
import (
    "archive/zip"
    "io"
    "strings"
)

func downloadPackage(baseURL, packageID string, version *semver.Version) (io.ReadCloser, error) {
    id := strings.ToLower(packageID)
    ver := strings.ToLower(version.String())

    url := fmt.Sprintf("%s%s/%s/%s.%s.nupkg", baseURL, id, ver, id, ver)

    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        resp.Body.Close()
        return nil, fmt.Errorf("download failed: %d", resp.StatusCode)
    }

    return resp.Body, nil
}

// .nupkg is a ZIP file
func extractNuspec(nupkgPath string) (*Nuspec, error) {
    r, err := zip.OpenReader(nupkgPath)
    if err != nil {
        return nil, err
    }
    defer r.Close()

    // Find .nuspec file (always at root, ends with .nuspec)
    for _, f := range r.File {
        if strings.HasSuffix(f.Name, ".nuspec") && !strings.Contains(f.Name, "/") {
            rc, err := f.Open()
            if err != nil {
                return nil, err
            }
            defer rc.Close()

            // Parse XML
            var nuspec Nuspec
            err = xml.NewDecoder(rc).Decode(&nuspec)
            return &nuspec, err
        }
    }
    return nil, errors.New(".nuspec not found")
}
```

---

## Go Implementation Recommendations

### Essential Libraries

```go
// HTTP client with retry
"github.com/hashicorp/go-retryablehttp"

// HTTP caching
"github.com/gregjones/httpcache"
"github.com/gregjones/httpcache/diskcache"

// Semantic versioning
"github.com/Masterminds/semver/v3"

// TUI framework
"github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/bubbles/list"
"github.com/charmbracelet/bubbles/textinput"
"github.com/charmbracelet/bubbles/table"

// XML parsing (for .nuspec)
"encoding/xml"

// ZIP reading (for .nupkg)
"archive/zip"
```

### Minimal API Surface

You DON'T need most of NuGet.Client! For LazyNuGet, implement only:

**Core Types**:
```go
type ServiceIndex struct {
    Resources []ServiceResource
}

type PackageSearchResult struct {
    TotalHits int64
    Data      []PackageMetadata
}

type PackageMetadata struct {
    ID              string
    Version         string
    Description     string
    Authors         []string
    DownloadCount   int64
    ProjectURL      string
    LicenseURL      string
    Tags            []string
    Versions        []VersionInfo
}

type VersionInfo struct {
    Version       string
    DownloadCount int64
}
```

**Core Operations**:
1. `FetchServiceIndex(url string) (*ServiceIndex, error)`
2. `SearchPackages(query string, opts SearchOptions) (*PackageSearchResult, error)`
3. `ListVersions(packageID string) ([]string, error)`
4. `DownloadPackage(packageID, version string) (io.ReadCloser, error)`
5. `GetPackageMetadata(packageID, version string) (*PackageMetadata, error)`
6. `InstallPackage(projectPath, packageID, version string) error`

**That's it!** You don't need:
- ❌ Package creation
- ❌ Package signing/verification (for MVP)
- ❌ Complex dependency resolution (can be added later)
- ❌ MSBuild integration
- ❌ Visual Studio APIs
- ❌ Feed management
- ❌ Push/publish

### Project Structure

```
lazynuget/
├── cmd/
│   └── lazynuget/
│       └── main.go              # Entry point
├── internal/
│   ├── nuget/
│   │   ├── client.go            # HTTP client + service discovery
│   │   ├── search.go            # Search API
│   │   ├── download.go          # Download API
│   │   ├── metadata.go          # Metadata API
│   │   ├── types.go             # Core types
│   │   └── version.go           # Version handling (wraps semver)
│   ├── project/
│   │   ├── discover.go          # Find .csproj/.sln files
│   │   ├── parse.go             # Parse .csproj XML
│   │   └── install.go           # Add PackageReference
│   ├── config/
│   │   └── config.go            # App config (sources, cache)
│   └── ui/
│       ├── app.go               # Bubbletea app model
│       ├── search.go            # Search view
│       ├── projects.go          # Project list view
│       └── installed.go         # Installed packages view
├── go.mod
└── README.md
```

---

## Critical Gotchas for Go Implementation

### 1. Case-Insensitive Package IDs

NuGet package IDs are **case-insensitive** but **case-preserving**:
```go
// "Newtonsoft.Json" == "newtonsoft.json" == "NEWTONSOFT.JSON"
// But APIs require lowercase in URLs

func normalizeID(id string) string {
    return strings.ToLower(id)
}
```

### 2. Version Normalization

Versions must be normalized for API calls:
```go
// "1.0" -> "1.0.0"
// "1.0.0.0" -> "1.0.0"
// Leading zeros removed: "1.01.1" -> "1.1.1"

func normalizeVersion(v *semver.Version) string {
    // Use semver library's String() method
    return v.String()
}
```

### 3. Metadata Can Be String OR Array

Authors, owners, @type fields can be string or array in JSON:
```json
// Both valid:
{ "authors": "John Doe" }
{ "authors": ["John", "Jane"] }
```

Use the `FlexibleStringArray` unmarshaler pattern shown above.

### 4. HTTP Endpoints Have Fallbacks

Service index can list multiple endpoints for same @type:
```go
// Try each endpoint until one succeeds
func searchWithFallback(endpoints []string, query string) (*SearchResult, error) {
    for i, endpoint := range endpoints {
        result, err := search(endpoint, query)
        if err == nil {
            return result, nil
        }
        if i == len(endpoints)-1 {
            return nil, err // Last one failed
        }
        // Try next endpoint
    }
    return nil, errors.New("all endpoints failed")
}
```

### 5. .nupkg Files Are ZIP Archives

```go
// Always:
// 1. .nupkg is a ZIP file
// 2. .nuspec is at root (no directory prefix)
// 3. .nuspec filename matches package ID (case-insensitive)

// Example:
// Newtonsoft.Json.13.0.3.nupkg contains:
//   newtonsoft.json.nuspec       <- Read this for metadata
//   lib/net45/Newtonsoft.Json.dll
//   lib/netstandard2.0/Newtonsoft.Json.dll
```

### 6. SemVer Level

Always add `semVerLevel=2.0.0` to queries to get latest versions:
```go
queryParams.Set("semVerLevel", "2.0.0")
```

Without this, you won't see packages with SemVer 2.0 versions (e.g., "1.0.0+build").

---

## Performance Optimization Tips

### 1. Cache Service Index

Fetch once per session, cache in memory:
```go
var (
    serviceIndexCache *ServiceIndex
    serviceIndexMutex sync.RWMutex
)

func getServiceIndex() (*ServiceIndex, error) {
    serviceIndexMutex.RLock()
    if serviceIndexCache != nil {
        defer serviceIndexMutex.RUnlock()
        return serviceIndexCache, nil
    }
    serviceIndexMutex.RUnlock()

    serviceIndexMutex.Lock()
    defer serviceIndexMutex.Unlock()

    // Fetch from network
    index, err := fetchServiceIndex()
    if err != nil {
        return nil, err
    }
    serviceIndexCache = index
    return index, nil
}
```

### 2. HTTP Response Caching

Use `httpcache` library with disk cache:
```go
cache := diskcache.New(filepath.Join(os.UserCacheDir(), "lazynuget", "http"))
transport := httpcache.NewTransport(cache)
client := &http.Client{Transport: transport}
```

### 3. Concurrent Requests

When listing installed packages, fetch metadata concurrently:
```go
type result struct {
    pkg *PackageMetadata
    err error
}

results := make(chan result, len(packages))
for _, pkgRef := range packages {
    go func(id, version string) {
        pkg, err := client.GetMetadata(id, version)
        results <- result{pkg, err}
    }(pkgRef.ID, pkgRef.Version)
}

for range packages {
    r := <-results
    // Process result
}
```

### 4. Stream Large Responses

For search results, use streaming JSON parser if handling many results:
```go
decoder := json.NewDecoder(resp.Body)
var result SearchResult
if err := decoder.Decode(&result); err != nil {
    return nil, err
}
```

---

## Testing Strategy

### 1. Use Real API in Tests (with caching)

```go
func TestSearch(t *testing.T) {
    client := NewClient(WithCache(true))
    results, err := client.Search("newtonsoft", SearchOptions{
        Take: 10,
    })
    require.NoError(t, err)
    assert.Greater(t, len(results.Data), 0)
}
```

### 2. Mock for CI/CD

Create interface for HTTP client, mock in tests:
```go
type HTTPClient interface {
    Get(url string) (*http.Response, error)
}

type NuGetClient struct {
    http HTTPClient
}

// Tests use mock
type MockHTTP struct {
    Responses map[string]*http.Response
}
```

### 3. Golden File Tests

Save real API responses, test parsing:
```go
func TestParseSearchResponse(t *testing.T) {
    data, _ := os.ReadFile("testdata/search_newtonsoft.json")
    var result SearchResult
    err := json.Unmarshal(data, &result)
    require.NoError(t, err)
    assert.Equal(t, "Newtonsoft.Json", result.Data[0].ID)
}
```

---

## Summary: What You Actually Need

**Total LOC Estimate for Complete Go Implementation**: ~15,000-20,000 lines

This is a COMPLETE, enterprise-grade NuGet client library that meets and exceeds the C# NuGet.Client.

**Core Packages** (Complete Implementation):

### API Layer (~3,000 lines)
1. `api/v3/` - NuGet v3 API client
   - `search.go` - Search API
   - `download.go` - Package download API
   - `metadata.go` - Metadata/registration API
   - `publish.go` - Push/publish API
   - `index.go` - Service index discovery
2. `api/v2/` - NuGet v2 API client (OData)
   - `odata.go` - OData protocol
   - `search.go` - Search API
   - `download.go` - Package download

### Core Models (~2,000 lines)
3. `models/` - Core types
   - `package.go` - Package identity, metadata
   - `version.go` - Version handling
   - `dependency.go` - Dependency specification
   - `framework.go` - Target framework
   - `metadata.go` - Package metadata

### Versioning (~1,500 lines)
4. `version/` - Semantic versioning (wraps Masterminds/semver)
   - `version.go` - Version parsing
   - `range.go` - Version range (dependencies)
   - `compare.go` - Version comparison
   - `normalize.go` - Normalization

### Framework Support (~1,500 lines)
5. `framework/` - Target framework moniker (TFM)
   - `framework.go` - Framework parsing
   - `parser.go` - TFM parser
   - `compat.go` - Compatibility checking
   - `aliases.go` - Framework aliases (net8.0, netstandard2.1, etc.)

### Package Creation (~2,000 lines)
6. `pack/` - Package creation
   - `builder.go` - Package builder
   - `writer.go` - .nupkg writer
   - `reader.go` - .nupkg reader
   - `validator.go` - Package validator

### Signing & Verification (~2,000 lines)
7. `signature/` - Package signing
   - `signer.go` - Package signer
   - `verifier.go` - Signature verifier
   - `cert.go` - Certificate handling
   - `timestamp.go` - Timestamping

### Dependency Resolution (~2,500 lines)
8. `resolver/` - Dependency resolution
   - `resolver.go` - Main resolver
   - `graph.go` - Dependency graph
   - `conflict.go` - Conflict detection
   - `strategy.go` - Resolution strategies

### Project File Handling (~1,500 lines)
9. `project/` - MSBuild project files
   - `csproj.go` - C# project
   - `fsproj.go` - F# project
   - `vbproj.go` - VB project
   - `sln.go` - Solution files
   - `packages.go` - packages.config

### HTTP Client (~2,000 lines)
10. `http/` - Advanced HTTP client
    - `client.go` - HTTP client with retry/cache
    - `retry.go` - Retry logic with backoff
    - `circuit.go` - Circuit breaker
    - `ratelimit.go` - Rate limiting
    - `progress.go` - Progress tracking

**Total Core Library**: ~18,000 lines

**CLI Tool**: ~1,500 lines

**Tests**: ~10,000 lines

**Grand Total**: ~30,000 lines for complete, enterprise-grade implementation

**Key Specs to Follow**:
- ✅ Service Index at `/v3/index.json`
- ✅ Search: `SearchQueryService` resource with query params
- ✅ Download: `PackageBaseAddress/{id-lower}/{version-lower}/{id-lower}.{version-lower}.nupkg`
- ✅ All IDs and versions lowercase in URLs
- ✅ Include `semVerLevel=2.0.0` in queries
- ✅ Handle string-or-array JSON fields
- ✅ .nupkg files are ZIPs with .nuspec at root

**Tricks to Copy from C#**:
1. Service index caching
2. HTTP retry with exponential backoff
3. Flexible JSON unmarshaling
4. Endpoint fallbacks
5. Case-insensitive ID comparison

**IMPORTANT: DO NOT SKIP ANYTHING**:
This review identifies what to implement for FULL feature parity and beyond. The gonuget library will be enterprise-grade and production-ready with ALL features:
- ✅ Package creation/packing
- ✅ Complex dependency resolution
- ✅ MSBuild integration (.csproj editing)
- ✅ Signature verification
- ✅ V2 AND V3 API support (complete compatibility)

---

## mtlog Integration

### Why mtlog is Perfect for gonuget

**mtlog** (`github.com/willibrandon/mtlog`) is a high-performance structured logging library for Go that brings Serilog-style message templates to Go. It's an excellent fit for gonuget:

1. **Zero-allocation logging** - 17.3 ns/op for simple messages
2. **Message templates** - Structured logging with `{PropertyName}` syntax
3. **Multiple sinks** - Console, File, Seq, Elasticsearch, Splunk, OTLP, Sentry
4. **HTTP middleware** - Built-in support for net/http, Gin, Echo, Fiber, Chi
5. **Per-message sampling** - Intelligent log volume control for production
6. **OpenTelemetry integration** - Automatic trace correlation
7. **Context-aware** - Deadline warnings, distributed tracing
8. **ForType logging** - Automatic SourceContext from Go types

### mtlog in gonuget Architecture

```go
import (
    "github.com/willibrandon/mtlog"
    "github.com/willibrandon/mtlog/core"
    "github.com/willibrandon/mtlog/sinks"
)

// Create logger for gonuget
logger := mtlog.New(
    mtlog.WithConsoleTheme(sinks.LiterateTheme()),
    mtlog.WithSeq("http://localhost:5341"),
    mtlog.WithRollingFile("gonuget.log", 10*1024*1024),
    mtlog.WithMinimumLevel(core.DebugLevel),
    mtlog.WithTimestamp(),
    mtlog.WithCallersInfo(),
)

// Type-based logging for sub-components
searchLogger := mtlog.ForType[SearchClient](logger)
searchLogger.Info("Searching for {PackageId} in {Source}", pkgID, source)
// Output: [12:34:56 INF] SearchClient: Searching for Newtonsoft.Json in nuget.org

downloadLogger := mtlog.ForType[DownloadClient](logger)
downloadLogger.With("size_mb", 2.5).Info("Downloaded {PackageId} version {Version}", id, ver)
// Output: [12:34:56 INF] DownloadClient: Downloaded Newtonsoft.Json version 13.0.3 (size_mb=2.5)
```

### HTTP Request/Response Logging

```go
// Use mtlog's HTTP middleware for automatic request logging
client := &http.Client{
    Transport: middleware.NewHTTPTransport(
        logger,
        middleware.WithLogRequests(true),
        middleware.WithLogResponses(true),
        middleware.WithSanitizeHeaders([]string{"Authorization", "X-NuGet-ApiKey"}),
    ),
}

// All HTTP calls are automatically logged:
// [12:34:56 DBG] HTTP Request: GET https://api.nuget.org/v3/search?q=Serilog
// [12:34:56 INF] HTTP Response: 200 OK in 123ms (status=200, duration_ms=123)
```

### Distributed Tracing with OTEL

```go
import "github.com/willibrandon/mtlog/adapters/otel"

// OTLP sink with automatic trace correlation
logger := mtlog.New(
    otel.WithOTLPSink(
        otel.WithOTLPEndpoint("localhost:4317"),
        otel.WithOTLPInsecure(),
        otel.WithOTLPBatching(100, 5*time.Second),
    ),
)

// Logs automatically include trace.id, span.id from context
ctx := context.Background()
logger.InfoContext(ctx, "Package {PackageId} downloaded", pkgID)
// Includes: trace.id, span.id if context has active span
```

### Production Sampling

```go
// Sample logs in high-volume scenarios
logger := mtlog.New(
    mtlog.WithConsole(),
    mtlog.SampleProfile("ProductionAPI"),  // 10% sampling
)

// Only 10% of API calls logged in detail
for i := 0; i < 10000; i++ {
    logger.Sample(10).Info("API call {Index}", i)  // Every 10th logged
}
```

### Why mtlog > Other Go Logging Libraries

| Feature | mtlog | zap | zerolog | logrus |
|---------|-------|-----|---------|--------|
| Message templates | ✅ | ❌ | ❌ | ❌ |
| Zero allocations | ✅ | ✅ | ✅ | ❌ |
| Structured fields | ✅ | ✅ | ✅ | ✅ |
| Multiple sinks | ✅ (10+) | ⚠️ (basic) | ⚠️ (basic) | ✅ |
| HTTP middleware | ✅ (5 frameworks) | ❌ | ❌ | ❌ |
| OTEL integration | ✅ (native) | ⚠️ (manual) | ⚠️ (manual) | ❌ |
| Sampling | ✅ (advanced) | ⚠️ (basic) | ⚠️ (basic) | ❌ |
| Seq integration | ✅ | ❌ | ❌ | ❌ |
| ForType logging | ✅ | ❌ | ❌ | ❌ |
| Performance | 17ns | 147ns | 36ns | 300ns+ |

mtlog offers the best combination of features and performance for this use case.

---

## Next Steps

### Phase 1: Core Foundation (2 weeks)
1. **Create gonuget repository** - GitHub, go.mod, structure
2. **HTTP client with mtlog** - Retry, cache, circuit breaker, logging
3. **Service discovery** - V3 index.json parsing
4. **Search API** - V3 search with pagination
5. **Download API** - V3 package download with progress
6. **Metadata API** - V3 registration/metadata
7. **Semantic versioning** - Parse, compare, ranges
8. **Framework compatibility** - TFM parsing, compat checking
9. **Unit tests** - Mock HTTP responses
10. **Integration tests** - Real NuGet.org API

### Phase 2: Advanced Features (2 weeks)
11. **Package creation** - Builder, writer, validator
12. **Package signing** - X.509 signing, timestamping
13. **Signature verification** - Verify signed packages
14. **Dependency resolution** - Graph building, conflict detection
15. **NuGet v2 API** - OData protocol, search, download
16. **Authentication** - Basic, Bearer, NTLM, OAuth2, API key
17. **Multi-feed support** - Source registry, per-source auth
18. **Progress tracking** - Download progress with channels
19. **CLI tool** - Search, download, install, pack, sign, push
20. **Benchmarks** - Performance vs C# client

### Phase 3: Production Ready (1 week)
21. **Comprehensive docs** - API reference, guides, examples
22. **Example gallery** - 20+ runnable examples
23. **Performance optimization** - Profile, optimize hot paths
24. **Error handling** - Rich error types, helpful messages
25. **CI/CD pipeline** - GitHub Actions, tests, releases
26. **Security scanning** - Dependency audits, SAST
27. **Package to registries** - Go package registry, Docker Hub

### Phase 4: LazyNuGet TUI (1 week)
28. **Bubbletea TUI** - Beautiful, responsive terminal UI
29. **Project discovery** - Find .csproj, .sln files
30. **Package browsing** - Search, filter, sort packages
31. **Installation workflow** - Add PackageReference to projects
32. **Update checking** - Find outdated packages
33. **Uninstallation** - Remove packages from projects
34. **Configuration** - Manage sources, cache, settings

**Total Timeline**: 6 weeks to complete, production-ready gonuget + LazyNuGet

You'll have:
- ✅ Enterprise-grade NuGet client library (gonuget)
- ✅ Beautiful TUI application (LazyNuGet)
- ✅ Feature parity and beyond vs C# NuGet.Client
- ✅ 2-3x better performance
- ✅ Structured logging with mtlog
- ✅ Full OTEL observability
- ✅ Single binary, no runtime dependencies
