package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExecuteHelpHandler handles help command execution
type ExecuteHelpHandler struct{}

func (h *ExecuteHelpHandler) ErrorCode() string { return "CLI_HELP_001" }

func (h *ExecuteHelpHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteHelpRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Execute dotnet nuget --help or dotnet nuget <command> --help
	var dotnetCommand string
	if req.Command == "" {
		dotnetCommand = "--help"
	} else {
		dotnetCommand = req.Command + " --help"
	}

	dotnetResult, err := ExecuteDotnetNuget(dotnetCommand, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	// Execute gonuget --help or gonuget <command> --help
	var gonugetCommand string
	if req.Command == "" {
		gonugetCommand = "--help"
	} else {
		gonugetCommand = req.Command + " --help"
	}

	gonugetResult, err := ExecuteGonuget(gonugetCommand, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Analyze outputs
	bothShowCommands := containsCommandList(dotnetResult.StdOut) && containsCommandList(gonugetResult.StdOut)
	bothShowUsage := containsUsageInfo(dotnetResult.StdOut) && containsUsageInfo(gonugetResult.StdOut)
	outputFormatSimilar := bothShowCommands || bothShowUsage

	return ExecuteHelpResponse{
		DotnetExitCode:      dotnetResult.ExitCode,
		GonugetExitCode:     gonugetResult.ExitCode,
		DotnetStdOut:        dotnetResult.StdOut,
		GonugetStdOut:       gonugetResult.StdOut,
		DotnetStdErr:        dotnetResult.StdErr,
		GonugetStdErr:       gonugetResult.StdErr,
		ExitCodesMatch:      dotnetResult.ExitCode == gonugetResult.ExitCode,
		BothShowCommands:    bothShowCommands,
		BothShowUsage:       bothShowUsage,
		OutputFormatSimilar: outputFormatSimilar,
	}, nil
}

// containsCommandList checks if output contains a list of commands
func containsCommandList(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "commands:") ||
		(strings.Contains(output, "version") && strings.Contains(output, "config"))
}

// containsUsageInfo checks if output contains usage information
func containsUsageInfo(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "usage:") || strings.Contains(output, "usage")
}
