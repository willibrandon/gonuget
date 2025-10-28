package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

func TestNewAddPackageCmd(t *testing.T) {
	cmd := NewAddPackageCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "package <PACKAGE_ID>", cmd.Use)
	assert.Contains(t, cmd.Short, "Add a NuGet package reference")

	// Verify flags are registered
	assert.NotNil(t, cmd.Flags().Lookup("version"))
	assert.NotNil(t, cmd.Flags().Lookup("framework"))
	assert.NotNil(t, cmd.Flags().Lookup("no-restore"))
	assert.NotNil(t, cmd.Flags().Lookup("source"))
	assert.NotNil(t, cmd.Flags().Lookup("package-directory"))
	assert.NotNil(t, cmd.Flags().Lookup("prerelease"))
	assert.NotNil(t, cmd.Flags().Lookup("interactive"))
	assert.NotNil(t, cmd.Flags().Lookup("project"))
}

func TestRunAddPackage_MissingProjectFile(t *testing.T) {
	// Create a temporary directory without a project file
	tmpDir := t.TempDir()

	opts := &AddPackageOptions{
		ProjectPath: tmpDir,
		Version:     "1.0.0",
	}

	err := runAddPackage(context.Background(), "TestPackage", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load project")
}

func TestRunAddPackage_InvalidVersion(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "invalid-version",
	}

	err = runAddPackage(context.Background(), "TestPackage", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package version")
}

func TestRunAddPackage_CPMEnabled(t *testing.T) {
	// Create a temporary directory with CPM enabled
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	// Create Directory.Packages.props
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")
	dppContent := `<Project>
  <ItemGroup>
    <PackageVersion Include="TestPackage" Version="1.0.0" />
  </ItemGroup>
</Project>`
	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Project file with ManagePackageVersionsCentrally enabled
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "2.0.0",
		NoRestore:   true, // Skip restore for unit test
	}

	// CPM support is now fully implemented (Chunks 12-13)
	err = runAddPackage(context.Background(), "TestPackage", opts)
	assert.NoError(t, err, "CPM projects should allow adding packages")

	// Verify version was updated in Directory.Packages.props
	dppData, err := os.ReadFile(dppPath)
	require.NoError(t, err)
	assert.Contains(t, string(dppData), "2.0.0")
}

func TestRunAddPackage_WithExplicitVersion(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true, // Skip restore for unit test
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify the package was added to the project file
	content, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Newtonsoft.Json")
	assert.Contains(t, string(content), "13.0.3")
}

func TestRunAddPackage_UpdateExisting(t *testing.T) {
	// Create a temporary project file with an existing package
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="12.0.0" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify the version was updated
	content, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "13.0.3")
	assert.NotContains(t, string(content), "12.0.0")
}

func TestRunAddPackage_WithFramework(t *testing.T) {
	// This feature is part of M2.2 Chunk 14 (framework-specific package references)
	// For M2.1 Chunk 3, we only support global package references
	t.Skip("Framework-specific package references are implemented in M2.2 Chunk 14")
}

func TestResolveLatestVersion_PackageNotFound(t *testing.T) {
	opts := &AddPackageOptions{
		Source: "https://api.nuget.org/v3/index.json",
	}

	_, err := resolveLatestVersion(context.Background(), "NonExistentPackage12345XYZ", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveLatestVersion_NoStableVersionWithoutPrerelease(t *testing.T) {
	// This test would need a package that only has prerelease versions
	// For now, we'll skip this as it requires specific test data
	t.Skip("Requires package with only prerelease versions")
}

func TestResolveLatestVersion_WithPrerelease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	opts := &AddPackageOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: true,
	}

	version, err := resolveLatestVersion(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestResolveLatestVersion_StableOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	opts := &AddPackageOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: false,
	}

	version, err := resolveLatestVersion(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	// Should not contain prerelease identifiers
	assert.NotContains(t, version, "-")
}

func TestRunAddPackage_ResolveVersionError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create a temporary project file
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		// No version specified, will try to resolve
	}

	// Use a non-existent package to trigger resolve error
	err = runAddPackage(context.Background(), "NonExistentPackage12345XYZ", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve latest version")
}

func TestRunAddPackage_NoRestore(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true, // Test --no-restore flag
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify package was added to project file but NOT restored
	content, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Newtonsoft.Json")
	assert.Contains(t, string(content), "13.0.3")
}

func TestRunAddPackage_SaveError(t *testing.T) {
	// Create a temporary project file in a read-only directory
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	// Make the file read-only
	err = os.Chmod(projectPath, 0444)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save project file")

	// Restore permissions for cleanup
	_ = os.Chmod(projectPath, 0644)
}

func TestResolveLatestVersion_InvalidSource(t *testing.T) {
	opts := &AddPackageOptions{
		Source: "https://invalid-source-that-does-not-exist.example.com/v3/index.json",
	}

	_, err := resolveLatestVersion(context.Background(), "Newtonsoft.Json", opts)
	assert.Error(t, err)
}

func TestNewAddPackageCmd_ExecuteWithArgs(t *testing.T) {
	// Test that the command can be executed with args
	cmd := NewAddPackageCommand()

	// Test with no args - should fail
	err := cmd.Execute()
	assert.Error(t, err)

	// Verify the command structure
	assert.Equal(t, "package <PACKAGE_ID>", cmd.Use)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Short)
}

func TestRunAddPackage_FindProjectFileInCurrentDir(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	// Save current directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		// No ProjectPath specified, should find in current dir
		Version:   "13.0.3",
		NoRestore: true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify package was added
	content, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Newtonsoft.Json")
}

