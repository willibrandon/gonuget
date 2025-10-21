package packaging

import (
	"strings"
	"testing"
)

// createTestPackageWithNuspec creates a test package reader with the given nuspec content
func createTestPackageWithNuspec(t *testing.T, filename string, nuspecContent string) *PackageReader {
	t.Helper()

	packageData := createTestPackage(t, map[string]string{
		filename: nuspecContent,
	}, false)

	reader, err := OpenPackageFromReaderAt(packageData, int64(packageData.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt() error = %v", err)
	}

	return reader
}

func TestParseNuspec_Minimal(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>A test package</description>
    <authors>Test Author</authors>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	if nuspec.Metadata.ID != "TestPackage" {
		t.Errorf("ID = %q, want %q", nuspec.Metadata.ID, "TestPackage")
	}

	if nuspec.Metadata.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", nuspec.Metadata.Version, "1.0.0")
	}

	if nuspec.Metadata.Description != "A test package" {
		t.Errorf("Description = %q, want %q", nuspec.Metadata.Description, "A test package")
	}

	if nuspec.Metadata.Authors != "Test Author" {
		t.Errorf("Authors = %q, want %q", nuspec.Metadata.Authors, "Test Author")
	}
}

func TestParseNuspec_Complete(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata minClientVersion="3.3.0">
    <id>Newtonsoft.Json</id>
    <version>13.0.3</version>
    <title>Json.NET</title>
    <authors>James Newton-King</authors>
    <owners>James Newton-King</owners>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <license type="expression">MIT</license>
    <licenseUrl>https://licenses.nuget.org/MIT</licenseUrl>
    <icon>icon.png</icon>
    <projectUrl>https://www.newtonsoft.com/json</projectUrl>
    <description>Json.NET is a popular high-performance JSON framework for .NET</description>
    <copyright>Copyright © James Newton-King 2008</copyright>
    <tags>json</tags>
    <repository type="git" url="https://github.com/JamesNK/Newtonsoft.Json" commit="abc123" branch="master" />
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ID", nuspec.Metadata.ID, "Newtonsoft.Json"},
		{"Version", nuspec.Metadata.Version, "13.0.3"},
		{"Title", nuspec.Metadata.Title, "Json.NET"},
		{"Authors", nuspec.Metadata.Authors, "James Newton-King"},
		{"Owners", nuspec.Metadata.Owners, "James Newton-King"},
		{"LicenseURL", nuspec.Metadata.LicenseURL, "https://licenses.nuget.org/MIT"},
		{"Icon", nuspec.Metadata.Icon, "icon.png"},
		{"ProjectURL", nuspec.Metadata.ProjectURL, "https://www.newtonsoft.com/json"},
		{"Description", nuspec.Metadata.Description, "Json.NET is a popular high-performance JSON framework for .NET"},
		{"Copyright", nuspec.Metadata.Copyright, "Copyright © James Newton-King 2008"},
		{"Tags", nuspec.Metadata.Tags, "json"},
		{"MinClientVersion", nuspec.Metadata.MinClientVersion, "3.3.0"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}

	if nuspec.Metadata.RequireLicenseAcceptance != false {
		t.Errorf("RequireLicenseAcceptance = %v, want %v", nuspec.Metadata.RequireLicenseAcceptance, false)
	}

	if nuspec.Metadata.License == nil {
		t.Fatal("License is nil")
	}
	if nuspec.Metadata.License.Type != "expression" {
		t.Errorf("License.Type = %q, want %q", nuspec.Metadata.License.Type, "expression")
	}
	if nuspec.Metadata.License.Text != "MIT" {
		t.Errorf("License.Text = %q, want %q", nuspec.Metadata.License.Text, "MIT")
	}

	if nuspec.Metadata.Repository == nil {
		t.Fatal("Repository is nil")
	}
	if nuspec.Metadata.Repository.Type != "git" {
		t.Errorf("Repository.Type = %q, want %q", nuspec.Metadata.Repository.Type, "git")
	}
	if nuspec.Metadata.Repository.URL != "https://github.com/JamesNK/Newtonsoft.Json" {
		t.Errorf("Repository.URL = %q, want %q", nuspec.Metadata.Repository.URL, "https://github.com/JamesNK/Newtonsoft.Json")
	}
	if nuspec.Metadata.Repository.Commit != "abc123" {
		t.Errorf("Repository.Commit = %q, want %q", nuspec.Metadata.Repository.Commit, "abc123")
	}
	if nuspec.Metadata.Repository.Branch != "master" {
		t.Errorf("Repository.Branch = %q, want %q", nuspec.Metadata.Repository.Branch, "master")
	}
}

