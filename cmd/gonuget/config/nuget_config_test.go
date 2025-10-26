// cmd/gonuget/config/nuget_config_test.go
package config

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseNuGetConfig(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" protocolVersion="3" />
  </packageSources>
  <config>
    <add key="globalPackagesFolder" value="~/.nuget/packages" />
  </config>
</configuration>`

	config, err := ParseNuGetConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuGetConfig() error = %v", err)
	}

	if config.PackageSources == nil {
		t.Fatal("PackageSources is nil")
	}

	if len(config.PackageSources.Add) != 1 {
		t.Errorf("expected 1 package source, got %d", len(config.PackageSources.Add))
	}

	source := config.PackageSources.Add[0]
	if source.Key != "nuget.org" {
		t.Errorf("source.Key = %q, want %q", source.Key, "nuget.org")
	}

	value := config.GetConfigValue("globalPackagesFolder")
	if value != "~/.nuget/packages" {
		t.Errorf("config value = %q, want %q", value, "~/.nuget/packages")
	}
}

func TestWriteNuGetConfig(t *testing.T) {
	config := NewDefaultConfig()
	config.SetConfigValue("test", "value")

	var buf bytes.Buffer
	if err := WriteNuGetConfig(&buf, config); err != nil {
		t.Fatalf("WriteNuGetConfig() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "configuration") {
		t.Error("output doesn't contain configuration element")
	}
	if !strings.Contains(output, "nuget.org") {
		t.Error("output doesn't contain nuget.org source")
	}
}

func TestNuGetConfig_AddPackageSource(t *testing.T) {
	config := &NuGetConfig{}

	source := PackageSource{
		Key:   "test",
		Value: "https://test.com/v3/index.json",
	}

	config.AddPackageSource(source)

	got := config.GetPackageSource("test")
	if got == nil {
		t.Fatal("GetPackageSource() returned nil")
	}
	if got.Key != "test" {
		t.Errorf("source.Key = %q, want %q", got.Key, "test")
	}
}

func TestNuGetConfig_RemovePackageSource(t *testing.T) {
	config := NewDefaultConfig()

	if !config.RemovePackageSource("nuget.org") {
		t.Error("RemovePackageSource() returned false, want true")
	}

	if config.GetPackageSource("nuget.org") != nil {
		t.Error("source still exists after removal")
	}
}

func TestDefaultConfigLocations(t *testing.T) {
	locations := DefaultConfigLocations()
	if len(locations) == 0 {
		t.Error("DefaultConfigLocations() returned empty slice")
	}

	// Should contain at least current directory and user directory
	foundCurrent := false
	foundUser := false
	for _, loc := range locations {
		if strings.Contains(loc, "NuGet.config") {
			if strings.Contains(loc, ".nuget") {
				foundUser = true
			} else {
				foundCurrent = true
			}
		}
	}

	if !foundCurrent && !foundUser {
		t.Error("DefaultConfigLocations() doesn't contain expected paths")
	}
}

func TestNuGetConfig_UpdatePackageSource(t *testing.T) {
	config := NewDefaultConfig()

	// Update existing source
	updated := PackageSource{
		Key:             "nuget.org",
		Value:           "https://custom.nuget.org/v3/index.json",
		ProtocolVersion: "3",
		Enabled:         "true",
	}

	config.AddPackageSource(updated)

	got := config.GetPackageSource("nuget.org")
	if got == nil {
		t.Fatal("GetPackageSource() returned nil")
	}
	if got.Value != "https://custom.nuget.org/v3/index.json" {
		t.Errorf("source.Value = %q, want %q", got.Value, "https://custom.nuget.org/v3/index.json")
	}

	// Should still have only one source
	if len(config.PackageSources.Add) != 1 {
		t.Errorf("expected 1 source after update, got %d", len(config.PackageSources.Add))
	}
}

func TestNuGetConfig_SetConfigValue(t *testing.T) {
	config := &NuGetConfig{}

	config.SetConfigValue("key1", "value1")
	config.SetConfigValue("key2", "value2")

	if got := config.GetConfigValue("key1"); got != "value1" {
		t.Errorf("GetConfigValue(key1) = %q, want %q", got, "value1")
	}
	if got := config.GetConfigValue("key2"); got != "value2" {
		t.Errorf("GetConfigValue(key2) = %q, want %q", got, "value2")
	}

	// Update existing
	config.SetConfigValue("key1", "updated")
	if got := config.GetConfigValue("key1"); got != "updated" {
		t.Errorf("GetConfigValue(key1) after update = %q, want %q", got, "updated")
	}

	// Should have 2 items
	if len(config.Config.Add) != 2 {
		t.Errorf("expected 2 config items, got %d", len(config.Config.Add))
	}
}

func TestNuGetConfig_GetPackageSource_Nil(t *testing.T) {
	config := &NuGetConfig{}

	if got := config.GetPackageSource("test"); got != nil {
		t.Error("GetPackageSource() should return nil for empty config")
	}
}

func TestNuGetConfig_RemovePackageSource_NotFound(t *testing.T) {
	config := NewDefaultConfig()

	if config.RemovePackageSource("nonexistent") {
		t.Error("RemovePackageSource() returned true for nonexistent source")
	}
}

func TestNuGetConfig_GetConfigValue_Empty(t *testing.T) {
	config := &NuGetConfig{}

	if got := config.GetConfigValue("test"); got != "" {
		t.Errorf("GetConfigValue() = %q, want empty string", got)
	}
}

func TestParseNuGetConfig_WithAPIKeys(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <apikeys>
    <add key="https://api.nuget.org/v3/index.json" value="encrypted-key" />
  </apikeys>
</configuration>`

	config, err := ParseNuGetConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuGetConfig() error = %v", err)
	}

	if config.APIKeys == nil {
		t.Fatal("APIKeys is nil")
	}

	if len(config.APIKeys.Add) != 1 {
		t.Errorf("expected 1 API key, got %d", len(config.APIKeys.Add))
	}
}

