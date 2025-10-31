package restore

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// CalculateDgSpecHash computes dependency graph hash for a project.
// Matches DependencyGraphSpec.GetHash() in NuGet.Client.
//
// Uses FNV-1a 64-bit hash algorithm (default in NuGet.Client since .NET 5).
// Reference: NuGet.ProjectModel/DependencyGraphSpec.cs GetHash() method
//
// The hash is computed over the complete dependency graph specification JSON,
// ensuring 100% compatibility with dotnet restore cache files.
func CalculateDgSpecHash(proj *project.Project) (string, error) {
	// Discover actual configuration from NuGet.config files
	cfg, err := DiscoverDgSpecConfig(proj)
	if err != nil {
		// Fall back to defaults if discovery fails
		cfg = DefaultDgSpecConfig()
	}
	return CalculateDgSpecHashWithConfig(proj, cfg)
}

// CalculateDgSpecHashWithConfig computes hash with custom configuration.
func CalculateDgSpecHashWithConfig(proj *project.Project, config *DgSpecConfig) (string, error) {
	// Apply defaults
	if config == nil {
		config = DefaultDgSpecConfig()
	}

	// Extract downloadDependencies from existing dgspec (if it exists)
	downloadDepsMap := make(map[string]map[string]string)
	frameworks := proj.GetTargetFrameworks()
	for _, tfm := range frameworks {
		deps := extractDownloadDependenciesFromDgSpec(proj.Path, tfm)
		if deps != nil {
			downloadDepsMap[tfm] = deps
		}
	}

	// Build the JSON structure
	hasher := NewDgSpecHasher(proj).
		WithPackagesPath(config.PackagesPath).
		WithFallbackFolders(config.FallbackFolders).
		WithSources(config.Sources).
		WithConfigPaths(config.ConfigPaths).
		WithRuntimeIDPath(config.RuntimeIDPath).
		WithSdkAnalysisLevel(config.SdkAnalysisLevel)

	// Only set downloadDependencies if we found any
	if len(downloadDepsMap) > 0 {
		hasher = hasher.WithDownloadDependencies(downloadDepsMap)
	}

	jsonBytes, err := hasher.GenerateJSON()
	if err != nil {
		return "", fmt.Errorf("generate dgspec JSON: %w", err)
	}

	// Compute FNV-1a hash
	fnv := NewFnvHash64()
	fnv.Update(jsonBytes)

	return fnv.GetHash(), nil
}

// DgSpecConfig holds configuration for dgSpec hash calculation.
type DgSpecConfig struct {
	PackagesPath     string
	FallbackFolders  []string
	Sources          []string
	ConfigPaths      []string
	RuntimeIDPath    string
	SdkAnalysisLevel string
}

// DefaultDgSpecConfig returns default configuration.
func DefaultDgSpecConfig() *DgSpecConfig {
	homeDir, _ := os.UserHomeDir()
	packagesPath := filepath.Join(homeDir, ".nuget", "packages")

	return &DgSpecConfig{
		PackagesPath: packagesPath,
		Sources: []string{
			"https://api.nuget.org/v3/index.json",
		},
		ConfigPaths:   []string{},
		RuntimeIDPath: detectRuntimeIDGraphPath(),
	}
}

