package restore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// mockConsole implements Console interface for testing
type mockConsole struct {
	messages []string
	errors   []string
	warnings []string
}

func (m *mockConsole) Printf(format string, args ...any) {
	m.messages = append(m.messages, fmt.Sprintf(format, args...))
}

func (m *mockConsole) Error(format string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func (m *mockConsole) Warning(format string, args ...any) {
	m.warnings = append(m.warnings, fmt.Sprintf(format, args...))
}

func (m *mockConsole) Output() io.Writer {
	return io.Discard
}

func TestRun_NoProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	console := &mockConsole{}
	opts := &Options{}

	// Change to empty directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	err = Run(context.Background(), nil, opts, console)
	if err == nil {
		t.Error("expected error for missing project file")
	}

	if !strings.Contains(err.Error(), "no project file found") {
		t.Errorf("expected 'no project file found' error, got: %v", err)
	}
}

func TestRun_WithProjectPath(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Create simple project with no packages
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	opts := &Options{}

	err := Run(context.Background(), []string{projPath}, opts, console)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check console output
	found := false
	for _, msg := range console.messages {
		if strings.Contains(msg, "Nothing to restore") {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'Nothing to restore' message")
	}
}

func TestRun_InvalidProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Create invalid XML
	content := `<Project invalid xml`
	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	opts := &Options{}

	err := Run(context.Background(), []string{projPath}, opts, console)
	if err == nil {
		t.Error("expected error for invalid project file")
	}

	if !strings.Contains(err.Error(), "failed to load project") {
		t.Errorf("expected 'failed to load project' error, got: %v", err)
	}
}

func TestNewRestorer(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		console Console
	}{
		{
			name: "create restorer with no sources",
			opts: &Options{
				Sources: []string{},
			},
			console: &mockConsole{},
		},
		{
			name: "create restorer with single source",
			opts: &Options{
				Sources: []string{"https://api.nuget.org/v3/index.json"},
			},
			console: &mockConsole{},
		},
		{
			name: "create restorer with multiple sources",
			opts: &Options{
				Sources: []string{
					"https://api.nuget.org/v3/index.json",
					"https://pkgs.dev.azure.com/test/_packaging/test/nuget/v3/index.json",
				},
			},
			console: &mockConsole{},
		},
		{
			name: "create restorer with invalid source",
			opts: &Options{
				Sources: []string{"not-a-valid-url"},
			},
			console: &mockConsole{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorer := NewRestorer(tt.opts, tt.console)
			if restorer == nil {
				t.Fatal("expected restorer but got nil")
			}

			if restorer.opts != tt.opts {
				t.Error("restorer options not set correctly")
			}

			if restorer.console != tt.console {
				t.Error("restorer console not set correctly")
			}

			if restorer.client == nil {
				t.Error("restorer client not initialized")
			}
		})
	}
}

func TestRestorer_Restore_NoPackages(t *testing.T) {
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: filepath.Join(tmpDir, "packages"),
	}

	restorer := NewRestorer(opts, console)
	packageRefs := []project.PackageReference{}

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(result.DirectPackages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(result.DirectPackages))
	}
}

func TestRestorer_Restore_WithForce(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Force:          true,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(result.DirectPackages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.DirectPackages))
	}

	// Verify package was downloaded
	// Use lowercase package ID to match NuGet.Client's VersionFolderPathResolver behavior
	packagePath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.1")
	nupkgPath := filepath.Join(packagePath, "newtonsoft.json.13.0.1.nupkg")

	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Error("package file was not downloaded")
	}
}

func TestRestorer_Restore_PackageAlreadyCached(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")

	// Pre-create package directory to simulate cached package
	// Use lowercase package ID to match NuGet.Client's VersionFolderPathResolver behavior
	packagePath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.1")
	if err := os.MkdirAll(packagePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .nupkg.sha512 marker file to indicate package is properly cached
	sha512Path := filepath.Join(packagePath, "newtonsoft.json.13.0.1.nupkg.sha512")
	if err := os.WriteFile(sha512Path, []byte("dummy-sha512-hash"), 0644); err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Force:          false,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	// First restore - should create cache file
	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(result.DirectPackages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.DirectPackages))
	}

	// Verify cache file was written
	cachePath := GetCacheFilePath(projPath)
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache file should exist: %v", err)
	}

	// Second restore - should hit cache (no-op)
	console2 := &mockConsole{}
	restorer2 := NewRestorer(opts, console2)
	result2, err := restorer2.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Errorf("unexpected error on second restore: %v", err)
	}

	if !result2.CacheHit {
		t.Error("expected cache hit on second restore")
	}

	if len(result2.DirectPackages) != 1 {
		t.Errorf("expected 1 package on cache hit, got %d", len(result2.DirectPackages))
	}
}

