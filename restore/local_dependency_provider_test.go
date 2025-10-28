package restore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

func TestLocalDependencyProvider_PackageExists(t *testing.T) {
	// Create temp packages folder
	tempDir := t.TempDir()

	provider := NewLocalDependencyProvider(tempDir)

	tests := []struct {
		name           string
		packageID      string
		packageVersion string
		setupFunc      func(string, string)
		expectedExists bool
	}{
		{
			name:           "package with metadata file exists",
			packageID:      "Newtonsoft.Json",
			packageVersion: "13.0.1",
			setupFunc: func(id, ver string) {
				// Create metadata file (primary completion marker)
				v, _ := version.Parse(ver)
				metadataPath := provider.resolver.GetNupkgMetadataPath(id, v)
				_ = os.MkdirAll(filepath.Dir(metadataPath), 0755)
				_ = os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)
			},
			expectedExists: true,
		},
		{
			name:           "package with hash file exists (fallback)",
			packageID:      "Serilog",
			packageVersion: "2.10.0",
			setupFunc: func(id, ver string) {
				// Create hash file (fallback completion marker)
				v, _ := version.Parse(ver)
				hashPath := provider.resolver.GetHashPath(id, v)
				_ = os.MkdirAll(filepath.Dir(hashPath), 0755)
				_ = os.WriteFile(hashPath, []byte("hash123"), 0644)
			},
			expectedExists: true,
		},
		{
			name:           "package does not exist",
			packageID:      "NonExistent",
			packageVersion: "1.0.0",
			setupFunc:      func(id, ver string) {}, // No setup
			expectedExists: false,
		},
		{
			name:           "package directory exists but no completion marker",
			packageID:      "Incomplete",
			packageVersion: "1.0.0",
			setupFunc: func(id, ver string) {
				// Create directory but no completion markers
				v, _ := version.Parse(ver)
				installPath := provider.resolver.GetInstallPath(id, v)
				_ = os.MkdirAll(installPath, 0755)
			},
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupFunc(tt.packageID, tt.packageVersion)

			// Test
			ver, _ := version.Parse(tt.packageVersion)
			exists := provider.packageExists(tt.packageID, ver)

			// Verify
			if exists != tt.expectedExists {
				t.Errorf("packageExists() = %v, want %v", exists, tt.expectedExists)
			}
		})
	}
}

func TestLocalDependencyProvider_GetDependencies_NotCached(t *testing.T) {
	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	ctx := context.Background()

	// Try to get dependencies for non-cached package
	groups, resolvedVer, err := provider.GetDependencies(ctx, "NonExistent", "1.0.0")

	if err != nil {
		t.Fatalf("GetDependencies() error = %v, want nil", err)
	}

	if groups != nil {
		t.Errorf("GetDependencies() = %v, want nil (not cached)", groups)
	}

	if resolvedVer != "" {
		t.Errorf("GetDependencies() resolved version = %s, want empty (not cached)", resolvedVer)
	}
}

func TestLocalDependencyProvider_GetDependencies_Cached(t *testing.T) {
	// This test requires a real cached package with .nuspec file
	// We'll create a mock .nuspec in the cache structure

	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	packageID := "TestPackage"
	packageVersion := "1.0.0"

	// Parse version
	ver, _ := version.Parse(packageVersion)

	// Create package structure
	installPath := provider.resolver.GetInstallPath(packageID, ver)
	_ = os.MkdirAll(installPath, 0755)

	// Create completion marker
	metadataPath := provider.resolver.GetNupkgMetadataPath(packageID, ver)
	_ = os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)

	// Create .nuspec file
	nuspecPath := provider.resolver.GetManifestFilePath(packageID, ver)
	nuspecContent := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>TestPackage</id>
    <version>1.0.0</version>
    <dependencies>
      <group targetFramework="net6.0">
        <dependency id="Newtonsoft.Json" version="13.0.1" />
        <dependency id="Serilog" version="[2.10.0,3.0.0)" />
      </group>
    </dependencies>
  </metadata>
