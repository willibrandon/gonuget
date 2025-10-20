# gonuget - Product Requirements Document: Protocol Implementation

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-10-19
**Owner:** Engineering

---

## Table of Contents

1. [Overview](#overview)
2. [NuGet v3 Protocol](#nuget-v3-protocol)
3. [NuGet v2 Protocol](#nuget-v2-protocol)
4. [Service Discovery](#service-discovery)
5. [Authentication](#authentication)
6. [Resource Types](#resource-types)
7. [Performance Requirements](#performance-requirements)
8. [Acceptance Criteria](#acceptance-criteria)

---

## Overview

This document specifies requirements for implementing NuGet v2 (OData) and v3 (JSON/REST) protocols.

**Protocol Versions:**
- **v3**: Modern JSON-based REST API (primary focus)
- **v2**: Legacy OData XML API (compatibility)

**Related Design Documents:**
- DESIGN-PROTOCOL.md - Protocol design and implementation

---

## NuGet v3 Protocol

### Requirement V3-001: Service Index

**Priority:** P0 (Critical)
**Component:** `protocol/v3` package

**Description:**
Fetch and parse service index (index.json) to discover available endpoints.

**Endpoint:**
```
GET {feedURL}/index.json
```

**Response Format:**
```json
{
  "version": "3.0.0",
  "resources": [
    {
      "@id": "https://api.nuget.org/v3/registration5-gz-semver2/{id-lower}/index.json",
      "@type": "RegistrationsBaseUrl/3.6.0",
      "comment": "Package metadata"
    },
    {
      "@id": "https://azuresearch-usnc.nuget.org/query",
      "@type": "SearchQueryService",
      "comment": "Query endpoint"
    }
  ]
}
```

**Functional Requirements:**

1. **Fetch service index:**
   - HTTP GET to {feedURL}/index.json
   - Parse JSON response
   - Extract resource list

2. **Resource discovery:**
   - Find resource by @type
   - Handle multiple resources with same @type (use first)
   - Handle resource version suffixes (e.g., "/3.6.0")

3. **Caching:**
   - Cache service index for 40 minutes (per C# client)
   - Revalidate on cache expiry
   - Support force refresh

4. **Error handling:**
   - 404: Feed not v3 compatible
   - Network errors: Retry with backoff
   - Invalid JSON: Return error

**Resource Types in Index:**
- `SearchQueryService` - Package search
- `RegistrationsBaseUrl` - Package metadata
- `PackageBaseAddress` - Package download
- `PackagePublish` - Package push endpoint
- `SearchAutocompleteService` - Autocomplete
- `ReportAbuseUriTemplate` - Abuse reporting
- `LegacyGallery` - Legacy UI URL

**API:**
```go
type ServiceIndex struct {
    Version string
    Resources []*ServiceResource
}

type ServiceResource struct {
    ID string      // @id
    Type string    // @type
    Comment string // comment (optional)
}

func FetchServiceIndex(ctx context.Context, feedURL string) (*ServiceIndex, error)
func (s *ServiceIndex) GetResourceURL(resourceType string) (string, error)
```

**Performance Requirements:**
- Response time: <500ms (network dependent)
- Cache hit time: <5ms
- Parse time: <10ms

**Acceptance Criteria:**
- ✅ Fetches and parses service index correctly
- ✅ Finds resources by type
- ✅ Caching works (40-minute TTL)
- ✅ Error handling complete

---

### Requirement V3-002: Package Search

**Priority:** P0 (Critical)
**Component:** `protocol/v3` package

**Description:**
Search for packages using SearchQueryService endpoint.

**Endpoint:**
```
GET {SearchQueryService}?q={query}&skip={skip}&take={take}&prerelease={bool}&semVerLevel=2.0.0
```

**Query Parameters:**
- `q`: Search query (package ID, tags, description)
- `skip`: Number of results to skip (pagination)
- `take`: Number of results to return (default 20, max 1000)
- `prerelease`: Include prerelease packages (default false)
- `semVerLevel`: SemVer level (2.0.0 for SemVer 2.0 support)

**Response Format:**
```json
{
  "totalHits": 1234,
  "data": [
    {
      "id": "Newtonsoft.Json",
      "version": "13.0.1",
      "versions": [
        {"version": "13.0.1", "@id": "..."},
        {"version": "13.0.2", "@id": "..."}
      ],
      "description": "Json.NET is a popular high-performance JSON framework for .NET",
      "authors": ["James Newton-King"],
      "iconUrl": "https://...",
      "licenseUrl": "https://...",
      "projectUrl": "https://...",
      "tags": ["json", "serialization"],
      "totalDownloads": 1234567890
    }
  ]
}
```

**Functional Requirements:**

1. **Query building:**
   - URL-encode query string
   - Build query parameters
   - Handle special characters

2. **Response parsing:**
   - Parse JSON response
   - Extract package metadata
   - Handle flexible string/array fields (authors, tags)

3. **Pagination:**
   - Support skip/take
   - Return total hit count
   - Handle large result sets

4. **Filtering:**
   - Prerelease filtering
   - SemVer level filtering
   - Framework filtering (via query)

**API:**
```go
type SearchRequest struct {
    Query string
    Skip int
    Take int
    IncludePrerelease bool
    SemVerLevel string
}

type SearchResponse struct {
    TotalHits int
    Data []*PackageSearchResult
}

type PackageSearchResult struct {
    ID string
    Version string
    Description string
    Authors []string
    IconURL string
    LicenseURL string
    ProjectURL string
    Tags []string
    TotalDownloads int64
    Versions []*PackageVersionInfo
}

func Search(ctx context.Context, serviceURL string, req *SearchRequest) (*SearchResponse, error)
```

**Edge Cases:**
- Empty query string (return popular packages)
- Zero results
- Malformed JSON response
- Missing fields (handle gracefully)

**Performance Requirements:**
- Query time: <1s (network dependent)
- Parse 100 results: <50ms

**Acceptance Criteria:**
- ✅ Searches return relevant packages
- ✅ Pagination works correctly
- ✅ Filtering applied correctly
- ✅ Edge cases handled

---

### Requirement V3-003: Package Metadata

**Priority:** P0 (Critical)
**Component:** `protocol/v3` package

**Description:**
Fetch package metadata from RegistrationsBaseUrl endpoint.

**Endpoint:**
```
GET {RegistrationsBaseUrl}/{id-lower}/index.json
GET {RegistrationsBaseUrl}/{id-lower}/{version-lower}.json
```

**Index Response (All Versions):**
```json
{
  "@id": "https://api.nuget.org/v3/registration5-gz-semver2/newtonsoft.json/index.json",
  "count": 1,
  "items": [
    {
      "@id": "https://...",
      "lower": "1.0.0",
      "upper": "13.0.2",
      "count": 100,
      "items": [
        {
          "catalogEntry": {
            "id": "Newtonsoft.Json",
            "version": "13.0.1",
            "description": "...",
            "authors": "James Newton-King",
            "dependencyGroups": [
              {
                "targetFramework": "netstandard2.0",
                "dependencies": [
                  {"id": "System.Text.Json", "range": "[5.0.0, )"}
                ]
              }
            ],
            "listed": true,
            "published": "2021-03-17T12:00:00Z"
          }
        }
      ]
    }
  ]
}
```

**Specific Version Response:**
```json
{
  "catalogEntry": {
    "id": "Newtonsoft.Json",
    "version": "13.0.1",
    "description": "...",
    "authors": "James Newton-King",
    "dependencyGroups": [...],
    "packageContent": "https://.../newtonsoft.json.13.0.1.nupkg"
  }
}
```

**Functional Requirements:**

1. **Fetch all versions:**
   - GET /index.json for package ID
   - Parse paginated items
   - Extract all version metadata

2. **Fetch specific version:**
   - GET /{version}.json for specific version
   - Parse catalog entry
   - Extract metadata

3. **Dependency parsing:**
   - Parse dependency groups
   - Parse framework-specific dependencies
   - Handle version ranges

4. **Flexible JSON parsing:**
   - Authors can be string or array
   - Tags can be string (comma-separated) or array
   - Handle missing fields

**API:**
```go
type RegistrationIndex struct {
    ID string
    Count int
    Items []*RegistrationPage
}

type RegistrationPage struct {
    ID string
    Lower string // Lower version bound
    Upper string // Upper version bound
    Count int
    Items []*RegistrationLeaf
}

type RegistrationLeaf struct {
    CatalogEntry *CatalogEntry
}

type CatalogEntry struct {
    ID string
    Version string
    Description string
    Authors []string
    DependencyGroups []*DependencyGroup
    PackageContent string // .nupkg download URL
    Listed bool
    Published time.Time
}

func FetchRegistrationIndex(ctx context.Context, baseURL, packageID string) (*RegistrationIndex, error)
func FetchRegistrationLeaf(ctx context.Context, baseURL, packageID, version string) (*RegistrationLeaf, error)
```

**Performance Requirements:**
- Fetch index: <1s
- Fetch leaf: <500ms
- Parse 100 versions: <100ms

**Acceptance Criteria:**
- ✅ Fetches all package versions
- ✅ Fetches specific version metadata
- ✅ Parses dependencies correctly
- ✅ Handles flexible JSON

---

### Requirement V3-004: Package Download

**Priority:** P0 (Critical)
**Component:** `protocol/v3` package

**Description:**
Download .nupkg package files from PackageBaseAddress endpoint.

**Endpoint:**
```
GET {PackageBaseAddress}/{id-lower}/{version-lower}/{id-lower}.{version-lower}.nupkg
```

**Example:**
```
https://api.nuget.org/v3-flatcontainer/newtonsoft.json/13.0.1/newtonsoft.json.13.0.1.nupkg
```

**Functional Requirements:**

1. **URL construction:**
   - Lowercase package ID
   - Lowercase version
   - Construct .nupkg URL

2. **Download:**
   - HTTP GET request
   - Stream response to io.Reader
   - Support partial downloads (resume)

3. **Validation:**
   - Verify Content-Length
   - Verify Content-Type (application/zip)
   - Optional: Verify hash

4. **Error handling:**
   - 404: Package not found
   - Network errors: Retry with backoff
   - Timeout: Configurable

**API:**
```go
func DownloadPackage(ctx context.Context, baseURL string, id string, version *NuGetVersion) (io.ReadCloser, error)
func DownloadPackageToFile(ctx context.Context, baseURL string, id string, version *NuGetVersion, path string) error
```

**Performance Requirements:**
- Streaming (no full buffer in memory)
- Support parallel downloads
- Resume support for large packages

**Acceptance Criteria:**
- ✅ Downloads packages correctly
- ✅ Streams efficiently
- ✅ Error handling complete

---

### Requirement V3-005: Autocomplete

**Priority:** P1 (High)
**Component:** `protocol/v3` package

**Description:**
Autocomplete package IDs and versions from SearchAutocompleteService.

**Endpoints:**
```
GET {SearchAutocompleteService}?q={query}&skip={skip}&take={take}&prerelease={bool}&semVerLevel=2.0.0
GET {SearchAutocompleteService}?id={packageId}&prerelease={bool}&semVerLevel=2.0.0
```

**Package ID Autocomplete Response:**
```json
{
  "totalHits": 10,
  "data": [
    "Newtonsoft.Json",
    "Newtonsoft.Json.Schema",
    "Newtonsoft.Json.Bson"
  ]
}
```

**Version Autocomplete Response:**
```json
{
  "totalHits": 5,
  "data": [
    "13.0.1",
    "13.0.2",
    "12.0.3",
    "11.0.2",
    "10.0.3"
  ]
}
```

**Functional Requirements:**

1. **Package ID autocomplete:**
   - Partial package ID match
   - Return matching package IDs
   - Support pagination

2. **Version autocomplete:**
   - Given package ID, return versions
   - Sort by version (descending)
   - Filter prerelease

**API:**
```go
func AutocompletePackageIDs(ctx context.Context, serviceURL string, query string, opts *AutocompleteOptions) ([]string, error)
func AutocompleteVersions(ctx context.Context, serviceURL string, packageID string, opts *AutocompleteOptions) ([]string, error)
```

**Acceptance Criteria:**
- ✅ Autocompletes package IDs
- ✅ Autocompletes versions
- ✅ Filtering works

---

### Requirement V3-006: Package Push

**Priority:** P1 (High)
**Component:** `protocol/v3` package

**Description:**
Push packages to PackagePublish endpoint.

**Endpoint:**
```
PUT {PackagePublish}
Content-Type: multipart/form-data
X-NuGet-ApiKey: {apiKey}
```

**Request:**
- Multipart form with .nupkg file
- API key in header or URL query

**Response:**
- 201 Created: Package published successfully
- 409 Conflict: Package version already exists
- 401 Unauthorized: Invalid API key
- 400 Bad Request: Invalid package

**Functional Requirements:**

1. **Upload package:**
   - Read .nupkg file
   - Create multipart request
   - Set API key header
   - Stream upload

2. **Validation:**
   - Verify package before upload
   - Check package signature
   - Verify .nuspec

3. **Error handling:**
   - Parse error response
   - Provide actionable messages

**API:**
```go
func PushPackage(ctx context.Context, serviceURL string, packagePath string, apiKey string) error
```

**Acceptance Criteria:**
- ✅ Pushes packages successfully
- ✅ Authentication works
- ✅ Error handling complete

---

## NuGet v2 Protocol

### Requirement V2-001: OData Feed Discovery

**Priority:** P1 (High)
**Component:** `protocol/v2` package

**Description:**
Discover v2 feed capabilities via OData service document.

**Endpoint:**
```
GET {feedURL}/$metadata
```

**Response:** OData XML schema

**Functional Requirements:**

1. **Feed detection:**
   - Check for OData metadata endpoint
   - Verify Packages entity set exists

2. **Capabilities:**
   - Detect search support
   - Detect filter support
   - Detect orderby support

**API:**
```go
func DetectV2Feed(ctx context.Context, feedURL string) (bool, error)
```

**Acceptance Criteria:**
- ✅ Detects v2 feeds correctly
- ✅ Identifies capabilities

---

### Requirement V2-002: Package Search (v2)

**Priority:** P1 (High)
**Component:** `protocol/v2` package

**Description:**
Search packages using v2 OData query.

**Endpoint:**
```
GET {feedURL}/Search()?$filter=...&$orderby=...&$skip=...&$top=...&searchTerm='...'&targetFramework='...'&includePrerelease=false
```

**Response Format:** OData XML or JSON

**Functional Requirements:**

1. **Query building:**
   - Build OData filter expressions
   - Build orderby clauses
   - Handle pagination ($skip, $top)

2. **Response parsing:**
   - Parse OData XML (Atom feed)
   - Extract package metadata
   - Handle properties

**API:**
```go
func SearchV2(ctx context.Context, feedURL string, req *SearchRequest) (*SearchResponse, error)
```

**Acceptance Criteria:**
- ✅ Searches v2 feeds
- ✅ Parses OData responses
- ✅ Pagination works

---

### Requirement V2-003: Package Metadata (v2)

**Priority:** P1 (High)
**Component:** `protocol/v2` package

**Description:**
Fetch package metadata from v2 feed.

**Endpoint:**
```
GET {feedURL}/Packages(Id='{id}',Version='{version}')
```

**Response:** OData entry (XML or JSON)

**Functional Requirements:**

1. **Fetch package:**
   - Query by ID and version
   - Parse OData entry
   - Extract metadata

2. **Dependencies:**
   - Parse pipe-separated dependency string
   - Parse version ranges
   - Parse target frameworks

**Dependency String Format:**
```
Id1:VersionRange1:TargetFramework1|Id2:VersionRange2:TargetFramework2
```

**API:**
```go
func FetchMetadataV2(ctx context.Context, feedURL string, id string, version *NuGetVersion) (*PackageMetadata, error)
```

**Acceptance Criteria:**
- ✅ Fetches metadata from v2
- ✅ Parses dependencies correctly
- ✅ Handles all metadata fields

---

### Requirement V2-004: Package Download (v2)

**Priority:** P1 (High)
**Component:** `protocol/v2` package

**Description:**
Download packages from v2 feed.

**Endpoint:**
```
GET {feedURL}/Download/{id}/{version}
```

**Or:**
```
GET {contentURL from metadata}
```

**Functional Requirements:**

1. **URL determination:**
   - Prefer content URL from metadata
   - Fallback to /Download endpoint

2. **Download:**
   - HTTP GET
   - Stream response

**API:**
```go
func DownloadPackageV2(ctx context.Context, feedURL string, id string, version *NuGetVersion) (io.ReadCloser, error)
```

**Acceptance Criteria:**
- ✅ Downloads from v2 feeds
- ✅ Handles both URL patterns

---

### Requirement V2-005: Package Push (v2)

**Priority:** P1 (High)
**Component:** `protocol/v2` package

**Description:**
Push packages to v2 feed.

**Endpoint:**
```
PUT {feedURL}/
Content-Type: multipart/form-data
X-NuGet-ApiKey: {apiKey}
```

**Functional Requirements:**

1. **Upload:**
   - Multipart form data
   - API key authentication
   - Stream upload

**API:**
```go
func PushPackageV2(ctx context.Context, feedURL string, packagePath string, apiKey string) error
```

**Acceptance Criteria:**
- ✅ Pushes to v2 feeds
- ✅ Authentication works

---

## Service Discovery

### Requirement SD-001: Protocol Detection

**Priority:** P0 (Critical)
**Component:** `protocol` package

**Description:**
Automatically detect whether feed is v2 or v3.

**Detection Strategy:**

1. **Try v3 first:**
   - Attempt to fetch /index.json
   - If successful and valid JSON → v3

2. **Fall back to v2:**
   - Attempt to fetch /$metadata
   - If successful and valid OData → v2

3. **Error:**
   - Neither worked → unsupported feed

**API:**
```go
type Protocol int

const (
    ProtocolUnknown Protocol = iota
    ProtocolV2
    ProtocolV3
)

func DetectProtocol(ctx context.Context, feedURL string) (Protocol, error)
```

**Acceptance Criteria:**
- ✅ Detects v3 feeds
- ✅ Detects v2 feeds
- ✅ Returns error for invalid feeds

---

### Requirement SD-002: Service Index Caching

**Priority:** P0 (Critical)
**Component:** `protocol/v3` package

**Description:**
Cache service index to reduce redundant requests.

**Caching Strategy:**

1. **TTL-based:**
   - Default TTL: 40 minutes (per C# client)
   - Configurable TTL

2. **Revalidation:**
   - Conditional GET with If-Modified-Since
   - ETag support

3. **Force refresh:**
   - Option to bypass cache
   - Invalidate on errors

**API:**
```go
type ServiceIndexCache interface {
    Get(feedURL string) (*ServiceIndex, bool)
    Set(feedURL string, index *ServiceIndex, ttl time.Duration)
    Invalidate(feedURL string)
}
```

**Acceptance Criteria:**
- ✅ Caches service indexes
- ✅ TTL honored
- ✅ Revalidation works

---

## Authentication

### Requirement AUTH-001: API Key Authentication

**Priority:** P0 (Critical)
**Component:** `auth` package

**Description:**
Support API key authentication for private feeds.

**Methods:**

1. **Header:**
   ```
   X-NuGet-ApiKey: {apiKey}
   ```

2. **Query parameter:**
   ```
   ?apiKey={apiKey}
   ```

**Functional Requirements:**

1. **Credential storage:**
   - Secure storage (not logged)
   - Per-source credentials

2. **Request decoration:**
   - Add header/query param
   - Support both methods

**API:**
```go
type APIKeyCredentials struct {
    APIKey string
    Method APIKeyMethod // Header or Query
}

func (c *APIKeyCredentials) DecorateRequest(req *http.Request) error
```

**Acceptance Criteria:**
- ✅ API key added to requests
- ✅ Both methods supported
- ✅ Credentials not logged

---

### Requirement AUTH-002: Bearer Token Authentication

**Priority:** P1 (High)
**Component:** `auth` package

**Description:**
Support bearer token authentication (OAuth, AAD).

**Method:**
```
Authorization: Bearer {token}
```

**Functional Requirements:**

1. **Token management:**
   - Store token securely
   - Token refresh (if supported)

2. **Request decoration:**
   - Add Authorization header

**API:**
```go
type BearerTokenCredentials struct {
    Token string
    RefreshToken string
    Expiry time.Time
}

func (c *BearerTokenCredentials) DecorateRequest(req *http.Request) error
```

**Acceptance Criteria:**
- ✅ Bearer token added to requests
- ✅ Token refresh supported

---

### Requirement AUTH-003: Basic Authentication

**Priority:** P1 (High)
**Component:** `auth` package

**Description:**
Support HTTP Basic authentication.

**Method:**
```
Authorization: Basic {base64(username:password)}
```

**Functional Requirements:**

1. **Credential encoding:**
   - Base64 encode username:password
   - Add Authorization header

**API:**
```go
type BasicCredentials struct {
    Username string
    Password string
}

func (c *BasicCredentials) DecorateRequest(req *http.Request) error
```

**Acceptance Criteria:**
- ✅ Basic auth header added
- ✅ Credentials encoded correctly

---

## Resource Types

### Requirement RT-001: Search Resource

**Priority:** P0 (Critical)
**Component:** `resources` package

**Description:**
Search resource abstraction.

**Interface:**
```go
type SearchResource interface {
    Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResponse, error)
}
```

**Implementations:**
- SearchResourceV3 (uses SearchQueryService)
- SearchResourceV2 (uses OData search)

**Acceptance Criteria:**
- ✅ Interface defined
- ✅ v3 implementation
- ✅ v2 implementation

---

### Requirement RT-002: Metadata Resource

**Priority:** P0 (Critical)
**Component:** `resources` package

**Description:**
Package metadata resource abstraction.

**Interface:**
```go
type MetadataResource interface {
    GetMetadata(ctx context.Context, id string, version *NuGetVersion) (*PackageMetadata, error)
    ListVersions(ctx context.Context, id string) ([]*NuGetVersion, error)
}
```

**Implementations:**
- MetadataResourceV3 (uses RegistrationsBaseUrl)
- MetadataResourceV2 (uses OData Packages)

**Acceptance Criteria:**
- ✅ Interface defined
- ✅ v3 implementation
- ✅ v2 implementation

---

### Requirement RT-003: Download Resource

**Priority:** P0 (Critical)
**Component:** `resources` package

**Description:**
Package download resource abstraction.

**Interface:**
```go
type DownloadResource interface {
    Download(ctx context.Context, id string, version *NuGetVersion) (io.ReadCloser, error)
    DownloadToFile(ctx context.Context, id string, version *NuGetVersion, path string) error
}
```

**Implementations:**
- DownloadResourceV3 (uses PackageBaseAddress)
- DownloadResourceV2 (uses Download endpoint)

**Acceptance Criteria:**
- ✅ Interface defined
- ✅ v3 implementation
- ✅ v2 implementation

---

### Requirement RT-004: Publish Resource

**Priority:** P1 (High)
**Component:** `resources` package

**Description:**
Package publish resource abstraction.

**Interface:**
```go
type PublishResource interface {
    Push(ctx context.Context, packagePath string) error
    Delete(ctx context.Context, id string, version *NuGetVersion) error
}
```

**Implementations:**
- PublishResourceV3 (uses PackagePublish)
- PublishResourceV2 (uses PUT endpoint)

**Acceptance Criteria:**
- ✅ Interface defined
- ✅ v3 implementation
- ✅ v2 implementation

---

## Performance Requirements

### Overall Performance

**Response Times:**
- Service index fetch: <500ms
- Package search: <1s
- Metadata fetch: <500ms
- Package download (10MB): <2s on 100Mbps

**Throughput:**
- Support 100 concurrent requests
- Download 50 packages in parallel

**Caching:**
- Service index cache hit: <5ms
- Metadata cache hit: <5ms

**Resource Usage:**
- Memory: <100MB for typical workload
- No memory leaks during long-running operations

---

## Acceptance Criteria

### Protocol Compliance

**v3 Protocol:**
- ✅ Implements all core resource types
- ✅ Parses service index correctly
- ✅ Search returns correct results
- ✅ Metadata parsing complete
- ✅ Download works reliably
- ✅ Tested against nuget.org

**v2 Protocol:**
- ✅ Detects v2 feeds
- ✅ Parses OData responses
- ✅ Search works
- ✅ Metadata fetch works
- ✅ Download works
- ✅ Tested against legacy feeds

### Compatibility Testing

**Test Feeds:**
- ✅ nuget.org (v3)
- ✅ Azure Artifacts (v3)
- ✅ MyGet (v3)
- ✅ Legacy v2 feeds

**Operations:**
- ✅ Search for packages
- ✅ List versions
- ✅ Fetch metadata
- ✅ Download packages
- ✅ Push packages (with auth)

### Error Handling

**Network Errors:**
- ✅ Timeout handling
- ✅ Retry logic
- ✅ Connection errors

**Protocol Errors:**
- ✅ Invalid JSON/XML
- ✅ Missing resources
- ✅ Malformed responses

**Authentication:**
- ✅ Invalid credentials
- ✅ Expired tokens
- ✅ Missing API keys

---

## Related Documents

- PRD-OVERVIEW.md - Product vision and goals
- PRD-CORE.md - Core library requirements
- PRD-PACKAGING.md - Package operations
- PRD-INFRASTRUCTURE.md - HTTP, caching, observability
- PRD-TESTING.md - Testing requirements
- PRD-RELEASE.md - Release criteria

---

**END OF PRD-PROTOCOL.md**