func TestFindProjectFile(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) ([]string, string)
		expectError bool
		errorMsg    string
	}{
		{
			name: "find with explicit path",
			setup: func(t *testing.T) ([]string, string) {
				tmpDir := t.TempDir()
				projPath := filepath.Join(tmpDir, "test.csproj")
				if err := os.WriteFile(projPath, []byte("<Project/>"), 0644); err != nil {
					t.Fatalf("failed to write project file: %v", err)
				}
				// Get absolute path WITHOUT resolving symlinks (matches findProjectFile behavior)
				absPath, err := filepath.Abs(projPath)
				if err != nil {
					t.Fatalf("failed to get absolute path: %v", err)
				}
				return []string{projPath}, absPath
			},
			expectError: false,
		},
		{
			name: "no project file in directory",
			setup: func(t *testing.T) ([]string, string) {
				tmpDir := t.TempDir()
				return nil, tmpDir
			},
			expectError: true,
			errorMsg:    "no project file found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, expected := tt.setup(t)

			path, err := findProjectFile(args)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if args != nil && path != expected {
					t.Errorf("expected path %s, got %s", expected, path)
				}
			}
		})
	}
}

func TestRun_WithPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	packagesFolder := filepath.Join(tmpDir, "packages")
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
		Verbosity:      "detailed", // Required for "Determining" and "Restored" messages
	}

	err := Run(context.Background(), []string{projPath}, opts, console)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify project.assets.json was created
	assetsPath := filepath.Join(tmpDir, "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("project.assets.json was not created")
	}

	// Verify package was downloaded
	// Use lowercase package ID to match NuGet.Client's VersionFolderPathResolver behavior
	packagePath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.1")
	nupkgPath := filepath.Join(packagePath, "newtonsoft.json.13.0.1.nupkg")
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Error("package file was not downloaded")
	}

	// Check console output
	foundRestore := false
	foundRestored := false
	for _, msg := range console.messages {
		if strings.Contains(msg, "Determining") {
			foundRestore = true
		}
		if strings.Contains(msg, "Restored") {
			foundRestored = true
		}
	}

	if !foundRestore {
		t.Error("expected 'Determining' message")
	}
	if !foundRestored {
		t.Error("expected 'Restored' message")
	}
}

func TestRun_FailedLockFileSave(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create obj directory as a file to cause lock file save to fail
	objPath := filepath.Join(tmpDir, "obj")
	if err := os.WriteFile(objPath, []byte("file"), 0444); err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	packagesFolder := filepath.Join(tmpDir, "packages")
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	err := Run(context.Background(), []string{projPath}, opts, console)
	if err == nil {
		t.Error("expected error for failed lock file save")
	}

	if !strings.Contains(err.Error(), "failed to save project.assets.json") {
		t.Errorf("expected 'failed to save project.assets.json' error, got: %v", err)
	}
}

func TestRestorer_Restore_DownloadError(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="NonExistentPackage999999" Version="1.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	_, err = restorer.Restore(context.Background(), proj, packageRefs)
	if err == nil {
		t.Error("expected error for nonexistent package")
	}

	// Should get package not found error (NU1101 or NU1102)
	errMsg := err.Error()
	if !strings.Contains(errMsg, "package version not found") && !strings.Contains(errMsg, "unresolved packages") {
		t.Errorf("expected package not found error for NonExistentPackage999999, got: %v", err)
	}
}

func TestRestorer_Restore_DefaultPackagesFolder(t *testing.T) {
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	console := &mockConsole{}
	opts := &Options{
		// No PackagesFolder specified - should use default
	}

	restorer := NewRestorer(opts, console)
	packageRefs := []project.PackageReference{}

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(result.DirectPackages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(result.DirectPackages))
	}
}