func TestRoundTrip(t *testing.T) {
	// Create config
	config := NewDefaultConfig()
	config.SetConfigValue("globalPackagesFolder", "~/.nuget/packages")

	// Write to buffer
	var buf bytes.Buffer
	if err := WriteNuGetConfig(&buf, config); err != nil {
		t.Fatalf("WriteNuGetConfig() error = %v", err)
	}

	// Parse back
	parsed, err := ParseNuGetConfig(&buf)
	if err != nil {
		t.Fatalf("ParseNuGetConfig() error = %v", err)
	}

	// Verify
	if parsed.GetConfigValue("globalPackagesFolder") != "~/.nuget/packages" {
		t.Error("round-trip failed to preserve config value")
	}

	if len(parsed.PackageSources.Add) != 1 {
		t.Errorf("round-trip failed to preserve package sources, got %d", len(parsed.PackageSources.Add))
	}
}

func TestGetUserConfigPath(t *testing.T) {
	path := GetUserConfigPath()
	if path == "" {
		t.Error("GetUserConfigPath() returned empty string")
	}

	// On Windows: %APPDATA%\NuGet\NuGet.Config
	// On Unix: ~/.nuget/NuGet/NuGet.Config
	if !strings.Contains(path, "NuGet") {
		t.Errorf("GetUserConfigPath() = %q, should contain NuGet", path)
	}
}

func TestDefaultPackageSources(t *testing.T) {
	sources := DefaultPackageSources()
	if len(sources) == 0 {
		t.Fatal("DefaultPackageSources() returned empty slice")
	}

	// Should have nuget.org
	found := false
	for _, source := range sources {
		if source.Key == "nuget.org" {
			found = true
			if source.Value != "https://api.nuget.org/v3/index.json" {
				t.Errorf("nuget.org value = %q, want https://api.nuget.org/v3/index.json", source.Value)
			}
			if source.ProtocolVersion != "3" {
				t.Errorf("nuget.org protocolVersion = %q, want 3", source.ProtocolVersion)
			}
		}
	}

	if !found {
		t.Error("DefaultPackageSources() doesn't contain nuget.org")
	}
}

func TestLoadAndSaveNuGetConfig(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/NuGet.config"

	// Save config
	config := NewDefaultConfig()
	config.SetConfigValue("testKey", "testValue")

	if err := SaveNuGetConfig(configPath, config); err != nil {
		t.Fatalf("SaveNuGetConfig() error = %v", err)
	}

	// Load it back
	loaded, err := LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("LoadNuGetConfig() error = %v", err)
	}

	if loaded.GetConfigValue("testKey") != "testValue" {
		t.Error("loaded config doesn't contain expected value")
	}
}

func TestFindConfigFile(t *testing.T) {
	// This test just verifies the function doesn't panic
	// It may or may not find a file depending on the environment
	_ = FindConfigFile()
}

func TestParseNuGetConfig_InvalidXML(t *testing.T) {
	xml := `not valid xml`

	_, err := ParseNuGetConfig(strings.NewReader(xml))
	if err == nil {
		t.Error("ParseNuGetConfig() should return error for invalid XML")
	}
}

