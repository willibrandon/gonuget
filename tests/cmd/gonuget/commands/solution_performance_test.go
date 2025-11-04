//go:build performance

package commands_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// TestSolutionPerformance_LargeSolution tests performance with 100+ projects
func TestSolutionPerformance_LargeSolution(t *testing.T) {
	// Skip if not running performance tests
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir := t.TempDir()
	numProjects := 120 // Test with 120 projects

	// Generate a large solution file
	var solutionBuilder strings.Builder
	solutionBuilder.WriteString(`Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
`)

	// Generate project entries
	for i := 1; i <= numProjects; i++ {
		guid := fmt.Sprintf("{%08X-1234-1234-1234-123456789012}", i)
		projectName := fmt.Sprintf("Project%03d", i)
		projectPath := fmt.Sprintf("projects\\%s\\%s.csproj", projectName, projectName)

		solutionBuilder.WriteString(fmt.Sprintf(
			`Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "%s", "%s", "%s"
EndProject
`, projectName, projectPath, guid))
	}

	// Add some solution folders for realism
	for i := 1; i <= 10; i++ {
		folderGuid := fmt.Sprintf("{F%07X-8FDC-42A3-9474-1A3956D46DE8}", i)
		folderName := fmt.Sprintf("Folder%02d", i)

		solutionBuilder.WriteString(fmt.Sprintf(
			`Project("{2150E333-8FDC-42A3-9474-1A3956D46DE8}") = "%s", "%s", "%s"
EndProject
`, folderName, folderName, folderGuid))
	}

	solutionBuilder.WriteString(`Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
		Release|Any CPU = Release|Any CPU
	EndGlobalSection
	GlobalSection(ProjectConfigurationPlatforms) = postSolution
`)

	// Add configuration for each project
	for i := 1; i <= numProjects; i++ {
		guid := fmt.Sprintf("{%08X-1234-1234-1234-123456789012}", i)
		solutionBuilder.WriteString(fmt.Sprintf(`		%s.Debug|Any CPU.ActiveCfg = Debug|Any CPU
		%s.Debug|Any CPU.Build.0 = Debug|Any CPU
		%s.Release|Any CPU.ActiveCfg = Release|Any CPU
		%s.Release|Any CPU.Build.0 = Release|Any CPU
`, guid, guid, guid, guid))
	}

	solutionBuilder.WriteString(`	EndGlobalSection
EndGlobal`)

	// Write solution file
	solutionPath := filepath.Join(tempDir, "LargeSolution.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionBuilder.String()), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project files
	projectTemplate := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="Microsoft.Extensions.Logging" Version="8.0.0" />
    <PackageReference Include="AutoMapper" Version="12.0.1" />
    <PackageReference Include="FluentValidation" Version="11.8.0" />
    <PackageReference Include="Polly" Version="8.2.0" />
  </ItemGroup>
</Project>`

	// Create all project files
	for i := 1; i <= numProjects; i++ {
		projectName := fmt.Sprintf("Project%03d", i)
		projectDir := filepath.Join(tempDir, "projects", projectName)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		projectPath := filepath.Join(projectDir, projectName+".csproj")

		// Vary the packages a bit for each project
		project := projectTemplate
		if i%3 == 0 {
			// Add extra packages to every third project
			project = strings.Replace(project, "</ItemGroup>",
				`    <PackageReference Include="Serilog" Version="3.1.1" />
    <PackageReference Include="Dapper" Version="2.1.21" />
  </ItemGroup>`, 1)
		}
		if i%5 == 0 {
			// Add test packages to every fifth project
			project = strings.Replace(project, "</ItemGroup>",
				`    <PackageReference Include="xunit" Version="2.6.1" />
    <PackageReference Include="Moq" Version="4.20.69" />
  </ItemGroup>`, 1)
		}

		if err := os.WriteFile(projectPath, []byte(project), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test performance
	t.Run("ListPackages_Performance", func(t *testing.T) {
		cmd := commands.NewPackageListCommand()
		outputBuffer := &bytes.Buffer{}
		cmd.SetOut(outputBuffer)
		cmd.SetErr(outputBuffer)
		cmd.SetArgs([]string{"--project", solutionPath})

		start := time.Now()
		err := cmd.Execute()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to list packages: %v", err)
		}

		output := outputBuffer.String()

		// Verify output contains expected number of projects
		projectCount := strings.Count(output, ".csproj")
		if projectCount != numProjects {
			t.Errorf("Expected %d projects in output, got %d", numProjects, projectCount)
		}

		// Verify performance requirement: must complete in < 30 seconds
		if elapsed > 30*time.Second {
			t.Errorf("Performance requirement failed: took %v, must complete in < 30s", elapsed)
		}

		// Log performance metrics
		t.Logf("Listed packages from %d projects in %v", numProjects, elapsed)
		t.Logf("Average time per project: %v", elapsed/time.Duration(numProjects))
	})

	// Test with some missing projects (should still be performant)
	t.Run("ListPackages_WithMissingProjects", func(t *testing.T) {
		// Delete some project files to simulate missing projects
		for i := 10; i <= 20; i++ {
			projectName := fmt.Sprintf("Project%03d", i)
			projectPath := filepath.Join(tempDir, "projects", projectName, projectName+".csproj")
			os.Remove(projectPath) // Ignore errors, just remove if exists
		}

		cmd := commands.NewPackageListCommand()
		outputBuffer := &bytes.Buffer{}
		cmd.SetOut(outputBuffer)
		cmd.SetErr(outputBuffer)
		cmd.SetArgs([]string{"--project", solutionPath})

		start := time.Now()
		err := cmd.Execute()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Should not fail with missing projects: %v", err)
		}

		// Should still be performant even with missing projects
		if elapsed > 30*time.Second {
			t.Errorf("Performance requirement failed with missing projects: took %v", elapsed)
		}

		t.Logf("Handled solution with missing projects in %v", elapsed)
	})
}

// TestSolutionPerformance_DeepNesting tests performance with deeply nested project structure
func TestSolutionPerformance_DeepNesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir := t.TempDir()

	// Create a solution with deeply nested paths (common in enterprise projects)
	var solutionBuilder strings.Builder
	solutionBuilder.WriteString(`Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
`)

	// Create projects in deeply nested folders
	numProjects := 50
	for i := 1; i <= numProjects; i++ {
		guid := fmt.Sprintf("{%08X-2234-2234-2234-223456789012}", i)
		projectName := fmt.Sprintf("DeepProject%02d", i)
		// Create a deeply nested path
		depth := (i % 5) + 3 // 3 to 7 levels deep
		pathParts := make([]string, depth)
		for j := 0; j < depth; j++ {
			pathParts[j] = fmt.Sprintf("Level%d", j+1)
		}
		projectPath := strings.Join(pathParts, "\\") + fmt.Sprintf("\\%s\\%s.csproj", projectName, projectName)

		solutionBuilder.WriteString(fmt.Sprintf(
			`Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "%s", "%s", "%s"
EndProject
`, projectName, projectPath, guid))

		// Create the actual project file
		fullPath := filepath.Join(tempDir, filepath.FromSlash(strings.ReplaceAll(projectPath, "\\", "/")))
		projectDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		project := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="TestPackage" Version="1.0.0" />
  </ItemGroup>
</Project>`

		if err := os.WriteFile(fullPath, []byte(project), 0644); err != nil {
			t.Fatal(err)
		}
	}

	solutionBuilder.WriteString(`Global
EndGlobal`)

	// Write solution file
	solutionPath := filepath.Join(tempDir, "DeepNesting.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionBuilder.String()), 0644); err != nil {
		t.Fatal(err)
	}

	// Test performance with deep nesting
	cmd := commands.NewPackageListCommand()
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)
	cmd.SetArgs([]string{"--project", solutionPath})

	start := time.Now()
	err := cmd.Execute()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Failed with deeply nested projects: %v", err)
	}

	// Should handle deep nesting efficiently
	if elapsed > 15*time.Second {
		t.Errorf("Deep nesting performance issue: took %v", elapsed)
	}

	t.Logf("Handled %d deeply nested projects in %v", numProjects, elapsed)
}
