# Implementation Guide: Milestone 2 - Protocol Implementation (Continued Part 2)

**Milestone:** M2 - Protocol Implementation (Final Section)
**Chunks:** M2.9 - M2.18
**Previous Files:**
- IMPL-M2-PROTOCOL.md (chunks M2.1-M2.5)
- IMPL-M2-PROTOCOL-CONTINUED.md (chunks M2.6-M2.8)

---

## [M2.9] Protocol v2 - Feed Detection

**Time Estimate:** 1 hour
**Dependencies:** M2.1 (HTTP client)
**Status:** Not started

### What You'll Build

Implement NuGet v2 OData feed detection. V2 feeds use OData/Atom XML format and need to be detected by checking for the OData service document.

### Step-by-Step Instructions

**Step 1: Create v2 protocol types**

Create `protocol/v2/types.go`:

```go
package v2

import (
	"encoding/xml"
)

// Service represents the OData service document
type Service struct {
	XMLName    xml.Name    `xml:"service"`
	Workspace  Workspace   `xml:"workspace"`
	Base       string      `xml:"base,attr"`
}

// Workspace contains collections in the OData service
type Workspace struct {
	Title       string       `xml:"title"`
	Collections []Collection `xml:"collection"`
}

// Collection represents an OData collection
type Collection struct {
	Href  string `xml:"href,attr"`
	Title string `xml:"title"`
}

// Feed represents an Atom feed response
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Entries []Entry  `xml:"entry"`
}

// Entry represents a single entry in an Atom feed
type Entry struct {
	XMLName    xml.Name   `xml:"entry"`
	ID         string     `xml:"id"`
	Title      string     `xml:"title"`
	Updated    string     `xml:"updated"`
	Properties Properties `xml:"properties"`
	Content    Content    `xml:"content"`
}

// Properties contains package metadata
type Properties struct {
	XMLName              xml.Name `xml:"properties"`
	ID                   string   `xml:"Id"`
	Version              string   `xml:"Version"`
	Description          string   `xml:"Description"`
	Authors              string   `xml:"Authors"`
	IconURL              string   `xml:"IconUrl"`
	LicenseURL           string   `xml:"LicenseUrl"`
	ProjectURL           string   `xml:"ProjectUrl"`
	Tags                 string   `xml:"Tags"`
	Dependencies         string   `xml:"Dependencies"`
	DownloadCount        int64    `xml:"DownloadCount"`
	IsPrerelease         bool     `xml:"IsPrerelease"`
	Published            string   `xml:"Published"`
	RequireLicenseAcceptance bool `xml:"RequireLicenseAcceptance"`
}

// Content contains the package download URL
type Content struct {
	Type string `xml:"type,attr"`
	Src  string `xml:"src,attr"`
}
```

**Step 2: Create feed detection client**

Create `protocol/v2/feed.go`:

```go
package v2

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// FeedClient provides v2 feed detection and access
type FeedClient struct {
	httpClient *nugethttp.Client
}

// NewFeedClient creates a new v2 feed client
func NewFeedClient(httpClient *nugethttp.Client) *FeedClient {
	return &FeedClient{
		httpClient: httpClient,
	}
}

// DetectV2Feed checks if a URL is a valid NuGet v2 feed
// Returns true if the feed is detected, false otherwise
func (c *FeedClient) DetectV2Feed(ctx context.Context, feedURL string) (bool, error) {
	// Try to fetch the service document
	// V2 feeds typically have a service document at the base URL
	serviceURL := feedURL
	if !strings.HasSuffix(serviceURL, "/") {
		serviceURL += "/"
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return false, fmt.Errorf("fetch service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "xml") && !strings.Contains(contentType, "atom") {
		return false, nil
	}

	// Try to parse as service document
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	var service Service
	if err := xml.Unmarshal(body, &service); err != nil {
		// Not a valid service document
		return false, nil
	}

	// Check if it has the expected collections
	for _, collection := range service.Workspace.Collections {
		if strings.Contains(strings.ToLower(collection.Href), "packages") {
			return true, nil
		}
	}

	return false, nil
}

// GetServiceDocument retrieves the OData service document
func (c *FeedClient) GetServiceDocument(ctx context.Context, feedURL string) (*Service, error) {
	serviceURL := feedURL
	if !strings.HasSuffix(serviceURL, "/") {
		serviceURL += "/"
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("service returned %d: %s", resp.StatusCode, body)
	}

	var service Service
	if err := xml.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}

	return &service, nil
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v2

# Format check
gofmt -l protocol/v2/
```

### Testing

Create `protocol/v2/feed_test.go`:

```go
package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const testServiceDocument = `<?xml version="1.0" encoding="utf-8"?>
<service xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2007/app" xmlns:atom="http://www.w3.org/2005/Atom">
  <workspace>
    <atom:title type="text">Default</atom:title>
    <collection href="Packages">
      <atom:title type="text">Packages</atom:title>
    </collection>
  </workspace>
</service>`

func setupV2Server(t *testing.T) (*httptest.Server, *FeedClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.Write([]byte(testServiceDocument))
			return
		}
		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	feedClient := NewFeedClient(httpClient)

	return server, feedClient
}

func TestFeedClient_DetectV2Feed(t *testing.T) {
	server, client := setupV2Server(t)
	defer server.Close()

	ctx := context.Background()

	detected, err := client.DetectV2Feed(ctx, server.URL)
	if err != nil {
		t.Fatalf("DetectV2Feed() error = %v", err)
	}

	if !detected {
		t.Error("DetectV2Feed() = false, want true")
	}
}

func TestFeedClient_DetectV2Feed_NotV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"3.0.0"}`))
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewFeedClient(httpClient)

	ctx := context.Background()

	detected, err := client.DetectV2Feed(ctx, server.URL)
	if err != nil {
		t.Fatalf("DetectV2Feed() error = %v", err)
	}

	if detected {
		t.Error("DetectV2Feed() = true, want false for non-v2 feed")
	}
}

