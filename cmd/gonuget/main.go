// cmd/gonuget/main.go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// Version information (set via ldflags during build)
var (
	version = "0.0.0-dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

// preprocessArgs converts dotnet-style colon syntax to Cobra-compatible equals syntax
// Converts: -v:quiet  -> -v=quiet
// Converts: -v:d      -> -v=d
func preprocessArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		// Check if arg matches -X:VALUE pattern (single letter flag with colon)
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg, ":") {
			// Convert -v:quiet to -v=quiet
			result = append(result, strings.Replace(arg, ":", "=", 1))
		} else {
			result = append(result, arg)
		}
	}
	return result
}

func main() {
	// Preprocess arguments to support dotnet-style colon syntax (e.g., -v:quiet)
	os.Args = preprocessArgs(os.Args)

	// Set version info
	cli.Version = version
	cli.Commit = commit
	cli.Date = date
	cli.BuiltBy = builtBy

	// Setup version after variables are set
	cli.SetupVersion()

	// Setup custom error handler for verb-first pattern detection
	commands.SetupCustomErrorHandler(cli.GetRootCommand())

	// Register top-level commands (exceptions to noun-first hierarchy)
	cli.AddCommand(commands.NewVersionCommand(cli.Console))
	cli.AddCommand(commands.NewConfigCommand(cli.Console))
	cli.AddCommand(commands.NewRestoreCommand(cli.Console))

	// Register noun-first parent commands with subcommands
	// Package namespace: gonuget package add|list|remove|search
	cli.AddCommand(commands.GetPackageCommand())

	// Source namespace: gonuget source add|list|remove|enable|disable|update
	cli.AddCommand(commands.GetSourceCommand())
	commands.RegisterSourceSubcommands(cli.Console)

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.Exit(130) // 128 + SIGINT
	}()

	// Execute CLI
	if err := cli.Execute(); err != nil {
		// Print error to stderr since SilenceErrors is true in rootCmd
		// Use os.Stderr directly to ensure error goes to stderr for interop testing
		// Don't print empty errors (used when NuGet errors are already formatted)
		if err.Error() != "" {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
