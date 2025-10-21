package core

import (
	"context"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/version"
)

func setupTestClient() (*Client, *RepositoryManager) {
	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  "https://api.nuget.org/v3/index.json",
		HTTPClient: httpClient,
	})

	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	return client, repoManager
}

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
