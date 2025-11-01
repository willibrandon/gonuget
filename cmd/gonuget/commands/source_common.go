package commands

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/zalando/go-keyring"
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

const (
	keychainService = "gonuget"
	keychainPrefix  = "keychain:"
)

// encodePassword stores password in OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
// Returns a marker string that references the keychain entry
func encodePassword(sourceName, password string) (string, error) {
	// Try to store in OS keychain
	err := keyring.Set(keychainService, sourceName, password)
	if err != nil {
		// Keychain not available - fall back to base64 encoding with warning
		// This can happen in headless environments, CI/CD, or if user denies access
		encoded := base64.StdEncoding.EncodeToString([]byte(password))
		return encoded, fmt.Errorf("keychain unavailable, using base64 encoding (less secure): %w", err)
	}

	// Return marker indicating password is in keychain
	return keychainPrefix + sourceName, nil
}

// decodePassword retrieves password from OS keychain or decodes base64-encoded password
func decodePassword(sourceName, encodedValue string) (string, error) {
	// Check if this is a keychain reference
	if keychainKey, found := strings.CutPrefix(encodedValue, keychainPrefix); found {
		// Validate that the keychain key matches the expected source name
		if keychainKey != sourceName {
			return "", fmt.Errorf("keychain key mismatch: expected %s, got %s", sourceName, keychainKey)
		}
		password, err := keyring.Get(keychainService, keychainKey)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve password from keychain: %w", err)
		}
		return password, nil
	}

	// Fall back to base64 decoding for legacy/fallback passwords
	decoded, err := base64.StdEncoding.DecodeString(encodedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode password: %w", err)
	}
	return string(decoded), nil
}

// deletePasswordFromKeychain removes password from OS keychain
func deletePasswordFromKeychain(sourceName string) error {
	err := keyring.Delete(keychainService, sourceName)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("failed to delete password from keychain: %w", err)
	}
	return nil
}

// loadSourceConfig loads a config file or creates a new one, returns config and path
func loadSourceConfig(configPath string) (*config.NuGetConfig, string, error) {
	// Track if config path was explicitly provided
	explicitPath := configPath != ""

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
	} else if explicitPath {
		// If user explicitly specified a config file path and it doesn't exist, return error
		return nil, "", fmt.Errorf("specified config file does not exist: %s", configPath)
	}

	// Create new config (only when no explicit path was provided)
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
// Returns (warning, error) where warning is non-empty if keychain fallback was used
func addOrUpdateCredential(cfg *config.NuGetConfig, sourceName string, username, password string, clearText bool, authTypes string) (string, error) {
	if cfg.PackageSourceCredentials == nil {
		cfg.PackageSourceCredentials = &config.PackageSourceCredentials{}
	}

	var warning string

	// Create credential items
	var items []config.Item
	if username != "" {
		items = append(items, config.Item{Key: "Username", Value: username})
	}
	if password != "" {
		if clearText {
			items = append(items, config.Item{Key: "ClearTextPassword", Value: password})
		} else {
			encodedPassword, err := encodePassword(sourceName, password)
			if err != nil {
				// err contains warning about keychain fallback
				warning = err.Error()
			}
			items = append(items, config.Item{Key: "Password", Value: encodedPassword})
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

	return warning, nil
}
