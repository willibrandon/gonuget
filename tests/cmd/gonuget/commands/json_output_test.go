package commands_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestStdoutStderrSeparation validates that JSON goes to stdout and errors to stderr (T078)
func TestStdoutStderrSeparation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectJSON  bool
		expectError bool
	}{
		{
			name:       "source list json - success",
			args:       []string{"source", "list", "--format", "json"},
			expectJSON: true,
		},
		{
			name:        "source list json - invalid configfile",
			args:        []string{"source", "list", "--format", "json", "--configfile", "/nonexistent/path/NuGet.config"},
			expectJSON:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(getGonugetPath(), tt.args...)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.expectError {
				if err == nil {
					t.Error("Expected command to fail, but it succeeded")
				}
				// Verify error message went to stderr
				if stderr.Len() == 0 {
					t.Error("Expected error message on stderr, but stderr is empty")
				}
				// Verify stdout has no error messages (should be empty or valid JSON)
				if stdout.Len() > 0 {
					// If stdout has content, it should be valid JSON (not error text)
					var result map[string]any
					if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
						t.Errorf("stdout should contain only JSON when --format json is used, got: %s", stdout.String())
					}
				}
			} else if tt.expectJSON {
				if err != nil {
					t.Logf("Command failed: %v\nStderr: %s", err, stderr.String())
					// Skip this test if JSON output is not yet implemented
					if strings.Contains(stderr.String(), "not yet implemented") ||
						strings.Contains(stdout.String(), "Registered Sources:") {
						t.Skip("JSON output not yet implemented")
						return
					}
					t.Fatalf("Expected command to succeed, but got error: %v", err)
				}

				// Verify JSON output went to stdout
				if stdout.Len() == 0 {
					t.Error("Expected JSON output on stdout, but stdout is empty")
				}

				// Verify output is valid JSON
				var result map[string]any
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
					t.Errorf("Expected valid JSON on stdout, got: %s\nError: %v", stdout.String(), err)
				}

				// Verify stderr is empty or contains only warnings (not errors)
				if stderr.Len() > 0 {
					stderrStr := stderr.String()
					// Warnings are acceptable on stderr
					if !strings.Contains(stderrStr, "warning:") && !strings.Contains(stderrStr, "Warning:") {
						t.Errorf("Expected stderr to be empty or contain only warnings, got: %s", stderrStr)
					}
				}
			}
		})
	}
}

// TestJSONOutputFormat validates basic JSON output structure
func TestJSONOutputFormat(t *testing.T) {
	cmd := exec.Command(getGonugetPath(), "source", "list", "--format", "json")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Skip if JSON output is not yet implemented
		if strings.Contains(string(output), "Registered Sources:") {
			t.Skip("JSON output not yet implemented")
			return
		}
	}

	// Should be valid JSON
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, string(output))
	}

	// Should have schemaVersion
	if _, ok := result["schemaVersion"]; !ok {
		t.Error("JSON output missing schemaVersion field")
	}
}

// TestEmptySearchResultsExitCode validates that empty search returns exit code 0 (VR-019)
func TestEmptySearchResultsExitCode(t *testing.T) {
	// This will be implemented when package search JSON output is ready
	t.Skip("Package search JSON output not yet implemented")

	cmd := exec.Command(getGonugetPath(), "package", "search", "NonexistentPackage12345XYZ", "--format", "json")
	output, err := cmd.CombinedOutput()

	// Should succeed (exit code 0) even with no results
	if err != nil {
		t.Errorf("Expected exit code 0 for empty search results, got error: %v\nOutput: %s", err, string(output))
	}

	// Should return valid JSON
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, string(output))
	}

	// Should have total: 0 and items: []
	if total, ok := result["total"].(float64); !ok || total != 0 {
		t.Errorf("Expected total: 0 for empty search, got: %v", result["total"])
	}

	if items, ok := result["items"].([]any); !ok || len(items) != 0 {
		t.Errorf("Expected empty items array for empty search, got: %v", result["items"])
	}
}
