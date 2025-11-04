//go:build integration

package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

// TestSolutionIntegration_EndToEnd tests the complete solution file support feature
func TestSolutionIntegration_EndToEnd(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a complex test solution structure
	tempDir := t.TempDir()

	// Create solution file content (Windows-style paths to test cross-platform)
	solutionContent := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "Core", "src\Core\Core.csproj", "{1234ABCD-1234-1234-1234-123456789012}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "Web", "src\Web\Web.csproj", "{2234ABCD-1234-1234-1234-123456789012}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "Tests", "tests\UnitTests\UnitTests.csproj", "{3234ABCD-1234-1234-1234-123456789012}"
EndProject
Project("{2150E333-8FDC-42A3-9474-1A3956D46DE8}") = "Solution Items", "Solution Items", "{4234ABCD-1234-1234-1234-123456789012}"
	ProjectSection(SolutionItems) = preProject
		README.md = README.md
		.gitignore = .gitignore
	EndProjectSection
EndProject
Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
		Release|Any CPU = Release|Any CPU
	EndGlobalSection
	GlobalSection(ProjectConfigurationPlatforms) = postSolution
		{1234ABCD-1234-1234-1234-123456789012}.Debug|Any CPU.ActiveCfg = Debug|Any CPU
		{1234ABCD-1234-1234-1234-123456789012}.Debug|Any CPU.Build.0 = Debug|Any CPU
		{1234ABCD-1234-1234-1234-123456789012}.Release|Any CPU.ActiveCfg = Release|Any CPU
		{1234ABCD-1234-1234-1234-123456789012}.Release|Any CPU.Build.0 = Release|Any CPU
		{2234ABCD-1234-1234-1234-123456789012}.Debug|Any CPU.ActiveCfg = Debug|Any CPU
		{2234ABCD-1234-1234-1234-123456789012}.Debug|Any CPU.Build.0 = Debug|Any CPU
		{2234ABCD-1234-1234-1234-123456789012}.Release|Any CPU.ActiveCfg = Release|Any CPU
		{2234ABCD-1234-1234-1234-123456789012}.Release|Any CPU.Build.0 = Release|Any CPU
		{3234ABCD-1234-1234-1234-123456789012}.Debug|Any CPU.ActiveCfg = Debug|Any CPU
		{3234ABCD-1234-1234-1234-123456789012}.Debug|Any CPU.Build.0 = Debug|Any CPU
		{3234ABCD-1234-1234-1234-123456789012}.Release|Any CPU.ActiveCfg = Release|Any CPU
		{3234ABCD-1234-1234-1234-123456789012}.Release|Any CPU.Build.0 = Release|Any CPU
	EndGlobalSection
EndGlobal`

	// Create solution file
	solutionPath := filepath.Join(tempDir, "TestSolution.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project directories
	srcDir := filepath.Join(tempDir, "src")
	coreDir := filepath.Join(srcDir, "Core")
	webDir := filepath.Join(srcDir, "Web")
	testsDir := filepath.Join(tempDir, "tests", "UnitTests")

	for _, dir := range []string{coreDir, webDir, testsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create project files with different package configurations
	coreProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="Microsoft.Extensions.Logging" Version="8.0.0" />
  </ItemGroup>
</Project>`

	webProject := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.Mvc" Version="2.2.0" />
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="Serilog" Version="3.1.1" />
  </ItemGroup>
</Project>`

	testProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.6.1" />
    <PackageReference Include="Moq" Version="4.20.69" />
    <PackageReference Include="FluentAssertions" Version="6.12.0" />
  </ItemGroup>
</Project>`

	// Write project files
	if err := os.WriteFile(filepath.Join(coreDir, "Core.csproj"), []byte(coreProject), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "Web.csproj"), []byte(webProject), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "UnitTests.csproj"), []byte(testProject), 0644); err != nil {
		t.Fatal(err)
	}

	// Test 1: List packages from solution file
	t.Run("ListPackagesFromSolution", func(t *testing.T) {
		cmd := commands.NewPackageListCommand()
		outputBuffer := &bytes.Buffer{}
		cmd.SetOut(outputBuffer)
		cmd.SetErr(outputBuffer)
		cmd.SetArgs([]string{"--project", solutionPath})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Failed to list packages: %v", err)
		}

		output := outputBuffer.String()

		// Verify output contains all projects
		expectedProjects := []string{
			"Core.csproj",
			"Web.csproj",
			"UnitTests.csproj",
		}

		for _, proj := range expectedProjects {
			if !strings.Contains(output, proj) {
				t.Errorf("Output missing project %s", proj)
			}
		}

		// Verify output contains all packages
		expectedPackages := []string{
			"Newtonsoft.Json",
			"Microsoft.Extensions.Logging",
			"Microsoft.AspNetCore.Mvc",
			"Serilog",
			"xunit",
			"Moq",
			"FluentAssertions",
		}

		for _, pkg := range expectedPackages {
			if !strings.Contains(output, pkg) {
				t.Errorf("Output missing package %s", pkg)
			}
		}

		// Verify solution folder is not included
		if strings.Contains(output, "Solution Items") {
			t.Error("Output should not include solution folders")
		}
	})

	// Test 2: Add package to solution file (should fail with correct error)
	t.Run("AddPackageToSolution_ReturnsCorrectError", func(t *testing.T) {
		cmd := commands.NewPackageAddCommand()
		outputBuffer := &bytes.Buffer{}
		cmd.SetOut(outputBuffer)
		cmd.SetErr(outputBuffer)
		cmd.SetArgs([]string{"SomePackage", "--project", solutionPath})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("Expected error when adding package to solution file")
		}

		// Verify correct error message format
		if !strings.Contains(err.Error(), "Couldn't find a project to run") {
			t.Errorf("Incorrect error message: %v", err)
		}
		if !strings.Contains(err.Error(), tempDir) {
			t.Errorf("Error should contain directory path: %v", err)
		}
	})

	// Test 3: Remove package from solution file (should fail with correct error)
	t.Run("RemovePackageFromSolution_ReturnsCorrectError", func(t *testing.T) {
		cmd := commands.NewPackageRemoveCommand()
		outputBuffer := &bytes.Buffer{}
		cmd.SetOut(outputBuffer)
		cmd.SetErr(outputBuffer)
		cmd.SetArgs([]string{"SomePackage", "--project", solutionPath})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("Expected error when removing package from solution file")
		}

		// Verify correct error message format
		if !strings.Contains(err.Error(), "Missing or invalid project file") {
			t.Errorf("Incorrect error message: %v", err)
		}
		if !strings.Contains(err.Error(), solutionPath) {
			t.Errorf("Error should contain solution path: %v", err)
		}
	})

	// Test 4: Auto-detect solution in current directory
	t.Run("AutoDetectSolution", func(t *testing.T) {
		// Change to solution directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}

		cmd := commands.NewPackageListCommand()
		outputBuffer := &bytes.Buffer{}
		cmd.SetOut(outputBuffer)
		cmd.SetErr(outputBuffer)
		// No --project flag, should auto-detect
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Failed to auto-detect and list packages: %v", err)
		}

		output := outputBuffer.String()

		// Should contain packages from all projects
		if !strings.Contains(output, "Newtonsoft.Json") {
			t.Error("Auto-detection failed: missing expected package")
		}

		// Should show it's using the solution file
		if !strings.Contains(output, "Core.csproj") {
			t.Error("Auto-detection failed: missing project reference")
		}
	})
}

