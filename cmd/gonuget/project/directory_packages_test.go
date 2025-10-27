package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDirectoryPackagesProps(t *testing.T) {
	// Create a temporary Directory.Packages.props file
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <PropertyGroup>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageVersion Include="Serilog" Version="3.1.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Test loading
	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)
	assert.NotNil(t, dpp)
	assert.Equal(t, dppPath, dpp.Path)
	assert.False(t, dpp.modified)

	// Verify structure
	assert.Len(t, dpp.Root.ItemGroups, 1)
	assert.Len(t, dpp.Root.ItemGroups[0].PackageVersions, 2)
	assert.Equal(t, "Newtonsoft.Json", dpp.Root.ItemGroups[0].PackageVersions[0].Include)
	assert.Equal(t, "13.0.3", dpp.Root.ItemGroups[0].PackageVersions[0].Version)
	assert.Equal(t, "Serilog", dpp.Root.ItemGroups[0].PackageVersions[1].Include)
	assert.Equal(t, "3.1.1", dpp.Root.ItemGroups[0].PackageVersions[1].Version)
}

func TestLoadDirectoryPackagesProps_FileNotFound(t *testing.T) {
	dpp, err := LoadDirectoryPackagesProps("/nonexistent/Directory.Packages.props")
	assert.Error(t, err)
	assert.Nil(t, dpp)
	assert.Contains(t, err.Error(), "failed to read Directory.Packages.props")
}

func TestLoadDirectoryPackagesProps_InvalidXML(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	err := os.WriteFile(dppPath, []byte("invalid xml content"), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	assert.Error(t, err)
	assert.Nil(t, dpp)
	assert.Contains(t, err.Error(), "failed to parse Directory.Packages.props")
}

func TestLoadDirectoryPackagesProps_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)
	assert.NotNil(t, dpp)
	assert.Len(t, dpp.Root.ItemGroups, 0)
}

func TestAddOrUpdatePackageVersion_Add(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Add new package
	updated, err := dpp.AddOrUpdatePackageVersion("Serilog", "3.1.1")
	require.NoError(t, err)
	assert.False(t, updated, "Should return false when adding new package")
	assert.True(t, dpp.modified)

	// Verify package was added
	assert.Len(t, dpp.Root.ItemGroups[0].PackageVersions, 2)
	assert.Equal(t, "Serilog", dpp.Root.ItemGroups[0].PackageVersions[1].Include)
	assert.Equal(t, "3.1.1", dpp.Root.ItemGroups[0].PackageVersions[1].Version)
}

func TestAddOrUpdatePackageVersion_Update(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Update existing package
	updated, err := dpp.AddOrUpdatePackageVersion("Newtonsoft.Json", "13.0.4")
	require.NoError(t, err)
	assert.True(t, updated, "Should return true when updating existing package")
	assert.True(t, dpp.modified)

	// Verify package was updated
	assert.Len(t, dpp.Root.ItemGroups[0].PackageVersions, 1)
	assert.Equal(t, "Newtonsoft.Json", dpp.Root.ItemGroups[0].PackageVersions[0].Include)
	assert.Equal(t, "13.0.4", dpp.Root.ItemGroups[0].PackageVersions[0].Version)
}

func TestAddOrUpdatePackageVersion_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Update with different case
	updated, err := dpp.AddOrUpdatePackageVersion("NEWTONSOFT.JSON", "13.0.4")
	require.NoError(t, err)
	assert.True(t, updated, "Should match case-insensitively")

	// Verify package was updated (case preserved)
	assert.Equal(t, "Newtonsoft.Json", dpp.Root.ItemGroups[0].PackageVersions[0].Include)
	assert.Equal(t, "13.0.4", dpp.Root.ItemGroups[0].PackageVersions[0].Version)
}

func TestAddOrUpdatePackageVersion_CreateItemGroup(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// File with no ItemGroups
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Add package (should create ItemGroup)
	updated, err := dpp.AddOrUpdatePackageVersion("Newtonsoft.Json", "13.0.3")
	require.NoError(t, err)
	assert.False(t, updated)

	// Verify ItemGroup was created
	assert.Len(t, dpp.Root.ItemGroups, 1)
	assert.Len(t, dpp.Root.ItemGroups[0].PackageVersions, 1)
	assert.Equal(t, "Newtonsoft.Json", dpp.Root.ItemGroups[0].PackageVersions[0].Include)
}

func TestGetPackageVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageVersion Include="Serilog" Version="3.1.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Test getting existing packages
	version := dpp.GetPackageVersion("Newtonsoft.Json")
	assert.Equal(t, "13.0.3", version)

	version = dpp.GetPackageVersion("Serilog")
	assert.Equal(t, "3.1.1", version)

	// Test getting non-existent package
	version = dpp.GetPackageVersion("NonExistent")
	assert.Equal(t, "", version)
}

