package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

func TestPackageList_WithSolutionFile(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test solution file
	solutionContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "WebApi", "src\\WebApi\\WebApi.csproj", "{1234}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "DataLayer", "src\\DataLayer\\DataLayer.csproj", "{5678}"
EndProject
Project("{2150E333-8FDC-42A3-9474-1A3956D46DE8}") = "Solution Items", "Solution Items", "{9ABC}"
EndProject
Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
		Release|Any CPU = Release|Any CPU
	EndGlobalSection
EndGlobal
`
	solutionPath := filepath.Join(tempDir, "TestSolution.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test project directories
	webApiDir := filepath.Join(tempDir, "src", "WebApi")
	dataLayerDir := filepath.Join(tempDir, "src", "DataLayer")
	if err := os.MkdirAll(webApiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataLayerDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test project files with package references
	webApiProject := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Microsoft.AspNetCore.OpenApi" Version="8.0.0" />
  </ItemGroup>
</Project>`

	dataLayerProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.EntityFrameworkCore" Version="8.0.0" />
    <PackageReference Include="Dapper" Version="2.1.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(filepath.Join(webApiDir, "WebApi.csproj"), []byte(webApiProject), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataLayerDir, "DataLayer.csproj"), []byte(dataLayerProject), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package list command
	cmd := commands.NewPackageListCommand()

	// Capture output
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)

	// Set arguments to the solution file
	cmd.SetArgs([]string{solutionPath})

	// Execute command
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output contains all projects
	output := outputBuffer.String()
	expectedProjects := []string{
		"WebApi.csproj",
		"DataLayer.csproj",
	}
	for _, project := range expectedProjects {
		if !strings.Contains(output, project) {
			t.Errorf("Output missing project %s\nActual output:\n%s", project, output)
		}
	}

	// Verify output contains all packages
	expectedPackages := []string{
		"Newtonsoft.Json 13.0.1",
		"Microsoft.AspNetCore.OpenApi 8.0.0",
		"Microsoft.EntityFrameworkCore 8.0.0",
		"Dapper 2.1.0",
	}
	for _, pkg := range expectedPackages {
		if !strings.Contains(output, pkg) {
			t.Errorf("Output missing package %s\nActual output:\n%s", pkg, output)
		}
	}

	// Verify solution folders are excluded
	if strings.Contains(output, "Solution Items") {
		t.Error("Output should not include solution folders")
	}
}

