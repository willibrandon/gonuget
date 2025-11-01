package commands

import (
	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// sourceCmd is the parent source command instance
var sourceCmd *cobra.Command

// NewSourceCommand creates the parent "source" command with subcommands
func NewSourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage package sources",
		Long: `Manage NuGet package sources in NuGet.config files.

This command provides operations for adding, listing, removing, enabling,
disabling, and updating package sources. All operations modify NuGet.config
XML files following Microsoft's schema.`,
		Example: `  # Add a package source
  gonuget source add https://api.nuget.org/v3/index.json --name nuget.org

  # List all sources
  gonuget source list

  # Remove a source
  gonuget source remove nuget.org

  # Enable/disable a source
  gonuget source enable nuget.org
  gonuget source disable nuget.org

  # Update a source
  gonuget source update nuget.org --source https://new-url.org/v3/index.json`,
		// Parent commands have no Run function - they are containers only
	}

	// Add persistent flags (inherited by all subcommands)
	cmd.PersistentFlags().String("configfile", "",
		"Path to NuGet.config file")

	// Store reference for subcommand registration
	sourceCmd = cmd

	return cmd
}

// GetSourceCommand returns the source command for registration with root
func GetSourceCommand() *cobra.Command {
	if sourceCmd == nil {
		sourceCmd = NewSourceCommand()
	}
	return sourceCmd
}

// RegisterSourceSubcommands registers all source subcommands with the source parent
func RegisterSourceSubcommands(console *output.Console) {
	sourceCmd := GetSourceCommand()
	sourceCmd.AddCommand(NewSourceAddCommand(console))
	sourceCmd.AddCommand(NewSourceListCommand(console))
	sourceCmd.AddCommand(NewSourceRemoveCommand(console))
	sourceCmd.AddCommand(NewSourceEnableCommand(console))
	sourceCmd.AddCommand(NewSourceDisableCommand(console))
	sourceCmd.AddCommand(NewSourceUpdateCommand(console))
}
