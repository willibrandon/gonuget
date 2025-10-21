package packaging

import (
	"net/url"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// mustParseURL is a test helper that parses a URL or panics
func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestValidatePackageID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "valid simple ID",
			id:      "MyPackage",
			wantErr: false,
		},
		{
			name:    "valid ID with dots",
			id:      "My.Package.Name",
			wantErr: false,
		},
		{
			name:    "valid ID with hyphens",
			id:      "My-Package-Name",
			wantErr: false,
		},
		{
			name:    "valid ID with underscores",
			id:      "My_Package_Name",
			wantErr: false,
		},
		{
			name:    "valid ID starting with underscore",
			id:      "_MyPackage",
			wantErr: false,
		},
		{
			name:    "valid ID with digits",
			id:      "Package123",
			wantErr: false,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
		},
		{
			name:    "ID starting with digit",
			id:      "1Package",
			wantErr: true,
		},
		{
			name:    "ID with spaces",
			id:      "My Package",
			wantErr: true,
		},
		{
			name:    "ID with special characters",
			id:      "My@Package",
			wantErr: true,
		},
		{
			name:    "ID too long",
			id:      strings.Repeat("a", 101),
			wantErr: true,
		},
		{
			name:    "ID at max length",
			id:      "a" + strings.Repeat("b", 99),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackageID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		pkgID   string
		pkgVer  *version.NuGetVersion
		groups  []PackageDependencyGroup
		wantErr bool
	}{
		{
			name:   "valid dependencies",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{ID: "Dep1", VersionRange: version.MustParseRange("[1.0.0, 2.0.0)")},
						{ID: "Dep2", VersionRange: version.MustParseRange("1.0.0")},
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "duplicate dependency in group",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{ID: "Dep1", VersionRange: version.MustParseRange("1.0.0")},
						{ID: "dep1", VersionRange: version.MustParseRange("2.0.0")},
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "self dependency",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{ID: "mypackage", VersionRange: version.MustParseRange("1.0.0")},
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "invalid version range - max < min",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{
							ID: "Dep1",
							VersionRange: &version.VersionRange{
								MinVersion:   version.MustParse("2.0.0"),
								MaxVersion:   version.MustParse("1.0.0"),
								MinInclusive: true,
								MaxInclusive: true,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "invalid version range - exclusive with equal versions",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{
							ID: "Dep1",
							VersionRange: &version.VersionRange{
								MinVersion:   version.MustParse("1.0.0"),
								MaxVersion:   version.MustParse("1.0.0"),
								MinInclusive: false,
								MaxInclusive: false,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "valid dependency with nil version range",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{ID: "Dep1", VersionRange: nil},
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "valid dependency with only min version",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{
							ID: "Dep1",
							VersionRange: &version.VersionRange{
								MinVersion:   version.MustParse("1.0.0"),
								MinInclusive: true,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "valid dependency with only max version",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{
							ID: "Dep1",
							VersionRange: &version.VersionRange{
								MaxVersion:   version.MustParse("2.0.0"),
								MaxInclusive: true,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "valid version range - equal versions with inclusive",
			pkgID:  "MyPackage",
			pkgVer: version.MustParse("1.0.0"),
			groups: []PackageDependencyGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					Dependencies: []PackageDependency{
						{
							ID: "Dep1",
							VersionRange: &version.VersionRange{
								MinVersion:   version.MustParse("1.0.0"),
								MaxVersion:   version.MustParse("1.0.0"),
								MinInclusive: true,
								MaxInclusive: true,
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDependencies(tt.pkgID, tt.pkgVer, tt.groups)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFiles(t *testing.T) {
	tests := []struct {
		name    string
		files   []PackageFile
		wantErr bool
	}{
		{
			name: "valid files",
			files: []PackageFile{
				{TargetPath: "lib/net6.0/MyLib.dll", Content: []byte("test")},
				{TargetPath: "content/readme.txt", Content: []byte("test")},
			},
			wantErr: false,
		},
		{
			name:    "no files",
			files:   []PackageFile{},
			wantErr: true,
		},
		{
			name: "duplicate files",
			files: []PackageFile{
				{TargetPath: "lib/net6.0/MyLib.dll", Content: []byte("test")},
				{TargetPath: "lib/NET6.0/MyLib.dll", Content: []byte("test")},
			},
			wantErr: true,
		},
		{
			name: "invalid path - traversal",
			files: []PackageFile{
				{TargetPath: "../../../etc/passwd", Content: []byte("test")},
			},
			wantErr: true,
		},
		{
			name: "invalid path - absolute",
			files: []PackageFile{
				{TargetPath: "/etc/passwd", Content: []byte("test")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFiles(tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLicense(t *testing.T) {
	tests := []struct {
		name     string
		metadata PackageMetadata
		files    []PackageFile
		wantErr  bool
	}{
		{
			name: "valid - no license acceptance required",
			metadata: PackageMetadata{
				RequireLicenseAcceptance: false,
			},
			files:   []PackageFile{},
			wantErr: false,
		},
		{
			name: "valid - license URL with acceptance",
			metadata: PackageMetadata{
				RequireLicenseAcceptance: true,
				LicenseURL:               mustParseURL("https://example.com/license"),
			},
			files:   []PackageFile{},
			wantErr: false,
		},
		{
			name: "valid - license file with acceptance",
			metadata: PackageMetadata{
				RequireLicenseAcceptance: true,
				LicenseMetadata: &LicenseMetadata{
					Type: "file",
					Text: "LICENSE.txt",
				},
			},
			files: []PackageFile{
				{TargetPath: "LICENSE.txt", Content: []byte("license text")},
			},
			wantErr: false,
		},
		{
			name: "invalid - acceptance required but no license",
			metadata: PackageMetadata{
				RequireLicenseAcceptance: true,
			},
			files:   []PackageFile{},
			wantErr: true,
		},
		{
			name: "invalid - both license URL and metadata",
			metadata: PackageMetadata{
				LicenseURL: mustParseURL("https://example.com/license"),
				LicenseMetadata: &LicenseMetadata{
					Type: "expression",
					Text: "MIT",
				},
			},
			files:   []PackageFile{},
			wantErr: true,
		},
		{
			name: "invalid - license file not found",
			metadata: PackageMetadata{
				LicenseMetadata: &LicenseMetadata{
					Type: "file",
					Text: "LICENSE.txt",
				},
			},
			files:   []PackageFile{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLicense(tt.metadata, tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLicense() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIcon(t *testing.T) {
	tests := []struct {
		name     string
		metadata PackageMetadata
		files    []PackageFile
		wantErr  bool
	}{
		{
			name:     "no icon",
			metadata: PackageMetadata{Icon: ""},
			files:    []PackageFile{},
			wantErr:  false,
		},
		{
			name:     "valid icon in icon folder",
			metadata: PackageMetadata{Icon: "icon/icon.png"},
			files: []PackageFile{
				{TargetPath: "icon/icon.png", Content: []byte("png")},
			},
			wantErr: false,
		},
		{
			name:     "valid icon at root",
			metadata: PackageMetadata{Icon: "icon.png"},
			files: []PackageFile{
				{TargetPath: "icon.png", Content: []byte("png")},
			},
			wantErr: false,
		},
		{
			name:     "icon file not found",
			metadata: PackageMetadata{Icon: "icon/icon.png"},
			files:    []PackageFile{},
			wantErr:  true,
		},
		{
			name:     "icon in wrong folder",
			metadata: PackageMetadata{Icon: "images/icon.png"},
			files: []PackageFile{
				{TargetPath: "images/icon.png", Content: []byte("png")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIcon(tt.metadata, tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIcon() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReadme(t *testing.T) {
	tests := []struct {
		name     string
		metadata PackageMetadata
		files    []PackageFile
		wantErr  bool
	}{
		{
			name:     "no readme",
			metadata: PackageMetadata{Readme: ""},
			files:    []PackageFile{},
			wantErr:  false,
		},
		{
			name:     "valid readme",
			metadata: PackageMetadata{Readme: "README.md"},
			files: []PackageFile{
				{TargetPath: "README.md", Content: []byte("readme")},
			},
			wantErr: false,
		},
		{
			name:     "readme file not found",
			metadata: PackageMetadata{Readme: "README.md"},
			files:    []PackageFile{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadme(tt.metadata, tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReadme() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFrameworkReferences(t *testing.T) {
	tests := []struct {
		name    string
		groups  []PackageFrameworkReferenceGroup
		wantErr bool
	}{
		{
			name: "valid framework references",
			groups: []PackageFrameworkReferenceGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					References:      []string{"System.Net.Http"},
				},
			},
			wantErr: false,
		},
		{
			name: "no target framework",
			groups: []PackageFrameworkReferenceGroup{
				{
					TargetFramework: nil,
					References:      []string{"System.Net.Http"},
				},
			},
			wantErr: true,
		},
		{
			name: "no references",
			groups: []PackageFrameworkReferenceGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					References:      []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate references",
			groups: []PackageFrameworkReferenceGroup{
				{
					TargetFramework: frameworks.MustParseFramework("net6.0"),
					References:      []string{"System.Net.Http", "system.net.http"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrameworkReferences(tt.groups)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFrameworkReferences() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuilderValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *PackageBuilder
		wantErr bool
	}{
		{
			name: "valid package with files",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetDescription("Test package")
				b.SetAuthors("Test Author")
				_ = b.AddFileFromBytes("lib/net6.0/MyLib.dll", []byte("test"))
				return b
			},
			wantErr: false,
		},
		{
			name: "valid package with dependencies only",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetDescription("Test package")
				b.SetAuthors("Test Author")
				b.AddDependency(frameworks.MustParseFramework("net6.0"), "Dep1", version.MustParseRange("1.0.0"))
				return b
			},
			wantErr: false,
		},
		{
			name: "invalid - bad package ID",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("123Invalid")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetDescription("Test package")
				b.SetAuthors("Test Author")
				_ = b.AddFileFromBytes("lib/net6.0/MyLib.dll", []byte("test"))
				return b
			},
			wantErr: true,
		},
		{
			name: "invalid - no version",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetDescription("Test package")
				b.SetAuthors("Test Author")
				_ = b.AddFileFromBytes("lib/net6.0/MyLib.dll", []byte("test"))
				return b
			},
			wantErr: true,
		},
		{
			name: "invalid - no description",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetAuthors("Test Author")
				_ = b.AddFileFromBytes("lib/net6.0/MyLib.dll", []byte("test"))
				return b
			},
			wantErr: true,
		},
		{
			name: "invalid - no authors",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetDescription("Test package")
				_ = b.AddFileFromBytes("lib/net6.0/MyLib.dll", []byte("test"))
				return b
			},
			wantErr: true,
		},
		{
			name: "invalid - empty package",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetDescription("Test package")
				b.SetAuthors("Test Author")
				return b
			},
			wantErr: true,
		},
		{
			name: "invalid - self dependency",
			setup: func() *PackageBuilder {
				b := NewPackageBuilder()
				b.SetID("MyPackage")
				b.SetVersion(version.MustParse("1.0.0"))
				b.SetDescription("Test package")
				b.SetAuthors("Test Author")
				b.AddDependency(frameworks.MustParseFramework("net6.0"), "MyPackage", version.MustParseRange("1.0.0"))
				return b
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.setup()
			err := b.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
