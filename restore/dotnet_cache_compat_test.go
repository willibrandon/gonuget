package restore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// TestDotnetCacheCompatibility verifies that gonuget correctly handles cache files
// created by dotnet and properly categorizes packages as direct vs transitive
// using the project file's PackageReferences.
func TestDotnetCacheCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dotnet integration test in short mode")
	}

	// Use the complex test scenario project (absolute path to avoid CWD issues)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectPath := filepath.Join(filepath.Dir(cwd), "tests", "test-scenarios", "complex", "test.csproj")
	cachePath := filepath.Join(filepath.Dir(cwd), "tests", "test-scenarios", "complex", "obj", "project.nuget.cache")

	// Load project for hash calculation
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		t.Fatalf("Failed to load project: %v", err)
	}

	// Step 1: Run dotnet restore to create cache WITHOUT directPackageFiles
	dotnetCmd := exec.Command("dotnet", "restore", projectPath, "-v:quiet")
	if err := dotnetCmd.Run(); err != nil {
		t.Fatalf("dotnet restore failed: %v", err)
	}

	// Verify dotnet created cache successfully
	cache, err := LoadCacheFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to load dotnet-created cache: %v", err)
	}

	if !cache.IsValid() {
		t.Fatalf("Expected dotnet cache to be valid")
	}

	// Step 2: Run gonuget restore (should hit cache)
	opts := &Options{
		Verbosity: "normal",
		Sources:   []string{"https://api.nuget.org/v3/index.json"},
	}

	console := &mockConsole{}
	restorer := NewRestorer(opts, console)

	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("gonuget restore failed: %v", err)
	}

	// Verify it was a cache hit
	if !result.CacheHit {
		t.Errorf("Expected cache hit, got fresh restore")
	}

	// The complex project has 7 direct dependencies
	expectedDirectPackages := map[string]bool{
		"microsoft.extensions.dependencyinjection": true,
		"microsoft.extensions.logging":             true,
		"microsoft.extensions.configuration":       true,
		"newtonsoft.json":                          true,
		"serilog":                                  true,
		"serilog.sinks.console":                    true,
		"serilog.sinks.file":                       true,
	}

	// BUG: When reading dotnet cache (no directPackageFiles), gonuget marks all as transitive
	if len(result.DirectPackages) != 7 {
		t.Errorf("Expected 7 direct packages, got %d", len(result.DirectPackages))
		t.Logf("Direct packages: %v", result.DirectPackages)
	}

	if len(result.TransitivePackages) != 5 {
		t.Errorf("Expected 5 transitive packages, got %d", len(result.TransitivePackages))
		t.Logf("Transitive packages: %v", result.TransitivePackages)
	}

	// Verify all direct packages are actually direct
	for _, pkg := range result.DirectPackages {
		normalizedID := strings.ToLower(pkg.ID)
		if !expectedDirectPackages[normalizedID] {
			t.Errorf("Package %q marked as direct but not in project file", pkg.ID)
		}
	}

	// Verify all transitive packages are NOT direct
	for _, pkg := range result.TransitivePackages {
		normalizedID := strings.ToLower(pkg.ID)
		if expectedDirectPackages[normalizedID] {
			t.Errorf("Package %q marked as transitive but IS in project file", pkg.ID)
		}
	}
}
