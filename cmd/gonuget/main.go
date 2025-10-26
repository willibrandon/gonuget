// cmd/gonuget/main.go
package main

import (
	"fmt"
	"os"
	"os/signal"
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

func main() {
	// Set version info
	cli.Version = version
	cli.Commit = commit
	cli.Date = date
	cli.BuiltBy = builtBy

	// Setup version after variables are set
	cli.SetupVersion()

	// Register commands
	cli.AddCommand(commands.NewVersionCommand(cli.Console))
	cli.AddCommand(commands.NewConfigCommand(cli.Console))

	// Register source management commands
	cli.AddCommand(commands.NewListCommand(cli.Console))
	cli.AddCommand(commands.NewAddCommand(cli.Console))
	cli.AddCommand(commands.NewRemoveCommand(cli.Console))
	cli.AddCommand(commands.NewEnableCommand(cli.Console))
	cli.AddCommand(commands.NewDisableCommand(cli.Console))
	cli.AddCommand(commands.NewUpdateCommand(cli.Console))

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
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
