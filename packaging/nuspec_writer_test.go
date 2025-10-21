package packaging

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

func TestGenerateNuspecXML_Minimal(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test description",
		Authors:     []string{"Test Author"},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify XML declaration
	if !strings.HasPrefix(xmlStr, "<?xml version=\"1.0\"") {
		t.Error("XML should start with declaration")
	}

	// Verify required fields
	if !strings.Contains(xmlStr, "<id>TestPackage</id>") {
		t.Error("XML should contain package ID")
	}

	if !strings.Contains(xmlStr, "<version>1.0.0</version>") {
		t.Error("XML should contain version")
	}

	if !strings.Contains(xmlStr, "<description>Test description</description>") {
		t.Error("XML should contain description")
	}

	if !strings.Contains(xmlStr, "<authors>Test Author</authors>") {
		t.Error("XML should contain authors")
	}

	// Verify namespace (checking for xmlns attribute in the package tag)
	// Minimal package with stable version should use V6 (most recent/default)
	expectedNamespace := `xmlns="` + NuspecNamespaceV6 + `"`
	if !strings.Contains(xmlStr, expectedNamespace) {
		t.Errorf("XML should contain v6 namespace: %s\nGot XML:\n%s", NuspecNamespaceV6, xmlStr)
	}
}

func TestGenerateNuspecXML_MultipleAuthors(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Author1", "Author2", "Author3"},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	if !strings.Contains(xmlStr, "<authors>Author1, Author2, Author3</authors>") {
		t.Error("XML should contain comma-separated authors")
	}
}

func TestGenerateNuspecXML_OptionalFields(t *testing.T) {
	metadata := PackageMetadata{
		ID:                       "TestPackage",
		Version:                  version.MustParse("1.2.3-beta"),
		Description:              "Test description",
		Authors:                  []string{"Test Author"},
		Title:                    "Test Title",
		Owners:                   []string{"Owner1", "Owner2"},
		Summary:                  "Test summary",
		ReleaseNotes:             "Release notes",
		Copyright:                "Copyright 2025",
		Language:                 "en-US",
		Tags:                     []string{"tag1", "tag2", "tag3"},
		Icon:                     "icon.png",
		Readme:                   "README.md",
		RequireLicenseAcceptance: true,
		DevelopmentDependency:    true,
		Serviceable:              true,
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	expectedElements := map[string]string{
		"title":                    "Test Title",
		"owners":                   "Owner1, Owner2",
		"summary":                  "Test summary",
		"releaseNotes":             "Release notes",
		"copyright":                "Copyright 2025",
		"language":                 "en-US",
		"tags":                     "tag1 tag2 tag3",
		"icon":                     "icon.png",
		"readme":                   "README.md",
		"requireLicenseAcceptance": "true",
		"developmentDependency":    "true",
		"serviceable":              "true",
	}

	for element, value := range expectedElements {
		expected := "<" + element + ">" + value + "</" + element + ">"
		if !strings.Contains(xmlStr, expected) {
			t.Errorf("XML should contain %s", expected)
		}
	}
}

func TestGenerateNuspecXML_WithDependencies(t *testing.T) {
	net60 := frameworks.MustParseFramework("net6.0")
	net48 := frameworks.MustParseFramework("net48")

	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: net60,
				Dependencies: []PackageDependency{
					{
						ID:           "Newtonsoft.Json",
						VersionRange: version.MustParseRange("[13.0.0,)"),
					},
					{
						ID:           "System.Text.Json",
						VersionRange: version.MustParseRange("[6.0.0,)"),
					},
				},
			},
			{
				TargetFramework: net48,
				Dependencies: []PackageDependency{
					{
						ID:           "Newtonsoft.Json",
						VersionRange: version.MustParseRange("[13.0.0,)"),
					},
				},
			},
		},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify dependencies section exists
	if !strings.Contains(xmlStr, "<dependencies>") {
		t.Error("XML should contain dependencies section")
	}

	// Verify dependency groups
	if !strings.Contains(xmlStr, "targetFramework=\"net6.0\"") {
		t.Error("XML should contain net6.0 dependency group")
	}

	if !strings.Contains(xmlStr, "targetFramework=\"net48\"") {
		t.Error("XML should contain net48 dependency group")
	}

	// Verify dependency packages
	if !strings.Contains(xmlStr, "id=\"Newtonsoft.Json\"") {
		t.Error("XML should contain Newtonsoft.Json dependency")
	}

	if !strings.Contains(xmlStr, "id=\"System.Text.Json\"") {
		t.Error("XML should contain System.Text.Json dependency")
	}

	// Verify version ranges
	if !strings.Contains(xmlStr, "version=\"[13.0.0, )\"") {
		t.Errorf("XML should contain version range [13.0.0, )\nGot XML:\n%s", xmlStr)
	}
}

