// cmd/gonuget/commands/config_test.go
package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// Config Get Tests

func TestConfigGet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("testKey", "testValue")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	// Set working directory to temp dir so config is found
	opts := &configGetOptions{workingDirectory: tmpDir}
	if err := runConfigGet(console, "testKey", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	if result != "testValue" {
		t.Errorf("output = %q, want %q", result, "testValue")
	}
}

func TestConfigGet_All(t *testing.T) {
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

	opts := &configGetOptions{workingDirectory: tmpDir}
	if err := runConfigGet(console, "all", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := out.String()
	// New format shows sections with indented key-value pairs
	if !strings.Contains(result, "packageSources:") {
		t.Errorf("output should contain 'packageSources:' section")
	}
	if !strings.Contains(result, "config:") {
		t.Errorf("output should contain 'config:' section")
	}
	if !strings.Contains(result, `key="key1" value="value1"`) {
		t.Errorf("output should contain key1=value1 in structured format")
	}
	if !strings.Contains(result, `key="key2" value="value2"`) {
		t.Errorf("output should contain key2=value2 in structured format")
	}
}

func TestConfigGet_NotFound(t *testing.T) {
	// This test validates that runConfigGet calls os.Exit(2) for missing keys
	// We use a subprocess pattern to test this behavior
	if os.Getenv("TEST_CONFIG_GET_NOT_FOUND") == "1" {
		tmpDir := os.Getenv("TEST_TEMP_DIR")

		var out bytes.Buffer
		console := output.NewConsole(&out, &out, output.VerbosityNormal)

		opts := &configGetOptions{workingDirectory: tmpDir}
		// This should call os.Exit(2) - ignore error since we expect exit
		_ = runConfigGet(console, "repositoryPath", opts)
		return
	}

	// Parent test process
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigGet_NotFound")
	cmd.Env = append(os.Environ(), "TEST_CONFIG_GET_NOT_FOUND=1", "TEST_TEMP_DIR="+tmpDir)
	err := cmd.Run()

	// Check that it exited with code 2
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2, got %d", exitErr.ExitCode())
		}
	} else {
		t.Errorf("Expected process to exit with code 2, got: %v", err)
	}
}

func TestConfigGet_ShowPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("relativePath", "./packages")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configGetOptions{
		workingDirectory: tmpDir,
		showPath:         true,
	}
	if err := runConfigGet(console, "relativePath", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	// Dotnet format: <value><TAB>file: <config-path>
	if !strings.Contains(result, "./packages") {
		t.Errorf("output should contain value './packages', got: %s", result)
	}
	if !strings.Contains(result, "\tfile: ") {
		t.Errorf("output should contain '\\tfile: ', got: %s", result)
	}
	if !strings.Contains(result, configPath) {
		t.Errorf("output should contain config path %s, got: %s", configPath, result)
	}
}

func TestConfigGet_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Use a real absolute path that works cross-platform
	absPath := filepath.Join(tmpDir, "absolute", "path")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("absPath", absPath)
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configGetOptions{
		workingDirectory: tmpDir,
		showPath:         true,
	}
	if err := runConfigGet(console, "absPath", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	// Dotnet format: <value><TAB>file: <config-path>
	if !strings.Contains(result, absPath) {
		t.Errorf("output should contain value '%s', got: %s", absPath, result)
	}
	if !strings.Contains(result, "\tfile: ") {
		t.Errorf("output should contain '\\tfile: ', got: %s", result)
	}
	if !strings.Contains(result, configPath) {
		t.Errorf("output should contain config file path %s, got: %s", configPath, result)
	}
}

// Config Set Tests

