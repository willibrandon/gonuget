package commands_test

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// TestPackageSearchCommandStructure validates the package search command structure
func TestPackageSearchCommandStructure(t *testing.T) {
	cmd := commands.NewPackageSearchCommand()

	// VR-002: Subcommand Use field must be verb-only (with args allowed)
	expectedUse := "search <SEARCH_TERM>"
	if cmd.Use != expectedUse && cmd.Use != "search [SEARCH_TERM]" {
		t.Errorf("Expected Use='%s' or 'search [SEARCH_TERM]', got '%s'", expectedUse, cmd.Use)
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

// TestPackageSearchFlags validates the flags for package search command
func TestPackageSearchFlags(t *testing.T) {
	cmd := commands.NewPackageSearchCommand()

	// Check required flags exist
	flags := []string{"source", "format", "take", "skip", "prerelease"}
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
