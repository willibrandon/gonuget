package commands_test

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// TestSourceEnableCommandStructure validates the source enable command structure
func TestSourceEnableCommandStructure(t *testing.T) {
	console := output.NewTestConsole()
	cmd := commands.NewSourceEnableCommand(console)

	// VR-002: Subcommand Use field must be verb-only (with args allowed)
	if cmd.Use != "enable <NAME>" {
		t.Errorf("Expected Use='enable <NAME>', got '%s'", cmd.Use)
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

// TestSourceEnableFlags validates the flags for source enable command
func TestSourceEnableFlags(t *testing.T) {
	console := output.NewTestConsole()
	cmd := commands.NewSourceEnableCommand(console)

	// Check required flags exist
	flags := []string{"configfile"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s to exist", flagName)
		}
	}
}
