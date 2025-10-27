package commands

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
)

// sourceOptions holds common options for all source commands
type sourceOptions struct {
	configFile               string
	name                     string
	source                   string
	username                 string
	password                 string
	storePasswordInClearText bool
	validAuthenticationTypes string
	protocolVersion          string
	allowInsecureConnections bool
	format                   string // detailed or short
}

// statusString returns the status as a string matching dotnet nuget output
func statusString(enabled string) string {
	if enabled == "" || enabled == "true" {
		return "Enabled"
	}
	return "Disabled"
}

// encodePassword provides basic password encoding (placeholder for future OS keychain integration)
func encodePassword(password string) string {
	// TODO: Use proper encryption with OS keychain in Phase 8
	// For now, use base64 encoding as placeholder
	encoded := base64.StdEncoding.EncodeToString([]byte(password))
	return encoded
}

// loadSourceConfig loads a config file or creates a new one, returns config and path
func loadSourceConfig(configPath string) (*config.NuGetConfig, string, error) {
	// Determine config path
	if configPath == "" {
		configPath = config.FindConfigFile()
		if configPath == "" {
			// Create default config in user location
			configPath = config.GetUserConfigPath()
		}
	}

	// Try to load existing config
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := config.LoadNuGetConfig(configPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load config: %w", err)
		}
		return cfg, configPath, nil
	}

	// Create new config
	cfg := config.NewDefaultConfig()
	// Clear default sources for fresh config
	cfg.PackageSources.Add = nil
	return cfg, configPath, nil
}

// isSourceEnabled checks if a source is enabled
func isSourceEnabled(source *config.PackageSource) bool {
	return source.Enabled == "" || source.Enabled == "true"
}

// validateSourceExists checks if a source exists in the config
func validateSourceExists(cfg *config.NuGetConfig, name string) bool {
	return cfg.GetPackageSource(name) != nil
}

// findSourceByName finds a package source by name
func findSourceByName(cfg *config.NuGetConfig, name string) (*config.PackageSource, error) {
	source := cfg.GetPackageSource(name)
	if source == nil {
		return nil, fmt.Errorf("package source with name '%s' not found", name)
	}
	return source, nil
}

// addOrUpdateCredential adds or updates credentials for a source
func addOrUpdateCredential(cfg *config.NuGetConfig, sourceName string, username, password string, clearText bool, authTypes string) {
	if cfg.PackageSourceCredentials == nil {
		cfg.PackageSourceCredentials = &config.PackageSourceCredentials{}
	}

	// Create credential items
	var items []config.Item
	if username != "" {
		items = append(items, config.Item{Key: "Username", Value: username})
	}
	if password != "" {
		if clearText {
			items = append(items, config.Item{Key: "ClearTextPassword", Value: password})
		} else {
			items = append(items, config.Item{Key: "Password", Value: encodePassword(password)})
		}
	}
	if authTypes != "" {
		items = append(items, config.Item{Key: "ValidAuthenticationTypes", Value: authTypes})
	}

	// Find or create credential entry
	found := false
	for i := range cfg.PackageSourceCredentials.Items {
		if cfg.PackageSourceCredentials.Items[i].XMLName.Local == sourceName {
			cfg.PackageSourceCredentials.Items[i].Add = items
			found = true
			break
		}
	}

	if !found {
		cfg.PackageSourceCredentials.Items = append(cfg.PackageSourceCredentials.Items, config.SourceCredential{
			XMLName: xml.Name{Local: sourceName},
			Add:     items,
		})
	}
}