func TestParseNuspec_InvalidXML(t *testing.T) {
	xml := `<?xml version="1.0"?><invalid>`

	_, err := ParseNuspec(strings.NewReader(xml))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestGetParsedIdentity(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		version     string
		wantErr     bool
		wantID      string
		wantVersion string
	}{
		{
			name:        "Valid identity",
			id:          "TestPackage",
			version:     "1.0.0",
			wantErr:     false,
			wantID:      "TestPackage",
			wantVersion: "1.0.0",
		},
		{
			name:        "Semantic version",
			id:          "TestPackage",
			version:     "1.0.0-beta.1+build.123",
			wantErr:     false,
			wantID:      "TestPackage",
			wantVersion: "1.0.0-beta.1+build.123",
		},
		{
			name:    "Invalid version",
			id:      "TestPackage",
			version: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nuspec := &Nuspec{
				Metadata: NuspecMetadata{
					ID:      tt.id,
					Version: tt.version,
				},
			}

			identity, err := nuspec.GetParsedIdentity()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetParsedIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if identity.ID != tt.wantID {
					t.Errorf("ID = %q, want %q", identity.ID, tt.wantID)
				}
				if identity.Version.String() != tt.wantVersion {
					t.Errorf("Version = %q, want %q", identity.Version.String(), tt.wantVersion)
				}
			}
		})
	}
}

