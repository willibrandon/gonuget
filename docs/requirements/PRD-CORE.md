# gonuget - Product Requirements Document: Core Library

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-10-19
**Owner:** Engineering

---

## Table of Contents

1. [Overview](#overview)
2. [Version Handling](#version-handling)
3. [Framework Handling](#framework-handling)
4. [Package Identity](#package-identity)
5. [Source Management](#source-management)
6. [Resource Provider System](#resource-provider-system)
7. [Client Abstractions](#client-abstractions)
8. [Configuration](#configuration)
9. [Error Handling](#error-handling)
10. [Acceptance Criteria](#acceptance-criteria)

---

## Overview

This document specifies detailed requirements for gonuget's core library functionality including version handling, framework compatibility, package identity, source management, and client abstractions.

**Related Design Documents:**
- DESIGN.md (Main architecture)
- DESIGN-VERSIONING.md (Version parsing and comparison)
- DESIGN-FRAMEWORKS.md (Framework compatibility)

---

## Version Handling

### Requirement V-001: NuGet SemVer 2.0 Parsing

**Priority:** P0 (Critical)
**Component:** `version` package

**Description:**
Must parse NuGet SemVer 2.0 version strings according to official specification.

**Format Support:**
```
Major.Minor.Patch[-Prerelease][+Metadata]
```

**Functional Requirements:**

1. **Parse valid versions:**
   - `1.0.0` → Major=1, Minor=0, Patch=0
   - `2.1.3-beta` → Major=2, Minor=1, Patch=3, Prerelease=["beta"]
   - `1.0.0-alpha.1` → Prerelease=["alpha", "1"]
   - `3.2.1+20241019` → Metadata="20241019"
   - `1.0.0-rc.1+build.123` → Both prerelease and metadata

2. **Handle edge cases:**
   - Leading zeros in numeric labels: `1.0.0-01` is valid
   - Mixed alphanumeric labels: `1.0.0-alpha1beta2`
   - Empty labels disallowed: `1.0.0-` is invalid
   - Metadata ignored in comparison: `1.0.0+a` == `1.0.0+b`

3. **Error handling:**
   - Return descriptive error for invalid format
   - No panics on any input string
   - Suggest corrections where possible

**Performance Requirements:**
- Parse time: <50ns/op
- Memory: 1 allocation maximum
- Zero allocations for cached/reused versions

**Test Requirements:**
- 100+ test cases covering valid formats
- 50+ test cases for invalid formats
- Fuzzing for robustness
- Cross-validation against C# NuGet.Versioning

**API:**
```go
func Parse(version string) (*NuGetVersion, error)
func MustParse(version string) *NuGetVersion // panic on error
```

**Acceptance Criteria:**
- ✅ All C# NuGet.Versioning test cases pass
- ✅ Performance: <50ns/op, ≤1 alloc
- ✅ 100% coverage for parsing logic
- ✅ No panics on invalid input

---

### Requirement V-002: Legacy 4-Part Version Support

**Priority:** P0 (Critical)
**Component:** `version` package

**Description:**
Support legacy 4-part version format used by older NuGet packages.

**Format:**
```
Major.Minor.Build.Revision
```

**Functional Requirements:**

1. **Parse 4-part versions:**
   - `1.0.0.0` → Major=1, Minor=0, Patch/Build=0, Revision=0
   - `2.5.3.1` → All four components parsed
   - Flag as legacy version (IsLegacyVersion=true)

2. **Normalize to SemVer:**
   - Map Build → Patch
   - Preserve Revision for comparison
   - `1.0.0.0` normalizes to `1.0.0` (Revision=0)

3. **Comparison rules:**
   - Legacy vs Legacy: Compare all four parts
   - Legacy vs SemVer: Compare Major.Minor.Patch, ignore Revision
   - `1.0.0.1` > `1.0.0.0` (legacy comparison)
   - `1.0.0.1` == `1.0.0` (legacy vs SemVer)

**API:**
```go
type NuGetVersion struct {
    Major, Minor, Patch int
    Revision int           // 0 if not legacy
    IsLegacyVersion bool
    ReleaseLabels []string
    Metadata string
}
```

**Acceptance Criteria:**
- ✅ Parse all 4-part version formats
- ✅ Comparison matches C# NuGet.Versioning behavior
- ✅ Normalization preserves comparison semantics

---

### Requirement V-003: Version Comparison

**Priority:** P0 (Critical)
**Component:** `version` package

**Description:**
Implement version comparison following NuGet rules.

**Comparison Rules:**

1. **Numeric components:** Major, then Minor, then Patch, then Revision (if legacy)
2. **Prerelease precedence:**
   - Release > Prerelease: `1.0.0` > `1.0.0-beta`
   - Label-by-label comparison: `1.0.0-alpha` < `1.0.0-beta`
   - Numeric < Alphanumeric: `1.0.0-1` < `1.0.0-alpha`
   - Longer label list > shorter: `1.0.0-alpha.1` < `1.0.0-alpha.1.2`
3. **Metadata ignored:** `1.0.0+a` == `1.0.0+b`

**Functional Requirements:**

1. **Compare() method:**
   - Returns -1 if v < other
   - Returns 0 if v == other
   - Returns 1 if v > other

2. **Convenience methods:**
   - `LessThan(other)`, `GreaterThan(other)`, `Equals(other)`
   - `LessThanOrEqual(other)`, `GreaterThanOrEqual(other)`

3. **Special cases:**
   - `nil` versions (handle gracefully)
   - Same object comparison (fast path)

**Performance Requirements:**
- Compare time: <20ns/op
- Zero allocations

**Test Requirements:**
- 200+ comparison test cases
- Cross-validation against C# for all cases
- Performance regression tests

**API:**
```go
func (v *NuGetVersion) Compare(other *NuGetVersion) int
func (v *NuGetVersion) Equals(other *NuGetVersion) bool
func (v *NuGetVersion) LessThan(other *NuGetVersion) bool
// ... more comparison methods
```

**Acceptance Criteria:**
- ✅ 100% match with C# comparison results
- ✅ Performance: <20ns/op, 0 allocs
- ✅ Handles all edge cases (nil, empty, etc.)

---

### Requirement V-004: Version Ranges

**Priority:** P0 (Critical)
**Component:** `version` package

**Description:**
Parse and evaluate version range specifications.

**Range Syntax:**

```
[1.0, 2.0]     // 1.0 ≤ x ≤ 2.0 (inclusive)
(1.0, 2.0)     // 1.0 < x < 2.0 (exclusive)
[1.0, 2.0)     // 1.0 ≤ x < 2.0 (mixed)
[1.0, )        // x ≥ 1.0 (open upper)
(, 2.0]        // x ≤ 2.0 (open lower)
1.0            // x ≥ 1.0 (implicit minimum)
```

**Functional Requirements:**

1. **Parse range strings:**
   - Support all bracket combinations: `[]`, `()`, `[)`, `(]`
   - Open-ended ranges: `[1.0, )`, `(, 2.0]`
   - Simple version: `1.0` → `[1.0, )`
   - Empty range: `(1.0, 1.0)` is valid but satisfies nothing

2. **Satisfies() check:**
   - `[1.0, 2.0].Satisfies(1.5)` → true
   - `[1.0, 2.0).Satisfies(2.0)` → false (exclusive upper)
   - `(, 2.0).Satisfies(1.5)` → true

3. **Best match selection:**
   - `FindBestMatch([1.0, 2.0], [1.1, 1.5, 2.0, 2.1])` → `1.5`
   - Highest version within range
   - Prerelease handling (exclude by default unless specified)

**API:**
```go
type VersionRange struct {
    MinVersion *NuGetVersion
    MaxVersion *NuGetVersion
    MinInclusive bool
    MaxInclusive bool
}

func ParseVersionRange(s string) (*VersionRange, error)
func (r *VersionRange) Satisfies(version *NuGetVersion) bool
func (r *VersionRange) FindBestMatch(versions []*NuGetVersion) *NuGetVersion
```

**Acceptance Criteria:**
- ✅ Parse all NuGet range syntaxes
- ✅ Satisfies() matches C# behavior
- ✅ FindBestMatch selects same version as C#

---

### Requirement V-005: Floating Versions

**Priority:** P1 (High)
**Component:** `version` package

**Description:**
Support floating version specifications for dynamic version resolution.

**Floating Syntax:**

```
1.*           // Latest 1.x version
1.2.*         // Latest 1.2.x version
*             // Latest stable version
1.0.0-*       // Latest 1.0.0 prerelease
```

**Functional Requirements:**

1. **Parse floating versions:**
   - `1.*` → Float minor and patch
   - `1.2.*` → Float patch only
   - `*` → Float all (latest stable)
   - `1.0.0-*` → Float prerelease labels

2. **Resolve to concrete version:**
   - Given list of available versions, find best match
   - `1.*` with [1.0, 1.5, 2.0] → 1.5
   - Respect prerelease rules

3. **Combine with ranges:**
   - `[1.*, 2.0)` is valid
   - Float within range constraints

**API:**
```go
type FloatingVersion struct {
    BaseVersion *NuGetVersion
    FloatRange FloatRange // Major, Minor, Patch, Prerelease
}

func ParseFloatingVersion(s string) (*FloatingVersion, error)
func (f *FloatingVersion) Resolve(available []*NuGetVersion) *NuGetVersion
```

**Acceptance Criteria:**
- ✅ Parse all floating patterns
- ✅ Resolution matches C# NuGet.Versioning

---

## Framework Handling

### Requirement F-001: TFM Parsing

**Priority:** P0 (Critical)
**Component:** `frameworks` package

**Description:**
Parse Target Framework Moniker (TFM) strings.

**TFM Formats:**

```
net8.0                    // .NET 8.0
netstandard2.1            // .NET Standard 2.1
netcoreapp3.1             // .NET Core 3.1
net48                     // .NET Framework 4.8
net5.0-windows            // .NET 5.0 for Windows
net6.0-android31.0        // .NET 6.0 for Android API 31
portable-net45+win8       // PCL Profile
```

**Functional Requirements:**

1. **Parse framework identifier:**
   - Extract framework name: `.NETFramework`, `.NETCoreApp`, `.NETStandard`
   - Parse version: `net48` → 4.8, `net8.0` → 8.0
   - Handle short names: `net`, `netstandard`, `netcoreapp`

2. **Parse platform:**
   - `net5.0-windows` → Platform=Windows
   - `net6.0-android31.0` → Platform=Android, PlatformVersion=31.0
   - `net6.0-ios15.0` → Platform=iOS, PlatformVersion=15.0

3. **PCL profiles:**
   - `portable-net45+win8+wpa81` → Profile259
   - Map to known profiles
   - Extract supported frameworks

**Performance Requirements:**
- Parse time: <100ns/op
- Memory: 1 allocation

**API:**
```go
type NuGetFramework struct {
    Framework string        // ".NETFramework", ".NETCoreApp", etc.
    Version   Version       // Framework version
    Platform  string        // Optional: "windows", "android", etc.
    PlatformVersion Version // Optional: Platform version
    Profile   string        // Optional: PCL profile
}

func ParseFramework(tfm string) (*NuGetFramework, error)
```

**Acceptance Criteria:**
- ✅ Parse all modern TFMs (net5.0+)
- ✅ Parse legacy TFMs (net45, netstandard1.x)
- ✅ Parse platform-specific TFMs
- ✅ Parse PCL profiles
- ✅ Performance: <100ns/op, ≤1 alloc

---

### Requirement F-002: Framework Compatibility

**Priority:** P0 (Critical)
**Component:** `frameworks` package

**Description:**
Determine if one framework is compatible with another.

**Compatibility Rules:**

1. **.NET Framework:**
   - Higher versions compatible with lower: net48 → net45 ✓
   - Not compatible with .NET Core/5+
   - Compatible with .NET Standard (with version limits):
     - netstandard1.0-1.6 → net45+
     - netstandard2.0 → net461+
     - netstandard2.1 → NOT compatible with .NET Framework

2. **.NET Core:**
   - netcoreapp3.1 compatible with netstandard2.1
   - netcoreapp2.x compatible with netstandard2.0

3. **.NET 5+:**
   - net5.0+ compatible with netstandard2.1
   - net5.0-windows compatible with net5.0

4. **.NET Standard:**
   - Higher versions: netstandard2.1 assets can target netstandard2.0 consumers? NO
   - Lower versions: netstandard2.0 assets can satisfy netstandard2.1 consumers? YES

**Functional Requirements:**

1. **IsCompatible() check:**
   - `net8.0.IsCompatible(netstandard2.1)` → true
   - `net48.IsCompatible(netstandard2.1)` → false
   - `netstandard2.0.IsCompatible(net461)` → true

2. **GetNearest() selection:**
   - Given target framework and list of package frameworks
   - Select most compatible framework
   - `net8.0 + [net45, netstandard2.0, net5.0]` → `net5.0`

3. **Platform compatibility:**
   - `net6.0-windows.IsCompatible(net6.0)` → false (more specific)
   - `net6.0.IsCompatible(net6.0-windows)` → true (less specific)

**API:**
```go
func (f *NuGetFramework) IsCompatible(target *NuGetFramework) bool
func GetNearest(target *NuGetFramework, frameworks []*NuGetFramework) *NuGetFramework
```

**Data Requirements:**
- Extract compatibility mappings from C# NuGet.Frameworks
- Generate Go code from DefaultFrameworkMappings.cs (~700 LOC)
- Validate mappings against C# test suite

**Performance Requirements:**
- IsCompatible check: <15ns/op
- Zero allocations

**Acceptance Criteria:**
- ✅ 100% match with C# compatibility results
- ✅ All mapping data extracted
- ✅ Performance: <15ns/op, 0 allocs
- ✅ 500+ test cases covering compatibility matrix

---

### Requirement F-003: PCL Profile Support

**Priority:** P1 (High)
**Component:** `frameworks` package

**Description:**
Support Portable Class Library (PCL) profile resolution.

**Profile Examples:**

```
Profile7    → net45, win8, wp8, wpa81
Profile259  → net45, win8, wpa81
Profile111  → net45, win8, wpa81
```

**Functional Requirements:**

1. **Parse PCL syntax:**
   - `portable-net45+win8` → Profile number lookup
   - Map to supported frameworks

2. **Compatibility check:**
   - Profile259 compatible with net45? → YES
   - Profile259 compatible with net40? → NO

3. **Profile mapping:**
   - Extract profile definitions from C# implementation
   - Generate `pcl_profiles_generated.go`

**API:**
```go
type PCLProfile struct {
    Number int
    Frameworks []*NuGetFramework
}

func ParsePCLProfile(s string) (*PCLProfile, error)
func (p *PCLProfile) IsCompatibleWith(target *NuGetFramework) bool
```

**Acceptance Criteria:**
- ✅ Parse all PCL profile syntaxes
- ✅ Accurate framework mappings
- ✅ Compatibility matches C#

---

### Requirement F-004: RID (Runtime Identifier) Support

**Priority:** P1 (High)
**Component:** `frameworks` package

**Description:**
Support Runtime Identifier (RID) for platform-specific assets.

**RID Examples:**

```
win-x64       // Windows x64
linux-x64     // Linux x64
osx-arm64     // macOS ARM64
win10-x64     // Windows 10 x64
ubuntu.20.04-x64  // Ubuntu 20.04 x64
```

**Functional Requirements:**

1. **Parse RID:**
   - Extract OS: `win`, `linux`, `osx`
   - Extract architecture: `x64`, `x86`, `arm`, `arm64`
   - Extract version: `win10`, `ubuntu.20.04`

2. **RID graph:**
   - `win10-x64` inherits from `win-x64`
   - `ubuntu.20.04-x64` inherits from `linux-x64`
   - Select most specific compatible RID

**API:**
```go
type RuntimeIdentifier struct {
    OS string
    Version string
    Architecture string
}

func ParseRID(s string) (*RuntimeIdentifier, error)
func (r *RuntimeIdentifier) IsCompatibleWith(other *RuntimeIdentifier) bool
```

**Acceptance Criteria:**
- ✅ Parse all RID formats
- ✅ RID graph compatibility

---

## Package Identity

### Requirement PI-001: PackageIdentity Type

**Priority:** P0 (Critical)
**Component:** `core` package

**Description:**
Represent unique package identification (ID + version).

**Functional Requirements:**

1. **Struct definition:**
   ```go
   type PackageIdentity struct {
       ID      string          // Case-insensitive
       Version *NuGetVersion
   }
   ```

2. **Case-insensitive ID:**
   - `Newtonsoft.Json` == `newtonsoft.json`
   - Preserve original casing
   - Normalize for comparison

3. **String representation:**
   - `ToString()` → "Newtonsoft.Json 13.0.1"
   - Parse from string: "Package.Id 1.0.0"

**API:**
```go
func NewPackageIdentity(id string, version *NuGetVersion) *PackageIdentity
func (pi *PackageIdentity) Equals(other *PackageIdentity) bool
func (pi *PackageIdentity) String() string
```

**Acceptance Criteria:**
- ✅ Case-insensitive ID comparison
- ✅ Version comparison support
- ✅ String parsing and formatting

---

### Requirement PI-002: PackageMetadata Type

**Priority:** P0 (Critical)
**Component:** `core` package

**Description:**
Represent complete package metadata from feed.

**Functional Requirements:**

```go
type PackageMetadata struct {
    Identity *PackageIdentity

    // Required fields
    Title string
    Description string
    Authors []string

    // Optional fields
    Owners []string
    ProjectURL string
    LicenseURL string
    IconURL string
    Tags []string

    // Dependencies
    DependencyGroups []*PackageDependencyGroup

    // Publishing info
    Published time.Time
    DownloadCount int64
    Listed bool
}

type PackageDependencyGroup struct {
    TargetFramework *NuGetFramework
    Dependencies []*PackageDependency
}

type PackageDependency struct {
    ID string
    VersionRange *VersionRange
}
```

**Acceptance Criteria:**
- ✅ All metadata fields supported
- ✅ Dependency groups parsed correctly
- ✅ Framework-specific dependencies

---

## Source Management

### Requirement SM-001: Source Definition

**Priority:** P0 (Critical)
**Component:** `core` package

**Description:**
Represent a NuGet package source (feed).

**Functional Requirements:**

```go
type Source struct {
    Name string          // Display name
    SourceURL string     // Feed URL
    Protocol Protocol    // V2 or V3

    // Authentication
    Credentials *Credentials

    // Configuration
    AllowInsecureConnections bool
    Timeout time.Duration
    MaxRetries int
}

type Credentials struct {
    Type CredentialType  // APIKey, Basic, Bearer
    Username string      // For Basic auth
    Password string      // For Basic auth or API key
    Token string         // For Bearer token
}
```

**API:**
```go
func NewSource(name, url string, options ...SourceOption) (*Source, error)

// Functional options
func WithCredentials(creds *Credentials) SourceOption
func WithTimeout(d time.Duration) SourceOption
func WithMaxRetries(n int) SourceOption
```

**Acceptance Criteria:**
- ✅ Support v2 and v3 feeds
- ✅ Multiple auth types
- ✅ Configuration via options

---

### Requirement SM-002: SourceRepository

**Priority:** P0 (Critical)
**Component:** `core` package

**Description:**
Manage resource providers for a source.

**Functional Requirements:**

1. **Resource provider registration:**
   - Register providers for source
   - Priority-based provider ordering
   - Cache created resources

2. **Resource retrieval:**
   - GetResource(ResourceType) returns resource
   - Lazy creation via providers
   - Thread-safe caching

**API:**
```go
type SourceRepository struct {
    source    *Source
    providers []ResourceProvider
    resources sync.Map
}

func NewSourceRepository(source *Source, providers []ResourceProvider) *SourceRepository
func (r *SourceRepository) GetResource(ctx context.Context, typ ResourceType) (Resource, error)
```

**Implementation Requirements:**
- ✅ Thread-safe resource caching
- ✅ Provider priority ordering
- ✅ Lazy resource creation
- ✅ Context propagation

**Acceptance Criteria:**
- ✅ Concurrent resource access safe
- ✅ Resources cached properly
- ✅ Provider fallback works

---

## Resource Provider System

### Requirement RP-001: ResourceProvider Interface

**Priority:** P0 (Critical)
**Component:** `core` package

**Description:**
Define resource provider abstraction.

**Interface:**
```go
type ResourceProvider interface {
    // Try to create resource for this source
    TryCreate(ctx context.Context, source *Source) (Resource, error)

    // Resource type this provider creates
    ResourceType() ResourceType

    // Provider priority (higher = preferred)
    Priority() int

    // Can this provider handle this source?
    CanProvide(source *Source) bool
}
```

**Resource Types:**
```go
const (
    SearchResourceType
    PackageMetadataResourceType
    DownloadResourceType
    PublishResourceType
    // ... more types
)
```

**Functional Requirements:**

1. **Provider selection:**
   - Sort providers by priority (descending)
   - Try providers until one succeeds
   - Cache successful provider for resource type

2. **Provider implementation:**
   - Each protocol (v2/v3) has providers
   - Providers check source compatibility
   - Return nil if cannot provide

**Acceptance Criteria:**
- ✅ Multiple providers per resource type
- ✅ Priority-based selection
- ✅ Fallback mechanism

---

### Requirement RP-002: Built-in Providers

**Priority:** P0 (Critical)
**Component:** `protocol/v3`, `protocol/v2` packages

**Description:**
Implement standard resource providers.

**v3 Providers:**
- SearchResourceV3Provider
- PackageMetadataResourceV3Provider
- DownloadResourceV3Provider
- PublishResourceV3Provider
- ServiceIndexResourceV3Provider

**v2 Providers:**
- SearchResourceV2Provider
- PackageMetadataResourceV2Provider
- DownloadResourceV2Provider
- PublishResourceV2Provider

**Functional Requirements:**

1. **Service discovery:**
   - V3: Fetch service index, cache for 40 minutes
   - V2: Known endpoint patterns

2. **Resource creation:**
   - Initialize HTTP client
   - Configure retry/timeout
   - Return resource implementation

**Acceptance Criteria:**
- ✅ All standard resources provided
- ✅ Both v2 and v3 supported
- ✅ Service index caching works

---

## Client Abstractions

### Requirement CA-001: NuGetClient

**Priority:** P0 (Critical)
**Component:** `client` package

**Description:**
Main client for NuGet operations.

**API:**
```go
type NuGetClient struct {
    sources []*SourceRepository
    cache   Cache
    logger  Logger
}

func New(options ...ClientOption) (*NuGetClient, error)

// Functional options
func WithSources(sources ...*Source) ClientOption
func WithCache(cache Cache) ClientOption
func WithLogger(logger Logger) ClientOption
```

**Core Operations:**
```go
// Search for packages
func (c *NuGetClient) Search(ctx context.Context, query string, opts ...SearchOption) ([]*PackageSearchResult, error)

// Get package metadata
func (c *NuGetClient) GetMetadata(ctx context.Context, id string, version *NuGetVersion) (*PackageMetadata, error)

// Download package
func (c *NuGetClient) Download(ctx context.Context, identity *PackageIdentity) (io.ReadCloser, error)

// List package versions
func (c *NuGetClient) ListVersions(ctx context.Context, id string) ([]*NuGetVersion, error)

// Resolve dependencies
func (c *NuGetClient) ResolveDependencies(ctx context.Context, identity *PackageIdentity, target *NuGetFramework) ([]*PackageIdentity, error)
```

**Functional Requirements:**

1. **Multi-source operations:**
   - Query all sources concurrently
   - Aggregate results
   - Deduplicate packages

2. **Caching:**
   - Cache metadata queries
   - Cache dependency resolution
   - Configurable TTL

3. **Error handling:**
   - Aggregate errors from sources
   - Partial success handling
   - Retry failed operations

**Acceptance Criteria:**
- ✅ All core operations implemented
- ✅ Multi-source aggregation works
- ✅ Caching reduces redundant queries
- ✅ Error handling graceful

---

### Requirement CA-002: Search Operations

**Priority:** P0 (Critical)
**Component:** `client` package

**Description:**
Package search functionality.

**API:**
```go
type SearchOptions struct {
    Skip           int
    Take           int
    Prerelease     bool
    IncludeDelisted bool
}

type PackageSearchResult struct {
    Identity      *PackageIdentity
    Title         string
    Description   string
    Authors       []string
    TotalDownloads int64
    Versions      []*NuGetVersion
}

func (c *NuGetClient) Search(ctx context.Context, query string, opts ...SearchOption) ([]*PackageSearchResult, error)
```

**Functional Options:**
```go
func WithSkip(n int) SearchOption
func WithTake(n int) SearchOption
func WithPrerelease(include bool) SearchOption
```

**Functional Requirements:**

1. **Query processing:**
   - Full-text search
   - Package ID prefix match
   - Tag filtering

2. **Result aggregation:**
   - Merge results from multiple sources
   - Deduplicate by package ID
   - Sort by relevance/downloads

3. **Pagination:**
   - Skip/Take support
   - Total count estimation

**Acceptance Criteria:**
- ✅ Search returns relevant packages
- ✅ Pagination works correctly
- ✅ Multi-source aggregation
- ✅ Deduplication accurate

---

### Requirement CA-003: Dependency Resolution

**Priority:** P0 (Critical)
**Component:** `resolver` package

**Description:**
Resolve package dependency trees.

**API:**
```go
type ResolutionOptions struct {
    TargetFramework *NuGetFramework
    AllowPrerelease bool
    AllowDowngrade bool
}

type ResolvedPackage struct {
    Identity *PackageIdentity
    Dependencies []*ResolvedPackage
}

func ResolveDependencies(ctx context.Context, client *NuGetClient, identity *PackageIdentity, opts *ResolutionOptions) (*ResolvedPackage, error)
```

**Functional Requirements:**

1. **Dependency tree walking:**
   - Fetch package metadata
   - Extract framework-specific dependencies
   - Recursively resolve transitive dependencies

2. **Version conflict resolution:**
   - Detect version conflicts
   - Apply resolution strategy (highest wins)
   - Optionally allow downgrades

3. **Framework compatibility:**
   - Select dependency group for target framework
   - Handle missing framework fallback

4. **Cycle detection:**
   - Detect circular dependencies
   - Report error with cycle path

**Algorithm:**
- Use breadth-first or depth-first traversal
- Cache resolved packages to avoid re-fetching
- Maintain version constraint satisfaction

**Performance Requirements:**
- Resolve 50 packages in <500ms
- Use caching aggressively
- Parallel metadata fetching

**Acceptance Criteria:**
- ✅ Correctly resolves dependency trees
- ✅ Detects version conflicts
- ✅ Handles circular dependencies
- ✅ Framework selection accurate
- ✅ Performance target met

---

## Configuration

### Requirement CFG-001: Configuration File

**Priority:** P1 (High)
**Component:** `config` package

**Description:**
Support NuGet.Config file format.

**File Format:**
```xml
<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" />
    <add key="MyFeed" value="https://example.com/nuget/v3/index.json" />
  </packageSources>
  <packageSourceCredentials>
    <MyFeed>
      <add key="Username" value="user" />
      <add key="ClearTextPassword" value="pass" />
    </MyFeed>
  </packageSourceCredentials>
</configuration>
```

**Functional Requirements:**

1. **Load configuration:**
   - Find NuGet.Config in standard locations
   - Parse XML format
   - Extract package sources and credentials

2. **Source list:**
   - Parse enabled/disabled sources
   - Preserve source order

3. **Credentials:**
   - Extract API keys
   - Extract username/password
   - Support encrypted passwords (optional)

**API:**
```go
type Config struct {
    Sources []*Source
}

func LoadConfig(path string) (*Config, error)
func LoadDefaultConfig() (*Config, error)
```

**Acceptance Criteria:**
- ✅ Parse NuGet.Config format
- ✅ Extract sources and credentials
- ✅ Handle missing config gracefully

---

## Error Handling

### Requirement ERR-001: Error Types

**Priority:** P0 (Critical)
**Component:** `errors` package

**Description:**
Define structured error types for common failures.

**Error Types:**
```go
type PackageNotFoundError struct {
    ID string
    Version *NuGetVersion
}

type VersionNotFoundError struct {
    ID string
    Version *NuGetVersion
}

type DependencyResolutionError struct {
    Package *PackageIdentity
    Cause error
}

type NetworkError struct {
    URL string
    StatusCode int
    Cause error
}

type AuthenticationError struct {
    Source string
    Cause error
}
```

**Functional Requirements:**

1. **Error wrapping:**
   - Preserve error chains
   - Support `errors.Is()` and `errors.As()`

2. **Contextual information:**
   - Include package ID, version, source
   - Preserve HTTP status codes
   - Include retry attempt number

3. **User-friendly messages:**
   - Clear error descriptions
   - Actionable suggestions

**API:**
```go
func NewPackageNotFoundError(id string, version *NuGetVersion) error
func IsPackageNotFoundError(err error) bool
```

**Acceptance Criteria:**
- ✅ All error types defined
- ✅ Error wrapping works
- ✅ Messages actionable

---

### Requirement ERR-002: Error Handling Patterns

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Consistent error handling across library.

**Requirements:**

1. **No panics:**
   - All public functions return errors
   - No panic on invalid input
   - Panic only for programmer errors (e.g., nil required parameter)

2. **Context cancellation:**
   - Check context.Done() before expensive operations
   - Return context.Err() on cancellation

3. **Error messages:**
   - Include operation context
   - Include relevant identifiers
   - Suggest remediation where possible

**Acceptance Criteria:**
- ✅ Zero panics in production code
- ✅ Context cancellation handled
- ✅ Error messages informative

---

## Acceptance Criteria

### Overall Requirements

**Code Quality:**
- ✅ 90%+ test coverage across all core packages
- ✅ Zero race conditions (race detector clean)
- ✅ No data races under concurrent load testing
- ✅ All public APIs documented with examples

**Performance:**
- ✅ Version.Parse: <50ns/op, ≤1 alloc
- ✅ Version.Compare: <20ns/op, 0 allocs
- ✅ Framework.Parse: <100ns/op, ≤1 alloc
- ✅ Framework.IsCompatible: <15ns/op, 0 allocs
- ✅ Dependency resolution (50 packages): <500ms

**Compatibility:**
- ✅ Version comparison matches C# NuGet.Versioning 100%
- ✅ Framework compatibility matches C# NuGet.Frameworks 100%
- ✅ Version range behavior matches C# 100%

**Robustness:**
- ✅ Fuzz testing for all parsers (100K iterations, no crashes)
- ✅ Handles malformed input gracefully
- ✅ No memory leaks (long-running stress test)

### Integration Requirements

**Cross-Package:**
- ✅ Client can use version, framework, packaging components seamlessly
- ✅ All packages use consistent error handling
- ✅ All packages support context cancellation
- ✅ All packages integrate with mtlog

**External Integration:**
- ✅ Works with real NuGet.org feed
- ✅ Works with Azure Artifacts
- ✅ Works with custom v2/v3 feeds

---

## Related Documents

- PRD-OVERVIEW.md - Product vision and goals
- PRD-PROTOCOL.md - Protocol implementation requirements
- PRD-PACKAGING.md - Package operations requirements
- PRD-INFRASTRUCTURE.md - HTTP, caching, observability
- PRD-TESTING.md - Testing strategy and requirements
- PRD-RELEASE.md - Release criteria and milestones

---

**END OF PRD-CORE.md**
