# gonuget Protocol Design

**Component**: `pkg/gonuget/api/`
**Version**: 1.0.0
**Status**: Draft

---

## Table of Contents

1. [Overview](#overview)
2. [NuGet v3 Protocol](#nuget-v3-protocol)
3. [NuGet v2 Protocol](#nuget-v2-protocol)
4. [Service Discovery](#service-discovery)
5. [Search Operations](#search-operations)
6. [Download Operations](#download-operations)
7. [Metadata Operations](#metadata-operations)
8. [Publish Operations](#publish-operations)
9. [Resource Provider Pattern](#resource-provider-pattern)
10. [Implementation Details](#implementation-details)

---

## Overview

gonuget must support both NuGet v2 and v3 protocols to work with all package sources:

- **NuGet v3**: Modern protocol (nuget.org, myget.org, Azure Artifacts)
- **NuGet v2**: Legacy OData protocol (older feeds, private feeds)

### Protocol Selection

```go
// Auto-detect protocol by trying v3 first, fall back to v2
source := &Source{URL: "https://api.nuget.org/v3/index.json"}
protocol := detectProtocol(source.URL)

// Explicit protocol
source := &Source{
    URL:      "https://nuget.org/api/v2",
    Protocol: ProtocolV2,
}
```

---

## NuGet v3 Protocol

### Overview

NuGet v3 is a REST API with JSON responses. It uses service discovery to find endpoints.

**Base URL**: `https://api.nuget.org/v3/index.json`

### Service Index

**File**: `pkg/gonuget/api/v3/index.go`

```go
package v3

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

// ServiceIndex represents the v3 service index
type ServiceIndex struct {
    Version   string            `json:"version"`
    Resources []ServiceResource `json:"resources"`

    // Cached endpoint lookups
    endpoints map[string][]string
    mu        sync.RWMutex
}

// ServiceResource represents a service endpoint
type ServiceResource struct {
    ID            string         `json:"@id"`
    Type          ResourceTypes  `json:"@type"` // Can be string or array
    Comment       string         `json:"comment"`
    ClientVersion string         `json:"clientVersion,omitempty"`
}

// ResourceTypes handles string or array JSON unmarshaling
type ResourceTypes []string

func (rt *ResourceTypes) UnmarshalJSON(data []byte) error {
    // Try array first
    var arr []string
    if err := json.Unmarshal(data, &arr); err == nil {
        *rt = arr
        return nil
    }

    // Fall back to string
    var str string
    if err := json.Unmarshal(data, &str); err != nil {
        return err
    }
    *rt = []string{str}
    return nil
}

// ServiceTypes defines known service type constants
const (
    ServiceTypeSearchQuery        = "SearchQueryService"
    ServiceTypeSearchQueryVersioned = "SearchQueryService/3.0.0-beta"
    ServiceTypeSearchQueryVersusioneds3 = "SearchQueryService/3.0.0-rc"
    ServiceTypeRegistration       = "RegistrationsBaseUrl"
    ServiceTypeRegistration3GZ    = "RegistrationsBaseUrl/3.0.0-rc"
    ServiceTypeRegistration3GZS2  = "RegistrationsBaseUrl/3.6.0"
    ServiceTypePackageBase        = "PackageBaseAddress/3.0.0"
    ServiceTypePackagePublish     = "PackagePublish/2.0.0"
    ServiceTypeAutocomplete       = "SearchAutocompleteService"
    ServiceTypeAutocompleteV3     = "SearchAutocompleteService/3.0.0-rc"
    ServiceTypeCatalog            = "Catalog/3.0.0"
)

// ServiceIndexClient fetches and caches the service index
type ServiceIndexClient struct {
    httpClient  *http.Client
    indexURL    string
    cache       *ServiceIndex
    cacheTTL    time.Duration
    cacheExpiry time.Time
    mu          sync.RWMutex
    logger      Logger
}

func NewServiceIndexClient(httpClient *http.Client, indexURL string, logger Logger) *ServiceIndexClient {
    return &ServiceIndexClient{
        httpClient: httpClient,
        indexURL:   indexURL,
        cacheTTL:   40 * time.Minute, // Cache for 40 minutes (C# default)
        logger:     logger,
    }
}

// GetServiceIndex fetches the service index (cached)
func (c *ServiceIndexClient) GetServiceIndex(ctx context.Context) (*ServiceIndex, error) {
    // Check cache (read lock)
    c.mu.RLock()
    if c.cache != nil && time.Now().Before(c.cacheExpiry) {
        index := c.cache
        c.mu.RUnlock()
        c.logger.Debug("Service index cache hit for {URL}", c.indexURL)
        return index, nil
    }
    c.mu.RUnlock()

    // Fetch from network (write lock)
    c.mu.Lock()
    defer c.mu.Unlock()

    // Double-check (another goroutine might have fetched)
    if c.cache != nil && time.Now().Before(c.cacheExpiry) {
        return c.cache, nil
    }

    c.logger.Info("Fetching service index from {URL}", c.indexURL)

    // Fetch index.json
    req, err := http.NewRequestWithContext(ctx, "GET", c.indexURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch service index: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("service index returned status %d", resp.StatusCode)
    }

    // Parse JSON
    var index ServiceIndex
    if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
        return nil, fmt.Errorf("failed to parse service index: %w", err)
    }

    // Build endpoint lookup map
    index.buildEndpointMap()

    // Cache
    c.cache = &index
    c.cacheExpiry = time.Now().Add(c.cacheTTL)

    return &index, nil
}

// buildEndpointMap builds a map of service type → endpoints
func (si *ServiceIndex) buildEndpointMap() {
    si.endpoints = make(map[string][]string)

    for _, resource := range si.Resources {
        for _, typ := range resource.Type {
            // Normalize type (remove version suffix for base lookup)
            baseType := typ
            si.endpoints[baseType] = append(si.endpoints[baseType], resource.ID)

            // Also add without version
            if idx := strings.Index(typ, "/"); idx > 0 {
                baseType = typ[:idx]
                si.endpoints[baseType] = append(si.endpoints[baseType], resource.ID)
            }
        }
    }
}

// GetServiceEntryURI gets the first endpoint for a service type
func (si *ServiceIndex) GetServiceEntryURI(serviceType string) (string, error) {
    si.mu.RLock()
    defer si.mu.RUnlock()

    endpoints, ok := si.endpoints[serviceType]
    if !ok || len(endpoints) == 0 {
        return "", fmt.Errorf("service type %s not found", serviceType)
    }

    return endpoints[0], nil
}

// GetServiceEntryURIs gets all endpoints for a service type
func (si *ServiceIndex) GetServiceEntryURIs(serviceType string) []string {
    si.mu.RLock()
    defer si.mu.RUnlock()

    return si.endpoints[serviceType]
}
```

---

## Search Operations

### V3 Search

**File**: `pkg/gonuget/api/v3/search.go`

```go
package v3

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strconv"
)

// SearchClient implements V3 search operations
type SearchClient struct {
    httpClient   *http.Client
    indexClient  *ServiceIndexClient
    logger       Logger
}

func NewSearchClient(httpClient *http.Client, indexClient *ServiceIndexClient, logger Logger) *SearchClient {
    return &SearchClient{
        httpClient:  httpClient,
        indexClient: indexClient,
        logger:      logger,
    }
}

// SearchOptions defines search parameters
type SearchOptions struct {
    Skip              int      // Number of results to skip
    Take              int      // Number of results to return (max 1000)
    Prerelease        bool     // Include prerelease versions
    IncludeDelisted   bool     // Include delisted packages
    SupportedFramework string   // Filter by framework (e.g., "net6.0")
    PackageType       string   // Filter by package type
    SemVerLevel       string   // SemVer level (default: "2.0.0")
}

// SearchResult represents search response
type SearchResult struct {
    TotalHits int64                  `json:"totalHits"`
    Data      []SearchResultMetadata `json:"data"`
}

// SearchResultMetadata represents a single search result
type SearchResultMetadata struct {
    ID             string                `json:"id"` // Package ID
    Version        string                `json:"version"` // Latest version
    Description    string                `json:"description"`
    Summary        string                `json:"summary"`
    Title          string                `json:"title"`
    IconURL        string                `json:"iconUrl"`
    LicenseURL     string                `json:"licenseUrl"`
    ProjectURL     string                `json:"projectUrl"`
    Tags           FlexibleStringArray   `json:"tags"`
    Authors        FlexibleStringArray   `json:"authors"`
    Owners         FlexibleStringArray   `json:"owners"`
    TotalDownloads int64                 `json:"totalDownloads"`
    Verified       bool                  `json:"verified"`
    Versions       []SearchResultVersion `json:"versions"`
}

// SearchResultVersion represents a version in search results
type SearchResultVersion struct {
    Version   string `json:"version"`
    Downloads int64  `json:"downloads"`
}

// FlexibleStringArray handles string or array JSON
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

    // Empty string → empty array
    if str == "" {
        *fsa = []string{}
        return nil
    }

    *fsa = []string{str}
    return nil
}

// Search performs a package search
func (sc *SearchClient) Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResult, error) {
    if opts == nil {
        opts = &SearchOptions{
            Take:        20,
            SemVerLevel: "2.0.0",
        }
    }

    // Get search endpoint from service index
    index, err := sc.indexClient.GetServiceIndex(ctx)
    if err != nil {
        return nil, err
    }

    searchEndpoint, err := index.GetServiceEntryURI(ServiceTypeSearchQuery)
    if err != nil {
        return nil, err
    }

    // Build query URL
    queryURL := buildSearchURL(searchEndpoint, query, opts)

    sc.logger.Debug("Searching: {URL}", queryURL)

    // Execute request
    req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := sc.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("search request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
    }

    // Parse response
    var result SearchResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to parse search results: %w", err)
    }

    sc.logger.Info("Search found {Count} results for query: {Query}", result.TotalHits, query)

    return &result, nil
}

// buildSearchURL builds the search query URL
func buildSearchURL(baseURL, query string, opts *SearchOptions) string {
    u, _ := url.Parse(baseURL)
    q := u.Query()

    q.Set("q", query)
    q.Set("skip", strconv.Itoa(opts.Skip))
    q.Set("take", strconv.Itoa(opts.Take))
    q.Set("prerelease", strconv.FormatBool(opts.Prerelease))

    // SemVer level (critical for getting latest versions)
    if opts.SemVerLevel != "" {
        q.Set("semVerLevel", opts.SemVerLevel)
    }

    if opts.IncludeDelisted {
        q.Set("includeDelisted", "true")
    }

    if opts.SupportedFramework != "" {
        q.Add("supportedFramework", opts.SupportedFramework)
    }

    if opts.PackageType != "" {
        q.Set("packageType", opts.PackageType)
    }

    u.RawQuery = q.Encode()
    return u.String()
}
```

---

## Download Operations

### V3 Package Download

**File**: `pkg/gonuget/api/v3/download.go`

```go
package v3

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strings"
)

// DownloadClient implements V3 download operations
type DownloadClient struct {
    httpClient  *http.Client
    indexClient *ServiceIndexClient
    logger      Logger
}

func NewDownloadClient(httpClient *http.Client, indexClient *ServiceIndexClient, logger Logger) *DownloadClient {
    return &DownloadClient{
        httpClient:  httpClient,
        indexClient: indexClient,
        logger:      logger,
    }
}

// ListVersions lists all versions of a package
func (dc *DownloadClient) ListVersions(ctx context.Context, packageID string) ([]string, error) {
    // Get package base address from service index
    index, err := dc.indexClient.GetServiceIndex(ctx)
    if err != nil {
        return nil, err
    }

    baseAddress, err := index.GetServiceEntryURI(ServiceTypePackageBase)
    if err != nil {
        return nil, err
    }

    // Build URL: {baseAddress}/{id-lower}/index.json
    idLower := strings.ToLower(packageID)
    indexURL := fmt.Sprintf("%s%s/index.json", baseAddress, idLower)

    dc.logger.Debug("Fetching versions from {URL}", indexURL)

    // Fetch index
    req, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := dc.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch versions: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("package %s not found", packageID)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("versions endpoint returned status %d", resp.StatusCode)
    }

    // Parse response
    var versionIndex struct {
        Versions []string `json:"versions"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&versionIndex); err != nil {
        return nil, fmt.Errorf("failed to parse versions: %w", err)
    }

    return versionIndex.Versions, nil
}

// DownloadPackage downloads a package .nupkg file
func (dc *DownloadClient) DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error) {
    // Get package base address from service index
    index, err := dc.indexClient.GetServiceIndex(ctx)
    if err != nil {
        return nil, err
    }

    baseAddress, err := index.GetServiceEntryURI(ServiceTypePackageBase)
    if err != nil {
        return nil, err
    }

    // Build download URL: {baseAddress}/{id-lower}/{version-lower}/{id-lower}.{version-lower}.nupkg
    idLower := strings.ToLower(packageID)
    versionLower := strings.ToLower(version)
    downloadURL := fmt.Sprintf("%s%s/%s/%s.%s.nupkg",
        baseAddress, idLower, versionLower, idLower, versionLower)

    dc.logger.Info("Downloading package {PackageId} {Version} from {URL}",
        packageID, version, downloadURL)

    // Download package
    req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := dc.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("download failed: %w", err)
    }

    if resp.StatusCode == http.StatusNotFound {
        resp.Body.Close()
        return nil, fmt.Errorf("package %s %s not found", packageID, version)
    }

    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
    }

    // Return body (caller must close)
    return resp.Body, nil
}
```

---

## Metadata Operations

### V3 Package Metadata

**File**: `pkg/gonuget/api/v3/metadata.go`

```go
package v3

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

// MetadataClient implements V3 metadata operations
type MetadataClient struct {
    httpClient  *http.Client
    indexClient *ServiceIndexClient
    logger      Logger
}

func NewMetadataClient(httpClient *http.Client, indexClient *ServiceIndexClient, logger Logger) *MetadataClient {
    return &MetadataClient{
        httpClient:  httpClient,
        indexClient: indexClient,
        logger:      logger,
    }
}

// GetPackageMetadata retrieves full metadata for a package
func (mc *MetadataClient) GetPackageMetadata(ctx context.Context, packageID string) (*RegistrationIndex, error) {
    // Get registration base URL from service index
    index, err := mc.indexClient.GetServiceIndex(ctx)
    if err != nil {
        return nil, err
    }

    // Try versioned registration endpoints first (compressed)
    registrationURL, err := index.GetServiceEntryURI(ServiceTypeRegistration3GZS2)
    if err != nil {
        // Fall back to base registration
        registrationURL, err = index.GetServiceEntryURI(ServiceTypeRegistration)
        if err != nil {
            return nil, err
        }
    }

    // Build URL: {registrationBase}/{id-lower}/index.json
    idLower := strings.ToLower(packageID)
    metadataURL := fmt.Sprintf("%s%s/index.json", registrationURL, idLower)

    mc.logger.Debug("Fetching metadata from {URL}", metadataURL)

    // Fetch metadata
    req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := mc.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("metadata request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("package %s not found", packageID)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("metadata endpoint returned status %d", resp.StatusCode)
    }

    // Parse registration index
    var regIndex RegistrationIndex
    if err := json.NewDecoder(resp.Body).Decode(&regIndex); err != nil {
        return nil, fmt.Errorf("failed to parse metadata: %w", err)
    }

    return &regIndex, nil
}

// RegistrationIndex represents package registration (all versions)
type RegistrationIndex struct {
    Count int               `json:"count"`
    Items []RegistrationPage `json:"items"`
}

// RegistrationPage represents a page of package versions
type RegistrationPage struct {
    ID    string              `json:"@id"`
    Count int                 `json:"count"`
    Items []RegistrationLeaf  `json:"items"`
    Lower string              `json:"lower"` // Lowest version in page
    Upper string              `json:"upper"` // Highest version in page
}

// RegistrationLeaf represents a single package version
type RegistrationLeaf struct {
    ID             string              `json:"@id"`
    CatalogEntry   CatalogEntry        `json:"catalogEntry"`
    PackageContent string              `json:"packageContent"` // Download URL
}

// CatalogEntry contains package metadata for a specific version
type CatalogEntry struct {
    ID                 string                 `json:"@id"`
    PackageID          string                 `json:"id"`
    Version            string                 `json:"version"`
    Description        string                 `json:"description"`
    Summary            string                 `json:"summary"`
    Title              string                 `json:"title"`
    Authors            FlexibleStringArray    `json:"authors"`
    Owners             FlexibleStringArray    `json:"owners"`
    IconURL            string                 `json:"iconUrl"`
    LicenseURL         string                 `json:"licenseUrl"`
    ProjectURL         string                 `json:"projectUrl"`
    Tags               FlexibleStringArray    `json:"tags"`
    DependencyGroups   []DependencyGroup      `json:"dependencyGroups"`
    Published          string                 `json:"published"` // ISO 8601
    RequireLicenseAcceptance bool             `json:"requireLicenseAcceptance"`
    Listed             bool                   `json:"listed"`
}

// DependencyGroup represents dependencies for a target framework
type DependencyGroup struct {
    TargetFramework string       `json:"targetFramework"`
    Dependencies    []Dependency `json:"dependencies"`
}

// Dependency represents a package dependency
type Dependency struct {
    ID    string `json:"id"`
    Range string `json:"range"` // Version range
}
```

---

## Publish Operations

### V3 Package Push

**File**: `pkg/gonuget/api/v3/publish.go`

```go
package v3

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
)

// PublishClient implements V3 publish operations
type PublishClient struct {
    httpClient  *http.Client
    indexClient *ServiceIndexClient
    apiKey      string
    logger      Logger
}

func NewPublishClient(httpClient *http.Client, indexClient *ServiceIndexClient, apiKey string, logger Logger) *PublishClient {
    return &PublishClient{
        httpClient:  httpClient,
        indexClient: indexClient,
        apiKey:      apiKey,
        logger:      logger,
    }
}

// PushPackage uploads a package to the feed
func (pc *PublishClient) PushPackage(ctx context.Context, nupkgPath string) error {
    // Get publish endpoint from service index
    index, err := pc.indexClient.GetServiceIndex(ctx)
    if err != nil {
        return err
    }

    publishEndpoint, err := index.GetServiceEntryURI(ServiceTypePackagePublish)
    if err != nil {
        return err
    }

    pc.logger.Info("Pushing package {Path} to {Endpoint}", nupkgPath, publishEndpoint)

    // Open package file
    file, err := os.Open(nupkgPath)
    if err != nil {
        return fmt.Errorf("failed to open package: %w", err)
    }
    defer file.Close()

    // Create multipart form
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)

    part, err := writer.CreateFormFile("package", filepath.Base(nupkgPath))
    if err != nil {
        return err
    }

    if _, err := io.Copy(part, file); err != nil {
        return err
    }

    if err := writer.Close(); err != nil {
        return err
    }

    // Create request
    req, err := http.NewRequestWithContext(ctx, "PUT", publishEndpoint, &body)
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", writer.FormDataContentType())
    req.Header.Set("X-NuGet-ApiKey", pc.apiKey)

    // Execute request
    resp, err := pc.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("push request failed: %w", err)
    }
    defer resp.Body.Close()

    // Check response
    if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
        pc.logger.Info("Package pushed successfully")
        return nil
    }

    if resp.StatusCode == http.StatusConflict {
        return fmt.Errorf("package already exists")
    }

    if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
        return fmt.Errorf("authentication failed (invalid API key)")
    }

    return fmt.Errorf("push failed with status %d", resp.StatusCode)
}
```

---

## NuGet v2 Protocol

### OData Protocol

**File**: `pkg/gonuget/api/v2/odata.go`

```go
package v2

