package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
)

func TestPackageAdd_RejectsSolutionFile(t *testing.T) {
	tests := []struct {
		name        string
		extension   string
		expectedErr string
	}{
		{
			name:        ".sln file",
			extension:   ".sln",
			expectedErr: "Couldn't find a project to run",
		},
		{
			name:        ".slnx file",
			extension:   ".slnx",
			expectedErr: "Couldn't find a project to run",
		},
		{
			name:        ".slnf file",
			extension:   ".slnf",
			expectedErr: "Couldn't find a project to run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tempDir := t.TempDir()
			solutionPath := filepath.Join(tempDir, "TestSolution"+tt.extension)

			// Create the solution file
			if err := os.WriteFile(solutionPath, []byte("test content"), 0644); err != nil {
				t.Fatal(err)
			}

			// Create package add command
			cmd := commands.NewPackageAddCommand()

			// Capture output
			outputBuffer := &bytes.Buffer{}
			cmd.SetOut(outputBuffer)
			cmd.SetErr(outputBuffer)

			// Set arguments to the solution file and a package
			cmd.SetArgs([]string{"Newtonsoft.Json", "--project", solutionPath})

			// Execute command - should fail
			err := cmd.Execute()
			if err == nil {
				t.Fatal("Expected error when adding package to solution file, got nil")
			}

			// Verify error message contains expected text
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedErr, err.Error())
			}

			// Verify error message contains the directory path
			if !strings.Contains(err.Error(), tempDir) {
				t.Errorf("Error message should contain directory path %q, got %q", tempDir, err.Error())
			}
		})
	}
}

func TestPackageAdd_RejectsSolutionFileAsPositionalArg(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	solutionPath := filepath.Join(tempDir, "TestSolution.sln")

	// Create a minimal .sln file
	solutionContent := `
Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
Global
EndGlobal
`
	if err := os.WriteFile(solutionPath, []byte(solutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create package add command
	cmd := commands.NewPackageAddCommand()

	// Capture output
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)

	// Set arguments - no --project flag, should auto-detect solution
	cmd.SetArgs([]string{"Newtonsoft.Json"})

	// Execute command - should fail with proper error
	err = cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when adding package to auto-detected solution file, got nil")
	}

	// Verify error message
	if !strings.Contains(err.Error(), "Couldn't find a project to run") {
		t.Errorf("Expected 'Couldn't find a project to run' error, got %q", err.Error())
	}

	// Verify error message contains the directory
	if !strings.Contains(err.Error(), tempDir) {
		t.Errorf("Error message should contain directory path %q, got %q", tempDir, err.Error())
	}
}

func TestPackageAdd_ErrorMessageFormat(t *testing.T) {
	// Create a temporary directory with a specific name to verify in error
	tempDir := t.TempDir()
	solutionPath := filepath.Join(tempDir, "MySolution.sln")

	// Create the solution file
	if err := os.WriteFile(solutionPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package add command
	cmd := commands.NewPackageAddCommand()

	// Capture output
	outputBuffer := &bytes.Buffer{}
	cmd.SetOut(outputBuffer)
	cmd.SetErr(outputBuffer)

	// Set arguments
	cmd.SetArgs([]string{"Moq", "--project", solutionPath})

	// Execute command
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// The error should match dotnet CLI format exactly
	expectedFormat := "Couldn't find a project to run. Ensure a project exists in"
	if !strings.Contains(err.Error(), expectedFormat) {
		t.Errorf("Error message format mismatch.\nExpected to contain: %q\nGot: %q",
			expectedFormat, err.Error())
	}

	// Should also mention --project flag
	if !strings.Contains(err.Error(), "or pass the path to the project using --project") {
		t.Error("Error message should mention --project flag option")
	}
}