func TestFeedClient_GetServiceDocument(t *testing.T) {
	server, client := setupV2Server(t)
	defer server.Close()

	ctx := context.Background()

	service, err := client.GetServiceDocument(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceDocument() error = %v", err)
	}

	if service.Workspace.Title != "Default" {
		t.Errorf("Workspace.Title = %q, want Default", service.Workspace.Title)
	}

	if len(service.Workspace.Collections) != 1 {
		t.Fatalf("len(Collections) = %d, want 1", len(service.Workspace.Collections))
	}

	collection := service.Workspace.Collections[0]
	if collection.Href != "Packages" {
		t.Errorf("Collection.Href = %q, want Packages", collection.Href)
	}

	if collection.Title != "Packages" {
		t.Errorf("Collection.Title = %q, want Packages", collection.Title)
	}
}
```

Run tests:

```bash
go test ./protocol/v2 -v
```

### Commit

```
feat: implement NuGet v2 feed detection

- Add OData service document types (Service, Workspace, Collection)
- Add Atom feed types (Feed, Entry, Properties)
- Implement v2 feed detection via service document
- Parse XML service document with collections
- Create comprehensive feed detection tests

Chunk: M2.9
Status: ✓ Complete
```

---

## [M2.10] Protocol v2 - Search

**Time Estimate:** 3 hours
**Dependencies:** M2.9 (V2 feed detection)
**Status:** Not started

### What You'll Build

Implement NuGet v2 package search using OData query syntax.

### Step-by-Step Instructions

**Step 1: Create v2 search client**

Create `protocol/v2/search.go`:

```go
package v2

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// SearchClient provides v2 search functionality
type SearchClient struct {
	httpClient *nugethttp.Client
}

// SearchOptions holds v2 search parameters
type SearchOptions struct {
	Query          string
	Skip           int
	Top            int
	Filter         string
	OrderBy        string
	IncludePrerelease bool
}

// SearchResult represents a v2 search result
type SearchResult struct {
	ID                   string
	Version              string
	Description          string
	Authors              string
	IconURL              string
	LicenseURL           string
	ProjectURL           string
	Tags                 []string
	Dependencies         string
	DownloadCount        int64
	IsPrerelease         bool
	Published            string
	RequireLicenseAcceptance bool
	DownloadURL          string
}

// NewSearchClient creates a new v2 search client
func NewSearchClient(httpClient *nugethttp.Client) *SearchClient {
	return &SearchClient{
		httpClient: httpClient,
	}
}

// Search searches for packages using OData query syntax
func (c *SearchClient) Search(ctx context.Context, feedURL string, opts SearchOptions) ([]SearchResult, error) {
	// Build search URL with OData parameters
	searchURL, err := c.buildSearchURL(feedURL, opts)
	if err != nil {
		return nil, fmt.Errorf("build search URL: %w", err)
	}

	// Execute search request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, body)
	}

	// Parse Atom feed response
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode feed: %w", err)
	}

	// Convert entries to search results
	results := make([]SearchResult, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		result := SearchResult{
			ID:                   entry.Properties.ID,
			Version:              entry.Properties.Version,
			Description:          entry.Properties.Description,
			Authors:              entry.Properties.Authors,
			IconURL:              entry.Properties.IconURL,
			LicenseURL:           entry.Properties.LicenseURL,
			ProjectURL:           entry.Properties.ProjectURL,
			Dependencies:         entry.Properties.Dependencies,
			DownloadCount:        entry.Properties.DownloadCount,
			IsPrerelease:         entry.Properties.IsPrerelease,
			Published:            entry.Properties.Published,
			RequireLicenseAcceptance: entry.Properties.RequireLicenseAcceptance,
			DownloadURL:          entry.Content.Src,
		}

		// Parse tags
		if entry.Properties.Tags != "" {
			result.Tags = strings.Split(entry.Properties.Tags, " ")
		}

		results = append(results, result)
	}

	return results, nil
}

func (c *SearchClient) buildSearchURL(feedURL string, opts SearchOptions) (string, error) {
	// Ensure feedURL ends with /
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 search endpoint: /Packages() or /Search()
	// Using /Packages() with $filter for compatibility
	searchURL := baseURL + "Packages()"

	// Build OData query parameters
	params := url.Values{}

	// $filter - combine query and prerelease filter
	var filters []string
	if opts.Query != "" {
		// Search in Id and Description
		filter := fmt.Sprintf("(substringof('%s',tolower(Id)) or substringof('%s',tolower(Description)))",
			strings.ToLower(opts.Query),
			strings.ToLower(opts.Query))
		filters = append(filters, filter)
	}

	if !opts.IncludePrerelease {
		filters = append(filters, "IsPrerelease eq false")
	}

	if opts.Filter != "" {
		filters = append(filters, opts.Filter)
	}

	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	// $orderby
	if opts.OrderBy != "" {
		params.Set("$orderby", opts.OrderBy)
	} else {
		params.Set("$orderby", "DownloadCount desc")
	}

	// $skip
	if opts.Skip > 0 {
		params.Set("$skip", strconv.Itoa(opts.Skip))
	}

	// $top
	if opts.Top > 0 {
		params.Set("$top", strconv.Itoa(opts.Top))
	} else {
		params.Set("$top", "20") // Default
	}

	// Build final URL
	if len(params) > 0 {
		searchURL += "?" + params.Encode()
	}

	return searchURL, nil
}

