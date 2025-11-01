package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCommand creates the completion command for generating shell completion scripts
func NewCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for gonuget.

The completion command generates shell-specific completion scripts that enable
TAB completion for gonuget commands, flags, and arguments.

Examples:
  # Generate bash completion script
  gonuget completion bash > /etc/bash_completion.d/gonuget

  # Generate zsh completion script
  gonuget completion zsh > "${fpath[1]}/_gonuget"

  # Generate PowerShell completion script
  gonuget completion powershell > gonuget.ps1
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, powershell)", shell)
			}
		},
	}

	return cmd
}
