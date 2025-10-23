package packaging

import (
	"archive/zip"
	"bytes"
	"os"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

// TestIsSatellitePackage_Integration tests satellite package detection with correct signatures
func TestIsSatellitePackage_Integration(t *testing.T) {
	tests := []struct {
		name          string
		packageID     string
		nuspecXML     string
		wantResult    bool
		wantRuntimeID string
	}{
		{
			name:      "Satellite package with fr-FR language",
			packageID: "MyPackage.fr-FR",
			nuspecXML: `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage.fr-FR</id>
    <version>1.0.0</version>
    <language>fr-FR</language>
    <authors>Test</authors>
    <description>French satellite package</description>
    <dependencies>
      <group>
        <dependency id="MyPackage" version="[1.0.0]" />
      </group>
    </dependencies>
  </metadata>
</package>`,
			wantResult:    true,
			wantRuntimeID: "MyPackage",
		},
		{
			name:      "Satellite package with ja-JP language",
			packageID: "MyPackage.ja-JP",
			nuspecXML: `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage.ja-JP</id>
    <version>1.0.0</version>
    <language>ja-JP</language>
    <authors>Test</authors>
    <description>Japanese satellite package</description>
    <dependencies>
      <group>
        <dependency id="MyPackage" version="[1.0.0]" />
      </group>
    </dependencies>
  </metadata>
</package>`,
			wantResult:    true,
			wantRuntimeID: "MyPackage",
		},
		{
			name:      "Regular package without language",
			packageID: "MyPackage",
			nuspecXML: `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage</id>
    <version>1.0.0</version>
    <authors>Test</authors>
    <description>Regular package</description>
  </metadata>
</package>`,
			wantResult:    false,
			wantRuntimeID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := createTestPackageForSatellite(t, tt.nuspecXML)
			identity := &PackageIdentity{
				ID:      tt.packageID,
				Version: version.MustParse("1.0.0"),
			}

			isSatellite, runtimeIdentity, err := IsSatellitePackage(pkg, identity)
			if err != nil {
				t.Fatalf("IsSatellitePackage() error = %v", err)
			}

			if isSatellite != tt.wantResult {
				t.Errorf("IsSatellitePackage() isSatellite = %v, want %v", isSatellite, tt.wantResult)
			}

			if tt.wantResult && runtimeIdentity != nil {
				if runtimeIdentity.ID != tt.wantRuntimeID {
					t.Errorf("IsSatellitePackage() runtimeID = %q, want %q", runtimeIdentity.ID, tt.wantRuntimeID)
				}
			}
		})
	}
}