func TestPackageList_WithSlnxFile(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test .slnx file
	slnxContent := `<?xml version="1.0" encoding="utf-8"?>
<Solution>
  <Folder Name="src">
    <Project Path="src/WebApi/WebApi.csproj" />
    <Project Path="src/DataLayer/DataLayer.csproj" />
  </Folder>
  <Folder Name="tests">
    <Project Path="tests/UnitTests/UnitTests.csproj" />
  </Folder>
  <Properties>
    <Property Name="VisualStudioVersion" Value="17.0.31903.59" />
  </Properties>
</Solution>`

	slnxPath := filepath.Join(tempDir, "TestSolution.slnx")
	if err := os.WriteFile(slnxPath, []byte(slnxContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test project directories to match the paths in the .slnx
	// .slnx has "WebApi/WebApi.csproj", "DataLayer/DataLayer.csproj", etc.
	// So these should be relative to where .slnx file is, inside src/ and tests/ folders
	webApiDir := filepath.Join(tempDir, "src", "WebApi")
	dataLayerDir := filepath.Join(tempDir, "src", "DataLayer")
	unitTestsDir := filepath.Join(tempDir, "tests", "UnitTests")
	if err := os.MkdirAll(webApiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataLayerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unitTestsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test project files
	webApiProject := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Serilog" Version="3.0.0" />
  </ItemGroup>
</Project>`

	dataLayerProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="AutoMapper" Version="12.0.0" />
  </ItemGroup>
</Project>`

	unitTestsProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.5.0" />
    <PackageReference Include="Moq" Version="4.20.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(filepath.Join(webApiDir, "WebApi.csproj"), []byte(webApiProject), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataLayerDir, "DataLayer.csproj"), []byte(dataLayerProject), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitTestsDir, "UnitTests.csproj"), []byte(unitTestsProject), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package list command
	cmd := commands.NewPackageListCommand()

	// Capture output
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)

	// Set arguments to the .slnx file
	cmd.SetArgs([]string{slnxPath})

	// Execute command
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output contains all packages from all projects
	output := outputBuffer.String()
	t.Logf("Output:\n%s", output)

	expectedPackages := []string{
		"Serilog",
		"AutoMapper",
		"xunit",
		"Moq",
	}
	for _, pkg := range expectedPackages {
		if !strings.Contains(output, pkg) {
			t.Errorf("Output missing package %s", pkg)
		}
	}
}

func TestPackageList_WithSlnfFile(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create parent solution file first
	solutionContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "ProjectA", "ProjectA\\ProjectA.csproj", "{1111}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "ProjectB", "ProjectB\\ProjectB.csproj", "{2222}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "ProjectC", "ProjectC\\ProjectC.csproj", "{3333}"
EndProject
Global
EndGlobal
`
	solutionPath := filepath.Join(tempDir, "Full.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a solution filter that only includes ProjectA and ProjectC
	filterContent := `{
  "solution": {
    "path": "Full.sln",
    "projects": [
      "ProjectA\\ProjectA.csproj",
      "ProjectC\\ProjectC.csproj"
    ]
  }
}`

	filterPath := filepath.Join(tempDir, "Filtered.slnf")
	if err := os.WriteFile(filterPath, []byte(filterContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test project directories
	projectADir := filepath.Join(tempDir, "ProjectA")
	projectBDir := filepath.Join(tempDir, "ProjectB")
	projectCDir := filepath.Join(tempDir, "ProjectC")
	for _, dir := range []string{projectADir, projectBDir, projectCDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create project files
	projectA := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="PackageA" Version="1.0.0" />
  </ItemGroup>
</Project>`

	projectB := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="PackageB" Version="2.0.0" />
  </ItemGroup>
</Project>`

	projectC := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="PackageC" Version="3.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(filepath.Join(projectADir, "ProjectA.csproj"), []byte(projectA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectBDir, "ProjectB.csproj"), []byte(projectB), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectCDir, "ProjectC.csproj"), []byte(projectC), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package list command
	cmd := commands.NewPackageListCommand()

	// Capture output
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)

	// Set arguments to the filter file
	cmd.SetArgs([]string{filterPath})

	// Execute command
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output contains only filtered projects' packages
	output := outputBuffer.String()
	if !strings.Contains(output, "PackageA") {
		t.Error("Output missing PackageA from filtered project")
	}
	if !strings.Contains(output, "PackageC") {
		t.Error("Output missing PackageC from filtered project")
	}
	// PackageB should NOT be in output as ProjectB is filtered out
	if strings.Contains(output, "PackageB") {
		t.Error("Output should not include PackageB from filtered-out project")
	}
}

func TestPackageList_WithMissingProjectFile(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test solution file with a missing project
	solutionContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "ExistingProject", "ExistingProject\\ExistingProject.csproj", "{1234}"
EndProject
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "MissingProject", "MissingProject\\MissingProject.csproj", "{5678}"
EndProject
Global
EndGlobal
`
	solutionPath := filepath.Join(tempDir, "TestSolution.sln")
	if err := os.WriteFile(solutionPath, []byte(solutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Only create the ExistingProject
	existingDir := filepath.Join(tempDir, "ExistingProject")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatal(err)
	}

	existingProject := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="TestPackage" Version="1.0.0" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(filepath.Join(existingDir, "ExistingProject.csproj"), []byte(existingProject), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package list command
	cmd := commands.NewPackageListCommand()

	// Capture output
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)

	// Set arguments to the solution file
	cmd.SetArgs([]string{solutionPath})

	// Execute with solution file - should silently skip missing project
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute should not fail with missing project: %v", err)
	}

	// Verify output contains the existing project's package
	output := outputBuffer.String()
	if !strings.Contains(output, "TestPackage") {
		t.Error("Output missing TestPackage from existing project")
	}

	// Should continue processing despite missing project
	if !strings.Contains(output, "ExistingProject") {
		t.Error("Should show output for existing project")
	}
}