</package>`

	_ = os.WriteFile(nuspecPath, []byte(nuspecContent), 0644)

	ctx := context.Background()

	// Test: Get ALL dependency groups (no framework filtering)
	groups, _, err := provider.GetDependencies(ctx, packageID, packageVersion)

	if err != nil {
		t.Fatalf("GetDependencies() error = %v, want nil", err)
	}

	if groups == nil {
		t.Fatal("GetDependencies() = nil, want non-nil")
	}

	// Verify we got 1 group (net6.0)
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}

	// Check group framework (should preserve original short format from nuspec)
	if groups[0].TargetFramework != "net6.0" {
		t.Errorf("groups[0].TargetFramework = %s, want net6.0", groups[0].TargetFramework)
	}

	// Check dependencies in group
	if len(groups[0].Dependencies) != 2 {
		t.Fatalf("len(groups[0].Dependencies) = %d, want 2", len(groups[0].Dependencies))
	}

	// Check first dependency
	if groups[0].Dependencies[0].ID != "Newtonsoft.Json" {
		t.Errorf("groups[0].Dependencies[0].ID = %s, want Newtonsoft.Json", groups[0].Dependencies[0].ID)
	}

	// Check second dependency
	if groups[0].Dependencies[1].ID != "Serilog" {
		t.Errorf("groups[0].Dependencies[1].ID = %s, want Serilog", groups[0].Dependencies[1].ID)
	}
}

func TestLocalDependencyProvider_GetDependencies_NoDependencies(t *testing.T) {
	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	packageID := "NoDepsPackage"
	packageVersion := "1.0.0"

	// Parse version
	ver, _ := version.Parse(packageVersion)

	// Create package structure
	installPath := provider.resolver.GetInstallPath(packageID, ver)
	_ = os.MkdirAll(installPath, 0755)

	// Create completion marker
	metadataPath := provider.resolver.GetNupkgMetadataPath(packageID, ver)
	_ = os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)

	// Create .nuspec file with no dependencies
	nuspecPath := provider.resolver.GetManifestFilePath(packageID, ver)
	nuspecContent := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>NoDepsPackage</id>
    <version>1.0.0</version>
  </metadata>
</package>`

	_ = os.WriteFile(nuspecPath, []byte(nuspecContent), 0644)

	ctx := context.Background()

	// Test: Get all dependency groups (should be empty)
	groups, _, err := provider.GetDependencies(ctx, packageID, packageVersion)

	if err != nil {
		t.Fatalf("GetDependencies() error = %v, want nil", err)
	}

	if groups == nil {
		t.Fatal("GetDependencies() = nil, want empty slice")
	}

	if len(groups) != 0 {
		t.Errorf("len(groups) = %d, want 0", len(groups))
	}
}

func TestLocalDependencyProvider_GetDependencies_FrameworkFallback(t *testing.T) {
	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	packageID := "MultiFrameworkPackage"
	packageVersion := "1.0.0"

	// Parse version
	ver, _ := version.Parse(packageVersion)

	// Create package structure
	installPath := provider.resolver.GetInstallPath(packageID, ver)
	_ = os.MkdirAll(installPath, 0755)

	// Create completion marker
	metadataPath := provider.resolver.GetNupkgMetadataPath(packageID, ver)
	_ = os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)

	// Create .nuspec with multiple framework groups
	// Provider should return ALL groups (walker will select best match)
	nuspecPath := provider.resolver.GetManifestFilePath(packageID, ver)
	nuspecContent := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>MultiFrameworkPackage</id>
    <version>1.0.0</version>
    <dependencies>
      <group targetFramework="net45">
        <dependency id="OldDep" version="1.0.0" />
      </group>
      <group targetFramework="net6.0">
        <dependency id="NewDep" version="2.0.0" />
      </group>
    </dependencies>
  </metadata>
