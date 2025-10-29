// Package config implements NuGet configuration management and parsing.
package config

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// NuGetConfig represents a NuGet.config file
type NuGetConfig struct {
	XMLName                  xml.Name                  `xml:"configuration"`
	PackageSources           *PackageSources           `xml:"packageSources"`
	DisabledPackageSources   *DisabledPackageSources   `xml:"disabledPackageSources,omitempty"`
	APIKeys                  *APIKeys                  `xml:"apikeys"`
	Config                   *Section                  `xml:"config"`
	TrustedSigners           *TrustedSigners           `xml:"trustedSigners"`
	PackageSourceCredentials *PackageSourceCredentials `xml:"packageSourceCredentials"`
}

// DisabledPackageSources contains disabled package source definitions
type DisabledPackageSources struct {
	Add []DisabledPackageSource `xml:"add"`
}

// DisabledPackageSource represents a disabled package source
type DisabledPackageSource struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// PackageSources contains package source definitions
type PackageSources struct {
	Clear bool            `xml:"clear"`
	Add   []PackageSource `xml:"add"`
}

// PackageSource represents a package source
type PackageSource struct {
	Key             string `xml:"key,attr"`
	Value           string `xml:"value,attr"`
	ProtocolVersion string `xml:"protocolVersion,attr,omitempty"`
	Enabled         string `xml:"enabled,attr,omitempty"`
}

// APIKeys contains API key mappings
type APIKeys struct {
	Clear bool     `xml:"clear"`
	Add   []APIKey `xml:"add"`
}

// APIKey represents an API key for a source
type APIKey struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// Section contains configuration settings
type Section struct {
	Clear bool   `xml:"clear"`
	Add   []Item `xml:"add"`
}

// Item represents a configuration key-value pair
type Item struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// TrustedSigners contains trusted signer definitions
type TrustedSigners struct {
	Clear bool            `xml:"clear"`
	Add   []TrustedSigner `xml:"add,omitempty"`
}

// TrustedSigner represents a trusted signer
type TrustedSigner struct {
	Name string `xml:"name,attr"`
	// Additional fields as needed
}

// PackageSourceCredentials contains credentials for sources
type PackageSourceCredentials struct {
	Items []SourceCredential `xml:",any"`
}

// SourceCredential represents credentials for a source
type SourceCredential struct {
	XMLName xml.Name
	Add     []Item `xml:"add"`
}

// LoadNuGetConfig loads a NuGet.config file
func LoadNuGetConfig(path string) (*NuGetConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	return ParseNuGetConfig(f)
}

// ParseNuGetConfig parses NuGet.config XML from a reader
func ParseNuGetConfig(r io.Reader) (*NuGetConfig, error) {
	var config NuGetConfig
	decoder := xml.NewDecoder(r)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config XML: %w", err)
	}

	return &config, nil
}

// SaveNuGetConfig saves a NuGet.config file
func SaveNuGetConfig(path string, config *NuGetConfig) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	return WriteNuGetConfig(f, config)
}

// WriteNuGetConfig writes NuGet.config XML to a writer
func WriteNuGetConfig(w io.Writer, config *NuGetConfig) error {
	// Write XML declaration
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}

	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config XML: %w", err)
	}

	return encoder.Flush()
}

// GetPackageSource gets a package source by key
func (c *NuGetConfig) GetPackageSource(key string) *PackageSource {
	if c.PackageSources == nil {
		return nil
	}

	for i := range c.PackageSources.Add {
		if c.PackageSources.Add[i].Key == key {
			return &c.PackageSources.Add[i]
		}
	}

	return nil
}

// AddPackageSource adds or updates a package source
func (c *NuGetConfig) AddPackageSource(source PackageSource) {
	if c.PackageSources == nil {
		c.PackageSources = &PackageSources{}
	}

	// Check if source exists
	for i := range c.PackageSources.Add {
		if c.PackageSources.Add[i].Key == source.Key {
			// Update existing
			c.PackageSources.Add[i] = source
			return
		}
	}

	// Add new
	c.PackageSources.Add = append(c.PackageSources.Add, source)
}

// RemovePackageSource removes a package source by key
func (c *NuGetConfig) RemovePackageSource(key string) bool {
	if c.PackageSources == nil {
		return false
	}

	for i := range c.PackageSources.Add {
		if c.PackageSources.Add[i].Key == key {
			c.PackageSources.Add = append(
				c.PackageSources.Add[:i],
				c.PackageSources.Add[i+1:]...,
			)
			return true
		}
	}

	return false
}

// GetConfigValue gets a configuration value by key
func (c *NuGetConfig) GetConfigValue(key string) string {
	if c.Config == nil {
		return ""
	}

	for _, item := range c.Config.Add {
		if item.Key == key {
			return item.Value
		}
	}

	return ""
}

// SetConfigValue sets a configuration value
func (c *NuGetConfig) SetConfigValue(key, value string) {
	if c.Config == nil {
		c.Config = &Section{}
	}

	// Check if exists
	for i := range c.Config.Add {
		if c.Config.Add[i].Key == key {
			c.Config.Add[i].Value = value
			return
		}
	}

	// Add new
	c.Config.Add = append(c.Config.Add, Item{Key: key, Value: value})
}

// DeleteConfigValue removes a configuration value by key
func (c *NuGetConfig) DeleteConfigValue(key string) {
	if c.Config == nil {
		return
	}

	var filtered []Item
	for _, item := range c.Config.Add {
		if item.Key != key {
			filtered = append(filtered, item)
		}
	}
	c.Config.Add = filtered
}

