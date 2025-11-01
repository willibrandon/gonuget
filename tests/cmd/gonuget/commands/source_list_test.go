package commands_test

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// TestSourceListCommandStructure validates the source list command structure
func TestSourceListCommandStructure(t *testing.T) {
	console := output.NewTestConsole()
	cmd := commands.NewSourceListCommand(console)

	// VR-002: Subcommand Use field must be verb-only
	if cmd.Use != "list" {
		t.Errorf("Expected Use='list', got '%s'", cmd.Use)
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

// TestSourceListFlags validates the flags for source list command
func TestSourceListFlags(t *testing.T) {
	console := output.NewTestConsole()
	cmd := commands.NewSourceListCommand(console)

	// Check required flags exist
	flags := []string{"format", "configfile"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s to exist", flagName)
		}
	}

	// VR-010: --format flag must accept "console" and "json"
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag != nil {
		// Default should be "console"
		if formatFlag.DefValue != "console" {
			t.Errorf("Expected --format default='console', got '%s'", formatFlag.DefValue)
		}
	}
}