func TestGenerateNuspecXML_WithFrameworkReferences(t *testing.T) {
	net60 := frameworks.MustParseFramework("net6.0")

	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
		FrameworkReferenceGroups: []PackageFrameworkReferenceGroup{
			{
				TargetFramework: net60,
				References: []string{
					"Microsoft.AspNetCore.App",
					"Microsoft.NETCore.App",
				},
			},
		},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify framework references section
	if !strings.Contains(xmlStr, "<frameworkReferences>") {
		t.Error("XML should contain frameworkReferences section")
	}

	// Verify framework reference group
	if !strings.Contains(xmlStr, "targetFramework=\"net6.0\"") {
		t.Error("XML should contain net6.0 framework reference group")
	}

	// Verify framework references
	if !strings.Contains(xmlStr, "name=\"Microsoft.AspNetCore.App\"") {
		t.Error("XML should contain Microsoft.AspNetCore.App reference")
	}

	if !strings.Contains(xmlStr, "name=\"Microsoft.NETCore.App\"") {
		t.Error("XML should contain Microsoft.NETCore.App reference")
	}
}

func TestGenerateNuspecXML_WithPackageTypes(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
		PackageTypes: []PackageTypeInfo{
			{
				Name:    "Dependency",
				Version: version.MustParse("1.0.0"),
			},
			{
				Name: "DotnetTool",
			},
		},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify package types
	if !strings.Contains(xmlStr, "name=\"Dependency\"") {
		t.Error("XML should contain Dependency package type")
	}

	if !strings.Contains(xmlStr, "name=\"DotnetTool\"") {
		t.Error("XML should contain DotnetTool package type")
	}

	if !strings.Contains(xmlStr, "version=\"1.0.0\"") {
		t.Error("XML should contain package type version")
	}
}

func TestGenerateNuspecXML_WithRepository(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
		Repository: &PackageRepositoryMetadata{
			Type:   "git",
			URL:    "https://github.com/example/repo",
			Branch: "main",
			Commit: "abc123def456",
		},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify repository metadata
	if !strings.Contains(xmlStr, "type=\"git\"") {
		t.Error("XML should contain repository type")
	}

	if !strings.Contains(xmlStr, "url=\"https://github.com/example/repo\"") {
		t.Error("XML should contain repository URL")
	}

	if !strings.Contains(xmlStr, "branch=\"main\"") {
		t.Error("XML should contain repository branch")
	}

	if !strings.Contains(xmlStr, "commit=\"abc123def456\"") {
		t.Error("XML should contain repository commit")
	}
}

func TestGenerateNuspecXML_WithLicense(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
		LicenseMetadata: &LicenseMetadata{
			Type:    "expression",
			Text:    "MIT",
			Version: "1.0.0",
		},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify license metadata
	if !strings.Contains(xmlStr, "type=\"expression\"") {
		t.Error("XML should contain license type")
	}

	if !strings.Contains(xmlStr, "version=\"1.0.0\"") {
		t.Error("XML should contain license version")
	}

	if !strings.Contains(xmlStr, ">MIT<") {
		t.Error("XML should contain license text")
	}
}

func TestGenerateNuspecXML_WithMinClientVersion(t *testing.T) {
	metadata := PackageMetadata{
		ID:               "TestPackage",
		Version:          version.MustParse("1.0.0"),
		Description:      "Test",
		Authors:          []string{"Test Author"},
		MinClientVersion: version.MustParse("5.0.0"),
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	if !strings.Contains(xmlStr, "minClientVersion=\"5.0.0\"") {
		t.Error("XML should contain minClientVersion attribute")
	}
}

func TestGenerateNuspecXML_WithDependencyIncludeExclude(t *testing.T) {
	net60 := frameworks.MustParseFramework("net6.0")

	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: net60,
				Dependencies: []PackageDependency{
					{
						ID:           "Newtonsoft.Json",
						VersionRange: version.MustParseRange("[13.0.0,)"),
						Include:      []string{"Compile", "Runtime"},
						Exclude:      []string{"Build", "Native"},
					},
				},
			},
		},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	xmlStr := string(xmlBytes)

	// Verify include/exclude attributes
	if !strings.Contains(xmlStr, "include=\"Compile;Runtime\"") {
		t.Error("XML should contain include attribute")
	}

	if !strings.Contains(xmlStr, "exclude=\"Build;Native\"") {
		t.Error("XML should contain exclude attribute")
	}
}

