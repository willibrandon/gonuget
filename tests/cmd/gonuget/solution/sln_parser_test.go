package solution_test

import (
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/solution"
)

func TestSlnParser_CanParse(t *testing.T) {
	parser := solution.NewSlnParser()

	tests := []struct {
		path string
		want bool
	}{
		{"solution.sln", true},
		{"Solution.SLN", true},
		{"My.Solution.sln", true},
		{"solution.slnx", false},
		{"solution.slnf", false},
		{"project.csproj", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := parser.CanParse(tt.path); got != tt.want {
				t.Errorf("CanParse(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSlnParser_Parse(t *testing.T) {
	parser := solution.NewSlnParser()

	t.Run("simple solution", func(t *testing.T) {
		sol, err := parser.Parse("testdata/simple.sln")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Check basic properties
		if sol.FormatVersion != "12.00" {
			t.Errorf("FormatVersion = %v, want 12.00", sol.FormatVersion)
		}

		// Check projects count (should have 2 projects, excluding solution folder)
		if len(sol.Projects) != 2 {
			t.Errorf("Projects count = %d, want 2", len(sol.Projects))
		}

		// Check solution folders count
		if len(sol.SolutionFolders) != 1 {
			t.Errorf("SolutionFolders count = %d, want 1", len(sol.SolutionFolders))
		}

		// Verify project names
		expectedNames := map[string]bool{
			"WebApi":    false,
			"DataLayer": false,
		}

		for _, project := range sol.Projects {
			if _, exists := expectedNames[project.Name]; exists {
				expectedNames[project.Name] = true
			}

			// Check that paths are normalized (no backslashes on Unix)
			if filepath.Separator == '/' && containsBackslash(project.Path) {
				t.Errorf("Project %q has Windows path on Unix: %s", project.Name, project.Path)
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("Expected project %q not found", name)
			}
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := parser.Parse("testdata/nonexistent.sln")
		if err == nil {
			t.Error("Parse() should error on missing file")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		_, err := parser.Parse("testdata/test.txt")
		if err == nil {
			t.Error("Parse() should error on non-.sln file")
		}
	})
}

func TestSlnParser_ProjectTypes(t *testing.T) {
	parser := solution.NewSlnParser()
	sol, err := parser.Parse("testdata/simple.sln")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	for _, project := range sol.Projects {
		// All projects in simple.sln should be C# projects
		if project.TypeGUID != solution.ProjectTypeCSProject {
			t.Errorf("Project %q has unexpected TypeGUID: %s", project.Name, project.TypeGUID)
		}

		// All should be recognized as .NET projects
		if !project.IsNETProject() {
			t.Errorf("Project %q should be recognized as .NET project", project.Name)
		}
	}
}

func containsBackslash(s string) bool {
	for _, r := range s {
		if r == '\\' {
			return true
		}
	}
	return false
}
