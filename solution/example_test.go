package solution_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/willibrandon/gonuget/solution"
)

// ExampleGetParser demonstrates how to get a parser for different solution file formats.
func ExampleGetParser() {
	// Create a temporary directory for the example
	tempDir, err := os.MkdirTemp("", "solution-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a sample .sln file
	slnPath := filepath.Join(tempDir, "Example.sln")
	slnContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "MyApp", "src\MyApp\MyApp.csproj", "{11111111-1111-1111-1111-111111111111}"
EndProject
Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
		Release|Any CPU = Release|Any CPU
	EndGlobalSection
EndGlobal
`
	if err := os.WriteFile(slnPath, []byte(slnContent), 0644); err != nil {
		panic(err)
	}

	// Get the appropriate parser for the solution file
	parser, err := solution.GetParser(slnPath)
	if err != nil {
		panic(err)
	}

	// Parse the solution file
	sol, err := parser.Parse(slnPath)
	if err != nil {
		panic(err)
	}

	// Display solution information
	fmt.Printf("Solution format version: %s\n", sol.FormatVersion)
	fmt.Printf("Number of projects: %d\n", len(sol.Projects))
	if len(sol.Projects) > 0 {
		fmt.Printf("First project: %s\n", sol.Projects[0].Name)
	}

	// Output:
	// Solution format version: 12.00
	// Number of projects: 1
	// First project: MyApp
}

// ExampleNewDetector demonstrates how to automatically detect solution files in a directory.
func ExampleNewDetector() {
	// Create a temporary directory for the example
	tempDir, err := os.MkdirTemp("", "detector-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a sample .sln file
	slnPath := filepath.Join(tempDir, "MyApp.sln")
	slnContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
Global
EndGlobal
`
	if err := os.WriteFile(slnPath, []byte(slnContent), 0644); err != nil {
		panic(err)
	}

	// Create a detector for the directory
	detector := solution.NewDetector(tempDir)

	// Detect solution files
	result, err := detector.DetectSolution()
	if err != nil {
		panic(err)
	}

	// Check detection results
	if result.Found {
		fmt.Printf("Found solution: %s\n", filepath.Base(result.SolutionPath))
		fmt.Printf("Format: %s\n", result.Format)
	}

	if result.Ambiguous {
		fmt.Printf("Multiple solutions found: %d\n", len(result.FoundFiles))
	}

	// Output:
	// Found solution: MyApp.sln
	// Format: sln
}

// ExampleSolution_GetProjects demonstrates how to get project paths from a parsed solution.
func ExampleSolution_GetProjects() {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "projects-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a sample .sln file with multiple projects
	slnPath := filepath.Join(tempDir, "MySolution.sln")
	slnContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "WebApp", "src\WebApp\WebApp.csproj", "{11111111-1111-1111-1111-111111111111}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "Core", "src\Core\Core.csproj", "{22222222-2222-2222-2222-222222222222}"
EndProject
Project("{2150E333-8FDC-42A3-9474-1A3956D46DE8}") = "Solution Items", "Solution Items", "{33333333-3333-3333-3333-333333333333}"
EndProject
Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
	EndGlobalSection
EndGlobal
`
	if err := os.WriteFile(slnPath, []byte(slnContent), 0644); err != nil {
		panic(err)
	}

	// Parse the solution
	parser, err := solution.GetParser(slnPath)
	if err != nil {
		panic(err)
	}

	sol, err := parser.Parse(slnPath)
	if err != nil {
		panic(err)
	}

	// Get all project paths (excludes solution folders)
	projectPaths := sol.GetProjects()

	fmt.Printf("Total projects: %d\n", len(projectPaths))
	for i, path := range projectPaths {
		// Show relative path for consistent output
		relPath, _ := filepath.Rel(tempDir, path)
		fmt.Printf("Project %d: %s\n", i+1, filepath.ToSlash(relPath))
	}

	// Output:
	// Total projects: 2
	// Project 1: src/WebApp/WebApp.csproj
	// Project 2: src/Core/Core.csproj
}
