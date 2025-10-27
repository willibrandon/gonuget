package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
	"github.com/willibrandon/gonuget/version"
)

func TestClient_GetRepositoryManager(t *testing.T) {
	client := NewClient(ClientConfig{})

	manager := client.GetRepositoryManager()
	if manager == nil {
		t.Error("GetRepositoryManager() returned nil")
	}
}

func TestClient_SetTargetFramework(t *testing.T) {
	client := NewClient(ClientConfig{})

	fw, _ := frameworks.ParseFramework("net6.0")
	client.SetTargetFramework(fw)

	got := client.GetTargetFramework()
	if got == nil {
		t.Fatal("GetTargetFramework() returned nil")
	}

	if got.Framework != ".NETCoreApp" {
		t.Errorf("Framework = %q, want .NETCoreApp", got.Framework)
	}
}

func TestClient_GetCompatibleDependencies_NoFramework(t *testing.T) {
	client := NewClient(ClientConfig{})

	fw, _ := frameworks.ParseFramework("net6.0")
	vr, _ := version.ParseVersionRange("[13.0.1,)")
	metadata := &PackageMetadata{
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: fw,
				Dependencies: []PackageDependency{
					{ID: "Newtonsoft.Json", VersionRange: vr},
				},
			},
		},
	}

	deps, err := client.GetCompatibleDependencies(metadata)
	if err != nil {
		t.Fatalf("GetCompatibleDependencies() error = %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("len(deps) = %d, want 1", len(deps))
	}
}

func TestClient_GetCompatibleDependencies_WithFramework(t *testing.T) {
	fw, _ := frameworks.ParseFramework("net6.0")

	client := NewClient(ClientConfig{
		TargetFramework: fw,
	})

	net6, _ := frameworks.ParseFramework("net6.0")
	net48, _ := frameworks.ParseFramework("net48")
	vr1, _ := version.ParseVersionRange("[13.0.1,)")
	vr2, _ := version.ParseVersionRange("[4.5.0,)")

	metadata := &PackageMetadata{
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: net6,
				Dependencies: []PackageDependency{
					{ID: "Newtonsoft.Json", VersionRange: vr1},
				},
			},
			{
				TargetFramework: net48,
				Dependencies: []PackageDependency{
					{ID: "System.Memory", VersionRange: vr2},
				},
			},
		},
	}

	deps, err := client.GetCompatibleDependencies(metadata)
	if err != nil {
		t.Fatalf("GetCompatibleDependencies() error = %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("len(deps) = %d, want 1", len(deps))
	}

	if deps[0].ID != "Newtonsoft.Json" {
		t.Errorf("deps[0].ID = %q, want Newtonsoft.Json", deps[0].ID)
	}
}

func TestClient_GetCompatibleDependencies_NoMatch(t *testing.T) {
	fw, _ := frameworks.ParseFramework("net8.0")

	client := NewClient(ClientConfig{
		TargetFramework: fw,
	})

	net35, _ := frameworks.ParseFramework("net35")
	vr, _ := version.ParseVersionRange("[1.0.0,)")

	metadata := &PackageMetadata{
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: net35,
				Dependencies: []PackageDependency{
					{ID: "OldPackage", VersionRange: vr},
				},
			},
		},
	}

	deps, err := client.GetCompatibleDependencies(metadata)
	if err != nil {
		t.Fatalf("GetCompatibleDependencies() error = %v", err)
	}

	// net8.0 is not compatible with net35, should return empty
	if len(deps) != 0 {
		t.Errorf("len(deps) = %d, want 0", len(deps))
	}
}

func TestClient_GetCompatibleDependencies_BestMatch(t *testing.T) {
	fw, _ := frameworks.ParseFramework("net8.0")

	client := NewClient(ClientConfig{
		TargetFramework: fw,
	})

	netstandard20, _ := frameworks.ParseFramework("netstandard2.0")
	net6, _ := frameworks.ParseFramework("net6.0")
	vr1, _ := version.ParseVersionRange("[1.0.0,)")
	vr2, _ := version.ParseVersionRange("[2.0.0,)")

	metadata := &PackageMetadata{
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: netstandard20,
				Dependencies: []PackageDependency{
					{ID: "StandardPackage", VersionRange: vr1},
				},
			},
			{
				TargetFramework: net6,
				Dependencies: []PackageDependency{
					{ID: "Net6Package", VersionRange: vr2},
				},
			},
		},
	}

	deps, err := client.GetCompatibleDependencies(metadata)
	if err != nil {
		t.Fatalf("GetCompatibleDependencies() error = %v", err)
	}

	// net8.0 should pick net6.0 (most specific compatible)
	if len(deps) != 1 {
		t.Errorf("len(deps) = %d, want 1", len(deps))
	}

	if deps[0].ID != "Net6Package" {
		t.Errorf("deps[0].ID = %q, want Net6Package", deps[0].ID)
	}
}