// FindPackagesById searches for all versions of a specific package ID
func (c *SearchClient) FindPackagesById(ctx context.Context, feedURL, packageID string) ([]SearchResult, error) {
	return c.Search(ctx, feedURL, SearchOptions{
		Filter:            fmt.Sprintf("Id eq '%s'", packageID),
		OrderBy:           "Version desc",
		Top:               100,
		IncludePrerelease: true,
	})
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v2

# Format check
gofmt -l protocol/v2/
```

### Testing

Create `protocol/v2/search_test.go`:

```go
package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const testFeedResponse = `<?xml version="1.0" encoding="utf-8"?>
<feed xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <title type="text">Packages</title>
  <id>http://schemas.datacontract.org/2004/07/</id>
  <updated>2023-01-01T00:00:00Z</updated>
  <entry>
    <id>https://www.nuget.org/api/v2/Packages(Id='Newtonsoft.Json',Version='13.0.3')</id>
    <title type="text">Newtonsoft.Json</title>
    <updated>2023-03-08T18:36:53Z</updated>
    <content type="application/zip" src="https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.3" />
    <m:properties>
      <d:Id>Newtonsoft.Json</d:Id>
      <d:Version>13.0.3</d:Version>
      <d:Description>Json.NET is a popular high-performance JSON framework for .NET</d:Description>
      <d:Authors>James Newton-King</d:Authors>
      <d:IconUrl>https://www.newtonsoft.com/content/images/nugeticon.png</d:IconUrl>
      <d:LicenseUrl>https://licenses.nuget.org/MIT</d:LicenseUrl>
      <d:ProjectUrl>https://www.newtonsoft.com/json</d:ProjectUrl>
      <d:Tags>json serialization</d:Tags>
      <d:Dependencies></d:Dependencies>
      <d:DownloadCount m:type="Edm.Int64">1000000000</d:DownloadCount>
      <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
      <d:Published>2023-03-08T18:36:53.147Z</d:Published>
      <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
    </m:properties>
  </entry>
</feed>`

func setupV2SearchServer(t *testing.T) (*httptest.Server, *SearchClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "Packages") {
			// Validate query parameters
			query := r.URL.Query()
			if query.Has("$filter") {
				filter := query.Get("$filter")
				// Basic validation
				if !strings.Contains(filter, "substringof") && !strings.Contains(filter, "Id eq") {
					t.Logf("Unexpected filter: %s", filter)
				}
			}

			w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
			w.Write([]byte(testFeedResponse))
			return
		}
		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	searchClient := NewSearchClient(httpClient)

	return server, searchClient
}

func TestSearchClient_Search(t *testing.T) {
	server, client := setupV2SearchServer(t)
	defer server.Close()

	ctx := context.Background()

	results, err := client.Search(ctx, server.URL, SearchOptions{
		Query:             "newtonsoft",
		Top:               20,
		IncludePrerelease: true,
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	result := results[0]

	if result.ID != "Newtonsoft.Json" {
		t.Errorf("ID = %q, want Newtonsoft.Json", result.ID)
	}

	if result.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", result.Version)
	}

	if result.Authors != "James Newton-King" {
		t.Errorf("Authors = %q, want James Newton-King", result.Authors)
	}

	if result.DownloadCount != 1000000000 {
		t.Errorf("DownloadCount = %d, want 1000000000", result.DownloadCount)
	}

	if result.IsPrerelease {
		t.Error("IsPrerelease = true, want false")
	}

	if len(result.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(result.Tags))
	}

	expectedTags := []string{"json", "serialization"}
	for i, want := range expectedTags {
		if result.Tags[i] != want {
			t.Errorf("Tags[%d] = %q, want %q", i, result.Tags[i], want)
		}
	}
}

