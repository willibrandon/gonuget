package commands

import (
	"github.com/spf13/cobra"
)

// packageCmd is the parent package command instance
var packageCmd *cobra.Command

// NewPackageCommand creates the parent "package" command with subcommands
func NewPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Manage package references",
		Long: `Manage NuGet package references in .NET project files.

This command provides operations for adding, listing, removing, and searching
packages. All operations modify or query .NET project files (.csproj, .fsproj, .vbproj).`,
		Example: `  # Add a package
  gonuget package add Newtonsoft.Json

  # List packages in a project
  gonuget package list

  # Remove a package
  gonuget package remove Newtonsoft.Json

  # Search for packages
  gonuget package search Serilog`,
		// Parent commands have no Run function - they are containers only
	}

	// Add persistent flags (inherited by all subcommands)
	cmd.PersistentFlags().StringP("project", "p", "",
		"Path to .NET project file")

	// Store reference for subcommand registration
	packageCmd = cmd

	return cmd
}

// GetPackageCommand returns the package command for registration with root
func GetPackageCommand() *cobra.Command {
	if packageCmd == nil {
		packageCmd = NewPackageCommand()
	}
	return packageCmd
}
