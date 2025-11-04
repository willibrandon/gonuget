package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestWarningWriter(t *testing.T) {
	tests := []struct {
		name     string
		action   func(w *output.WarningWriter)
		expected string
	}{
		{
			name: "Basic warning",
			action: func(w *output.WarningWriter) {
				w.Warning("Something went wrong")
			},
			expected: "Warning: Something went wrong\n",
		},
		{
			name: "Warning with formatting",
			action: func(w *output.WarningWriter) {
				w.Warning("File %s not found at line %d", "test.go", 42)
			},
			expected: "Warning: File test.go not found at line 42\n",
		},
		{
			name: "Project warning",
			action: func(w *output.WarningWriter) {
				w.WriteProjectWarning("/path/to/project.csproj", "invalid framework")
			},
			expected: "Warning: Project '/path/to/project.csproj': invalid framework\n",
		},
		{
			name: "Missing project warning",
			action: func(w *output.WarningWriter) {
				w.WriteMissingProjectWarning("/path/to/missing.csproj")
			},
			expected: "Warning: Project file not found: /path/to/missing.csproj\n",
		},
		{
			name: "Solution warning",
			action: func(w *output.WarningWriter) {
				w.WriteSolutionWarning("/path/to/solution.sln", "contains no projects")
			},
			expected: "Warning: Solution '/path/to/solution.sln': contains no projects\n",
		},
		{
			name: "Warn alias",
			action: func(w *output.WarningWriter) {
				w.Warn("Using alias method")
			},
			expected: "Warning: Using alias method\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := output.NewWarningWriterWithOutput(&buf)

			tt.action(w)

			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}

			// Verify format starts with "Warning: "
			if !strings.HasPrefix(result, "Warning: ") {
				t.Error("Warning output should start with 'Warning: ' prefix")
			}
		})
	}
}

func TestGlobalWarningFunctions(t *testing.T) {
	// Test that global functions exist and can be called
	// (We can't easily test their output since they write to stderr)

	// These should not panic
	output.Warning("test warning")
	output.ProjectWarning("/test.csproj", "test message")
	output.MissingProjectWarning("/missing.csproj")
	output.SolutionWarning("/test.sln", "test message")
}

func TestWarningOutput_ConsistentFormat(t *testing.T) {
	// Test that all warnings follow consistent format
	var buf bytes.Buffer
	w := output.NewWarningWriterWithOutput(&buf)

	// Generate various warnings
	w.Warning("Simple warning")
	w.WriteProjectWarning("project.csproj", "project issue")
	w.WriteMissingProjectWarning("missing.csproj")
	w.WriteSolutionWarning("solution.sln", "solution issue")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify all lines start with "Warning: "
	for i, line := range lines {
		if !strings.HasPrefix(line, "Warning: ") {
			t.Errorf("Line %d doesn't start with 'Warning: ': %s", i+1, line)
		}
	}

	// Verify we got all expected lines
	if len(lines) != 4 {
		t.Errorf("Expected 4 warning lines, got %d", len(lines))
	}
}

func TestWarningWriter_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	w := output.NewWarningWriterWithOutput(&buf)

	// Empty warning should still show prefix
	w.Warning("")

	result := buf.String()
	if result != "Warning: \n" {
		t.Errorf("Empty warning should produce 'Warning: \\n', got %q", result)
	}
}

func TestWarningWriter_MultilineMessage(t *testing.T) {
	var buf bytes.Buffer
	w := output.NewWarningWriterWithOutput(&buf)

	// Multiline messages should be handled properly
	w.Warning("Line 1\nLine 2\nLine 3")

	result := buf.String()
	expected := "Warning: Line 1\nLine 2\nLine 3\n"

	if result != expected {
		t.Errorf("Multiline warning not handled correctly.\nExpected: %q\nGot: %q", expected, result)
	}
}
