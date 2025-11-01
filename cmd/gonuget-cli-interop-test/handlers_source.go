package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExecuteSourceListHandler handles list source command
type ExecuteSourceListHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExecuteSourceListHandler) ErrorCode() string { return "CLI_SRC_LIST_001" }

// Handle processes the request.
func (h *ExecuteSourceListHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteSourceListRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build dotnet nuget command (verb-first: list source)
	dotnetCmd := "list source"
	if req.ConfigFile != "" {
		dotnetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}
	if req.Format != "" {
		dotnetCmd += fmt.Sprintf(" --format %s", req.Format)
	}

	// Build gonuget command (noun-first: source list)
	gonugetCmd := "source list"
	if req.ConfigFile != "" {
		gonugetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}
	if req.Format != "" {
		gonugetCmd += fmt.Sprintf(" --format %s", req.Format)
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

	// Normalize output for comparison
	normalizedDotnet := normalizeSourceListOutput(dotnetResult.StdOut)
	normalizedGonuget := normalizeSourceListOutput(gonugetResult.StdOut)

	return ExecuteSourceListResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   normalizedDotnet == normalizedGonuget,
	}, nil
}

// ExecuteSourceAddHandler handles add source command
type ExecuteSourceAddHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExecuteSourceAddHandler) ErrorCode() string { return "CLI_SRC_ADD_001" }

// Handle processes the request.
func (h *ExecuteSourceAddHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteSourceAddRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Copy config for gonuget so both CLIs start with same state
	gonugetConfigPath, cleanup, err := copyConfigForGonuget(req.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("copy config: %w", err)
	}
	defer cleanup()

	// Build dotnet nuget command
	dotnetCmd := fmt.Sprintf("add source %s --name %s", req.Source, req.Name)
	if req.ConfigFile != "" {
		dotnetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}
	if req.Username != "" {
		dotnetCmd += fmt.Sprintf(" --username %s", req.Username)
	}
	if req.Password != "" {
		dotnetCmd += fmt.Sprintf(" --password %s", req.Password)
	}
	if req.StorePasswordInClearText {
		dotnetCmd += " --store-password-in-clear-text"
	}
	if req.ValidAuthenticationTypes != "" {
		dotnetCmd += fmt.Sprintf(" --valid-authentication-types %s", req.ValidAuthenticationTypes)
	}
	if req.ProtocolVersion != "" {
		dotnetCmd += fmt.Sprintf(" --protocol-version %s", req.ProtocolVersion)
	}
	if req.AllowInsecureConnections {
		dotnetCmd += " --allow-insecure-connections"
	}

	// Build gonuget command (noun-first: source add, with copied config)
	gonugetCmd := fmt.Sprintf("source add %s --name %s", req.Source, req.Name)
	if gonugetConfigPath != "" {
		gonugetCmd += fmt.Sprintf(" --configfile %s", gonugetConfigPath)
	}
	if req.Username != "" {
		gonugetCmd += fmt.Sprintf(" --username %s", req.Username)
	}
	if req.Password != "" {
		gonugetCmd += fmt.Sprintf(" --password %s", req.Password)
	}
	if req.StorePasswordInClearText {
		gonugetCmd += " --store-password-in-clear-text"
	}
	if req.ValidAuthenticationTypes != "" {
		gonugetCmd += fmt.Sprintf(" --valid-authentication-types %s", req.ValidAuthenticationTypes)
	}
	if req.ProtocolVersion != "" {
		gonugetCmd += fmt.Sprintf(" --protocol-version %s", req.ProtocolVersion)
	}
	if req.AllowInsecureConnections {
		gonugetCmd += " --allow-insecure-connections"
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

	return ExecuteSourceAddResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   dotnetResult.ExitCode == gonugetResult.ExitCode,
	}, nil
}

// ExecuteSourceRemoveHandler handles remove source command
type ExecuteSourceRemoveHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExecuteSourceRemoveHandler) ErrorCode() string { return "CLI_SRC_REM_001" }

// Handle processes the request.
func (h *ExecuteSourceRemoveHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteSourceRemoveRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Copy config for gonuget so both CLIs start with same state
	gonugetConfigPath, cleanup, err := copyConfigForGonuget(req.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("copy config: %w", err)
	}
	defer cleanup()

	// Build dotnet nuget command (dotnet uses positional arg for name)
	dotnetCmd := fmt.Sprintf("remove source %s", req.Name)
	if req.ConfigFile != "" {
		dotnetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}

	// Build gonuget command (noun-first: source remove, with copied config)
	gonugetCmd := fmt.Sprintf("source remove %s", req.Name)
	if gonugetConfigPath != "" {
		gonugetCmd += fmt.Sprintf(" --configfile %s", gonugetConfigPath)
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

	return ExecuteSourceRemoveResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   dotnetResult.ExitCode == gonugetResult.ExitCode,
	}, nil
}

// ExecuteSourceEnableHandler handles enable source command
type ExecuteSourceEnableHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExecuteSourceEnableHandler) ErrorCode() string { return "CLI_SRC_EN_001" }

