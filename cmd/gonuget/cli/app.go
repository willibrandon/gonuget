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

	// Disable Cobra's built-in help command (dotnet nuget doesn't have a help command)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Set custom help function to match dotnet nuget --help format
	rootCmd.SetHelpFunc(customHelpFunc)
}

// customHelpFunc provides custom help output matching dotnet nuget --help format
// Only applies custom formatting to the root command; subcommands use Cobra's default help
func customHelpFunc(cmd *cobra.Command, args []string) {
	// Only use custom help for root command; let subcommands use Cobra's default
	if cmd != cmd.Root() {
		// Use Cobra's default template-based help for subcommands
		usage := cmd.Long
		if usage == "" {
			usage = cmd.Short
		}
		if usage != "" {
			Console.Println(usage)
			Console.Println("")
		}
		Console.Print(cmd.UsageString())
		return
	}

	version := cmd.Root().Version
	if version == "" {
		version = "dev"
	}

	Console.Println("NuGet Command Line " + version)
	Console.Println("")
	Console.Println("Usage: gonuget [options] [command]")
	Console.Println("")
	Console.Println("Options:")
	Console.Println("  -h|--help  Show help information")
	Console.Println("  --version  Show version information")
	Console.Println("")
	Console.Println("Commands:")

	// Commands to hide from help output (match dotnet nuget behavior)
	hideCommands := map[string]bool{
		"completion": true, // Cobra auto-generated
		"version":    true, // Only a flag in dotnet nuget, not a command
	}

	// Print commands in alphabetical order (like dotnet nuget)
	for _, subCmd := range cmd.Root().Commands() {
		if subCmd.Hidden || hideCommands[subCmd.Name()] {
			continue
		}
		name := subCmd.Name()
		short := subCmd.Short
		if short == "" {
			short = subCmd.Long
		}
		Console.Println("  " + padRight(name, 8) + " " + short)
	}

	Console.Println("")
	Console.Println("Use \"gonuget [command] --help\" for more information about a command.")
}

// padRight pads a string to the right with spaces
func padRight(s string, length int) string {
	for len(s) < length {
		s += " "
	}
	return s
}

// GetRootCommand returns the root command for use by help command
func GetRootCommand() *cobra.Command {
	return rootCmd
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