func TestLoadNuGetConfig_NotFound(t *testing.T) {
	_, err := LoadNuGetConfig("/nonexistent/path/NuGet.config")
	if err == nil {
		t.Error("LoadNuGetConfig() should return error for nonexistent file")
	}
}

func TestSaveNuGetConfig_CreateDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/subdir/NuGet.config"

	config := NewDefaultConfig()
	if err := SaveNuGetConfig(configPath, config); err != nil {
		t.Fatalf("SaveNuGetConfig() error = %v", err)
	}

	// Verify directory was created
	if _, err := LoadNuGetConfig(configPath); err != nil {
		t.Errorf("failed to load saved config: %v", err)
	}
}

func TestNuGetConfig_RemovePackageSource_NilSources(t *testing.T) {
	config := &NuGetConfig{
		PackageSources: nil,
	}

	if config.RemovePackageSource("test") {
		t.Error("RemovePackageSource() should return false when PackageSources is nil")
	}
}

func TestNuGetConfig_GetConfigValue_NilConfig(t *testing.T) {
	config := &NuGetConfig{
		Config: nil,
	}

	if got := config.GetConfigValue("test"); got != "" {
		t.Errorf("GetConfigValue() = %q, want empty string when Config is nil", got)
	}
}

func TestNuGetConfig_AddPackageSource_MultipleUpdates(t *testing.T) {
	config := &NuGetConfig{}

	// Add first source
	config.AddPackageSource(PackageSource{Key: "source1", Value: "url1"})

	// Add second source
	config.AddPackageSource(PackageSource{Key: "source2", Value: "url2"})

	// Update first source
	config.AddPackageSource(PackageSource{Key: "source1", Value: "updated-url1"})

	if len(config.PackageSources.Add) != 2 {
		t.Errorf("expected 2 sources, got %d", len(config.PackageSources.Add))
	}

	source1 := config.GetPackageSource("source1")
	if source1 == nil || source1.Value != "updated-url1" {
		t.Error("source1 should be updated")
	}

	source2 := config.GetPackageSource("source2")
	if source2 == nil || source2.Value != "url2" {
		t.Error("source2 should remain unchanged")
	}
}

func TestFindConfigFile_WithExisting(t *testing.T) {
	// Create a temp config in user's home directory structure
	tmpDir := t.TempDir()

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	_ = os.Chdir(tmpDir)

	// Create config in current directory
	configPath := tmpDir + "/NuGet.config"
	config := NewDefaultConfig()
	if err := SaveNuGetConfig(configPath, config); err != nil {
		t.Fatalf("SaveNuGetConfig() error = %v", err)
	}

	// FindConfigFile should find it in current directory
	found := FindConfigFile()
	if found == "" {
		t.Error("FindConfigFile() should find config in current directory")
	}
}

func TestParseNuGetConfig_EmptyConfig(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
</configuration>`

	config, err := ParseNuGetConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuGetConfig() error = %v", err)
	}

	if config == nil {
		t.Fatal("config should not be nil")
	}

	// Should have nil sections
	if config.PackageSources != nil && len(config.PackageSources.Add) > 0 {
		t.Error("empty config should have no sources")
	}
}

func TestWriteNuGetConfig_ComplexConfig(t *testing.T) {
	config := &NuGetConfig{
		PackageSources: &PackageSources{
			Add: []PackageSource{
				{Key: "source1", Value: "url1", ProtocolVersion: "3", Enabled: "true"},
				{Key: "source2", Value: "url2", ProtocolVersion: "2"},
			},
		},
		APIKeys: &APIKeys{
			Add: []APIKey{
				{Key: "source1", Value: "key1"},
			},
		},
		Config: &ConfigSection{
			Add: []ConfigItem{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteNuGetConfig(&buf, config); err != nil {
		t.Fatalf("WriteNuGetConfig() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "source1") {
		t.Error("output should contain source1")
	}
	if !strings.Contains(output, "apikeys") {
		t.Error("output should contain apikeys section")
	}
	if !strings.Contains(output, "key1") {
		t.Error("output should contain config keys")
	}
}

func TestDefaultConfigLocations_NotEmpty(t *testing.T) {
	locations := DefaultConfigLocations()

	if len(locations) < 2 {
		t.Errorf("DefaultConfigLocations() returned %d locations, want at least 2", len(locations))
	}

	// Should contain at least one path with NuGet.config
	hasNuGetConfig := false
	for _, loc := range locations {
		if strings.Contains(loc, "NuGet.config") {
			hasNuGetConfig = true
			break
		}
	}

	if !hasNuGetConfig {
		t.Error("DefaultConfigLocations() should contain paths with NuGet.config")
	}
}
