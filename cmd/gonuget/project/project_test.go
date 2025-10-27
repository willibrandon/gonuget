package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProject_SDKStyle(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	projectXML := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectXML), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)
	assert.Equal(t, projectPath, proj.Path)
	assert.True(t, proj.IsSDKStyle())
	assert.Equal(t, "Microsoft.NET.Sdk", proj.Root.Sdk)
	assert.Equal(t, "net8.0", proj.TargetFramework)
	assert.False(t, proj.modified)
}

func TestLoadProject_MultiTFM(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	projectXML := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net6.0;net7.0;net8.0</TargetFrameworks>
  </PropertyGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectXML), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"net6.0", "net7.0", "net8.0"}, proj.TargetFrameworks)
}

func TestLoadProject_WithPackageReferences(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	projectXML := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="System.Text.Json" Version="8.0.0" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(projectPath, []byte(projectXML), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	refs := proj.GetPackageReferences()
	assert.Len(t, refs, 2)
	assert.Equal(t, "Newtonsoft.Json", refs[0].Include)
	assert.Equal(t, "13.0.3", refs[0].Version)
	assert.Equal(t, "System.Text.Json", refs[1].Include)
	assert.Equal(t, "8.0.0", refs[1].Version)
}

func TestLoadProject_InvalidXML(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	invalidXML := `<?xml version="1.0"?>
<Project>
  <Unclosed>`

	err := os.WriteFile(projectPath, []byte(invalidXML), 0644)
	require.NoError(t, err)

	_, err = LoadProject(projectPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse project XML")
}

func TestLoadProject_FileNotFound(t *testing.T) {
	_, err := LoadProject("/nonexistent/path/Test.csproj")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read project file")
}

func TestSave_CreatesValidXML(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	// Create initial project
	proj := &Project{
		Path: projectPath,
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			PropertyGroup: []PropertyGroup{
				{TargetFramework: "net8.0"},
			},
		},
		modified: true,
	}

	err := proj.Save()
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(projectPath)
	require.NoError(t, err)

	// Verify UTF-8 BOM
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, data[:3])

	// Verify XML declaration
	content := string(data[3:])
	assert.Contains(t, content, `<?xml version="1.0" encoding="utf-8"?>`)
	assert.Contains(t, content, `<Project Sdk="Microsoft.NET.Sdk">`)
	assert.Contains(t, content, `<TargetFramework>net8.0</TargetFramework>`)
}

func TestSave_SkipsIfNotModified(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	proj := &Project{
		Path: projectPath,
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
		},
		modified: false,
	}

	err := proj.Save()
	require.NoError(t, err)

	// Verify file was NOT created
	_, err = os.Stat(projectPath)
	assert.True(t, os.IsNotExist(err))
}

func TestAddOrUpdatePackageReference_AddNew(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
		},
		modified: false,
	}

	updated, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", nil)
	require.NoError(t, err)
	assert.False(t, updated) // This is a new addition
	assert.True(t, proj.modified)

	refs := proj.GetPackageReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, "Newtonsoft.Json", refs[0].Include)
	assert.Equal(t, "13.0.3", refs[0].Version)
}

func TestAddOrUpdatePackageReference_UpdateExisting(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			ItemGroups: []ItemGroup{
				{
					PackageReferences: []PackageReference{
						{Include: "Newtonsoft.Json", Version: "12.0.0"},
					},
				},
			},
		},
		modified: false,
	}

	updated, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", nil)
	require.NoError(t, err)
	assert.True(t, updated) // This is an update
	assert.True(t, proj.modified)

	refs := proj.GetPackageReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, "Newtonsoft.Json", refs[0].Include)
	assert.Equal(t, "13.0.3", refs[0].Version)
}

func TestAddOrUpdatePackageReference_CaseInsensitive(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			ItemGroups: []ItemGroup{
				{
					PackageReferences: []PackageReference{
						{Include: "Newtonsoft.Json", Version: "12.0.0"},
					},
				},
			},
		},
	}

	// Update with different casing
	updated, err := proj.AddOrUpdatePackageReference("newtonsoft.json", "13.0.3", nil)
	require.NoError(t, err)
	assert.True(t, updated)

	refs := proj.GetPackageReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, "13.0.3", refs[0].Version)
}

