package packaging

import (
	"archive/zip"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

func TestNewPackageBuilder(t *testing.T) {
	builder := NewPackageBuilder()

	if builder == nil {
		t.Fatal("NewPackageBuilder() returned nil")
	}

	if builder.filePaths == nil {
		t.Error("filePaths map not initialized")
	}

	if builder.createdTime.IsZero() {
		t.Error("createdTime not set")
	}
}

func TestBuilderFluentAPI(t *testing.T) {
	ver := version.MustParse("1.2.3")
	minVer := version.MustParse("5.0.0")

	builder := NewPackageBuilder().
		SetID("TestPackage").
		SetVersion(ver).
		SetDescription("Test description").
		SetAuthors("Author1", "Author2").
		SetTitle("Test Title").
		SetOwners("Owner1", "Owner2").
		SetTags("tag1", "tag2", "tag3").
		SetCopyright("Copyright 2025").
		SetSummary("Test summary").
		SetReleaseNotes("Release notes").
		SetLanguage("en-US").
		SetRequireLicenseAcceptance(true).
		SetDevelopmentDependency(true).
		SetServiceable(true).
		SetIcon("icon.png").
		SetReadme("README.md").
		SetMinClientVersion(minVer)

	metadata := builder.GetMetadata()

	if metadata.ID != "TestPackage" {
		t.Errorf("ID = %q, want TestPackage", metadata.ID)
	}

	if metadata.Version.String() != "1.2.3" {
		t.Errorf("Version = %s, want 1.2.3", metadata.Version)
	}

	if metadata.Description != "Test description" {
		t.Errorf("Description = %q, want Test description", metadata.Description)
	}

	if len(metadata.Authors) != 2 {
		t.Errorf("len(Authors) = %d, want 2", len(metadata.Authors))
	}

	if metadata.Title != "Test Title" {
		t.Errorf("Title = %q, want Test Title", metadata.Title)
	}

	if !metadata.RequireLicenseAcceptance {
		t.Error("RequireLicenseAcceptance should be true")
	}

	if !metadata.DevelopmentDependency {
		t.Error("DevelopmentDependency should be true")
	}

	if !metadata.Serviceable {
		t.Error("Serviceable should be true")
	}

	if metadata.Icon != "icon.png" {
		t.Errorf("Icon = %q, want icon.png", metadata.Icon)
	}

	if metadata.Readme != "README.md" {
		t.Errorf("Readme = %q, want README.md", metadata.Readme)
	}

	if metadata.MinClientVersion.String() != "5.0.0" {
		t.Errorf("MinClientVersion = %s, want 5.0.0", metadata.MinClientVersion)
	}
}

func TestBuilderSetProjectURL(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.SetProjectURL("https://github.com/example/repo")
	if err != nil {
		t.Fatalf("SetProjectURL() error = %v", err)
	}

	if builder.metadata.ProjectURL == nil {
		t.Fatal("ProjectURL not set")
	}

	if builder.metadata.ProjectURL.String() != "https://github.com/example/repo" {
		t.Errorf("ProjectURL = %s, want https://github.com/example/repo", builder.metadata.ProjectURL)
	}
}

func TestBuilderSetProjectURL_Invalid(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.SetProjectURL("://invalid-url")
	if err == nil {
		t.Error("SetProjectURL() expected error for invalid URL")
	}
}

func TestBuilderSetIconURL(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.SetIconURL("https://example.com/icon.png")
	if err != nil {
		t.Fatalf("SetIconURL() error = %v", err)
	}

	if builder.metadata.IconURL == nil {
		t.Fatal("IconURL not set")
	}
}

func TestBuilderSetLicenseURL(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.SetLicenseURL("https://example.com/license")
	if err != nil {
		t.Fatalf("SetLicenseURL() error = %v", err)
	}

	if builder.metadata.LicenseURL == nil {
		t.Fatal("LicenseURL not set")
	}
}

func TestBuilderAddDependency(t *testing.T) {
	builder := NewPackageBuilder()

	net60 := frameworks.MustParseFramework("net6.0")
	versionRange := version.MustParseRange("[1.0.0, 2.0.0)")

	builder.AddDependency(net60, "Newtonsoft.Json", versionRange)

	metadata := builder.GetMetadata()

	if len(metadata.DependencyGroups) != 1 {
		t.Fatalf("len(DependencyGroups) = %d, want 1", len(metadata.DependencyGroups))
	}

	group := metadata.DependencyGroups[0]

	if group.TargetFramework.String() != "net6.0" {
		t.Errorf("TargetFramework = %s, want net6.0", group.TargetFramework)
	}

	if len(group.Dependencies) != 1 {
		t.Fatalf("len(Dependencies) = %d, want 1", len(group.Dependencies))
	}

	dep := group.Dependencies[0]

	if dep.ID != "Newtonsoft.Json" {
		t.Errorf("Dependency ID = %q, want Newtonsoft.Json", dep.ID)
	}
}

func TestBuilderAddDependency_SameFramework(t *testing.T) {
	builder := NewPackageBuilder()

	net60 := frameworks.MustParseFramework("net6.0")
	vr1 := version.MustParseRange("[1.0.0,)")
	vr2 := version.MustParseRange("[2.0.0,)")

	builder.AddDependency(net60, "Package1", vr1)
	builder.AddDependency(net60, "Package2", vr2)

	metadata := builder.GetMetadata()

	if len(metadata.DependencyGroups) != 1 {
		t.Fatalf("len(DependencyGroups) = %d, want 1", len(metadata.DependencyGroups))
	}

	group := metadata.DependencyGroups[0]

	if len(group.Dependencies) != 2 {
		t.Fatalf("len(Dependencies) = %d, want 2", len(group.Dependencies))
	}
}

func TestBuilderAddDependency_DifferentFrameworks(t *testing.T) {
	builder := NewPackageBuilder()

	net60 := frameworks.MustParseFramework("net6.0")
	net48 := frameworks.MustParseFramework("net48")
	vr := version.MustParseRange("[1.0.0,)")

	builder.AddDependency(net60, "Package1", vr)
	builder.AddDependency(net48, "Package2", vr)

	metadata := builder.GetMetadata()

	if len(metadata.DependencyGroups) != 2 {
		t.Fatalf("len(DependencyGroups) = %d, want 2", len(metadata.DependencyGroups))
	}
}

func TestBuilderAddFile(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.AddFile("/source/test.dll", "lib/net6.0/test.dll")
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	files := builder.GetFiles()

	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}

	file := files[0]

	if file.SourcePath != "/source/test.dll" {
		t.Errorf("SourcePath = %q, want /source/test.dll", file.SourcePath)
	}

	if file.TargetPath != "lib/net6.0/test.dll" {
		t.Errorf("TargetPath = %q, want lib/net6.0/test.dll", file.TargetPath)
	}
}

