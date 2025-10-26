package main

import (
	"encoding/json"
	"fmt"
)

// ExecuteCommandPairHandler executes both dotnet nuget and gonuget commands
type ExecuteCommandPairHandler struct{}

func (h *ExecuteCommandPairHandler) ErrorCode() string { return "CLI_EXEC_001" }

func (h *ExecuteCommandPairHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteCommandPairRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Execute dotnet nuget command
	dotnetResult, err := ExecuteDotnetNuget(req.DotnetCommand, req.WorkingDir, "", req.Timeout)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	// Execute gonuget command
	gonugetResult, err := ExecuteGonuget(req.GonugetCommand, req.WorkingDir, "", req.Timeout)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Normalize outputs for comparison
	normalizedDotnet := NormalizeOutput(dotnetResult.StdOut)
	normalizedGonuget := NormalizeOutput(gonugetResult.StdOut)

	return ExecuteCommandPairResponse{
		DotnetExitCode:          dotnetResult.ExitCode,
		DotnetStdOut:            dotnetResult.StdOut,
		DotnetStdErr:            dotnetResult.StdErr,
		DotnetSuccess:           dotnetResult.Success,
		GonugetExitCode:         gonugetResult.ExitCode,
		GonugetStdOut:           gonugetResult.StdOut,
		GonugetStdErr:           gonugetResult.StdErr,
		GonugetSuccess:          gonugetResult.Success,
		NormalizedDotnetStdOut:  normalizedDotnet,
		NormalizedGonugetStdOut: normalizedGonuget,
		OutputMatches:           normalizedDotnet == normalizedGonuget,
	}, nil
}

// ExecuteConfigGetHandler handles config get command
type ExecuteConfigGetHandler struct{}

func (h *ExecuteConfigGetHandler) ErrorCode() string { return "CLI_CFG_GET_001" }

func (h *ExecuteConfigGetHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteConfigGetRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build dotnet nuget command
	dotnetCmd := fmt.Sprintf("config get %s", req.Key)
	if req.ShowPath {
		dotnetCmd += " --show-path"
	}
	if req.WorkingDirFlag != "" {
		dotnetCmd += fmt.Sprintf(" --working-directory %s", req.WorkingDirFlag)
	}

	// Build gonuget command
	gonugetCmd := fmt.Sprintf("config get %s", req.Key)
	if req.ShowPath {
		gonugetCmd += " --show-path"
	}
	if req.WorkingDirFlag != "" {
		gonugetCmd += fmt.Sprintf(" --working-directory %s", req.WorkingDirFlag)
	}

	// Execute both commands
	// Note: config commands find NuGet.config in working directory hierarchy
	dotnetResult, err := ExecuteDotnetNuget(dotnetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	gonugetResult, err := ExecuteGonuget(gonugetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Normalize and compare
	normalizedDotnet := NormalizeOutput(dotnetResult.StdOut)
	normalizedGonuget := NormalizeOutput(gonugetResult.StdOut)

	return ExecuteConfigGetResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   normalizedDotnet == normalizedGonuget,
	}, nil
}

// ExecuteConfigSetHandler handles config set command
type ExecuteConfigSetHandler struct{}

func (h *ExecuteConfigSetHandler) ErrorCode() string { return "CLI_CFG_SET_001" }

func (h *ExecuteConfigSetHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteConfigSetRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build commands
	dotnetCmd := fmt.Sprintf("config set %s %s", req.Key, req.Value)
	gonugetCmd := fmt.Sprintf("config set %s %s", req.Key, req.Value)

	// Execute both commands
	// Note: config commands find NuGet.config in working directory hierarchy
	dotnetResult, err := ExecuteDotnetNuget(dotnetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	gonugetResult, err := ExecuteGonuget(gonugetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Normalize and compare
	normalizedDotnet := NormalizeOutput(dotnetResult.StdOut)
	normalizedGonuget := NormalizeOutput(gonugetResult.StdOut)

	return ExecuteConfigSetResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   normalizedDotnet == normalizedGonuget,
	}, nil
}

// ExecuteConfigUnsetHandler handles config unset command
type ExecuteConfigUnsetHandler struct{}

func (h *ExecuteConfigUnsetHandler) ErrorCode() string { return "CLI_CFG_UNSET_001" }

func (h *ExecuteConfigUnsetHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteConfigUnsetRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build commands
	dotnetCmd := fmt.Sprintf("config unset %s", req.Key)
	gonugetCmd := fmt.Sprintf("config unset %s", req.Key)

	// Execute both commands
	// Note: config commands find NuGet.config in working directory hierarchy
	dotnetResult, err := ExecuteDotnetNuget(dotnetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	gonugetResult, err := ExecuteGonuget(gonugetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Normalize and compare
	normalizedDotnet := NormalizeOutput(dotnetResult.StdOut)
	normalizedGonuget := NormalizeOutput(gonugetResult.StdOut)

	return ExecuteConfigUnsetResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   normalizedDotnet == normalizedGonuget,
	}, nil
}

// ExecuteConfigPathsHandler handles config paths command
type ExecuteConfigPathsHandler struct{}

func (h *ExecuteConfigPathsHandler) ErrorCode() string { return "CLI_CFG_PATHS_001" }

func (h *ExecuteConfigPathsHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteConfigPathsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build commands
	dotnetCmd := "config paths"
	gonugetCmd := "config paths"

	if req.WorkingDirFlag != "" {
		dotnetCmd += fmt.Sprintf(" --working-directory %s", req.WorkingDirFlag)
		gonugetCmd += fmt.Sprintf(" --working-directory %s", req.WorkingDirFlag)
	}

	// Execute both commands
	dotnetResult, err := ExecuteDotnetNuget(dotnetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	gonugetResult, err := ExecuteGonuget(gonugetCmd, req.WorkingDir, "", 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Normalize and compare (paths output may have formatting differences)
	normalizedDotnet := NormalizeOutput(dotnetResult.StdOut)
	normalizedGonuget := NormalizeOutput(gonugetResult.StdOut)

	return ExecuteConfigPathsResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   normalizedDotnet == normalizedGonuget,
	}, nil
}