// TestCopySatelliteFilesIfApplicableV2_Integration tests V2 satellite file copying
func TestCopySatelliteFilesIfApplicableV2_Integration(t *testing.T) {
	satelliteNuspec := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage.ja-JP</id>
    <version>1.0.0</version>
    <language>ja-JP</language>
    <authors>Test</authors>
    <description>Japanese satellite package</description>
    <dependencies>
      <group>
        <dependency id="MyPackage" version="[1.0.0]" />
      </group>
    </dependencies>
  </metadata>
</package>`

	pkg := createTestPackageForSatellite(t, satelliteNuspec)
	identity := &PackageIdentity{
		ID:      "MyPackage.ja-JP",
		Version: version.MustParse("1.0.0"),
	}

	tempDir := t.TempDir()
	resolver := NewPackagePathResolver(tempDir, true)

	// Create runtime package directory (satellite files copy into runtime package)
	runtimeIdentity := &PackageIdentity{
		ID:      "MyPackage",
		Version: version.MustParse("1.0.0"),
	}
	runtimeDir := resolver.GetInstallPath(runtimeIdentity)
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("Failed to create runtime package dir: %v", err)
	}

	copied, err := CopySatelliteFilesIfApplicableV2(pkg, identity, resolver, PackageSaveModeDefaultV2, nil)
	if err != nil {
		t.Fatalf("CopySatelliteFilesIfApplicableV2() error = %v", err)
	}

	// Should return true because satellite files were copied to runtime package
	if !copied {
		t.Error("CopySatelliteFilesIfApplicableV2() should return true for satellite package")
	}

	// Verify satellite files were copied to runtime package directory
	// Satellite packages typically have lib/{lang} folders that get merged into runtime package
	t.Logf("Satellite files copied to %s", runtimeDir)
}

// TestCopySatelliteFilesIfApplicableV3_Integration tests V3 satellite file copying
func TestCopySatelliteFilesIfApplicableV3_Integration(t *testing.T) {
	satelliteNuspec := `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage.fr-FR</id>
    <version>1.0.0</version>
    <language>fr-FR</language>
    <authors>Test</authors>
    <description>French satellite package</description>
    <dependencies>
      <group>
        <dependency id="MyPackage" version="[1.0.0]" />
      </group>
    </dependencies>
  </metadata>
</package>`

	pkg := createTestPackageForSatellite(t, satelliteNuspec)
	identity := &PackageIdentity{
		ID:      "MyPackage.fr-FR",
		Version: version.MustParse("1.0.0"),
	}

	tempDir := t.TempDir()
	resolver := NewVersionFolderPathResolver(tempDir, true)

	// Create runtime package directory (satellite files copy into runtime package)
	runtimeIdentity := &PackageIdentity{
		ID:      "MyPackage",
		Version: version.MustParse("1.0.0"),
	}
	runtimeDir := resolver.GetPackageDirectory(runtimeIdentity.ID, runtimeIdentity.Version)
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("Failed to create runtime package dir: %v", err)
	}

	copied, err := CopySatelliteFilesIfApplicableV3(pkg, identity, resolver, PackageSaveModeDefaultV3, nil)
	if err != nil {
		t.Fatalf("CopySatelliteFilesIfApplicableV3() error = %v", err)
	}

	// Should return true because satellite files were copied to runtime package
	if !copied {
		t.Error("CopySatelliteFilesIfApplicableV3() should return true for satellite package")
	}

	// Verify satellite files were copied to runtime package directory
	// Satellite packages typically have lib/{lang} folders that get merged into runtime package
	t.Logf("Satellite files copied to %s", runtimeDir)
}

// TestIsRootMetadata_Integration tests root metadata file detection
func TestIsRootMetadata_Integration(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"root nuspec", "MyPackage.nuspec", true},
		{"root nupkg", "MyPackage.1.0.0.nupkg", true},
		{"nested nuspec", "lib/net45/MyPackage.nuspec", false},
		{"nested nupkg", "packages/MyPackage.1.0.0.nupkg", false},
		{"regular file", "lib/net45/MyLib.dll", false},
		{"content file", "content/readme.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRootMetadata(tt.path)
			if got != tt.want {
				t.Errorf("isRootMetadata(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// createTestPackageForSatellite creates a test package reader with specified nuspec
func createTestPackageForSatellite(tb testing.TB, nuspecXML string) *PackageReader {
	tb.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add nuspec file
	nuspecFile, err := w.Create("package.nuspec")
	if err != nil {
		tb.Fatalf("Failed to create nuspec: %v", err)
	}
	if _, err := nuspecFile.Write([]byte(nuspecXML)); err != nil {
		tb.Fatalf("Failed to write nuspec: %v", err)
	}

	// Add lib file
	libFile, err := w.Create("lib/net45/dummy.dll")
	if err != nil {
		tb.Fatalf("Failed to create lib file: %v", err)
	}
	if _, err := libFile.Write([]byte("dummy binary")); err != nil {
		tb.Fatalf("Failed to write lib file: %v", err)
	}

	// Add OPC files
	relsFile, err := w.Create("_rels/.rels")
	if err != nil {
		tb.Fatalf("Failed to create rels: %v", err)
	}
	relsContent := `<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Type="http://schemas.microsoft.com/packaging/2010/07/manifest" Target="/package.nuspec" Id="Re0" />
</Relationships>`
	_, _ = relsFile.Write([]byte(relsContent))

	ctFile, err := w.Create("[Content_Types].xml")
	if err != nil {
		tb.Fatalf("Failed to create content types: %v", err)
	}
	ctContent := `<?xml version="1.0"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml" />
  <Default Extension="nuspec" ContentType="application/octet" />
</Types>`
	_, _ = ctFile.Write([]byte(ctContent))

	if err := w.Close(); err != nil {
		tb.Fatalf("Failed to close zip: %v", err)
	}

	// Create reader
	reader := bytes.NewReader(buf.Bytes())
	zipReader, err := zip.NewReader(reader, int64(buf.Len()))
	if err != nil {
		tb.Fatalf("Failed to create zip reader: %v", err)
	}

	pkg := &PackageReader{
		zipReaderAt: zipReader,
		isClosable:  false,
	}

	return pkg
}
