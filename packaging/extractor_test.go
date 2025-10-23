package packaging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

func TestPackageSaveMode_HasFlag(t *testing.T) {
	tests := []struct {
		name     string
		mode     PackageSaveMode
		flag     PackageSaveMode
		expected bool
	}{
		{
			name:     "Has nuspec flag",
			mode:     PackageSaveModeNuspec | PackageSaveModeFiles,
			flag:     PackageSaveModeNuspec,
			expected: true,
		},
		{
			name:     "Does not have nupkg flag",
			mode:     PackageSaveModeNuspec | PackageSaveModeFiles,
			flag:     PackageSaveModeNupkg,
			expected: false,
		},
		{
			name:     "V2 default has nupkg",
			mode:     PackageSaveModeDefaultV2,
			flag:     PackageSaveModeNupkg,
			expected: true,
		},
		{
			name:     "V2 default does not have nuspec",
			mode:     PackageSaveModeDefaultV2,
			flag:     PackageSaveModeNuspec,
			expected: false,
		},
		{
			name:     "V3 default has all",
			mode:     PackageSaveModeDefaultV3,
			flag:     PackageSaveModeNuspec | PackageSaveModeNupkg | PackageSaveModeFiles,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.HasFlag(tt.flag); got != tt.expected {
				t.Errorf("HasFlag() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPackagePathResolver(t *testing.T) {
	ver := version.MustParse("1.2.3")
	identity := &PackageIdentity{
		ID:      "Newtonsoft.Json",
		Version: ver,
	}

	tests := []struct {
		name               string
		rootDirectory      string
		useSideBySidePaths bool
		wantInstallPath    string
		wantPackageFile    string
		wantManifest       string
	}{
		{
			name:               "Side-by-side paths",
			rootDirectory:      "/packages",
			useSideBySidePaths: true,
			wantInstallPath:    filepath.Join("/packages", "Newtonsoft.Json.1.2.3"),
			wantPackageFile:    filepath.Join("/packages", "Newtonsoft.Json.1.2.3", "Newtonsoft.Json.1.2.3.nupkg"),
			wantManifest:       "Newtonsoft.Json.nuspec",
		},
		{
			name:               "Single directory",
			rootDirectory:      "/packages",
			useSideBySidePaths: false,
			wantInstallPath:    filepath.Join("/packages", "Newtonsoft.Json"),
			wantPackageFile:    filepath.Join("/packages", "Newtonsoft.Json", "Newtonsoft.Json.1.2.3.nupkg"),
			wantManifest:       "Newtonsoft.Json.nuspec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewPackagePathResolver(tt.rootDirectory, tt.useSideBySidePaths)

			if got := resolver.GetInstallPath(identity); got != tt.wantInstallPath {
				t.Errorf("GetInstallPath() = %v, want %v", got, tt.wantInstallPath)
			}

			if got := resolver.GetPackageFilePath(identity); got != tt.wantPackageFile {
				t.Errorf("GetPackageFilePath() = %v, want %v", got, tt.wantPackageFile)
			}

			if got := resolver.GetManifestFileName(identity); got != tt.wantManifest {
				t.Errorf("GetManifestFileName() = %v, want %v", got, tt.wantManifest)
			}
		})
	}
}

func TestVersionFolderPathResolver(t *testing.T) {
	ver := version.MustParse("1.2.3-beta")

	tests := []struct {
		name            string
		rootPath        string
		isLowercase     bool
		packageID       string
		wantInstallPath string
		wantPackageFile string
		wantManifest    string
		wantHash        string
		wantMetadata    string
	}{
		{
			name:            "Lowercase normalization",
			rootPath:        "/packages",
			isLowercase:     true,
			packageID:       "Newtonsoft.Json",
			wantInstallPath: filepath.Join("/packages", "newtonsoft.json", "1.2.3-beta"),
			wantPackageFile: filepath.Join("/packages", "newtonsoft.json", "1.2.3-beta", "newtonsoft.json.1.2.3-beta.nupkg"),
			wantManifest:    filepath.Join("/packages", "newtonsoft.json", "1.2.3-beta", "newtonsoft.json.nuspec"),
			wantHash:        filepath.Join("/packages", "newtonsoft.json", "1.2.3-beta", "newtonsoft.json.1.2.3-beta.nupkg.sha512"),
			wantMetadata:    filepath.Join("/packages", "newtonsoft.json", "1.2.3-beta", ".nupkg.metadata"),
		},
		{
			name:            "No lowercase normalization",
			rootPath:        "/packages",
			isLowercase:     false,
			packageID:       "Newtonsoft.Json",
			wantInstallPath: filepath.Join("/packages", "Newtonsoft.Json", "1.2.3-beta"),
			wantPackageFile: filepath.Join("/packages", "Newtonsoft.Json", "1.2.3-beta", "Newtonsoft.Json.1.2.3-beta.nupkg"),
			wantManifest:    filepath.Join("/packages", "Newtonsoft.Json", "1.2.3-beta", "Newtonsoft.Json.nuspec"),
			wantHash:        filepath.Join("/packages", "Newtonsoft.Json", "1.2.3-beta", "Newtonsoft.Json.1.2.3-beta.nupkg.sha512"),
			wantMetadata:    filepath.Join("/packages", "Newtonsoft.Json", "1.2.3-beta", ".nupkg.metadata"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewVersionFolderPathResolver(tt.rootPath, tt.isLowercase)

			if got := resolver.GetInstallPath(tt.packageID, ver); got != tt.wantInstallPath {
				t.Errorf("GetInstallPath() = %v, want %v", got, tt.wantInstallPath)
			}

			if got := resolver.GetPackageFilePath(tt.packageID, ver); got != tt.wantPackageFile {
				t.Errorf("GetPackageFilePath() = %v, want %v", got, tt.wantPackageFile)
			}

			if got := resolver.GetManifestFilePath(tt.packageID, ver); got != tt.wantManifest {
				t.Errorf("GetManifestFilePath() = %v, want %v", got, tt.wantManifest)
			}

			if got := resolver.GetHashPath(tt.packageID, ver); got != tt.wantHash {
				t.Errorf("GetHashPath() = %v, want %v", got, tt.wantHash)
			}

			if got := resolver.GetNupkgMetadataPath(tt.packageID, ver); got != tt.wantMetadata {
				t.Errorf("GetNupkgMetadataPath() = %v, want %v", got, tt.wantMetadata)
			}
		})
	}
}

func TestNupkgMetadataFile(t *testing.T) {
	metadata := NewNupkgMetadataFile("abc123hash", "https://example.com/packages")

	if metadata.Version != 2 {
		t.Errorf("Version = %d, want 2", metadata.Version)
	}

	if metadata.ContentHash != "abc123hash" {
		t.Errorf("ContentHash = %s, want abc123hash", metadata.ContentHash)
	}

	if metadata.Source != "https://example.com/packages" {
		t.Errorf("Source = %s, want https://example.com/packages", metadata.Source)
	}

	// Test write and read
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".nupkg.metadata")

	if err := metadata.WriteToFile(path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	read, err := ReadNupkgMetadataFile(path)
	if err != nil {
		t.Fatalf("ReadNupkgMetadataFile() error = %v", err)
	}

	if read.Version != metadata.Version {
		t.Errorf("Read Version = %d, want %d", read.Version, metadata.Version)
	}

	if read.ContentHash != metadata.ContentHash {
		t.Errorf("Read ContentHash = %s, want %s", read.ContentHash, metadata.ContentHash)
	}

	if read.Source != metadata.Source {
		t.Errorf("Read Source = %s, want %s", read.Source, metadata.Source)
	}
}

func TestShouldExcludeFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Content types", "[content_types].xml", true},
		{"Rels folder", "_rels/.rels", true},
		{"Package folder", "package/services/metadata/core-properties/abc.psmdcp", true},
		{"PSMDCP file", "foo.psmdcp", true},
		{"Root nupkg", "package.nupkg", true},
		{"Root nuspec", "package.nuspec", true},
		{"Nested nupkg", "lib/net45/package.nupkg", false},
		{"DLL file", "lib/net45/Newtonsoft.Json.dll", false},
		{"XML doc", "lib/net45/Newtonsoft.Json.xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldExcludeFile(tt.path); got != tt.expected {
				t.Errorf("shouldExcludeFile(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestXMLDocIdentification(t *testing.T) {
	packageFiles := []string{
		"lib/net45/Newtonsoft.Json.dll",
		"lib/net45/Newtonsoft.Json.xml",
		"lib/net45/OtherFile.xml", // No corresponding DLL
		"ref/net6.0/MyLib.dll",
		"ref/net6.0/MyLib.xml",
		"content/data.xml", // Not in lib/ or ref/
	}

	extractor := NewPackageFileExtractor(packageFiles, XMLDocFileSaveModeCompress)

	tests := []struct {
		name       string
		file       string
		isXmlDoc   bool
		shouldSkip bool
	}{
		{"Newtonsoft.Json.xml is doc", "lib/net45/Newtonsoft.Json.xml", true, false},
		{"MyLib.xml is doc", "ref/net6.0/MyLib.xml", true, false},
		{"OtherFile.xml is not doc", "lib/net45/OtherFile.xml", false, false},
		{"data.xml is not doc", "content/data.xml", false, false},
		{"DLL is not doc", "lib/net45/Newtonsoft.Json.dll", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractor.xmlDocFiles[tt.file]; got != tt.isXmlDoc {
				t.Errorf("xmlDocFiles[%q] = %v, want %v", tt.file, got, tt.isXmlDoc)
			}
		})
	}
}

func TestDefaultExtractionContext(t *testing.T) {
	ctx := DefaultExtractionContext()

	if ctx.PackageSaveMode != PackageSaveModeDefaultV3 {
		t.Errorf("PackageSaveMode = %v, want %v", ctx.PackageSaveMode, PackageSaveModeDefaultV3)
	}

	if ctx.XMLDocFileSaveMode != XMLDocFileSaveModeNone {
		t.Errorf("XMLDocFileSaveMode = %v, want %v", ctx.XMLDocFileSaveMode, XMLDocFileSaveModeNone)
	}

	if !ctx.CopySatelliteFiles {
		t.Errorf("CopySatelliteFiles = %v, want true", ctx.CopySatelliteFiles)
	}
}

func TestGenerateTempFileName(t *testing.T) {
	// Generate multiple temp file names and ensure they're unique
	names := make(map[string]bool)
	for range 100 {
		name := generateTempFileName()

		if len(name) != 32 { // 16 bytes -> 32 hex chars
			t.Errorf("generateTempFileName() length = %d, want 32", len(name))
		}

		if names[name] {
			t.Errorf("generateTempFileName() generated duplicate: %s", name)
		}

		names[name] = true
	}
}

func TestCleanDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create some files and subdirectories
	_ = os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("test"), 0644)
	_ = os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "subdir", "file3.txt"), []byte("test"), 0644)

	// Clean directory
	if err := cleanDirectory(tempDir); err != nil {
		t.Fatalf("cleanDirectory() error = %v", err)
	}

	// Verify directory is empty but still exists
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Directory not empty after clean, entries = %d", len(entries))
	}

	// Verify directory itself still exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Directory was deleted instead of cleaned")
	}
}

func TestIsRootMetadata(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Root nuspec", "package.nuspec", true},
		{"Root nupkg", "package.nupkg", true},
		{"Nested nuspec", "lib/net45/package.nuspec", false},
		{"Nested nupkg", "lib/net45/package.nupkg", false},
		{"Regular file", "readme.txt", false},
		{"DLL file", "lib.dll", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRootMetadata(tt.path); got != tt.expected {
				t.Errorf("isRootMetadata(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsMetadataFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"SHA512 hash", "package.nupkg.sha512", true},
		{"Metadata file", ".nupkg.metadata", true},
		{"Regular nupkg", "package.nupkg", false},
		{"Nuspec", "package.nuspec", false},
		{"DLL", "lib.dll", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMetadataFile(tt.path); got != tt.expected {
				t.Errorf("isMetadataFile(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
