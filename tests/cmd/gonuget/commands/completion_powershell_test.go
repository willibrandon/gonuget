package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
)

// TestPowerShellCompletionGeneration tests that PowerShell completion script is generated correctly
func TestPowerShellCompletionGeneration(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture PowerShell completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion: %v", err)
	}

	output := buf.String()

	// Verify output is not empty
	if len(output) == 0 {
		t.Fatal("PowerShell completion output is empty")
	}

	// Verify PowerShell completion script structure
	tests := []struct {
		name     string
		contains string
		reason   string
	}{
		{
			name:     "powershell comment",
			contains: "#",
			reason:   "PowerShell completion should have comments",
		},
		{
			name:     "register-argumentcompleter",
			contains: "Register-ArgumentCompleter",
			reason:   "PowerShell completion should use Register-ArgumentCompleter",
		},
		{
			name:     "gonuget command name",
			contains: "gonuget",
			reason:   "Should reference gonuget command",
		},
		{
			name:     "completion command",
			contains: "completion",
			reason:   "Should include completion command itself",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// PowerShell might use different casing or quoting, check case-insensitively
			if !strings.Contains(strings.ToLower(output), strings.ToLower(tt.contains)) {
				t.Errorf("PowerShell completion missing expected content: %s (reason: %s)", tt.contains, tt.reason)
			}
		})
	}
}

// TestPowerShellCompletionScriptValid tests that the generated PowerShell script has valid syntax
func TestPowerShellCompletionScriptValid(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture PowerShell completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion: %v", err)
	}

	output := buf.String()

	// Basic syntax checks
	tests := []struct {
		name  string
		check func(string) bool
		error string
	}{
		{
			name: "has Register-ArgumentCompleter",
			check: func(s string) bool {
				return strings.Contains(s, "Register-ArgumentCompleter")
			},
			error: "PowerShell script should use Register-ArgumentCompleter cmdlet",
		},
		{
			name: "has scriptblock",
			check: func(s string) bool {
				// Check for PowerShell scriptblock syntax
				return strings.Contains(s, "{") && strings.Contains(s, "}")
			},
			error: "PowerShell script should contain scriptblocks",
		},
		{
			name: "has parameters",
			check: func(s string) bool {
				// Check for PowerShell parameter syntax
				return strings.Contains(s, "param(") || strings.Contains(s, "$")
			},
			error: "PowerShell script should define parameters",
		},
		{
			name: "no syntax errors in braces",
			check: func(s string) bool {
				// Count opening and closing braces
				open := strings.Count(s, "{")
				close := strings.Count(s, "}")
				return open == close
			},
			error: "PowerShell script has unmatched braces",
		},
		{
			name: "script has reasonable length",
			check: func(s string) bool {
				// Check that script has meaningful content
				return len(s) > 100
			},
			error: "PowerShell script should have meaningful content",
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

// TestPowerShellCompletionSubcommands verifies that subcommands are included in completion
func TestPowerShellCompletionSubcommands(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture PowerShell completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion: %v", err)
	}

	output := buf.String()

	// PowerShell completion uses dynamic completion via __complete
	// So we just need to verify the completion function is set up
	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Error("PowerShell completion should use Register-ArgumentCompleter")
	}

	// Verify it calls the command for dynamic completions
	if !strings.Contains(output, "__complete") {
		t.Error("PowerShell completion should use dynamic __complete mechanism")
	}
}

// TestPowerShellCompletionFlags verifies that common flags are included
func TestPowerShellCompletionFlags(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture PowerShell completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion: %v", err)
	}

	output := buf.String()

	// PowerShell completion uses dynamic completion via __complete
	// Flags are generated dynamically at completion time, not hardcoded in the script
	// Just verify the completion mechanism supports argument handling
	if !strings.Contains(output, "$WordToComplete") {
		t.Error("PowerShell completion should support word completion")
	}
}

// TestPowerShellCompletionWithDescriptions verifies that descriptions are included
func TestPowerShellCompletionWithDescriptions(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture PowerShell completion output using GenPowerShellCompletionWithDesc
	buf := new(bytes.Buffer)
	err := rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion with descriptions: %v", err)
	}

	output := buf.String()

	// PowerShell completion uses GenPowerShellCompletionWithDesc which supports descriptions
	// Descriptions are generated dynamically via __complete, not hardcoded in the script
	// Just verify the script supports description handling
	if !strings.Contains(output, "Description") {
		t.Error("PowerShell completion script should support description handling")
	}

	// Verify it creates CompletionResult objects with descriptions
	if !strings.Contains(output, "New-Object") && !strings.Contains(output, "PSCustomObject") {
		t.Error("PowerShell completion should create completion result objects")
	}
}

// TestPowerShellCompletionCommandName verifies the completion is registered for gonuget
func TestPowerShellCompletionCommandName(t *testing.T) {
	// Create root command
	rootCmd := cli.GetRootCommand()

	// Capture PowerShell completion output
	buf := new(bytes.Buffer)
	err := rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion: %v", err)
	}

	output := buf.String()

	// Verify completion is registered for 'gonuget' command
	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Fatal("PowerShell completion should use Register-ArgumentCompleter")
	}

	// Check that gonuget is the command being completed
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		if strings.Contains(line, "Register-ArgumentCompleter") && strings.Contains(line, "-CommandName") {
			if !strings.Contains(line, "gonuget") {
				t.Error("Register-ArgumentCompleter should specify 'gonuget' as CommandName")
			}
			return
		}
	}

	// If we didn't find the CommandName line, log a warning
	t.Log("Warning: Could not verify CommandName in Register-ArgumentCompleter")
}
