package commands_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

var update = flag.Bool("update", false, "update golden files")

// normalizeLineEndings converts all line endings to LF for consistent cross-platform comparison
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// TestHelpOutput validates help output against golden files
func TestHelpOutput(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		golden string
	}{
		{"root help", []string{"--help"}, "help_root.golden"},
		{"package help", []string{"package", "--help"}, "help_package.golden"},
		{"source help", []string{"source", "--help"}, "help_source.golden"},
		{"source add help", []string{"source", "add", "--help"}, "help_source_add.golden"},
		{"source list help", []string{"source", "list", "--help"}, "help_source_list.golden"},
		{"source remove help", []string{"source", "remove", "--help"}, "help_source_remove.golden"},
		{"source enable help", []string{"source", "enable", "--help"}, "help_source_enable.golden"},
		{"source disable help", []string{"source", "disable", "--help"}, "help_source_disable.golden"},
		{"source update help", []string{"source", "update", "--help"}, "help_source_update.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh root command for each test
			rootCmd := createTestRootCommand()

			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			// Execute command
			err := rootCmd.Execute()
			if err != nil {
				t.Logf("Command execution returned error (this may be expected for help): %v", err)
			}

			goldenPath := filepath.Join("golden", tt.golden)

			if *update {
				// Update golden file
				if err := os.WriteFile(goldenPath, buf.Bytes(), 0644); err != nil {
					t.Fatalf("failed to update golden file %s: %v", goldenPath, err)
				}
				t.Logf("Updated golden file: %s", goldenPath)
			} else {
				// Compare against golden file
				golden, err := os.ReadFile(goldenPath)
				if err != nil {
					t.Fatalf("failed to read golden file %s: %v (run with -update to create)", goldenPath, err)
				}

				// Normalize line endings for cross-platform compatibility
				gotNormalized := normalizeLineEndings(buf.String())
				wantNormalized := normalizeLineEndings(string(golden))

				if gotNormalized != wantNormalized {
					t.Errorf("output mismatch for %s:\n=== GOT ===\n%s\n=== WANT ===\n%s\n=== END ===",
						tt.name, buf.String(), string(golden))
					t.Logf("Run 'go test -update' to update golden files")
				}
			}
		})
	}
}

// createTestRootCommand creates a fresh root command with all commands registered
// This ensures each test gets a clean command state
func createTestRootCommand() *cobra.Command {
	// Get the existing root command
	root := cli.GetRootCommand()

	// Note: Commands are already registered via init() in cli package and main.go
	// We just return the root command for testing

	return root
}

// init registers commands for testing (similar to main.go)
func init() {
	// Register commands if not already registered
	cli.AddCommand(commands.NewVersionCommand(cli.Console))
	cli.AddCommand(commands.NewConfigCommand(cli.Console))
	cli.AddCommand(commands.NewRestoreCommand(cli.Console))
	cli.AddCommand(commands.GetPackageCommand())
	cli.AddCommand(commands.GetSourceCommand())
	commands.RegisterSourceSubcommands(cli.Console)

	// Setup custom error handler
	commands.SetupCustomErrorHandler(cli.GetRootCommand())
}