// detectLibraryPacksPath finds the implicit library-packs source path.
// Matches Microsoft.NET.NuGetOfflineCache.targets behavior:
// _WorkloadLibraryPacksFolder = $(NetCoreRoot)/library-packs
// This source is implicitly added by the SDK when the directory exists.
func detectLibraryPacksPath() string {
	// Get dotnet root by running 'dotnet --list-sdks' and extracting the parent of sdk folder
	cmd := exec.Command("dotnet", "--list-sdks")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output to find SDK path
	// Output format: "9.0.100 [/usr/share/dotnet/sdk]" or "9.0.100 [C:\Program Files\dotnet\sdk]"
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse: "version [path]" - find the bracket to handle paths with spaces
		bracketStart := strings.Index(line, "[")
		bracketEnd := strings.Index(line, "]")
		if bracketStart >= 0 && bracketEnd > bracketStart {
			sdkPath := line[bracketStart+1 : bracketEnd]
			// NetCoreRoot is parent of sdk folder
			netCoreRoot := filepath.Dir(sdkPath)
			libraryPacksPath := filepath.Join(netCoreRoot, "library-packs")

			// Check if library-packs exists (matching SDK condition)
			if _, err := os.Stat(libraryPacksPath); err == nil {
				return libraryPacksPath
			}
			// If library-packs doesn't exist, return empty (don't check other SDKs)
			return ""
		}
	}

	return ""
}

// detectSdkFeatureBand extracts the SDK feature band from a version string.
// Examples:
//   - "10.0.100-rc.2.25502.107" -> "10.0.100"
//   - "9.0.300" -> "9.0.300"
//
// The feature band is always the hundreds-value: major.minor.patch where patch ends in 00.
func detectSdkFeatureBand(sdkVersion string) string {
	// Parse version to extract major.minor.patch
	// Version format: major.minor.patch[-prerelease][+build]
	parts := strings.Split(sdkVersion, "-")
	versionCore := parts[0]

	// Split into major.minor.patch
	versionParts := strings.Split(versionCore, ".")
	if len(versionParts) < 3 {
		return ""
	}

	// Feature band is major.minor.(patch rounded down to hundreds)
	// For example: 10.0.107 -> 10.0.100, 9.0.354 -> 9.0.300
	major := versionParts[0]
	minor := versionParts[1]
	patch := versionParts[2]

	// Parse patch and round down to hundreds
	var patchNum int
	_, _ = fmt.Sscanf(patch, "%d", &patchNum) // Ignore error, default to 0
	featureBand := patchNum / 100 * 100

	return fmt.Sprintf("%s.%s.%d", major, minor, featureBand)
}

// detectRuntimeIDGraphPath finds the actual SDK path from dotnet.
// Matches dotnet's behavior of using the highest installed SDK version.
func detectRuntimeIDGraphPath() string {
	// Try to get SDK path from dotnet --list-sdks
	cmd := exec.Command("dotnet", "--list-sdks")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to common paths if dotnet command fails
		return getDefaultRuntimeIDPath()
	}

	// Parse output to find highest SDK version
	// Output format: "9.0.100 [/usr/share/dotnet/sdk]"
	lines := strings.Split(string(output), "\n")
	var highestVersion string
	var sdkBase string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse: "version [path]"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			version := parts[0]
			path := strings.Trim(parts[1], "[]")

			if highestVersion == "" || version > highestVersion {
				highestVersion = version
				sdkBase = path
			}
		}
	}

	if highestVersion != "" && sdkBase != "" {
		return filepath.Join(sdkBase, highestVersion, "PortableRuntimeIdentifierGraph.json")
	}

	return getDefaultRuntimeIDPath()
}

// getDefaultRuntimeIDPath returns platform-specific default SDK path.
func getDefaultRuntimeIDPath() string {
	// Try common SDK locations by platform
	homeDir, _ := os.UserHomeDir()

	possiblePaths := []string{
		// macOS Homebrew
		"/usr/local/share/dotnet/sdk",
		// macOS Apple Silicon
		"/opt/homebrew/share/dotnet/sdk",
		// Linux
		"/usr/share/dotnet/sdk",
		// Windows
		filepath.Join(os.Getenv("ProgramFiles"), "dotnet", "sdk"),
		// User-local installation
		filepath.Join(homeDir, ".dotnet", "sdk"),
	}

	for _, basePath := range possiblePaths {
		if entries, err := os.ReadDir(basePath); err == nil {
			// Find highest version directory
			var highest string
			for _, entry := range entries {
				if entry.IsDir() {
					version := entry.Name()
					if highest == "" || version > highest {
						highest = version
					}
				}
			}
			if highest != "" {
				return filepath.Join(basePath, highest, "PortableRuntimeIdentifierGraph.json")
			}
		}
	}

	// Last resort fallback (won't match any real hash but won't crash)
	return ""
}

