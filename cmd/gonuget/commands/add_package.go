package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/restore"
	"github.com/willibrandon/gonuget/version"
)

// AddPackageOptions holds the configuration for the add package command.
type AddPackageOptions struct {
	ProjectPath      string
	Version          string
	Framework        string
	NoRestore        bool
	Source           string
	PackageDirectory string
	Prerelease       bool
	Interactive      bool
}

// NewAddPackageCommand creates the 'add package' subcommand.
func NewAddPackageCommand() *cobra.Command {
	opts := &AddPackageOptions{}

	cmd := &cobra.Command{
		Use:   "package <PACKAGE_ID>",
		Short: "Add a NuGet package reference to a project file",
		Long: `Add a NuGet package reference to a project file.

This command adds or updates a package reference in a .NET project file (.csproj, .fsproj, .vbproj).
If no version is specified, the latest stable version is resolved from the package source.

Examples:
  gonuget add package Newtonsoft.Json
  gonuget add package Newtonsoft.Json --version 13.0.3
  gonuget add package Newtonsoft.Json --framework net8.0
  gonuget add package Newtonsoft.Json --prerelease`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageID := args[0]
			return runAddPackage(cmd.Context(), packageID, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Version, "version", "v", "", "The version of the package to add")
	cmd.Flags().StringVarP(&opts.Framework, "framework", "f", "", "Add the reference only when targeting a specific framework")
	cmd.Flags().BoolVar(&opts.NoRestore, "no-restore", false, "Don't perform an implicit restore after adding the package")
	cmd.Flags().StringVarP(&opts.Source, "source", "s", "", "The NuGet package source to use during the restore")
	cmd.Flags().StringVar(&opts.PackageDirectory, "package-directory", "", "The directory where to restore the packages")
	cmd.Flags().BoolVar(&opts.Prerelease, "prerelease", false, "Allow prerelease packages to be installed")
	cmd.Flags().BoolVar(&opts.Interactive, "interactive", false, "Allow the command to stop and wait for user input or action")
	cmd.Flags().StringVar(&opts.ProjectPath, "project", "", "The project file to operate on (defaults to current directory)")

	return cmd
}

// runAddPackage implements the add package command logic.
func runAddPackage(ctx context.Context, packageID string, opts *AddPackageOptions) error {
	// 1. Find the project file
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

	// 2. Load the project
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project %s: %w", projectPath, err)
	}

	// 3. Check for Central Package Management
	if proj.IsCentralPackageManagementEnabled() {
		return fmt.Errorf("this project uses Central Package Management (CPM). Package versions must be managed in Directory.Packages.props. Use 'gonuget add package %s' in the solution directory instead", packageID)
	}

	// 4. Resolve version if not specified
	packageVersion := opts.Version
	if packageVersion == "" {
		resolvedVersion, err := resolveLatestVersion(ctx, packageID, opts)
		if err != nil {
			return fmt.Errorf("failed to resolve latest version for %s: %w", packageID, err)
		}
		packageVersion = resolvedVersion
		fmt.Printf("Resolved version: %s\n", packageVersion)
	}

	// 5. Validate version format
	if _, err := version.Parse(packageVersion); err != nil {
		return fmt.Errorf("invalid package version '%s': %w", packageVersion, err)
	}

	// 6. Determine target frameworks
	var frameworks []string
	if opts.Framework != "" {
		frameworks = []string{opts.Framework}
	}

	// 7. Add or update the package reference
	updated, err := proj.AddOrUpdatePackageReference(packageID, packageVersion, frameworks)
	if err != nil {
		return fmt.Errorf("failed to add package reference: %w", err)
	}

	// 8. Save the project file
	if err := proj.Save(); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	// 9. Report success
	if updated {
		fmt.Printf("Updated package '%s' from project '%s'\n", packageID, projectPath)
	} else {
		fmt.Printf("Added package '%s' to project '%s'\n", packageID, projectPath)
	}

	// 10. Perform restore if needed (M2.1 Chunk 5-7)
	if !opts.NoRestore {
		fmt.Println("Restore is not yet implemented (coming in Chunk 5)")
		// TODO: Implement restore in Chunk 5
	}

	return nil
}

// resolveLatestVersion resolves the latest version of a package from the package source.
func resolveLatestVersion(ctx context.Context, packageID string, opts *AddPackageOptions) (string, error) {
	// Create a client with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Call library function
	return restore.ResolveLatestVersion(ctx, packageID, &restore.ResolveLatestVersionOptions{
		Source:     opts.Source,
		Prerelease: opts.Prerelease,
	})
}
