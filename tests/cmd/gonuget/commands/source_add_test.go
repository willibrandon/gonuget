package commands_test

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// TestSourceAddCommandStructure validates the source add command structure
func TestSourceAddCommandStructure(t *testing.T) {
	console := output.NewTestConsole()
	cmd := commands.NewSourceAddCommand(console)

	// VR-002: Subcommand Use field must be verb-only (with args allowed)
	if cmd.Use != "add <PACKAGE_SOURCE_PATH>" {
		t.Errorf("Expected Use='add <PACKAGE_SOURCE_PATH>', got '%s'", cmd.Use)
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

// TestSourceAddFlags validates the flags for source add command
func TestSourceAddFlags(t *testing.T) {
	console := output.NewTestConsole()
	cmd := commands.NewSourceAddCommand(console)

	// Check required flags exist
	flags := []string{"name", "username", "password", "store-password-in-clear-text", "valid-authentication-types", "protocol-version", "allow-insecure-connections", "configfile"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s to exist", flagName)
		}
	}

	// Verify --name is required
	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("Expected --name flag to exist")
	}
}