// DiscoverDgSpecConfig discovers configuration from project directory.
// Reads NuGet.config files and returns configuration matching dotnet's behavior.
func DiscoverDgSpecConfig(proj *project.Project) (*DgSpecConfig, error) {
	projectDir := filepath.Dir(proj.Path)

	// Get all NuGet.config files in hierarchy (matches dotnet behavior)
	allConfigPaths := config.GetConfigHierarchy(projectDir)

	// Filter to only existing files (dotnet only includes files that exist)
	// Also resolve symlinks and get real paths to match dotnet behavior
	var configPaths []string
	for _, path := range allConfigPaths {
		if _, err := os.Stat(path); err == nil {
			// On Windows, skip symlink resolution as dotnet doesn't resolve them
			// and EvalSymlinks can change path casing on case-insensitive filesystems
			// On Unix (macOS/Linux), resolve symlinks (e.g., /tmp -> /private/tmp on macOS)
			if runtime.GOOS == "windows" {
				configPaths = append(configPaths, path)
			} else {
				if realPath, err := filepath.EvalSymlinks(path); err == nil {
					configPaths = append(configPaths, realPath)
				} else {
					configPaths = append(configPaths, path)
				}
			}
		}
	}

	// Load and merge all configs to get sources
	// NuGet processes configs from least specific to most specific,
	// and <clear /> clears all previously accumulated sources.
	var allSources []string
	sourceSet := make(map[string]bool)

	// Process configs in reverse order (least specific first)
	for i := len(configPaths) - 1; i >= 0; i-- {
		configPath := configPaths[i]
		cfg, err := config.LoadNuGetConfig(configPath)
		if err != nil {
			continue // Skip invalid configs
		}

		// Check if this config clears all parent sources
		// In NuGet, any <clear> element (even <clear>false</clear>) triggers a clear
		if cfg.PackageSources != nil && cfg.PackageSources.Clear != nil {
			// Clear all previously accumulated sources
			allSources = nil
			sourceSet = make(map[string]bool)
		}

		// Get enabled sources from this config
		for _, src := range cfg.GetEnabledPackageSources() {
			sourceValue := src.Value

			// Normalize local file paths to use native separators
			// (URLs should be left as-is)
			if !strings.HasPrefix(sourceValue, "http://") && !strings.HasPrefix(sourceValue, "https://") {
				// This is a local path - normalize separators
				sourceValue = filepath.FromSlash(sourceValue)
			}

			if !sourceSet[sourceValue] {
				sourceSet[sourceValue] = true
				allSources = append(allSources, sourceValue)
			}
		}
	}

	// Add implicit library-packs source (added by Microsoft.NET.NuGetOfflineCache.targets)
	// This matches the SDK behavior: if library-packs exists, it's implicitly added to sources
	libraryPacksPath := detectLibraryPacksPath()
	if libraryPacksPath != "" {
		if !sourceSet[libraryPacksPath] {
			sourceSet[libraryPacksPath] = true
			allSources = append(allSources, libraryPacksPath)
		}
	}

	// If no sources found, use default
	if len(allSources) == 0 {
		allSources = []string{"https://api.nuget.org/v3/index.json"}
	}

	// Get packages path
	homeDir, _ := os.UserHomeDir()
	packagesPath := filepath.Join(homeDir, ".nuget", "packages")

	// Check if any config overrides packages path
	for _, configPath := range configPaths {
		cfg, err := config.LoadNuGetConfig(configPath)
		if err != nil {
			continue
		}
		if globalPackagesFolder := cfg.GetConfigValue("globalPackagesFolder"); globalPackagesFolder != "" {
			packagesPath = globalPackagesFolder
			break
		}
	}

	// Collect fallback folders from configs
	var fallbackFolders []string
	fallbackSet := make(map[string]bool)

	for _, configPath := range configPaths {
		cfg, err := config.LoadNuGetConfig(configPath)
		if err != nil {
			continue
		}

		// Get fallback folders from this config
		if cfg.FallbackPackageFolders != nil {
			for _, folder := range cfg.FallbackPackageFolders.Add {
				folderPath := folder.Value

				// Normalize path to use native separators
				if !strings.HasPrefix(folderPath, "http://") && !strings.HasPrefix(folderPath, "https://") {
					folderPath = filepath.FromSlash(folderPath)
				}

				if !fallbackSet[folderPath] {
					fallbackSet[folderPath] = true
					fallbackFolders = append(fallbackFolders, folderPath)
				}
			}
		}
	}

	// Try to read runtimeIdentifierGraphPath from existing dgspec.json
	// This ensures we use the exact same path dotnet used
	runtimeIDPath := extractRuntimeIDPathFromDgSpec(proj.Path)
	if runtimeIDPath == "" {
		// Fallback to detection if dgspec doesn't exist
		runtimeIDPath = detectRuntimeIDGraphPath()
	}

	// Try to read SdkAnalysisLevel from existing dgspec.json
	// This ensures we use the exact same value dotnet used
	sdkAnalysisLevel := extractSdkAnalysisLevelFromDgSpec(proj.Path)
	if sdkAnalysisLevel == "" {
		// Fallback to detection if dgspec doesn't exist
		// Extract feature band from SDK version
		cmd := exec.Command("dotnet", "--list-sdks")
		if output, err := cmd.Output(); err == nil {
			lines := strings.SplitSeq(string(output), "\n")
			for line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				// Parse: "version [path]"
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					sdkVersion := parts[0]
					sdkAnalysisLevel = detectSdkFeatureBand(sdkVersion)
					break // Use first (highest) SDK
				}
			}
		}
	}

	return &DgSpecConfig{
		PackagesPath:     packagesPath,
		FallbackFolders:  fallbackFolders,
		Sources:          allSources,
		ConfigPaths:      configPaths,
		RuntimeIDPath:    runtimeIDPath,
		SdkAnalysisLevel: sdkAnalysisLevel,
	}, nil
}

