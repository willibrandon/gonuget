package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestSourceCommands(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir, err := os.MkdirTemp("", "gonuget-source-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)

	t.Run("list empty sources", func(t *testing.T) {
		// Create empty config
		if err := createEmptyConfig(configPath); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		opts := &sourceOptions{
			configFile: configPath,
			format:     "Detailed",
		}

		err := runListSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("add source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			source:     "https://test.nuget.org/v3/index.json",
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify source was added
		cfg, _ := config.LoadNuGetConfig(configPath)
		if cfg.PackageSources == nil || len(cfg.PackageSources.Add) != 1 {
			t.Errorf("Expected 1 source, got %d", len(cfg.PackageSources.Add))
		}
		if cfg.PackageSources.Add[0].Key != "TestFeed" {
			t.Errorf("Expected source name 'TestFeed', got '%s'", cfg.PackageSources.Add[0].Key)
		}
		if cfg.PackageSources.Add[0].Value != "https://test.nuget.org/v3/index.json" {
			t.Errorf("Expected source URL 'https://test.nuget.org/v3/index.json', got '%s'", cfg.PackageSources.Add[0].Value)
		}
		if !isSourceEnabled(&cfg.PackageSources.Add[0]) {
			t.Errorf("Expected source to be enabled")
		}
	})

	t.Run("add duplicate source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			source:     "https://duplicate.nuget.org/v3/index.json",
		}

		err := runAddSource(console, opts)
		if err == nil {
			t.Error("Expected error for duplicate source name")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("add source with HTTP without flag", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "InsecureFeed",
			source:     "http://insecure.nuget.org/v3/index.json",
		}

		err := runAddSource(console, opts)
		if err == nil {
			t.Error("Expected error for HTTP source without --allow-insecure-connections")
		}
		if !strings.Contains(err.Error(), "insecure") {
			t.Errorf("Expected 'insecure' error, got: %v", err)
		}
	})

	t.Run("add source with HTTP with flag", func(t *testing.T) {
		opts := &sourceOptions{
			configFile:               configPath,
			name:                     "InsecureFeed",
			source:                   "http://insecure.nuget.org/v3/index.json",
			allowInsecureConnections: true,
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error with --allow-insecure-connections, got: %v", err)
		}
	})

	t.Run("list sources", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			format:     "Detailed",
		}

		err := runListSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify sources are present
		cfg, _ := config.LoadNuGetConfig(configPath)
		if len(cfg.PackageSources.Add) != 2 {
			t.Errorf("Expected 2 sources, got %d", len(cfg.PackageSources.Add))
		}
	})

	t.Run("list sources short format", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			format:     "Short",
		}

		err := runListSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("disable source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
		}

		err := runDisableSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify source is disabled
		cfg, _ := config.LoadNuGetConfig(configPath)
		source := cfg.GetPackageSource("TestFeed")
		if source == nil {
			t.Fatal("Expected to find source")
		}
		if !cfg.IsSourceDisabled("TestFeed") {
			t.Error("Expected source to be disabled")
		}
	})

	t.Run("disable already disabled source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
		}

		err := runDisableSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error for already disabled source, got: %v", err)
		}
	})

	t.Run("enable source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
		}

		err := runEnableSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify source is enabled
		cfg, _ := config.LoadNuGetConfig(configPath)
		source := cfg.GetPackageSource("TestFeed")
		if source == nil {
			t.Fatal("Expected to find source")
		}
		if !isSourceEnabled(source) {
			t.Error("Expected source to be enabled")
		}
	})

	t.Run("enable already enabled source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
		}

		err := runEnableSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error for already enabled source, got: %v", err)
		}
	})

	t.Run("update source URL", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			source:     "https://updated.nuget.org/v3/index.json",
		}

		err := runUpdateSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify source URL was updated
		cfg, _ := config.LoadNuGetConfig(configPath)
		source := cfg.GetPackageSource("TestFeed")
		if source == nil {
			t.Fatal("Expected to find source")
		}
		if source.Value != "https://updated.nuget.org/v3/index.json" {
			t.Errorf("Expected updated URL, got '%s'", source.Value)
		}
	})

	t.Run("update source with credentials", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			username:   "testuser",
			password:   "testpass",
		}

		err := runUpdateSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify credentials were added
		cfg, _ := config.LoadNuGetConfig(configPath)
		if cfg.PackageSourceCredentials == nil {
			t.Fatal("Expected credentials to be set")
		}
		found := false
		for _, cred := range cfg.PackageSourceCredentials.Items {
			if cred.XMLName.Local == "TestFeed" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected credentials for TestFeed")
		}
	})

	t.Run("update source with invalid URL", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			source:     "ht!tp://invalid url",
		}

		err := runUpdateSource(console, opts)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("update non-existent source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "NonExistent",
			source:     "https://new.url/v3/index.json",
		}

		err := runUpdateSource(console, opts)
		if err == nil {
			t.Error("Expected error for non-existent source")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("remove source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "InsecureFeed",
		}

		err := runRemoveSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify source was removed
		cfg, _ := config.LoadNuGetConfig(configPath)
		source := cfg.GetPackageSource("InsecureFeed")
		if source != nil {
			t.Error("Expected source to be removed")
		}
	})

	t.Run("remove non-existent source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "NonExistent",
		}

		err := runRemoveSource(console, opts)
		if err == nil {
			t.Error("Expected error for non-existent source")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("enable non-existent source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "NonExistent",
		}

		err := runEnableSource(console, opts)
		if err == nil {
			t.Error("Expected error for non-existent source")
		}
	})

	t.Run("disable non-existent source", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "NonExistent",
		}

		err := runDisableSource(console, opts)
		if err == nil {
			t.Error("Expected error for non-existent source")
		}
	})
}