func TestAddOrUpdatePackageReference_WithFrameworks_M21Error(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
		},
	}

	// M2.1: Framework-specific references should error
	_, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", []string{"net8.0"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "framework-specific references not supported in M2.1")
}

func TestAddOrUpdatePackageReference_AddToExistingItemGroup(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			ItemGroups: []ItemGroup{
				{
					PackageReferences: []PackageReference{
						{Include: "Existing.Package", Version: "1.0.0"},
					},
				},
			},
		},
	}

	updated, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", nil)
	require.NoError(t, err)
	assert.False(t, updated)

	// Should be added to existing ItemGroup
	assert.Len(t, proj.Root.ItemGroups, 1)
	assert.Len(t, proj.Root.ItemGroups[0].PackageReferences, 2)
}

func TestRemovePackageReference_Found(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			ItemGroups: []ItemGroup{
				{
					PackageReferences: []PackageReference{
						{Include: "Newtonsoft.Json", Version: "13.0.3"},
						{Include: "System.Text.Json", Version: "8.0.0"},
					},
				},
			},
		},
	}

	removed := proj.RemovePackageReference("Newtonsoft.Json")
	assert.True(t, removed)
	assert.True(t, proj.modified)

	refs := proj.GetPackageReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, "System.Text.Json", refs[0].Include)
}

func TestRemovePackageReference_NotFound(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			ItemGroups: []ItemGroup{
				{
					PackageReferences: []PackageReference{
						{Include: "Newtonsoft.Json", Version: "13.0.3"},
					},
				},
			},
		},
		modified: false,
	}

	removed := proj.RemovePackageReference("NonExistent.Package")
	assert.False(t, removed)
	assert.False(t, proj.modified)
}

func TestIsCentralPackageManagementEnabled_Detected(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")
	cpmPath := filepath.Join(tempDir, "Directory.Packages.props")

	// Create CPM file
	err := os.WriteFile(cpmPath, []byte("<Project></Project>"), 0644)
	require.NoError(t, err)

	proj := &Project{
		Path: projectPath,
		Root: &RootElement{},
	}

	assert.True(t, proj.IsCentralPackageManagementEnabled())
}

func TestIsCentralPackageManagementEnabled_NotDetected(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	proj := &Project{
		Path: projectPath,
		Root: &RootElement{},
	}

	assert.False(t, proj.IsCentralPackageManagementEnabled())
}

func TestFindProjectFile_Single(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.csproj")

	err := os.WriteFile(projectPath, []byte("<Project/>"), 0644)
	require.NoError(t, err)

	found, err := FindProjectFile(tempDir)
	require.NoError(t, err)
	assert.Equal(t, projectPath, found)
}

func TestFindProjectFile_FSharp(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.fsproj")

	err := os.WriteFile(projectPath, []byte("<Project/>"), 0644)
	require.NoError(t, err)

	found, err := FindProjectFile(tempDir)
	require.NoError(t, err)
	assert.Equal(t, projectPath, found)
}

func TestFindProjectFile_VB(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "Test.vbproj")

	err := os.WriteFile(projectPath, []byte("<Project/>"), 0644)
	require.NoError(t, err)

	found, err := FindProjectFile(tempDir)
	require.NoError(t, err)
	assert.Equal(t, projectPath, found)
}

func TestFindProjectFile_Multiple(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "Test1.csproj"), []byte("<Project/>"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "Test2.csproj"), []byte("<Project/>"), 0644)
	require.NoError(t, err)

	_, err = FindProjectFile(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple project files found")
}

func TestFindProjectFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	_, err := FindProjectFile(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project file found")
}

func TestIsSDKStyle(t *testing.T) {
	tests := []struct {
		name     string
		sdk      string
		expected bool
	}{
		{"SDK style", "Microsoft.NET.Sdk", true},
		{"SDK style web", "Microsoft.NET.Sdk.Web", true},
		{"Legacy", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj := &Project{
				Root: &RootElement{
					Sdk: tt.sdk,
				},
			}
			assert.Equal(t, tt.expected, proj.IsSDKStyle())
		})
	}
}
