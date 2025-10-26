package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CommandResult holds execution results from a single command
type CommandResult struct {
	ExitCode int
	StdOut   string
	StdErr   string
	Success  bool
}

// ExecuteCommand executes a command and captures output
func ExecuteCommand(executable string, args []string, workingDir string, timeout int) (*CommandResult, error) {
	if timeout == 0 {
		timeout = 30 // default 30 seconds
	}

	cmd := exec.Command(executable, args...)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, fmt.Errorf("command failed: %w", err)
			}
		}

		result := &CommandResult{
			ExitCode: exitCode,
			StdOut:   stdout.String(),
			StdErr:   stderr.String(),
			Success:  exitCode == 0,
		}

		return result, nil

	case <-time.After(time.Duration(timeout) * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("command timed out after %d seconds", timeout)
	}
}

// ExecuteDotnetNuget executes a dotnet nuget command
func ExecuteDotnetNuget(command string, workingDir string, configFile string, timeout int) (*CommandResult, error) {
	dotnetExe := "dotnet"
	if runtime.GOOS == "windows" {
		dotnetExe = "dotnet.exe"
	}

	args := []string{"nuget"}
	args = append(args, strings.Fields(command)...)

	if configFile != "" {
		args = append(args, "--configfile", configFile)
	}

	return ExecuteCommand(dotnetExe, args, workingDir, timeout)
}

// ExecuteGonuget executes a gonuget command
func ExecuteGonuget(command string, workingDir string, configFile string, timeout int) (*CommandResult, error) {
	gonugetExe := findGonugetExecutable()

	args := strings.Fields(command)

	if configFile != "" {
		args = append(args, "--configfile", configFile)
	}

	return ExecuteCommand(gonugetExe, args, workingDir, timeout)
}

// findGonugetExecutable locates the gonuget binary
func findGonugetExecutable() string {
	exeName := "gonuget"
	if runtime.GOOS == "windows" {
		exeName = "gonuget.exe"
	}

	// Check current directory
	if path, err := exec.LookPath("./" + exeName); err == nil {
		if absPath, err := filepath.Abs(path); err == nil {
			return absPath
		}
		return path
	}

	// Check parent directories (repository root)
	for i := range 10 {
		prefix := strings.Repeat("../", i)
		path := prefix + exeName
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := exec.LookPath(absPath); err == nil {
				return absPath
			}
		}
	}

	// Check PATH
	if path, err := exec.LookPath(exeName); err == nil {
		return path
	}

	// Default to just the name (will fail if not in PATH)
	return exeName
}