func TestBuilderAddFile_Normalize(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.AddFile("/source/test.dll", "lib\\net6.0\\test.dll")
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	files := builder.GetFiles()

	if files[0].TargetPath != "lib/net6.0/test.dll" {
		t.Errorf("TargetPath = %q, want lib/net6.0/test.dll (normalized)", files[0].TargetPath)
	}
}

func TestBuilderAddFile_Duplicate(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.AddFile("/source/test.dll", "lib/net6.0/test.dll")
	if err != nil {
		t.Fatalf("First AddFile() error = %v", err)
	}

	err = builder.AddFile("/source/test2.dll", "lib/net6.0/test.dll")
	if err == nil {
		t.Error("AddFile() expected error for duplicate path")
	}

	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("Error message should contain 'duplicate', got %v", err)
	}
}

func TestBuilderAddFile_InvalidPath(t *testing.T) {
	builder := NewPackageBuilder()

	tests := []struct {
		name       string
		targetPath string
	}{
		{"path traversal", "../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"empty path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.AddFile("/source/test.dll", tt.targetPath)
			if err == nil {
				t.Errorf("AddFile() expected error for path %q", tt.targetPath)
			}
		})
	}
}

func TestBuilderAddFileFromBytes(t *testing.T) {
	builder := NewPackageBuilder()

	content := []byte("file content")

	err := builder.AddFileFromBytes("lib/net6.0/test.dll", content)
	if err != nil {
		t.Fatalf("AddFileFromBytes() error = %v", err)
	}

	files := builder.GetFiles()

	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}

	file := files[0]

	if file.SourcePath != "" {
		t.Errorf("SourcePath should be empty for in-memory file")
	}

	if file.TargetPath != "lib/net6.0/test.dll" {
		t.Errorf("TargetPath = %q, want lib/net6.0/test.dll", file.TargetPath)
	}

	if !bytes.Equal(file.Content, content) {
		t.Error("Content does not match")
	}
}

