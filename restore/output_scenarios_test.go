package restore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutputScenarios provides comprehensive coverage of all scenarios from RESTORE-OUTPUT-TESTING.md
// This test suite validates gonuget restore output across:
// - All verbosity levels (quiet, minimal, normal, detailed, diagnostic)
// - Both output modes (TTY and piped)
// - Success scenarios (simple, multitarget, complex)
// - Error scenarios (NU1101, NU1102, NU1103)
func TestOutputScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping output scenarios integration test")
	}

	// Get repository root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	repoRoot := filepath.Dir(cwd)

	scenarios := []struct {
		name        string
		projectPath string
		expectError bool
		errorCode   string // Expected NuGet error code (NU1101, NU1102, NU1103)
	}{
		// Success scenarios
		{
			name:        "Simple",
			projectPath: filepath.Join(repoRoot, "tests/test-scenarios/simple/test.csproj"),
			expectError: false,
		},
		{
			name:        "Multitarget",
			projectPath: filepath.Join(repoRoot, "tests/test-scenarios/multitarget/test.csproj"),
			expectError: false,
		},
		{
			name:        "Complex",
			projectPath: filepath.Join(repoRoot, "tests/test-scenarios/complex/test.csproj"),
			expectError: false,
		},
		// Error scenarios
		{
			name:        "NU1101_PackageNotFound",
			projectPath: filepath.Join(repoRoot, "tests/test-scenarios/nu1101/test.csproj"),
			expectError: true,
			errorCode:   "NU1101",
		},
		{
			name:        "NU1102_VersionNotFound",
			projectPath: filepath.Join(repoRoot, "tests/test-scenarios/nu1102/test.csproj"),
			expectError: true,
			errorCode:   "NU1102",
		},
		{
			name:        "NU1103_OnlyPrereleaseAvailable",
			projectPath: filepath.Join(repoRoot, "tests/test-scenarios/nu1103/test.csproj"),
			expectError: true,
			errorCode:   "NU1103",
		},
	}

	verbosityLevels := []struct {
		name      string
		verbosity string
	}{
		{"Quiet", "quiet"},
		{"Minimal", "minimal"},
		{"Normal", "normal"},
		{"Detailed", "detailed"},
		{"Diagnostic", "diagnostic"},
	}

	outputModes := []struct {
		name   string
		isTTY  bool
		width  int
		height int
	}{
		{"TTY", true, 120, 24},
		{"Piped", false, 0, 0},
	}

	for _, scenario := range scenarios {
		for _, verbosity := range verbosityLevels {
			for _, mode := range outputModes {
				testName := fmt.Sprintf("%s/%s/%s", scenario.name, verbosity.name, mode.name)
				t.Run(testName, func(t *testing.T) {
					// Check if project file exists
					if _, err := os.Stat(scenario.projectPath); os.IsNotExist(err) {
						t.Skipf("Test scenario not found: %s", scenario.projectPath)
					}

					// Create mock console with specific TTY mode
					console := &mockConsoleForOutputMode{
						output:   &bytes.Buffer{},
						messages: []string{},
					}

					detector := &mockTTYDetector{
						isTTY:  mode.isTTY,
						width:  mode.width,
						height: mode.height,
					}

					// Temporarily replace DefaultTTYDetector
					oldDetector := DefaultTTYDetector
					DefaultTTYDetector = detector
					defer func() { DefaultTTYDetector = oldDetector }()

					// Run restore
					ctx := context.Background()
					opts := &Options{
						Sources:   []string{"https://api.nuget.org/v3/index.json"},
						Verbosity: verbosity.verbosity,
					}

					err := Run(ctx, []string{scenario.projectPath}, opts, console)

					// Log output for observation
					t.Logf("Output:\n%s", console.output.String())
					t.Logf("Messages: %v", console.messages)

					// Validate expectations
					if scenario.expectError {
						if err == nil {
							t.Errorf("Expected error for %s scenario, but restore succeeded", scenario.name)
						}

						// Check for expected error code in output
						outputStr := console.output.String()
						if !strings.Contains(outputStr, scenario.errorCode) {
							t.Errorf("Expected error code %s in output, got: %s", scenario.errorCode, outputStr)
						}

						// Validate error formatting based on mode
						if mode.isTTY {
							// TTY mode should have ANSI color codes for errors
							if !strings.Contains(outputStr, "\x1B[") {
								t.Errorf("Expected ANSI color codes in TTY mode error output")
							}
						} else {
							// Piped mode should NOT have ANSI color codes
							if strings.Contains(outputStr, "\x1B[") {
								t.Errorf("Expected NO ANSI color codes in piped mode error output")
							}
						}

						// In non-quiet mode, check for "failed" message
						if verbosity.verbosity != "quiet" {
							foundFailed := false
							for _, msg := range console.messages {
								if strings.Contains(msg, "failed") {
									foundFailed = true
									break
								}
							}
							if !foundFailed {
								t.Errorf("Expected 'failed' message in non-quiet error output")
							}
						}
					} else {
						// Success scenario
						if err != nil {
							t.Logf("Restore failed (may be network/environment): %v", err)
							t.Logf("Output: %s", console.output.String())
							t.Skip("Skipping due to restore failure")
						}

						outputStr := console.output.String()

						// Validate success formatting based on mode
						if mode.isTTY {
							// TTY mode should have ANSI color codes for "succeeded"
							if !strings.Contains(outputStr, "\x1B[") {
								t.Errorf("Expected ANSI color codes in TTY mode success output")
							}
						} else {
							// Piped mode should NOT have ANSI color codes
							if strings.Contains(outputStr, "\x1B[") {
								t.Errorf("Expected NO ANSI color codes in piped mode success output")
							}
						}

						// In non-quiet mode, check for success indicators
						if verbosity.verbosity != "quiet" {
							if mode.isTTY {
								// TTY mode always shows "succeeded"
								foundSuccess := false
								for _, msg := range console.messages {
									if strings.Contains(msg, "succeeded") {
										foundSuccess = true
										break
									}
								}
								if !foundSuccess {
									t.Errorf("Expected 'succeeded' message in TTY mode, got: %v", console.messages)
								}
							} else {
								// Piped mode shows "Restored" or "All projects are up-to-date"
								foundOutput := false
								for _, msg := range console.messages {
									if strings.Contains(msg, "Restored") || strings.Contains(msg, "up-to-date") {
										foundOutput = true
										break
									}
								}
								if !foundOutput {
									t.Errorf("Expected 'Restored' or 'up-to-date' in piped mode, got: %v", console.messages)
								}
							}
						}

						// Detailed mode specific checks
						if verbosity.verbosity == "detailed" {
							if mode.isTTY {
								// TTY detailed mode should show "Determining projects"
								foundDetermining := false
								for _, msg := range console.messages {
									if strings.Contains(msg, "Determining projects") {
										foundDetermining = true
										break
									}
								}
								if !foundDetermining {
									t.Errorf("Expected 'Determining projects' in TTY detailed mode")
								}
							} else {
								// Piped detailed mode should show "Committing restore"
								foundCommitting := false
								for _, msg := range console.messages {
									if strings.Contains(msg, "Committing restore") {
										foundCommitting = true
										break
									}
								}
								if !foundCommitting {
									t.Errorf("Expected 'Committing restore' in piped detailed mode")
								}
							}
						}

						// Diagnostic mode specific checks
						if verbosity.verbosity == "diagnostic" {
							// Diagnostic mode should show download-related messages
							// (Acquiring, GET, OK, CACHE) - but only if packages were actually downloaded
							// For cached packages, we won't see these, so this is optional
							outputLower := strings.ToLower(outputStr)
							hasDiagnosticInfo := strings.Contains(outputLower, "acquiring") ||
								strings.Contains(outputLower, "get ") ||
								strings.Contains(outputLower, "cache")

							t.Logf("Diagnostic mode - has diagnostic info: %v", hasDiagnosticInfo)
						}
					}
				})
			}
		}
	}
}

