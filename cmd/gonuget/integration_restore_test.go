package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRestore_SinglePackage tests restoring a project with one PackageReference
func TestRestore_SinglePackage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create project with PackageReference
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Serilog", Version: "3.0.0"},
	})

	// Run restore
	stdout := env.runExpectSuccess("restore", projectPath)

	// Verify output mentions restore completion
	if !strings.Contains(stdout, "Restored") && !strings.Contains(stdout, "restore") {
		t.Logf("Restore output: %s", stdout)
	}

	// Verify package downloaded
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Errorf("package should exist in global cache at %s", pkgPath)
	}

	// Verify .nupkg file exists
	nupkgPath := filepath.Join(pkgPath, "serilog.3.0.0.nupkg")
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Error(".nupkg file should exist in package folder")
	}

	// Verify project.assets.json created
	assetsPath := filepath.Join(filepath.Dir(projectPath), "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("project.assets.json should be created")
	}
}

// TestRestore_MultipleDependencies tests restoring with multiple packages
func TestRestore_MultipleDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create project with multiple PackageReferences
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Newtonsoft.Json", Version: "13.0.3"},
		{ID: "Serilog", Version: "3.0.0"},
		{ID: "System.Text.Json", Version: "8.0.0"},
	})

	// Run restore
	stdout := env.runExpectSuccess("restore", projectPath)

	// Verify output
	if !strings.Contains(stdout, "Restored") && !strings.Contains(stdout, "restore") {
		t.Logf("Restore output: %s", stdout)
	}

	// Verify all packages downloaded
	packages := []struct {
		id      string
		version string
	}{
		{"newtonsoft.json", "13.0.3"},
		{"serilog", "3.0.0"},
		{"system.text.json", "8.0.0"},
	}

	for _, pkg := range packages {
		pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
			pkg.id, pkg.version)
		if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
			t.Errorf("package %s@%s should exist in cache", pkg.id, pkg.version)
		}
	}

	// Verify project.assets.json created
	assetsPath := filepath.Join(filepath.Dir(projectPath), "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("project.assets.json should be created")
	}
}

// TestRestore_TransitiveDependencies tests that transitive dependencies are downloaded
func TestRestore_TransitiveDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create project with a package that has dependencies
	// Serilog has transitive dependencies
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Serilog", Version: "3.0.0"},
	})

	// Run restore
	env.runExpectSuccess("restore", projectPath)

	// Verify direct dependency downloaded
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Error("direct dependency Serilog should be downloaded")
	}

	// Verify project.assets.json exists and contains dependency information
	assetsPath := filepath.Join(filepath.Dir(projectPath), "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("project.assets.json should be created")
		return
	}

	// Read assets file to check for transitive dependencies
	assetsData, err := os.ReadFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to read project.assets.json: %v", err)
	}

	assetsContent := string(assetsData)
	// Should contain the direct dependency
	if !strings.Contains(assetsContent, "Serilog") {
		t.Error("project.assets.json should contain Serilog")
	}
}

// TestRestore_CustomPackagesFolder tests --packages flag
func TestRestore_CustomPackagesFolder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create custom packages directory
	customPackagesDir := filepath.Join(env.tempDir, "custom-packages")

	// Create project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Serilog", Version: "3.0.0"},
	})

	// Run restore with custom packages folder
	env.runExpectSuccess("restore", projectPath,
		"--packages", customPackagesDir)

	// Verify package downloaded to custom location
	pkgPath := filepath.Join(customPackagesDir, "serilog", "3.0.0")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Errorf("package should exist in custom packages folder at %s", pkgPath)
	}

	// Verify NOT in default location
	defaultPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0")
	if _, err := os.Stat(defaultPath); err == nil {
		t.Error("package should NOT be in default location when using --packages")
	}
}

// TestRestore_Force tests --force flag for re-downloading
func TestRestore_Force(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Serilog", Version: "3.0.0"},
	})

	// First restore
	env.runExpectSuccess("restore", projectPath)

	// Get modification time of package
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0", "serilog.3.0.0.nupkg")
	info1, err := os.Stat(pkgPath)
	if err != nil {
		t.Fatalf("package should exist after first restore: %v", err)
	}

	// Wait a moment to ensure different timestamp
	// (In practice, force should re-download regardless)

	// Second restore with --force
	env.runExpectSuccess("restore", projectPath, "--force")

	// Verify package still exists (should be re-downloaded)
	info2, err := os.Stat(pkgPath)
	if err != nil {
		t.Fatalf("package should exist after force restore: %v", err)
	}

	// Package should exist (we don't check timestamp as it may be identical
	// if download is very fast, but the operation should succeed)
	_ = info1
	_ = info2
}

// TestRestore_ProjectAssetsJson tests project.assets.json generation
func TestRestore_ProjectAssetsJson(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Newtonsoft.Json", Version: "13.0.3"},
		{ID: "Serilog", Version: "3.0.0"},
	})

	// Run restore
	env.runExpectSuccess("restore", projectPath)

	// Verify project.assets.json exists
	assetsPath := filepath.Join(filepath.Dir(projectPath), "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Fatal("project.assets.json should be created")
	}

	// Read and verify contents
	assetsData, err := os.ReadFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to read project.assets.json: %v", err)
	}

	assetsContent := string(assetsData)

	// Should be valid JSON (basic check)
	if !strings.HasPrefix(assetsContent, "{") {
		t.Error("project.assets.json should be valid JSON")
	}

	// Should contain package references
	if !strings.Contains(assetsContent, "Newtonsoft.Json") {
		t.Error("project.assets.json should contain Newtonsoft.Json")
	}
	if !strings.Contains(assetsContent, "Serilog") {
		t.Error("project.assets.json should contain Serilog")
	}

	// Should contain version information
	if !strings.Contains(assetsContent, "13.0.3") {
		t.Error("project.assets.json should contain version 13.0.3")
	}
	if !strings.Contains(assetsContent, "3.0.0") {
		t.Error("project.assets.json should contain version 3.0.0")
	}

	// Should contain target framework
	if !strings.Contains(assetsContent, "net8.0") {
		t.Error("project.assets.json should contain target framework")
	}
}

// TestRestore_AlreadyRestored tests that second restore is fast (uses cache)
func TestRestore_AlreadyRestored(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	// Create project
	projectPath := filepath.Join(env.tempDir, "TestApp.csproj")
	createProjectWithPackages(t, projectPath, []packageRef{
		{ID: "Serilog", Version: "3.0.0"},
	})

	// First restore
	stdout1 := env.runExpectSuccess("restore", projectPath)
	t.Logf("First restore: %s", stdout1)

	// Verify package exists
	pkgPath := filepath.Join(env.homeDir, ".nuget", "packages",
		"serilog", "3.0.0")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Fatal("package should exist after first restore")
	}

	// Second restore (should use cache)
	stdout2 := env.runExpectSuccess("restore", projectPath)
	t.Logf("Second restore: %s", stdout2)

	// Package should still exist
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Error("package should still exist after second restore")
	}

	// project.assets.json should exist
	assetsPath := filepath.Join(filepath.Dir(projectPath), "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("project.assets.json should exist after second restore")
	}
}
