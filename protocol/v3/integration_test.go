package v3

import (
	"context"
	"slices"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const (
	nugetOrgV3 = "https://api.nuget.org/v3/index.json"
)

// TestIntegration_ServiceIndex tests against real NuGet.org
func TestIntegration_ServiceIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)

	ctx := context.Background()
	index, err := client.GetServiceIndex(ctx, nugetOrgV3)
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if index.Version != "3.0.0" {
		t.Errorf("GetServiceIndex() version = %s, want 3.0.0", index.Version)
	}

	if len(index.Resources) == 0 {
		t.Error("GetServiceIndex() returned no resources")
	}
}

// TestIntegration_Search tests search against real NuGet.org
func TestIntegration_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	indexClient := NewServiceIndexClient(httpClient)

	ctx := context.Background()

	searchClient := NewSearchClient(httpClient, indexClient)
	response, err := searchClient.SearchSimple(ctx, nugetOrgV3, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(response.Data) == 0 {
		t.Error("Search() returned no results for Newtonsoft.Json")
	}

	// Verify first result is Newtonsoft.Json
	if response.Data[0].PackageID != "Newtonsoft.Json" {
		t.Errorf("Search() first result = %s, want Newtonsoft.Json", response.Data[0].PackageID)
	}
}

// TestIntegration_Metadata tests metadata retrieval from real NuGet.org
func TestIntegration_Metadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	indexClient := NewServiceIndexClient(httpClient)

	ctx := context.Background()

	metadataClient := NewMetadataClient(httpClient, indexClient)

	// Test getting specific version metadata
	metadata, err := metadataClient.GetVersionMetadata(ctx, nugetOrgV3, "Newtonsoft.Json", "13.0.1")
	if err != nil {
		t.Fatalf("GetVersionMetadata() error = %v", err)
	}

	if metadata.PackageID != "Newtonsoft.Json" {
		t.Errorf("GetVersionMetadata() PackageID = %s, want Newtonsoft.Json", metadata.PackageID)
	}

	if metadata.Version != "13.0.1" {
		t.Errorf("GetVersionMetadata() Version = %s, want 13.0.1", metadata.Version)
	}
}

// TestIntegration_ListVersions tests version listing from real NuGet.org
func TestIntegration_ListVersions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	indexClient := NewServiceIndexClient(httpClient)

	ctx := context.Background()

	metadataClient := NewMetadataClient(httpClient, indexClient)

	versions, err := metadataClient.ListVersions(ctx, nugetOrgV3, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) == 0 {
		t.Error("ListVersions() returned no versions for Newtonsoft.Json")
	}

	// Verify we got expected versions
	found13 := false
	for _, v := range versions {
		if v == "13.0.1" || v == "13.0.2" || v == "13.0.3" {
			found13 = true
			break
		}
	}

	if !found13 {
		t.Error("ListVersions() did not return expected version 13.x")
	}
}

// TestIntegration_Download tests package download from real NuGet.org
func TestIntegration_Download(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	indexClient := NewServiceIndexClient(httpClient)

	ctx := context.Background()

	downloadClient := NewDownloadClient(httpClient, indexClient)

	// Download a small package
	rc, err := downloadClient.DownloadPackage(ctx, nugetOrgV3, "Newtonsoft.Json", "13.0.1")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	// Just verify we got data
	buf := make([]byte, 1024)
	n, err := rc.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Read() error = %v", err)
	}

	if n == 0 {
		t.Error("DownloadPackage() returned empty data")
	}
}

// TestIntegration_Autocomplete tests autocomplete from real NuGet.org
func TestIntegration_Autocomplete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	indexClient := NewServiceIndexClient(httpClient)

	ctx := context.Background()

	autocompleteClient := NewAutocompleteClient(httpClient, indexClient)

	// Test package ID autocomplete
	response, err := autocompleteClient.AutocompletePackageIDs(ctx, nugetOrgV3, "Newtonsoft", 0, 10, false)
	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}

	if len(response.Data) == 0 {
		t.Error("AutocompletePackageIDs() returned no results for 'Newtonsoft'")
	}

	// Verify we got Newtonsoft.Json
	if !slices.Contains(response.Data, "Newtonsoft.Json") {
		t.Error("AutocompletePackageIDs() did not return Newtonsoft.Json")
	}
}
