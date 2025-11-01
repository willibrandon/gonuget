package commands

import (
	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewSourceListCommand creates the "source list" command matching dotnet nuget
func NewSourceListCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{
		format: "console", // Default format
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured NuGet sources",
		Long: `List all package sources from NuGet.config hierarchy.

This command matches: dotnet nuget list source

Examples:
  gonuget source list
  gonuget source list --format console
  gonuget source list --format json
  gonuget source list --configfile /path/to/NuGet.config`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListSource(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")
	cmd.Flags().StringVar(&opts.format, "format", "console", "The format of the list command output: console or json")

	return cmd
}

func runListSource(console *output.Console, opts *sourceOptions) error {
	cfg, _, err := loadSourceConfig(opts.configFile)
	if err != nil {
		return err
	}

	if cfg.PackageSources == nil || len(cfg.PackageSources.Add) == 0 {
		console.Info("No package sources configured.")
		return nil
	}

	// Match dotnet nuget output format exactly
	console.Info("Registered Sources:")

	for i, source := range cfg.PackageSources.Add {
		// Check if source is in disabledPackageSources list (matches dotnet behavior)
		status := "Enabled"
		if cfg.IsSourceDisabled(source.Key) {
			status = "Disabled"
		}
		console.Info("  %d.  %s [%s]", i+1, source.Key, status)
		if opts.format == "Detailed" {
			console.Info("      %s", source.Value)
		}
	}

	return nil
}
