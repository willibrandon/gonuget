package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewRemoveCommand creates the "remove source" command matching dotnet nuget
func NewRemoveCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{}

	cmd := &cobra.Command{
		Use:   "remove source <name>",
		Short: "Remove a NuGet source.",
		Long: `Remove a package source from NuGet.config.

This command matches: dotnet nuget remove source

Examples:
  gonuget remove source "MyFeed"
  gonuget remove source "Azure" --configfile /path/to/NuGet.config`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "source" {
				return fmt.Errorf("unknown command %q for \"remove\"", args[0])
			}
			opts.name = args[1]
			return runRemoveSource(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")

	return cmd
}

func runRemoveSource(console *output.Console, opts *sourceOptions) error {
	cfg, configPath, err := loadSourceConfig(opts.configFile)
	if err != nil {
		return err
	}

	// Check if source exists
	if !validateSourceExists(cfg, opts.name) {
		return fmt.Errorf("package source with name '%s' not found", opts.name)
	}

	// Remove the source
	if !cfg.RemovePackageSource(opts.name) {
		return fmt.Errorf("failed to remove source: %s", opts.name)
	}

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Info("Package source with name '%s' removed successfully.", opts.name)
	return nil
}
