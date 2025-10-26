package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewListCommand creates the "list source" command matching dotnet nuget
func NewListCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{
		format: "Detailed", // Default matches dotnet nuget
	}

	cmd := &cobra.Command{
		Use:   "list source",
		Short: "List configured NuGet sources.",
		Long: `List all package sources from NuGet.config hierarchy.

This command matches: dotnet nuget list source

Examples:
  gonuget list source
  gonuget list source --format Detailed
  gonuget list source --format Short
  gonuget list source --configfile /path/to/NuGet.config`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "source" {
				return fmt.Errorf("unknown command %q for \"list\"", args[0])
			}
			return runListSource(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")
	cmd.Flags().StringVar(&opts.format, "format", "Detailed", "The format of the list command output: `Detailed` (the default) and `Short`.")

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
