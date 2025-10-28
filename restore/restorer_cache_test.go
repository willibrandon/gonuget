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
	assert.Equal(t, projectPath, cache.ProjectFilePath)
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
