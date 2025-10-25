// cmd/gonuget/commands/config_test.go
package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestConfigCommand_Get(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("testKey", "testValue")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	if err := runConfig(console, []string{"testKey"}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	if result != "testValue" {
		t.Errorf("output = %q, want %q", result, "testValue")
	}
}

func TestConfigCommand_Set(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	if err := runConfig(console, []string{"newKey", "newValue"}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	// Verify config was saved
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	value := cfg.GetConfigValue("newKey")
	if value != "newValue" {
		t.Errorf("config value = %q, want %q", value, "newValue")
	}
}

func TestConfigCommand_SetMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{
		configFile: configPath,
		set:        []string{"key1=value1", "key2=value2"},
	}
	if err := runConfig(console, []string{}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if got := cfg.GetConfigValue("key1"); got != "value1" {
		t.Errorf("key1 = %q, want %q", got, "value1")
	}
	if got := cfg.GetConfigValue("key2"); got != "value2" {
		t.Errorf("key2 = %q, want %q", got, "value2")
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{"key=value", "key", "value", false},
		{"key=", "key", "", false},
		{"=value", "", "value", false},
		{"invalid", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, value, err := parseKeyValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if value != tt.wantValue {
				t.Errorf("value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

func TestConfigCommand_List(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("key1", "value1")
	cfg.SetConfigValue("key2", "value2")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	if err := runConfig(console, []string{}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "key1") || !strings.Contains(result, "value1") {
		t.Errorf("output doesn't contain key1/value1")
	}
	if !strings.Contains(result, "key2") || !strings.Contains(result, "value2") {
		t.Errorf("output doesn't contain key2/value2")
	}
}

func TestConfigCommand_GetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	err := runConfig(console, []string{"nonexistent"}, opts)
	if err == nil {
		t.Error("runConfig() should return error for nonexistent key")
	}
}

func TestConfigCommand_AsPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("relativePath", "./packages")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath, asPath: true}
	if err := runConfig(console, []string{"relativePath"}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	if !filepath.IsAbs(result) {
		t.Errorf("AsPath should return absolute path, got: %s", result)
	}
}

func TestConfigCommand_TooManyArgs(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	err := runConfig(console, []string{"key", "value", "extra"}, opts)
	if err == nil {
		t.Error("runConfig() should return error for too many arguments")
	}
}

func TestLoadOrCreateConfig(t *testing.T) {
	// Test loading existing config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("test", "value")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := loadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("loadOrCreateConfig() error = %v", err)
	}

	if loaded.GetConfigValue("test") != "value" {
		t.Error("loadOrCreateConfig() should load existing config")
	}

	// Test creating new config
	nonexistent := filepath.Join(tmpDir, "nonexistent.config")
	created, err := loadOrCreateConfig(nonexistent)
	if err != nil {
		t.Fatalf("loadOrCreateConfig() error = %v", err)
	}

	if created == nil {
		t.Error("loadOrCreateConfig() should create new config")
	}
}

func TestNewConfigCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewConfigCommand(console)
	if cmd == nil {
		t.Fatal("NewConfigCommand() returned nil")
	}

	if cmd.Use != "config [key] [value]" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "config [key] [value]")
	}

	if cmd.Short == "" {
		t.Error("cmd.Short is empty")
	}
}

func TestConfigCommand_ListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config with no values
	cfg := config.NewDefaultConfig()
	cfg.Config = nil // Force nil config section
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	if err := runConfig(console, []string{}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "No configuration values set") {
		t.Errorf("expected 'No configuration values set' message, got: %s", result)
	}
}

func TestConfigCommand_GetAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("absPath", "/absolute/path")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath, asPath: true}
	if err := runConfig(console, []string{"absPath"}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	if result != "/absolute/path" {
		t.Errorf("AsPath with absolute path should return as-is, got: %s", result)
	}
}

func TestConfigCommand_SetInvalidKeyValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{
		configFile: configPath,
		set:        []string{"invalidpair"},
	}
	err := runConfig(console, []string{}, opts)
	if err == nil {
		t.Error("runConfig() should return error for invalid key=value pair")
	}
}

func TestConfigCommand_FindConfigFile(t *testing.T) {
	// Test with no config file specified (uses default locations)
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{} // No configFile specified
	// This should use FindConfigFile() and create config if needed
	err := runConfig(console, []string{"someKey", "someValue"}, opts)
	// Error is expected since we're testing the path, but it should not panic
	_ = err
}

func TestListAllConfig_WithEmptyItems(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	// Config with empty Add slice
	cfg := &config.NuGetConfig{
		Config: &config.ConfigSection{
			Add: []config.ConfigItem{},
		},
	}

	if err := listAllConfig(console, cfg); err != nil {
		t.Fatalf("listAllConfig() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "No configuration values set") {
		t.Errorf("expected 'No configuration values set' for empty items")
	}
}

func TestGetConfigValue_PathError(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cfg := config.NewDefaultConfig()
	// Set an invalid path that might cause issues
	cfg.SetConfigValue("testKey", "relative/path")

	// This should work and expand the relative path
	err := getConfigValue(console, cfg, "testKey", true)
	if err != nil {
		t.Fatalf("getConfigValue() error = %v", err)
	}
}
