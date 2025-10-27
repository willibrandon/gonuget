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

func TestAddOrUpdatePackageReference_WithFramework(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			PropertyGroup: []PropertyGroup{
				{TargetFrameworks: "net8.0;net48"},
			},
		},
		TargetFrameworks: []string{"net8.0", "net48"},
	}

	// Add package for specific framework
	updated, err := proj.AddOrUpdatePackageReference("System.Drawing.Common", "8.0.0", []string{"net48"})
	require.NoError(t, err)
	assert.False(t, updated) // New addition

	// Verify conditional ItemGroup was created
	require.Len(t, proj.Root.ItemGroups, 1)
	assert.Equal(t, "'$(TargetFramework)' == 'net48'", proj.Root.ItemGroups[0].Condition)
	require.Len(t, proj.Root.ItemGroups[0].PackageReferences, 1)
	assert.Equal(t, "System.Drawing.Common", proj.Root.ItemGroups[0].PackageReferences[0].Include)
	assert.Equal(t, "8.0.0", proj.Root.ItemGroups[0].PackageReferences[0].Version)

	// Add another package for multiple frameworks
	// Matching dotnet behavior: Creates SEPARATE conditional ItemGroups (one per framework)
	// But reuses existing ItemGroup if condition matches
	updated, err = proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", []string{"net8.0", "net48"})
	require.NoError(t, err)
	assert.False(t, updated)

	// Verify ItemGroups: net48 (System.Drawing.Common + Newtonsoft.Json), net8.0 (Newtonsoft.Json)
	require.Len(t, proj.Root.ItemGroups, 2)

	// First ItemGroup: net48 with both packages
	assert.Equal(t, "'$(TargetFramework)' == 'net48'", proj.Root.ItemGroups[0].Condition)
	require.Len(t, proj.Root.ItemGroups[0].PackageReferences, 2)
	assert.Equal(t, "System.Drawing.Common", proj.Root.ItemGroups[0].PackageReferences[0].Include)
	assert.Equal(t, "Newtonsoft.Json", proj.Root.ItemGroups[0].PackageReferences[1].Include)

	// Second ItemGroup: net8.0 with Newtonsoft.Json
	assert.Equal(t, "'$(TargetFramework)' == 'net8.0'", proj.Root.ItemGroups[1].Condition)
	require.Len(t, proj.Root.ItemGroups[1].PackageReferences, 1)
	assert.Equal(t, "Newtonsoft.Json", proj.Root.ItemGroups[1].PackageReferences[0].Include)
}

func TestAddOrUpdatePackageReference_WithFramework_UpdateExisting(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			PropertyGroup: []PropertyGroup{
				{TargetFrameworks: "net8.0;net48"},
			},
			ItemGroups: []ItemGroup{
				{
					Condition: "'$(TargetFramework)' == 'net48'",
					PackageReferences: []PackageReference{
						{Include: "System.Drawing.Common", Version: "7.0.0"},
					},
				},
			},
		},
		TargetFrameworks: []string{"net8.0", "net48"},
	}

	// Update existing framework-specific package
	updated, err := proj.AddOrUpdatePackageReference("System.Drawing.Common", "8.0.0", []string{"net48"})
	require.NoError(t, err)
	assert.True(t, updated) // This is an update

	// Verify version was updated
	require.Len(t, proj.Root.ItemGroups, 1)
	assert.Equal(t, "8.0.0", proj.Root.ItemGroups[0].PackageReferences[0].Version)
}

func TestAddOrUpdatePackageReference_WithFramework_InvalidFramework(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
		},
	}

	// Invalid framework should error
	_, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", []string{"invalid-tfm"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target framework")
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

func TestIsCentralPackageManagementEnabled(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "CPM enabled",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`,
			expected: true,
		},
		{
			name: "CPM enabled case insensitive",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <ManagePackageVersionsCentrally>True</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`,
			expected: true,
		},
		{
			name: "CPM not enabled",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`,
			expected: false,
		},
		{
			name: "CPM explicitly disabled",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <ManagePackageVersionsCentrally>false</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			projectPath := filepath.Join(tempDir, "Test.csproj")

			err := os.WriteFile(projectPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			proj, err := LoadProject(projectPath)
			require.NoError(t, err)

			got := proj.IsCentralPackageManagementEnabled()
			assert.Equal(t, tt.expected, got, "CPM detection mismatch")
		})
	}
}

func TestGetDirectoryPackagesPropsPath(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		createPropsFile  bool
		propsInParent    bool
		expectedBaseName string
	}{
		{
			name: "Default location",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`,
			createPropsFile:  false,
			propsInParent:    false,
			expectedBaseName: "Directory.Packages.props",
		},
		{
			name: "Props file exists in project directory",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`,
			createPropsFile:  true,
			propsInParent:    false,
			expectedBaseName: "Directory.Packages.props",
		},
		{
			name: "Props file in parent directory",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>`,
			createPropsFile:  false,
			propsInParent:    true,
			expectedBaseName: "Directory.Packages.props",
		},
		{
			name: "Custom path specified",
			content: `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
    <DirectoryPackagesPropsPath>custom/packages.props</DirectoryPackagesPropsPath>
  </PropertyGroup>
</Project>`,
			createPropsFile:  false,
			propsInParent:    false,
			expectedBaseName: "packages.props",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "src")
			err := os.MkdirAll(projectDir, 0755)
			require.NoError(t, err)

			projectPath := filepath.Join(projectDir, "Test.csproj")
			err = os.WriteFile(projectPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			if tt.createPropsFile {
				propsPath := filepath.Join(projectDir, "Directory.Packages.props")
				err = os.WriteFile(propsPath, []byte("<Project></Project>"), 0644)
				require.NoError(t, err)
			}

			if tt.propsInParent {
				propsPath := filepath.Join(tempDir, "Directory.Packages.props")
				err = os.WriteFile(propsPath, []byte("<Project></Project>"), 0644)
				require.NoError(t, err)
			}

			proj, err := LoadProject(projectPath)
			require.NoError(t, err)

			got := proj.GetDirectoryPackagesPropsPath()
			assert.Equal(t, tt.expectedBaseName, filepath.Base(got), "Props path base name mismatch")

			// For non-custom paths, verify the path is in expected directory
			if tt.name != "Custom path specified" {
				if tt.propsInParent {
					assert.Equal(t, tempDir, filepath.Dir(got), "Props should be in parent directory")
				} else {
					assert.Equal(t, projectDir, filepath.Dir(got), "Props should be in project directory")
				}
			}
		})
	}
}