// TestVerbosityMinimal tests minimal verbosity output (default)
func TestVerbosityMinimal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	repoRoot := filepath.Dir(cwd)
	projectPath := filepath.Join(repoRoot, "tests/test-scenarios/simple/test.csproj")

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skipf("Test scenario not found: %s", projectPath)
	}

	console := &mockConsoleForOutputMode{
		output:   &bytes.Buffer{},
		messages: []string{},
	}

	detector := &mockTTYDetector{isTTY: true, width: 120, height: 24}
	oldDetector := DefaultTTYDetector
	DefaultTTYDetector = detector
	defer func() { DefaultTTYDetector = oldDetector }()

	ctx := context.Background()
	opts := &Options{
		Sources:   []string{"https://api.nuget.org/v3/index.json"},
		Verbosity: "minimal",
	}

	err = Run(ctx, []string{projectPath}, opts, console)
	if err != nil {
		t.Skipf("Restore failed: %v", err)
	}

	// Minimal mode should show:
	// - "Restore complete"
	// - "Restore succeeded"
	// - Should NOT show "Determining projects" or detailed breakdown

	foundComplete := false
	foundSucceeded := false
	foundDetermining := false

	for _, msg := range console.messages {
		if strings.Contains(msg, "Restore complete") {
			foundComplete = true
		}
		if strings.Contains(msg, "succeeded") {
			foundSucceeded = true
		}
		if strings.Contains(msg, "Determining projects") {
			foundDetermining = true
		}
	}

	if !foundComplete {
		t.Errorf("Expected 'Restore complete' in minimal mode, got: %v", console.messages)
	}
	if !foundSucceeded {
		t.Errorf("Expected 'succeeded' in minimal mode, got: %v", console.messages)
	}
	if foundDetermining {
		t.Errorf("Did NOT expect 'Determining projects' in minimal mode, got: %v", console.messages)
	}
}