// Handle processes the request.
func (h *ExecuteSourceEnableHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteSourceEnableRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build dotnet nuget command (dotnet uses positional arg for name)
	dotnetCmd := fmt.Sprintf("enable source %s", req.Name)
	if req.ConfigFile != "" {
		dotnetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}

	// Build gonuget command (noun-first: source enable)
	gonugetCmd := fmt.Sprintf("source enable %s", req.Name)
	if req.ConfigFile != "" {
		gonugetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
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

	return ExecuteSourceEnableResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   dotnetResult.ExitCode == gonugetResult.ExitCode,
	}, nil
}

// ExecuteSourceDisableHandler handles disable source command
type ExecuteSourceDisableHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExecuteSourceDisableHandler) ErrorCode() string { return "CLI_SRC_DIS_001" }

// Handle processes the request.
func (h *ExecuteSourceDisableHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteSourceDisableRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build dotnet nuget command (dotnet uses positional arg for name)
	dotnetCmd := fmt.Sprintf("disable source %s", req.Name)
	if req.ConfigFile != "" {
		dotnetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}

	// Build gonuget command (noun-first: source disable)
	gonugetCmd := fmt.Sprintf("source disable %s", req.Name)
	if req.ConfigFile != "" {
		gonugetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
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

	return ExecuteSourceDisableResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   dotnetResult.ExitCode == gonugetResult.ExitCode,
	}, nil
}

// ExecuteSourceUpdateHandler handles update source command
type ExecuteSourceUpdateHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExecuteSourceUpdateHandler) ErrorCode() string { return "CLI_SRC_UPD_001" }

// Handle processes the request.
func (h *ExecuteSourceUpdateHandler) Handle(data json.RawMessage) (any, error) {
	var req ExecuteSourceUpdateRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build dotnet nuget command (verb-first: update source)
	dotnetCmd := fmt.Sprintf("update source %s", req.Name)
	if req.Source != "" {
		dotnetCmd += fmt.Sprintf(" --source %s", req.Source)
	}
	if req.ConfigFile != "" {
		dotnetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}
	if req.Username != "" {
		dotnetCmd += fmt.Sprintf(" --username %s", req.Username)
	}
	if req.Password != "" {
		dotnetCmd += fmt.Sprintf(" --password %s", req.Password)
	}
	if req.StorePasswordInClearText {
		dotnetCmd += " --store-password-in-clear-text"
	}
	if req.ValidAuthenticationTypes != "" {
		dotnetCmd += fmt.Sprintf(" --valid-authentication-types %s", req.ValidAuthenticationTypes)
	}
	if req.ProtocolVersion != "" {
		dotnetCmd += fmt.Sprintf(" --protocol-version %s", req.ProtocolVersion)
	}
	if req.AllowInsecureConnections {
		dotnetCmd += " --allow-insecure-connections"
	}

	// Build gonuget command (noun-first: source update)
	gonugetCmd := fmt.Sprintf("source update %s", req.Name)
	if req.Source != "" {
		gonugetCmd += fmt.Sprintf(" --source %s", req.Source)
	}
	if req.ConfigFile != "" {
		gonugetCmd += fmt.Sprintf(" --configfile %s", req.ConfigFile)
	}
	if req.Username != "" {
		gonugetCmd += fmt.Sprintf(" --username %s", req.Username)
	}
	if req.Password != "" {
		gonugetCmd += fmt.Sprintf(" --password %s", req.Password)
	}
	if req.StorePasswordInClearText {
		gonugetCmd += " --store-password-in-clear-text"
	}
	if req.ValidAuthenticationTypes != "" {
		gonugetCmd += fmt.Sprintf(" --valid-authentication-types %s", req.ValidAuthenticationTypes)
	}
	if req.ProtocolVersion != "" {
		gonugetCmd += fmt.Sprintf(" --protocol-version %s", req.ProtocolVersion)
	}
	if req.AllowInsecureConnections {
		gonugetCmd += " --allow-insecure-connections"
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

	return ExecuteSourceUpdateResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetStdOut:    dotnetResult.StdOut,
		GonugetStdOut:   gonugetResult.StdOut,
		DotnetStdErr:    dotnetResult.StdErr,
		GonugetStdErr:   gonugetResult.StdErr,
		OutputMatches:   dotnetResult.ExitCode == gonugetResult.ExitCode,
	}, nil
}

// normalizeSourceListOutput normalizes source list output for comparison
func normalizeSourceListOutput(output string) string {
	// Normalize whitespace
	normalized := strings.TrimSpace(output)
	// Remove multiple spaces
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

// copyConfigForGonuget creates a temporary copy of a config file for gonuget to use.
// This ensures both dotnet and gonuget start with the same initial config state.
// Returns the path to the copy and a cleanup function.
func copyConfigForGonuget(originalPath string) (string, func(), error) {
	if originalPath == "" {
		return "", func() {}, nil
	}

	// Create temp file in same directory to ensure same hierarchy behavior
	dir := filepath.Dir(originalPath)
	tmpFile, err := os.CreateTemp(dir, "NuGet.config.gonuget.*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Copy original to temp
	src, err := os.Open(originalPath)
	if err != nil {
		os.Remove(tmpPath)
		return "", func() {}, fmt.Errorf("open original config: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return "", func() {}, fmt.Errorf("create temp config: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(tmpPath)
		return "", func() {}, fmt.Errorf("copy config: %w", err)
	}

	cleanup := func() {
		os.Remove(tmpPath)
	}

	return tmpPath, cleanup, nil
}