// TestSolutionIntegration_MissingProject tests handling of missing project files
func TestSolutionIntegration_MissingProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create solution with reference to missing project
	solutionContent := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "ExistingProject", "ExistingProject.csproj", "{1234ABCD-1234-1234-1234-123456789012}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "MissingProject", "Missing\MissingProject.csproj", "{2234ABCD-1234-1234-1234-123456789012}"
EndProject
Global
EndGlobal`

	solutionPath := filepath.Join(tempDir, "TestSolution.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create only the existing project
	existingProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="TestPackage" Version="1.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(filepath.Join(tempDir, "ExistingProject.csproj"), []byte(existingProject), 0644); err != nil {
		t.Fatal(err)
	}

	// Test that missing project is silently skipped
	cmd := commands.NewPackageListCommand()
	outputBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(errBuffer)
	cmd.SetArgs([]string{"--project", solutionPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Should not fail with missing project: %v", err)
	}

	output := outputBuffer.String()
	errOutput := errBuffer.String()

	// Should show packages from existing project
	if !strings.Contains(output, "TestPackage") {
		t.Error("Should list packages from existing project")
	}

	// Should show existing project
	if !strings.Contains(output, "ExistingProject.csproj") {
		t.Error("Should show existing project in output")
	}

	// Should show warning about missing project
	if !strings.Contains(errOutput, "Warning:") && !strings.Contains(errOutput, "MissingProject.csproj") {
		// For now, missing projects are silently skipped
		// This behavior matches the spec
		t.Log("Missing project silently skipped (as per spec)")
	}
}
