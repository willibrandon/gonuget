package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// PackageListOptions holds the configuration for the package list command.
type PackageListOptions struct {
	ProjectPath string
	Format      string
}

// NewPackageListCommand creates the 'package list' subcommand.
func NewPackageListCommand() *cobra.Command {
	opts := &PackageListOptions{}

	cmd := &cobra.Command{
		Use:   "list [PROJECT]",
		Short: "List package references in a project file",
		Long: `List all NuGet package references in a .NET project file.

This command displays all package references from a .NET project file (.csproj, .fsproj, .vbproj).
Output can be formatted as console (human-readable) or JSON.

Examples:
  gonuget package list
  gonuget package list --project MyProject.csproj
  gonuget package list --format json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If project is provided as positional arg, use it
			if len(args) == 1 {
				opts.ProjectPath = args[0]
			}
			return runPackageList(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ProjectPath, "project", "", "The project file to operate on (defaults to current directory)")
	cmd.Flags().StringVar(&opts.Format, "format", "console", "Output format: console or json")

	return cmd
}

// runPackageList implements the package list command logic.
func runPackageList(opts *PackageListOptions) error {
	start := time.Now()

	// Find the project file
	projectPath := opts.ProjectPath
	if projectPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		foundPath, err := project.FindProjectFile(currentDir)
		if err != nil {
			return fmt.Errorf("failed to find project file: %w", err)
		}
		projectPath = foundPath
	}

	// Make path absolute for consistent output
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		absPath = projectPath
	}

	// Load the project
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project %s: %w", projectPath, err)
	}

	// Get package references
	packageRefs := proj.GetPackageReferences()

	// Get target framework
	framework := proj.TargetFramework
	if framework == "" {
		return fmt.Errorf("project does not specify a TargetFramework")
	}

	// Output based on format
	if opts.Format == "json" {
		return outputPackageListJSON(absPath, framework, packageRefs, start)
	}

	return outputPackageListConsole(projectPath, packageRefs)
}

// outputPackageListConsole outputs package references in human-readable format
func outputPackageListConsole(projectPath string, packageRefs []project.PackageReference) error {
	fmt.Printf("Project '%s' has the following package references:\n", filepath.Base(projectPath))
	fmt.Println()

	if len(packageRefs) == 0 {
		fmt.Println("   [No package references found]")
		return nil
	}

	for _, ref := range packageRefs {
		if ref.Version != "" {
			fmt.Printf("   %s %s\n", ref.Include, ref.Version)
		} else {
			fmt.Printf("   %s (version managed centrally)\n", ref.Include)
		}
	}

	return nil
}

// outputPackageListJSON outputs package references in JSON format matching schema
func outputPackageListJSON(projectPath, framework string, packageRefs []project.PackageReference, start time.Time) error {
	jsonOutput := output.NewPackageListOutput(projectPath, framework, start)

	// Convert project.PackageReference to output.PackageReference
	for _, ref := range packageRefs {
		jsonOutput.Packages = append(jsonOutput.Packages, output.PackageReference{
			ID:              ref.Include,
			Version:         ref.Version,
			Type:            "direct", // All references from .csproj are direct
			ResolvedVersion: ref.Version,
		})
	}

	// Update elapsed time
	jsonOutput.ElapsedMs = output.MeasureElapsed(start)

	// Write JSON to stdout
	return output.WriteJSON(os.Stdout, jsonOutput)
}

// init registers the package list subcommand with the package parent command
func init() {
	packageCmd := GetPackageCommand()
	packageCmd.AddCommand(NewPackageListCommand())
}
