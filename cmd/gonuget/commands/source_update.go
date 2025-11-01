package commands

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewSourceUpdateCommand creates the "source update" command matching dotnet nuget
func NewSourceUpdateCommand(console *output.Console) *cobra.Command {
	opts := &sourceOptions{}

	cmd := &cobra.Command{
		Use:   "update <NAME>",
		Short: "Update a NuGet source",
		Long: `Update properties of an existing package source.

This command matches: dotnet nuget update source

Examples:
  gonuget source update MyFeed --source https://new.url/v3/index.json
  gonuget source update Azure --username newuser --password newpass
  gonuget source update Private --store-password-in-clear-text`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runUpdateSource(console, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.source, "source", "s", "", "Path to the package source.")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "Username to be used when connecting to an authenticated source.")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "Password to be used when connecting to an authenticated source.")
	cmd.Flags().BoolVar(&opts.storePasswordInClearText, "store-password-in-clear-text", false, "Enables storing portable package source credentials by disabling password encryption.")
	cmd.Flags().StringVar(&opts.validAuthenticationTypes, "valid-authentication-types", "", "Comma-separated list of valid authentication types for this source. Set this to basic if the server advertises NTLM or Negotiate and your credentials must be sent using the Basic mechanism, for instance when using a PAT with on-premises Azure DevOps Server. Other valid values include negotiate, kerberos, ntlm, and digest, but these values are unlikely to be useful.")
	cmd.Flags().StringVar(&opts.protocolVersion, "protocol-version", "", "The NuGet server protocol version to be used. Currently supported versions are 2 and 3. See https://learn.microsoft.com/nuget/api/overview for information about the version 3 protocol. Defaults to 2 if not specified.")
	cmd.Flags().BoolVar(&opts.allowInsecureConnections, "allow-insecure-connections", false, "Allows HTTP connections for adding or updating packages. Note: This method is not secure. For secure options, see https://aka.ms/nuget-https-everywhere for more information.")
	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "The NuGet configuration file. If specified, only the settings from this file will be used. If not specified, the hierarchy of configuration files from the current directory will be used.")

	return cmd
}

func runUpdateSource(console *output.Console, opts *sourceOptions) error {
	cfg, configPath, err := loadSourceConfig(opts.configFile)
	if err != nil {
		return err
	}

	// Check if source exists
	source, err := findSourceByName(cfg, opts.name)
	if err != nil {
		return err
	}

	// Update source URL if provided
	if opts.source != "" {
		// Validate source URL
		parsedURL, err := url.Parse(opts.source)
		if err != nil {
			return fmt.Errorf("invalid source URL: %w", err)
		}

		// Check for insecure HTTP connections
		if parsedURL.Scheme == "http" && !opts.allowInsecureConnections {
			return fmt.Errorf("HTTP source '%s' is insecure. Use --allow-insecure-connections to proceed anyway. For secure options, see https://aka.ms/nuget-https-everywhere for more information", opts.source)
		}

		source.Value = opts.source
		// Only set protocol version if it's not the default (2)
		// This matches dotnet nuget behavior which doesn't write protocolVersion="2"
		if opts.protocolVersion != "" && opts.protocolVersion != "2" {
			source.ProtocolVersion = opts.protocolVersion
		} else if opts.protocolVersion == "2" {
			// Clear protocol version if explicitly set to 2 (default)
			source.ProtocolVersion = ""
		}
		cfg.AddPackageSource(*source)
	}

	// Update credentials if provided
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

	console.Info("Package source with name '%s' updated successfully.", opts.name)
	return nil
}
