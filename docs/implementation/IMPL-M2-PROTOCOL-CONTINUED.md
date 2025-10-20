# Implementation Guide: Milestone 2 - Protocol Implementation (Continued)

**Milestone:** M2 - Protocol Implementation (Continued)
**Chunks:** M2.6 - M2.18
**Previous File:** IMPL-M2-FOUNDATION.md (chunks M2.1-M2.5)

---

## [M2.6] Protocol v3 - Metadata

**Time Estimate:** 4 hours
**Dependencies:** M2.4 (Service index), M1.2 (Version types), M1.8 (Framework types)
**Status:** Not started

### What You'll Build

Implement NuGet v3 package metadata retrieval from the RegistrationsBaseUrl resource. This provides detailed package information including all versions, dependencies, and framework groups.

### Step-by-Step Instructions

**Step 1: Create metadata types**

Add to `protocol/v3/types.go`:

```go
// RegistrationIndex represents the top-level registration index
type RegistrationIndex struct {
	Count int                  `json:"count"`
	Items []RegistrationPage   `json:"items"`
}

// RegistrationPage represents a page of registration entries
type RegistrationPage struct {
	ID    string              `json:"@id"`
	Count int                 `json:"count"`
	Items []RegistrationLeaf  `json:"items,omitempty"`
	Lower string              `json:"lower"`
	Upper string              `json:"upper"`
}

// RegistrationLeaf represents a single package version registration
type RegistrationLeaf struct {
	ID           string                `json:"@id"`
	CatalogEntry *RegistrationCatalog  `json:"catalogEntry"`
	PackageContent string              `json:"packageContent"`
}

// RegistrationCatalog contains detailed package metadata
type RegistrationCatalog struct {
	ID                 string                      `json:"@id"`
	PackageID          string                      `json:"id"`
	Version            string                      `json:"version"`
	Authors            string                      `json:"authors,omitempty"`
	Description        string                      `json:"description,omitempty"`
	IconURL            string                      `json:"iconUrl,omitempty"`
	LicenseURL         string                      `json:"licenseUrl,omitempty"`
	LicenseExpression  string                      `json:"licenseExpression,omitempty"`
	ProjectURL         string                      `json:"projectUrl,omitempty"`
	Published          string                      `json:"published,omitempty"`
	RequireLicenseAcceptance bool                  `json:"requireLicenseAcceptance"`
	Summary            string                      `json:"summary,omitempty"`
	Tags               string                      `json:"tags,omitempty"`
	Title              string                      `json:"title,omitempty"`
	DependencyGroups   []DependencyGroup           `json:"dependencyGroups,omitempty"`
	PackageTypes       []PackageType               `json:"packageTypes,omitempty"`
}

// DependencyGroup represents dependencies for a specific target framework
type DependencyGroup struct {
	TargetFramework string       `json:"targetFramework,omitempty"`
	Dependencies    []Dependency `json:"dependencies,omitempty"`
}

// Dependency represents a single package dependency
type Dependency struct {
	ID    string `json:"id"`
	Range string `json:"range,omitempty"`
}

// PackageType represents the type of package
type PackageType struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}
```

**Step 2: Create metadata client**

