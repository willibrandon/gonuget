package restore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

	// Build the JSON structure
	hasher := NewDgSpecHasher(proj).
		WithPackagesPath(config.PackagesPath).
		WithSources(config.Sources).
		WithConfigPaths(config.ConfigPaths).
		WithRuntimeIDPath(config.RuntimeIDPath)

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
	PackagesPath  string
	Sources       []string
	ConfigPaths   []string
	RuntimeIDPath string
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
		RuntimeIDPath: "/usr/local/share/dotnet/sdk/9.0.306/RuntimeIdentifierGraph.json",
	}
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
			// Resolve symlinks to get real path (e.g., /tmp -> /private/tmp on macOS)
			if realPath, err := filepath.EvalSymlinks(path); err == nil {
				configPaths = append(configPaths, realPath)
			} else {
				configPaths = append(configPaths, path)
			}
		}
	}

	// Load and merge all configs to get sources
	var allSources []string
	sourceSet := make(map[string]bool)

	for _, configPath := range configPaths {
		cfg, err := config.LoadNuGetConfig(configPath)
		if err != nil {
			continue // Skip invalid configs
		}

		// Get enabled sources from this config
		for _, src := range cfg.GetEnabledPackageSources() {
			if !sourceSet[src.Value] {
				sourceSet[src.Value] = true
				allSources = append(allSources, src.Value)
			}
		}
	}

	// Sort sources for determinism (dotnet sorts them)
	sort.Strings(allSources)

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

	return &DgSpecConfig{
		PackagesPath:  packagesPath,
		Sources:       allSources,
		ConfigPaths:   configPaths,
		RuntimeIDPath: "/usr/local/share/dotnet/sdk/9.0.306/RuntimeIdentifierGraph.json",
	}, nil
}