func TestSearchClient_FindPackagesById(t *testing.T) {
	server, client := setupV2SearchServer(t)
	defer server.Close()

	ctx := context.Background()

	results, err := client.FindPackagesById(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("FindPackagesById() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	if results[0].ID != "Newtonsoft.Json" {
		t.Errorf("ID = %q, want Newtonsoft.Json", results[0].ID)
	}
}

func TestSearchClient_BuildSearchURL(t *testing.T) {
	client := NewSearchClient(nil)

	tests := []struct {
		name     string
		feedURL  string
		opts     SearchOptions
		wantURL  string
		wantQuery map[string]string
	}{
		{
			name:    "basic search",
			feedURL: "https://api.nuget.org/v2/",
			opts: SearchOptions{
				Query: "newtonsoft",
				Top:   20,
			},
			wantURL: "https://api.nuget.org/v2/Packages()",
			wantQuery: map[string]string{
				"$top":     "20",
				"$orderby": "DownloadCount desc",
			},
		},
		{
			name:    "with skip",
			feedURL: "https://api.nuget.org/v2/",
			opts: SearchOptions{
				Query: "json",
				Skip:  10,
				Top:   5,
			},
			wantQuery: map[string]string{
				"$skip": "10",
				"$top":  "5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildSearchURL(tt.feedURL, tt.opts)
			if err != nil {
				t.Fatalf("buildSearchURL() error = %v", err)
			}

			if !strings.HasPrefix(got, tt.wantURL) {
				t.Errorf("URL = %q, want prefix %q", got, tt.wantURL)
			}

			// Check query parameters
			for key, wantValue := range tt.wantQuery {
				if !strings.Contains(got, key+"="+wantValue) {
					t.Errorf("URL missing %s=%s", key, wantValue)
				}
			}
		})
	}
}
```

Run tests:

```bash
go test ./protocol/v2 -v -run TestSearch
```

### Commit

```
feat: implement NuGet v2 package search

- Add SearchClient with OData query support
- Build search URLs with $filter, $orderby, $skip, $top
- Parse Atom feed XML responses
- Convert entries to SearchResult structs
- Add FindPackagesById helper
- Support prerelease filtering
- Create comprehensive search tests

Chunk: M2.10
Status: ✓ Complete
```

---

## [M2.11] Protocol v2 - Metadata

**Time Estimate:** 3 hours
**Dependencies:** M2.9 (V2 feed detection)
**Status:** Not started

### What You'll Build

Implement NuGet v2 package metadata retrieval using OData endpoints.

### Step-by-Step Instructions

**Step 1: Create v2 metadata client**

Create `protocol/v2/metadata.go`:

```go
package v2

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// MetadataClient provides v2 metadata functionality
type MetadataClient struct {
	httpClient *nugethttp.Client
}

// PackageMetadata represents complete package metadata
type PackageMetadata struct {
	ID                       string
	Version                  string
	Title                    string
	Description              string
	Summary                  string
	Authors                  string
	Owners                   string
	IconURL                  string
	LicenseURL               string
	ProjectURL               string
	Tags                     []string
	Dependencies             string
	DownloadCount            int64
	IsPrerelease             bool
	Published                string
	RequireLicenseAcceptance bool
	MinClientVersion         string
	ReleaseNotes             string
	Copyright                string
	Language                 string
	PackageHash              string
	PackageHashAlgorithm     string
	PackageSize              int64
	DownloadURL              string
}

// NewMetadataClient creates a new v2 metadata client
func NewMetadataClient(httpClient *nugethttp.Client) *MetadataClient {
	return &MetadataClient{
		httpClient: httpClient,
	}
}

// GetPackageMetadata retrieves metadata for a specific package version
func (c *MetadataClient) GetPackageMetadata(ctx context.Context, feedURL, packageID, version string) (*PackageMetadata, error) {
	// Build metadata URL
	// Format: /Packages(Id='packageId',Version='version')
	metadataURL, err := c.buildMetadataURL(feedURL, packageID, version)
	if err != nil {
		return nil, fmt.Errorf("build metadata URL: %w", err)
	}

	// Execute request
	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("metadata request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s %s not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("metadata returned %d: %s", resp.StatusCode, body)
	}

	// Parse entry response
	var entry Entry
	if err := xml.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode entry: %w", err)
	}

	// Convert to PackageMetadata
	metadata := &PackageMetadata{
		ID:                       entry.Properties.ID,
		Version:                  entry.Properties.Version,
		Description:              entry.Properties.Description,
		Authors:                  entry.Properties.Authors,
		IconURL:                  entry.Properties.IconURL,
		LicenseURL:               entry.Properties.LicenseURL,
		ProjectURL:               entry.Properties.ProjectURL,
		Dependencies:             entry.Properties.Dependencies,
		DownloadCount:            entry.Properties.DownloadCount,
		IsPrerelease:             entry.Properties.IsPrerelease,
		Published:                entry.Properties.Published,
		RequireLicenseAcceptance: entry.Properties.RequireLicenseAcceptance,
		DownloadURL:              entry.Content.Src,
	}

	// Parse tags
	if entry.Properties.Tags != "" {
		metadata.Tags = strings.Split(entry.Properties.Tags, " ")
	}

	return metadata, nil
}

// ListVersions lists all versions of a package
func (c *MetadataClient) ListVersions(ctx context.Context, feedURL, packageID string) ([]string, error) {
	// Use FindPackagesByID endpoint
	// Format: /FindPackagesById()?id='packageId'
	listURL, err := c.buildListVersionsURL(feedURL, packageID)
	if err != nil {
		return nil, fmt.Errorf("build list URL: %w", err)
	}

	// Execute request
	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list versions request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("list versions returned %d: %s", resp.StatusCode, body)
	}

	// Parse feed response
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode feed: %w", err)
	}

	// Extract versions
	versions := make([]string, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		versions = append(versions, entry.Properties.Version)
	}

	return versions, nil
}

func (c *MetadataClient) buildMetadataURL(feedURL, packageID, version string) (string, error) {
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// URL encode the ID and version
	encodedID := url.QueryEscape(packageID)
	encodedVersion := url.QueryEscape(version)

	metadataURL := fmt.Sprintf("%sPackages(Id='%s',Version='%s')",
		baseURL, encodedID, encodedVersion)

	return metadataURL, nil
}