Create `protocol/v3/metadata.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/yourusername/gonuget/http"
)

// MetadataClient provides package metadata functionality
type MetadataClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// NewMetadataClient creates a new metadata client
func NewMetadataClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *MetadataClient {
	return &MetadataClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// GetPackageMetadata retrieves metadata for a specific package ID
// Returns all versions and their metadata
func (c *MetadataClient) GetPackageMetadata(ctx context.Context, sourceURL, packageID string) (*RegistrationIndex, error) {
	// Get registration base URL from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeRegistrationsBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("get registration URL: %w", err)
	}

	// Build registration index URL
	// Format: {baseURL}/{packageID}/index.json
	packageIDLower := strings.ToLower(packageID)
	registrationURL := strings.TrimSuffix(baseURL, "/") + "/" + packageIDLower + "/index.json"

	// Fetch registration index
	req, err := http.NewRequest("GET", registrationURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch registration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %q not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("registration returned %d: %s", resp.StatusCode, body)
	}

	var index RegistrationIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("decode registration: %w", err)
	}

	// Fetch inline pages if items are not populated
	for i := range index.Items {
		if len(index.Items[i].Items) == 0 && index.Items[i].ID != "" {
			page, err := c.fetchRegistrationPage(ctx, index.Items[i].ID)
			if err != nil {
				return nil, fmt.Errorf("fetch page %s: %w", index.Items[i].ID, err)
			}
			index.Items[i] = *page
		}
	}

	return &index, nil
}

// GetVersionMetadata retrieves metadata for a specific package version
func (c *MetadataClient) GetVersionMetadata(ctx context.Context, sourceURL, packageID, version string) (*RegistrationCatalog, error) {
	index, err := c.GetPackageMetadata(ctx, sourceURL, packageID)
	if err != nil {
		return nil, err
	}

	// Search for version in all pages
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry != nil && leaf.CatalogEntry.Version == version {
				return leaf.CatalogEntry, nil
			}
		}
	}

	return nil, fmt.Errorf("version %q not found for package %q", version, packageID)
}

// ListVersions returns all available versions for a package
func (c *MetadataClient) ListVersions(ctx context.Context, sourceURL, packageID string) ([]string, error) {
	index, err := c.GetPackageMetadata(ctx, sourceURL, packageID)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry != nil {
				versions = append(versions, leaf.CatalogEntry.Version)
			}
		}
	}

	return versions, nil
}

func (c *MetadataClient) fetchRegistrationPage(ctx context.Context, pageURL string) (*RegistrationPage, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("page returned %d: %s", resp.StatusCode, body)
	}

	var page RegistrationPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}

	return &page, nil
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v3

# Format check
gofmt -l protocol/v3/
```

### Testing