func TestGetAuthors(t *testing.T) {
	tests := []struct {
		name    string
		authors string
		want    []string
	}{
		{
			name:    "Single author",
			authors: "John Doe",
			want:    []string{"John Doe"},
		},
		{
			name:    "Multiple authors",
			authors: "John Doe, Jane Smith, Bob Johnson",
			want:    []string{"John Doe", "Jane Smith", "Bob Johnson"},
		},
		{
			name:    "Authors with spaces",
			authors: "John Doe , Jane Smith , Bob Johnson",
			want:    []string{"John Doe", "Jane Smith", "Bob Johnson"},
		},
		{
			name:    "Empty authors",
			authors: "",
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nuspec := &Nuspec{
				Metadata: NuspecMetadata{
					Authors: tt.authors,
				},
			}

			got := nuspec.GetAuthors()
			if len(got) != len(tt.want) {
				t.Errorf("len(GetAuthors()) = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetAuthors()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetOwners(t *testing.T) {
	tests := []struct {
		name   string
		owners string
		want   []string
	}{
		{
			name:   "Single owner",
			owners: "John Doe",
			want:   []string{"John Doe"},
		},
		{
			name:   "Multiple owners",
			owners: "John Doe, Jane Smith",
			want:   []string{"John Doe", "Jane Smith"},
		},
		{
			name:   "Empty owners",
			owners: "",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nuspec := &Nuspec{
				Metadata: NuspecMetadata{
					Owners: tt.owners,
				},
			}

			got := nuspec.GetOwners()
			if len(got) != len(tt.want) {
				t.Errorf("len(GetOwners()) = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetOwners()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetTags(t *testing.T) {
	tests := []struct {
		name string
		tags string
		want []string
	}{
		{
			name: "Single tag",
			tags: "json",
			want: []string{"json"},
		},
		{
			name: "Multiple tags",
			tags: "json serialization dotnet",
			want: []string{"json", "serialization", "dotnet"},
		},
		{
			name: "Tags with extra spaces",
			tags: "json  serialization   dotnet",
			want: []string{"json", "serialization", "dotnet"},
		},
		{
			name: "Empty tags",
			tags: "",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nuspec := &Nuspec{
				Metadata: NuspecMetadata{
					Tags: tt.tags,
				},
			}

			got := nuspec.GetTags()
			if len(got) != len(tt.want) {
				t.Errorf("len(GetTags()) = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetTags()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetDependencyGroups_Legacy(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <dependencies>
      <dependency id="Newtonsoft.Json" version="13.0.1" />
      <dependency id="System.Text.Json" version="[6.0.0,7.0.0)" />
    </dependencies>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetDependencyGroups()
	if err != nil {
		t.Fatalf("GetDependencyGroups() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}

	group := groups[0]

	// Should be AnyFramework
	if !group.TargetFramework.IsAny() {
		t.Error("expected AnyFramework for legacy dependencies")
	}

	if len(group.Dependencies) != 2 {
		t.Fatalf("len(Dependencies) = %d, want 2", len(group.Dependencies))
	}

	// Check first dependency
	dep1 := group.Dependencies[0]
	if dep1.ID != "Newtonsoft.Json" {
		t.Errorf("Dependencies[0].ID = %q, want %q", dep1.ID, "Newtonsoft.Json")
	}
	if dep1.VersionRange == nil {
		t.Fatal("Dependencies[0].VersionRange is nil")
	}
	if !dep1.VersionRange.MinInclusive || dep1.VersionRange.MinVersion.String() != "13.0.1" {
		t.Errorf("Dependencies[0].VersionRange = %v, want >= 13.0.1", dep1.VersionRange)
	}

	// Check second dependency
	dep2 := group.Dependencies[1]
	if dep2.ID != "System.Text.Json" {
		t.Errorf("Dependencies[1].ID = %q, want %q", dep2.ID, "System.Text.Json")
	}
	if dep2.VersionRange == nil {
		t.Fatal("Dependencies[1].VersionRange is nil")
	}
	if !dep2.VersionRange.MinInclusive || dep2.VersionRange.MaxVersion == nil || dep2.VersionRange.MaxInclusive {
		t.Errorf("Dependencies[1].VersionRange = %v, want [6.0.0, 7.0.0)", dep2.VersionRange)
	}
}

func TestGetDependencyGroups_Grouped(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <dependencies>
      <group targetFramework="net6.0">
        <dependency id="System.Text.Json" version="6.0.0" />
      </group>
      <group targetFramework="net48">
        <dependency id="Newtonsoft.Json" version="13.0.1" />
      </group>
    </dependencies>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetDependencyGroups()
	if err != nil {
		t.Fatalf("GetDependencyGroups() error = %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}

	// Check net6.0 group
	net6Group := groups[0]
	if net6Group.TargetFramework.Framework != ".NETCoreApp" {
		t.Errorf("groups[0].TargetFramework.Framework = %q, want %q", net6Group.TargetFramework.Framework, ".NETCoreApp")
	}
	if len(net6Group.Dependencies) != 1 {
		t.Fatalf("len(groups[0].Dependencies) = %d, want 1", len(net6Group.Dependencies))
	}
	if net6Group.Dependencies[0].ID != "System.Text.Json" {
		t.Errorf("groups[0].Dependencies[0].ID = %q, want %q", net6Group.Dependencies[0].ID, "System.Text.Json")
	}

	// Check net48 group
	net48Group := groups[1]
	if net48Group.TargetFramework.Framework != ".NETFramework" {
		t.Errorf("groups[1].TargetFramework.Framework = %q, want %q", net48Group.TargetFramework.Framework, ".NETFramework")
	}
	if len(net48Group.Dependencies) != 1 {
		t.Fatalf("len(groups[1].Dependencies) = %d, want 1", len(net48Group.Dependencies))
	}
	if net48Group.Dependencies[0].ID != "Newtonsoft.Json" {
		t.Errorf("groups[1].Dependencies[0].ID = %q, want %q", net48Group.Dependencies[0].ID, "Newtonsoft.Json")
	}
}

func TestGetDependencyGroups_EmptyGroup(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <dependencies>
      <group targetFramework="" />
    </dependencies>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetDependencyGroups()
	if err != nil {
		t.Fatalf("GetDependencyGroups() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}

	// Empty targetFramework should be treated as "any"
	if !groups[0].TargetFramework.IsAny() {
		t.Error("expected AnyFramework for empty targetFramework")
	}
}

func TestGetDependencyGroups_WithAssetFilters(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <dependencies>
      <group targetFramework="net6.0">
        <dependency id="TestDep" version="1.0.0" include="Compile,Runtime" exclude="Build" />
      </group>
    </dependencies>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetDependencyGroups()
	if err != nil {
		t.Fatalf("GetDependencyGroups() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}

	dep := groups[0].Dependencies[0]

	if len(dep.Include) != 2 {
		t.Fatalf("len(Include) = %d, want 2 (got %v)", len(dep.Include), dep.Include)
	}
	if dep.Include[0] != "Compile" || dep.Include[1] != "Runtime" {
		t.Errorf("Include = %v, want [Compile Runtime]", dep.Include)
	}

	if len(dep.Exclude) != 1 {
		t.Fatalf("len(Exclude) = %d, want 1 (got %v)", len(dep.Exclude), dep.Exclude)
	}
	if dep.Exclude[0] != "Build" {
		t.Errorf("Exclude = %v, want [Build]", dep.Exclude)
	}
}

func TestGetDependencyGroups_NoDependencies(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetDependencyGroups()
	if err != nil {
		t.Fatalf("GetDependencyGroups() error = %v", err)
	}

	if len(groups) != 0 {
		t.Errorf("len(groups) = %d, want 0", len(groups))
	}
}

func TestGetFrameworkReferenceGroups(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <frameworkReferences>
      <group targetFramework="net6.0">
        <frameworkReference name="Microsoft.AspNetCore.App" />
        <frameworkReference name="Microsoft.WindowsDesktop.App" />
      </group>
      <group targetFramework="net7.0">
        <frameworkReference name="Microsoft.AspNetCore.App" />
      </group>
    </frameworkReferences>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetFrameworkReferenceGroups()
	if err != nil {
		t.Fatalf("GetFrameworkReferenceGroups() error = %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}

	// Check net6.0 group
	net6Group := groups[0]
	if net6Group.TargetFramework.Framework != ".NETCoreApp" {
		t.Errorf("groups[0].TargetFramework.Framework = %q, want %q", net6Group.TargetFramework.Framework, ".NETCoreApp")
	}
	if len(net6Group.References) != 2 {
		t.Fatalf("len(groups[0].References) = %d, want 2", len(net6Group.References))
	}
	if net6Group.References[0] != "Microsoft.AspNetCore.App" {
		t.Errorf("groups[0].References[0] = %q, want %q", net6Group.References[0], "Microsoft.AspNetCore.App")
	}
	if net6Group.References[1] != "Microsoft.WindowsDesktop.App" {
		t.Errorf("groups[0].References[1] = %q, want %q", net6Group.References[1], "Microsoft.WindowsDesktop.App")
	}

	// Check net7.0 group
	net7Group := groups[1]
	if net7Group.TargetFramework.Framework != ".NETCoreApp" {
		t.Errorf("groups[1].TargetFramework.Framework = %q, want %q", net7Group.TargetFramework.Framework, ".NETCoreApp")
	}
	if len(net7Group.References) != 1 {
		t.Fatalf("len(groups[1].References) = %d, want 1", len(net7Group.References))
	}
	if net7Group.References[0] != "Microsoft.AspNetCore.App" {
		t.Errorf("groups[1].References[0] = %q, want %q", net7Group.References[0], "Microsoft.AspNetCore.App")
	}
}

func TestGetFrameworkReferenceGroups_NoReferences(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	groups, err := nuspec.GetFrameworkReferenceGroups()
	if err != nil {
		t.Fatalf("GetFrameworkReferenceGroups() error = %v", err)
	}

	if len(groups) != 0 {
		t.Errorf("len(groups) = %d, want 0", len(groups))
	}
}

func TestParsedDependency_NoVersionRange(t *testing.T) {
	// Dependency without version attribute should have nil VersionRange
	deps := []Dependency{
		{ID: "TestPackage"},
	}

	parsed, err := parseDependencies(deps)
	if err != nil {
		t.Fatalf("parseDependencies() error = %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("len(parsed) = %d, want 1", len(parsed))
	}

	if parsed[0].VersionRange != nil {
		t.Error("expected nil VersionRange for dependency without version")
	}
}

func TestParseDependencies_InvalidVersion(t *testing.T) {
	deps := []Dependency{
		{ID: "TestPackage", Version: "invalid[version]range"},
	}

	_, err := parseDependencies(deps)
	if err == nil {
		t.Error("expected error for invalid version range")
	}
}

func TestParseNuspec_Files(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
  </metadata>
  <files>
    <file src="bin\Release\*.dll" target="lib\net6.0" />
    <file src="content\**\*" target="contentFiles" exclude="**\*.pdb" />
  </files>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	if len(nuspec.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(nuspec.Files))
	}

	file1 := nuspec.Files[0]
	if file1.Source != `bin\Release\*.dll` {
		t.Errorf("Files[0].Source = %q, want %q", file1.Source, `bin\Release\*.dll`)
	}
	if file1.Target != `lib\net6.0` {
		t.Errorf("Files[0].Target = %q, want %q", file1.Target, `lib\net6.0`)
	}

	file2 := nuspec.Files[1]
	if file2.Exclude != `**\*.pdb` {
		t.Errorf("Files[1].Exclude = %q, want %q", file2.Exclude, `**\*.pdb`)
	}
}

func TestParseNuspec_PackageTypes(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <packageTypes>
      <packageType name="Dependency" />
      <packageType name="DotnetTool" version="1.0" />
    </packageTypes>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	if len(nuspec.Metadata.PackageTypes) != 2 {
		t.Fatalf("len(PackageTypes) = %d, want 2", len(nuspec.Metadata.PackageTypes))
	}

	pt1 := nuspec.Metadata.PackageTypes[0]
	if pt1.Name != "Dependency" {
		t.Errorf("PackageTypes[0].Name = %q, want %q", pt1.Name, "Dependency")
	}

	pt2 := nuspec.Metadata.PackageTypes[1]
	if pt2.Name != "DotnetTool" {
		t.Errorf("PackageTypes[1].Name = %q, want %q", pt2.Name, "DotnetTool")
	}
	if pt2.Version != "1.0" {
		t.Errorf("PackageTypes[1].Version = %q, want %q", pt2.Version, "1.0")
	}
}

func TestParseNuspec_ContentFiles(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <contentFiles>
      <files include="**/*.cs" buildAction="Compile" />
      <files include="**/*.txt" copyToOutput="true" flatten="false" />
    </contentFiles>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	if len(nuspec.Metadata.ContentFiles) != 2 {
		t.Fatalf("len(ContentFiles) = %d, want 2", len(nuspec.Metadata.ContentFiles))
	}

	cf1 := nuspec.Metadata.ContentFiles[0]
	if cf1.Include != "**/*.cs" {
		t.Errorf("ContentFiles[0].Include = %q, want %q", cf1.Include, "**/*.cs")
	}
	if cf1.BuildAction != "Compile" {
		t.Errorf("ContentFiles[0].BuildAction = %q, want %q", cf1.BuildAction, "Compile")
	}

	cf2 := nuspec.Metadata.ContentFiles[1]
	if cf2.CopyToOutput != "true" {
		t.Errorf("ContentFiles[1].CopyToOutput = %q, want %q", cf2.CopyToOutput, "true")
	}
	if cf2.Flatten != "false" {
		t.Errorf("ContentFiles[1].Flatten = %q, want %q", cf2.Flatten, "false")
	}
}

func TestParseNuspec_FrameworkAssemblies(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <frameworkAssemblies>
      <frameworkAssembly assemblyName="System.Data" targetFramework="net45" />
      <frameworkAssembly assemblyName="System.Xml" targetFramework="net48" />
    </frameworkAssemblies>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	if len(nuspec.Metadata.FrameworkAssemblies) != 2 {
		t.Fatalf("len(FrameworkAssemblies) = %d, want 2", len(nuspec.Metadata.FrameworkAssemblies))
	}

	fa1 := nuspec.Metadata.FrameworkAssemblies[0]
	if fa1.AssemblyName != "System.Data" {
		t.Errorf("FrameworkAssemblies[0].AssemblyName = %q, want %q", fa1.AssemblyName, "System.Data")
	}
	if fa1.TargetFramework != "net45" {
		t.Errorf("FrameworkAssemblies[0].TargetFramework = %q, want %q", fa1.TargetFramework, "net45")
	}
}

func TestParseNuspec_References(t *testing.T) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <description>Test</description>
    <authors>Test</authors>
    <references>
      <group targetFramework="net6.0">
        <reference file="MyLib.dll" />
        <reference file="MyLib.Core.dll" />
      </group>
    </references>
  </metadata>
</package>`

	nuspec, err := ParseNuspec(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuspec() error = %v", err)
	}

	if nuspec.Metadata.References == nil {
		t.Fatal("References is nil")
	}

	if len(nuspec.Metadata.References.Groups) != 1 {
		t.Fatalf("len(References.Groups) = %d, want 1", len(nuspec.Metadata.References.Groups))
	}

	group := nuspec.Metadata.References.Groups[0]
	if group.TargetFramework != "net6.0" {
		t.Errorf("References.Groups[0].TargetFramework = %q, want %q", group.TargetFramework, "net6.0")
	}

	if len(group.References) != 2 {
		t.Fatalf("len(References.Groups[0].References) = %d, want 2", len(group.References))
	}

	if group.References[0].File != "MyLib.dll" {
		t.Errorf("References.Groups[0].References[0].File = %q, want %q", group.References[0].File, "MyLib.dll")
	}
}

func TestPackageReader_GetNuspec(t *testing.T) {
	nuspecXML := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.2.3</version>
    <description>Test package</description>
    <authors>Test Author</authors>
  </metadata>
</package>`

	reader := createTestPackageWithNuspec(t, "TestPackage.nuspec", nuspecXML)
	defer func() { _ = reader.Close() }()

	nuspec, err := reader.GetNuspec()
	if err != nil {
		t.Fatalf("GetNuspec() error = %v", err)
	}

	if nuspec.Metadata.ID != "TestPackage" {
		t.Errorf("ID = %q, want %q", nuspec.Metadata.ID, "TestPackage")
	}

	if nuspec.Metadata.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", nuspec.Metadata.Version, "1.2.3")
	}

	if nuspec.Metadata.Description != "Test package" {
		t.Errorf("Description = %q, want %q", nuspec.Metadata.Description, "Test package")
	}

	if nuspec.Metadata.Authors != "Test Author" {
		t.Errorf("Authors = %q, want %q", nuspec.Metadata.Authors, "Test Author")
	}
}

func TestPackageReader_GetIdentity(t *testing.T) {
	nuspecXML := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage</id>
    <version>2.3.4-beta.1</version>
    <description>Test</description>
    <authors>Test</authors>
  </metadata>
</package>`

	reader := createTestPackageWithNuspec(t, "MyPackage.nuspec", nuspecXML)
	defer func() { _ = reader.Close() }()

	identity, err := reader.GetIdentity()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	if identity.ID != "MyPackage" {
		t.Errorf("ID = %q, want %q", identity.ID, "MyPackage")
	}

	if identity.Version.String() != "2.3.4-beta.1" {
		t.Errorf("Version = %q, want %q", identity.Version.String(), "2.3.4-beta.1")
	}

	// Test that identity is cached
	identity2, err := reader.GetIdentity()
	if err != nil {
		t.Fatalf("GetIdentity() second call error = %v", err)
	}

	if identity2 != identity {
		t.Error("GetIdentity() should return cached identity (same pointer)")
	}
}

func TestPackageReader_GetIdentity_InvalidVersion(t *testing.T) {
	nuspecXML := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage</id>
    <version>invalid-version</version>
    <description>Test</description>
    <authors>Test</authors>
  </metadata>
</package>`

	reader := createTestPackageWithNuspec(t, "MyPackage.nuspec", nuspecXML)
	defer func() { _ = reader.Close() }()

	_, err := reader.GetIdentity()
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func BenchmarkParseNuspec(b *testing.B) {
	xml := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata minClientVersion="3.3.0">
    <id>Newtonsoft.Json</id>
    <version>13.0.3</version>
    <title>Json.NET</title>
    <authors>James Newton-King</authors>
    <owners>James Newton-King</owners>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <license type="expression">MIT</license>
    <projectUrl>https://www.newtonsoft.com/json</projectUrl>
    <description>Json.NET is a popular high-performance JSON framework for .NET</description>
    <dependencies>
      <group targetFramework="net6.0">
        <dependency id="System.Text.Json" version="6.0.0" />
      </group>
      <group targetFramework="net48">
        <dependency id="System.Runtime" version="4.3.0" />
      </group>
    </dependencies>
  </metadata>
</package>`

	b.ResetTimer()
	for b.Loop() {
		_, err := ParseNuspec(strings.NewReader(xml))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetDependencyGroups(b *testing.B) {
	nuspec := &Nuspec{
		Metadata: NuspecMetadata{
			ID:      "Test",
			Version: "1.0.0",
			Dependencies: &DependenciesElement{
				Groups: []DependencyGroup{
					{
						TargetFramework: "net6.0",
						Dependencies: []Dependency{
							{ID: "Dep1", Version: "1.0.0"},
							{ID: "Dep2", Version: "[2.0.0,3.0.0)"},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := nuspec.GetDependencyGroups()
		if err != nil {
			b.Fatal(err)
		}
	}
}
