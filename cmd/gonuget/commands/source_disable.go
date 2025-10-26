package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewDisableCommand creates the "disable source" command matching dotnet nuget
func NewDisableCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{}

	cmd := &cobra.Command{
		Use:   "disable source <name>",
		Short: "Disable a NuGet source.",
		Long: `Disable an enabled package source.

This command matches: dotnet nuget disable source

Examples:
  gonuget disable source "MyFeed"
  gonuget disable source "Azure" --configfile /path/to/NuGet.config`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "source" {
				return fmt.Errorf("unknown command %q for \"disable\"", args[0])
			}
			opts.name = args[1]
			return runDisableSource(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")

	return cmd
}

func runDisableSource(console *output.Console, opts *sourceOptions) error {
	cfg, configPath, err := loadSourceConfig(opts.configFile)
	if err != nil {
		return err
	}

	// Check if source exists
	_, err = findSourceByName(cfg, opts.name)
	if err != nil {
		return err
	}

	// Check if already disabled
	if cfg.IsSourceDisabled(opts.name) {
		console.Info("Package source '%s' is already disabled.", opts.name)
		return nil
	}

	// Disable the source using disabledPackageSources section
	cfg.DisableSource(opts.name)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Info("Package source with name '%s' disabled successfully.", opts.name)
	return nil
}