Create `protocol/v3/metadata_test.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/yourusername/gonuget/http"
)

var testRegistrationIndex = &RegistrationIndex{
	Count: 1,
	Items: []RegistrationPage{
		{
			ID:    "https://api.nuget.org/v3/registration5-gz-semver2/newtonsoft.json/page/1.0.0/13.0.3.json",
			Count: 2,
			Lower: "1.0.0",
			Upper: "13.0.3",
			Items: []RegistrationLeaf{
				{
					ID: "https://api.nuget.org/v3/registration5-gz-semver2/newtonsoft.json/13.0.2.json",
					CatalogEntry: &RegistrationCatalog{
						ID:          "https://api.nuget.org/v3/catalog0/data/2022.02.20.06.18.49/newtonsoft.json.13.0.2.json",
						PackageID:   "Newtonsoft.Json",
						Version:     "13.0.2",
						Authors:     "James Newton-King",
						Description: "Json.NET is a popular high-performance JSON framework for .NET",
						LicenseExpression: "MIT",
						ProjectURL:  "https://www.newtonsoft.com/json",
						Tags:        "json serialization",
						DependencyGroups: []DependencyGroup{
							{
								TargetFramework: "net6.0",
								Dependencies:    []Dependency{},
							},
						},
					},
					PackageContent: "https://api.nuget.org/v3-flatcontainer/newtonsoft.json/13.0.2/newtonsoft.json.13.0.2.nupkg",
				},
				{
					ID: "https://api.nuget.org/v3/registration5-gz-semver2/newtonsoft.json/13.0.3.json",
					CatalogEntry: &RegistrationCatalog{
						ID:          "https://api.nuget.org/v3/catalog0/data/2023.03.08.18.36.53/newtonsoft.json.13.0.3.json",
						PackageID:   "Newtonsoft.Json",
						Version:     "13.0.3",
						Authors:     "James Newton-King",
						Description: "Json.NET is a popular high-performance JSON framework for .NET",
						LicenseExpression: "MIT",
						ProjectURL:  "https://www.newtonsoft.com/json",
						Tags:        "json serialization",
						DependencyGroups: []DependencyGroup{
							{
								TargetFramework: "net6.0",
								Dependencies:    []Dependency{},
							},
						},
					},
					PackageContent: "https://api.nuget.org/v3-flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg",
				},
			},
		},
	},
}

func setupMetadataServer(t *testing.T) (*httptest.Server, *MetadataClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/registration/",
					Type: ResourceTypeRegistrationsBaseUrl,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})

	// Registration endpoint
	mux.HandleFunc("/registration/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/registration/")

		if strings.HasPrefix(path, "newtonsoft.json/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(testRegistrationIndex)
			return
		}

		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	metadataClient := NewMetadataClient(httpClient, serviceIndexClient)

	return server, metadataClient
}

func TestMetadataClient_GetPackageMetadata(t *testing.T) {
	server, client := setupMetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	index, err := client.GetPackageMetadata(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	if index.Count != 1 {
		t.Errorf("Count = %d, want 1", index.Count)
	}

	if len(index.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(index.Items))
	}

	page := index.Items[0]
	if page.Count != 2 {
		t.Errorf("page.Count = %d, want 2", page.Count)
	}

	if len(page.Items) != 2 {
		t.Errorf("len(page.Items) = %d, want 2", len(page.Items))
	}

	// Check first version
	leaf := page.Items[0]
	if leaf.CatalogEntry == nil {
		t.Fatal("CatalogEntry is nil")
	}

	if leaf.CatalogEntry.PackageID != "Newtonsoft.Json" {
		t.Errorf("PackageID = %q, want Newtonsoft.Json", leaf.CatalogEntry.PackageID)
	}

	if leaf.CatalogEntry.Version != "13.0.2" {
		t.Errorf("Version = %q, want 13.0.2", leaf.CatalogEntry.Version)
	}

	if leaf.CatalogEntry.LicenseExpression != "MIT" {
		t.Errorf("LicenseExpression = %q, want MIT", leaf.CatalogEntry.LicenseExpression)
	}

	if len(leaf.CatalogEntry.DependencyGroups) != 1 {
		t.Errorf("len(DependencyGroups) = %d, want 1", len(leaf.CatalogEntry.DependencyGroups))
	}
}

func TestMetadataClient_GetVersionMetadata(t *testing.T) {
	server, client := setupMetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	catalog, err := client.GetVersionMetadata(ctx, server.URL, "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("GetVersionMetadata() error = %v", err)
	}

	if catalog.PackageID != "Newtonsoft.Json" {
		t.Errorf("PackageID = %q, want Newtonsoft.Json", catalog.PackageID)
	}

	if catalog.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", catalog.Version)
	}

	if catalog.Authors != "James Newton-King" {
		t.Errorf("Authors = %q, want James Newton-King", catalog.Authors)
	}
}

func TestMetadataClient_GetVersionMetadata_NotFound(t *testing.T) {
	server, client := setupMetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.GetVersionMetadata(ctx, server.URL, "Newtonsoft.Json", "99.0.0")
	if err == nil {
		t.Error("expected error for non-existent version")
	}

	expectedMsg := "version \"99.0.0\" not found"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestMetadataClient_ListVersions(t *testing.T) {
	server, client := setupMetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	versions, err := client.ListVersions(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) != 2 {
		t.Fatalf("len(versions) = %d, want 2", len(versions))
	}

	expected := []string{"13.0.2", "13.0.3"}
	for i, want := range expected {
		if versions[i] != want {
			t.Errorf("versions[%d] = %q, want %q", i, versions[i], want)
		}
	}
}

func TestMetadataClient_PackageNotFound(t *testing.T) {
	server, client := setupMetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.GetPackageMetadata(ctx, server.URL, "NonExistent.Package")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	expectedMsg := "not found"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}
```

Run tests:

```bash
go test ./protocol/v3 -v -run TestMetadata
```

### Commit

```
feat: implement NuGet v3 package metadata retrieval

- Add RegistrationIndex, RegistrationPage, RegistrationLeaf types
- Add RegistrationCatalog with dependency groups
- Implement GetPackageMetadata for all versions
- Implement GetVersionMetadata for specific version
- Add ListVersions helper
- Support paged registration fetching
- Create comprehensive metadata tests

Chunk: M2.6
Status: ✓ Complete
```

