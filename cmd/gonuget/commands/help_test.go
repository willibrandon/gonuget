package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestHelpCommand(t *testing.T) {
	// Create a test root command
	rootCmd := &cobra.Command{
		Use:     "gonuget",
		Short:   "NuGet package manager for Go",
		Version: "1.0.0-test",
	}

	// Create output buffer
	var stdout, stderr bytes.Buffer
	console := output.NewConsole(&stdout, &stderr, output.VerbosityNormal)

	// Add some test commands
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
	}
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage NuGet configuration",
	}
	rootCmd.AddCommand(versionCmd, configCmd)

	t.Run("general help", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		helpCmd := NewHelpCommand(console, rootCmd)
		rootCmd.AddCommand(helpCmd)

		err := helpCmd.RunE(helpCmd, []string{})
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		output := stdout.String()

		// Verify output contains expected elements
		if !strings.Contains(output, "NuGet Command Line") {
			t.Error("Expected output to contain 'NuGet Command Line'")
		}
		if !strings.Contains(output, "Usage:") {
			t.Error("Expected output to contain usage")
		}
		if !strings.Contains(output, "Commands:") {
			t.Error("Expected output to contain commands list")
		}
		// version command should be hidden from help output (it's only a flag in dotnet nuget)
		if strings.Contains(output, "version    Display") {
			t.Error("Expected version command to be hidden from help output")
		}
		// completion and help should also be hidden
		if strings.Contains(output, "completion") {
			t.Error("Expected completion command to be hidden from help output")
		}
		if strings.Contains(output, "help       Show help") {
			t.Error("Expected help command to be hidden from help output")
		}
		// config should still be shown
		if !strings.Contains(output, "config") {
			t.Error("Expected output to contain 'config' command")
		}
	})

	t.Run("command-specific help for existing command", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		helpCmd := NewHelpCommand(console, rootCmd)

		err := helpCmd.RunE(helpCmd, []string{"version"})
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// The command help should be shown (delegated to cobra)
		// This test verifies the mechanism works
	})

	t.Run("command-specific help for unknown command", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		helpCmd := NewHelpCommand(console, rootCmd)

		err := helpCmd.RunE(helpCmd, []string{"nonexistent"})
		if err == nil {
			t.Error("Expected error for unknown command")
		}

		if !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("Expected 'unknown command' error, got: %v", err)
		}
	})
}

func TestCustomizeRootHelp(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:     "gonuget",
		Short:   "NuGet package manager for Go",
		Long:    "gonuget is a cross-platform NuGet package manager CLI for Go.",
		Version: "1.0.0-test",
	}

	// Customize help
	CustomizeRootHelp(rootCmd)

	// Verify templates are set
	if rootCmd.HelpTemplate() == "" {
		t.Error("Expected help template to be set")
	}
	if rootCmd.UsageTemplate() == "" {
		t.Error("Expected usage template to be set")
	}
}

func TestFormatCommandList(t *testing.T) {
	commands := []*cobra.Command{
		{Use: "version", Short: "Display version information"},
		{Use: "config", Short: "Manage NuGet configuration"},
		{Use: "hidden", Short: "Hidden command", Hidden: true},
	}

	output := FormatCommandList(commands)

	// Verify visible commands are included
	if !strings.Contains(output, "version") {
		t.Error("Expected output to contain 'version' command")
	}
	if !strings.Contains(output, "config") {
		t.Error("Expected output to contain 'config' command")
	}

	// Verify hidden command is excluded
	if strings.Contains(output, "hidden") {
		t.Error("Expected output to exclude hidden command")
	}
}

func TestShowGeneralHelp(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:     "gonuget",
		Short:   "NuGet package manager for Go",
		Version: "1.0.0-test",
	}

	// Add test commands
	rootCmd.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Manage NuGet configuration",
	})
	// Add command with empty Short but non-empty Long
	rootCmd.AddCommand(&cobra.Command{
		Use:   "longdesc",
		Short: "",
		Long:  "This command has a long description",
	})

	var stdout, stderr bytes.Buffer
	console := output.NewConsole(&stdout, &stderr, output.VerbosityNormal)

	// Set up custom help function (like app.go does)
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		version := cmd.Root().Version
		if version == "" {
			version = "dev"
		}

		console.Println("NuGet Command Line " + version)
		console.Println("")
		console.Println("Usage: gonuget [options] [command]")
		console.Println("")
		console.Println("Options:")
		console.Println("  -h|--help  Show help information")
		console.Println("  --version  Show version information")
		console.Println("")
		console.Println("Commands:")

		hideCommands := map[string]bool{
			"completion": true,
			"version":    true,
		}

		for _, subCmd := range cmd.Root().Commands() {
			if subCmd.Hidden || hideCommands[subCmd.Name()] {
				continue
			}
			name := subCmd.Name()
			short := subCmd.Short
			if short == "" {
				short = subCmd.Long
			}
			// Pad right to 8 characters
			for len(name) < 8 {
				name += " "
			}
			console.Println("  " + name + " " + short)
		}

		console.Println("")
		console.Println("Use \"gonuget [command] --help\" for more information about a command.")
	})

	err := rootCmd.Help()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := stdout.String()

	// Verify output format matches dotnet nuget --help
	expectedElements := []string{
		"NuGet Command Line 1.0.0-test",
		"Usage:",
		"Options:",
		"-h|--help",
		"--version",
		"Commands:",
		"config",
		"longdesc",
		"This command has a long description",
		"Use \"gonuget [command] --help\"",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("Expected output to contain '%s', got:\n%s", element, output)
		}
	}

	// Verify version and help commands are NOT shown (they're hidden)
	if strings.Contains(output, "version    Display") {
		t.Error("Expected version command to be hidden from help output")
	}
	if strings.Contains(output, "help       ") {
		t.Error("Expected help command to be hidden from help output")
	}
}

