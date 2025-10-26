// cmd/gonuget/main.go
package main

import (
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

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.Exit(130) // 128 + SIGINT
	}()

	// Execute CLI
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