---

## [M2.7] Protocol v3 - Download

**Time Estimate:** 2 hours
**Dependencies:** M2.4 (Service index), M1.2 (Version types)
**Status:** Not started

### What You'll Build

Implement package download from PackageBaseAddress resource with .nupkg file retrieval and .nuspec manifest fetching.

### Step-by-Step Instructions

**Step 1: Create download client**

Create `protocol/v3/download.go`:

```go
package v3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/yourusername/gonuget/http"
)

// DownloadClient provides package download functionality
type DownloadClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// NewDownloadClient creates a new download client
func NewDownloadClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *DownloadClient {
	return &DownloadClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// DownloadPackage downloads a .nupkg file and returns the response body
// Caller is responsible for closing the response body
func (c *DownloadClient) DownloadPackage(ctx context.Context, sourceURL, packageID, version string) (io.ReadCloser, error) {
	// Get package base address from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypePackageBaseAddress)
	if err != nil {
		return nil, fmt.Errorf("get package base URL: %w", err)
	}

	// Build download URL
	// Format: {baseURL}/{packageID}/{version}/{packageID}.{version}.nupkg
	packageIDLower := strings.ToLower(packageID)
	versionLower := strings.ToLower(version)
	downloadURL := fmt.Sprintf("%s%s/%s/%s.%s.nupkg",
		strings.TrimSuffix(baseURL, "/"),
		packageIDLower,
		versionLower,
		packageIDLower,
		versionLower,
	)

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

// DownloadNuspec downloads the .nuspec manifest file for a package
// Caller is responsible for closing the response body
func (c *DownloadClient) DownloadNuspec(ctx context.Context, sourceURL, packageID, version string) (io.ReadCloser, error) {
	// Get package base address from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypePackageBaseAddress)
	if err != nil {
		return nil, fmt.Errorf("get package base URL: %w", err)
	}

	// Build nuspec URL
	// Format: {baseURL}/{packageID}/{version}/{packageID}.nuspec
	packageIDLower := strings.ToLower(packageID)
	versionLower := strings.ToLower(version)
	nuspecURL := fmt.Sprintf("%s%s/%s/%s.nuspec",
		strings.TrimSuffix(baseURL, "/"),
		packageIDLower,
		versionLower,
		packageIDLower,
	)

	// Execute request
	req, err := http.NewRequest("GET", nuspecURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("nuspec request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("nuspec for %s %s not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, fmt.Errorf("nuspec returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

// GetPackageVersions lists all available versions for a package
// Uses the package base address versions endpoint
func (c *DownloadClient) GetPackageVersions(ctx context.Context, sourceURL, packageID string) ([]string, error) {
	// Get package base address from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypePackageBaseAddress)
	if err != nil {
		return nil, fmt.Errorf("get package base URL: %w", err)
	}

	// Build versions URL
	// Format: {baseURL}/{packageID}/index.json
	packageIDLower := strings.ToLower(packageID)
	versionsURL := fmt.Sprintf("%s%s/index.json",
		strings.TrimSuffix(baseURL, "/"),
		packageIDLower,
	)

	// Execute request
	req, err := http.NewRequest("GET", versionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("versions request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("versions returned %d: %s", resp.StatusCode, body)
	}

	// Parse versions response
	var versionsResp struct {
		Versions []string `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionsResp); err != nil {
		return nil, fmt.Errorf("decode versions: %w", err)
	}

	return versionsResp.Versions, nil
}
```

Add import:

```go
import (
	"encoding/json"
)
```

### Verification Steps

```bash
# Build
go build ./protocol/v3

# Format check
gofmt -l protocol/v3/
```

### Testing

Create `protocol/v3/download_test.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/yourusername/gonuget/http"
)

