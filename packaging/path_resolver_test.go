package packaging

import (
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

// TestGetPackageDownloadMarkerFileName tests V2 download marker filename
func TestGetPackageDownloadMarkerFileName(t *testing.T) {
	resolver := NewPackagePathResolver("/packages", false)

	identity := &PackageIdentity{
		ID:      "TestPackage",
		Version: version.MustParse("1.0.0"),
	}

	markerFile := resolver.GetPackageDownloadMarkerFileName(identity)
	expected := "TestPackage.packagedownload.marker"

	if markerFile != expected {
		t.Errorf("GetPackageDownloadMarkerFileName() = %q, want %q", markerFile, expected)
	}
}

// TestGetVersionListDirectory tests V3 version list directory
func TestGetVersionListDirectory(t *testing.T) {
	tests := []struct {
		name          string
		rootPath      string
		packageID     string
		useSideBySide bool
		want          string
	}{
		{
			name:          "Lowercase normalized ID",
			rootPath:      "/global-packages",
			packageID:     "NuGet.Versioning",
			useSideBySide: true,
			want:          filepath.Join("/global-packages", "nuget.versioning"),
		},
		{
			name:          "Already lowercase ID",
			rootPath:      "/global-packages",
			packageID:     "testpackage",
			useSideBySide: true,
			want:          filepath.Join("/global-packages", "testpackage"),
		},
		{
			name:          "Mixed case ID",
			rootPath:      "/global-packages",
			packageID:     "MyPackage.Core",
			useSideBySide: true,
			want:          filepath.Join("/global-packages", "mypackage.core"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewVersionFolderPathResolver(tt.rootPath, tt.useSideBySide)
			got := resolver.GetVersionListDirectory(tt.packageID)

			if got != tt.want {
				t.Errorf("GetVersionListDirectory() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPackagePathResolver_GetInstallPath tests V2 install path
func TestPackagePathResolver_GetInstallPath(t *testing.T) {
	tests := []struct {
		name          string
		rootPath      string
		packageID     string
		version       string
		useSideBySide bool
		want          string
	}{
		{
			name:          "Side-by-side layout",
			rootPath:      "/packages",
			packageID:     "TestPackage",
			version:       "1.0.0",
			useSideBySide: true,
			want:          filepath.Join("/packages", "TestPackage.1.0.0"),
		},
		{
			name:          "Non-side-by-side layout",
			rootPath:      "/packages",
			packageID:     "TestPackage",
			version:       "1.0.0",
			useSideBySide: false,
			want:          filepath.Join("/packages", "TestPackage"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewPackagePathResolver(tt.rootPath, tt.useSideBySide)
			identity := &PackageIdentity{
				ID:      tt.packageID,
				Version: version.MustParse(tt.version),
			}

			got := resolver.GetInstallPath(identity)

			if got != tt.want {
				t.Errorf("GetInstallPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestVersionFolderPathResolver_GetPackageDirectory tests V3 package directory
func TestVersionFolderPathResolver_GetPackageDirectory(t *testing.T) {
	tests := []struct {
		name      string
		rootPath  string
		packageID string
		version   string
		want      string
	}{
		{
			name:      "Lowercase normalized paths",
			rootPath:  "/global-packages",
			packageID: "NuGet.Versioning",
			version:   "5.0.0",
			want:      filepath.Join("/global-packages", "nuget.versioning", "5.0.0"),
		},
		{
			name:      "Prerelease version",
			rootPath:  "/global-packages",
			packageID: "TestPackage",
			version:   "1.0.0-beta",
			want:      filepath.Join("/global-packages", "testpackage", "1.0.0-beta"),
		},
		{
			name:      "Version with metadata",
			rootPath:  "/global-packages",
			packageID: "MyPackage",
			version:   "2.0.0+build.123",
			want:      filepath.Join("/global-packages", "mypackage", "2.0.0+build.123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewVersionFolderPathResolver(tt.rootPath, true)
			v := version.MustParse(tt.version)

			got := resolver.GetPackageDirectory(tt.packageID, v)

			if got != tt.want {
				t.Errorf("GetPackageDirectory() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestVersionFolderPathResolver_Paths tests all V3 path functions
func TestVersionFolderPathResolver_Paths(t *testing.T) {
	resolver := NewVersionFolderPathResolver("/global-packages", true)
	packageID := "TestPackage"
	ver := version.MustParse("1.0.0")

	// Test GetPackageFilePath
	nupkgPath := resolver.GetPackageFilePath(packageID, ver)
	expectedNupkg := filepath.Join("/global-packages", "testpackage", "1.0.0", "testpackage.1.0.0.nupkg")
	if nupkgPath != expectedNupkg {
		t.Errorf("GetPackageFilePath() = %q, want %q", nupkgPath, expectedNupkg)
	}

	// Test GetManifestFilePath
	nuspecPath := resolver.GetManifestFilePath(packageID, ver)
	expectedNuspec := filepath.Join("/global-packages", "testpackage", "1.0.0", "testpackage.nuspec")
	if nuspecPath != expectedNuspec {
		t.Errorf("GetManifestFilePath() = %q, want %q", nuspecPath, expectedNuspec)
	}

	// Test GetHashPath
	hashPath := resolver.GetHashPath(packageID, ver)
	expectedHash := filepath.Join("/global-packages", "testpackage", "1.0.0", "testpackage.1.0.0.nupkg.sha512")
	if hashPath != expectedHash {
		t.Errorf("GetHashPath() = %q, want %q", hashPath, expectedHash)
	}

	// Test GetNupkgMetadataPath
	metadataPath := resolver.GetNupkgMetadataPath(packageID, ver)
	expectedMetadata := filepath.Join("/global-packages", "testpackage", "1.0.0", ".nupkg.metadata")
	if metadataPath != expectedMetadata {
		t.Errorf("GetNupkgMetadataPath() = %q, want %q", metadataPath, expectedMetadata)
	}
}