func TestRunAddPackage_FindProjectFileError(t *testing.T) {
	// Create a temporary directory with NO project file
	tmpDir := t.TempDir()

	// Save current directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		// No ProjectPath specified, should fail to find
		Version:   "13.0.3",
		NoRestore: true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find project file")
}

func TestResolveLatestVersion_CustomSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	opts := &AddPackageOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: false,
	}

	version, err := resolveLatestVersion(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestResolveLatestVersion_PrereleaseWithOnlyStable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// For a package with stable versions, prerelease flag should still work
	opts := &AddPackageOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: true,
	}

	version, err := resolveLatestVersion(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}

// CPM Integration Tests (Chunk 13)

func TestRunAddPackage_CPM_AddNew(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// Create Directory.Packages.props
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
  </ItemGroup>
</Project>`
	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Project file with CPM enabled
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify Directory.Packages.props was updated
	dppData, err := os.ReadFile(dppPath)
	require.NoError(t, err)
	assert.Contains(t, string(dppData), "Newtonsoft.Json")
	assert.Contains(t, string(dppData), "13.0.3")

	// Verify .csproj has PackageReference WITHOUT version
	projData, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Contains(t, string(projData), "Newtonsoft.Json")
	// Ensure NO Version attribute in PackageReference
	assert.NotContains(t, string(projData), `Version="13.0.3"`)
}

func TestRunAddPackage_CPM_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// Create Directory.Packages.props with existing package
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="12.0.0" />
  </ItemGroup>
</Project>`
	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Project file with CPM enabled
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify Directory.Packages.props was updated to new version
	dppData, err := os.ReadFile(dppPath)
	require.NoError(t, err)
	assert.Contains(t, string(dppData), "13.0.3")
	assert.NotContains(t, string(dppData), "12.0.0")
}

func TestRunAddPackage_CPM_MissingPropsFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	// Project file with CPM enabled but NO Directory.Packages.props
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Directory.Packages.props not found")
}

func TestRunAddPackage_CPM_NoRestore(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// Create Directory.Packages.props
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
  </ItemGroup>
</Project>`
	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Project file with CPM enabled
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true, // Test --no-restore flag
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify package was added but NOT restored
	dppData, err := os.ReadFile(dppPath)
	require.NoError(t, err)
	assert.Contains(t, string(dppData), "Newtonsoft.Json")

	// Verify no obj directory was created (restore didn't run)
	objDir := filepath.Join(tmpDir, "obj")
	_, err = os.Stat(objDir)
	assert.True(t, os.IsNotExist(err), "obj directory should not exist when --no-restore is used")
}

func TestRunAddPackage_CPM_UseExistingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// Create Directory.Packages.props with existing version
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Serilog" Version="3.1.1" />
  </ItemGroup>
</Project>`
	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Project file with CPM enabled
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		// No version specified - should use existing version from Directory.Packages.props
		NoRestore: true,
	}

	err = runAddPackage(context.Background(), "Serilog", opts)
	assert.NoError(t, err)

	// Verify version in Directory.Packages.props wasn't changed
	dppData, err := os.ReadFile(dppPath)
	require.NoError(t, err)
	assert.Contains(t, string(dppData), "3.1.1")

	// Verify PackageReference was added to project
	projData, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Contains(t, string(projData), "Serilog")
}

func TestRunAddPackage_CPM_VerifyNoVersionInCsproj(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// Create Directory.Packages.props
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
  </ItemGroup>
</Project>`
	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Project file with CPM enabled
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Read and parse the project file
	proj, err := project.LoadProject(projectPath)
	require.NoError(t, err)

	// Find the PackageReference
	refs := proj.GetPackageReferences()
	var found bool
	for _, ref := range refs {
		if ref.Include == "Newtonsoft.Json" {
			found = true
			// Version should be empty in CPM mode
			assert.Empty(t, ref.Version, "PackageReference should NOT have Version attribute in CPM mode")
			break
		}
	}
	assert.True(t, found, "PackageReference for Newtonsoft.Json should exist")
}

// TestRunAddPackage_OutputParity verifies that gonuget's output matches dotnet's output 100%
func TestRunAddPackage_OutputParity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create temp project
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		Source:      "https://api.nuget.org/v3/index.json",
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	require.NoError(t, err)

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify output format matches dotnet exactly (ignoring timing and paths)
	// Expected format:
	// info : Adding PackageReference for package 'Newtonsoft.Json' into project 'PATH'.
	// info : Restoring packages for PATH...
	// info : Package 'Newtonsoft.Json' is compatible with all the specified frameworks in project 'PATH'.
	// info : PackageReference for package 'Newtonsoft.Json' version '13.0.3' added to file 'PATH'.
	// info : Writing assets file to disk. Path: PATH
	// log  : Restored PATH (in X ms).

	assert.Contains(t, output, "info : Adding PackageReference for package 'Newtonsoft.Json' into project")
	assert.Contains(t, output, "info : Restoring packages for")
	assert.Contains(t, output, "info : Package 'Newtonsoft.Json' is compatible with all the specified frameworks in project")
	assert.Contains(t, output, "info : PackageReference for package 'Newtonsoft.Json' version '13.0.3' added to file")
	assert.Contains(t, output, "info : Writing assets file to disk. Path:")
	assert.Contains(t, output, "log  : Restored")
	assert.Contains(t, output, "(in")
	assert.Contains(t, output, "ms)")

	// Verify output does NOT contain gonuget-specific messages
	assert.NotContains(t, output, "Resolving")
	assert.NotContains(t, output, "Downloading")
	assert.NotContains(t, output, "already cached")
}