func TestGetPackageVersion_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Test case-insensitive matching
	version := dpp.GetPackageVersion("NEWTONSOFT.JSON")
	assert.Equal(t, "13.0.3", version)

	version = dpp.GetPackageVersion("newtonsoft.json")
	assert.Equal(t, "13.0.3", version)
}

func TestRemovePackageVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageVersion Include="Serilog" Version="3.1.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Remove existing package
	removed := dpp.RemovePackageVersion("Newtonsoft.Json")
	assert.True(t, removed)
	assert.True(t, dpp.modified)

	// Verify package was removed
	assert.Len(t, dpp.Root.ItemGroups[0].PackageVersions, 1)
	assert.Equal(t, "Serilog", dpp.Root.ItemGroups[0].PackageVersions[0].Include)
}

func TestRemovePackageVersion_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Try to remove non-existent package
	removed := dpp.RemovePackageVersion("NonExistent")
	assert.False(t, removed)
	assert.False(t, dpp.modified)
}

func TestRemovePackageVersion_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Remove with different case
	removed := dpp.RemovePackageVersion("NEWTONSOFT.JSON")
	assert.True(t, removed)
}

func TestSave_DirectoryPackagesProps(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Modify and save
	_, err = dpp.AddOrUpdatePackageVersion("Serilog", "3.1.1")
	require.NoError(t, err)

	err = dpp.Save()
	require.NoError(t, err)
	assert.False(t, dpp.modified, "modified flag should be reset after save")

	// Read back and verify
	content, err := os.ReadFile(dppPath)
	require.NoError(t, err)

	// Verify UTF-8 BOM
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, content[0:3], "File should start with UTF-8 BOM")

	// Verify XML content
	assert.Contains(t, string(content), "Newtonsoft.Json")
	assert.Contains(t, string(content), "13.0.3")
	assert.Contains(t, string(content), "Serilog")
	assert.Contains(t, string(content), "3.1.1")
}

func TestSave_DirectoryPackagesProps_NoModification(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	// Get original modification time
	originalInfo, err := os.Stat(dppPath)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Save without modifications
	err = dpp.Save()
	require.NoError(t, err)

	// Verify file was not modified
	newInfo, err := os.Stat(dppPath)
	require.NoError(t, err)
	assert.Equal(t, originalInfo.ModTime(), newInfo.ModTime(), "File should not be modified")
}

func TestSave_DirectoryPackagesProps_ReadOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Make file read-only
	err = os.Chmod(dppPath, 0444)
	require.NoError(t, err)

	// Modify and try to save
	_, err = dpp.AddOrUpdatePackageVersion("Serilog", "3.1.1")
	require.NoError(t, err)

	err = dpp.Save()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create file")

	// Restore permissions for cleanup
	_ = os.Chmod(dppPath, 0644)
}

func TestSave_DirectoryPackagesProps_MultipleItemGroups(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// File with multiple ItemGroups
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
  <ItemGroup>
    <PackageVersion Include="Serilog" Version="3.1.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Verify multiple ItemGroups were loaded
	assert.Len(t, dpp.Root.ItemGroups, 2)

	// Update package in second ItemGroup
	updated, err := dpp.AddOrUpdatePackageVersion("Serilog", "3.2.0")
	require.NoError(t, err)
	assert.True(t, updated)

	// Save
	err = dpp.Save()
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(dppPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "3.2.0")
}

func TestFindOrCreateItemGroup(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// File with existing ItemGroup
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Should return existing ItemGroup
	ig := dpp.findOrCreateItemGroup()
	assert.NotNil(t, ig)
	assert.Len(t, dpp.Root.ItemGroups, 1, "Should not create new ItemGroup")
	assert.Len(t, ig.PackageVersions, 1)
}

func TestFindOrCreateItemGroup_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	dppPath := filepath.Join(tmpDir, "Directory.Packages.props")

	// File with no ItemGroups
	dppContent := `<?xml version="1.0" encoding="utf-8"?>
<Project>
</Project>`

	err := os.WriteFile(dppPath, []byte(dppContent), 0644)
	require.NoError(t, err)

	dpp, err := LoadDirectoryPackagesProps(dppPath)
	require.NoError(t, err)

	// Should create new ItemGroup
	ig := dpp.findOrCreateItemGroup()
	assert.NotNil(t, ig)
	assert.Len(t, dpp.Root.ItemGroups, 1, "Should create new ItemGroup")
	assert.Len(t, ig.PackageVersions, 0)
}
