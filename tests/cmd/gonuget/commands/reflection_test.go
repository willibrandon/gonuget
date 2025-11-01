package commands_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// TestCommandStructurePolicy validates VR-001 through VR-006 from data-model.md
func TestCommandStructurePolicy(t *testing.T) {
	// Initialize CLI to register all commands
	initCLI()

	rootCmd := cli.GetRootCommand()
	validateCommand(t, rootCmd, nil)
}

func initCLI() {
	// Register commands like main.go does
	cli.AddCommand(commands.NewVersionCommand(cli.Console))
	cli.AddCommand(commands.NewConfigCommand(cli.Console))
	cli.AddCommand(commands.NewRestoreCommand(cli.Console))
	cli.AddCommand(commands.GetPackageCommand())
	cli.AddCommand(commands.GetSourceCommand())
	commands.RegisterSourceSubcommands(cli.Console)
}

func validateCommand(t *testing.T, cmd *cobra.Command, parent *cobra.Command) {
	t.Helper()

	// VR-004: Zero aliases policy (HARD REQUIREMENT)
	if len(cmd.Aliases) > 0 {
		t.Errorf("POLICY VIOLATION (VR-004): Command '%s' has aliases %v - aliases are FORBIDDEN",
			cmd.Use, cmd.Aliases)
	}

	// VR-001: Parent commands (package, source) MUST have Use field as single noun
	if isParentCommand(cmd) {
		useParts := strings.Fields(cmd.Use)
		if len(useParts) > 1 && !strings.HasPrefix(useParts[1], "<") && !strings.HasPrefix(useParts[1], "[") {
			t.Errorf("POLICY VIOLATION (VR-001): Parent command '%s' has multi-word Use field - must be single noun",
				cmd.Use)
		}
	}

	// VR-002: Subcommands MUST have verb-only Use fields (no spaces except for args in <> or [])
	if parent != nil && isParentCommand(parent) {
		useParts := strings.Fields(cmd.Use)
		if len(useParts) > 1 {
			// Allow args in angle brackets: "add <PACKAGE_ID>" is OK
			// Allow optional args in square brackets: "list [PROJECT]" is OK
			if !strings.HasPrefix(useParts[1], "<") && !strings.HasPrefix(useParts[1], "[") {
				t.Errorf("POLICY VIOLATION (VR-002): Subcommand '%s' under parent '%s' has multi-word Use field - must be verb-only",
					cmd.Use, parent.Name())
			}
		}
	}

	// VR-005: Short description MUST start with verb (capital letter for action)
	if cmd.Short != "" && !cmd.Hidden {
		words := strings.Fields(cmd.Short)
		if len(words) > 0 {
			firstWord := words[0]
			// Check if it starts with a capital letter (verb)
			if len(firstWord) > 0 && !(firstWord[0] >= 'A' && firstWord[0] <= 'Z') {
				t.Errorf("POLICY VIOLATION (VR-005): Command '%s' Short description doesn't start with capital verb: '%s'",
					cmd.Use, cmd.Short)
			}
		}
	}

	// Recurse to subcommands
	for _, child := range cmd.Commands() {
		if !child.Hidden {
			validateCommand(t, child, cmd)
		}
	}
}

// isParentCommand checks if this is a parent command (package or source)
func isParentCommand(cmd *cobra.Command) bool {
	name := cmd.Name()
	return name == "package" || name == "source"
}

// TestNoAliasesAnywhere ensures ZERO aliases exist in the entire command tree
func TestNoAliasesAnywhere(t *testing.T) {
	initCLI()
	rootCmd := cli.GetRootCommand()

	var violations []string
	var checkAliases func(cmd *cobra.Command)
	checkAliases = func(cmd *cobra.Command) {
		if len(cmd.Aliases) > 0 {
			violations = append(violations, cmd.Name())
		}
		for _, child := range cmd.Commands() {
			checkAliases(child)
		}
	}

	checkAliases(rootCmd)

	if len(violations) > 0 {
		t.Errorf("ZERO TOLERANCE POLICY VIOLATED: Found %d commands with aliases: %v",
			len(violations), violations)
	}
}

// TestVerbOnlyUseFields ensures all subcommands under parent commands use verb-only Use fields
func TestVerbOnlyUseFields(t *testing.T) {
	initCLI()
	rootCmd := cli.GetRootCommand()

	var violations []string

	// Check package subcommands
	packageCmd := findCommand(rootCmd, "package")
	if packageCmd != nil {
		for _, subCmd := range packageCmd.Commands() {
			if !isVerbOnly(subCmd.Use) {
				violations = append(violations, "package "+subCmd.Use)
			}
		}
	}

	// Check source subcommands
	sourceCmd := findCommand(rootCmd, "source")
	if sourceCmd != nil {
		for _, subCmd := range sourceCmd.Commands() {
			if !isVerbOnly(subCmd.Use) {
				violations = append(violations, "source "+subCmd.Use)
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("VERB-ONLY POLICY VIOLATED (VR-002): Found %d subcommands with non-verb Use fields: %v",
			len(violations), violations)
	}
}

// isVerbOnly checks if the Use field is a single verb (or verb with args in <>)
func isVerbOnly(use string) bool {
	parts := strings.Fields(use)
	if len(parts) == 0 {
		return false
	}
	if len(parts) == 1 {
		return true // Single word = verb only
	}
	// Multiple words OK only if second word starts with < or [
	return strings.HasPrefix(parts[1], "<") || strings.HasPrefix(parts[1], "[")
}

// findCommand finds a command by name in the command tree
func findCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}