func TestBuilderAddFileFromReader(t *testing.T) {
	builder := NewPackageBuilder()

	reader := strings.NewReader("file content")

	err := builder.AddFileFromReader("lib/net6.0/test.dll", reader)
	if err != nil {
		t.Fatalf("AddFileFromReader() error = %v", err)
	}

	files := builder.GetFiles()

	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}

	file := files[0]

	if file.Reader == nil {
		t.Error("Reader should be set")
	}

	if file.SourcePath != "" {
		t.Error("SourcePath should be empty for reader-based file")
	}
}

func TestPopulateFromNuspec(t *testing.T) {
	nuspecXML := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata minClientVersion="5.0.0">
    <id>TestPackage</id>
    <version>1.2.3-beta</version>
    <authors>Author1, Author2</authors>
    <owners>Owner1</owners>
    <description>Test description</description>
    <title>Test Title</title>
    <summary>Test summary</summary>
    <releaseNotes>Release notes</releaseNotes>
    <copyright>Copyright 2025</copyright>
    <language>en-US</language>
    <tags>tag1 tag2 tag3</tags>
    <projectUrl>https://github.com/example/repo</projectUrl>
    <iconUrl>https://example.com/icon.png</iconUrl>
    <licenseUrl>https://example.com/license</licenseUrl>
    <requireLicenseAcceptance>true</requireLicenseAcceptance>
    <developmentDependency>true</developmentDependency>
    <serviceable>true</serviceable>
    <icon>icon.png</icon>
    <readme>README.md</readme>
    <repository type="git" url="https://github.com/example/repo" branch="main" commit="abc123" />
    <packageTypes>
      <packageType name="Dependency" />
    </packageTypes>
    <dependencies>
      <group targetFramework="net6.0">
        <dependency id="Newtonsoft.Json" version="[13.0.0, )" />
      </group>
    </dependencies>
    <frameworkReferences>
      <group targetFramework="net6.0">
        <frameworkReference name="Microsoft.AspNetCore.App" />
      </group>
    </frameworkReferences>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(nuspecXML))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	builder := NewPackageBuilder()
	err = builder.PopulateFromNuspec(nuspec)
	if err != nil {
		t.Fatalf("PopulateFromNuspec() error = %v", err)
	}

	metadata := builder.GetMetadata()

	// Verify required fields
	if metadata.ID != "TestPackage" {
		t.Errorf("ID = %q, want TestPackage", metadata.ID)
	}

	if metadata.Version.String() != "1.2.3-beta" {
		t.Errorf("Version = %s, want 1.2.3-beta", metadata.Version)
	}

	if metadata.Description != "Test description" {
		t.Errorf("Description = %q, want Test description", metadata.Description)
	}

	if len(metadata.Authors) != 2 {
		t.Fatalf("len(Authors) = %d, want 2", len(metadata.Authors))
	}

	// Verify optional fields
	if metadata.Title != "Test Title" {
		t.Errorf("Title = %q, want Test Title", metadata.Title)
	}

	if metadata.Summary != "Test summary" {
		t.Errorf("Summary = %q, want Test summary", metadata.Summary)
	}

	if metadata.Copyright != "Copyright 2025" {
		t.Errorf("Copyright = %q, want Copyright 2025", metadata.Copyright)
	}

	if !metadata.RequireLicenseAcceptance {
		t.Error("RequireLicenseAcceptance should be true")
	}

	if !metadata.DevelopmentDependency {
		t.Error("DevelopmentDependency should be true")
	}

	if !metadata.Serviceable {
		t.Error("Serviceable should be true")
	}

	// Verify URLs
	if metadata.ProjectURL == nil || metadata.ProjectURL.String() != "https://github.com/example/repo" {
		t.Errorf("ProjectURL = %v, want https://github.com/example/repo", metadata.ProjectURL)
	}

	// Verify repository metadata
	if metadata.Repository == nil {
		t.Fatal("Repository metadata should be set")
	}

	if metadata.Repository.Type != "git" {
		t.Errorf("Repository.Type = %q, want git", metadata.Repository.Type)
	}

	// Verify min client version
	if metadata.MinClientVersion == nil || metadata.MinClientVersion.String() != "5.0.0" {
		t.Errorf("MinClientVersion = %v, want 5.0.0", metadata.MinClientVersion)
	}

	// Verify package types
	if len(metadata.PackageTypes) != 1 {
		t.Fatalf("len(PackageTypes) = %d, want 1", len(metadata.PackageTypes))
	}

	if metadata.PackageTypes[0].Name != "Dependency" {
		t.Errorf("PackageType.Name = %q, want Dependency", metadata.PackageTypes[0].Name)
	}

	// Verify dependencies
	if len(metadata.DependencyGroups) != 1 {
		t.Fatalf("len(DependencyGroups) = %d, want 1", len(metadata.DependencyGroups))
	}

	depGroup := metadata.DependencyGroups[0]

	if depGroup.TargetFramework.String() != "net6.0" {
		t.Errorf("DependencyGroup TargetFramework = %s, want net6.0", depGroup.TargetFramework)
	}

	if len(depGroup.Dependencies) != 1 {
		t.Fatalf("len(Dependencies) = %d, want 1", len(depGroup.Dependencies))
	}

	dep := depGroup.Dependencies[0]

	if dep.ID != "Newtonsoft.Json" {
		t.Errorf("Dependency ID = %q, want Newtonsoft.Json", dep.ID)
	}

	// Verify framework references
	if len(metadata.FrameworkReferenceGroups) != 1 {
		t.Fatalf("len(FrameworkReferenceGroups) = %d, want 1", len(metadata.FrameworkReferenceGroups))
	}

	fwRefGroup := metadata.FrameworkReferenceGroups[0]

	if fwRefGroup.TargetFramework.String() != "net6.0" {
		t.Errorf("FrameworkReferenceGroup TargetFramework = %s, want net6.0", fwRefGroup.TargetFramework)
	}

	if len(fwRefGroup.References) != 1 {
		t.Fatalf("len(References) = %d, want 1", len(fwRefGroup.References))
	}

	if fwRefGroup.References[0] != "Microsoft.AspNetCore.App" {
		t.Errorf("Reference = %q, want Microsoft.AspNetCore.App", fwRefGroup.References[0])
	}
}

