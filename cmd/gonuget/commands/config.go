// cmd/gonuget/commands/config.go
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewConfigCommand creates the config command with get/set/unset/paths subcommands
func NewConfigCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage NuGet configuration",
		Long: `Gets, sets, unsets, or displays paths for NuGet configuration values.

This command has four subcommands:
  - get:   Get a configuration value (or all with "all")
  - set:   Set a configuration value
  - unset: Remove a configuration value
  - paths: Display configuration file paths

Examples:
  gonuget config get repositoryPath
  gonuget config get all
  gonuget config set repositoryPath ~/packages
  gonuget config unset repositoryPath
  gonuget config paths`,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(newConfigGetCommand(console))
	cmd.AddCommand(newConfigSetCommand(console))
	cmd.AddCommand(newConfigUnsetCommand(console))
	cmd.AddCommand(newConfigPathsCommand(console))

	return cmd
}

// Config Get Subcommand

type configGetOptions struct {
	workingDirectory string
	showPath         bool
}

func newConfigGetCommand(console *output.Console) *cobra.Command {
	opts := &configGetOptions{}

	cmd := &cobra.Command{
		Use:   "get <all-or-config-key>",
		Short: "Get a configuration value",
		Long: `Get a NuGet configuration value by key, or get all values with "all".

Examples:
  gonuget config get repositoryPath
  gonuget config get all
  gonuget config get globalPackagesFolder --show-path`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(console, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.workingDirectory, "working-directory", "", "Working directory for config hierarchy resolution")
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "Return value as filesystem path")

	return cmd
}

func runConfigGet(console *output.Console, allOrConfigKey string, opts *configGetOptions) error {
	// Handle "all" keyword - merge all configs in hierarchy
	if strings.EqualFold(allOrConfigKey, "all") {
		return listAllConfigFromHierarchy(console, opts.workingDirectory)
	}

	// For specific key, use first config file found
	configPath := determineConfigPath(opts.workingDirectory)

	// Load config
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get specific value
	value := cfg.GetConfigValue(allOrConfigKey)
	if value == "" {
		// Match dotnet nuget behavior: exit code 2 for key not found
		console.Printf("error: Key '%s' not found.\n", allOrConfigKey)
		os.Exit(2)
	}

	// Handle --show-path (match dotnet nuget behavior)
	if opts.showPath {
		// Dotnet format: <value><TAB>file: <config-path>
		console.Printf("%s\tfile: %s\n", value, configPath)
	} else {
		console.Println(value)
	}

	return nil
}

// Config Set Subcommand

func newConfigSetCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <config-key> <config-value>",
		Short: "Set a configuration value",
		Long: `Set a NuGet configuration value.

Examples:
  gonuget config set repositoryPath ~/packages
  gonuget config set globalPackagesFolder ~/.nuget/packages
  gonuget config set http_proxy http://proxy:8080`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(console, args[0], args[1])
		},
	}

	return cmd
}

func runConfigSet(console *output.Console, configKey string, configValue string) error {
	// Validate config key (match dotnet nuget behavior)
	if !isValidConfigKey(configKey) {
		console.Printf("error: '%s' is not a valid config key in config section.\n", configKey)
		os.Exit(1)
	}

	// Determine config file from hierarchy (matches dotnet nuget behavior)
	configPath := config.FindConfigFile()
	if configPath == "" {
		return fmt.Errorf("unable to find a NuGet.config file. Create one in the current or parent directory")
	}

	// Load or create config
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set value
	cfg.SetConfigValue(configKey, configValue)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Println(fmt.Sprintf("Successfully updated config file at '%s'.", configPath))
	return nil
}

// Config Unset Subcommand

func newConfigUnsetCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <config-key>",
		Short: "Remove a configuration value",
		Long: `Remove a NuGet configuration value.

Examples:
  gonuget config unset repositoryPath
  gonuget config unset http_proxy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigUnset(console, args[0])
		},
	}

	return cmd
}

func runConfigUnset(console *output.Console, configKey string) error {
	// Validate config key (match dotnet nuget behavior)
	if !isValidConfigKey(configKey) {
		console.Printf("error: '%s' is not a valid config key in config section.\n", configKey)
		os.Exit(1)
	}

	// Determine config file from hierarchy (matches dotnet nuget behavior)
	configPath := config.FindConfigFile()
	if configPath == "" {
		return fmt.Errorf("unable to find a NuGet.config file. Create one in the current or parent directory")
	}

	// Load config
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Remove value
	cfg.DeleteConfigValue(configKey)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Println(fmt.Sprintf("Successfully updated config file at '%s'.", configPath))
	return nil
}

// Config Paths Subcommand

type configPathsOptions struct {
	workingDirectory string
}

func newConfigPathsCommand(console *output.Console) *cobra.Command {
	opts := &configPathsOptions{}

	cmd := &cobra.Command{
		Use:   "paths",
		Short: "Display configuration file paths",
		Long: `Display the paths to NuGet configuration files in the hierarchy.