func setupDownloadServer(t *testing.T) (*httptest.Server, *DownloadClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/packages/",
					Type: ResourceTypePackageBaseAddress,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})

	// Package download endpoint
	mux.HandleFunc("/packages/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/packages/")

		// Handle versions endpoint
		if strings.HasSuffix(path, "/index.json") {
			packageID := strings.TrimSuffix(path, "/index.json")
			if packageID == "newtonsoft.json" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"versions": []string{"13.0.1", "13.0.2", "13.0.3"},
				})
				return
			}
			http.NotFound(w, r)
			return
		}

		// Handle .nupkg download
		if strings.HasSuffix(path, ".nupkg") {
			if strings.Contains(path, "newtonsoft.json/13.0.3/") {
				w.Header().Set("Content-Type", "application/zip")
				w.Write([]byte("PK\x03\x04")) // ZIP file signature
				w.Write([]byte("fake nupkg content"))
				return
			}
			http.NotFound(w, r)
			return
		}

		// Handle .nuspec download
		if strings.HasSuffix(path, ".nuspec") {
			if strings.Contains(path, "newtonsoft.json/13.0.3/") {
				w.Header().Set("Content-Type", "application/xml")
				w.Write([]byte(`<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>Newtonsoft.Json</id>
    <version>13.0.3</version>
  </metadata>
</package>`))
				return
			}
			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	downloadClient := NewDownloadClient(httpClient, serviceIndexClient)

	return server, downloadClient
}