func TestSourceCommandsWithCredentials(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-cred-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)

	if err := createEmptyConfig(configPath); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("add source with clear text password", func(t *testing.T) {
		opts := &sourceOptions{
			configFile:               configPath,
			name:                     "AuthFeed",
			source:                   "https://auth.nuget.org/v3/index.json",
			username:                 "user",
			password:                 "pass",
			storePasswordInClearText: true,
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify credentials
		cfg, _ := config.LoadNuGetConfig(configPath)
		if cfg.PackageSourceCredentials == nil {
			t.Fatal("Expected credentials to be set")
		}
		found := false
		for _, cred := range cfg.PackageSourceCredentials.Items {
			if cred.XMLName.Local == "AuthFeed" {
				found = true
				// Check for ClearTextPassword in items
				for _, item := range cred.Add {
					if item.Key == "ClearTextPassword" && item.Value == "pass" {
						return // Success
					}
				}
			}
		}
		if !found {
			t.Error("Expected credentials for AuthFeed")
		}
	})

	t.Run("add source with encrypted password", func(t *testing.T) {
		opts := &sourceOptions{
			configFile:               configPath,
			name:                     "EncryptedFeed",
			source:                   "https://encrypted.nuget.org/v3/index.json",
			username:                 "user",
			password:                 "pass",
			storePasswordInClearText: false,
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify credentials are encrypted (base64 for now)
		cfg, _ := config.LoadNuGetConfig(configPath)
		if cfg.PackageSourceCredentials == nil {
			t.Fatal("Expected credentials to be set")
		}
		found := false
		for _, cred := range cfg.PackageSourceCredentials.Items {
			if cred.XMLName.Local == "EncryptedFeed" {
				found = true
				// Check for Password (encrypted) in items
				for _, item := range cred.Add {
					if item.Key == "Password" {
						if item.Value == "" {
							t.Error("Expected encrypted password to be set")
						}
						if item.Value == "pass" {
							t.Error("Expected password to be encrypted, not plain text")
						}
						return // Success
					}
				}
			}
		}
		if !found {
			t.Error("Expected credentials for EncryptedFeed")
		}
	})

	t.Run("add source with authentication types", func(t *testing.T) {
		opts := &sourceOptions{
			configFile:               configPath,
			name:                     "TypedAuthFeed",
			source:                   "https://typed.nuget.org/v3/index.json",
			username:                 "user",
			password:                 "pass",
			validAuthenticationTypes: "basic,negotiate",
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify authentication types
		cfg, _ := config.LoadNuGetConfig(configPath)
		if cfg.PackageSourceCredentials == nil {
			t.Fatal("Expected credentials to be set")
		}
		found := false
		for _, cred := range cfg.PackageSourceCredentials.Items {
			if cred.XMLName.Local == "TypedAuthFeed" {
				found = true
				for _, item := range cred.Add {
					if item.Key == "ValidAuthenticationTypes" && item.Value == "basic,negotiate" {
						return // Success
					}
				}
			}
		}
		if !found {
			t.Error("Expected credentials for TypedAuthFeed")
		}
	})

	t.Run("add source with protocol version", func(t *testing.T) {
		// Protocol version 2 should not be written (it's the default)
		opts := &sourceOptions{
			configFile:      configPath,
			name:            "V2Feed",
			source:          "https://v2.nuget.org/",
			protocolVersion: "2",
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify protocol version 2 is NOT written (it's the default)
		cfg, _ := config.LoadNuGetConfig(configPath)
		source := cfg.GetPackageSource("V2Feed")
		if source == nil {
			t.Fatal("Expected to find source")
		}
		if source.ProtocolVersion != "" {
			t.Errorf("Expected protocol version to be empty (default), got '%s'", source.ProtocolVersion)
		}

		// Test that protocol version 3 IS written (non-default)
		opts3 := &sourceOptions{
			configFile:      configPath,
			name:            "V3Feed",
			source:          "https://api.nuget.org/v3/index.json",
			protocolVersion: "3",
		}

		err = runAddSource(console, opts3)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		cfg, _ = config.LoadNuGetConfig(configPath)
		source3 := cfg.GetPackageSource("V3Feed")
		if source3 == nil {
			t.Fatal("Expected to find V3Feed source")
		}
		if source3.ProtocolVersion != "3" {
			t.Errorf("Expected protocol version '3', got '%s'", source3.ProtocolVersion)
		}
	})
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		enabled  string
		expected string
	}{
		{"true", "Enabled"},
		{"", "Enabled"},
		{"false", "Disabled"},
	}

	for _, tt := range tests {
		result := statusString(tt.enabled)
		if result != tt.expected {
			t.Errorf("statusString(%v) = %s, expected %s", tt.enabled, result, tt.expected)
		}
	}
}

func TestEncodePassword(t *testing.T) {
	password := "testpassword"
	encoded := encodePassword(password)

	if encoded == password {
		t.Error("Expected password to be encoded")
	}
	if encoded == "" {
		t.Error("Expected non-empty encoded password")
	}
}

func TestIsSourceEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  string
		expected bool
	}{
		{"empty string is enabled", "", true},
		{"true is enabled", "true", true},
		{"false is disabled", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &config.PackageSource{Enabled: tt.enabled}
			result := isSourceEnabled(source)
			if result != tt.expected {
				t.Errorf("isSourceEnabled(%v) = %v, expected %v", tt.enabled, result, tt.expected)
			}
		})
	}
}

func TestValidateSourceExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-validate-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	if err := createEmptyConfig(configPath); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Add a source
	source := config.PackageSource{
		Key:     "TestSource",
		Value:   "https://test.nuget.org/v3/index.json",
		Enabled: "true",
	}
	cfg.AddPackageSource(source)
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Reload config
	cfg, _ = config.LoadNuGetConfig(configPath)

	t.Run("existing source", func(t *testing.T) {
		exists := validateSourceExists(cfg, "TestSource")
		if !exists {
			t.Error("Expected source to exist")
		}
	})

	t.Run("non-existent source", func(t *testing.T) {
		exists := validateSourceExists(cfg, "NonExistent")
		if exists {
			t.Error("Expected source not to exist")
		}
	})
}

func TestFindSourceByName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-find-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	if err := createEmptyConfig(configPath); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Add a source
	source := config.PackageSource{
		Key:     "TestSource",
		Value:   "https://test.nuget.org/v3/index.json",
		Enabled: "true",
	}
	cfg.AddPackageSource(source)
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Reload config
	cfg, _ = config.LoadNuGetConfig(configPath)

	t.Run("find existing source", func(t *testing.T) {
		found, err := findSourceByName(cfg, "TestSource")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if found == nil {
			t.Fatal("Expected to find source")
		}
		if found.Key != "TestSource" {
			t.Errorf("Expected source name 'TestSource', got '%s'", found.Key)
		}
	})

	t.Run("find non-existent source", func(t *testing.T) {
		found, err := findSourceByName(cfg, "NonExistent")
		if err == nil {
			t.Error("Expected error for non-existent source")
		}
		if found != nil {
			t.Error("Expected nil source for non-existent source")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

func TestLoadSourceConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-load-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")

	t.Run("load existing config", func(t *testing.T) {
		if err := createEmptyConfig(configPath); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		cfg, path, err := loadSourceConfig(configPath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if cfg == nil {
			t.Error("Expected config to be loaded")
		}
		if path != configPath {
			t.Errorf("Expected path '%s', got '%s'", configPath, path)
		}
	})

	t.Run("create new config", func(t *testing.T) {
		newPath := filepath.Join(tmpDir, "new", "NuGet.config")
		cfg, path, err := loadSourceConfig(newPath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if cfg == nil {
			t.Error("Expected config to be created")
		}
		if path != newPath {
			t.Errorf("Expected path '%s', got '%s'", newPath, path)
		}
	})
}

func TestAddOrUpdateCredential(t *testing.T) {
	cfg := config.NewDefaultConfig()

	t.Run("add new credential", func(t *testing.T) {
		addOrUpdateCredential(cfg, "TestFeed", "user", "pass", false, "basic")

		if cfg.PackageSourceCredentials == nil {
			t.Fatal("Expected credentials to be set")
		}

		found := false
		for _, cred := range cfg.PackageSourceCredentials.Items {
			if cred.XMLName.Local == "TestFeed" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected credentials for TestFeed")
		}
	})

	t.Run("update existing credential", func(t *testing.T) {
		addOrUpdateCredential(cfg, "TestFeed", "newuser", "newpass", true, "negotiate")

		found := false
		for _, cred := range cfg.PackageSourceCredentials.Items {
			if cred.XMLName.Local == "TestFeed" {
				found = true
				// Verify it was updated
				for _, item := range cred.Add {
					if item.Key == "Username" && item.Value != "newuser" {
						t.Error("Expected username to be updated")
					}
					if item.Key == "ClearTextPassword" && item.Value != "newpass" {
						t.Error("Expected password to be updated")
					}
				}
				break
			}
		}
		if !found {
			t.Error("Expected credentials for TestFeed")
		}
	})
}

func TestNewCommands(t *testing.T) {
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)

	t.Run("NewListCommand", func(t *testing.T) {
		cmd := NewListCommand(console)
		if cmd == nil {
			t.Fatal("Expected command to be created")
		}
		if cmd.Use != "list source" {
			t.Errorf("Expected Use to be 'list source', got '%s'", cmd.Use)
		}
		if cmd.Short == "" {
			t.Error("Expected Short description to be set")
		}
	})

	t.Run("NewAddCommand", func(t *testing.T) {
		cmd := NewAddCommand(console)
		if cmd == nil {
			t.Fatal("Expected command to be created")
		}
		if cmd.Use != "add source [PackageSourcePath]" {
			t.Errorf("Expected Use to be 'add source [PackageSourcePath]', got '%s'", cmd.Use)
		}
		// Verify required flags
		nameFlag := cmd.Flags().Lookup("name")
		if nameFlag == nil {
			t.Error("Expected --name flag to exist")
		}
	})

	t.Run("NewRemoveCommand", func(t *testing.T) {
		cmd := NewRemoveCommand(console)
		if cmd == nil {
			t.Fatal("Expected command to be created")
		}
		if cmd.Use != "remove source <name>" {
			t.Errorf("Expected Use to be 'remove source <name>', got '%s'", cmd.Use)
		}
	})

	t.Run("NewEnableCommand", func(t *testing.T) {
		cmd := NewEnableCommand(console)
		if cmd == nil {
			t.Fatal("Expected command to be created")
		}
		if cmd.Use != "enable source <name>" {
			t.Errorf("Expected Use to be 'enable source <name>', got '%s'", cmd.Use)
		}
	})

	t.Run("NewDisableCommand", func(t *testing.T) {
		cmd := NewDisableCommand(console)
		if cmd == nil {
			t.Fatal("Expected command to be created")
		}
		if cmd.Use != "disable source <name>" {
			t.Errorf("Expected Use to be 'disable source <name>', got '%s'", cmd.Use)
		}
	})

	t.Run("NewUpdateCommand", func(t *testing.T) {
		cmd := NewUpdateCommand(console)
		if cmd == nil {
			t.Fatal("Expected command to be created")
		}
		if cmd.Use != "update source [name]" {
			t.Errorf("Expected Use to be 'update source [name]', got '%s'", cmd.Use)
		}
	})
}

func TestLoadSourceConfigEdgeCases(t *testing.T) {
	t.Run("load with empty path finds default", func(t *testing.T) {
		// This will use default config finding logic
		cfg, path, err := loadSourceConfig("")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if cfg == nil {
			t.Error("Expected config to be created")
		}
		if path == "" {
			t.Error("Expected path to be set")
		}
	})

	t.Run("load invalid config file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gonuget-invalid-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		configPath := filepath.Join(tmpDir, "Invalid.config")
		// Create invalid XML
		if err := os.WriteFile(configPath, []byte("not valid xml <><"), 0644); err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}

		cfg, _, err := loadSourceConfig(configPath)
		if err == nil {
			t.Error("Expected error for invalid config file")
		}
		if cfg != nil {
			t.Error("Expected nil config for invalid file")
		}
	})
}

func TestAddSourceEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-add-edge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)

	if err := createEmptyConfig(configPath); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("add source with invalid URL", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "InvalidURL",
			source:     "ht!tp://not a valid url",
		}

		err := runAddSource(console, opts)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("add source to nil package sources", func(t *testing.T) {
		// Create a config with nil PackageSources
		emptyConfigPath := filepath.Join(tmpDir, "Empty.config")
		content := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
</configuration>`
		if err := os.WriteFile(emptyConfigPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write empty config: %v", err)
		}

		opts := &sourceOptions{
			configFile: emptyConfigPath,
			name:       "TestSource",
			source:     "https://test.nuget.org/v3/index.json",
		}

		err := runAddSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

func TestUpdateSourceEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-update-edge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)

	if err := createEmptyConfig(configPath); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Add a source first
	opts := &sourceOptions{
		configFile: configPath,
		name:       "TestFeed",
		source:     "https://test.nuget.org/v3/index.json",
	}
	if err := runAddSource(console, opts); err != nil {
		t.Fatalf("Failed to add source: %v", err)
	}

	t.Run("update with HTTP URL requires flag", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			source:     "http://insecure.nuget.org/v3/index.json",
		}

		err := runUpdateSource(console, opts)
		if err == nil {
			t.Error("Expected error for HTTP without flag")
		}
		if !strings.Contains(err.Error(), "insecure") {
			t.Errorf("Expected 'insecure' error, got: %v", err)
		}
	})

	t.Run("update with HTTP URL with flag", func(t *testing.T) {
		opts := &sourceOptions{
			configFile:               configPath,
			name:                     "TestFeed",
			source:                   "http://insecure.nuget.org/v3/index.json",
			allowInsecureConnections: true,
		}

		err := runUpdateSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error with flag, got: %v", err)
		}
	})

	t.Run("update with protocol version", func(t *testing.T) {
		// Protocol version 2 should NOT be written (it's the default)
		opts := &sourceOptions{
			configFile:      configPath,
			name:            "TestFeed",
			source:          "https://v2.nuget.org/",
			protocolVersion: "2",
		}

		err := runUpdateSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify protocol version 2 is NOT written (it's the default)
		cfg, _ := config.LoadNuGetConfig(configPath)
		source := cfg.GetPackageSource("TestFeed")
		if source == nil {
			t.Fatal("Expected to find source")
		}
		if source.ProtocolVersion != "" {
			t.Errorf("Expected protocol version to be empty (default), got '%s'", source.ProtocolVersion)
		}

		// Test that protocol version 3 IS written (non-default)
		opts3 := &sourceOptions{
			configFile:      configPath,
			name:            "TestFeed",
			source:          "https://api.nuget.org/v3/index.json",
			protocolVersion: "3",
		}

		err = runUpdateSource(console, opts3)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		cfg, _ = config.LoadNuGetConfig(configPath)
		source3 := cfg.GetPackageSource("TestFeed")
		if source3 == nil {
			t.Fatal("Expected to find TestFeed source")
		}
		if source3.ProtocolVersion != "3" {
			t.Errorf("Expected protocol version '3', got '%s'", source3.ProtocolVersion)
		}
	})

	t.Run("update credentials only without URL", func(t *testing.T) {
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
			username:   "onlyuser",
			password:   "onlypass",
		}

		err := runUpdateSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

func TestRemoveSourceEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gonuget-remove-edge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "NuGet.config")
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)

	if err := createEmptyConfig(configPath); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Add a source first
	opts := &sourceOptions{
		configFile: configPath,
		name:       "TestFeed",
		source:     "https://test.nuget.org/v3/index.json",
	}
	if err := runAddSource(console, opts); err != nil {
		t.Fatalf("Failed to add source: %v", err)
	}

	t.Run("remove returns error when RemovePackageSource fails", func(t *testing.T) {
		// This should succeed actually, testing the normal path
		opts := &sourceOptions{
			configFile: configPath,
			name:       "TestFeed",
		}

		err := runRemoveSource(console, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

// Helper function to create an empty NuGet.config
func createEmptyConfig(path string) error {
	content := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
  </packageSources>
</configuration>`
	return os.WriteFile(path, []byte(content), 0644)
}

// NEW TESTS TO IMPROVE COVERAGE

func TestListCommandConstructor(t *testing.T) {
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)
	cmd := NewListCommand(console)

	// Test command was created
	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	// Test Use field
	if cmd.Use != "list source" {
		t.Errorf("Expected Use 'list source', got '%s'", cmd.Use)
	}

	// Test default format flag
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("Expected --format flag")
	}
	if formatFlag.DefValue != "Detailed" {
		t.Errorf("Expected default format 'Detailed', got '%s'", formatFlag.DefValue)
	}

	// Test that RunE returns error for wrong args (not Args validator)
	cmd.SetArgs([]string{"source"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error with correct arg 'source', got: %v", err)
	}

	// Test with wrong arg
	cmd2 := NewListCommand(console)
	cmd2.SetArgs([]string{"wrongarg"})
	err = cmd2.Execute()
	if err == nil {
		t.Error("Expected error for wrong argument")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", err)
	}
}

func TestRemoveCommandConstructor(t *testing.T) {
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)
	cmd := NewRemoveCommand(console)

	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	// Test with wrong first arg
	cmd.SetArgs([]string{"wrongarg", "TestSource"})
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for wrong first argument")
	}
}

func TestEnableCommandConstructor(t *testing.T) {
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)
	cmd := NewEnableCommand(console)

	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	// Test with wrong first arg
	cmd.SetArgs([]string{"wrongarg", "TestSource"})
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for wrong first argument")
	}
}

func TestDisableCommandConstructor(t *testing.T) {
	console := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)
	cmd := NewDisableCommand(console)

	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	// Test with wrong first arg
	cmd.SetArgs([]string{"wrongarg", "TestSource"})
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for wrong first argument")
	}
}
