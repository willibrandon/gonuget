// cmd/gonuget/config/defaults.go
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultConfigLocations returns the list of NuGet.config locations to search
// in precedence order
func DefaultConfigLocations() []string {
	var locations []string

	// Current directory
	if cwd, err := os.Getwd(); err == nil {
		locations = append(locations, filepath.Join(cwd, "NuGet.config"))
		locations = append(locations, filepath.Join(cwd, ".nuget", "NuGet.config"))
	}

	// User config
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(home, ".nuget", "NuGet", "NuGet.Config"))
	}

	// System config (platform-specific)
	if runtime.GOOS == "windows" {
		// Windows: %ProgramData%\NuGet\Config
		if programData := os.Getenv("ProgramData"); programData != "" {
			locations = append(locations, filepath.Join(programData, "NuGet", "Config", "Microsoft.VisualStudio.Offline.config"))
		}
	} else {
		// Unix: /etc/nuget
		locations = append(locations, "/etc/nuget/NuGet.config")
	}

	return locations
}

// FindConfigFile finds the first existing NuGet.config file
func FindConfigFile() string {
	for _, loc := range DefaultConfigLocations() {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	return ""
}

// GetUserConfigPath returns the user-level NuGet.config path
func GetUserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".nuget", "NuGet", "NuGet.Config")
}

// DefaultPackageSources returns the default package sources
func DefaultPackageSources() []PackageSource {
	return []PackageSource{
		{
			Key:             "nuget.org",
			Value:           "https://api.nuget.org/v3/index.json",
			ProtocolVersion: "3",
			Enabled:         "true",
		},
	}
}

// NewDefaultConfig creates a new config with default values
func NewDefaultConfig() *NuGetConfig {
	config := &NuGetConfig{
		PackageSources: &PackageSources{
			Add: DefaultPackageSources(),
		},
		Config: &ConfigSection{},
	}

	return config
}