func TestDownloadClient_DownloadPackage(t *testing.T) {
	server, client := setupDownloadServer(t)
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
	server, client := setupDownloadServer(t)
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

func TestDownloadClient_DownloadNuspec(t *testing.T) {
	server, client := setupDownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadNuspec(ctx, server.URL, "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("DownloadNuspec() error = %v", err)
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "<?xml") {
		t.Error("nuspec content missing XML declaration")
	}

	if !strings.Contains(contentStr, "<id>Newtonsoft.Json</id>") {
		t.Error("nuspec content missing package ID")
	}

	if !strings.Contains(contentStr, "<version>13.0.3</version>") {
		t.Error("nuspec content missing version")
	}
}

func TestDownloadClient_DownloadNuspec_NotFound(t *testing.T) {
	server, client := setupDownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadNuspec(ctx, server.URL, "NonExistent.Package", "1.0.0")
	if err == nil {
		t.Error("expected error for non-existent nuspec")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_GetPackageVersions(t *testing.T) {
	server, client := setupDownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	versions, err := client.GetPackageVersions(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("GetPackageVersions() error = %v", err)
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

func TestDownloadClient_GetPackageVersions_NotFound(t *testing.T) {
	server, client := setupDownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.GetPackageVersions(ctx, server.URL, "NonExistent.Package")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}
```

Run tests:

```bash
go test ./protocol/v3 -v -run TestDownload
```

### Commit

```
feat: implement NuGet v3 package download

- Add DownloadPackage for .nupkg file retrieval
- Add DownloadNuspec for manifest retrieval
- Add GetPackageVersions for version listing
- Build correct PackageBaseAddress URLs
- Handle 404 errors for missing packages
- Create comprehensive download tests

Chunk: M2.7
Status: ✓ Complete
```

---

## [M2.8] Protocol v3 - Autocomplete

**Time Estimate:** 2 hours
**Dependencies:** M2.4 (Service index)
**Status:** Not started

### What You'll Build

Implement package ID and version autocomplete using the SearchAutocompleteService resource.

### Step-by-Step Instructions

**Step 1: Create autocomplete types**

Add to `protocol/v3/types.go`:

```go
// AutocompleteResponse represents the response from autocomplete API
type AutocompleteResponse struct {
	TotalHits int      `json:"totalHits"`
	Data      []string `json:"data"`
	Context   interface{} `json:"@context,omitempty"`
}
```

**Step 2: Create autocomplete client**

Create `protocol/v3/autocomplete.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	nugethttp "github.com/yourusername/gonuget/http"
)

// AutocompleteClient provides package and version autocomplete functionality
type AutocompleteClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// AutocompleteOptions holds autocomplete parameters
type AutocompleteOptions struct {
	Query       string
	Skip        int
	Take        int
	Prerelease  bool
	SemVerLevel string
}

// NewAutocompleteClient creates a new autocomplete client
func NewAutocompleteClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *AutocompleteClient {
	return &AutocompleteClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// AutocompletePackageIDs provides package ID autocomplete
// Returns package IDs that match the query
func (c *AutocompleteClient) AutocompletePackageIDs(ctx context.Context, sourceURL string, opts AutocompleteOptions) (*AutocompleteResponse, error) {
	// Get autocomplete endpoint from service index
	autocompleteURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeSearchAutocompleteService)
	if err != nil {
		return nil, fmt.Errorf("get autocomplete URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	if opts.Query != "" {
		params.Set("q", opts.Query)
	}
	if opts.Skip > 0 {
		params.Set("skip", strconv.Itoa(opts.Skip))
	}
	if opts.Take > 0 {
		params.Set("take", strconv.Itoa(opts.Take))
	} else {
		params.Set("take", "20") // Default
	}
	params.Set("prerelease", strconv.FormatBool(opts.Prerelease))
	if opts.SemVerLevel != "" {
		params.Set("semVerLevel", opts.SemVerLevel)
	}

	// Build full URL
	fullURL := autocompleteURL
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	// Execute request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("autocomplete returned %d: %s", resp.StatusCode, body)
	}

	var autocompleteResp AutocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&autocompleteResp); err != nil {
		return nil, fmt.Errorf("decode autocomplete response: %w", err)
	}

	return &autocompleteResp, nil
}

// AutocompletePackageVersions provides version autocomplete for a package
// Returns available versions for the specified package ID
func (c *AutocompleteClient) AutocompletePackageVersions(ctx context.Context, sourceURL, packageID string, opts AutocompleteOptions) (*AutocompleteResponse, error) {
	// Get autocomplete endpoint from service index
	autocompleteURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeSearchAutocompleteService)
	if err != nil {
		return nil, fmt.Errorf("get autocomplete URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("id", packageID)
	if opts.Prerelease {
		params.Set("prerelease", "true")
	} else {
		params.Set("prerelease", "false")
	}
	if opts.SemVerLevel != "" {
		params.Set("semVerLevel", opts.SemVerLevel)
	}

	// Build full URL
	fullURL := autocompleteURL + "?" + params.Encode()

	// Execute request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("version autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("version autocomplete returned %d: %s", resp.StatusCode, body)
	}

	var autocompleteResp AutocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&autocompleteResp); err != nil {
		return nil, fmt.Errorf("decode version autocomplete response: %w", err)
	}

	return &autocompleteResp, nil
}
```

### Verification Steps

```bash
# Build
go build ./protocol/v3

# Format check
gofmt -l protocol/v3/
```

### Testing

Create `protocol/v3/autocomplete_test.go`:

```go
package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/yourusername/gonuget/http"
)

func setupAutocompleteServer(t *testing.T) (*httptest.Server, *AutocompleteClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/autocomplete",
					Type: ResourceTypeSearchAutocompleteService,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})

	// Autocomplete endpoint
	mux.HandleFunc("/autocomplete", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		var resp AutocompleteResponse

		// Package ID autocomplete
		if query.Has("q") {
			q := query.Get("q")
			resp = AutocompleteResponse{
				TotalHits: 3,
				Data: []string{
					"Newtonsoft.Json",
					"Newtonsoft.Json.Bson",
					"Newtonsoft.Json.Schema",
				},
			}

			// Filter based on query
			if !strings.Contains(strings.ToLower(q), "newtonsoft") {
				resp.TotalHits = 0
				resp.Data = []string{}
			}
		} else if query.Has("id") {
			// Version autocomplete
			packageID := query.Get("id")
			if strings.EqualFold(packageID, "Newtonsoft.Json") {
				resp = AutocompleteResponse{
					TotalHits: 3,
					Data: []string{
						"13.0.1",
						"13.0.2",
						"13.0.3",
					},
				}
			} else {
				resp = AutocompleteResponse{
					TotalHits: 0,
					Data:      []string{},
				}
			}
		} else {
			http.Error(w, "missing required parameter", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	autocompleteClient := NewAutocompleteClient(httpClient, serviceIndexClient)

	return server, autocompleteClient
}

func TestAutocompleteClient_AutocompletePackageIDs(t *testing.T) {
	server, client := setupAutocompleteServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.AutocompletePackageIDs(ctx, server.URL, AutocompleteOptions{
		Query:      "newtonsoft",
		Take:       20,
		Prerelease: true,
	})

	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}

	if resp.TotalHits != 3 {
		t.Errorf("TotalHits = %d, want 3", resp.TotalHits)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("len(Data) = %d, want 3", len(resp.Data))
	}

	expected := []string{
		"Newtonsoft.Json",
		"Newtonsoft.Json.Bson",
		"Newtonsoft.Json.Schema",
	}

	for i, want := range expected {
		if resp.Data[i] != want {
			t.Errorf("Data[%d] = %q, want %q", i, resp.Data[i], want)
		}
	}
}

func TestAutocompleteClient_AutocompletePackageIDs_NoResults(t *testing.T) {
	server, client := setupAutocompleteServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.AutocompletePackageIDs(ctx, server.URL, AutocompleteOptions{
		Query:      "nonexistent",
		Take:       20,
		Prerelease: true,
	})

	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}

	if resp.TotalHits != 0 {
		t.Errorf("TotalHits = %d, want 0", resp.TotalHits)
	}

	if len(resp.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(resp.Data))
	}
}

