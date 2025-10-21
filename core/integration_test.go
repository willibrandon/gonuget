package core

import (
	"context"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/version"
)

const (
	nugetOrgV3 = "https://api.nuget.org/v3"
)

// TestIntegration_Client_SearchPackages tests search against real NuGet.org
func TestIntegration_Client_SearchPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	results, err := client.SearchPackages(ctx, "Newtonsoft.Json", SearchOptions{
		Skip: 0,
		Take: 1,
	})
	if err != nil {
		t.Fatalf("SearchPackages() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("SearchPackages() returned no results")
	}

	if len(results["nuget.org"]) == 0 {
		t.Error("SearchPackages() returned no results for nuget.org")
	}
}

// TestIntegration_Client_GetPackageMetadata tests metadata retrieval from real NuGet.org
func TestIntegration_Client_GetPackageMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	metadata, err := client.GetPackageMetadata(ctx, "Newtonsoft.Json", "13.0.1")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	if metadata.ID != "Newtonsoft.Json" {
		t.Errorf("GetPackageMetadata() ID = %s, want Newtonsoft.Json", metadata.ID)
	}

	if metadata.Version != "13.0.1" {
		t.Errorf("GetPackageMetadata() Version = %s, want 13.0.1", metadata.Version)
	}
}

// TestIntegration_Client_ListVersions tests version listing from real NuGet.org
func TestIntegration_Client_ListVersions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	versions, err := client.ListVersions(ctx, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) == 0 {
		t.Error("ListVersions() returned no versions")
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

// TestIntegration_Client_FindBestVersion tests version resolution from real NuGet.org
func TestIntegration_Client_FindBestVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	vr, _ := version.ParseVersionRange("[13.0.0,14.0.0)")

	bestVer, err := client.FindBestVersion(ctx, "Newtonsoft.Json", vr)
	if err != nil {
		t.Fatalf("FindBestVersion() error = %v", err)
	}

	// Should find latest 13.x version
	if bestVer.Major != 13 {
		t.Errorf("FindBestVersion() major = %d, want 13", bestVer.Major)
	}
}

// TestIntegration_Client_ResolvePackageVersion tests version resolution from real NuGet.org
func TestIntegration_Client_ResolvePackageVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()

	tests := []struct {
		name       string
		versionStr string
		wantMajor  int
	}{
		{
			name:       "exact version",
			versionStr: "13.0.1",
			wantMajor:  13,
		},
		{
			name:       "version range",
			versionStr: "[13.0.0,14.0.0)",
			wantMajor:  13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvedVer, err := client.ResolvePackageVersion(ctx, "Newtonsoft.Json", tt.versionStr, false)
			if err != nil {
				t.Fatalf("ResolvePackageVersion() error = %v", err)
			}

			if resolvedVer.Major != tt.wantMajor {
				t.Errorf("ResolvePackageVersion() major = %d, want %d", resolvedVer.Major, tt.wantMajor)
			}
		})
	}
}

// TestIntegration_Client_DownloadPackage tests package download from real NuGet.org
func TestIntegration_Client_DownloadPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
	})

	ctx := context.Background()
	rc, err := client.DownloadPackage(ctx, "Newtonsoft.Json", "13.0.1")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer rc.Close()

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

// TestIntegration_RepositoryManager_MultipleRepos tests searching across multiple repos
func TestIntegration_RepositoryManager_MultipleRepos(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	manager := NewRepositoryManager()

	// Add nuget.org
	nugetOrg := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	manager.AddRepository(nugetOrg)

	ctx := context.Background()
	results, err := manager.SearchAll(ctx, "Newtonsoft.Json", SearchOptions{
		Skip: 0,
		Take: 1,
	})
	if err != nil {
		t.Fatalf("SearchAll() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("SearchAll() returned no results")
	}

	if len(results["nuget.org"]) == 0 {
		t.Error("SearchAll() returned no results for nuget.org")
	}
}

// TestIntegration_Client_GetCompatibleDependencies tests dependency filtering
func TestIntegration_Client_GetCompatibleDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	httpClient := nugethttp.NewClient(nil)
	repoManager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  nugetOrgV3,
		HTTPClient: httpClient,
	})
	repoManager.AddRepository(repo)

	fw, _ := frameworks.ParseFramework("net6.0")
	client := NewClient(ClientConfig{
		RepositoryManager: repoManager,
		TargetFramework:   fw,
	})

	ctx := context.Background()
	protoMetadata, err := client.GetPackageMetadata(ctx, "Newtonsoft.Json", "13.0.1")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	// Convert ProtocolMetadata to PackageMetadata for dependency testing
	ver, _ := version.Parse(protoMetadata.Version)
	metadata := &PackageMetadata{
		Identity: PackageIdentity{
			ID:      protoMetadata.ID,
			Version: ver,
		},
		DependencyGroups: make([]PackageDependencyGroup, len(protoMetadata.Dependencies)),
	}

	// Convert dependency groups
	for i, protoGroup := range protoMetadata.Dependencies {
		fw, err := frameworks.ParseFramework(protoGroup.TargetFramework)
		if err != nil {
			// Skip invalid frameworks
			continue
		}

		deps := make([]PackageDependency, len(protoGroup.Dependencies))
		for j, protoDep := range protoGroup.Dependencies {
			vr, _ := version.ParseVersionRange(protoDep.Range)
			deps[j] = PackageDependency{
				ID:           protoDep.ID,
				VersionRange: vr,
			}
		}

		metadata.DependencyGroups[i] = PackageDependencyGroup{
			TargetFramework: fw,
			Dependencies:    deps,
		}
	}

	// Note: Real Newtonsoft.Json has complex dependency groups
	// This test just verifies the API works
	deps, err := client.GetCompatibleDependencies(metadata)
	if err != nil {
		t.Fatalf("GetCompatibleDependencies() error = %v", err)
	}

	// Just verify no error - Newtonsoft.Json may or may not have deps
	_ = deps
}