import (
    "context"
    "encoding/xml"
    "fmt"
    "net/http"
    "net/url"
)

// V2Client implements NuGet v2 (OData) protocol
type V2Client struct {
    httpClient *http.Client
    baseURL    string
    logger     Logger
}

func NewV2Client(httpClient *http.Client, baseURL string, logger Logger) *V2Client {
    return &V2Client{
        httpClient: httpClient,
        baseURL:    baseURL,
        logger:     logger,
    }
}

// Search searches for packages using OData
func (c *V2Client) Search(ctx context.Context, query string, opts *SearchOptions) (*V2SearchResult, error) {
    // Build OData query
    // URL: /Packages()?$filter=substringof('query',Id)&$orderby=DownloadCount desc&$skip=0&$top=20

    u, _ := url.Parse(c.baseURL)
    u.Path += "/Packages()"

    q := u.Query()

    // Search filter
    if query != "" {
        filter := fmt.Sprintf("substringof('%s',tolower(Id)) or substringof('%s',tolower(Description))",
            query, query)
        q.Set("$filter", filter)
    }

    // Pagination
    q.Set("$skip", strconv.Itoa(opts.Skip))
    q.Set("$top", strconv.Itoa(opts.Take))

    // Ordering
    q.Set("$orderby", "DownloadCount desc")

    u.RawQuery = q.Encode()

    c.logger.Debug("V2 search: {URL}", u.String())

    // Execute request
    req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
    }

    // Parse OData XML response
    var result V2SearchResult
    if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

