package solution_test

import (
	"testing"

	"github.com/willibrandon/gonuget/solution"
)

func TestSlnfParser_CanParse(t *testing.T) {
	parser := solution.NewSlnfParser()

	tests := []struct {
		path string
		want bool
	}{
		{"solution.slnf", true},
		{"Solution.SLNF", true},
		{"My.Solution.slnf", true},
		{"solution.sln", false},
		{"solution.slnx", false},
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

func TestSlnfParser_Parse(t *testing.T) {
	parser := solution.NewSlnfParser()

	t.Run("filter with parent solution", func(t *testing.T) {
		sol, err := parser.Parse("testdata/filter.slnf")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Filter should only include the 2 specified projects
		if len(sol.Projects) != 2 {
			t.Errorf("Projects count = %d, want 2", len(sol.Projects))
		}

		// Verify filtered projects
		expectedProjects := map[string]bool{
			"WebApi":    false,
			"DataLayer": false,
		}

		for _, project := range sol.Projects {
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
				t.Errorf("Expected project %q not found in filter", name)
			}
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := parser.Parse("testdata/nonexistent.slnf")
		if err == nil {
			t.Error("Parse() should error on missing file")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		_, err := parser.Parse("testdata/test.txt")
		if err == nil {
			t.Error("Parse() should error on non-.slnf file")
		}
	})
}
