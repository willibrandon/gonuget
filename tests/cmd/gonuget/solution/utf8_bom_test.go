package solution_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/solution"
)

// UTF-8 BOM bytes
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func TestSolutionParser_UTF8BOM(t *testing.T) {
	tempDir := t.TempDir()

	// Test solution content
	solutionContent := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "TestProject", "TestProject.csproj", "{12345678-1234-1234-1234-123456789012}"
EndProject
Global
EndGlobal`

	tests := []struct {
		name        string
		withBOM     bool
		description string
	}{
		{
			name:        "UTF8_WithBOM",
			withBOM:     true,
			description: "Solution file with UTF-8 BOM",
		},
		{
			name:        "UTF8_WithoutBOM",
			withBOM:     false,
			description: "Solution file without BOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create solution file
			solutionPath := filepath.Join(tempDir, tt.name+".sln")

			var content []byte
			if tt.withBOM {
				// Prepend BOM to content
				content = make([]byte, len(utf8BOM))
				copy(content, utf8BOM)
				content = append(content, []byte(solutionContent)...)
			} else {
				content = []byte(solutionContent)
			}

			if err := os.WriteFile(solutionPath, content, 0644); err != nil {
				t.Fatal(err)
			}

			// Create test project file
			projectPath := filepath.Join(tempDir, "TestProject.csproj")
			projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
			if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Parse the solution file
			parser, err := solution.GetParser(solutionPath)
			if err != nil {
				t.Fatalf("Failed to get parser: %v", err)
			}

			sol, err := parser.Parse(solutionPath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", tt.description, err)
			}

			// Verify parsing was successful
			projects := sol.GetProjects()
			if len(projects) != 1 {
				t.Errorf("Expected 1 project, got %d", len(projects))
			}

			// Verify project details
			if len(projects) > 0 {
				expectedName := "TestProject.csproj"
				if !strings.HasSuffix(projects[0], expectedName) {
					t.Errorf("Expected project path to end with %s, got %s", expectedName, projects[0])
				}
			}
		})
	}
}

func TestSolutionParser_MixedEncodings(t *testing.T) {
	tempDir := t.TempDir()

	// Test with various encodings and special characters
	testCases := []struct {
		name         string
		solutionName string
		projectName  string
		withBOM      bool
	}{
		{
			name:         "ASCII_Only",
			solutionName: "SimpleSolution",
			projectName:  "SimpleProject",
			withBOM:      false,
		},
		{
			name:         "UTF8_SpecialChars",
			solutionName: "Lösung", // German
			projectName:  "Projekt",
			withBOM:      true,
		},
		{
			name:         "UTF8_Unicode",
			solutionName: "解决方案", // Chinese
			projectName:  "项目",
			withBOM:      true,
		},
		{
			name:         "UTF8_Emoji",
			solutionName: "Solution🚀",
			projectName:  "Project✨",
			withBOM:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create solution content with special characters
			solutionContent := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
# ` + tc.solutionName + `
Project("{9A19103F-16F7-4668-BE54-9A1E7A4F7556}") = "` + tc.projectName + `", "` + tc.projectName + `.csproj", "{12345678-1234-1234-1234-123456789012}"
EndProject
Global
EndGlobal`

			solutionPath := filepath.Join(tempDir, tc.name+".sln")

			var content []byte
			if tc.withBOM {
				content = make([]byte, len(utf8BOM))
				copy(content, utf8BOM)
				content = append(content, []byte(solutionContent)...)
			} else {
				content = []byte(solutionContent)
			}

			if err := os.WriteFile(solutionPath, content, 0644); err != nil {
				t.Fatal(err)
			}

			// Create project file with matching name
			projectPath := filepath.Join(tempDir, tc.projectName+".csproj")
			projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
			if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Parse the solution
			parser, err := solution.GetParser(solutionPath)
			if err != nil {
				t.Fatalf("Failed to get parser for %s: %v", tc.name, err)
			}

			sol, err := parser.Parse(solutionPath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", tc.name, err)
			}

			// Verify the project was found
			projects := sol.GetProjects()
			if len(projects) != 1 {
				t.Errorf("%s: Expected 1 project, got %d", tc.name, len(projects))
			}

			if len(projects) > 0 && !strings.Contains(projects[0], tc.projectName) {
				t.Errorf("%s: Project name not preserved correctly, expected to contain %s", tc.name, tc.projectName)
			}
		})
	}
}

