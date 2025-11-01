package commands

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewSourceAddCommand creates the "source add" subcommand
func NewSourceAddCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{}

	cmd := &cobra.Command{
		Use:   "add <PACKAGE_SOURCE_PATH>",
		Short: "Add a NuGet source",
		Long: `Add a new package source to NuGet.config.

This command matches: dotnet nuget add source <URL>

Examples:
  gonuget source add https://api.nuget.org/v3/index.json --name "MyFeed"
  gonuget source add https://pkgs.dev.azure.com/org/_packaging/feed/nuget/v3/index.json --name "Azure" --username user --password pass
  gonuget source add https://private.feed.com/v3/index.json --name "Private" --store-password-in-clear-text`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.source = args[0]
			return runAddSource(console, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Name of the source.")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "Username to be used when connecting to an authenticated source.")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "Password to be used when connecting to an authenticated source.")
	cmd.Flags().BoolVar(&opts.storePasswordInClearText, "store-password-in-clear-text", false, "Enables storing portable package source credentials by disabling password encryption.")
	cmd.Flags().StringVar(&opts.validAuthenticationTypes, "valid-authentication-types", "", "Comma-separated list of valid authentication types for this source. Set this to basic if the server advertises NTLM or Negotiate and your credentials must be sent using the Basic mechanism, for instance when using a PAT with on-premises Azure DevOps Server. Other valid values include negotiate, kerberos, ntlm, and digest, but these values are unlikely to be useful.")
	cmd.Flags().StringVar(&opts.protocolVersion, "protocol-version", "", "The NuGet server protocol version to be used. Currently supported versions are 2 and 3. See https://learn.microsoft.com/nuget/api/overview for information about the version 3 protocol. Defaults to 2 if not specified.")
	cmd.Flags().BoolVar(&opts.allowInsecureConnections, "allow-insecure-connections", false, "Allows HTTP connections for adding or updating packages. Note: This method is not secure. For secure options, see https://aka.ms/nuget-https-everywhere for more information.")
	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runAddSource(console *output.Console, opts *sourceOptions) error {
	// Validate source URL
	parsedURL, err := url.Parse(opts.source)
	if err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}

	// Check for insecure HTTP connections
	if parsedURL.Scheme == "http" && !opts.allowInsecureConnections {
		return fmt.Errorf("HTTP source '%s' is insecure. Use --allow-insecure-connections to proceed anyway. For secure options, see https://aka.ms/nuget-https-everywhere for more information", opts.source)
	}

	cfg, configPath, err := loadSourceConfig(opts.configFile)
	if err != nil {
		return err
	}

	// Check if source already exists
	if cfg.PackageSources != nil {
		for _, source := range cfg.PackageSources.Add {
			if strings.EqualFold(source.Key, opts.name) {
				return fmt.Errorf("package source with name '%s' already exists", opts.name)
			}
		}
	}

	// Add the source
	newSource := config.PackageSource{
		Key:     opts.name,
		Value:   opts.source,
		Enabled: "true",
	}

	// Only set protocol version if it's not the default (2)
	// This matches dotnet nuget behavior which doesn't write protocolVersion="2"
	if opts.protocolVersion != "" && opts.protocolVersion != "2" {
		newSource.ProtocolVersion = opts.protocolVersion
	}

	cfg.AddPackageSource(newSource)

	// Handle credentials if provided
	if opts.username != "" || opts.password != "" {
		if opts.storePasswordInClearText {
			console.Warning("WARNING: Storing password in clear text is not secure!")
		}
		addOrUpdateCredential(cfg, opts.name, opts.username, opts.password, opts.storePasswordInClearText, opts.validAuthenticationTypes)
	}

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Info("Package source with name '%s' added successfully.", opts.name)
	return nil
}