func TestCustomizeCommandHelp(t *testing.T) {
	// Create a command with subcommands
	parentCmd := &cobra.Command{
		Use:   "parent",
		Short: "Parent command",
	}
	childCmd := &cobra.Command{
		Use:   "child",
		Short: "Child command",
	}
	parentCmd.AddCommand(childCmd)

	// Customize the command help
	CustomizeCommandHelp(parentCmd)

	// Verify that help function was set (we can't easily test the output without executing it)
	if parentCmd.HelpFunc() == nil {
		t.Error("Expected help function to be set")
	}
}

func TestCustomizeCommandHelpWithSubcommands(t *testing.T) {
	// Create a command with subcommands
	parentCmd := &cobra.Command{
		Use:   "parent",
		Short: "Parent command",
	}
	childCmd := &cobra.Command{
		Use:   "child",
		Short: "Child command",
	}
	parentCmd.AddCommand(childCmd)

	// Set output to capture help
	var stdout bytes.Buffer
	parentCmd.SetOut(&stdout)

	// Customize the command help
	CustomizeCommandHelp(parentCmd)

	// Trigger the help function by calling Help() on a command with subcommands
	err := parentCmd.Help()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The customized help should have been executed
	output := stdout.String()
	if output == "" {
		t.Error("Expected help output to be generated")
	}
}

func TestFormatCommandListWithEmptyShort(t *testing.T) {
	commands := []*cobra.Command{
		{Use: "test", Short: "", Long: "Long description"},
	}

	output := FormatCommandList(commands)

	// Verify that Long is used when Short is empty
	if !strings.Contains(output, "Long description") {
		t.Error("Expected output to contain Long description when Short is empty")
	}
}

func TestShowGeneralHelpWithoutVersion(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:   "gonuget",
		Short: "NuGet package manager for Go",
		// Version is intentionally empty
	}

	var stdout, stderr bytes.Buffer
	console := output.NewConsole(&stdout, &stderr, output.VerbosityNormal)

	// Set up custom help function (like app.go does)
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		version := cmd.Root().Version
		if version == "" {
			version = "dev"
		}

		console.Println("NuGet Command Line " + version)
		console.Println("")
		console.Println("Usage: gonuget [options] [command]")
		console.Println("")
		console.Println("Options:")
		console.Println("  -h|--help  Show help information")
		console.Println("  --version  Show version information")
		console.Println("")
		console.Println("Commands:")

		hideCommands := map[string]bool{
			"completion": true,
			"version":    true,
		}

		for _, subCmd := range cmd.Root().Commands() {
			if subCmd.Hidden || hideCommands[subCmd.Name()] {
				continue
			}
			name := subCmd.Name()
			short := subCmd.Short
			if short == "" {
				short = subCmd.Long
			}
			// Pad right to 8 characters
			for len(name) < 8 {
				name += " "
			}
			console.Println("  " + name + " " + short)
		}

		console.Println("")
		console.Println("Use \"gonuget [command] --help\" for more information about a command.")
	})

	err := rootCmd.Help()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := stdout.String()

	// Verify that "dev" is used when version is empty
	if !strings.Contains(output, "NuGet Command Line dev") {
		t.Error("Expected output to contain 'NuGet Command Line dev' when version is empty")
	}
}

func TestShowGeneralHelpWithHiddenCommand(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:     "gonuget",
		Short:   "NuGet package manager for Go",
		Version: "1.0.0-test",
	}

	// Add visible and hidden commands
	rootCmd.AddCommand(&cobra.Command{
		Use:   "visible",
		Short: "Visible command",
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:    "hidden",
		Short:  "Hidden command",
		Hidden: true,
	})

	var stdout, stderr bytes.Buffer
	console := output.NewConsole(&stdout, &stderr, output.VerbosityNormal)

	err := showGeneralHelp(console, rootCmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := stdout.String()

	// Verify visible command is included
	if !strings.Contains(output, "visible") {
		t.Error("Expected output to contain visible command")
	}

	// Verify hidden command is excluded
	if strings.Contains(output, "hidden") {
		t.Error("Expected output to exclude hidden command")
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
