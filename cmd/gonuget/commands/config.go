// cmd/gonuget/commands/config.go
package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

type configOptions struct {
	configFile string
	asPath     bool
	set        []string
}

// NewConfigCommand creates the config command
func NewConfigCommand(console *output.Console) *cobra.Command {
	opts := &configOptions{}

	cmd := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Get or set NuGet configuration values",
		Long: `Get or set NuGet configuration values.

Examples:
  gonuget config                                    # List all config values
  gonuget config repositoryPath                     # Get specific value
  gonuget config repositoryPath ~/packages          # Set value
  gonuget config -Set key1=value1 -Set key2=value2  # Set multiple values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(console, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "ConfigFile", "", "Config file to use")
	cmd.Flags().BoolVar(&opts.asPath, "AsPath", false, "Return value as filesystem path")
	cmd.Flags().StringArrayVar(&opts.set, "Set", []string{}, "Set key=value pair(s)")

	return cmd
}

func runConfig(console *output.Console, args []string, opts *configOptions) error {
	// Determine config file
	configPath := opts.configFile
	if configPath == "" {
		configPath = config.FindConfigFile()
		if configPath == "" {
			// Use user config
			configPath = config.GetUserConfigPath()
		}
	}

	// Load config
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle --Set flags
	if len(opts.set) > 0 {
		for _, pair := range opts.set {
			key, value, err := parseKeyValue(pair)
			if err != nil {
				return err
			}
			cfg.SetConfigValue(key, value)
		}

		if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		console.Success("Configuration updated")
		return nil
	}

	// Handle get/set via args
	switch len(args) {
	case 0:
		// List all config values
		return listAllConfig(console, cfg)
	case 1:
		// Get value
		return getConfigValue(console, cfg, args[0], opts.asPath)
	case 2:
		// Set value
		cfg.SetConfigValue(args[0], args[1])
		if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		console.Success("Set config value: %s = %s", args[0], args[1])
		return nil
	default:
		return fmt.Errorf("too many arguments")
	}
}

func loadOrCreateConfig(path string) (*config.NuGetConfig, error) {
	cfg, err := config.LoadNuGetConfig(path)
	if err != nil {
		// Create new default config
		cfg = config.NewDefaultConfig()
	}
	return cfg, nil
}

func listAllConfig(console *output.Console, cfg *config.NuGetConfig) error {
	if cfg.Config == nil || len(cfg.Config.Add) == 0 {
		console.Info("No configuration values set")
		return nil
	}

	console.Println("Configuration values:")
	for _, item := range cfg.Config.Add {
		console.Printf("  %s = %s\n", item.Key, item.Value)
	}

	return nil
}

func getConfigValue(console *output.Console, cfg *config.NuGetConfig, key string, asPath bool) error {
	value := cfg.GetConfigValue(key)
	if value == "" {
		return fmt.Errorf("configuration value not found: %s", key)
	}

	if asPath {
		// Expand path
		if filepath.IsAbs(value) {
			console.Println(value)
		} else {
			absPath, err := filepath.Abs(value)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}
			console.Println(absPath)
		}
	} else {
		console.Println(value)
	}

	return nil
}

func parseKeyValue(pair string) (string, string, error) {
	for i, r := range pair {
		if r == '=' {
			return pair[:i], pair[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid key=value pair: %s", pair)
}
