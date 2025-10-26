package commands

import (
	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewAddCommand creates the parent "add" command with subcommands for "source" and "package"
func NewAddCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a NuGet source or package reference",
		Long: `Add a NuGet source or package reference.

Subcommands:
  source  - Add a package source to NuGet.config
  package - Add a package reference to a project file

Examples:
  gonuget add source https://api.nuget.org/v3/index.json --name "MyFeed"
  gonuget add package Newtonsoft.Json --version 13.0.3`,
	}

	// Add subcommands
	cmd.AddCommand(NewAddSourceCommand(console))
	cmd.AddCommand(NewAddPackageCommand())

	return cmd
}