func (c *MetadataClient) buildListVersionsURL(feedURL, packageID string) (string, error) {
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Build FindPackagesById URL
	listURL := baseURL + "FindPackagesById()"

	// Add id parameter
	params := url.Values{}
	params.Set("id", packageID)

	return listURL + "?" + params.Encode(), nil
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v2

# Format check
gofmt -l protocol/v2/
```

### Testing

Create `protocol/v2/metadata_test.go`:

```go
package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const testEntryResponse = `<?xml version="1.0" encoding="utf-8"?>
<entry xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <id>https://www.nuget.org/api/v2/Packages(Id='Newtonsoft.Json',Version='13.0.3')</id>
  <title type="text">Newtonsoft.Json</title>
  <updated>2023-03-08T18:36:53Z</updated>
  <content type="application/zip" src="https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.3" />
  <m:properties>
    <d:Id>Newtonsoft.Json</d:Id>
    <d:Version>13.0.3</d:Version>
    <d:Description>Json.NET is a popular high-performance JSON framework for .NET</d:Description>
    <d:Authors>James Newton-King</d:Authors>
    <d:IconUrl>https://www.newtonsoft.com/content/images/nugeticon.png</d:IconUrl>
    <d:LicenseUrl>https://licenses.nuget.org/MIT</d:LicenseUrl>
    <d:ProjectUrl>https://www.newtonsoft.com/json</d:ProjectUrl>
    <d:Tags>json serialization</d:Tags>
    <d:Dependencies></d:Dependencies>
    <d:DownloadCount m:type="Edm.Int64">1000000000</d:DownloadCount>
    <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
    <d:Published>2023-03-08T18:36:53.147Z</d:Published>
    <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
  </m:properties>
</entry>`

const testVersionsResponse = `<?xml version="1.0" encoding="utf-8"?>
<feed xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <title type="text">Packages</title>
  <id>http://schemas.datacontract.org/2004/07/</id>
  <updated>2023-01-01T00:00:00Z</updated>
  <entry>
    <m:properties>
      <d:Version>13.0.1</d:Version>
    </m:properties>
  </entry>
  <entry>
    <m:properties>
      <d:Version>13.0.2</d:Version>
    </m:properties>
  </entry>
  <entry>
    <m:properties>
      <d:Version>13.0.3</d:Version>
    </m:properties>
  </entry>
</feed>`

func setupV2MetadataServer(t *testing.T) (*httptest.Server, *MetadataClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle specific package metadata
		if strings.Contains(path, "Packages(Id=") {
			if strings.Contains(path, "Newtonsoft.Json") && strings.Contains(path, "13.0.3") {
				w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
				w.Write([]byte(testEntryResponse))
				return
			}
			http.NotFound(w, r)
			return
		}

		// Handle FindPackagesById
		if strings.Contains(path, "FindPackagesById") {
			query := r.URL.Query()
			if query.Get("id") == "Newtonsoft.Json" {
				w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
				w.Write([]byte(testVersionsResponse))
				return
			}
			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	metadataClient := NewMetadataClient(httpClient)

	return server, metadataClient
}

func TestMetadataClient_GetPackageMetadata(t *testing.T) {
	server, client := setupV2MetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	metadata, err := client.GetPackageMetadata(ctx, server.URL, "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	if metadata.ID != "Newtonsoft.Json" {
		t.Errorf("ID = %q, want Newtonsoft.Json", metadata.ID)
	}

	if metadata.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", metadata.Version)
	}

	if metadata.Authors != "James Newton-King" {
		t.Errorf("Authors = %q, want James Newton-King", metadata.Authors)
	}

	if metadata.DownloadCount != 1000000000 {
		t.Errorf("DownloadCount = %d, want 1000000000", metadata.DownloadCount)
	}

	if metadata.IsPrerelease {
		t.Error("IsPrerelease = true, want false")
	}

	if len(metadata.Tags) != 2 {
		t.Fatalf("len(Tags) = %d, want 2", len(metadata.Tags))
	}

	expectedTags := []string{"json", "serialization"}
	for i, want := range expectedTags {
		if metadata.Tags[i] != want {
			t.Errorf("Tags[%d] = %q, want %q", i, metadata.Tags[i], want)
		}
	}

	expectedURL := "https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.3"
	if metadata.DownloadURL != expectedURL {
		t.Errorf("DownloadURL = %q, want %q", metadata.DownloadURL, expectedURL)
	}
}

func TestMetadataClient_GetPackageMetadata_NotFound(t *testing.T) {
	server, client := setupV2MetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.GetPackageMetadata(ctx, server.URL, "NonExistent.Package", "1.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestMetadataClient_ListVersions(t *testing.T) {
	server, client := setupV2MetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	versions, err := client.ListVersions(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	expected := []string{"13.0.1", "13.0.2", "13.0.3"}
	if len(versions) != len(expected) {
		t.Fatalf("len(versions) = %d, want %d", len(versions), len(expected))
	}

	for i, want := range expected {
		if versions[i] != want {
			t.Errorf("versions[%d] = %q, want %q", i, versions[i], want)
		}
	}
}

func TestMetadataClient_BuildMetadataURL(t *testing.T) {
	client := NewMetadataClient(nil)

	tests := []struct {
		name      string
		feedURL   string
		packageID string
		version   string
		want      string
	}{
		{
			name:      "basic",
			feedURL:   "https://api.nuget.org/v2/",
			packageID: "Newtonsoft.Json",
			version:   "13.0.3",
			want:      "https://api.nuget.org/v2/Packages(Id='Newtonsoft.Json',Version='13.0.3')",
		},
		{
			name:      "with spaces",
			feedURL:   "https://api.nuget.org/v2/",
			packageID: "My Package",
			version:   "1.0.0",
			want:      "https://api.nuget.org/v2/Packages(Id='My+Package',Version='1.0.0')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildMetadataURL(tt.feedURL, tt.packageID, tt.version)
			if err != nil {
				t.Fatalf("buildMetadataURL() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("buildMetadataURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

Run tests:

```bash
go test ./protocol/v2 -v -run TestMetadata
```

### Commit

```
feat: implement NuGet v2 package metadata retrieval

- Add MetadataClient with OData endpoint support
- Implement GetPackageMetadata for specific versions
- Implement ListVersions using FindPackagesById
- Build Packages(Id='',Version='') URLs
- Parse Atom entry XML responses
- Create comprehensive metadata tests

Chunk: M2.11
Status: ✓ Complete
```

---

## [M2.12] Protocol v2 - Download

**Time Estimate:** 2 hours
**Dependencies:** M2.11 (V2 metadata)
**Status:** Not started

### What You'll Build

Implement package download from NuGet v2 feeds using the package download endpoint.

### Step-by-Step Instructions

**Step 1: Create v2 download client**

Create `protocol/v2/download.go`:

```go
package v2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// DownloadClient provides v2 package download functionality
type DownloadClient struct {
	httpClient *nugethttp.Client
}

// NewDownloadClient creates a new v2 download client
func NewDownloadClient(httpClient *nugethttp.Client) *DownloadClient {
	return &DownloadClient{
		httpClient: httpClient,
	}
}

// DownloadPackage downloads a .nupkg file and returns the response body
// Caller is responsible for closing the response body
func (c *DownloadClient) DownloadPackage(ctx context.Context, feedURL, packageID, version string) (io.ReadCloser, error) {
	// Build download URL
	// Format: /package/{id}/{version}
	downloadURL, err := c.buildDownloadURL(feedURL, packageID, version)
	if err != nil {
		return nil, fmt.Errorf("build download URL: %w", err)
	}

	// Execute download request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("package %s %s not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

// DownloadLatestPackage downloads the latest version of a package
// Caller is responsible for closing the response body
func (c *DownloadClient) DownloadLatestPackage(ctx context.Context, feedURL, packageID string) (io.ReadCloser, error) {
	// Build latest download URL
	// Format: /package/{id}
	downloadURL, err := c.buildLatestDownloadURL(feedURL, packageID)
	if err != nil {
		return nil, fmt.Errorf("build download URL: %w", err)
	}

	// Execute download request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("package %s not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (c *DownloadClient) buildDownloadURL(feedURL, packageID, version string) (string, error) {
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 download endpoint: /package/{id}/{version}
	downloadURL := fmt.Sprintf("%spackage/%s/%s", baseURL, packageID, version)

	return downloadURL, nil
}

func (c *DownloadClient) buildLatestDownloadURL(feedURL, packageID string) (string, error) {
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 latest download endpoint: /package/{id}
	downloadURL := fmt.Sprintf("%spackage/%s", baseURL, packageID)

	return downloadURL, nil
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v2

# Format check
gofmt -l protocol/v2/
```

### Testing

Create `protocol/v2/download_test.go`:

```go
package v2

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

func setupV2DownloadServer(t *testing.T) (*httptest.Server, *DownloadClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle versioned download
		if strings.HasPrefix(path, "/package/") {
			parts := strings.Split(strings.TrimPrefix(path, "/package/"), "/")

			if len(parts) == 2 {
				// /package/{id}/{version}
				packageID := parts[0]
				version := parts[1]

				if packageID == "Newtonsoft.Json" && version == "13.0.3" {
					w.Header().Set("Content-Type", "application/zip")
					w.Write([]byte("PK\x03\x04")) // ZIP signature
					w.Write([]byte("fake nupkg content"))
					return
				}
			} else if len(parts) == 1 {
				// /package/{id} (latest)
				packageID := parts[0]

				if packageID == "Newtonsoft.Json" {
					w.Header().Set("Content-Type", "application/zip")
					w.Write([]byte("PK\x03\x04")) // ZIP signature
					w.Write([]byte("fake latest nupkg content"))
					return
				}
			}

			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	downloadClient := NewDownloadClient(httpClient)

	return server, downloadClient
}

func TestDownloadClient_DownloadPackage(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadPackage(ctx, server.URL, "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Check ZIP signature
	if len(content) < 4 || content[0] != 'P' || content[1] != 'K' {
		t.Error("downloaded content does not have ZIP signature")
	}

	if !strings.Contains(string(content), "fake nupkg content") {
		t.Error("downloaded content missing expected data")
	}
}

func TestDownloadClient_DownloadPackage_NotFound(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadPackage(ctx, server.URL, "NonExistent.Package", "1.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_DownloadLatestPackage(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadLatestPackage(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("DownloadLatestPackage() error = %v", err)
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Check ZIP signature
	if len(content) < 4 || content[0] != 'P' || content[1] != 'K' {
		t.Error("downloaded content does not have ZIP signature")
	}

	if !strings.Contains(string(content), "fake latest nupkg content") {
		t.Error("downloaded content missing expected data")
	}
}

func TestDownloadClient_DownloadLatestPackage_NotFound(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadLatestPackage(ctx, server.URL, "NonExistent.Package")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_BuildDownloadURL(t *testing.T) {
	client := NewDownloadClient(nil)

	tests := []struct {
		name      string
		feedURL   string
		packageID string
		version   string
		want      string
	}{
		{
			name:      "basic",
			feedURL:   "https://api.nuget.org/v2/",
			packageID: "Newtonsoft.Json",
			version:   "13.0.3",
			want:      "https://api.nuget.org/v2/package/Newtonsoft.Json/13.0.3",
		},
		{
			name:      "no trailing slash",
			feedURL:   "https://api.nuget.org/v2",
			packageID: "Newtonsoft.Json",
			version:   "13.0.3",
			want:      "https://api.nuget.org/v2/package/Newtonsoft.Json/13.0.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildDownloadURL(tt.feedURL, tt.packageID, tt.version)
			if err != nil {
				t.Fatalf("buildDownloadURL() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("buildDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadClient_BuildLatestDownloadURL(t *testing.T) {
	client := NewDownloadClient(nil)

	got, err := client.buildLatestDownloadURL("https://api.nuget.org/v2/", "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("buildLatestDownloadURL() error = %v", err)
	}

	want := "https://api.nuget.org/v2/package/Newtonsoft.Json"
	if got != want {
		t.Errorf("buildLatestDownloadURL() = %q, want %q", got, want)
	}
}
```

Run tests:

```bash
go test ./protocol/v2 -v -run TestDownload
go test ./protocol/v2 -v
```

### Commit

```
feat: implement NuGet v2 package download

- Add DownloadPackage for versioned downloads
- Add DownloadLatestPackage for latest version
- Build /package/{id}/{version} URLs
- Handle 404 errors for missing packages
- Create comprehensive download tests

Chunk: M2.12
Status: ✓ Complete
```

---

I'll continue with the authentication chunks (M2.13-M2.15) and remaining protocol chunks in the next section of this file.

## [M2.13] Authentication - API Key

**Time Estimate:** 1 hour
**Dependencies:** M2.1 (HTTP client)
**Status:** Not started

### What You'll Build

Implement API key authentication for NuGet feeds using the X-NuGet-ApiKey header.

### Step-by-Step Instructions

**Step 1: Create authentication types**

Create `auth/types.go`:

```go
package auth

import (
	"net/http"
)

// Authenticator is the interface for NuGet authentication
type Authenticator interface {
	// Authenticate adds authentication to the request
	Authenticate(req *http.Request) error
}

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeNone    AuthType = "none"
	AuthTypeAPIKey  AuthType = "apikey"
	AuthTypeBearer  AuthType = "bearer"
	AuthTypeBasic   AuthType = "basic"
)
```

**Step 2: Create API key authenticator**

Create `auth/apikey.go`:

```go
package auth

import (
	"net/http"
)

// APIKeyAuthenticator implements API key authentication
type APIKeyAuthenticator struct {
	apiKey string
}

// NewAPIKeyAuthenticator creates a new API key authenticator
func NewAPIKeyAuthenticator(apiKey string) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKey: apiKey,
	}
}

// Authenticate adds the X-NuGet-ApiKey header to the request
func (a *APIKeyAuthenticator) Authenticate(req *http.Request) error {
	if a.apiKey != "" {
		req.Header.Set("X-NuGet-ApiKey", a.apiKey)
	}
	return nil
}

// Type returns the authentication type
func (a *APIKeyAuthenticator) Type() AuthType {
	return AuthTypeAPIKey
}
```

### Verification Steps

```bash
# Build
go build ./auth

# Format check
gofmt -l auth/
```

### Testing

Create `auth/apikey_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuthenticator_Authenticate(t *testing.T) {
	apiKey := "test-api-key-12345"
	auth := NewAPIKeyAuthenticator(apiKey)

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("X-NuGet-ApiKey")
	if got != apiKey {
		t.Errorf("X-NuGet-ApiKey = %q, want %q", got, apiKey)
	}
}

func TestAPIKeyAuthenticator_EmptyKey(t *testing.T) {
	auth := NewAPIKeyAuthenticator("")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should not set header if key is empty
	got := req.Header.Get("X-NuGet-ApiKey")
	if got != "" {
		t.Errorf("X-NuGet-ApiKey = %q, want empty", got)
	}
}

func TestAPIKeyAuthenticator_Type(t *testing.T) {
	auth := NewAPIKeyAuthenticator("test-key")

	if auth.Type() != AuthTypeAPIKey {
		t.Errorf("Type() = %q, want %q", auth.Type(), AuthTypeAPIKey)
	}
}

func TestAPIKeyAuthenticator_RealRequest(t *testing.T) {
	apiKey := "test-api-key"
	auth := NewAPIKeyAuthenticator(apiKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey := r.Header.Get("X-NuGet-ApiKey")
		if gotKey != apiKey {
			t.Errorf("X-NuGet-ApiKey = %q, want %q", gotKey, apiKey)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	err = auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
```

Run tests:

```bash
go test ./auth -v -run TestAPIKey
```

### Commit

```
feat: implement API key authentication

- Add Authenticator interface
- Implement APIKeyAuthenticator with X-NuGet-ApiKey header
- Add AuthType enumeration
- Create comprehensive API key tests

Chunk: M2.13
Status: ✓ Complete
```

---

## [M2.14] Authentication - Bearer Token

**Time Estimate:** 1 hour
**Dependencies:** M2.13 (Auth types)
**Status:** Not started

### What You'll Build

Implement bearer token authentication for NuGet feeds using the Authorization header.

### Step-by-Step Instructions

**Step 1: Create bearer token authenticator**

Create `auth/bearer.go`:

```go
package auth

import (
	"fmt"
	"net/http"
)

// BearerAuthenticator implements bearer token authentication
type BearerAuthenticator struct {
	token string
}

// NewBearerAuthenticator creates a new bearer token authenticator
func NewBearerAuthenticator(token string) *BearerAuthenticator {
	return &BearerAuthenticator{
		token: token,
	}
}

// Authenticate adds the Authorization: Bearer header to the request
func (a *BearerAuthenticator) Authenticate(req *http.Request) error {
	if a.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	}
	return nil
}

// Type returns the authentication type
func (a *BearerAuthenticator) Type() AuthType {
	return AuthTypeBearer
}
```

### Verification Steps

```bash
# Build
go build ./auth

# Format check
gofmt -l auth/
```

### Testing

Create `auth/bearer_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearerAuthenticator_Authenticate(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature"
	auth := NewBearerAuthenticator(token)

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("Authorization")
	want := "Bearer " + token
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestBearerAuthenticator_EmptyToken(t *testing.T) {
	auth := NewBearerAuthenticator("")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should not set header if token is empty
	got := req.Header.Get("Authorization")
	if got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestBearerAuthenticator_Type(t *testing.T) {
	auth := NewBearerAuthenticator("test-token")

	if auth.Type() != AuthTypeBearer {
		t.Errorf("Type() = %q, want %q", auth.Type(), AuthTypeBearer)
	}
}

func TestBearerAuthenticator_RealRequest(t *testing.T) {
	token := "test-bearer-token-12345"
	auth := NewBearerAuthenticator(token)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth := r.Header.Get("Authorization")
		wantAuth := "Bearer " + token

		if gotAuth != wantAuth {
			t.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
		}

		// Verify it starts with "Bearer "
		if !strings.HasPrefix(gotAuth, "Bearer ") {
			t.Error("Authorization header missing 'Bearer ' prefix")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	err = auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
```

Run tests:

```bash
go test ./auth -v -run TestBearer
```

### Commit

```
feat: implement bearer token authentication

- Implement BearerAuthenticator with Authorization header
- Format token as "Bearer {token}"
- Create comprehensive bearer token tests

Chunk: M2.14
Status: ✓ Complete
```

---

## [M2.15] Authentication - Basic Auth

**Time Estimate:** 1 hour
**Dependencies:** M2.13 (Auth types)
**Status:** Not started

### What You'll Build

Implement HTTP basic authentication for NuGet feeds using base64-encoded username:password.

### Step-by-Step Instructions

**Step 1: Create basic auth authenticator**

Create `auth/basic.go`:

```go
package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// BasicAuthenticator implements HTTP basic authentication
type BasicAuthenticator struct {
	username string
	password string
}

// NewBasicAuthenticator creates a new basic auth authenticator
func NewBasicAuthenticator(username, password string) *BasicAuthenticator {
	return &BasicAuthenticator{
		username: username,
		password: password,
	}
}

// Authenticate adds the Authorization: Basic header to the request
func (a *BasicAuthenticator) Authenticate(req *http.Request) error {
	if a.username != "" || a.password != "" {
		// Encode username:password as base64
		credentials := fmt.Sprintf("%s:%s", a.username, a.password)
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encoded))
	}
	return nil
}

// Type returns the authentication type
func (a *BasicAuthenticator) Type() AuthType {
	return AuthTypeBasic
}
```

### Verification Steps

```bash
# Build
go build ./auth

# Format check
gofmt -l auth/
```

### Testing

Create `auth/basic_test.go`:

```go
package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasicAuthenticator_Authenticate(t *testing.T) {
	username := "testuser"
	password := "testpass"
	auth := NewBasicAuthenticator(username, password)

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("Authorization")
	if got == "" {
		t.Fatal("Authorization header not set")
	}

	// Should start with "Basic "
	if !strings.HasPrefix(got, "Basic ") {
		t.Errorf("Authorization = %q, want prefix 'Basic '", got)
	}

	// Decode and verify
	encoded := strings.TrimPrefix(got, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	want := username + ":" + password
	if string(decoded) != want {
		t.Errorf("decoded credentials = %q, want %q", decoded, want)
	}
}

func TestBasicAuthenticator_EmptyCredentials(t *testing.T) {
	auth := NewBasicAuthenticator("", "")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should still set header even if empty (edge case)
	// Some implementations might want this behavior
	got := req.Header.Get("Authorization")
	if got != "" {
		// Verify it's properly encoded empty credentials
		encoded := strings.TrimPrefix(got, "Basic ")
		decoded, _ := base64.StdEncoding.DecodeString(encoded)
		if string(decoded) != ":" {
			t.Errorf("decoded = %q, want ':'", decoded)
		}
	}
}

func TestBasicAuthenticator_OnlyUsername(t *testing.T) {
	auth := NewBasicAuthenticator("testuser", "")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("Authorization")
	encoded := strings.TrimPrefix(got, "Basic ")
	decoded, _ := base64.StdEncoding.DecodeString(encoded)

	want := "testuser:"
	if string(decoded) != want {
		t.Errorf("decoded = %q, want %q", decoded, want)
	}
}

func TestBasicAuthenticator_Type(t *testing.T) {
	auth := NewBasicAuthenticator("user", "pass")

	if auth.Type() != AuthTypeBasic {
		t.Errorf("Type() = %q, want %q", auth.Type(), AuthTypeBasic)
	}
}

func TestBasicAuthenticator_RealRequest(t *testing.T) {
	username := "testuser"
	password := "testpass"
	auth := NewBasicAuthenticator(username, password)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth := r.Header.Get("Authorization")

		// Verify format
		if !strings.HasPrefix(gotAuth, "Basic ") {
			t.Error("Authorization header missing 'Basic ' prefix")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Decode and verify credentials
		encoded := strings.TrimPrefix(gotAuth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Errorf("base64 decode error = %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		want := username + ":" + password
		if string(decoded) != want {
			t.Errorf("credentials = %q, want %q", decoded, want)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	err = auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
```

Run tests:

```bash
go test ./auth -v -run TestBasic
go test ./auth -v
```

### Commit

```
feat: implement HTTP basic authentication

- Implement BasicAuthenticator with base64 encoding
- Format as "Basic {base64(username:password)}"
- Support username-only authentication
- Create comprehensive basic auth tests

Chunk: M2.15
Status: ✓ Complete
```

---
