package restore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

func TestRestorer_Restore_WritesCacheFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	// Create test project
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(content), 0644)
	require.NoError(t, err)

	proj, err := project.LoadProject(projectPath)
	require.NoError(t, err)

	// Create restorer
	opts := &Options{
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
		PackagesFolder: filepath.Join(tmpDir, "packages"),
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// Run restore
	packageRefs := proj.GetPackageReferences()
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify cache file was written
	cachePath := GetCacheFilePath(projectPath)
	_, err = os.Stat(cachePath)
	require.NoError(t, err, "cache file should exist")

	// Load and verify cache contents
	cache, err := LoadCacheFile(cachePath)
	require.NoError(t, err)
	assert.True(t, cache.IsValid())
	assert.True(t, cache.Success)
	// Both paths should match (no symlink resolution to match dotnet behavior)
	assert.Equal(t, proj.Path, cache.ProjectFilePath)
	assert.NotEmpty(t, cache.DgSpecHash)
	assert.NotEmpty(t, cache.ExpectedPackageFiles)

	// Verify expected package files include Newtonsoft.Json
	foundNewtonsoft := false
	for _, pkgPath := range cache.ExpectedPackageFiles {
		if strings.Contains(strings.ToLower(pkgPath), "newtonsoft.json") {
			foundNewtonsoft = true
			break
		}
	}
	assert.True(t, foundNewtonsoft, "cache should include Newtonsoft.Json")
}

func TestRestorer_Restore_NoOp_CacheHit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(content), 0644)
	require.NoError(t, err)

	proj, err := project.LoadProject(projectPath)
	require.NoError(t, err)

	opts := &Options{
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
		PackagesFolder: filepath.Join(tmpDir, "packages"),
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// First restore - should hit network
	packageRefs := proj.GetPackageReferences()
	result1, err := restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)
	require.NotNil(t, result1)
	assert.False(t, result1.CacheHit, "first restore should not be cache hit")

	// Verify cache was written
	cachePath := GetCacheFilePath(projectPath)
	_, err = os.Stat(cachePath)
	require.NoError(t, err)

	// Second restore - should use cache (no network)
	result2, err := restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.True(t, result2.CacheHit, "second restore should be cache hit")

	// Results should be identical
	assert.Equal(t, len(result1.DirectPackages), len(result2.DirectPackages))
	assert.Equal(t, len(result1.TransitivePackages), len(result2.TransitivePackages))
}

func TestRestorer_Restore_NoOp_HashMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	// Initial project with Newtonsoft.Json 13.0.3
	content1 := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(content1), 0644)
	require.NoError(t, err)

	proj, err := project.LoadProject(projectPath)
	require.NoError(t, err)

	opts := &Options{
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
		PackagesFolder: filepath.Join(tmpDir, "packages"),
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// First restore
	packageRefs := proj.GetPackageReferences()
	_, err = restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)

	// Modify project (change version to 13.0.2)
	content2 := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.2" />
  </ItemGroup>
</Project>`

	err = os.WriteFile(projectPath, []byte(content2), 0644)
	require.NoError(t, err)

	proj, err = project.LoadProject(projectPath)
	require.NoError(t, err)

	// Second restore - should NOT use cache (hash changed)
	packageRefs = proj.GetPackageReferences()
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)
	assert.False(t, result.CacheHit, "restore should not use cache when hash changes")

	// Should have downloaded new version
	found := false
	for _, pkg := range result.DirectPackages {
		if strings.ToLower(pkg.ID) == "newtonsoft.json" && pkg.Version == "13.0.2" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have downloaded 13.0.2")
}

func TestRestorer_Restore_ForceFlag_IgnoresCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(content), 0644)
	require.NoError(t, err)

	proj, err := project.LoadProject(projectPath)
	require.NoError(t, err)

	opts := &Options{
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
		PackagesFolder: filepath.Join(tmpDir, "packages"),
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// First restore
	packageRefs := proj.GetPackageReferences()
	_, err = restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)

	// Second restore with --force flag
	opts.Force = true
	restorer = NewRestorer(opts, console)
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have performed full restore (not cached)
	assert.False(t, result.CacheHit, "restore with Force should not use cache")
	assert.NotEmpty(t, result.DirectPackages)
}
