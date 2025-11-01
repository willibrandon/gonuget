package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/restore"
)

// ResolveLatestVersionHandler handles version resolution requests.
type ResolveLatestVersionHandler struct{}

// ErrorCode returns the error code for version resolution failures.
func (h *ResolveLatestVersionHandler) ErrorCode() string {
	return "RESTORE_001"
}

// ResolveLatestVersionRequest represents a request to resolve the latest package version.
type ResolveLatestVersionRequest struct {
	PackageID  string `json:"packageId"`
	Source     string `json:"source"`
	Prerelease bool   `json:"prerelease"`
}

// ResolveLatestVersionResponse contains the resolved package version.
type ResolveLatestVersionResponse struct {
	Version string `json:"version"`
}

// Handle processes the version resolution request.
func (h *ResolveLatestVersionHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ResolveLatestVersionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	version, err := restore.ResolveLatestVersion(ctx, req.PackageID, &restore.ResolveLatestVersionOptions{
		Source:     req.Source,
		Prerelease: req.Prerelease,
	})
	if err != nil {
		return nil, err
	}

	return ResolveLatestVersionResponse{Version: version}, nil
}

// ParseLockFileHandler handles lock file parsing.
type ParseLockFileHandler struct{}

// ErrorCode returns the error code for lock file parsing failures.
func (h *ParseLockFileHandler) ErrorCode() string {
	return "RESTORE_002"
}

// ParseLockFileRequest represents a request to parse a project.assets.json lock file.
type ParseLockFileRequest struct {
	LockFilePath string `json:"lockFilePath"`
}

// ParseLockFileResponse contains the parsed lock file data.
type ParseLockFileResponse struct {
	Version                     int                      `json:"version"`
	Targets                     map[string]Target        `json:"targets"`
	Libraries                   map[string]Library       `json:"libraries"`
	ProjectFileDependencyGroups map[string][]string      `json:"projectFileDependencyGroups"`
	PackageFolders              map[string]PackageFolder `json:"packageFolders"`
	Project                     ProjectInfo              `json:"project"`
}

// Target represents a target framework in the lock file.
type Target struct{}

// Library represents a library entry in the lock file.
type Library struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// PackageFolder represents a package folder path.
type PackageFolder struct{}

// ProjectInfo contains project metadata from the lock file.
type ProjectInfo struct {
	Version    string                   `json:"version"`
	Restore    RestoreInfo              `json:"restore"`
	Frameworks map[string]FrameworkInfo `json:"frameworks"`
}

// RestoreInfo contains restore-specific project information.
type RestoreInfo struct {
	ProjectUniqueName        string                          `json:"projectUniqueName"`
	ProjectName              string                          `json:"projectName"`
	ProjectPath              string                          `json:"projectPath"`
	PackagesPath             string                          `json:"packagesPath"`
	OutputPath               string                          `json:"outputPath"`
	ProjectStyle             string                          `json:"projectStyle"`
	Sources                  map[string]SourceInfo           `json:"sources"`
	FallbackFolders          []string                        `json:"fallbackFolders"`
	ConfigFilePaths          []string                        `json:"configFilePaths"`
	OriginalTargetFrameworks []string                        `json:"originalTargetFrameworks"`
	Frameworks               map[string]RestoreFrameworkInfo `json:"frameworks"`
}

// SourceInfo contains package source metadata.
type SourceInfo struct{}

// RestoreFrameworkInfo contains framework-specific restore information.
type RestoreFrameworkInfo struct {
	TargetAlias       string                 `json:"targetAlias"`
	ProjectReferences map[string]interface{} `json:"projectReferences"`
}

// FrameworkInfo contains framework-specific dependency information.
type FrameworkInfo struct {
	TargetAlias  string                    `json:"targetAlias"`
	Dependencies map[string]DependencyInfo `json:"dependencies"`
}

// DependencyInfo represents a package dependency.
type DependencyInfo struct {
	Target  string `json:"target"`
	Version string `json:"version"`
}