func TestAutocompleteClient_AutocompletePackageVersions(t *testing.T) {
	server, client := setupAutocompleteServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.AutocompletePackageVersions(ctx, server.URL, "Newtonsoft.Json", AutocompleteOptions{
		Prerelease: true,
	})

	if err != nil {
		t.Fatalf("AutocompletePackageVersions() error = %v", err)
	}

	if resp.TotalHits != 3 {
		t.Errorf("TotalHits = %d, want 3", resp.TotalHits)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("len(Data) = %d, want 3", len(resp.Data))
	}

	expected := []string{"13.0.1", "13.0.2", "13.0.3"}

	for i, want := range expected {
		if resp.Data[i] != want {
			t.Errorf("Data[%d] = %q, want %q", i, resp.Data[i], want)
		}
	}
}

func TestAutocompleteClient_AutocompletePackageVersions_NotFound(t *testing.T) {
	server, client := setupAutocompleteServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.AutocompletePackageVersions(ctx, server.URL, "NonExistent.Package", AutocompleteOptions{
		Prerelease: true,
	})

	if err != nil {
		t.Fatalf("AutocompletePackageVersions() error = %v", err)
	}

	if resp.TotalHits != 0 {
		t.Errorf("TotalHits = %d, want 0", resp.TotalHits)
	}

	if len(resp.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(resp.Data))
	}
}

func TestAutocompleteClient_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			index := &ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/autocomplete",
						Type: ResourceTypeSearchAutocompleteService,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
			return
		}

		query := r.URL.Query()

		// Validate pagination parameters
		skip := query.Get("skip")
		if skip != "10" {
			t.Errorf("skip = %q, want 10", skip)
		}

		take := query.Get("take")
		if take != "5" {
			t.Errorf("take = %q, want 5", take)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&AutocompleteResponse{
			TotalHits: 100,
			Data:      []string{},
		})
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewAutocompleteClient(httpClient, serviceIndexClient)

	ctx := context.Background()

	_, err := client.AutocompletePackageIDs(ctx, server.URL, AutocompleteOptions{
		Query: "test",
		Skip:  10,
		Take:  5,
	})

	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}
}
```

Run tests:

```bash
go test ./protocol/v3 -v -run TestAutocomplete
```

### Commit

```
feat: implement NuGet v3 autocomplete

- Add AutocompleteResponse type
- Implement package ID autocomplete
- Implement package version autocomplete
- Support pagination (skip/take)
- Support prerelease and SemVer level filtering
- Create comprehensive autocomplete tests

Chunk: M2.8
Status: ✓ Complete
```

---
