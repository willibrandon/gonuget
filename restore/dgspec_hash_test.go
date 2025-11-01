package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

func TestCalculateDgSpecHash_Deterministic(t *testing.T) {
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

	// Calculate hash twice
	hash1, err := CalculateDgSpecHash(proj)
	require.NoError(t, err)

	hash2, err := CalculateDgSpecHash(proj)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
}

func TestCalculateDgSpecHash_Changes(t *testing.T) {
	tmpDir := t.TempDir()

	// Project 1: Newtonsoft.Json 13.0.3
	project1Path := filepath.Join(tmpDir, "project1.csproj")
	content1 := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`
	err := os.WriteFile(project1Path, []byte(content1), 0644)
	require.NoError(t, err)

	proj1, err := project.LoadProject(project1Path)
	require.NoError(t, err)
	hash1, err := CalculateDgSpecHash(proj1)
	require.NoError(t, err)

	// Project 2: Newtonsoft.Json 13.0.2 (different version)
	project2Path := filepath.Join(tmpDir, "project2.csproj")
	content2 := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.2" />
  </ItemGroup>
</Project>`
	err = os.WriteFile(project2Path, []byte(content2), 0644)
	require.NoError(t, err)

	proj2, err := project.LoadProject(project2Path)
	require.NoError(t, err)
	hash2, err := CalculateDgSpecHash(proj2)
	require.NoError(t, err)

	// Hashes should differ
	assert.NotEqual(t, hash1, hash2)
}

func TestCalculateDgSpecHash_FrameworkChange(t *testing.T) {
	tmpDir := t.TempDir()

	// Project 1: net8.0
	project1Path := filepath.Join(tmpDir, "project1.csproj")
	content1 := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(project1Path, []byte(content1), 0644)
	require.NoError(t, err)

	proj1, err := project.LoadProject(project1Path)
	require.NoError(t, err)
	hash1, err := CalculateDgSpecHash(proj1)
	require.NoError(t, err)

	// Project 2: net7.0
	project2Path := filepath.Join(tmpDir, "project2.csproj")
	content2 := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net7.0</TargetFramework>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(project2Path, []byte(content2), 0644)
	require.NoError(t, err)

	proj2, err := project.LoadProject(project2Path)
	require.NoError(t, err)
	hash2, err := CalculateDgSpecHash(proj2)
	require.NoError(t, err)

	// Hashes should differ
	assert.NotEqual(t, hash1, hash2)
}

func TestDefaultDgSpecConfig(t *testing.T) {
	cfg := DefaultDgSpecConfig()

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.PackagesPath)
	assert.Contains(t, cfg.PackagesPath, ".nuget")
	assert.Contains(t, cfg.PackagesPath, "packages")

	// Should have default NuGet source
	assert.Len(t, cfg.Sources, 1)
	assert.Equal(t, "https://api.nuget.org/v3/index.json", cfg.Sources[0])

	// ConfigPaths should be empty (initialized but empty)
	assert.NotNil(t, cfg.ConfigPaths)
	assert.Empty(t, cfg.ConfigPaths)
}

func TestGetDefaultRuntimeIDPath(t *testing.T) {
	path := getDefaultRuntimeIDPath()
	// Path may be empty if dotnet SDK is not installed
	// Just verify it doesn't panic
	_ = path
}

func TestDgSpecHasher_WithDownloadDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(content), 0644)
	require.NoError(t, err)

	proj, err := project.LoadProject(projectPath)
	require.NoError(t, err)

	downloadDeps := map[string]map[string]string{
		"net8.0": {
			"Test.Package": "1.0.0",
		},
	}

	hasher := NewDgSpecHasher(proj).
		WithPackagesPath("/tmp/packages").
		WithDownloadDependencies(downloadDeps)

	json, err := hasher.GenerateJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, json)
	jsonStr := string(json)
	assert.Contains(t, jsonStr, "downloadDependencies")
	assert.Contains(t, jsonStr, "Test.Package")
}