// Handle processes the lock file parsing request.
func (h *ParseLockFileHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseLockFileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Read the lock file
	lockFileData, err := os.ReadFile(req.LockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	// Parse into our LockFile structure
	var lockFile restore.LockFile
	if err := json.Unmarshal(lockFileData, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	// Convert to response format
	response := ParseLockFileResponse{
		Version:                     lockFile.Version,
		Targets:                     make(map[string]Target),
		Libraries:                   make(map[string]Library),
		ProjectFileDependencyGroups: lockFile.ProjectFileDependencyGroups,
		PackageFolders:              make(map[string]PackageFolder),
		Project: ProjectInfo{
			Version: lockFile.Project.Version,
			Restore: RestoreInfo{
				ProjectUniqueName:        lockFile.Project.Restore.ProjectUniqueName,
				ProjectName:              lockFile.Project.Restore.ProjectName,
				ProjectPath:              lockFile.Project.Restore.ProjectPath,
				PackagesPath:             lockFile.Project.Restore.PackagesPath,
				OutputPath:               lockFile.Project.Restore.OutputPath,
				ProjectStyle:             lockFile.Project.Restore.ProjectStyle,
				Sources:                  make(map[string]SourceInfo),
				FallbackFolders:          lockFile.Project.Restore.FallbackFolders,
				ConfigFilePaths:          lockFile.Project.Restore.ConfigFilePaths,
				OriginalTargetFrameworks: lockFile.Project.Restore.OriginalTargetFrameworks,
				Frameworks:               make(map[string]RestoreFrameworkInfo),
			},
			Frameworks: make(map[string]FrameworkInfo),
		},
	}

	// Convert targets
	for k := range lockFile.Targets {
		response.Targets[k] = Target{}
	}

	// Convert libraries
	for k, v := range lockFile.Libraries {
		response.Libraries[k] = Library{
			Type: v.Type,
			Path: v.Path,
		}
	}

	// Convert package folders
	for k := range lockFile.PackageFolders {
		response.PackageFolders[k] = PackageFolder{}
	}

	// Convert sources
	for k := range lockFile.Project.Restore.Sources {
		response.Project.Restore.Sources[k] = SourceInfo{}
	}

	// Convert restore frameworks
	for k, v := range lockFile.Project.Restore.Frameworks {
		response.Project.Restore.Frameworks[k] = RestoreFrameworkInfo{
			TargetAlias:       v.TargetAlias,
			ProjectReferences: v.ProjectReferences,
		}
	}

	// Convert frameworks
	for k, v := range lockFile.Project.Frameworks {
		deps := make(map[string]DependencyInfo)
		for depK, depV := range v.Dependencies {
			deps[depK] = DependencyInfo{
				Target:  depV.Target,
				Version: depV.Version,
			}
		}
		response.Project.Frameworks[k] = FrameworkInfo{
			TargetAlias:  v.TargetAlias,
			Dependencies: deps,
		}
	}

	return response, nil
}

// RestoreDirectDependenciesHandler handles restore operations.
type RestoreDirectDependenciesHandler struct{}

// ErrorCode returns the error code for restore failures.
func (h *RestoreDirectDependenciesHandler) ErrorCode() string {
	return "RESTORE_003"
}

// RestoreDirectDependenciesRequest represents a request to restore package dependencies.
type RestoreDirectDependenciesRequest struct {
	ProjectPath    string   `json:"projectPath"`
	PackagesFolder string   `json:"packagesFolder"`
	Sources        []string `json:"sources"`
	NoCache        bool     `json:"noCache"`
	Force          bool     `json:"force"`
}

// RestoreDirectDependenciesResponse contains the results of a restore operation.
type RestoreDirectDependenciesResponse struct {
	Success           bool     `json:"success"`
	LockFilePath      string   `json:"lockFilePath"`
	ElapsedMs         int64    `json:"elapsedMs"`
	InstalledPackages []string `json:"installedPackages"`
}

// Handle processes the restore request.
func (h *RestoreDirectDependenciesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req RestoreDirectDependenciesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create temporary console for restore output
	console := &silentConsole{}

	// Create restore options
	opts := &restore.Options{
		Sources:        req.Sources,
		PackagesFolder: req.PackagesFolder,
		Force:          req.Force,
		NoCache:        req.NoCache,
	}

	// Execute restore
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := restore.Run(ctx, []string{req.ProjectPath}, opts, console)
	elapsed := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("restore failed: %w", err)
	}

	// Read project to get lock file path
	proj, err := project.LoadProject(req.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	lockFilePath := filepath.Join(filepath.Dir(proj.Path), "obj", "project.assets.json")

	// Parse lock file to get installed packages
	var installedPackages []string
	if lockFileData, err := os.ReadFile(lockFilePath); err == nil {
		var lockFile restore.LockFile
		if err := json.Unmarshal(lockFileData, &lockFile); err == nil {
			for libKey := range lockFile.Libraries {
				installedPackages = append(installedPackages, libKey)
			}
		}
	}

	return RestoreDirectDependenciesResponse{
		Success:           true,
		LockFilePath:      lockFilePath,
		ElapsedMs:         elapsed.Milliseconds(),
		InstalledPackages: installedPackages,
	}, nil
}

// silentConsole is a console that discards all output
type silentConsole struct{}

func (c *silentConsole) Printf(format string, args ...any)  {}
func (c *silentConsole) Error(format string, args ...any)   {}
func (c *silentConsole) Warning(format string, args ...any) {}
func (c *silentConsole) Output() io.Writer                  { return io.Discard }

// RestoreTransitiveHandler handles restore operations with full transitive dependency categorization.
type RestoreTransitiveHandler struct{}

// ErrorCode returns the error code for restore failures.
func (h *RestoreTransitiveHandler) ErrorCode() string {
	return "RESTORE_TRANSITIVE_001"
}

// Handle processes the restore transitive request.
func (h *RestoreTransitiveHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req RestoreTransitiveRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create temporary console for restore output
	console := &silentConsole{}

	// Create restore options
	opts := &restore.Options{
		Sources:        req.Sources,
		PackagesFolder: req.PackagesFolder,
		Force:          req.Force,
		NoCache:        req.NoCache,
	}

	// Execute restore
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := restore.RunWithResult(ctx, []string{req.ProjectPath}, opts, console)
	elapsed := time.Since(start)

	// Load project to get lock file path
	proj, loadErr := project.LoadProject(req.ProjectPath)
	if loadErr != nil {
		return nil, fmt.Errorf("failed to load project: %w", loadErr)
	}

	lockFilePath := filepath.Join(filepath.Dir(proj.Path), "obj", "project.assets.json")

	// Handle restore errors - check both generic error and structured NuGetErrors
	errorMessages := make([]string, 0)

	// First, check if result has structured NuGetError entries with error codes
	if result != nil && len(result.Errors) > 0 {
		// Use structured NuGetErrors which include proper error codes (NU1101, NU1102, etc.)
		for _, nugetErr := range result.Errors {
			// Format without ANSI colors for interop test consumption
			errorMessages = append(errorMessages, nugetErr.FormatError(false))
		}
	} else if err != nil {
		// Fallback to generic error if no structured errors
		errorMessages = append(errorMessages, err.Error())
	}

	// If we have errors, restore failed
	if len(errorMessages) > 0 {
		return RestoreTransitiveResponse{
			Success:            false,
			DirectPackages:     []RestoredPackageInfo{},
			TransitivePackages: []RestoredPackageInfo{},
			UnresolvedPackages: []UnresolvedPackage{},
			LockFilePath:       lockFilePath,
			ElapsedMs:          elapsed.Milliseconds(),
			ErrorMessages:      errorMessages,
		}, nil
	}

	// Categorize packages as direct vs transitive
	// Initialize as empty slices (not nil) to avoid null in JSON serialization
	directPackages := make([]RestoredPackageInfo, 0)
	transitivePackages := make([]RestoredPackageInfo, 0)

	if result != nil {
		for _, pkg := range result.DirectPackages {
			directPackages = append(directPackages, RestoredPackageInfo{
				PackageID: pkg.ID,
				Version:   pkg.Version,
				Path:      pkg.Path,
				IsDirect:  true,
			})
		}

		for _, pkg := range result.TransitivePackages {
			transitivePackages = append(transitivePackages, RestoredPackageInfo{
				PackageID: pkg.ID,
				Version:   pkg.Version,
				Path:      pkg.Path,
				IsDirect:  false,
			})
		}
	}

	return RestoreTransitiveResponse{
		Success:            true,
		DirectPackages:     directPackages,
		TransitivePackages: transitivePackages,
		UnresolvedPackages: []UnresolvedPackage{},
		LockFilePath:       lockFilePath,
		ElapsedMs:          elapsed.Milliseconds(),
		ErrorMessages:      []string{},
	}, nil
}

// CompareProjectAssetsHandler compares two project.assets.json files semantically.
type CompareProjectAssetsHandler struct{}

// ErrorCode returns the error code for comparison failures.
func (h *CompareProjectAssetsHandler) ErrorCode() string {
	return "COMPARE_ASSETS_001"
}

// Handle processes the comparison request.
func (h *CompareProjectAssetsHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req CompareProjectAssetsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Read both lock files
	gonugetData, err := os.ReadFile(req.GonugetLockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read gonuget lock file: %w", err)
	}

	nugetData, err := os.ReadFile(req.NugetLockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read nuget lock file: %w", err)
	}

	// Parse both lock files
	var gonugetLockFile restore.LockFile
	if err := json.Unmarshal(gonugetData, &gonugetLockFile); err != nil {
		return nil, fmt.Errorf("failed to parse gonuget lock file: %w", err)
	}

	var nugetLockFile restore.LockFile
	if err := json.Unmarshal(nugetData, &nugetLockFile); err != nil {
		return nil, fmt.Errorf("failed to parse nuget lock file: %w", err)
	}

	// Compare lock files
	differences := []string{}
	librariesMatch := true
	projectFileDependencyGroupsMatch := true
	versionsMatch := true
	pathsMatch := true

	// Compare Libraries map keys
	if len(gonugetLockFile.Libraries) != len(nugetLockFile.Libraries) {
		librariesMatch = false
		differences = append(differences, fmt.Sprintf("Libraries count mismatch: gonuget=%d, nuget=%d",
			len(gonugetLockFile.Libraries), len(nugetLockFile.Libraries)))
	}

	for key, gonugetLib := range gonugetLockFile.Libraries {
		nugetLib, exists := nugetLockFile.Libraries[key]
		if !exists {
			librariesMatch = false
			differences = append(differences, fmt.Sprintf("Library key missing in nuget: %s", key))
			continue
		}

		// Compare versions
		if gonugetLib.Type != nugetLib.Type {
			versionsMatch = false
			differences = append(differences, fmt.Sprintf("Library type mismatch for %s: gonuget=%s, nuget=%s",
				key, gonugetLib.Type, nugetLib.Type))
		}

		// Compare paths (should be lowercase)
		if gonugetLib.Path != nugetLib.Path {
			pathsMatch = false
			differences = append(differences, fmt.Sprintf("Library path mismatch for %s: gonuget=%s, nuget=%s",
				key, gonugetLib.Path, nugetLib.Path))
		}
	}

	// Compare ProjectFileDependencyGroups
	for framework, gonugetDeps := range gonugetLockFile.ProjectFileDependencyGroups {
		nugetDeps, exists := nugetLockFile.ProjectFileDependencyGroups[framework]
		if !exists {
			projectFileDependencyGroupsMatch = false
			differences = append(differences, fmt.Sprintf("Framework missing in nuget ProjectFileDependencyGroups: %s", framework))
			continue
		}

		if len(gonugetDeps) != len(nugetDeps) {
			projectFileDependencyGroupsMatch = false
			differences = append(differences, fmt.Sprintf("Dependency count mismatch for %s: gonuget=%d, nuget=%d",
				framework, len(gonugetDeps), len(nugetDeps)))
		}
	}

	areEqual := librariesMatch && projectFileDependencyGroupsMatch && versionsMatch && pathsMatch

	return CompareProjectAssetsResponse{
		AreEqual:                         areEqual,
		LibrariesMatch:                   librariesMatch,
		ProjectFileDependencyGroupsMatch: projectFileDependencyGroupsMatch,
		VersionsMatch:                    versionsMatch,
		PathsMatch:                       pathsMatch,
		Differences:                      differences,
	}, nil
}

// ValidateErrorMessagesHandler validates error message format between gonuget and NuGet.Client.
type ValidateErrorMessagesHandler struct{}

// ErrorCode returns the error code for validation failures.
func (h *ValidateErrorMessagesHandler) ErrorCode() string {
	return "VALIDATE_ERRORS_001"
}

// Handle processes the validation request.
func (h *ValidateErrorMessagesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ValidateErrorMessagesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Extract error code from messages (NU1101, NU1102, NU1103)
	errorCode := extractErrorCode(req.GonugetError)
	if errorCode == "" {
		errorCode = extractErrorCode(req.NugetError)
	}

	// Compare messages with tolerance for formatting
	differences := []string{}
	match := req.GonugetError == req.NugetError

	if !match {
		// Check for formatting differences (whitespace, line endings)
		gonugetNormalized := normalizeErrorMessage(req.GonugetError)
		nugetNormalized := normalizeErrorMessage(req.NugetError)

		if gonugetNormalized != nugetNormalized {
			differences = append(differences, "Error message content differs")
			differences = append(differences, fmt.Sprintf("gonuget: %s", req.GonugetError))
			differences = append(differences, fmt.Sprintf("nuget: %s", req.NugetError))
		} else {
			// Only formatting differences - consider it a match
			match = true
		}
	}

	return ValidateErrorMessagesResponse{
		ErrorCode:          errorCode,
		GonugetMessage:     req.GonugetError,
		NuGetClientMessage: req.NugetError,
		Match:              match,
		Differences:        differences,
	}, nil
}

// extractErrorCode extracts the NuGet error code from an error message.
func extractErrorCode(message string) string {
	// Match NU1101, NU1102, NU1103 patterns
	if len(message) >= 6 {
		prefix := message[:6]
		if prefix == "NU1101" || prefix == "NU1102" || prefix == "NU1103" {
			return prefix
		}
	}
	return ""
}

// normalizeErrorMessage normalizes an error message for comparison.
func normalizeErrorMessage(message string) string {
	// Remove leading/trailing whitespace
	// Normalize line endings
	// Normalize multiple spaces to single space
	normalized := message
	normalized = strings.TrimSpace(normalized)
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Normalize multiple spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	return normalized
}
