// cmd/gonuget/cli/app.go
package cli

import (
	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
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

// Console is the global console for CLI commands
var Console *output.Console

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Initialize console
	Console = output.DefaultConsole()

	// Add common flags that will be used by subcommands
	rootCmd.PersistentFlags().StringP("configfile", "", "", "NuGet configuration file to use")
	rootCmd.PersistentFlags().StringP("verbosity", "", "normal", "Display verbosity (quiet, normal, detailed)")
	rootCmd.PersistentFlags().BoolP("non-interactive", "", false, "Do not prompt for user input or confirmations")
}

// SetupVersion configures version information after variables are set
func SetupVersion() {
	rootCmd.SetVersionTemplate(GetFullVersion() + "\n")
	rootCmd.Version = GetVersion()
}

// AddCommand adds a command to the root command
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