</package>`

	_ = os.WriteFile(nuspecPath, []byte(nuspecContent), 0644)

	ctx := context.Background()

	// Test: Get ALL dependency groups (no filtering by framework)
	groups, _, err := provider.GetDependencies(ctx, packageID, packageVersion)

	if err != nil {
		t.Fatalf("GetDependencies() error = %v, want nil", err)
	}

	if groups == nil {
		t.Fatal("GetDependencies() = nil, want non-nil")
	}

	// Verify we got 2 groups (net45 and net6.0)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}

	// Check first group (net45 - should preserve original short format from nuspec)
	if groups[0].TargetFramework != "net45" {
		t.Errorf("groups[0].TargetFramework = %s, want net45", groups[0].TargetFramework)
	}
	if len(groups[0].Dependencies) != 1 {
		t.Fatalf("len(groups[0].Dependencies) = %d, want 1", len(groups[0].Dependencies))
	}
	if groups[0].Dependencies[0].ID != "OldDep" {
		t.Errorf("groups[0].Dependencies[0].ID = %s, want OldDep", groups[0].Dependencies[0].ID)
	}

	// Check second group (net6.0 - should preserve original short format from nuspec)
	if groups[1].TargetFramework != "net6.0" {
		t.Errorf("groups[1].TargetFramework = %s, want net6.0", groups[1].TargetFramework)
	}
	if len(groups[1].Dependencies) != 1 {
		t.Fatalf("len(groups[1].Dependencies) = %d, want 1", len(groups[1].Dependencies))
	}
	if groups[1].Dependencies[0].ID != "NewDep" {
		t.Errorf("groups[1].Dependencies[0].ID = %s, want NewDep", groups[1].Dependencies[0].ID)
	}
}

func TestLocalDependencyProvider_GetDependencies_AnyFramework(t *testing.T) {
	tempDir := t.TempDir()
	provider := NewLocalDependencyProvider(tempDir)

	packageID := "AnyFrameworkPackage"
	packageVersion := "1.0.0"

	// Parse version
	ver, _ := version.Parse(packageVersion)

	// Create package structure
	installPath := provider.resolver.GetInstallPath(packageID, ver)
	_ = os.MkdirAll(installPath, 0755)

	// Create completion marker
	metadataPath := provider.resolver.GetNupkgMetadataPath(packageID, ver)
	_ = os.WriteFile(metadataPath, []byte(`{"version":2}`), 0644)

	// Create .nuspec with "any" framework group (empty targetFramework)
	nuspecPath := provider.resolver.GetManifestFilePath(packageID, ver)
	nuspecContent := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>AnyFrameworkPackage</id>
    <version>1.0.0</version>
    <dependencies>
      <group>
        <dependency id="UniversalDep" version="1.0.0" />
      </group>
    </dependencies>
  </metadata>
</package>`

	_ = os.WriteFile(nuspecPath, []byte(nuspecContent), 0644)

	ctx := context.Background()

	// Test: Get all dependency groups (should return "any" framework group)
	groups, _, err := provider.GetDependencies(ctx, packageID, packageVersion)

	if err != nil {
		t.Fatalf("GetDependencies() error = %v, want nil", err)
	}

	if groups == nil {
		t.Fatal("GetDependencies() = nil, want non-nil")
	}

	// Verify we got 1 group (any framework)
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}

	// Check that target framework is "any" (AnyFramework.String() returns "any")
	if groups[0].TargetFramework != "any" {
		t.Errorf("groups[0].TargetFramework = %s, want any", groups[0].TargetFramework)
	}

	// Check dependencies in group
	if len(groups[0].Dependencies) != 1 {
		t.Fatalf("len(groups[0].Dependencies) = %d, want 1", len(groups[0].Dependencies))
	}

	if groups[0].Dependencies[0].ID != "UniversalDep" {
		t.Errorf("groups[0].Dependencies[0].ID = %s, want UniversalDep", groups[0].Dependencies[0].ID)
	}
}
