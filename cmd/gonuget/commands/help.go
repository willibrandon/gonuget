package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewHelpCommand creates the help command that matches dotnet nuget --help
func NewHelpCommand(console *output.Console, rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Show help information",
		Long:  `Show help information about gonuget or a specific command.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return showGeneralHelp(console, rootCmd)
			}
			return showCommandHelp(rootCmd, args[0])
		},
		// Disable default help flag since we're implementing custom help
		DisableFlagParsing: false,
	}

	return cmd
}

func showGeneralHelp(console *output.Console, rootCmd *cobra.Command) error {
	// Match dotnet nuget help output format
	version := rootCmd.Version
	if version == "" {
		version = "dev"
	}

	console.Println(fmt.Sprintf("NuGet Command Line %s", version))
	console.Println("")
	console.Println("Usage: gonuget [options] [command]")
	console.Println("")
	console.Println("Options:")
	console.Println("  -h|--help  Show help information")
	console.Println("  --version  Show version information")
	console.Println("")
	console.Println("Commands:")

	// Get all commands and sort them
	commands := rootCmd.Commands()

	// Commands to hide from help output (match dotnet nuget behavior)
	hideCommands := map[string]bool{
		"completion": true, // Cobra auto-generated
		"help":       true, // Redundant with --help flag
		"version":    true, // Only a flag in dotnet nuget, not a command
	}

	// Print commands in alphabetical order (like dotnet nuget)
	for _, cmd := range commands {
		if cmd.Hidden || hideCommands[cmd.Name()] {
			continue
		}
		// Format: "  command    Description" with 8-char column width like dotnet
		name := cmd.Name()
		short := cmd.Short
		if short == "" {
			short = cmd.Long
		}
		console.Println(fmt.Sprintf("  %-8s %s", name, short))
	}

	console.Println("")
	console.Println("Use \"gonuget [command] --help\" for more information about a command.")

	return nil
}

func showCommandHelp(rootCmd *cobra.Command, commandName string) error {
	// Find the command
	cmd, _, err := rootCmd.Find([]string{commandName})
	if err != nil || cmd == rootCmd {
		return fmt.Errorf("unknown command: %s\n\nRun 'gonuget --help' for usage", commandName)
	}

	// Show the command's help
	return cmd.Help()
}

// CustomizeRootHelp customizes the root command help template to match dotnet nuget.
func CustomizeRootHelp(rootCmd *cobra.Command) {
	rootCmd.SetHelpTemplate(getRootHelpTemplate())
	rootCmd.SetUsageTemplate(getRootUsageTemplate())
}

func getRootHelpTemplate() string {
	return `{{with .Long}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

func getRootUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

// CustomizeCommandHelp customizes command help to match dotnet nuget format.
func CustomizeCommandHelp(cmd *cobra.Command) {
	// Set custom help function
	originalHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		// Use custom template for subcommands
		if c.HasSubCommands() {
			c.SetUsageTemplate(getSubcommandUsageTemplate())
		}
		originalHelpFunc(c, args)
	})
}

func getSubcommandUsageTemplate() string {
	return `Usage: {{.UseLine}}{{if .HasAvailableSubCommands}}

Options:
  -h|--help  Show help information
{{if .HasAvailableSubCommands}}
Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}{{if not .HasAvailableSubCommands}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{end}}
`
}

// FormatCommandList formats a list of commands for display
func FormatCommandList(commands []*cobra.Command) string {
	var sb strings.Builder

	for _, cmd := range commands {
		if cmd.Hidden {
			continue
		}
		name := cmd.Name()
		short := cmd.Short
		if short == "" {
			short = cmd.Long
		}
		sb.WriteString(fmt.Sprintf("  %-10s %s\n", name, short))
	}

	return sb.String()
}
