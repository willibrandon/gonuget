// cmd/gonuget/config/nuget_config.go
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
	APIKeys                  *APIKeys                  `xml:"apikeys"`
	Config                   *ConfigSection            `xml:"config"`
	TrustedSigners           *TrustedSigners           `xml:"trustedSigners"`
	PackageSourceCredentials *PackageSourceCredentials `xml:"packageSourceCredentials"`
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

// ConfigSection contains configuration settings
type ConfigSection struct {
	Clear bool         `xml:"clear"`
	Add   []ConfigItem `xml:"add"`
}

// ConfigItem represents a configuration key-value pair
type ConfigItem struct {
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
	Add     []ConfigItem `xml:"add"`
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
		c.Config = &ConfigSection{}
	}

	// Check if exists
	for i := range c.Config.Add {
		if c.Config.Add[i].Key == key {
			c.Config.Add[i].Value = value
			return
		}
	}

	// Add new
	c.Config.Add = append(c.Config.Add, ConfigItem{Key: key, Value: value})
}
