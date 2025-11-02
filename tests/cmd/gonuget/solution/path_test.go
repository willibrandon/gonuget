package solution_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/solution"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Windows path with backslashes",
			input:    "src\\MyProject\\MyProject.csproj",
			expected: "src/MyProject/MyProject.csproj",
		},
		{
			name:     "Already normalized path",
			input:    "src/MyProject/MyProject.csproj",
			expected: "src/MyProject/MyProject.csproj",
		},
		{
			name:     "Path with double backslashes",
			input:    "src\\\\MyProject\\\\MyProject.csproj",
			expected: "src/MyProject/MyProject.csproj",
		},
		{
			name:     "Path with mixed separators",
			input:    "src\\MyProject/SubFolder\\Project.csproj",
			expected: "src/MyProject/SubFolder/Project.csproj",
		},
		{
			name:     "Path with duplicate forward slashes",
			input:    "src//MyProject///Project.csproj",
			expected: "src/MyProject/Project.csproj",
		},
		{
			name:     "Relative parent path",
			input:    "..\\..\\shared\\Common.csproj",
			expected: "../../shared/Common.csproj",
		},
		{
			name:     "Current directory path",
			input:    ".\\MyProject.csproj",
			expected: "./MyProject.csproj",
		},
		{
			name:     "UNC path",
			input:    "\\\\server\\share\\project\\MyProject.csproj",
			expected: "//server/share/project/MyProject.csproj",
		},
		{
			name:     "Windows absolute path",
			input:    "C:\\Users\\Developer\\Projects\\MyProject.csproj",
			expected: "C:/Users/Developer/Projects/MyProject.csproj",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "Path with trailing slash",
			input:    "src\\MyProject\\",
			expected: "src/MyProject/",
		},
		{
			name:     "Path with spaces",
			input:    "My Documents\\Visual Studio Projects\\App.csproj",
			expected: "My Documents/Visual Studio Projects/App.csproj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solution.NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolveProjectPath(t *testing.T) {
	// Get a temp directory for testing
	solutionDir := t.TempDir()

	tests := []struct {
		name        string
		solutionDir string
		projectPath string
		expected    string
	}{
		{
			name:        "Relative path in same directory",
			solutionDir: solutionDir,
			projectPath: "MyProject.csproj",
			expected:    filepath.Join(solutionDir, "MyProject.csproj"),
		},
		{
			name:        "Relative path in subdirectory",
			solutionDir: solutionDir,
			projectPath: "src\\MyProject\\MyProject.csproj",
			expected:    filepath.Join(solutionDir, "src", "MyProject", "MyProject.csproj"),
		},
		{
			name:        "Relative path with parent directory",
			solutionDir: filepath.Join(solutionDir, "solution"),
			projectPath: "..\\shared\\Common.csproj",
			expected:    filepath.Join(solutionDir, "shared", "Common.csproj"),
		},
		{
			name:        "Already absolute Unix path",
			solutionDir: solutionDir,
			projectPath: "/home/user/projects/MyProject.csproj",
			expected:    "/home/user/projects/MyProject.csproj",
		},
		{
			name:        "Empty project path",
			solutionDir: solutionDir,
			projectPath: "",
			expected:    "",
		},
		{
			name:        "Current directory reference",
			solutionDir: solutionDir,
			projectPath: ".\\MyProject.csproj",
			expected:    filepath.Join(solutionDir, "MyProject.csproj"),
		},
		{
			name:        "Complex relative path",
			solutionDir: solutionDir,
			projectPath: "..\\..\\external\\libs\\Library.csproj",
			expected:    filepath.Clean(filepath.Join(solutionDir, "..", "..", "external", "libs", "Library.csproj")),
		},
	}

	// Add Windows-specific tests if on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name        string
			solutionDir string
			projectPath string
			expected    string
		}{
			name:        "Windows absolute path",
			solutionDir: solutionDir,
			projectPath: "C:\\Projects\\MyProject.csproj",
			expected:    "C:\\Projects\\MyProject.csproj",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solution.ResolveProjectPath(tt.solutionDir, tt.projectPath)
			if result != tt.expected {
				t.Errorf("ResolveProjectPath(%q, %q) = %q, want %q",
					tt.solutionDir, tt.projectPath, result, tt.expected)
			}
		})
	}
}

func TestConvertToSystemPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Will be adjusted based on OS
	}{
		{
			name:  "Forward slash path",
			input: "src/MyProject/MyProject.csproj",
		},
		{
			name:  "Backslash path",
			input: "src\\MyProject\\MyProject.csproj",
		},
		{
			name:  "Mixed separators",
			input: "src\\MyProject/SubFolder\\Project.csproj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solution.ConvertToSystemPath(tt.input)

			// Verify the path uses the correct separator for this OS
			if runtime.GOOS == "windows" {
				if strings.Contains(result, "/") {
					t.Errorf("ConvertToSystemPath(%q) = %q, should not contain / on Windows",
						tt.input, result)
				}
			} else {
				if strings.Contains(result, "\\") {
					t.Errorf("ConvertToSystemPath(%q) = %q, should not contain \\ on Unix",
						tt.input, result)
				}
			}
		})
	}
}

func TestPathResolver(t *testing.T) {
	solutionDir := t.TempDir()
	resolver := solution.NewPathResolver(solutionDir)

	tests := []struct {
		name        string
		projectPath string
		expected    string
	}{
		{
			name:        "Simple relative path",
			projectPath: "MyProject.csproj",
			expected:    filepath.Join(solutionDir, "MyProject.csproj"),
		},
		{
			name:        "Windows-style relative path",
			projectPath: "src\\MyProject\\MyProject.csproj",
			expected:    filepath.Join(solutionDir, "src", "MyProject", "MyProject.csproj"),
		},
		{
			name:        "Unix-style relative path",
			projectPath: "src/MyProject/MyProject.csproj",
			expected:    filepath.Join(solutionDir, "src", "MyProject", "MyProject.csproj"),
		},
		{
			name:        "Parent directory reference",
			projectPath: "../shared/Common.csproj",
			expected:    filepath.Clean(filepath.Join(solutionDir, "..", "shared", "Common.csproj")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolvePath(tt.projectPath)
			if result != tt.expected {
				t.Errorf("ResolvePath(%q) = %q, want %q",
					tt.projectPath, result, tt.expected)
			}
		})
	}
}

func TestUNCPathHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("UNC paths are Windows-specific")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic UNC path",
			input:    "\\\\server\\share\\project.csproj",
			expected: "\\\\server\\share\\project.csproj",
		},
		{
			name:     "UNC path with forward slashes",
			input:    "//server/share/project.csproj",
			expected: "\\\\server\\share\\project.csproj",
		},
		{
			name:     "UNC path with mixed separators",
			input:    "\\\\server/share\\folder/project.csproj",
			expected: "\\\\server\\share\\folder\\project.csproj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solution.ConvertToSystemPath(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToSystemPath(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestMixedPlatformSolutionFiles(t *testing.T) {
	solutionDir := t.TempDir()

	// Simulate a solution file created on Windows with Windows paths
	windowsPaths := []string{
		"src\\Core\\Core.csproj",
		"src\\Web\\Web.csproj",
		"..\\shared\\Common\\Common.csproj",
		"tests\\UnitTests\\UnitTests.csproj",
	}

	// Test that these paths work correctly on any platform
	for _, path := range windowsPaths {
		t.Run("Windows path: "+path, func(t *testing.T) {
			resolved := solution.ResolveProjectPath(solutionDir, path)

			// The resolved path should:
			// 1. Be absolute
			// 2. Use the correct separator for the current OS
			// 3. Be a valid path
			if !filepath.IsAbs(resolved) {
				t.Errorf("Resolved path %q is not absolute", resolved)
			}

			// Check that the path uses the correct separator
			if runtime.GOOS == "windows" {
				if strings.Contains(resolved, "/") && !strings.HasPrefix(resolved, "//") {
					t.Errorf("Resolved path %q contains / on Windows", resolved)
				}
			} else {
				if strings.Contains(resolved, "\\") {
					t.Errorf("Resolved path %q contains \\ on Unix", resolved)
				}
			}
		})
	}
}

func TestDuplicateSeparatorNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Double forward slashes",
			input:    "src//project//file.csproj",
			expected: "src/project/file.csproj",
		},
		{
			name:     "Double backslashes",
			input:    "src\\\\project\\\\file.csproj",
			expected: "src/project/file.csproj",
		},
		{
			name:     "Triple slashes",
			input:    "src///project///file.csproj",
			expected: "src/project/file.csproj",
		},
		{
			name:     "Mixed multiple separators",
			input:    "src\\\\//project//\\\\file.csproj",
			expected: "src/project/file.csproj",
		},
		{
			name:     "Leading double slashes (UNC)",
			input:    "//server/share/file.csproj",
			expected: "//server/share/file.csproj", // Preserve UNC prefix
		},
		{
			name:     "Leading double backslashes (UNC)",
			input:    "\\\\server\\share\\file.csproj",
			expected: "//server/share/file.csproj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solution.NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}