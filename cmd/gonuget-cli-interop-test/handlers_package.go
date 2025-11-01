package main

import (
	"encoding/json"
	"fmt"
)

// ExecuteAddPackageHandler handles execute_add_package requests.
type ExecuteAddPackageHandler struct{}

// Handle processes the request.
func (h *ExecuteAddPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	return HandleExecuteAddPackage(data)
}

// ErrorCode returns the error code prefix for this handler.
func (h *ExecuteAddPackageHandler) ErrorCode() string {
	return "PKG_001"
}

// HandleExecuteAddPackage executes both dotnet add package and gonuget add package.
func HandleExecuteAddPackage(data json.RawMessage) (interface{}, error) {
	var req struct {
		ProjectPath string `json:"projectPath"`
		PackageID   string `json:"packageId"`
		WorkingDir  string `json:"workingDir"`
		Version     string `json:"version,omitempty"`
		Source      string `json:"source,omitempty"`
		NoRestore   bool   `json:"noRestore,omitempty"`
		Prerelease  bool   `json:"prerelease,omitempty"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Build dotnet add package command
	dotnetArgs := []string{"add", req.ProjectPath, "package", req.PackageID}
	if req.Version != "" {
		dotnetArgs = append(dotnetArgs, "--version", req.Version)
	}
	if req.Source != "" {
		dotnetArgs = append(dotnetArgs, "--source", req.Source)
	}
	if req.NoRestore {
		dotnetArgs = append(dotnetArgs, "--no-restore")
	}
	if req.Prerelease {
		dotnetArgs = append(dotnetArgs, "--prerelease")
	}

	// Execute dotnet add package
	dotnetResult, err := ExecuteCommand("dotnet", dotnetArgs, req.WorkingDir, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to execute dotnet: %w", err)
	}

	// Build gonuget package add command (noun-first)
	gonugetArgs := []string{"package", "add", req.PackageID, "--project", req.ProjectPath}
	if req.Version != "" {
		gonugetArgs = append(gonugetArgs, "--version", req.Version)
	}
	if req.Source != "" {
		gonugetArgs = append(gonugetArgs, "--source", req.Source)
	}
	if req.NoRestore {
		gonugetArgs = append(gonugetArgs, "--no-restore")
	}
	if req.Prerelease {
		gonugetArgs = append(gonugetArgs, "--prerelease")
	}

	// Find gonuget executable
	gonugetExe := findGonugetExecutable()

	// Execute gonuget add package
	gonugetResult, err := ExecuteCommand(gonugetExe, gonugetArgs, req.WorkingDir, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to execute gonuget: %w", err)
	}

	return ExecuteAddPackageResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetExitCode: gonugetResult.ExitCode,
		GonugetStdOut:   gonugetResult.StdOut,
		GonugetStdErr:   gonugetResult.StdErr,
	}, nil
}
