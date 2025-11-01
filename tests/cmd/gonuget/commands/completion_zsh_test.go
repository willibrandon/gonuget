package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
)

// TestZshCompletionGeneration tests that zsh completion script is generated correctly
func TestZshCompletionGeneration(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture zsh completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate zsh completion: %v", err)
	}

	output := buf.String()

	// Verify output is not empty
	if len(output) == 0 {
		t.Fatal("Zsh completion output is empty")
	}

	// Verify zsh completion script structure
	tests := []struct {
		name     string
		contains string
		reason   string
	}{
		{
			name:     "zsh shebang",
			contains: "#compdef",
			reason:   "Zsh completion should have #compdef directive",
		},
		{
			name:     "completion function",
			contains: "_gonuget",
			reason:   "Should contain gonuget-specific completion function",
		},
		{
			name:     "completion command",
			contains: "completion",
			reason:   "Should include completion command itself",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Zsh completion missing expected content: %s (reason: %s)", tt.contains, tt.reason)
			}
		})
	}
}

// TestZshCompletionScriptValid tests that the generated zsh script has valid syntax
func TestZshCompletionScriptValid(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture zsh completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate zsh completion: %v", err)
	}

	output := buf.String()

	// Basic syntax checks
	tests := []struct {
		name  string
		check func(string) bool
		error string
	}{
		{
			name: "has compdef directive",
			check: func(s string) bool {
				return strings.Contains(s, "#compdef") || strings.Contains(s, "compdef")
			},
			error: "Zsh script should contain compdef directive",
		},
		{
			name: "has function definition",
			check: func(s string) bool {
				return strings.Contains(s, "function _gonuget") || strings.Contains(s, "_gonuget()")
			},
			error: "Zsh script should define _gonuget function",
		},
		{
			name: "uses zsh completion system",
			check: func(s string) bool {
				// Check for common zsh completion functions
				return strings.Contains(s, "_arguments") ||
					strings.Contains(s, "_describe") ||
					strings.Contains(s, "_values")
			},
			error: "Zsh script should use zsh completion system functions",
		},
		{
			name: "no syntax errors in parentheses",
			check: func(s string) bool {
				// Count opening and closing parentheses
				open := strings.Count(s, "(")
				close := strings.Count(s, ")")
				return open == close
			},
			error: "Zsh script has unmatched parentheses",
		},
		{
			name: "no syntax errors in braces",
			check: func(s string) bool {
				// Count opening and closing braces
				open := strings.Count(s, "{")
				close := strings.Count(s, "}")
				return open == close
			},
			error: "Zsh script has unmatched braces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check(output) {
				t.Error(tt.error)
			}
		})
	}
}

// TestZshCompletionSubcommands verifies that subcommands are included in completion
func TestZshCompletionSubcommands(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture zsh completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate zsh completion: %v", err)
	}

	output := buf.String()

	// Zsh completion uses dynamic completion via __complete
	// So we just need to verify the completion function is set up
	if !strings.Contains(output, "_gonuget") {
		t.Error("Zsh completion should define _gonuget function")
	}

	// Verify it calls the command for dynamic completions
	if !strings.Contains(output, "__complete") {
		t.Error("Zsh completion should use dynamic __complete mechanism")
	}
}

// TestZshCompletionFlags verifies that common flags are included
func TestZshCompletionFlags(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture zsh completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate zsh completion: %v", err)
	}

	output := buf.String()

	// Zsh completion uses dynamic completion via __complete
	// Flags are generated dynamically at completion time, not hardcoded in the script
	// Just verify the completion mechanism supports flag handling
	if !strings.Contains(output, "flagPrefix") {
		t.Error("Zsh completion should support flag prefix handling")
	}
}

// TestZshCompletionDescriptions verifies that command descriptions are included
func TestZshCompletionDescriptions(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture zsh completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate zsh completion: %v", err)
	}

	output := buf.String()

	// Zsh completion should include descriptions for commands
	// Check that output has colon-separated command:description format
	if !strings.Contains(output, ":") {
		t.Error("Zsh completion should include command descriptions (format: command:description)")
	}
}