// Chunk 14: Framework-specific reference helper tests

func TestBuildFrameworkCondition(t *testing.T) {
	tests := []struct {
		name       string
		frameworks []string
		expected   string
	}{
		{
			name:       "Empty frameworks",
			frameworks: []string{},
			expected:   "",
		},
		{
			name:       "Single framework",
			frameworks: []string{"net8.0"},
			expected:   "'$(TargetFramework)' == 'net8.0'",
		},
		{
			name:       "Multiple frameworks",
			frameworks: []string{"net8.0", "net48"},
			expected:   "'$(TargetFramework)' == 'net8.0' OR '$(TargetFramework)' == 'net48'",
		},
		{
			name:       "Three frameworks",
			frameworks: []string{"net6.0", "net7.0", "net8.0"},
			expected:   "'$(TargetFramework)' == 'net6.0' OR '$(TargetFramework)' == 'net7.0' OR '$(TargetFramework)' == 'net8.0'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFrameworkCondition(tt.frameworks)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizeCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		expected  string
	}{
		{
			name:      "Empty condition",
			condition: "",
			expected:  "",
		},
		{
			name:      "Single quotes",
			condition: "'$(TargetFramework)' == 'net8.0'",
			expected:  "'$(targetframework)' == 'net8.0'",
		},
		{
			name:      "Double quotes",
			condition: `"$(TargetFramework)" == "net8.0"`,
			expected:  "'$(targetframework)' == 'net8.0'",
		},
		{
			name:      "Extra whitespace",
			condition: "  '$(TargetFramework)'   ==   'net8.0'  ",
			expected:  "'$(targetframework)' == 'net8.0'",
		},
		{
			name:      "Mixed case",
			condition: "'$(TARGETFRAMEWORK)' == 'NET8.0'",
			expected:  "'$(targetframework)' == 'net8.0'",
		},
		{
			name:      "OR condition",
			condition: "'$(TargetFramework)' == 'net8.0' OR '$(TargetFramework)' == 'net48'",
			expected:  "'$(targetframework)' == 'net8.0' or '$(targetframework)' == 'net48'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCondition(tt.condition)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetTargetFrameworks_Single(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			PropertyGroup: []PropertyGroup{
				{TargetFramework: "net8.0"},
			},
		},
		TargetFramework: "net8.0",
	}

	frameworks := proj.GetTargetFrameworks()
	assert.Equal(t, []string{"net8.0"}, frameworks)
}

func TestGetTargetFrameworks_Multiple(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			PropertyGroup: []PropertyGroup{
				{TargetFrameworks: "net6.0;net7.0;net8.0"},
			},
		},
		TargetFrameworks: []string{"net6.0", "net7.0", "net8.0"},
	}

	frameworks := proj.GetTargetFrameworks()
	assert.Equal(t, []string{"net6.0", "net7.0", "net8.0"}, frameworks)
}

func TestGetTargetFrameworks_None(t *testing.T) {
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
		},
	}

	frameworks := proj.GetTargetFrameworks()
	assert.Empty(t, frameworks)
}

func TestGetTargetFrameworks_FallbackFromPropertyGroup(t *testing.T) {
	// Test fallback when cached fields are not populated
	proj := &Project{
		Root: &RootElement{
			Sdk: "Microsoft.NET.Sdk",
			PropertyGroup: []PropertyGroup{
				{TargetFramework: "net8.0"},
			},
		},
		// Note: TargetFramework field not set (simulates load without cache)
	}

	frameworks := proj.GetTargetFrameworks()
	assert.Equal(t, []string{"net8.0"}, frameworks)
}

func TestIsMultiTargeting(t *testing.T) {
	tests := []struct {
		name             string
		targetFramework  string
		targetFrameworks []string
		expected         bool
	}{
		{
			name:            "Single framework",
			targetFramework: "net8.0",
			expected:        false,
		},
		{
			name:             "Multiple frameworks",
			targetFrameworks: []string{"net6.0", "net7.0", "net8.0"},
			expected:         true,
		},
		{
			name:             "Two frameworks",
			targetFrameworks: []string{"net8.0", "net48"},
			expected:         true,
		},
		{
			name:     "No frameworks",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj := &Project{
				Root: &RootElement{
					Sdk: "Microsoft.NET.Sdk",
				},
				TargetFramework:  tt.targetFramework,
				TargetFrameworks: tt.targetFrameworks,
			}

			got := proj.IsMultiTargeting()
			assert.Equal(t, tt.expected, got)
		})
	}
}
