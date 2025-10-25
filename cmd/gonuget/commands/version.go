// cmd/gonuget/commands/version.go
package commands

import (
	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewVersionCommand creates the version command
func NewVersionCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  `Display detailed version information including commit, build date, and builder.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(console)
		},
	}

	return cmd
}

func runVersion(console *output.Console) error {
	console.Println(cli.GetFullVersion())
	return nil
}