func TestFindProjectFile_GetWorkingDirectory(t *testing.T) {
	// Test case where explicit path is provided
	// Create a real file since findProjectFile resolves symlinks (requires file to exist)
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")
	if err := os.WriteFile(projPath, []byte("<Project/>"), 0644); err != nil {
		t.Fatalf("failed to write project file: %v", err)
	}

	args := []string{projPath}
	path, err := findProjectFile(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// The returned path should be absolute WITHOUT resolving symlinks (matches dotnet behavior)
	absPath, _ := filepath.Abs(projPath)
	if path != absPath {
		t.Errorf("expected %s, got %s", absPath, path)
	}
}

// TestRestorer_Restore_PackageExtraction verifies that packages are fully extracted (not just downloaded).
func TestRestorer_Restore_PackageExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if len(result.DirectPackages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.DirectPackages))
	}

	// Verify package extraction (matching dotnet restore behavior)
	// Use lowercase package ID to match NuGet.Client's VersionFolderPathResolver behavior
	packagePath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.1")

	// 1. Verify .nupkg file exists
	nupkgPath := filepath.Join(packagePath, "newtonsoft.json.13.0.1.nupkg")
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Error("package .nupkg file was not downloaded")
	}

	// 2. Verify .nuspec file was extracted
	nuspecPath := filepath.Join(packagePath, "newtonsoft.json.nuspec")
	if _, err := os.Stat(nuspecPath); os.IsNotExist(err) {
		t.Error("package .nuspec file was not extracted")
	}

	// 3. Verify lib/ directory was extracted
	libDir := filepath.Join(packagePath, "lib")
	if _, err := os.Stat(libDir); os.IsNotExist(err) {
		t.Error("package lib/ directory was not extracted")
	}

	// 4. Verify at least one framework folder exists under lib/
	libEntries, err := os.ReadDir(libDir)
	if err != nil {
		t.Fatalf("failed to read lib directory: %v", err)
	}
	if len(libEntries) == 0 {
		t.Error("lib/ directory is empty, no framework folders extracted")
	}

	// 5. Verify DLL files were extracted (Newtonsoft.Json has DLLs)
	foundDLL := false
	for _, entry := range libEntries {
		if !entry.IsDir() {
			continue
		}
		frameworkDir := filepath.Join(libDir, entry.Name())
		files, err := os.ReadDir(frameworkDir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".dll") {
				foundDLL = true
				break
			}
		}
		if foundDLL {
			break
		}
	}
	if !foundDLL {
		t.Error("no .dll files found in lib/ subdirectories")
	}
}

// TestRestorer_Restore_ExtractionParity verifies extraction matches dotnet restore behavior.
func TestRestorer_Restore_ExtractionParity(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Use a package with known structure for verification
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Serilog" Version="4.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	_, err = restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify complete package structure (matching dotnet restore)
	packagePath := filepath.Join(packagesFolder, "serilog", "4.0.0")

	// Check expected files/directories that dotnet restore creates
	expectedPaths := []string{
		filepath.Join(packagePath, "serilog.4.0.0.nupkg"),
		filepath.Join(packagePath, "serilog.nuspec"),
		filepath.Join(packagePath, "lib"),
		filepath.Join(packagePath, "icon.png"), // Serilog has an icon
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected path does not exist: %s", path)
		}
	}
}

// TestRestorer_Restore_ExtractMultiplePackages verifies multiple packages are extracted correctly.
func TestRestorer_Restore_ExtractMultiplePackages(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Serilog" Version="4.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if len(result.DirectPackages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(result.DirectPackages))
	}

	// Verify both packages were extracted
	packages := []struct {
		id      string
		version string
	}{
		{"newtonsoft.json", "13.0.1"},
		{"serilog", "4.0.0"},
	}

	for _, pkg := range packages {
		packagePath := filepath.Join(packagesFolder, pkg.id, pkg.version)

		// Check .nuspec extraction for each package
		nuspecPath := filepath.Join(packagePath, pkg.id+".nuspec")
		if _, err := os.Stat(nuspecPath); os.IsNotExist(err) {
			t.Errorf("package %s .nuspec file was not extracted", pkg.id)
		}

		// Check lib/ directory for each package
		libDir := filepath.Join(packagePath, "lib")
		if _, err := os.Stat(libDir); os.IsNotExist(err) {
			t.Errorf("package %s lib/ directory was not extracted", pkg.id)
		}
	}
}

// TestRestorer_Restore_V3Protocol verifies that V3 feeds use the correct extraction method.
func TestRestorer_Restore_V3Protocol(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if len(result.DirectPackages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.DirectPackages))
	}

	// Verify V3 layout: lowercase package ID, .nupkg.metadata marker
	packagePath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.1")

	// V3 layout creates .nupkg.metadata file
	metadataPath := filepath.Join(packagePath, ".nupkg.metadata")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("V3 layout should create .nupkg.metadata file")
	}

	// V3 layout creates .nupkg.sha512 file
	shaPath := filepath.Join(packagePath, "newtonsoft.json.13.0.1.nupkg.sha512")
	if _, err := os.Stat(shaPath); os.IsNotExist(err) {
		t.Error("V3 layout should create .nupkg.sha512 file")
	}
}