// V2SearchResult represents OData search response
type V2SearchResult struct {
    XMLName xml.Name    `xml:"feed"`
    Entries []V2Package `xml:"entry"`
}

// V2Package represents a package in V2 format
type V2Package struct {
    XMLName    xml.Name      `xml:"entry"`
    ID         string        `xml:"id"`
    Title      string        `xml:"title"`
    Properties V2Properties  `xml:"properties"`
}

// V2Properties contains package properties
type V2Properties struct {
    ID              string `xml:"Id"`
    Version         string `xml:"Version"`
    Description     string `xml:"Description"`
    Authors         string `xml:"Authors"`
    DownloadCount   int64  `xml:"DownloadCount"`
    IconUrl         string `xml:"IconUrl"`
    LicenseUrl      string `xml:"LicenseUrl"`
    ProjectUrl      string `xml:"ProjectUrl"`
    Tags            string `xml:"Tags"`
}

// DownloadPackage downloads a package using V2 API
func (c *V2Client) DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error) {
    // V2 download URL: /package/{id}/{version}
    downloadURL := fmt.Sprintf("%s/package/%s/%s", c.baseURL, packageID, version)

    c.logger.Info("Downloading package from V2: {URL}", downloadURL)

    req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
    }

    return resp.Body, nil
}
```

---

## Resource Provider Pattern

### Resource Interface

**File**: `internal/resource/provider.go`

```go
package resource

