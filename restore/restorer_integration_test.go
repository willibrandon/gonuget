package restore

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// TestRestore_NewtonsoftJson verifies package with NO transitive dependencies.
func TestRestore_NewtonsoftJson(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	// Create temporary directory for packages
	tmpDir := t.TempDir()
	packagesFolder := filepath.Join(tmpDir, "packages")

	// Create test project with Newtonsoft.Json reference
	projDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	projectPath := filepath.Join(projDir, "test.csproj")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projectPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create restorer
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// Execute restore
	packageRefs := proj.GetPackageReferences()
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify: Only direct package, no transitive dependencies
	if len(result.DirectPackages) != 1 {
		t.Errorf("Expected 1 direct package, got %d", len(result.DirectPackages))
	}

	if len(result.TransitivePackages) != 0 {
		t.Errorf("Expected 0 transitive packages, got %d: %v", len(result.TransitivePackages), result.TransitivePackages)
	}

	if result.DirectPackages[0].ID != "Newtonsoft.Json" {
		t.Errorf("Expected Newtonsoft.Json, got %s", result.DirectPackages[0].ID)
	}

	if result.DirectPackages[0].Version != "13.0.3" {
		t.Errorf("Expected version 13.0.3, got %s", result.DirectPackages[0].Version)
	}
}

// TestRestore_SerilogSinksFile verifies package with single transitive dependency.
func TestRestore_SerilogSinksFile(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	// Create temporary directory for packages
	tmpDir := t.TempDir()
	packagesFolder := filepath.Join(tmpDir, "packages")

	// Create test project with Serilog.Sinks.File reference
	projDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Serilog.Sinks.File" Version="5.0.0" />
  </ItemGroup>
</Project>`

	projectPath := filepath.Join(projDir, "test.csproj")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projectPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create restorer
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// Execute restore
	packageRefs := proj.GetPackageReferences()
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify: 1 direct package (Serilog.Sinks.File)
	if len(result.DirectPackages) != 1 {
		t.Errorf("Expected 1 direct package, got %d", len(result.DirectPackages))
	}

	if result.DirectPackages[0].ID != "Serilog.Sinks.File" {
		t.Errorf("Expected Serilog.Sinks.File, got %s", result.DirectPackages[0].ID)
	}

	// Verify: At least 1 transitive package (Serilog)
	if len(result.TransitivePackages) < 1 {
		t.Errorf("Expected at least 1 transitive package, got %d", len(result.TransitivePackages))
	}

	// Find Serilog in transitive packages
	foundSerilog := false
	for _, pkg := range result.TransitivePackages {
		if pkg.ID == "Serilog" {
			foundSerilog = true
			if pkg.IsDirect {
				t.Error("Serilog should not be marked as direct dependency")
			}
			break
		}
	}

	if !foundSerilog {
		t.Error("Expected to find Serilog in transitive packages")
	}
}

// TestRestore_MicrosoftExtensionsLogging verifies package with multiple transitive dependencies.
func TestRestore_MicrosoftExtensionsLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	// Create temporary directory for packages
	tmpDir := t.TempDir()
	packagesFolder := filepath.Join(tmpDir, "packages")

	// Create test project with Microsoft.Extensions.Logging reference
	projDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.Extensions.Logging" Version="8.0.0" />
  </ItemGroup>
</Project>`

	projectPath := filepath.Join(projDir, "test.csproj")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projectPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create restorer
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}
	console := &testConsole{}
	restorer := NewRestorer(opts, console)

	// Execute restore
	packageRefs := proj.GetPackageReferences()
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify: 1 direct package
	if len(result.DirectPackages) != 1 {
		t.Errorf("Expected 1 direct package, got %d", len(result.DirectPackages))
	}

	if result.DirectPackages[0].ID != "Microsoft.Extensions.Logging" {
		t.Errorf("Expected Microsoft.Extensions.Logging, got %s", result.DirectPackages[0].ID)
	}

	// Verify: Multiple transitive packages (Microsoft.Extensions.Logging has several dependencies)
	if len(result.TransitivePackages) < 2 {
		t.Errorf("Expected at least 2 transitive packages, got %d", len(result.TransitivePackages))
	}

	// Verify no duplicates in all packages
	allPackages := result.AllPackages()
	seen := make(map[string]bool)
	for _, pkg := range allPackages {
		key := pkg.ID + "/" + pkg.Version
		if seen[key] {
			t.Errorf("Duplicate package found: %s", key)
		}
		seen[key] = true
	}
}

type testConsole struct{}

func (c *testConsole) Printf(format string, args ...any)  {}
func (c *testConsole) Error(format string, args ...any)   {}
func (c *testConsole) Warning(format string, args ...any) {}
func (c *testConsole) Output() io.Writer                  { return io.Discard }
