package commands

import (
	"github.com/spf13/cobra"
)

// completeSourceNames provides dynamic completion for source names from NuGet.config
func completeSourceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load config to get source names
	cfg, _, err := loadSourceConfig("")
	if err != nil || cfg.PackageSources == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var sourceNames []string
	for _, source := range cfg.PackageSources.Add {
		sourceNames = append(sourceNames, source.Key)
	}

	return sourceNames, cobra.ShellCompDirectiveNoFileComp
}

// Future completion helpers can be added here:
// - completeProjectFiles: for --project flag (.csproj, .fsproj, .vbproj)
// - completeConfigFiles: for --configfile flag (NuGet.config)