import "context"

// ResourceType identifies a resource capability
type ResourceType string

const (
    TypeSearch       ResourceType = "search"
    TypeDownload     ResourceType = "download"
    TypeMetadata     ResourceType = "metadata"
    TypePublish      ResourceType = "publish"
    TypeAutocomplete ResourceType = "autocomplete"
)

// Resource represents a capability
type Resource interface {
    Type() ResourceType
}

// Provider creates resources for a source
type Provider interface {
    TryCreate(ctx context.Context, source *Source) (Resource, error)
    ResourceType() ResourceType
    Priority() int
}

// Registry manages resource providers
type Registry struct {
    providers []Provider
}

func NewRegistry() *Registry {
    return &Registry{
        providers: make([]Provider, 0),
    }
}

func (r *Registry) RegisterProvider(p Provider) {
    r.providers = append(r.providers, p)

    // Sort by priority (highest first)
    sort.Slice(r.providers, func(i, j int) bool {
        return r.providers[i].Priority() > r.providers[j].Priority()
    })
}

func (r *Registry) GetResource(ctx context.Context, source *Source, typ ResourceType) (Resource, error) {
    for _, provider := range r.providers {
        if provider.ResourceType() != typ {
            continue
        }

        resource, err := provider.TryCreate(ctx, source)
        if err == nil {
            return resource, nil
        }
    }

    return nil, fmt.Errorf("no provider for resource type %s", typ)
}
```

### V3 Search Provider

```go
package resource

