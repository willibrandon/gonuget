package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExecuteVersionHandler handles version command
type ExecuteVersionHandler struct{}

func (h *ExecuteVersionHandler) ErrorCode() string { return "CLI_VER_001" }

func (h *ExecuteVersionHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteVersionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Execute dotnet nuget --version
	dotnetResult, err := ExecuteDotnetNuget("--version", req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	// Execute gonuget version
	gonugetResult, err := ExecuteGonuget("version", req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Check if both outputs contain version information
	outputFormatSimilar := containsVersionInfo(dotnetResult.StdOut) && containsVersionInfo(gonugetResult.StdOut)

	return ExecuteVersionResponse{
		DotnetExitCode:      dotnetResult.ExitCode,
		GonugetExitCode:     gonugetResult.ExitCode,
		DotnetStdOut:        dotnetResult.StdOut,
		GonugetStdOut:       gonugetResult.StdOut,
		DotnetStdErr:        dotnetResult.StdErr,
		GonugetStdErr:       gonugetResult.StdErr,
		ExitCodesMatch:      dotnetResult.ExitCode == gonugetResult.ExitCode,
		OutputFormatSimilar: outputFormatSimilar,
	}, nil
}

// containsVersionInfo checks if output contains version information
func containsVersionInfo(output string) bool {
	output = strings.ToLower(output)
	// Check for version number patterns
	return strings.Contains(output, "version") ||
		strings.Contains(output, "nuget") ||
		strings.Contains(output, "gonuget") ||
		// Check for version pattern like "1.2.3" or "1.2.3.4"
		strings.ContainsAny(output, "0123456789.")
}
