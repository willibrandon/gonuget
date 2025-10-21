package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
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
						ID:                "https://api.nuget.org/v3/catalog0/data/2022.02.20.06.18.49/newtonsoft.json.13.0.2.json",
						PackageID:         "Newtonsoft.Json",
						Version:           "13.0.2",
						Authors:           "James Newton-King",
						Description:       "Json.NET is a popular high-performance JSON framework for .NET",
						LicenseExpression: "MIT",
						ProjectURL:        "https://www.newtonsoft.com/json",
						Tags:              []string{"json", "serialization"},
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
						ID:                "https://api.nuget.org/v3/catalog0/data/2023.03.08.18.36.53/newtonsoft.json.13.0.3.json",
						PackageID:         "Newtonsoft.Json",
						Version:           "13.0.3",
						Authors:           "James Newton-King",
						Description:       "Json.NET is a popular high-performance JSON framework for .NET",
						LicenseExpression: "MIT",
						ProjectURL:        "https://www.newtonsoft.com/json",
						Tags:              []string{"json", "serialization"},
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

func setupMetadataServer() (*httptest.Server, *MetadataClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/registration/",
					Type: ResourceTypeRegistrationsBaseURL,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(index)
	})

	// Registration endpoint
	mux.HandleFunc("/registration/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/registration/")

		if strings.HasPrefix(path, "newtonsoft.json/") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(testRegistrationIndex)
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
	server, client := setupMetadataServer()
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
	server, client := setupMetadataServer()
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
	server, client := setupMetadataServer()
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
	server, client := setupMetadataServer()
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
	server, client := setupMetadataServer()
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
