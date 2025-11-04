package solution_test

import (
	"testing"

	"github.com/willibrandon/gonuget/solution"
)

func TestSlnxParser_CanParse(t *testing.T) {
	parser := solution.NewSlnxParser()

	tests := []struct {
		path string
		want bool
	}{
		{"solution.slnx", true},
		{"Solution.SLNX", true},
		{"My.Solution.slnx", true},
		{"solution.sln", false},
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

func TestSlnxParser_Parse(t *testing.T) {
	parser := solution.NewSlnxParser()

	t.Run("simple slnx", func(t *testing.T) {
		sol, err := parser.Parse("testdata/simple.slnx")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Check projects count
		if len(sol.Projects) != 3 {
			t.Errorf("Projects count = %d, want 3", len(sol.Projects))
		}

		// Check solution folders
		if len(sol.SolutionFolders) != 2 {
			t.Errorf("SolutionFolders count = %d, want 2", len(sol.SolutionFolders))
		}

		// Verify project names
		expectedProjects := map[string]bool{
			"WebApi":    false,
			"DataLayer": false,
			"UnitTests": false,
		}

		for _, project := range sol.Projects {
			// Extract project name from path if not set
			if project.Name == "" {
				t.Errorf("Project name should not be empty")
			}

			if _, exists := expectedProjects[project.Name]; exists {
				expectedProjects[project.Name] = true
			}

			// Check normalized paths
			if containsBackslash(project.Path) {
				t.Errorf("Project %q has backslash in path: %s", project.Name, project.Path)
			}
		}

		for name, found := range expectedProjects {
			if !found {
				t.Errorf("Expected project %q not found", name)
			}
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := parser.Parse("testdata/nonexistent.slnx")
		if err == nil {
			t.Error("Parse() should error on missing file")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		_, err := parser.Parse("testdata/test.txt")
		if err == nil {
			t.Error("Parse() should error on non-.slnx file")
		}
	})
}