func TestNewPackageBuilderFromNuspec(t *testing.T) {
	// Create a temporary nuspec file
	nuspecXML := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <authors>Test Author</authors>
    <description>Test description</description>
  </metadata>
</package>`

	tmpFile := t.TempDir() + "/test.nuspec"
	err := os.WriteFile(tmpFile, []byte(nuspecXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp nuspec: %v", err)
	}

	builder, err := NewPackageBuilderFromNuspec(tmpFile)
	if err != nil {
		t.Fatalf("NewPackageBuilderFromNuspec() error = %v", err)
	}

	metadata := builder.GetMetadata()

	if metadata.ID != "TestPackage" {
		t.Errorf("ID = %q, want TestPackage", metadata.ID)
	}

	if metadata.Version.String() != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", metadata.Version)
	}
}

func TestNewPackageBuilderFromNuspec_FileNotFound(t *testing.T) {
	_, err := NewPackageBuilderFromNuspec("/nonexistent/test.nuspec")
	if err == nil {
		t.Error("NewPackageBuilderFromNuspec() expected error for non-existent file")
	}
}

func TestBuilderSetRepository(t *testing.T) {
	builder := NewPackageBuilder()

	repo := &PackageRepositoryMetadata{
		Type:   "git",
		URL:    "https://github.com/example/repo",
		Branch: "main",
		Commit: "abc123",
	}

	builder.SetRepository(repo)

	metadata := builder.GetMetadata()

	if metadata.Repository == nil {
		t.Fatal("Repository should be set")
	}

	if metadata.Repository.Type != "git" {
		t.Errorf("Repository.Type = %q, want git", metadata.Repository.Type)
	}
}

func TestBuilderSetLicenseMetadata(t *testing.T) {
	builder := NewPackageBuilder()

	license := &LicenseMetadata{
		Type:    "expression",
		Text:    "MIT",
		Version: "1.0.0",
	}

	builder.SetLicenseMetadata(license)

	metadata := builder.GetMetadata()

	if metadata.LicenseMetadata == nil {
		t.Fatal("LicenseMetadata should be set")
	}

	if metadata.LicenseMetadata.Type != "expression" {
		t.Errorf("LicenseMetadata.Type = %q, want expression", metadata.LicenseMetadata.Type)
	}
}

func TestBuilderAddDependencyGroup(t *testing.T) {
	builder := NewPackageBuilder()

	net60 := frameworks.MustParseFramework("net6.0")
	vr := version.MustParseRange("[1.0.0,)")

	group := PackageDependencyGroup{
		TargetFramework: net60,
		Dependencies: []PackageDependency{
			{ID: "Package1", VersionRange: vr},
			{ID: "Package2", VersionRange: vr},
		},
	}

	builder.AddDependencyGroup(group)

	metadata := builder.GetMetadata()

	if len(metadata.DependencyGroups) != 1 {
		t.Fatalf("len(DependencyGroups) = %d, want 1", len(metadata.DependencyGroups))
	}

	if len(metadata.DependencyGroups[0].Dependencies) != 2 {
		t.Errorf("len(Dependencies) = %d, want 2", len(metadata.DependencyGroups[0].Dependencies))
	}
}

func TestBuilderAddFrameworkReferenceGroup(t *testing.T) {
	builder := NewPackageBuilder()

	net60 := frameworks.MustParseFramework("net6.0")

	group := PackageFrameworkReferenceGroup{
		TargetFramework: net60,
		References:      []string{"Microsoft.AspNetCore.App"},
	}

	builder.AddFrameworkReferenceGroup(group)

	metadata := builder.GetMetadata()

	if len(metadata.FrameworkReferenceGroups) != 1 {
		t.Fatalf("len(FrameworkReferenceGroups) = %d, want 1", len(metadata.FrameworkReferenceGroups))
	}
}

func TestBuilderAddPackageType(t *testing.T) {
	builder := NewPackageBuilder()

	pkgType := PackageTypeInfo{
		Name:    "Dependency",
		Version: version.MustParse("1.0.0"),
	}

	builder.AddPackageType(pkgType)

	metadata := builder.GetMetadata()

	if len(metadata.PackageTypes) != 1 {
		t.Fatalf("len(PackageTypes) = %d, want 1", len(metadata.PackageTypes))
	}

	if metadata.PackageTypes[0].Name != "Dependency" {
		t.Errorf("PackageType.Name = %q, want Dependency", metadata.PackageTypes[0].Name)
	}
}

func TestNormalizePackagePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"forward slashes", "lib/net6.0/test.dll", "lib/net6.0/test.dll"},
		{"backslashes", "lib\\net6.0\\test.dll", "lib/net6.0/test.dll"},
		{"mixed", "lib/net6.0\\test.dll", "lib/net6.0/test.dll"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePackagePath(tt.input)
			if got != tt.expected {
				t.Errorf("normalizePackagePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFrameworksEqual(t *testing.T) {
	net60 := frameworks.MustParseFramework("net6.0")
	net60_2 := frameworks.MustParseFramework("net6.0")
	net48 := frameworks.MustParseFramework("net48")

	tests := []struct {
		name     string
		a        *frameworks.NuGetFramework
		b        *frameworks.NuGetFramework
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", net60, nil, false},
		{"other nil", nil, net60, false},
		{"same framework", net60, net60_2, true},
		{"different framework", net60, net48, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := frameworksEqual(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("frameworksEqual() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuilderWriteOPCFiles(t *testing.T) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	builder := NewPackageBuilder().
		SetID("TestPackage").
		SetVersion(version.MustParse("1.0.0")).
		SetDescription("Test").
		SetAuthors("Test Author")

	_ = builder.AddFile("test.dll", "lib/net6.0/test.dll")

	nuspecFileName := "TestPackage.nuspec"

	err := builder.writeOPCFiles(zipWriter, nuspecFileName)
	if err != nil {
		t.Fatalf("writeOPCFiles() error = %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close ZIP error = %v", err)
	}

	// Verify ZIP contents
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Open ZIP error = %v", err)
	}

	// Check for required OPC files
	var foundContentTypes, foundRels, foundCoreProps bool

	for _, file := range zipReader.File {
		switch {
		case file.Name == OPCContentTypesPath:
			foundContentTypes = true
		case file.Name == OPCRelationshipsPath:
			foundRels = true
		case strings.HasPrefix(file.Name, OPCCorePropertiesPath) && strings.HasSuffix(file.Name, ".psmdcp"):
			foundCoreProps = true
		}
	}

	if !foundContentTypes {
		t.Error("[Content_Types].xml not found")
	}

	if !foundRels {
		t.Error("_rels/.rels not found")
	}

	if !foundCoreProps {
		t.Error("Core properties file not found")
	}
}

func TestBuilderAddFileFromBytes_Duplicate(t *testing.T) {
	builder := NewPackageBuilder()

	content := []byte("test content")
	targetPath := "lib/net6.0/test.dll"

	err := builder.AddFileFromBytes(targetPath, content)
	if err != nil {
		t.Fatalf("First AddFileFromBytes() error = %v", err)
	}

	// Try to add duplicate
	err = builder.AddFileFromBytes(targetPath, content)
	if err == nil {
		t.Error("Expected error for duplicate file, got nil")
	}
}

func TestBuilderAddFileFromReader_Duplicate(t *testing.T) {
	builder := NewPackageBuilder()

	reader1 := strings.NewReader("content 1")
	reader2 := strings.NewReader("content 2")
	targetPath := "lib/net6.0/test.dll"

	err := builder.AddFileFromReader(targetPath, reader1)
	if err != nil {
		t.Fatalf("First AddFileFromReader() error = %v", err)
	}

	// Try to add duplicate
	err = builder.AddFileFromReader(targetPath, reader2)
	if err == nil {
		t.Error("Expected error for duplicate file, got nil")
	}
}

func TestBuilderSetIconURL_Invalid(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.SetIconURL("://invalid-url")
	if err == nil {
		t.Error("Expected error for invalid icon URL, got nil")
	}
}

func TestBuilderSetLicenseURL_Invalid(t *testing.T) {
	builder := NewPackageBuilder()

	err := builder.SetLicenseURL("://invalid-url")
	if err == nil {
		t.Error("Expected error for invalid license URL, got nil")
	}
}