Examples:
  gonuget config paths
  gonuget config paths --working-directory /path/to/project`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPaths(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.workingDirectory, "working-directory", "", "Working directory for config hierarchy resolution")

	return cmd
}

func runConfigPaths(console *output.Console, opts *configPathsOptions) error {
	// Get config hierarchy
	paths := config.GetConfigHierarchy(opts.workingDirectory)

	console.Println("NuGet configuration file paths:")
	for _, path := range paths {
		exists := "✓"
		if _, err := filepath.Abs(path); err != nil {
			exists = "✗"
		}
		console.Printf("  %s %s\n", exists, path)
	}

	return nil
}

// Helper functions

// isValidConfigKey checks if a config key is valid according to NuGet spec
// See: https://learn.microsoft.com/en-us/nuget/reference/nuget-config-file#config-section
func isValidConfigKey(key string) bool {
	validKeys := map[string]bool{
		"defaultPushSource":            true,
		"dependencyVersion":            true,
		"globalPackagesFolder":         true,
		"http_proxy":                   true,
		"http_proxy.user":              true,
		"http_proxy.password":          true,
		"no_proxy":                     true,
		"maxHttpRequestsPerSource":     true,
		"repositoryPath":               true,
		"signatureValidationMode":      true,
		"updatePackageLastAccessTime":  true,
	}
	return validKeys[key]
}

func determineConfigPath(workingDirectory string) string {
	if workingDirectory != "" {
		// Start from working directory
		return config.FindConfigFileFrom(workingDirectory)
	}
	// Use current directory
	configPath := config.FindConfigFile()
	if configPath == "" {
		configPath = config.GetUserConfigPath()
	}
	return configPath
}

func loadOrCreateConfig(path string) (*config.NuGetConfig, error) {
	cfg, err := config.LoadNuGetConfig(path)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func listAllConfigFromHierarchy(console *output.Console, workingDirectory string) error {
	// Get all config files in hierarchy
	paths := config.GetConfigHierarchy(workingDirectory)

	// Merge configs
	merged := mergeConfigs(paths)

	return listAllConfig(console, merged)
}

func mergeConfigs(paths []string) *config.NuGetConfig {
	merged := &config.NuGetConfig{}
	packageSourceMap := make(map[string]config.PackageSource)
	configItemMap := make(map[string]config.ConfigItem)
	apiKeyMap := make(map[string]config.APIKey)

	// Load and merge all configs (later files override earlier)
	for _, path := range paths {
		cfg, err := config.LoadNuGetConfig(path)
		if err != nil {
			// Skip files that don't exist or can't be read
			continue
		}

		// Merge package sources
		if cfg.PackageSources != nil {
			for _, src := range cfg.PackageSources.Add {
				packageSourceMap[src.Key] = src
			}
		}

		// Merge config items
		if cfg.Config != nil {
			for _, item := range cfg.Config.Add {
				configItemMap[item.Key] = item
			}
		}

		// Merge API keys
		if cfg.APIKeys != nil {
			for _, key := range cfg.APIKeys.Add {
				apiKeyMap[key.Key] = key
			}
		}
	}

	// Convert maps back to slices
	if len(packageSourceMap) > 0 {
		merged.PackageSources = &config.PackageSources{}
		for _, src := range packageSourceMap {
			merged.PackageSources.Add = append(merged.PackageSources.Add, src)
		}
	}

	if len(configItemMap) > 0 {
		merged.Config = &config.ConfigSection{}
		for _, item := range configItemMap {
			merged.Config.Add = append(merged.Config.Add, item)
		}
	}

	if len(apiKeyMap) > 0 {
		merged.APIKeys = &config.APIKeys{}
		for _, key := range apiKeyMap {
			merged.APIKeys.Add = append(merged.APIKeys.Add, key)
		}
	}

	return merged
}

func listAllConfig(console *output.Console, cfg *config.NuGetConfig) error {
	hasContent := false

	// Package Sources
	if cfg.PackageSources != nil && len(cfg.PackageSources.Add) > 0 {
		console.Println("packageSources:")
		for _, src := range cfg.PackageSources.Add {
			output := fmt.Sprintf("\tadd key=\"%s\" value=\"%s\"", src.Key, src.Value)
			if src.ProtocolVersion != "" {
				output += fmt.Sprintf(" protocolVersion=\"%s\"", src.ProtocolVersion)
			}
			if src.Enabled != "" {
				output += fmt.Sprintf(" enabled=\"%s\"", src.Enabled)
			}
			console.Println(output)
		}
		console.Println("")
		hasContent = true
	}

	// Config Section
	if cfg.Config != nil && len(cfg.Config.Add) > 0 {
		console.Println("config:")
		for _, item := range cfg.Config.Add {
			console.Printf("\tadd key=\"%s\" value=\"%s\"\n", item.Key, item.Value)
		}
		console.Println("")
		hasContent = true
	}

	// API Keys
	if cfg.APIKeys != nil && len(cfg.APIKeys.Add) > 0 {
		console.Println("apikeys:")
		for _, key := range cfg.APIKeys.Add {
			console.Printf("\tadd key=\"%s\" value=\"%s\"\n", key.Key, key.Value)
		}
		console.Println("")
		hasContent = true
	}

	// Trusted Signers
	if cfg.TrustedSigners != nil && len(cfg.TrustedSigners.Add) > 0 {
		console.Println("trustedSigners:")
		for _, signer := range cfg.TrustedSigners.Add {
			console.Printf("\tadd name=\"%s\"\n", signer.Name)
		}
		console.Println("")
		hasContent = true
	}

	// Package Source Credentials
	if cfg.PackageSourceCredentials != nil && len(cfg.PackageSourceCredentials.Items) > 0 {
		console.Println("packageSourceCredentials:")
		for _, cred := range cfg.PackageSourceCredentials.Items {
			console.Printf("\t%s:\n", cred.XMLName.Local)
			for _, item := range cred.Add {
				console.Printf("\t\tadd key=\"%s\" value=\"%s\"\n", item.Key, item.Value)
			}
		}
		console.Println("")
		hasContent = true
	}

	if !hasContent {
		console.Println("No configuration values found.")
	}

	return nil
}