// FindConfigFileFrom finds config file starting from specified directory
func FindConfigFileFrom(startDir string) string {
	dir := startDir
	for {
		configPath := filepath.Join(dir, "NuGet.config")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Fall back to user config
	return GetUserConfigPath()
}

// GetConfigHierarchy returns all config file paths in the hierarchy
func GetConfigHierarchy(workingDirectory string) []string {
	var paths []string

	// Start directory
	startDir := workingDirectory
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			startDir = "."
		}
	}

	// Walk up directory tree
	dir := startDir
	for {
		// Check for both casings (NuGet.Config and NuGet.config)
		// Prefer the one that actually exists on disk
		configPath := filepath.Join(dir, "NuGet.Config")
		if _, err := os.Stat(configPath); err == nil {
			paths = append(paths, configPath)
		} else {
			// Try lowercase if capital C doesn't exist
			configPath = filepath.Join(dir, "NuGet.config")
			if _, err := os.Stat(configPath); err == nil {
				paths = append(paths, configPath)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Add user config
	userConfig := GetUserConfigPath()
	paths = append(paths, userConfig)

	// Add machine-wide config (platform-specific)
	machineConfig := getMachineWideConfigPath()
	if machineConfig != "" {
		paths = append(paths, machineConfig)
	}

	return paths
}

// getMachineWideConfigPath returns the machine-wide config path
func getMachineWideConfigPath() string {
	// Platform-specific logic
	if programData := os.Getenv("ProgramData"); programData != "" {
		// Windows
		return filepath.Join(programData, "NuGet", "Config", "NuGet.config")
	}
	// Unix-like systems
	return "/etc/nuget/NuGet.config"
}

// IsSourceDisabled checks if a source is disabled
func (c *NuGetConfig) IsSourceDisabled(key string) bool {
	if c.DisabledPackageSources == nil {
		return false
	}

	for _, disabled := range c.DisabledPackageSources.Add {
		if disabled.Key == key && disabled.Value == "true" {
			return true
		}
	}

	return false
}

// DisableSource disables a package source
func (c *NuGetConfig) DisableSource(key string) {
	if c.DisabledPackageSources == nil {
		c.DisabledPackageSources = &DisabledPackageSources{}
	}

	// Check if already disabled
	for i := range c.DisabledPackageSources.Add {
		if c.DisabledPackageSources.Add[i].Key == key {
			c.DisabledPackageSources.Add[i].Value = "true"
			return
		}
	}

	// Add to disabled list
	c.DisabledPackageSources.Add = append(c.DisabledPackageSources.Add, DisabledPackageSource{
		Key:   key,
		Value: "true",
	})
}

// EnableSource enables a package source
func (c *NuGetConfig) EnableSource(key string) {
	if c.DisabledPackageSources == nil {
		return
	}

	// Remove from disabled list
	var filtered []DisabledPackageSource
	for _, disabled := range c.DisabledPackageSources.Add {
		if disabled.Key != key {
			filtered = append(filtered, disabled)
		}
	}

	c.DisabledPackageSources.Add = filtered

	// Clean up empty section
	if len(c.DisabledPackageSources.Add) == 0 {
		c.DisabledPackageSources = nil
	}
}

// GetEnabledPackageSources returns all enabled package sources from the config.
// A source is enabled if it's not in the disabledPackageSources section or if its enabled attribute is "true".
func (c *NuGetConfig) GetEnabledPackageSources() []PackageSource {
	if c.PackageSources == nil {
		return []PackageSource{}
	}

	var enabled []PackageSource
	for _, source := range c.PackageSources.Add {
		// Check if explicitly disabled in disabledPackageSources section
		if c.IsSourceDisabled(source.Key) {
			continue
		}

		// Check if disabled via enabled attribute
		if source.Enabled == "false" {
			continue
		}

		enabled = append(enabled, source)
	}

	return enabled
}

// GetEnabledSourcesOrDefault returns enabled package sources from the config hierarchy,
// or default sources if none are configured. This matches NuGet.Client behavior where
// the default nuget.org source is always available as a fallback.
//
// The function searches for config files starting from startDir and walking up the directory tree,
// then checks the user config location. If no sources are found in any config, it returns
// the default sources (nuget.org).
func GetEnabledSourcesOrDefault(startDir string) []PackageSource {
	// Try to find and load config from the hierarchy
	configPath := FindConfigFileFrom(startDir)
	if configPath != "" {
		cfg, err := LoadNuGetConfig(configPath)
		if err == nil {
			sources := cfg.GetEnabledPackageSources()
			if len(sources) > 0 {
				return sources
			}
		}
	}

	// If no sources found in local config, try user config
	userConfigPath := GetUserConfigPath()
	if userConfigPath != "" {
		cfg, err := LoadNuGetConfig(userConfigPath)
		if err == nil {
			sources := cfg.GetEnabledPackageSources()
			if len(sources) > 0 {
				return sources
			}
		}
	}

	// If still no sources found, ensure user config exists and return default sources
	// This matches NuGet.Client behavior where it auto-creates the config with defaults
	_ = EnsureUserConfigExists()
	return DefaultPackageSources()
}

// EnsureUserConfigExists creates the user-level NuGet.Config file with default sources
// if it doesn't already exist. This matches NuGet.Client's behavior of auto-creating
// the config file when any NuGet operation is performed.
func EnsureUserConfigExists() error {
	configPath := GetUserConfigPath()
	if configPath == "" {
		return fmt.Errorf("unable to determine user config path")
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config already exists
	}

	// Create the config with default sources
	cfg := NewDefaultConfig()
	return SaveNuGetConfig(configPath, cfg)
}
