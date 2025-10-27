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
				Packages: []PackageInfo{
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
				Packages: []PackageInfo{
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
				Packages: []PackageInfo{},
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
				Packages: []PackageInfo{
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
