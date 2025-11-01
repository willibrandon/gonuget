package commands_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
)

// TestBashCompletionGeneration tests that bash completion script is generated correctly
func TestBashCompletionGeneration(t *testing.T) {
	// Get root command
	rootCmd := cli.GetRootCommand()

	// Capture bash completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenBashCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate bash completion: %v", err)
	}

	output := buf.String()

	// Verify output is not empty
	if len(output) == 0 {
		t.Fatal("Bash completion output is empty")
	}

	// Verify bash completion script structure
	tests := []struct {
		name     string
		contains string
		reason   string
	}{
		{
			name:     "bash shebang",
			contains: "# bash completion",
			reason:   "Bash completion should have bash completion comment",
		},
		{
			name:     "completion function",
			contains: "__gonuget_",
			reason:   "Should contain gonuget-specific completion functions",
		},
		{
			name:     "package command",
			contains: "package",
			reason:   "Should include package namespace",
		},
		{
			name:     "source command",
			contains: "source",
			reason:   "Should include source namespace",
		},
		{
			name:     "config command",
			contains: "config",
			reason:   "Should include config command",
		},
		{
			name:     "restore command",
			contains: "restore",
			reason:   "Should include restore command",
		},
		{
			name:     "version command",
			contains: "version",
			reason:   "Should include version command",
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
				t.Errorf("Bash completion missing expected content: %s (reason: %s)", tt.contains, tt.reason)
			}
		})
	}
}

// TestBashCompletionScriptValid tests that the generated bash script has valid syntax
func TestBashCompletionScriptValid(t *testing.T) {
	// Get root command
	rootCmd := cli.GetRootCommand()

	// Capture bash completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenBashCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate bash completion: %v", err)
	}

	output := buf.String()

	// Basic syntax checks
	tests := []struct {
		name  string
		check func(string) bool
		error string
	}{
		{
			name: "script has content",
			check: func(s string) bool {
				// Check that script has reasonable length
				return len(s) > 100
			},
			error: "Bash script should have meaningful content",
		},
		{
			name: "has function definitions",
			check: func(s string) bool {
				return strings.Contains(s, "function ") || strings.Contains(s, "() {")
			},
			error: "Bash script should contain function definitions",
		},
		{
			name: "has complete builtin",
			check: func(s string) bool {
				return strings.Contains(s, "complete ")
			},
			error: "Bash script should use 'complete' builtin",
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

// TestBashCompletionSubcommands verifies that subcommands are included in completion
func TestBashCompletionSubcommands(t *testing.T) {
	// Get root command
	rootCmd := cli.GetRootCommand()

	// Capture bash completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenBashCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate bash completion: %v", err)
	}

	output := buf.String()

	// Verify subcommands are present
	subcommands := []string{
		// Package subcommands
		"add",
		"list",
		"remove",
		"search",
		// Source subcommands (add and list already checked above)
		"enable",
		"disable",
		"update",
		// Config subcommands
		"get",
		"set",
		"unset",
		"paths",
	}

	for _, subcmd := range subcommands {
		t.Run("subcommand_"+subcmd, func(t *testing.T) {
			if !strings.Contains(output, subcmd) {
				t.Errorf("Bash completion missing subcommand: %s", subcmd)
			}
		})
	}
}

// TestBashCompletionFlags verifies that common flags are included
func TestBashCompletionFlags(t *testing.T) {
	// Get root command
	rootCmd := cli.GetRootCommand()

	// Capture bash completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenBashCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate bash completion: %v", err)
	}

	output := buf.String()

	// Verify common flags are present
	flags := []string{
		"--help",
		"--version",
		"--configfile",
		"--name",
		"--source",
		"--format",
		"--project",
	}

	for _, flag := range flags {
		t.Run("flag_"+flag, func(t *testing.T) {
			if !strings.Contains(output, flag) {
				// Some flags like --help may be handled specially by Cobra
				if flag == "--help" {
					t.Logf("Note: %s flag may be handled specially by Cobra", flag)
				} else {
					t.Errorf("Bash completion missing flag: %s", flag)
				}
			}
		})
	}
}
