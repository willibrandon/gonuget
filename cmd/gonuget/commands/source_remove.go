package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewSourceRemoveCommand creates the "source remove" command matching dotnet nuget
func NewSourceRemoveCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{}

	cmd := &cobra.Command{
		Use:   "remove <NAME>",
		Short: "Remove a NuGet source",
		Long: `Remove a package source from NuGet.config.

This command matches: dotnet nuget remove source

Examples:
  gonuget source remove "MyFeed"
  gonuget source remove "Azure" --configfile /path/to/NuGet.config`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSourceNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
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

	// Remove password from keychain if it exists
	// Ignore errors - password might not be in keychain (could be cleartext or base64)
	_ = deletePasswordFromKeychain(opts.name)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Info("Package source with name '%s' removed successfully.", opts.name)
	return nil
}
