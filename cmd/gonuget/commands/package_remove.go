package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/cmd/gonuget/solution"
)

// PackageRemoveOptions holds the configuration for the package remove command.
type PackageRemoveOptions struct {
	ProjectPath string
}

// NewPackageRemoveCommand creates the 'package remove' subcommand.
func NewPackageRemoveCommand() *cobra.Command {
	opts := &PackageRemoveOptions{}

	cmd := &cobra.Command{
		Use:   "remove <PACKAGE_ID>",
		Short: "Remove a package reference from a project file",
		Long: `Remove a NuGet package reference from a .NET project file.

This command removes a package reference from a .NET project file (.csproj, .fsproj, .vbproj).
If the project uses Central Package Management, the package version in Directory.Packages.props
is NOT removed (only the PackageReference in the project file).

Examples:
  gonuget package remove Newtonsoft.Json
  gonuget package remove Newtonsoft.Json --project MyProject.csproj`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageID := args[0]
			return runPackageRemove(packageID, opts)
		},
	}

	cmd.Flags().StringVar(&opts.ProjectPath, "project", "", "The project file to operate on (defaults to current directory)")

	return cmd
}

// runPackageRemove implements the package remove command logic.
func runPackageRemove(packageID string, opts *PackageRemoveOptions) error {
	// Find the project file
	projectPath := opts.ProjectPath
	if projectPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try to detect a solution file first
		detector := solution.NewDetector(currentDir)
		result, err := detector.DetectSolution()
		if err != nil {
			return fmt.Errorf("failed to detect solution: %w", err)
		}

		if result.Found {
			// Solution file found - this is not supported for remove operation
			absPath, _ := filepath.Abs(result.SolutionPath)
			return &InvalidProjectFileError{Path: absPath}
		}

		// No solution file, try to find a project file
		foundPath, err := project.FindProjectFile(currentDir)
		if err != nil {
			return fmt.Errorf("failed to find project file: %w", err)
		}
		projectPath = foundPath
	} else {
		// Check if the provided path is a solution file
		if solution.IsSolutionFile(projectPath) {
			// Return error for solution file
			absPath, _ := filepath.Abs(projectPath)
			return &InvalidProjectFileError{Path: absPath}
		}
	}

	// Load the project
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project %s: %w", projectPath, err)
	}

	// Check if package exists
	packageRefs := proj.GetPackageReferences()
	found := false
	for _, ref := range packageRefs {
		if ref.Include == packageID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("package '%s' not found in project '%s'", packageID, projectPath)
	}

	// Remove the package reference
	if !proj.RemovePackageReference(packageID) {
		return fmt.Errorf("failed to remove package reference '%s'", packageID)
	}

	// Save the project file
	if err := proj.Save(); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	fmt.Printf("info : Package '%s' removed from project '%s'\n", packageID, projectPath)

	return nil
}

// init registers the package remove subcommand with the package parent command
func init() {
	packageCmd := GetPackageCommand()
	packageCmd.AddCommand(NewPackageRemoveCommand())
}