type V3SearchProvider struct {
    httpClient *http.Client
    logger     Logger
}

func (p *V3SearchProvider) TryCreate(ctx context.Context, source *Source) (Resource, error) {
    // Check if source is V3
    if source.Protocol != ProtocolV3 {
        return nil, fmt.Errorf("not a V3 source")
    }

    indexClient := v3.NewServiceIndexClient(p.httpClient, source.URL, p.logger)

    return v3.NewSearchClient(p.httpClient, indexClient, p.logger), nil
}

func (p *V3SearchProvider) ResourceType() ResourceType {
    return TypeSearch
}

func (p *V3SearchProvider) Priority() int {
    return 100 // Higher priority than V2
}
```

---

## Implementation Details

### Package Dependencies

```go
require (
    // Standard library only for core protocol
    "encoding/json"
    "encoding/xml"
    "net/http"
    "net/url"
)
```

### Error Handling

```go
// Protocol-specific errors
type ProtocolError struct {
    Operation  string
    StatusCode int
    Message    string
}

func (e *ProtocolError) Error() string {
    return fmt.Sprintf("%s failed (HTTP %d): %s", e.Operation, e.StatusCode, e.Message)
}

// Package not found
type PackageNotFoundError struct {
    PackageID string
    Version   string
}

func (e *PackageNotFoundError) Error() string {
    return fmt.Sprintf("package %s %s not found", e.PackageID, e.Version)
}
```

### Performance Optimizations

1. **Service index caching**: Cache for 40 minutes
2. **Connection reuse**: HTTP keep-alive
3. **Streaming downloads**: Don't buffer entire package in memory
4. **Concurrent requests**: Parallel metadata fetching

---

**Document Status**: Draft v1.0
**Last Updated**: 2025-01-19
**Next Review**: After implementation
