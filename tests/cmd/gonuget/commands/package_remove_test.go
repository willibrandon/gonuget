package commands_test

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// TestPackageRemoveCommandStructure validates the package remove command structure
func TestPackageRemoveCommandStructure(t *testing.T) {
	cmd := commands.NewPackageRemoveCommand()

	// VR-002: Subcommand Use field must be verb-only (with args allowed)
	if cmd.Use != "remove <PACKAGE_ID>" {
		t.Errorf("Expected Use='remove <PACKAGE_ID>', got '%s'", cmd.Use)
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

// TestPackageRemoveFlags validates the flags for package remove command
func TestPackageRemoveFlags(t *testing.T) {
	cmd := commands.NewPackageRemoveCommand()

	// Check required flags exist
	flags := []string{"project"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s to exist", flagName)
		}
	}
}

// TestPackageRemoveArgs validates the args requirement
func TestPackageRemoveArgs(t *testing.T) {
	cmd := commands.NewPackageRemoveCommand()

	// Should require exactly 1 arg (PACKAGE_ID)
	if cmd.Args == nil {
		t.Error("Args validator should be set")
	}
}