func TestConfigSet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create initial config file (dotnet nuget requires file to exist)
	cfg := config.NewDefaultConfig()
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to create initial config: %v", err)
	}

	// Change to temp dir so FindConfigFile() finds our test config
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	// Use valid config key
	if err := runConfigSet(console, "repositoryPath", "~/packages"); err != nil {
		t.Fatalf("runConfigSet() error = %v", err)
	}

	// Verify config was saved
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	value := cfg.GetConfigValue("repositoryPath")
	if value != "~/packages" {
		t.Errorf("config value = %q, want %q", value, "~/packages")
	}

	// Verify output message
	if !strings.Contains(out.String(), configPath) {
		t.Errorf("output should mention config file path")
	}
}

func TestConfigSet_Update(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config with initial value
	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("repositoryPath", "~/old-packages")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Change to temp dir so FindConfigFile() finds our test config
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	// Update value
	if err := runConfigSet(console, "repositoryPath", "~/new-packages"); err != nil {
		t.Fatalf("runConfigSet() error = %v", err)
	}

	// Verify value was updated
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	value := cfg.GetConfigValue("repositoryPath")
	if value != "~/new-packages" {
		t.Errorf("config value = %q, want %q (should be updated)", value, "~/new-packages")
	}
}

// Config Unset Tests

func TestConfigUnset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config with value
	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("repositoryPath", "~/packages")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Change to temp dir so FindConfigFile() finds our test config
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	if err := runConfigUnset(console, "repositoryPath"); err != nil {
		t.Fatalf("runConfigUnset() error = %v", err)
	}

	// Verify value was removed
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	value := cfg.GetConfigValue("repositoryPath")
	if value != "" {
		t.Errorf("config value should be empty after unset, got: %q", value)
	}

	// Verify output message
	if !strings.Contains(out.String(), configPath) {
		t.Errorf("output should mention config file path")
	}
}

func TestConfigUnset_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config without the key
	cfg := config.NewDefaultConfig()
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Change to temp dir so FindConfigFile() finds our test config
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	// Should not error even if key doesn't exist (matches dotnet nuget behavior)
	// Use a valid config key that just doesn't have a value set
	if err := runConfigUnset(console, "repositoryPath"); err != nil {
		t.Fatalf("runConfigUnset() should not error for nonexistent key, got: %v", err)
	}
}

// Config Paths Tests

func TestConfigPaths(t *testing.T) {
	tmpDir := t.TempDir()

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configPathsOptions{workingDirectory: tmpDir}
	if err := runConfigPaths(console, opts); err != nil {
		t.Fatalf("runConfigPaths() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "NuGet configuration file paths:") {
		t.Error("output should contain header")
	}
}

func TestConfigPaths_Default(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configPathsOptions{}
	if err := runConfigPaths(console, opts); err != nil {
		t.Fatalf("runConfigPaths() error = %v", err)
	}

	result := out.String()
	// Should list at least user config
	if !strings.Contains(result, ".nuget") {
		t.Error("output should contain user config path (.nuget)")
	}
}

// Command Structure Tests

func TestNewConfigCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewConfigCommand(console)
	if cmd == nil {
		t.Fatal("NewConfigCommand() returned nil")
	}

	if cmd.Use != "config" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "config")
	}

	if cmd.Short == "" {
		t.Error("cmd.Short is empty")
	}

	// Check subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) != 4 {
		t.Errorf("expected 4 subcommands, got %d", len(subcommands))
	}

	// Verify subcommand names
	var foundGet, foundSet, foundUnset, foundPaths bool
	for _, subcmd := range subcommands {
		switch subcmd.Use {
		case "get <all-or-config-key>":
			foundGet = true
		case "set <config-key> <config-value>":
			foundSet = true
		case "unset <config-key>":
			foundUnset = true
		case "paths":
			foundPaths = true
		}
	}

	if !foundGet {
		t.Error("missing 'get' subcommand")
	}
	if !foundSet {
		t.Error("missing 'set' subcommand")
	}
	if !foundUnset {
		t.Error("missing 'unset' subcommand")
	}
	if !foundPaths {
		t.Error("missing 'paths' subcommand")
	}
}

func TestNewConfigGetCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := newConfigGetCommand(console)
	if cmd == nil {
		t.Fatal("newConfigGetCommand() returned nil")
	}

	if cmd.Use != "get <all-or-config-key>" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "get <all-or-config-key>")
	}

	// Check flags
	if cmd.Flags().Lookup("working-directory") == nil {
		t.Error("missing --working-directory flag")
	}
	if cmd.Flags().Lookup("show-path") == nil {
		t.Error("missing --show-path flag")
	}
}

func TestNewConfigSetCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := newConfigSetCommand(console)
	if cmd == nil {
		t.Fatal("newConfigSetCommand() returned nil")
	}

	if cmd.Use != "set <config-key> <config-value>" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "set <config-key> <config-value>")
	}
}

func TestNewConfigUnsetCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := newConfigUnsetCommand(console)
	if cmd == nil {
		t.Fatal("newConfigUnsetCommand() returned nil")
	}

	if cmd.Use != "unset <config-key>" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "unset <config-key>")
	}
}

func TestNewConfigPathsCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := newConfigPathsCommand(console)
	if cmd == nil {
		t.Fatal("newConfigPathsCommand() returned nil")
	}

	if cmd.Use != "paths" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "paths")
	}

	// Check flags
	if cmd.Flags().Lookup("working-directory") == nil {
		t.Error("missing --working-directory flag")
	}
}

// Helper Function Tests

func TestListAllConfig_Empty(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cfg := config.NewDefaultConfig()
	cfg.Config = nil // Force nil config section
	cfg.PackageSources = nil // Remove default nuget.org source

	if err := listAllConfig(console, cfg); err != nil {
		t.Fatalf("listAllConfig() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "No configuration values found.") {
		t.Errorf("expected 'No configuration values found.' message, got: %s", result)
	}
}

func TestListAllConfig_EmptyItems(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cfg := &config.NuGetConfig{
		Config: &config.ConfigSection{
			Add: []config.ConfigItem{},
		},
	}

	if err := listAllConfig(console, cfg); err != nil {
		t.Fatalf("listAllConfig() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "No configuration values found.") {
		t.Errorf("expected 'No configuration values found.' for empty items")
	}
}

func TestListAllConfig_WithItems(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("key1", "value1")
	cfg.SetConfigValue("key2", "value2")

	if err := listAllConfig(console, cfg); err != nil {
		t.Fatalf("listAllConfig() error = %v", err)
	}

	result := out.String()
	// Output format: add key="<key>" value="<value>"
	if !strings.Contains(result, `key="key1" value="value1"`) {
		t.Errorf("output should contain 'key=\"key1\" value=\"value1\"', got: %s", result)
	}
	if !strings.Contains(result, `key="key2" value="value2"`) {
		t.Errorf("output should contain 'key=\"key2\" value=\"value2\"', got: %s", result)
	}
}

func TestLoadOrCreateConfig_Load(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config
	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("test", "value")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load it
	loaded, err := loadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("loadOrCreateConfig() error = %v", err)
	}

	if loaded.GetConfigValue("test") != "value" {
		t.Error("loadOrCreateConfig() should load existing config")
	}
}

func TestLoadOrCreateConfig_Create(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistent := filepath.Join(tmpDir, "nonexistent.config")

	// Should return error for nonexistent config (doesn't actually create)
	_, err := loadOrCreateConfig(nonexistent)
	if err == nil {
		t.Error("loadOrCreateConfig() should return error for nonexistent file")
	}
}

func TestDetermineConfigPath_WithWorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config
	cfg := config.NewDefaultConfig()
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Should find config from working directory
	found := determineConfigPath(tmpDir)
	if found != configPath {
		t.Errorf("determineConfigPath() = %q, want %q", found, configPath)
	}
}

func TestDetermineConfigPath_Default(t *testing.T) {
	// Should fall back to user config
	found := determineConfigPath("")
	if found == "" {
		t.Error("determineConfigPath() should return user config path")
	}
	if !strings.Contains(found, ".nuget") {
		t.Errorf("determineConfigPath() should contain .nuget, got: %s", found)
	}
}