// TestRestorer_Restore_V2Protocol verifies that V2 feeds use the correct extraction method.
func TestRestorer_Restore_V2Protocol(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		// Use a V2 feed (nuget.org V2 API)
		Sources: []string{"https://www.nuget.org/api/v2"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if len(result.DirectPackages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.DirectPackages))
	}

	// Note: Even though we specify V2 URL, nuget.org auto-detects as V3
	// So we expect V3 layout (lowercase ID, separate version folder, with .nupkg.metadata)
	packagePath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.1")

	// V3 layout creates .nupkg.metadata file
	metadataPath := filepath.Join(packagePath, ".nupkg.metadata")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("V3 layout should create .nupkg.metadata file")
	}

	// Verify package contents were extracted
	nuspecPath := filepath.Join(packagePath, "newtonsoft.json.nuspec")
	if _, err := os.Stat(nuspecPath); os.IsNotExist(err) {
		t.Error("Package layout should extract .nuspec file")
	}

	libDir := filepath.Join(packagePath, "lib")
	if _, err := os.Stat(libDir); os.IsNotExist(err) {
		t.Error("Package layout should extract lib/ directory")
	}
}

func TestHasPrereleaseVersionsOnly(t *testing.T) {
	tests := []struct {
		name         string
		versionRange string
		versions     []string
		expected     bool
		description  string
	}{
		{
			name:         "empty versions",
			versionRange: "[1.0.0,2.0.0)",
			versions:     []string{},
			expected:     false,
			description:  "No versions available",
		},
		{
			name:         "only stable versions satisfy range",
			versionRange: "[1.0.0,2.0.0)",
			versions:     []string{"1.0.0", "1.5.0"},
			expected:     false,
			description:  "Stable versions satisfy the range",
		},
		{
			name:         "only prerelease versions satisfy range",
			versionRange: "[1.0.0,2.0.0)",
			versions:     []string{"1.0.0-alpha", "1.0.0-beta", "1.5.0-rc1"},
			expected:     true,
			description:  "Only prerelease versions satisfy the range",
		},
		{
			name:         "mixed versions in range",
			versionRange: "[1.0.0,2.0.0)",
			versions:     []string{"1.0.0-alpha", "1.0.0", "1.5.0-beta"},
			expected:     false,
			description:  "Both stable and prerelease satisfy range",
		},
		{
			name:         "prerelease in range, stable outside range",
			versionRange: "[1.0.0,2.0.0)",
			versions:     []string{"1.0.0-alpha", "1.5.0-beta", "2.0.0", "3.0.0"},
			expected:     true,
			description:  "Only prerelease versions satisfy range, stable versions exist but outside range",
		},
		{
			name:         "no versions satisfy range",
			versionRange: "[5.0.0,6.0.0)",
			versions:     []string{"1.0.0", "2.0.0", "3.0.0"},
			expected:     false,
			description:  "No versions satisfy the range at all",
		},
		{
			name:         "exact prerelease version",
			versionRange: "[1.0.0-alpha]",
			versions:     []string{"1.0.0-alpha", "1.0.0"},
			expected:     true,
			description:  "Exact prerelease version satisfies, stable doesn't (not exact match)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPrereleaseVersionsOnly(tt.versionRange, tt.versions)
			if result != tt.expected {
				t.Errorf("hasPrereleaseVersionsOnly(%q, %v) = %v, want %v (%s)",
					tt.versionRange, tt.versions, result, tt.expected, tt.description)
			}
		})
	}
}

func TestIsPrereleaseAllowed(t *testing.T) {
	tests := []struct {
		name         string
		versionRange string
		expected     bool
		description  string
	}{
		{
			name:         "exact stable version",
			versionRange: "[1.0.0]",
			expected:     false,
			description:  "Exact version with no prerelease",
		},
		{
			name:         "exact prerelease version",
			versionRange: "[1.0.0-alpha]",
			expected:     true,
			description:  "Exact version with prerelease",
		},
		{
			name:         "range with stable min and max",
			versionRange: "[1.0.0,2.0.0)",
			expected:     false,
			description:  "Range with stable minimum and maximum",
		},
		{
			name:         "range with prerelease min",
			versionRange: "[1.0.0-alpha,2.0.0)",
			expected:     true,
			description:  "Range with prerelease minimum",
		},
		{
			name:         "range with prerelease max",
			versionRange: "[1.0.0,2.0.0-beta)",
			expected:     true,
			description:  "Range with prerelease maximum",
		},
		{
			name:         "open ended range stable",
			versionRange: "1.0.0",
			expected:     false,
			description:  "Open-ended range with stable version",
		},
		{
			name:         "invalid version range",
			versionRange: "invalid",
			expected:     false,
			description:  "Invalid range defaults to not allowing prerelease",
		},
		{
			name:         "asterisk wildcard",
			versionRange: "*",
			expected:     false,
			description:  "Wildcard does not have prerelease in min/max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrereleaseAllowed(tt.versionRange)
			if result != tt.expected {
				t.Errorf("isPrereleaseAllowed(%q) = %v, want %v (%s)",
					tt.versionRange, result, tt.expected, tt.description)
			}
		})
	}
}
