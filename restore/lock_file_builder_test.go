package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

func TestLockFileBuilder_Build(t *testing.T) {
	tests := []struct {
		name           string
		setupProject   func(t *testing.T) *project.Project
		result         *Result
		validateResult func(t *testing.T, lf *LockFile)
	}{
		{
			name: "build lock file with single package",
			setupProject: func(t *testing.T) *project.Project {
				tmpDir := t.TempDir()
				projPath := filepath.Join(tmpDir, "test.csproj")

				content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

				if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}

				proj, err := project.LoadProject(projPath)
				if err != nil {
					t.Fatal(err)
				}

				return proj
			},
			result: &Result{
				DirectPackages: []PackageInfo{
					{
						ID:      "Newtonsoft.Json",
						Version: "13.0.3",
						Path:    "/tmp/packages/newtonsoft.json/13.0.3",
					},
				},
			},
			validateResult: func(t *testing.T, lf *LockFile) {
				if lf.Version != 3 {
					t.Errorf("expected version 3, got %d", lf.Version)
				}

				if len(lf.Libraries) != 1 {
					t.Errorf("expected 1 library, got %d", len(lf.Libraries))
				}

				key := "Newtonsoft.Json/13.0.3"
				lib, ok := lf.Libraries[key]
				if !ok {
					t.Errorf("library %s not found", key)
				}

				if lib.Type != "package" {
					t.Errorf("expected type 'package', got %s", lib.Type)
				}

				if lf.Project.Version != "1.0.0" {
					t.Errorf("expected project version 1.0.0, got %s", lf.Project.Version)
				}
			},
		},
		{
			name: "build lock file with multiple packages",
			setupProject: func(t *testing.T) *project.Project {
				tmpDir := t.TempDir()
				projPath := filepath.Join(tmpDir, "test.csproj")

				content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="System.Text.Json" Version="8.0.0" />
  </ItemGroup>
</Project>`

				if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}

				proj, err := project.LoadProject(projPath)
				if err != nil {
					t.Fatal(err)
				}

				return proj
			},
			result: &Result{
				DirectPackages: []PackageInfo{
					{
						ID:      "Newtonsoft.Json",
						Version: "13.0.3",
						Path:    "/tmp/packages/newtonsoft.json/13.0.3",
					},
					{
						ID:      "System.Text.Json",
						Version: "8.0.0",
						Path:    "/tmp/packages/system.text.json/8.0.0",
					},
				},
			},
			validateResult: func(t *testing.T, lf *LockFile) {
				if len(lf.Libraries) != 2 {
					t.Errorf("expected 2 libraries, got %d", len(lf.Libraries))
				}

				if _, ok := lf.Libraries["Newtonsoft.Json/13.0.3"]; !ok {
					t.Error("Newtonsoft.Json not found in libraries")
				}

				if _, ok := lf.Libraries["System.Text.Json/8.0.0"]; !ok {
					t.Error("System.Text.Json not found in libraries")
				}
			},
		},
		{
			name: "build lock file with no packages",
			setupProject: func(t *testing.T) *project.Project {
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

				return proj
			},
			result: &Result{
				DirectPackages: []PackageInfo{},
			},
			validateResult: func(t *testing.T, lf *LockFile) {
				if len(lf.Libraries) != 0 {
					t.Errorf("expected 0 libraries, got %d", len(lf.Libraries))
				}
			},
		},
		{
			name: "build lock file with multi-TFM project",
			setupProject: func(t *testing.T) *project.Project {
				tmpDir := t.TempDir()
				projPath := filepath.Join(tmpDir, "test.csproj")

				content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net6.0;net7.0;net8.0</TargetFrameworks>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

				if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}

				proj, err := project.LoadProject(projPath)
				if err != nil {
					t.Fatal(err)
				}

				return proj
			},
			result: &Result{
				DirectPackages: []PackageInfo{
					{
						ID:      "Newtonsoft.Json",
						Version: "13.0.3",
						Path:    "/tmp/packages/newtonsoft.json/13.0.3",
					},
				},
			},
			validateResult: func(t *testing.T, lf *LockFile) {
				// For Chunk 5, we use first TFM only
				tfm := lf.Project.Restore.OriginalTargetFrameworks[0]
				if tfm != "net6.0" {
					t.Errorf("expected first TFM to be net6.0, got %s", tfm)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj := tt.setupProject(t)
			builder := NewLockFileBuilder()

			lockFile := builder.Build(proj, tt.result)

			if lockFile == nil {
				t.Fatal("expected lock file but got nil")
			}

			tt.validateResult(t, lockFile)
		})
	}
}

func TestNewLockFileBuilder(t *testing.T) {
	builder := NewLockFileBuilder()
	if builder == nil {
		t.Error("expected builder but got nil")
	}
}

// TestLockFileBuilder_ProjectFileDependencyGroups_OnlyDirectDeps validates that
// ProjectFileDependencyGroups contains ONLY direct dependencies, not transitive.
// Matches dotnet's behavior: direct deps in ProjectFileDependencyGroups, all deps in Libraries.
func TestLockFileBuilder_ProjectFileDependencyGroups_OnlyDirectDeps(t *testing.T) {
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Project with Serilog.Sinks.File (direct) which depends on Serilog (transitive)
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Serilog.Sinks.File" Version="5.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	home, _ := os.UserHomeDir()
	packagesPath := filepath.Join(home, ".nuget", "packages")

	// Simulate restore result with direct and transitive packages
	result := &Result{
		DirectPackages: []PackageInfo{
			{
				ID:       "Serilog.Sinks.File",
				Version:  "5.0.0",
				Path:     filepath.Join(packagesPath, "serilog.sinks.file", "5.0.0"),
				IsDirect: true,
			},
		},
		TransitivePackages: []PackageInfo{
			{
				ID:       "Serilog",
				Version:  "2.10.0",
				Path:     filepath.Join(packagesPath, "serilog", "2.10.0"),
				IsDirect: false,
			},
		},
	}

	builder := NewLockFileBuilder()
	lockFile := builder.Build(proj, result)

	// Verify ProjectFileDependencyGroups contains ONLY direct dependency
	tfm := "net9.0"
	directDeps := lockFile.ProjectFileDependencyGroups[tfm]
	if len(directDeps) != 1 {
		t.Errorf("Expected 1 direct dependency in ProjectFileDependencyGroups, got %d", len(directDeps))
	}

	expectedDep := "Serilog.Sinks.File >= 5.0.0"
	if len(directDeps) > 0 && directDeps[0] != expectedDep {
		t.Errorf("Expected ProjectFileDependencyGroups[%s] to contain %q, got %q", tfm, expectedDep, directDeps[0])
	}

	// Verify ProjectFileDependencyGroups does NOT contain transitive dependency
	for _, dep := range directDeps {
		if contains(dep, "Serilog") && !contains(dep, "Serilog.Sinks.File") {
			t.Errorf("ProjectFileDependencyGroups should NOT contain transitive dependency Serilog, but found: %q", dep)
		}
	}

	// Verify Libraries contains BOTH direct and transitive packages
	if len(lockFile.Libraries) != 2 {
		t.Errorf("Expected 2 libraries (direct + transitive), got %d", len(lockFile.Libraries))
	}

	// Verify both packages are in Libraries
	if _, ok := lockFile.Libraries["Serilog.Sinks.File/5.0.0"]; !ok {
		t.Error("Libraries should contain direct dependency Serilog.Sinks.File/5.0.0")
	}

	if _, ok := lockFile.Libraries["Serilog/2.10.0"]; !ok {
		t.Error("Libraries should contain transitive dependency Serilog/2.10.0")
	}
}

// TestLockFileBuilder_Libraries_LowercasePaths validates that Libraries paths
// use lowercase package IDs, matching dotnet's behavior.
func TestLockFileBuilder_Libraries_LowercasePaths(t *testing.T) {
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	home, _ := os.UserHomeDir()
	packagesPath := filepath.Join(home, ".nuget", "packages")

	// Package ID has mixed case, but path should be lowercase
	result := &Result{
		DirectPackages: []PackageInfo{
			{
				ID:       "Newtonsoft.Json",
				Version:  "13.0.3",
				Path:     filepath.Join(packagesPath, "newtonsoft.json", "13.0.3"),
				IsDirect: true,
			},
		},
	}

	builder := NewLockFileBuilder()
	lockFile := builder.Build(proj, result)

	// Verify library key uses original case
	key := "Newtonsoft.Json/13.0.3"
	lib, ok := lockFile.Libraries[key]
	if !ok {
		t.Fatalf("Library %s not found", key)
	}

	// Verify path uses lowercase package ID (relative path format matching NuGet.Client)
	// Expected: newtonsoft.json/13.0.3 (relative, not absolute)
	expectedPath := "newtonsoft.json/13.0.3"
	if lib.Path != expectedPath {
		t.Errorf("Expected relative path with lowercase package ID:\n  got:  %s\n  want: %s", lib.Path, expectedPath)
	}

	// Verify path does NOT use original case
	wrongPath := "Newtonsoft.Json/13.0.3"
	if lib.Path == wrongPath {
		t.Error("Path should use lowercase package ID, not original case")
	}
}

// TestLockFileBuilder_Build_MultiFramework validates lock file generation for
// projects targeting multiple frameworks.
func TestLockFileBuilder_Build_MultiFramework(t *testing.T) {
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Multi-framework project
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net6.0;net7.0;net8.0</TargetFrameworks>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		t.Fatal(err)
	}

	home, _ := os.UserHomeDir()
	packagesPath := filepath.Join(home, ".nuget", "packages")

	result := &Result{
		DirectPackages: []PackageInfo{
			{
				ID:       "Newtonsoft.Json",
				Version:  "13.0.3",
				Path:     filepath.Join(packagesPath, "newtonsoft.json", "13.0.3"),
				IsDirect: true,
			},
		},
	}

	builder := NewLockFileBuilder()
	lockFile := builder.Build(proj, result)

	// Verify targets exist for all frameworks
	if len(lockFile.Targets) == 0 {
		t.Error("Expected targets for multi-framework project")
	}

	// Verify project file dependency groups for all frameworks
	tfms := proj.GetTargetFrameworks()
	if len(tfms) != 3 {
		t.Errorf("Expected 3 target frameworks, got %d", len(tfms))
	}

	for _, tfm := range tfms {
		deps, exists := lockFile.ProjectFileDependencyGroups[tfm]
		if !exists {
			t.Errorf("Missing ProjectFileDependencyGroups for %s", tfm)
			continue
		}
		if len(deps) == 0 {
			t.Errorf("Expected dependencies for %s", tfm)
		}
	}
}

// contains checks if string s contains substring substr (helper for tests).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