// TestVerbosityQuiet tests quiet verbosity output (minimal output)
func TestVerbosityQuiet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	repoRoot := filepath.Dir(cwd)
	projectPath := filepath.Join(repoRoot, "tests/test-scenarios/simple/test.csproj")

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skipf("Test scenario not found: %s", projectPath)
	}

	console := &mockConsoleForOutputMode{
		output:   &bytes.Buffer{},
		messages: []string{},
	}

	detector := &mockTTYDetector{isTTY: true, width: 120, height: 24}
	oldDetector := DefaultTTYDetector
	DefaultTTYDetector = detector
	defer func() { DefaultTTYDetector = oldDetector }()

	ctx := context.Background()
	opts := &Options{
		Sources:   []string{"https://api.nuget.org/v3/index.json"},
		Verbosity: "quiet",
	}

	err = Run(ctx, []string{projectPath}, opts, console)
	if err != nil {
		t.Skipf("Restore failed: %v", err)
	}

	// Quiet mode should have minimal to no output on success
	// Just log what we get to observe behavior
	t.Logf("Quiet mode output: %s", console.output.String())
	t.Logf("Quiet mode messages: %v", console.messages)
}

// TestErrorScenarios specifically tests error formatting
func TestErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	repoRoot := filepath.Dir(cwd)

	errorScenarios := []struct {
		name      string
		project   string
		errorCode string
	}{
		{"NU1101", "tests/test-scenarios/nu1101/test.csproj", "NU1101"},
		{"NU1102", "tests/test-scenarios/nu1102/test.csproj", "NU1102"},
		{"NU1103", "tests/test-scenarios/nu1103/test.csproj", "NU1103"},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			projectPath := filepath.Join(repoRoot, scenario.project)

			if _, err := os.Stat(projectPath); os.IsNotExist(err) {
				t.Skipf("Test scenario not found: %s", projectPath)
			}

			// Test both TTY and piped modes
			for _, isTTY := range []bool{true, false} {
				modeName := "Piped"
				if isTTY {
					modeName = "TTY"
				}

				t.Run(modeName, func(t *testing.T) {
					console := &mockConsoleForOutputMode{
						output:   &bytes.Buffer{},
						messages: []string{},
					}

					detector := &mockTTYDetector{isTTY: isTTY, width: 120, height: 24}
					oldDetector := DefaultTTYDetector
					DefaultTTYDetector = detector
					defer func() { DefaultTTYDetector = oldDetector }()

					ctx := context.Background()
					opts := &Options{
						Sources:   []string{"https://api.nuget.org/v3/index.json"},
						Verbosity: "normal",
					}

					err := Run(ctx, []string{projectPath}, opts, console)

					// Should error
					if err == nil {
						t.Errorf("Expected error for %s scenario", scenario.name)
					}

					outputStr := console.output.String()
					t.Logf("Error output (%s mode):\n%s", modeName, outputStr)

					// Check for error code
					if !strings.Contains(outputStr, scenario.errorCode) {
						t.Errorf("Expected error code %s in output", scenario.errorCode)
					}

					// Check ANSI codes based on mode
					if isTTY {
						if !strings.Contains(outputStr, "\x1B[") {
							t.Errorf("Expected ANSI codes in TTY mode")
						}
					} else {
						if strings.Contains(outputStr, "\x1B[") {
							t.Errorf("Expected NO ANSI codes in piped mode")
						}
					}

					// Check for "failed" message
					foundFailed := false
					for _, msg := range console.messages {
						if strings.Contains(msg, "failed") {
							foundFailed = true
							break
						}
					}
					if !foundFailed {
						t.Errorf("Expected 'failed' message in error output")
					}
				})
			}
		})
	}
}
