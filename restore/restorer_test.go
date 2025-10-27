package restore

import (
	"context"
	"fmt"
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

	if len(result.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(result.Packages))
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

	if len(result.Packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.Packages))
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

	console := &mockConsole{}
	opts := &Options{
		PackagesFolder: packagesFolder,
		Force:          false,
		Sources:        []string{"https://api.nuget.org/v3/index.json"},
	}

	restorer := NewRestorer(opts, console)
	packageRefs := proj.GetPackageReferences()

	result, err := restorer.Restore(context.Background(), proj, packageRefs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(result.Packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(result.Packages))
	}

	// Check console output for "already cached" message
	found := false
	for _, msg := range console.messages {
		if strings.Contains(msg, "already cached") {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'already cached' message")
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
				return []string{projPath}, projPath
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
		if strings.Contains(msg, "Restoring packages") {
			foundRestore = true
		}
		if strings.Contains(msg, "Restored") {
			foundRestored = true
		}
	}

	if !foundRestore {
		t.Error("expected 'Restoring packages' message")
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

	if !strings.Contains(err.Error(), "failed to download package") {
		t.Errorf("expected 'failed to download package' error, got: %v", err)
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

	if len(result.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(result.Packages))
	}
}

func TestFindProjectFile_GetWorkingDirectory(t *testing.T) {
	// Test case where explicit path is provided
	args := []string{"/some/path/test.csproj"}
	path, err := findProjectFile(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if path != "/some/path/test.csproj" {
		t.Errorf("expected /some/path/test.csproj, got %s", path)
	}
}
