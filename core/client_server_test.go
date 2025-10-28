package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/version"
)

// createTestServer creates a test NuGet v3 server
func createTestServer() *httptest.Server {
	// Map of lowercase package IDs to their canonical casing
	// This mimics how real NuGet.org preserves the original casing
	packageCasing := map[string]string{
		"testpkg":     "TestPkg",
		"testpackage": "TestPackage",
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/registration/"):
			// Registration/metadata
			// Extract package ID from path like /registration/testpkg/index.json
			path := strings.TrimPrefix(r.URL.Path, "/registration/")
			path = strings.TrimSuffix(path, "/index.json")
			packageIDLower := path

			// Get canonical casing for this package ID
			// In real NuGet v3, the catalog entry preserves the original package ID casing
			canonicalID := packageCasing[packageIDLower]
			if canonicalID == "" {
				// If not in our map, just use the lowercase version
				canonicalID = packageIDLower
			}

			// Return registration index with all versions
			response := map[string]any{
				"count": 1,
				"items": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/" + packageIDLower + "/page.json",
						"lower": "1.0.0",
						"upper": "2.0.0",
						"count": 3,
						"items": []map[string]any{
							{
								"@id": "http://" + r.Host + "/registration/" + packageIDLower + "/1.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/" + packageIDLower + "/1.0.0.json",
									"id":      canonicalID,
									"version": "1.0.0",
								},
								"packageContent": "http://" + r.Host + "/download/" + packageIDLower + "/1.0.0/" + packageIDLower + ".1.0.0.nupkg",
							},
							{
								"@id": "http://" + r.Host + "/registration/" + packageIDLower + "/1.5.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/" + packageIDLower + "/1.5.0.json",
									"id":      canonicalID,
									"version": "1.5.0",
								},
								"packageContent": "http://" + r.Host + "/download/" + packageIDLower + "/1.5.0/" + packageIDLower + ".1.5.0.nupkg",
							},
							{
								"@id": "http://" + r.Host + "/registration/" + packageIDLower + "/2.0.0.json",
								"catalogEntry": map[string]any{
									"@id":     "http://" + r.Host + "/catalog/" + packageIDLower + "/2.0.0.json",
									"id":      canonicalID,
									"version": "2.0.0",
								},
								"packageContent": "http://" + r.Host + "/download/" + packageIDLower + "/2.0.0/" + packageIDLower + ".2.0.0.nupkg",
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		case r.URL.Path == "/index.json":
			// Service index (must come AFTER more specific paths like /registration/)
			w.Header().Set("Content-Type", "application/json")
			index := map[string]any{
				"version": "3.0.0",
				"resources": []map[string]any{
					{
						"@id":   "http://" + r.Host + "/registration/",
						"@type": "RegistrationsBaseUrl",
					},
					{
						"@id":   "http://" + r.Host + "/search",
						"@type": "SearchQueryService",
					},
					{
						"@id":   "http://" + r.Host + "/download/",
						"@type": "PackageBaseAddress",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case strings.Contains(r.URL.Path, "/search"):
			// Search
			query := r.URL.Query().Get("q")
			response := map[string]any{
				"totalHits": 1,
				"data": []map[string]any{
					{
						"id":      query,
						"version": "1.0.0",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/download/"):
			// Download
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write([]byte("fake package content"))

		default:
			http.NotFound(w, r)
		}
	}))
}

func TestClient_FindBestVersion(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	vr, _ := version.ParseVersionRange("[1.0.0,2.0.0)")

	bestVer, err := client.FindBestVersion(ctx, "TestPackage", vr)
	if err != nil {
		t.Fatalf("FindBestVersion() error = %v", err)
	}

	if bestVer.String() != "1.0.0" {
		t.Errorf("FindBestVersion() = %s, want 1.0.0", bestVer.String())
	}
}

func TestClient_FindBestVersion_NoMatch(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	vr, _ := version.ParseVersionRange("[3.0.0,)")

	_, err := client.FindBestVersion(ctx, "TestPackage", vr)
	if err == nil {
		t.Error("FindBestVersion() expected error for no matching version")
	}
}

func TestClient_ResolvePackageVersion_Exact(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	resolvedVer, err := client.ResolvePackageVersion(ctx, "TestPackage", "1.0.0", false)
	if err != nil {
		t.Fatalf("ResolvePackageVersion() error = %v", err)
	}

	if resolvedVer.String() != "1.0.0" {
		t.Errorf("ResolvePackageVersion() = %s, want 1.0.0", resolvedVer.String())
	}
}

func TestClient_ResolvePackageVersion_Range(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	resolvedVer, err := client.ResolvePackageVersion(ctx, "TestPackage", "[1.0.0,2.0.0)", false)
	if err != nil {
		t.Fatalf("ResolvePackageVersion() error = %v", err)
	}

	if resolvedVer.String() != "1.0.0" {
		t.Errorf("ResolvePackageVersion() = %s, want 1.0.0", resolvedVer.String())
	}
}

func TestClient_ResolvePackageVersion_NotFound(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	_, err := client.ResolvePackageVersion(ctx, "TestPackage", "99.0.0", false)
	if err == nil {
		t.Error("ResolvePackageVersion() expected error for version not found")
	}
}

func TestClient_SearchPackages(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	results, err := client.SearchPackages(ctx, "TestQuery", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchPackages() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchPackages() returned %d repos, want 1", len(results))
	}

	if len(results["test"]) != 1 {
		t.Errorf("SearchPackages() returned %d results for test, want 1", len(results["test"]))
	}
}

func TestSourceRepository_GetMetadata(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	ctx := context.Background()
	metadata, err := repo.GetMetadata(ctx, nil, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	if metadata.ID != "TestPkg" {
		t.Errorf("GetMetadata() ID = %s, want TestPkg", metadata.ID)
	}
}

func TestSourceRepository_ListVersions(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	ctx := context.Background()
	versions, err := repo.ListVersions(ctx, nil, "TestPkg")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("ListVersions() returned %d versions, want 3", len(versions))
	}
}

func TestSourceRepository_Search(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	ctx := context.Background()
	results, err := repo.Search(ctx, nil, "test", SearchOptions{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Search() returned %d results, want 1", len(results))
	}
}

func TestSourceRepository_DownloadPackage(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	ctx := context.Background()
	rc, err := repo.DownloadPackage(ctx, nil, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	data, _ := io.ReadAll(rc)
	if string(data) != "fake package content" {
		t.Errorf("DownloadPackage() data = %s, want 'fake package content'", string(data))
	}
}

func TestRepositoryManager_SearchAll(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	manager := NewRepositoryManager()

	repo1 := NewSourceRepository(RepositoryConfig{
		Name:       "repo1",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	repo2 := NewSourceRepository(RepositoryConfig{
		Name:       "repo2",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	_ = manager.AddRepository(repo1)
	_ = manager.AddRepository(repo2)

	ctx := context.Background()
	results, err := manager.SearchAll(ctx, nil, "test", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAll() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchAll() returned %d repos, want 2", len(results))
	}

	if len(results["repo1"]) != 1 {
		t.Errorf("SearchAll() repo1 returned %d results, want 1", len(results["repo1"]))
	}

	if len(results["repo2"]) != 1 {
		t.Errorf("SearchAll() repo2 returned %d results, want 1", len(results["repo2"]))
	}
}

func TestClient_GetPackageMetadata_MultipleRepos(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	// Add two repos
	repo1 := NewSourceRepository(RepositoryConfig{
		Name:       "repo1",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	repo2 := NewSourceRepository(RepositoryConfig{
		Name:       "repo2",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	_ = repoManager.AddRepository(repo1)
	_ = repoManager.AddRepository(repo2)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	metadata, err := client.GetPackageMetadata(ctx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	if metadata.ID != "TestPkg" {
		t.Errorf("GetPackageMetadata() ID = %s, want TestPkg", metadata.ID)
	}
}

func TestClient_ListVersions_MultipleRepos(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	// Add two repos (both will return same versions, should be deduplicated)
	repo1 := NewSourceRepository(RepositoryConfig{
		Name:       "repo1",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	repo2 := NewSourceRepository(RepositoryConfig{
		Name:       "repo2",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})

	_ = repoManager.AddRepository(repo1)
	_ = repoManager.AddRepository(repo2)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	versions, err := client.ListVersions(ctx, "TestPkg")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	// Should deduplicate versions from multiple repos
	if len(versions) != 3 {
		t.Errorf("ListVersions() returned %d versions, want 3", len(versions))
	}
}

func TestClient_DownloadPackage_MultipleRepos(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	rc, err := client.DownloadPackage(ctx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	data, _ := io.ReadAll(rc)
	if len(data) == 0 {
		t.Error("DownloadPackage() returned empty data")
	}
}
