package restore

import (
	"context"
	"os"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

func TestLocalDependencyProvider_VersionRangeResolution(t *testing.T) {
	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	packageID := "TestPackage"

	// Create multiple cached versions
	versions := []string{"1.0.0", "2.0.0", "2.5.0", "3.0.0", "4.0.0-beta"}

	for _, verStr := range versions {
		ver, _ := version.Parse(verStr)
		installPath := provider.resolver.GetInstallPath(packageID, ver)
		os.MkdirAll(installPath, 0755)

		// Create completion marker
		metadataPath := provider.resolver.GetNupkgMetadataPath(packageID, ver)
		os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)

		// Create nuspec
		nuspecPath := provider.resolver.GetManifestFilePath(packageID, ver)
		nuspecContent := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>` + verStr + `</version>
    <description>Test</description>
    <authors>Test</authors>
    <dependencies>
      <group targetFramework="net6.0" />
    </dependencies>
  </metadata>
</package>`
		os.WriteFile(nuspecPath, []byte(nuspecContent), 0644)
	}

	ctx := context.Background()

	tests := []struct {
		name           string
		versionRange   string
		expectedVersion string
		shouldFind     bool
	}{
		{
			name:           "open range from 2.0.0",
			versionRange:   "[2.0.0, )",
			expectedVersion: "2.0.0", // Favor lower: minimum satisfying version
			shouldFind:     true,
		},
		{
			name:           "closed range",
			versionRange:   "[1.0.0, 2.5.0]",
			expectedVersion: "1.0.0", // Favor lower: minimum satisfying version
			shouldFind:     true,
		},
		{
			name:           "exclusive lower bound",
			versionRange:   "(2.0.0, 3.0.0]",
			expectedVersion: "2.5.0", // Favor lower: 2.0.0 excluded, so 2.5.0 is minimum
			shouldFind:     true,
		},
		{
			name:           "minimum version only",
			versionRange:   "2.5.0",
			expectedVersion: "2.5.0", // Favor lower: 2.5.0 is minimum satisfying >= 2.5.0
			shouldFind:     true,
		},
		{
			name:           "range with no match",
			versionRange:   "[5.0.0, )",
			expectedVersion: "",
			shouldFind:     false,
		},
		{
			name:           "range excluding all stable versions",
			versionRange:   "[4.0.0, )",
			expectedVersion: "", // 4.0.0-beta doesn't satisfy ">= 4.0.0" (prerelease < release)
			shouldFind:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, resolvedVer, err := provider.GetDependencies(ctx, packageID, tt.versionRange)

			if err != nil {
				t.Fatalf("GetDependencies() error = %v, want nil", err)
			}

			if tt.shouldFind {
				if groups == nil {
					t.Fatalf("GetDependencies() returned nil, want non-nil")
				}

				if resolvedVer != tt.expectedVersion {
					t.Errorf("GetDependencies() resolved version = %s, want %s", resolvedVer, tt.expectedVersion)
				}

				if len(groups) != 1 {
					t.Errorf("GetDependencies() returned %d groups, want 1", len(groups))
				}
			} else {
				if groups != nil {
					t.Errorf("GetDependencies() returned %v, want nil (no match)", groups)
				}

				if resolvedVer != "" {
					t.Errorf("GetDependencies() resolved version = %s, want empty", resolvedVer)
				}
			}
		})
	}
}

func TestLocalDependencyProvider_ExactVersionHandling(t *testing.T) {
	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	packageID := "ExactTest"
	packageVersion := "1.2.3"
	ver, _ := version.Parse(packageVersion)

	// Create package
	installPath := provider.resolver.GetInstallPath(packageID, ver)
	os.MkdirAll(installPath, 0755)

	metadataPath := provider.resolver.GetNupkgMetadataPath(packageID, ver)
	os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)

	nuspecPath := provider.resolver.GetManifestFilePath(packageID, ver)
	nuspecContent := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>ExactTest</id>
    <version>1.2.3</version>
    <description>Test</description>
    <authors>Test</authors>
  </metadata>
</package>`
	os.WriteFile(nuspecPath, []byte(nuspecContent), 0644)

	ctx := context.Background()

	// Test exact version (no range chars)
	groups, resolvedVer, err := provider.GetDependencies(ctx, packageID, packageVersion)

	if err != nil {
		t.Fatalf("GetDependencies() error = %v, want nil", err)
	}

	if groups == nil {
		t.Fatal("GetDependencies() returned nil, want non-nil")
	}

	if resolvedVer != packageVersion {
		t.Errorf("GetDependencies() resolved version = %s, want %s", resolvedVer, packageVersion)
	}
}
