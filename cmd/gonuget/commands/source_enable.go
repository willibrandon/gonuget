package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewEnableCommand creates the "enable source" command matching dotnet nuget
func NewEnableCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{}

	cmd := &cobra.Command{
		Use:   "enable source <name>",
		Short: "Enable a NuGet source.",
		Long: `Enable a disabled package source.

This command matches: dotnet nuget enable source

Examples:
  gonuget enable source "MyFeed"
  gonuget enable source "Azure" --configfile /path/to/NuGet.config`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "source" {
				return fmt.Errorf("unknown command %q for \"enable\"", args[0])
			}
			opts.name = args[1]
			return runEnableSource(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")

	return cmd
}

func runEnableSource(console *output.Console, opts *sourceOptions) error {
	cfg, configPath, err := loadSourceConfig(opts.configFile)
	if err != nil {
		return err
	}

	// Check if source exists
	_, err = findSourceByName(cfg, opts.name)
	if err != nil {
		return err
	}

	// Check if already enabled
	if !cfg.IsSourceDisabled(opts.name) {
		console.Info("Package source '%s' is already enabled.", opts.name)
		return nil
	}

	// Enable the source by removing from disabledPackageSources
	cfg.EnableSource(opts.name)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Info("Package source with name '%s' enabled successfully.", opts.name)
	return nil
}