// extractRuntimeIDPathFromDgSpec reads the runtimeIdentifierGraphPath from
// an existing dgspec.json file that dotnet created. Returns empty string if
// the file doesn't exist or can't be parsed.
func extractRuntimeIDPathFromDgSpec(projectPath string) string {
	// Construct dgspec.json path: {projectDir}/obj/{projectFileName}.nuget.dgspec.json
	projectDir := filepath.Dir(projectPath)
	projectFileName := filepath.Base(projectPath)
	dgspecPath := filepath.Join(projectDir, "obj", projectFileName+".nuget.dgspec.json")

	// Read the file
	data, err := os.ReadFile(dgspecPath)
	if err != nil {
		return "" // File doesn't exist or can't be read
	}

	// Parse JSON to extract runtimeIdentifierGraphPath
	// Structure: {"projects": {"<projectPath>": {"frameworks": {"<tfm>": {"runtimeIdentifierGraphPath": "..."}}}}}
	var dgspec map[string]any
	if err := json.Unmarshal(data, &dgspec); err != nil {
		return ""
	}

	projects, ok := dgspec["projects"].(map[string]any)
	if !ok {
		return ""
	}

	// Get the project entry (should be only one, but we take the first)
	for _, projectData := range projects {
		projectMap, ok := projectData.(map[string]any)
		if !ok {
			continue
		}

		frameworks, ok := projectMap["frameworks"].(map[string]any)
		if !ok {
			continue
		}

		// Get first framework entry
		for _, frameworkData := range frameworks {
			frameworkMap, ok := frameworkData.(map[string]any)
			if !ok {
				continue
			}

			if ridPath, ok := frameworkMap["runtimeIdentifierGraphPath"].(string); ok {
				return ridPath
			}
		}
	}

	return ""
}

