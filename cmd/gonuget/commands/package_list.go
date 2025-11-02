package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/cmd/gonuget/solution"
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
			return runPackageList(opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.ProjectPath, "project", "", "The project file to operate on (defaults to current directory)")
	cmd.Flags().StringVar(&opts.Format, "format", "console", "Output format: console or json")

	return cmd
}

// runPackageList implements the package list command logic.
func runPackageList(opts *PackageListOptions, w io.Writer) error {
	start := time.Now()

	// Find the project or solution file
	targetPath := opts.ProjectPath
	if targetPath == "" {
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
			if result.Ambiguous {
				return fmt.Errorf("multiple solution files found: %v. Please specify which one to use", result.FoundFiles)
			}
			targetPath = result.SolutionPath
		} else {
			// Fall back to finding a project file
			foundPath, err := project.FindProjectFile(currentDir)
			if err != nil {
				return fmt.Errorf("failed to find project or solution file: %w", err)
			}
			targetPath = foundPath
		}
	}

	// Check if it's a solution file
	if solution.IsSolutionFile(targetPath) {
		return runPackageListForSolution(targetPath, opts.Format, start, w)
	}

	// Handle as a single project file
	return runPackageListForProject(targetPath, opts.Format, start, w)
}

// runPackageListForSolution handles listing packages for all projects in a solution
func runPackageListForSolution(solutionPath string, format string, start time.Time, w io.Writer) error {
	// Parse the solution file
	sol, err := solution.ParseSolution(solutionPath)
	if err != nil {
		return fmt.Errorf("failed to parse solution %s: %w", solutionPath, err)
	}

	// Get all .NET project paths (excluding solution folders)
	projectPaths := sol.GetProjects()
	if len(projectPaths) == 0 {
		fmt.Fprintf(w, "Solution '%s' contains no projects.\n", filepath.Base(solutionPath))
		return nil
	}

	// Process projects
	if format == "json" {
		return outputSolutionPackageListJSON(solutionPath, sol, projectPaths, start, w)
	}

	return outputSolutionPackageListConsole(solutionPath, sol, projectPaths, w)
}

// runPackageListForProject handles listing packages for a single project
func runPackageListForProject(projectPath string, format string, start time.Time, w io.Writer) error {
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
	if format == "json" {
		return outputPackageListJSON(absPath, framework, packageRefs, start, w)
	}

	return outputPackageListConsole(projectPath, packageRefs, w)
}

// outputPackageListConsole outputs package references in human-readable format
func outputPackageListConsole(projectPath string, packageRefs []project.PackageReference, w io.Writer) error {
	fmt.Fprintf(w, "Project '%s' has the following package references:\n", filepath.Base(projectPath))
	fmt.Fprintln(w)

	if len(packageRefs) == 0 {
		fmt.Fprintln(w, "   [No package references found]")
		return nil
	}

	for _, ref := range packageRefs {
		if ref.Version != "" {
			fmt.Fprintf(w, "   %s %s\n", ref.Include, ref.Version)
		} else {
			fmt.Fprintf(w, "   %s (version managed centrally)\n", ref.Include)
		}
	}

	return nil
}

// outputPackageListJSON outputs package references in JSON format matching schema
func outputPackageListJSON(projectPath, framework string, packageRefs []project.PackageReference, start time.Time, w io.Writer) error {
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

	// Write JSON to writer
	return output.WriteJSON(w, jsonOutput)
}

// outputSolutionPackageListConsole outputs packages from all projects in console format
func outputSolutionPackageListConsole(solutionPath string, sol *solution.Solution, projectPaths []string, w io.Writer) error {
	fmt.Fprintf(w, "Solution '%s' has the following package references:\n", filepath.Base(solutionPath))
	fmt.Fprintln(w)

	totalPackages := 0
	successCount := 0

	// Create warning writer for stderr output
	warningWriter := output.NewWarningWriter()

	// Process each project
	for _, projectPath := range projectPaths {
		// Make absolute path based on solution directory
		absProjectPath := projectPath
		if !filepath.IsAbs(projectPath) {
			absProjectPath = filepath.Join(sol.SolutionDir, projectPath)
		}

		// Check if project file exists
		if _, err := os.Stat(absProjectPath); os.IsNotExist(err) {
			// Show warning for missing project file (T041: warning output formatting)
			warningWriter.WriteMissingProjectWarning(absProjectPath)
			continue
		}

		// Load the project
		proj, err := project.LoadProject(absProjectPath)
		if err != nil {
			// Skip projects that can't be loaded
			fmt.Fprintf(w, "   [Warning: Could not load project %s]\n", filepath.Base(projectPath))
			continue
		}

		successCount++
		packageRefs := proj.GetPackageReferences()

		if len(packageRefs) > 0 {
			fmt.Fprintf(w, "   Project '%s' has the following package references:\n", filepath.Base(projectPath))
			for _, ref := range packageRefs {
				if ref.Version != "" {
					fmt.Fprintf(w, "      > %s %s\n", ref.Include, ref.Version)
				} else {
					fmt.Fprintf(w, "      > %s (version managed centrally)\n", ref.Include)
				}
				totalPackages++
			}
			fmt.Fprintln(w)
		}
	}

	if totalPackages == 0 {
		fmt.Fprintln(w, "   [No package references found in any projects]")
	} else {
		fmt.Fprintf(w, "Total packages: %d across %d projects\n", totalPackages, successCount)
	}

	return nil
}

// outputSolutionPackageListJSON outputs packages from all projects in JSON format
func outputSolutionPackageListJSON(solutionPath string, sol *solution.Solution, projectPaths []string, start time.Time, w io.Writer) error {
	// Create a combined output structure
	type solutionOutput struct {
		Solution  string                     `json:"solution"`
		Projects  []output.PackageListOutput `json:"projects"`
		ElapsedMs int64                     `json:"elapsedMs"`
	}

	result := solutionOutput{
		Solution: solutionPath,
		Projects: []output.PackageListOutput{},
	}

	// Create warning writer for stderr output
	warningWriter := output.NewWarningWriter()

	// Process each project
	for _, projectPath := range projectPaths {
		// Make absolute path based on solution directory
		absProjectPath := projectPath
		if !filepath.IsAbs(projectPath) {
			absProjectPath = filepath.Join(sol.SolutionDir, projectPath)
		}

		// Check if project file exists
		if _, err := os.Stat(absProjectPath); os.IsNotExist(err) {
			// Show warning for missing project file (JSON mode still shows warnings to stderr)
			warningWriter.WriteMissingProjectWarning(absProjectPath)
			continue
		}

		// Load the project
		proj, err := project.LoadProject(absProjectPath)
		if err != nil {
			// Skip projects that can't be loaded
			continue
		}

		// Get framework
		framework := proj.TargetFramework
		if framework == "" {
			continue // Skip projects without target framework
		}

		// Create project output
		projectOutput := output.NewPackageListOutput(absProjectPath, framework, start)

		// Convert package references
		for _, ref := range proj.GetPackageReferences() {
			projectOutput.Packages = append(projectOutput.Packages, output.PackageReference{
				ID:              ref.Include,
				Version:         ref.Version,
				Type:            "direct",
				ResolvedVersion: ref.Version,
			})
		}

		result.Projects = append(result.Projects, *projectOutput)
	}

	// Update elapsed time
	result.ElapsedMs = output.MeasureElapsed(start)

	// Write JSON to writer
	return output.WriteJSON(w, result)
}

// init registers the package list subcommand with the package parent command
func init() {
	packageCmd := GetPackageCommand()
	packageCmd.AddCommand(NewPackageListCommand())
}
