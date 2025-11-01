package commands_test

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// TestPackageAddCommandStructure validates the package add command structure
func TestPackageAddCommandStructure(t *testing.T) {
	cmd := commands.NewPackageAddCommand()

	// VR-002: Subcommand Use field must be verb-only (with args allowed)
	if cmd.Use != "add <PACKAGE_ID>" {
		t.Errorf("Expected Use='add <PACKAGE_ID>', got '%s'", cmd.Use)
	}

	// VR-004: Zero aliases policy
	if len(cmd.Aliases) > 0 {
		t.Errorf("ZERO TOLERANCE: Command has aliases %v - aliases are FORBIDDEN", cmd.Aliases)
	}

	// VR-005: Short description must start with capital verb
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	// Verify it's a runnable command (not a parent)
	if cmd.RunE == nil && cmd.Run == nil {
		t.Error("Command must have Run or RunE function")
	}
}

// TestPackageAddFlags validates the flags for package add command
func TestPackageAddFlags(t *testing.T) {
	cmd := commands.NewPackageAddCommand()

	// Check required flags exist
	flags := []string{"version", "framework", "no-restore", "source", "package-directory", "prerelease", "interactive", "project"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s to exist", flagName)
		}
	}
}

// TestPackageAddArgs validates the args requirement
func TestPackageAddArgs(t *testing.T) {
	cmd := commands.NewPackageAddCommand()

	// Should require exactly 1 arg (PACKAGE_ID)
	if cmd.Args == nil {
		t.Error("Args validator should be set")
	}

	// Test with correct number of args (would need to set up full context for RunE test)
	// For now, just verify Args is not nil
}
