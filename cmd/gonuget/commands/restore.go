package commands

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
	"github.com/willibrandon/gonuget/restore"
)

// NewRestoreCommand creates the restore command.
func NewRestoreCommand(console *output.Console) *cobra.Command {
	opts := &restore.Options{}

	cmd := &cobra.Command{
		Use:   "restore [<PROJECT|SOLUTION>]",
		Short: "Restore NuGet packages",
		Long: `Restores packages based on PackageReference elements in the project file.

Downloads packages to the global package cache and generates project.assets.json.

Examples:
  gonuget restore
  gonuget restore MyApp.csproj
  gonuget restore --packages /custom/packages
  gonuget restore --force
  gonuget restore -v:quiet`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load sources from NuGet.config if not provided via --source flag
			if len(opts.Sources) == 0 {
				// Determine directory to search for config
				var searchDir string
				if len(args) > 0 {
					searchDir = filepath.Dir(args[0])
				} else {
					var err error
					searchDir, err = os.Getwd()
					if err != nil {
						searchDir = "."
					}
				}

				// Load sources from config with fallback to defaults
				sources := config.GetEnabledSourcesOrDefault(searchDir)
				for _, source := range sources {
					opts.Sources = append(opts.Sources, source.Value)
				}
			}

			// CLI just calls library function
			return restore.Run(cmd.Context(), args, opts, console)
		},
	}

	// Flag binding
	cmd.Flags().StringSliceVarP(&opts.Sources, "source", "s", nil, "Package source(s) to use")
	cmd.Flags().StringVar(&opts.PackagesFolder, "packages", "", "Custom global packages folder")
	cmd.Flags().StringVar(&opts.ConfigFile, "configfile", "", "NuGet configuration file")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force re-download even if packages exist")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "Don't use HTTP cache")
	cmd.Flags().BoolVar(&opts.NoDependencies, "no-dependencies", false, "Only restore direct references")
	cmd.Flags().StringVarP(&opts.Verbosity, "verbosity", "v", "minimal", "Verbosity level: q[uiet], m[inimal], n[ormal], d[etailed], or diag[nostic]")

	return cmd
}