func TestSlnxParser_UTF8BOM(t *testing.T) {
	tempDir := t.TempDir()

	// XML solution content for .slnx format
	// Note: slnx expects projects to be inside a Folder element
	slnxContent := `<?xml version="1.0" encoding="utf-8"?>
<Solution>
  <Folder>
    <Project Path="TestProject.csproj" />
  </Folder>
</Solution>`

	tests := []struct {
		name    string
		withBOM bool
	}{
		{name: "SLNX_WithBOM", withBOM: true},
		{name: "SLNX_WithoutBOM", withBOM: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solutionPath := filepath.Join(tempDir, tt.name+".slnx")

			var content []byte
			if tt.withBOM {
				content = make([]byte, len(utf8BOM))
				copy(content, utf8BOM)
				content = append(content, []byte(slnxContent)...)
			} else {
				content = []byte(slnxContent)
			}

			if err := os.WriteFile(solutionPath, content, 0644); err != nil {
				t.Fatal(err)
			}

			// Create project file
			projectPath := filepath.Join(tempDir, "TestProject.csproj")
			projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
			if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Parse the .slnx file
			parser, err := solution.GetParser(solutionPath)
			if err != nil {
				t.Fatalf("Failed to get parser: %v", err)
			}

			sol, err := parser.Parse(solutionPath)
			if err != nil {
				t.Fatalf("Failed to parse .slnx with BOM=%v: %v", tt.withBOM, err)
			}

			// Verify parsing
			projects := sol.GetProjects()
			if len(projects) != 1 {
				t.Errorf("Expected 1 project, got %d", len(projects))
			}
		})
	}
}

func TestBOMDetection(t *testing.T) {
	// Test that we can detect and handle various BOMs correctly
	testCases := []struct {
		name        string
		bom         []byte
		encoding    string
		shouldParse bool
	}{
		{
			name:        "UTF8_BOM",
			bom:         []byte{0xEF, 0xBB, 0xBF},
			encoding:    "UTF-8",
			shouldParse: true,
		},
		{
			name:        "No_BOM",
			bom:         []byte{},
			encoding:    "UTF-8",
			shouldParse: true,
		},
		{
			name:        "UTF16_LE_BOM",
			bom:         []byte{0xFF, 0xFE},
			encoding:    "UTF-16 LE",
			shouldParse: false, // We only support UTF-8 for now
		},
		{
			name:        "UTF16_BE_BOM",
			bom:         []byte{0xFE, 0xFF},
			encoding:    "UTF-16 BE",
			shouldParse: false, // We only support UTF-8 for now
		},
	}

	tempDir := t.TempDir()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			solutionPath := filepath.Join(tempDir, tc.name+".sln")

			// For UTF-16, we'd need to encode the content properly
			// For now, just test UTF-8 variants
			if tc.encoding == "UTF-8" {
				content := tc.bom
				content = append(content, []byte(`Microsoft Visual Studio Solution File, Format Version 12.00
Global
EndGlobal`)...)

				if err := os.WriteFile(solutionPath, content, 0644); err != nil {
					t.Fatal(err)
				}

				parser, err := solution.GetParser(solutionPath)
				if err != nil && tc.shouldParse {
					t.Fatalf("Failed to get parser: %v", err)
				}

				if parser != nil {
					sol, err := parser.Parse(solutionPath)
					if tc.shouldParse {
						if err != nil {
							t.Fatalf("Failed to parse %s: %v", tc.encoding, err)
						}
						if sol == nil {
							t.Error("Expected non-nil solution")
						}
					} else if err == nil {
						t.Errorf("Expected parse error for %s encoding", tc.encoding)
					}
				}
			}
		})
	}
}

// TestBOMPreservation tests that BOMs are preserved when reading and potentially modifying files
func TestBOMPreservation(t *testing.T) {
	tempDir := t.TempDir()

	// Create a solution file with BOM
	solutionPath := filepath.Join(tempDir, "WithBOM.sln")
	originalContent := make([]byte, len(utf8BOM))
	copy(originalContent, utf8BOM)
	originalContent = append(originalContent, []byte(`Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
Global
EndGlobal`)...)

	if err := os.WriteFile(solutionPath, originalContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Read the file
	data, err := os.ReadFile(solutionPath)
	if err != nil {
		t.Fatal(err)
	}

	// Check that BOM is still present
	if !bytes.HasPrefix(data, utf8BOM) {
		t.Error("BOM was not preserved when reading file")
	}

	// Parse the file
	parser, err := solution.GetParser(solutionPath)
	if err != nil {
		t.Fatal(err)
	}

	sol, err := parser.Parse(solutionPath)
	if err != nil {
		t.Fatal(err)
	}

	if sol == nil {
		t.Error("Expected non-nil solution")
	}

	// Verify the file still has BOM after parsing
	data2, err := os.ReadFile(solutionPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, data2) {
		t.Error("File content changed after parsing (BOM might be lost)")
	}
}