// extractSdkAnalysisLevelFromDgSpec reads the SdkAnalysisLevel from
// an existing dgspec.json file that dotnet created. Returns empty string if
// the file doesn't exist or can't be parsed.
func extractSdkAnalysisLevelFromDgSpec(projectPath string) string {
	// Construct dgspec.json path: {projectDir}/obj/{projectFileName}.nuget.dgspec.json
	projectDir := filepath.Dir(projectPath)
	projectFileName := filepath.Base(projectPath)
	dgspecPath := filepath.Join(projectDir, "obj", projectFileName+".nuget.dgspec.json")

	// Read the file
	data, err := os.ReadFile(dgspecPath)
	if err != nil {
		return "" // File doesn't exist or can't be read
	}

	// Parse JSON to extract SdkAnalysisLevel
	// Structure: {"projects": {"<projectPath>": {"restore": {"SdkAnalysisLevel": "..."}}}}
	var dgspec map[string]any
	if err := json.Unmarshal(data, &dgspec); err != nil {
		return ""
	}

	projects, ok := dgspec["projects"].(map[string]any)
	if !ok {
		return ""
	}

	// Get the project entry (should be only one, but we take the first)
	for _, projectData := range projects {
		projectMap, ok := projectData.(map[string]any)
		if !ok {
			continue
		}

		restore, ok := projectMap["restore"].(map[string]any)
		if !ok {
			continue
		}

		if sdkAnalysisLevel, ok := restore["SdkAnalysisLevel"].(string); ok {
			return sdkAnalysisLevel
		}
	}

	return ""
}

// extractDownloadDependenciesFromDgSpec reads the downloadDependencies from
// an existing dgspec.json file that dotnet created. Returns nil if
// the file doesn't exist or can't be parsed.
func extractDownloadDependenciesFromDgSpec(projectPath, tfm string) map[string]string {
	// Construct dgspec.json path: {projectDir}/obj/{projectFileName}.nuget.dgspec.json
	projectDir := filepath.Dir(projectPath)
	projectFileName := filepath.Base(projectPath)
	dgspecPath := filepath.Join(projectDir, "obj", projectFileName+".nuget.dgspec.json")

	// Read the file
	data, err := os.ReadFile(dgspecPath)
	if err != nil {
		return nil // File doesn't exist or can't be read
	}

	// Parse JSON to extract downloadDependencies
	// Structure: {"projects": {"<projectPath>": {"frameworks": {"<tfm>": {"downloadDependencies": [...]}}}}}
	var dgspec map[string]any
	if err := json.Unmarshal(data, &dgspec); err != nil {
		return nil
	}

	projects, ok := dgspec["projects"].(map[string]any)
	if !ok {
		return nil
	}

	// Get the project entry (should be only one, but we take the first)
	for _, projectData := range projects {
		projectMap, ok := projectData.(map[string]any)
		if !ok {
			continue
		}

		frameworks, ok := projectMap["frameworks"].(map[string]any)
		if !ok {
			continue
		}

		// Get the framework entry for the specified TFM
		frameworkData, ok := frameworks[tfm].(map[string]any)
		if !ok {
			continue
		}

		downloadDeps, ok := frameworkData["downloadDependencies"].([]any)
		if !ok {
			return nil
		}

		// Build map of name -> version
		result := make(map[string]string)
		for _, dep := range downloadDeps {
			depMap, ok := dep.(map[string]any)
			if !ok {
				continue
			}
			name, ok := depMap["name"].(string)
			if !ok {
				continue
			}
			version, ok := depMap["version"].(string)
			if !ok {
				continue
			}
			result[name] = version
		}

		return result
	}

	return nil
}
