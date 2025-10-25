// cmd/gonuget/cli/app.go
package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gonuget",
	Short: "NuGet package manager CLI",
	Long: `gonuget is a cross-platform NuGet package manager CLI for Go.

Complete documentation is available at https://github.com/willibrandon/gonuget`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help when no command is provided
		_ = cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Handle --version flag
	rootCmd.SetVersionTemplate(GetFullVersion() + "\n")
	rootCmd.Version = GetVersion()

	// Add common flags that will be used by subcommands
	rootCmd.PersistentFlags().StringP("configfile", "", "", "NuGet configuration file to use")
	rootCmd.PersistentFlags().StringP("verbosity", "", "normal", "Display verbosity (quiet, normal, detailed)")
	rootCmd.PersistentFlags().BoolP("non-interactive", "", false, "Do not prompt for user input or confirmations")
}
