package main

import (
	"encoding/json"
	"fmt"
)

// ExecuteRestoreHandler handles execute_restore requests.
type ExecuteRestoreHandler struct{}

// Handle processes the request.
func (h *ExecuteRestoreHandler) Handle(data json.RawMessage) (interface{}, error) {
	return HandleExecuteRestore(data)
}

// ErrorCode returns the error code prefix for this handler.
func (h *ExecuteRestoreHandler) ErrorCode() string {
	return "RESTORE_001"
}

// HandleExecuteRestore executes both dotnet restore and gonuget restore.
func HandleExecuteRestore(data json.RawMessage) (interface{}, error) {
	var req struct {
		ProjectPath string `json:"projectPath"`
		WorkingDir  string `json:"workingDir"`
		Source      string `json:"source,omitempty"`
		Packages    string `json:"packages,omitempty"`
		Force       bool   `json:"force,omitempty"`
		NoCache     bool   `json:"noCache,omitempty"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Build dotnet restore command
	dotnetArgs := []string{"restore", req.ProjectPath}
	if req.Source != "" {
		dotnetArgs = append(dotnetArgs, "--source", req.Source)
	}
	if req.Packages != "" {
		dotnetArgs = append(dotnetArgs, "--packages", req.Packages)
	}
	if req.Force {
		dotnetArgs = append(dotnetArgs, "--force")
	}
	if req.NoCache {
		dotnetArgs = append(dotnetArgs, "--no-cache")
	}

	// Execute dotnet restore
	dotnetResult, err := ExecuteCommand("dotnet", dotnetArgs, req.WorkingDir, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to execute dotnet: %w", err)
	}

	// Build gonuget restore command
	gonugetArgs := []string{"restore", req.ProjectPath}
	if req.Source != "" {
		gonugetArgs = append(gonugetArgs, "--source", req.Source)
	}
	if req.Packages != "" {
		gonugetArgs = append(gonugetArgs, "--packages", req.Packages)
	}
	if req.Force {
		gonugetArgs = append(gonugetArgs, "--force")
	}
	if req.NoCache {
		gonugetArgs = append(gonugetArgs, "--no-cache")
	}

	// Find gonuget executable
	gonugetExe := findGonugetExecutable()

	// Execute gonuget restore
	gonugetResult, err := ExecuteCommand(gonugetExe, gonugetArgs, req.WorkingDir, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to execute gonuget: %w", err)
	}

	return ExecuteRestoreResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetExitCode: gonugetResult.ExitCode,
		GonugetStdOut:   gonugetResult.StdOut,
		GonugetStdErr:   gonugetResult.StdErr,
	}, nil
}