func TestClient_GetCompatibleDependencies_EmptyDependencies(t *testing.T) {
	client := NewClient(ClientConfig{})

	metadata := &PackageMetadata{
		DependencyGroups: []PackageDependencyGroup{},
	}

	deps, err := client.GetCompatibleDependencies(metadata)
	if err != nil {
		t.Fatalf("GetCompatibleDependencies() error = %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("len(deps) = %d, want 0", len(deps))
	}
}

func TestClient_PackageIdentity_String(t *testing.T) {
	ver := version.MustParse("1.2.3")
	identity := NewPackageIdentity("TestPackage", ver)

	got := identity.String()
	want := "TestPackage 1.2.3"

	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestClient_NoRepositories(t *testing.T) {
	client := NewClient(ClientConfig{})
	ctx := context.Background()

	// Test GetPackageMetadata
	_, err := client.GetPackageMetadata(ctx, "test", "1.0.0")
	if err == nil {
		t.Error("GetPackageMetadata() expected error for no repositories")
	}

	// Test ListVersions
	_, err = client.ListVersions(ctx, "test")
	if err == nil {
		t.Error("ListVersions() expected error for no repositories")
	}

	// Test DownloadPackage
	_, err = client.DownloadPackage(ctx, "test", "1.0.0")
	if err == nil {
		t.Error("DownloadPackage() expected error for no repositories")
	}
}

func TestClient_GetPackageMetadata_RepositoryErrors(t *testing.T) {
	// Create a test server that returns 404
	server404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server404.Close()

	// Create a test server that returns 500
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server500.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	// Add two repositories that will both fail
	repo1 := NewSourceRepository(RepositoryConfig{
		Name:       "repo-404",
		SourceURL:  server404.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo1)

	repo2 := NewSourceRepository(RepositoryConfig{
		Name:       "repo-500",
		SourceURL:  server500.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repo2)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()

	// All repositories fail - should return error
	_, err := client.GetPackageMetadata(ctx, "test", "1.0.0")
	if err == nil {
		t.Error("GetPackageMetadata() expected error when all repositories fail")
	}

	// Error message should indicate package not found
	if !strings.Contains(err.Error(), "package not found") {
		t.Errorf("GetPackageMetadata() error = %v, want error containing 'package not found'", err)
	}
}

func TestClient_GetPackageMetadata_FallbackToSecondRepo(t *testing.T) {
	// Create a test server that returns 404
	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer serverFail.Close()

	// Create a working test server
	serverSuccess := createTestServer()
	defer serverSuccess.Close()

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	// Add failing repository first
	repoFail := NewSourceRepository(RepositoryConfig{
		Name:       "repo-fail",
		SourceURL:  serverFail.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repoFail)

	// Add working repository second
	repoSuccess := NewSourceRepository(RepositoryConfig{
		Name:       "repo-success",
		SourceURL:  serverSuccess.URL + "/index.json",
		HTTPClient: httpClient,
	})
	_ = repoManager.AddRepository(repoSuccess)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()

	// First repo fails, should fall back to second repo
	metadata, err := client.GetPackageMetadata(ctx, "TestPkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v, expected success with fallback", err)
	}

	if metadata == nil {
		t.Fatal("GetPackageMetadata() returned nil metadata")
	}

	if metadata.ID != "TestPkg" {
		t.Errorf("GetPackageMetadata() ID = %s, want TestPkg", metadata.ID)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("GetPackageMetadata() Version = %s, want 1.0.0", metadata.Version)
	}
}

func TestClient_ResolvePackageDependencies_NoResolver(t *testing.T) {
	// Client without target framework should not initialize resolver
	client := NewClient(ClientConfig{})
	ctx := context.Background()

	_, err := client.ResolvePackageDependencies(ctx, "test", "1.0.0")
	if err == nil {
		t.Error("ResolvePackageDependencies() expected error when resolver not initialized")
	}
	if err.Error() != "resolver not initialized" {
		t.Errorf("ResolvePackageDependencies() error = %v, want 'resolver not initialized'", err)
	}
}

func TestClient_ResolverInitialization(t *testing.T) {
	// Create repository manager
	repoManager := NewRepositoryManager()

	// Test 1: Client without target framework should not have resolver
	clientNoFW := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})
	if clientNoFW.resolver != nil {
		t.Error("Client without target framework should not have resolver initialized")
	}

	// Test 2: Client with target framework should have resolver
	fw, _ := frameworks.ParseFramework("net8.0")
	clientWithFW := NewClient(ClientConfig{
		RepositoryManager: repoManager,
		TargetFramework:   fw,
	})
	if clientWithFW.resolver == nil {
		t.Error("Client with target framework should have resolver initialized")
	}
}

func TestClientMetadataAdapter_NoRepositories(t *testing.T) {
	// Create client with no repositories
	client := NewClient(ClientConfig{})

	// Create V3 clients for the adapter
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	metadataClient := v3.NewMetadataClient(httpClient, serviceIndexClient)

	adapter := &clientMetadataAdapter{
		client:           client,
		v3MetadataClient: metadataClient,
		v3ServiceClient:  serviceIndexClient,
	}
	ctx := context.Background()

	_, err := adapter.GetPackageMetadata(ctx, "", "test", "*")
	if err == nil {
		t.Error("GetPackageMetadata() expected error for package not found")
	}
	// Note: V3 API will return "package not found" error, not "no repositories configured"
}
