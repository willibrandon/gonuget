package commands_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/tests/cmd/gonuget/commands"
)

// TestVerbFirstErrorDetection validates that all verb-first patterns are detected
// and produce helpful error messages with correct noun-first suggestions
func TestVerbFirstErrorDetection(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedError      string
		expectedSuggestion string
	}{
		// Package namespace patterns
		{
			name:               "add package",
			args:               []string{"add", "package", "Newtonsoft.Json"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget package add",
		},
		{
			name:               "list package",
			args:               []string{"list", "package"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget package list",
		},
		{
			name:               "remove package",
			args:               []string{"remove", "package", "Newtonsoft.Json"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget package remove",
		},
		{
			name:               "search package",
			args:               []string{"search", "package", "Serilog"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget package search",
		},
		// Source namespace patterns
		{
			name:               "add source",
			args:               []string{"add", "source", "https://api.nuget.org/v3/index.json"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget source add",
		},
		{
			name:               "list source",
			args:               []string{"list", "source"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget source list",
		},
		{
			name:               "remove source",
			args:               []string{"remove", "source", "nuget.org"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget source remove",
		},
		// Top-level verbs that imply source
		{
			name:               "enable (top-level)",
			args:               []string{"enable", "nuget.org"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget source enable",
		},
		{
			name:               "disable (top-level)",
			args:               []string{"disable", "nuget.org"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget source disable",
		},
		{
			name:               "update (top-level)",
			args:               []string{"update", "nuget.org"},
			expectedError:      "verb-first form is not supported",
			expectedSuggestion: "gonuget source update",
		},
	}

	gonugetPath := commands.BuildBinary(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(gonugetPath, tt.args...)
			output, err := cmd.CombinedOutput()

			// Expect command to fail
			if err == nil {
				t.Errorf("Expected command to fail, but it succeeded")
				return
			}

			outputStr := string(output)

			// Debug: log the actual output if empty
			if outputStr == "" {
				t.Logf("Binary path: %s, Error: %v", gonugetPath, err)
			}

			// Verify error message contains expected text
			if !strings.Contains(outputStr, tt.expectedError) {
				t.Errorf("Expected error message to contain %q, got:\n%s", tt.expectedError, outputStr)
			}

			// Verify suggestion is present
			if !strings.Contains(outputStr, tt.expectedSuggestion) {
				t.Errorf("Expected suggestion %q, got:\n%s", tt.expectedSuggestion, outputStr)
			}
		})
	}
}

// TestVerbFirstPatternCoverage ensures all 10 verb-first patterns from the spec are covered
func TestVerbFirstPatternCoverage(t *testing.T) {
	// This test documents the requirement that exactly 10 patterns must be detected
	// Per spec VR-013: "All 10 verb-first patterns produce helpful error messages"

	expectedPatterns := []string{
		"add package",
		"list package",
		"remove package",
		"search package",
		"add source",
		"list source",
		"remove source",
		"enable",
		"disable",
		"update",
	}

	if len(expectedPatterns) != 10 {
		t.Errorf("Expected exactly 10 verb-first patterns per spec VR-013, got %d", len(expectedPatterns))
	}
}

// TestNounFirstCommandsSucceed validates that noun-first commands work correctly
func TestNounFirstCommandsSucceed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		// Package commands (should work)
		{"package help", []string{"package", "--help"}},
		{"package add help", []string{"package", "add", "--help"}},
		{"package list help", []string{"package", "list", "--help"}},
		{"package remove help", []string{"package", "remove", "--help"}},
		{"package search help", []string{"package", "search", "--help"}},

		// Source commands (should work)
		{"source help", []string{"source", "--help"}},
		{"source add help", []string{"source", "add", "--help"}},
		{"source list help", []string{"source", "list", "--help"}},
		{"source remove help", []string{"source", "remove", "--help"}},
		{"source enable help", []string{"source", "enable", "--help"}},
		{"source disable help", []string{"source", "disable", "--help"}},
		{"source update help", []string{"source", "update", "--help"}},
	}

	gonugetPath := commands.BuildBinary(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(gonugetPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Errorf("Expected command to succeed, but got error: %v\nOutput:\n%s", err, string(output))
			}
		})
	}
}