func TestGenerateNuspecXML_ValidXML(t *testing.T) {
	// Test that generated XML is valid and can be parsed
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.0.0"),
		Description: "Test",
		Authors:     []string{"Test Author"},
	}

	xmlBytes, err := GenerateNuspecXML(metadata)
	if err != nil {
		t.Fatalf("GenerateNuspecXML() error = %v", err)
	}

	// Try to parse the generated XML
	var nuspec Nuspec
	err = xml.Unmarshal(xmlBytes, &nuspec)
	if err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Verify parsed values
	if nuspec.Metadata.ID != "TestPackage" {
		t.Errorf("Parsed ID = %q, want TestPackage", nuspec.Metadata.ID)
	}

	if nuspec.Metadata.Version != "1.0.0" {
		t.Errorf("Parsed Version = %q, want 1.0.0", nuspec.Metadata.Version)
	}
}

func TestDetermineNuspecNamespace(t *testing.T) {
	tests := []struct {
		name     string
		metadata PackageMetadata
		expected string
	}{
		{
			name: "V6 default - stable version, no frameworks",
			metadata: PackageMetadata{
				ID:          "TestPackage",
				Version:     version.MustParse("1.0.0"),
				Description: "Test",
				Authors:     []string{"Test Author"},
			},
			expected: NuspecNamespaceV6,
		},
		{
			name: "V3 - prerelease version",
			metadata: PackageMetadata{
				ID:          "TestPackage",
				Version:     version.MustParse("1.0.0-beta"),
				Description: "Test",
				Authors:     []string{"Test Author"},
			},
			expected: NuspecNamespaceV3,
		},
		{
			name: "V4 - dependencies with target framework",
			metadata: PackageMetadata{
				ID:          "TestPackage",
				Version:     version.MustParse("1.0.0"),
				Description: "Test",
				Authors:     []string{"Test Author"},
				DependencyGroups: []PackageDependencyGroup{
					{
						TargetFramework: frameworks.MustParseFramework("net6.0"),
						Dependencies: []PackageDependency{
							{ID: "Newtonsoft.Json"},
						},
					},
				},
			},
			expected: NuspecNamespaceV4,
		},
		{
			name: "V5 - framework references with target framework",
			metadata: PackageMetadata{
				ID:          "TestPackage",
				Version:     version.MustParse("1.0.0"),
				Description: "Test",
				Authors:     []string{"Test Author"},
				FrameworkReferenceGroups: []PackageFrameworkReferenceGroup{
					{
						TargetFramework: frameworks.MustParseFramework("net6.0"),
						References:      []string{"Microsoft.AspNetCore.App"},
					},
				},
			},
			expected: NuspecNamespaceV5,
		},
		{
			name: "V5 overrides V4 - both references and dependencies",
			metadata: PackageMetadata{
				ID:          "TestPackage",
				Version:     version.MustParse("1.0.0"),
				Description: "Test",
				Authors:     []string{"Test Author"},
				DependencyGroups: []PackageDependencyGroup{
					{
						TargetFramework: frameworks.MustParseFramework("net6.0"),
						Dependencies:    []PackageDependency{{ID: "Package1"}},
					},
				},
				FrameworkReferenceGroups: []PackageFrameworkReferenceGroup{
					{
						TargetFramework: frameworks.MustParseFramework("net6.0"),
						References:      []string{"Microsoft.AspNetCore.App"},
					},
				},
			},
			expected: NuspecNamespaceV5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace := determineNuspecNamespace(tt.metadata)
			if namespace != tt.expected {
				t.Errorf("determineNuspecNamespace() = %s, want %s", namespace, tt.expected)
			}
		})
	}
}

func TestBuildNuspecStructure(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     version.MustParse("1.2.3"),
		Description: "Test description",
		Authors:     []string{"Author1", "Author2"},
	}

	nuspec := buildNuspecStructure(metadata, NuspecNamespaceV5)

	if nuspec == nil {
		t.Fatal("buildNuspecStructure() returned nil")
	}

	if nuspec.Xmlns != NuspecNamespaceV5 {
		t.Errorf("Xmlns = %s, want %s", nuspec.Xmlns, NuspecNamespaceV5)
	}

	if nuspec.XMLName.Local != "package" {
		t.Errorf("XMLName.Local = %s, want package", nuspec.XMLName.Local)
	}

	if nuspec.Metadata.ID != "TestPackage" {
		t.Errorf("Metadata.ID = %s, want TestPackage", nuspec.Metadata.ID)
	}

	if nuspec.Metadata.Version != "1.2.3" {
		t.Errorf("Metadata.Version = %s, want 1.2.3", nuspec.Metadata.Version)
	}

	if nuspec.Metadata.Authors != "Author1, Author2" {
		t.Errorf("Metadata.Authors = %s, want Author1, Author2", nuspec.Metadata.Authors)
	}
}
