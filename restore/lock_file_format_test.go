package restore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLockFile_Save(t *testing.T) {
	tests := []struct {
		name        string
		lockFile    *LockFile
		expectError bool
	}{
		{
			name: "save basic lock file",
			lockFile: &LockFile{
				Version: 3,
				Targets: map[string]Target{
					"net8.0": {},
				},
				Libraries: map[string]Library{
					"Newtonsoft.Json/13.0.3": {
						Type: "package",
						Path: "newtonsoft.json/13.0.3",
					},
				},
				ProjectFileDependencyGroups: map[string][]string{
					"net8.0": {"Newtonsoft.Json >= 13.0.3"},
				},
				PackageFolders: map[string]PackageFolder{
					"/tmp/packages": {},
				},
				Project: ProjectInfo{
					Version: "1.0.0",
					Restore: Info{
						ProjectUniqueName:        "/tmp/test.csproj",
						ProjectName:              "test",
						ProjectPath:              "/tmp/test.csproj",
						PackagesPath:             "/tmp/packages",
						OutputPath:               "/tmp/obj",
						ProjectStyle:             "PackageReference",
						Sources:                  map[string]SourceInfo{},
						FallbackFolders:          []string{},
						ConfigFilePaths:          []string{},
						OriginalTargetFrameworks: []string{"net8.0"},
						Frameworks: map[string]FrameworkInfo{
							"net8.0": {
								TargetAlias:       "net8.0",
								ProjectReferences: map[string]any{},
							},
						},
					},
					Frameworks: map[string]ProjectFrameworkInfo{
						"net8.0": {
							TargetAlias: "net8.0",
							Dependencies: map[string]DependencyInfo{
								"Newtonsoft.Json": {
									Target:  "Package",
									Version: "13.0.3",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "save empty lock file",
			lockFile: &LockFile{
				Version:                     3,
				Targets:                     map[string]Target{},
				Libraries:                   map[string]Library{},
				ProjectFileDependencyGroups: map[string][]string{},
				PackageFolders:              map[string]PackageFolder{},
				Project: ProjectInfo{
					Version: "1.0.0",
					Restore: Info{
						Sources:         map[string]SourceInfo{},
						FallbackFolders: []string{},
						ConfigFilePaths: []string{},
						Frameworks:      map[string]FrameworkInfo{},
					},
					Frameworks: map[string]ProjectFrameworkInfo{},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			assetsPath := filepath.Join(tmpDir, "project.assets.json")

			// Save lock file
			err := tt.lockFile.Save(assetsPath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify file exists
				if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
					t.Error("lock file was not created")
				}

				// Verify file is valid JSON
				data, err := os.ReadFile(assetsPath)
				if err != nil {
					t.Errorf("failed to read lock file: %v", err)
				}

				if len(data) == 0 {
					t.Error("lock file is empty")
				}

				// Check for basic JSON structure
				content := string(data)
				if content[0] != '{' || content[len(content)-1] != '}' {
					t.Error("lock file is not valid JSON")
				}
			}
		})
	}
}

func TestLockFile_Save_InvalidPath(t *testing.T) {
	lockFile := &LockFile{
		Version: 3,
	}

	// Use null device path which is invalid on all platforms
	var invalidPath string
	if runtime.GOOS == "windows" {
		invalidPath = `\\?\NUL\invalid\path\project.assets.json`
	} else {
		invalidPath = "/dev/null/invalid/path/project.assets.json"
	}

	err := lockFile.Save(invalidPath)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestLockFile_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	objDir := filepath.Join(tmpDir, "obj")
	assetsPath := filepath.Join(objDir, "project.assets.json")

	lockFile := &LockFile{
		Version:                     3,
		Targets:                     map[string]Target{},
		Libraries:                   map[string]Library{},
		ProjectFileDependencyGroups: map[string][]string{},
		PackageFolders:              map[string]PackageFolder{},
		Project: ProjectInfo{
			Version: "1.0.0",
			Restore: Info{
				Sources:         map[string]SourceInfo{},
				FallbackFolders: []string{},
				ConfigFilePaths: []string{},
				Frameworks:      map[string]FrameworkInfo{},
			},
			Frameworks: map[string]ProjectFrameworkInfo{},
		},
	}

	err := lockFile.Save(assetsPath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(objDir); os.IsNotExist(err) {
		t.Error("directory was not created")
	}

	// Verify file was created
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		t.Error("lock file was not created")
	}
}

func TestLockFile_Save_WriteError(t *testing.T) {
	tmpDir := t.TempDir()

	lockFile := &LockFile{
		Version:                     3,
		Targets:                     map[string]Target{},
		Libraries:                   map[string]Library{},
		ProjectFileDependencyGroups: map[string][]string{},
		PackageFolders:              map[string]PackageFolder{},
		Project: ProjectInfo{
			Version: "1.0.0",
			Restore: Info{
				Sources:         map[string]SourceInfo{},
				FallbackFolders: []string{},
				ConfigFilePaths: []string{},
				Frameworks:      map[string]FrameworkInfo{},
			},
			Frameworks: map[string]ProjectFrameworkInfo{},
		},
	}

	// Create a file where we want a directory, causing mkdir to fail
	filePath := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to save to a path where a file exists instead of directory
	assetsPath := filepath.Join(filePath, "project.assets.json")
	err := lockFile.Save(assetsPath)
	if err == nil {
		t.Error("expected error when directory creation fails")
	}
}
