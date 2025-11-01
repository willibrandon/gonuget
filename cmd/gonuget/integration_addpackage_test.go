package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAddPackage_RealPackage_Latest tests adding a package without specifying version (latest)
func TestAddPackage_RealPackage_Latest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create test project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createBasicProject(t, projectPath)

	// Add package (latest version)
	stdout := env.runExpectSuccess("package", "add", "Serilog",
		"--project", projectPath)

	// Verify output mentions PackageReference
	if !strings.Contains(stdout, "PackageReference") && !strings.Contains(stdout, "Added") {
		t.Errorf("should report PackageReference added, got: %s", stdout)
	}

	// Verify project file modified
	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("failed to read project file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Serilog") {
		t.Error("project file should contain PackageReference for Serilog")
	}
	if !strings.Contains(content, `Include="Serilog"`) {
		t.Error("project file should have Include attribute")
	}

	// Verify package downloaded to cache (lowercase package ID in path)
	cacheDir := filepath.Join(env.homeDir, ".nuget", "packages", "serilog")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("package should be downloaded to global cache")
	}
}

// TestAddPackage_RealPackage_SpecificVersion tests adding a package with specific version
func TestAddPackage_RealPackage_SpecificVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create test project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createBasicProject(t, projectPath)

	// Add package with specific version
	stdout := env.runExpectSuccess("package", "add", "Newtonsoft.Json",
		"--version", "13.0.3",
		"--project", projectPath)

	// Verify output
	if !strings.Contains(stdout, "PackageReference") && !strings.Contains(stdout, "Added") {
		t.Errorf("should report success, got: %s", stdout)
	}

	// Verify project file has correct version
	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("failed to read project file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Newtonsoft.Json") {
		t.Error("project file should contain Newtonsoft.Json")
	}
	if !strings.Contains(content, "13.0.3") {
		t.Error("project file should contain version 13.0.3")
	}

	// Verify exact version downloaded
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"newtonsoft.json", "13.0.3")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Errorf("package version 13.0.3 should exist in cache at %s", pkgPath)
	}
}

// TestAddPackage_MultiplePackages tests adding multiple packages sequentially
func TestAddPackage_MultiplePackages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create test project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createBasicProject(t, projectPath)

	// Add multiple packages
	packages := []struct {
		id      string
		version string
	}{
		{"Serilog", "3.0.0"},
		{"Newtonsoft.Json", "13.0.3"},
	}

	for _, pkg := range packages {
		env.runExpectSuccess("package", "add", pkg.id,
			"--version", pkg.version,
			"--project", projectPath)
	}

	// Verify all packages in project file
	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("failed to read project file: %v", err)
	}
	content := string(data)

	for _, pkg := range packages {
		if !strings.Contains(content, pkg.id) {
			t.Errorf("project should contain %s", pkg.id)
		}
		if !strings.Contains(content, pkg.version) {
			t.Errorf("project should contain version %s", pkg.version)
		}
	}

	// Verify all packages downloaded
	for _, pkg := range packages {
		pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
			strings.ToLower(pkg.id), pkg.version)
		if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
			t.Errorf("package %s@%s should exist in cache", pkg.id, pkg.version)
		}
	}
}

// TestAddPackage_WithRestore tests that package is actually downloaded to cache
func TestAddPackage_WithRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create test project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createBasicProject(t, projectPath)

	// Add package (restore happens by default)
	env.runExpectSuccess("package", "add", "Serilog",
		"--version", "3.0.0",
		"--project", projectPath)

	// Verify package downloaded
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Error("package should be downloaded with default restore behavior")
	}

	// Verify package contents exist
	nupkgPath := filepath.Join(pkgPath, "serilog.3.0.0.nupkg")
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Error("package .nupkg file should exist")
	}

	// Verify project.assets.json created
	projectDir := filepath.Dir(projectPath)
	assetsPath := filepath.Join(projectDir, "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("project.assets.json should be created")
	}
}

// TestAddPackage_NoRestore_Flag tests --no-restore flag
func TestAddPackage_NoRestore_Flag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create test project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createBasicProject(t, projectPath)

	// Add package with --no-restore
	stdout := env.runExpectSuccess("package", "add", "Serilog",
		"--version", "3.0.0",
		"--project", projectPath,
		"--no-restore")

	// Verify success message
	if !strings.Contains(stdout, "PackageReference") && !strings.Contains(stdout, "Added") {
		t.Errorf("should report success, got: %s", stdout)
	}

	// Verify project file modified
	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("failed to read project file: %v", err)
	}
	if !strings.Contains(string(data), "Serilog") {
		t.Error("project file should contain PackageReference")
	}

	// Verify package NOT downloaded (since --no-restore)
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0")
	if _, err := os.Stat(pkgPath); err == nil {
		t.Error("package should NOT be downloaded when using --no-restore")
	}

	// Verify project.assets.json NOT created
	projectDir := filepath.Dir(projectPath)
	assetsPath := filepath.Join(projectDir, "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); err == nil {
		t.Error("project.assets.json should NOT be created with --no-restore")
	}
}
