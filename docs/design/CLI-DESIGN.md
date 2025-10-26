# gonuget CLI Tool Design Specification

**Status**: Design Phase
**Target**: 100% parity with nuget.exe CLI and dotnet nuget commands
**Quality**: Production-ready, enterprise-grade
**Created**: 2025-10-25
**Last Updated**: 2025-10-25

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Command Specification](#command-specification)
4. [Configuration Management](#configuration-management)
5. [Output & Formatting](#output--formatting)
6. [Authentication & Security](#authentication--security)
7. [Error Handling](#error-handling)
8. [Performance & Optimization](#performance--optimization)
9. [Cross-Platform Support](#cross-platform-support)
10. [Testing Strategy](#testing-strategy)
11. [Implementation Phases](#implementation-phases)
12. [Quality Requirements](#quality-requirements)

---

## Executive Summary

The gonuget CLI tool provides a production-ready command-line interface for NuGet package management operations with 100% functional parity to the official nuget.exe tool and dotnet nuget commands. Built on the gonuget library, it delivers native performance, cross-platform support, and seamless integration with existing NuGet workflows.

### Design Goals

1. **Complete Parity**: All nuget.exe commands and dotnet nuget commands with identical behavior
2. **Native Performance**: Sub-100ms startup time, efficient resource usage
3. **Superior UX**: Modern CLI patterns, colored output, progress indicators
4. **Production Quality**: Error handling, logging, telemetry, graceful degradation
5. **Cross-Platform**: Windows, Linux, macOS with platform-specific optimizations
6. **Extensibility**: NuGet Credential Provider protocol compatibility
7. **Backwards Compatibility**: Drop-in replacement for nuget.exe and dotnet nuget in CI/CD pipelines

### Key Differentiators

- **Single Binary**: No runtime dependencies (vs .NET Framework/SDK requirement)
- **Faster Startup**: Native binary vs CLR warm-up time
- **Lower Memory**: Efficient Go runtime vs .NET GC overhead
- **Modern Output**: Built-in progress bars, structured logging, JSON output
- **HTTP/3 Support**: Next-generation protocol for faster downloads
- **Unified CLI**: Single tool combining nuget.exe and dotnet nuget functionality

---

## Architecture Overview

### Directory Structure

```
cmd/gonuget/
├── main.go                    # Entry point, signal handling, global setup
├── cli/
│   ├── app.go                 # CLI application setup (cobra/urfave)
│   ├── context.go             # Global execution context
│   ├── flags.go               # Common flag definitions
│   └── version.go             # Version information
├── commands/
│   ├── add.go                 # Add command implementation
│   ├── clientcerts.go         # Client certificates management
│   ├── config.go              # Configuration management (get/set/list)
│   ├── delete.go              # Package deletion
│   ├── help.go                # Help command
│   ├── init.go                # Initialize offline feed
│   ├── install.go             # Package installation
│   ├── list.go                # List packages (deprecated, use search)
│   ├── locals.go              # Local cache management
│   ├── pack.go                # Create .nupkg packages
│   ├── push.go                # Push packages to feed
│   ├── restore.go             # Restore packages
│   ├── search.go              # Search package feeds
│   ├── setapikey.go           # API key management
│   ├── sign.go                # Package signing
│   ├── source_list.go         # List package sources
│   ├── source_add.go          # Add package source
│   ├── source_remove.go       # Remove package source
│   ├── source_update.go       # Update package source
│   ├── source_enable.go       # Enable package source
│   ├── source_disable.go      # Disable package source
│   ├── spec.go                # Generate .nuspec files
│   ├── trustedsigners.go      # Trusted signer management
│   ├── update.go              # Update packages
│   ├── verify.go              # Verify package signatures
│   └── base.go                # Base command interface
├── config/
│   ├── settings.go            # NuGet.config parsing/writing
│   ├── sources.go             # Package source configuration
│   ├── credentials.go         # Credential storage
│   └── defaults.go            # Default configuration values
├── output/
│   ├── console.go             # Console abstraction
│   ├── formatter.go           # Output formatting (table, json, etc.)
│   ├── progress.go            # Progress bars and spinners
│   ├── colors.go              # Color schemes
│   └── logger.go              # Structured logging
├── auth/
│   ├── provider.go            # Authentication provider interface
│   ├── apikey.go              # API key authentication
│   ├── basic.go               # Basic auth
│   ├── bearer.go              # Bearer token auth
│   ├── interactive.go         # Interactive credential prompts
│   └── credprovider/
│       ├── discovery.go       # Credential provider discovery
│       ├── executor.go        # External provider execution
│       ├── protocol.go        # Provider protocol (JSON request/response)
│       └── cache.go           # Credential caching
├── telemetry/
│   ├── collector.go           # Usage telemetry collection
│   ├── reporter.go            # Anonymous usage reporting
│   └── consent.go             # Telemetry opt-in/opt-out
└── internal/
    ├── signals.go             # Signal handling (Ctrl+C, etc.)
    ├── updates.go             # Self-update mechanism
    └── validators.go          # Input validation helpers
```

### Technology Stack

**CLI Framework**: [cobra](https://github.com/spf13/cobra)
- Industry standard for Go CLIs (kubectl, docker, hugo)
- Excellent subcommand support
- Built-in help generation
- POSIX-compliant flag parsing

**Configuration**: [viper](https://github.com/spf13/viper)
- Reads NuGet.config (XML) and gonuget.yaml
- Environment variable support
- Configuration merging and precedence

**Output Formatting**:
- [tablewriter](https://github.com/olekukonko/tablewriter) for tables
- [color](https://github.com/fatih/color) for colored output
- [progressbar](https://github.com/schollz/progressbar) for progress indicators

**Authentication**:
- OS keychain integration via [go-keyring](https://github.com/zalando/go-keyring)
- Encrypted credential storage
- NuGet Credential Provider protocol support (external executables)

---

## Command Specification

### Command: add package

**Synopsis**: Adds a NuGet package reference to a project file

```bash
gonuget add package <PACKAGE_ID> [options]
gonuget add <PROJECT> package <PACKAGE_ID> [options]
```

**Arguments**:
- `PACKAGE_ID` (string, required): Package ID to add
- `PROJECT` (string, optional): Path to project file (.csproj)

**Flags**:
- `--version` (string): Package version to add (default: latest stable)
- `--framework` (string, repeatable): Target framework(s) for package reference
- `--no-restore` (bool): Don't restore packages after adding
- `--source` (string, repeatable): Package source(s) to use
- `--package-directory` (string): Custom package directory
- `--prerelease` (bool): Include prerelease versions when resolving latest
- `--interactive` (bool): Allow interactive credential prompts

**Behavior**:
1. Locate project file (specified or auto-discover in current directory)
2. Load and parse project file (.csproj)
3. **Check for Central Package Management (CPM)**:
   - If CPM enabled (`ManagePackageVersionsCentrally=true`):
     - Load `Directory.Packages.props`
     - Add/update `<PackageVersion>` in Directory.Packages.props
     - Add `<PackageReference>` WITHOUT version to .csproj
   - If CPM not enabled:
     - Add/update `<PackageReference>` WITH version to .csproj
4. Resolve version (if not specified): query sources for latest
5. Save modified files with UTF-8 BOM
6. Restore packages (unless `--no-restore`)

**Central Package Management (CPM) Support**:
gonuget fully supports CPM as implemented in `dotnet add package`:
- Detects `ManagePackageVersionsCentrally` property in project
- Loads and modifies `Directory.Packages.props`
- Adds `<PackageVersion>` entries with version
- Adds `<PackageReference>` entries WITHOUT version to project
- Supports `VersionOverride` attribute
- Compatible with multi-project CPM scenarios

**Exit Codes**:
- 0: Success
- 1: Package not found
- 2: Project file not found or invalid
- 3: Version resolution failed
- 4: File write failed

**Example**:
```bash
# Add latest stable version
gonuget add package Newtonsoft.Json

# Add specific version
gonuget add package Newtonsoft.Json --version 13.0.3

# Add to specific project
gonuget add MyApp.csproj package Newtonsoft.Json

# Add framework-specific reference
gonuget add package System.Text.Json --framework net8.0

# Add without restoring
gonuget add package Serilog --no-restore

# Works with CPM projects automatically
# If Directory.Packages.props exists with ManagePackageVersionsCentrally=true:
#   - Updates Directory.Packages.props with version
#   - Updates .csproj with PackageReference WITHOUT version
```

**dotnet nuget Parity**:
This command matches `dotnet add package` behavior exactly, including full CPM support.

**Reference Implementation**:
dotnet/sdk: `src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs`

---

### Command: config

**Synopsis**: Gets, sets, unsets, or displays paths for NuGet configuration values

**Subcommands**:
- `get`: Get a configuration value (or all values with "all")
- `set`: Set a configuration value
- `unset`: Remove a configuration value
- `paths`: Display config file paths

```bash
gonuget config get <all-or-config-key> [options]
gonuget config set <config-key> <config-value> [options]
gonuget config unset <config-key> [options]
gonuget config paths [options]
```

**Flags (get)**:
- `--show-path` (bool): Return value as filesystem path (resolves relative paths)
- `--working-directory` (string): Working directory for config hierarchy resolution

**Flags (set)**:
- `--configfile` (string): Specific config file to modify

**Flags (unset)**:
- `--configfile` (string): Specific config file to modify

**Flags (paths)**:
- `--working-directory` (string): Working directory for config hierarchy resolution

**Configuration Keys**:
- `repositoryPath`: Global packages folder
- `globalPackagesFolder`: NuGet v3 cache location
- `http_proxy`: HTTP proxy URL
- `http_proxy.user`: Proxy username
- `http_proxy.password`: Proxy password (encrypted)
- `signatureValidationMode`: Package signature validation (accept, require)
- `defaultPushSource`: Default push destination

**Example**:
```bash
# Get configuration value
gonuget config get repositoryPath

# Get all configuration values
gonuget config get all

# Set configuration value
gonuget config set repositoryPath ~/packages

# Unset configuration value
gonuget config unset repositoryPath

# Show configuration file paths
gonuget config paths

# Get as path (expands relative paths to absolute)
gonuget config get repositoryPath --show-path

# Set in specific config file
gonuget config set repositoryPath ~/packages --configfile ./NuGet.config
```

**dotnet nuget Parity**:
This command matches `dotnet nuget config` behavior exactly, including:
- Positional arguments for get/set/unset (not flags)
- `--show-path` flag (not `--as-path`)
- `--working-directory` for config hierarchy resolution
- `paths` subcommand for displaying config file locations
- `all` keyword in get to display all config values

---

### Command: delete

**Synopsis**: Deletes a package from a package source

```bash
gonuget delete <package-id> <version> [api-key] [options]
```

**Arguments**:
- `package-id` (string, required): Package ID
- `version` (string, required): Package version
- `api-key` (string, optional): API key (can also use -ApiKey flag)

**Flags**:
- `-Source, -s` (string): Package source URL (default: nuget.org)
- `-ApiKey, -k` (string): API key for authentication
- `-NonInteractive` (bool): Don't prompt for confirmation
- `-NoServiceEndpoint` (bool): Append package ID to source URL directly

**Behavior**:
1. Resolve package source from config or use provided URL
2. Authenticate using API key from argument, flag, or stored config
3. Send DELETE request to feed (V3: PackagePublish resource, V2: /api/v2/package/{id}/{version})
4. Confirm deletion unless `-NonInteractive`
5. Report success/failure

**Exit Codes**:
- 0: Success
- 1: Package not found
- 2: Authentication failed
- 3: Package locked or deletion not allowed
- 4: Network error

**Example**:
```bash
# Delete from nuget.org
gonuget delete MyPackage 1.0.0 -ApiKey oy2abc...

# Delete from private feed
gonuget delete MyPackage 1.0.0 -Source https://private.feed.com/v3/index.json \
  -ApiKey abc123

# Non-interactive (for CI/CD)
gonuget delete MyPackage 1.0.0 -NonInteractive
```

---

### Command: help

**Synopsis**: Displays help information for commands

```bash
gonuget help [command]
gonuget [command] --help
```

**Flags**:
- `-All` (bool): Show help for all commands
- `-Markdown` (bool): Output help in Markdown format

**Behavior**:
1. If no command: Display overview with command list
2. If command specified: Display detailed help for that command
3. If `-All`: Display all command help sequentially
4. If `-Markdown`: Generate Markdown documentation

**Output Format**:
```
USAGE:
  gonuget <command> [options]

FOUNDATION COMMANDS:
  help             Show help information
  version          Show version information
  config           Configuration management (get, set, list)

SOURCE MANAGEMENT:
  list source      List configured package sources
  add source       Add a new package source
  remove source    Remove a package source
  update source    Update an existing source
  enable source    Enable a package source
  disable source   Disable a package source

PACKAGE OPERATIONS:
  add package      Add package reference to project (with CPM support)
  search           Search for packages
  restore          Restore packages for a project
  pack             Create a .nupkg package
  push             Push a package to a feed
  delete           Delete a package from a feed

SIGNING & SECURITY:
  sign             Sign a package
  verify           Verify package signatures
  trusted-signers  Manage trusted signers

CACHE MANAGEMENT:
  locals           Manage local caches

Run 'gonuget help <command>' for detailed help.
```

---

### Command: locals

**Synopsis**: Manages local NuGet caches

```bash
gonuget locals [resource-name] [options]
```

**Resource Names**:
- `http-cache`: HTTP cache directory
- `global-packages`: Global packages folder (NuGet v3 cache)
- `temp`: Temporary directory
- `plugins-cache`: NuGet protocol plugin cache (for package download/auth plugins)
- `all`: All of the above

**Flags**:
- `-Clear, -c` (bool): Clear the specified cache(s)
- `-List, -l` (bool): List cache locations

**Behavior**:
1. If `-List`: Display cache paths and sizes
2. If `-Clear`: Prompt for confirmation (unless `-NonInteractive`), then delete cache contents
3. If no flags: Display help

**Output**:
```
gonuget locals all -List

http-cache: /home/user/.nuget/v3-cache (128 MB)
global-packages: /home/user/.nuget/packages (1.4 GB)
temp: /tmp/gonuget (0 B)
plugins-cache: /home/user/.nuget/plugins (0 B)
```

---

### Command: pack

**Synopsis**: Creates a NuGet package from a .nuspec or project file

```bash
gonuget pack [<nuspec-or-project>] [options]
```

**Arguments**:
- `nuspec-or-project` (string, optional): Path to .nuspec, .csproj, .vbproj, .fsproj, or .go (default: first found in current dir)

**Flags**:
- `-OutputDirectory, -o` (string): Output directory (default: current dir)
- `-BasePath, -b` (string): Base path for files referenced in .nuspec
- `-Version, -v` (string): Override package version
- `-Suffix` (string): Version suffix (e.g., "beta", appended as "-beta")
- `-Properties, -p` (string, repeatable): Property overrides (e.g., "Configuration=Release")
- `-Symbols` (bool): Create symbols package (.symbols.nupkg or .snupkg)
- `-SymbolPackageFormat` (string): Symbol package format (symbols.nupkg, snupkg)
- `-Tool` (bool): Mark as tool package
- `-Build` (bool): Build project before packing (for project files)
- `-NoDefaultExcludes` (bool): Don't exclude default patterns (.git, .hg, .svn, etc.)
- `-NoPackageAnalysis` (bool): Skip package validation rules
- `-ExcludeEmptyDirectories` (bool): Exclude empty directories
- `-IncludeReferencedProjects` (bool): Include referenced projects as dependencies or content
- `-MinClientVersion` (string): Set minClientVersion attribute
- `-Exclude` (string, repeatable): File patterns to exclude
- `-InstallPackageToOutputPath` (bool): Copy dependencies to output
- `-OutputFileNamesWithoutVersion` (bool): Output filename without version
- `-PackagesDirectory` (string): Packages directory for build
- `-SolutionDirectory` (string): Solution directory
- `-MSBuildPath` (string): Path to MSBuild (for building .NET projects)
- `-ContentTargetFolders` (string): Semicolon-delimited folders (for project packaging)

**Project File Support**:

**MSBuild Integration** (REQUIRED for 100% parity):
- Parse .csproj, .vbproj, .fsproj files
- Extract metadata from MSBuild properties
- Support MSBuildPath and MSBuildVersion flags
- Build project before packing (if `-Build` specified)
- IncludeReferencedProjects support
- Property substitution using MSBuild evaluation
- PackagesDirectory and SolutionDirectory resolution
- Integration with NuGet.Build.Tasks for dependency discovery

For Go projects, recognize special `gonuget.yaml` manifest:
```yaml
metadata:
  id: MyGoPackage
  version: 1.0.0
  authors: ["Author Name"]
  description: Package description
  projectUrl: https://github.com/user/repo
  license: MIT
  tags: [cli, tool]
files:
  - src: bin/mytool
    target: tools/
  - src: README.md
    target: docs/
```

**Behavior**:
1. Locate and parse .nuspec or project file
2. Resolve version (from file, flag, or git tags)
3. Apply property substitutions
4. Collect files based on file patterns
5. Validate package contents
6. Create .nupkg (ZIP + OPC conventions)
7. Create .symbols.nupkg/.snupkg if requested
8. Output package details

**Output**:
```
Packing MyPackage 1.0.0...
  Reading MyPackage.nuspec... ✓
  Collecting files... ✓
    lib/net8.0/MyPackage.dll
    README.md
  Creating MyPackage.1.0.0.nupkg... ✓
Successfully created package './MyPackage.1.0.0.nupkg' (245 KB)
```

---

### Command: push

**Synopsis**: Pushes a package to a package source

```bash
gonuget push <package> [api-key] [options]
```

**Arguments**:
- `package` (string, required): Path to .nupkg file
- `api-key` (string, optional): API key for authentication

**Flags**:
- `-Source, -s` (string): Target package source URL
- `-ApiKey, -k` (string): API key for authentication
- `-SymbolSource` (string): Symbol server URL
- `-SymbolApiKey` (string): Symbol server API key
- `-Timeout, -t` (int): Push timeout in seconds (default: 300)
- `-DisableBuffering` (bool): Disable response buffering for large packages
- `-NoSymbols` (bool): Don't push symbols package even if present
- `-NoServiceEndpoint` (bool): Append package ID to source URL
- `-SkipDuplicate` (bool): Succeed if package already exists
- `-AllowInsecureConnections` (bool): Allow HTTP (non-HTTPS) feeds

**Behavior**:
1. Validate package file exists and is valid
2. Resolve target source (flag, config, or default)
3. Authenticate using API key
4. Upload .nupkg with progress indicator
5. If .symbols.nupkg/.snupkg present and `-NoSymbols` not set, push symbols
6. Poll for processing status (if feed supports it)
7. Report success or detailed error

**Progress Output**:
```
Pushing MyPackage.1.0.0.nupkg to https://api.nuget.org/v3/index.json...
Uploading [████████████████████████] 100% (2.4 MB/s)
Package published successfully.
Your package is available at: https://www.nuget.org/packages/MyPackage/1.0.0
```

**Exit Codes**:
- 0: Success
- 1: Package file invalid or not found
- 2: Authentication failed (401/403)
- 3: Package already exists (409) and `-SkipDuplicate` not set
- 4: Network error or timeout
- 5: Server error (500+)

---

### Command: restore

**Synopsis**: Restores packages referenced in a project or solution

```bash
gonuget restore [<project-or-solution>] [options]
```

**Arguments**:
- `project-or-solution` (string, optional): Path to .sln, .csproj, packages.config, or project.json

**Flags**:
- `-Source, -s` (string, repeatable): Package source(s)
- `-PackagesDirectory, -o` (string): Packages folder location
- `-SolutionDirectory` (string): Solution root directory
- `-MSBuildPath` (string): Path to MSBuild
- `-Recursive, -r` (bool): Restore all projects in subdirectories
- `-Force` (bool): Force restore even if packages already exist
- `-DisableParallelProcessing` (bool): Restore packages sequentially
- `-RequireConsent` (bool): Require package restore consent
- `-NoCache` (bool): Don't use local cache
- `-DirectDownload` (bool): Download directly without cache
- `-UseLockFile` (bool): Use and generate lock file (packages.lock.json)
- `-LockedMode` (bool): Lock file must exist and be up-to-date
- `-LockFilePath` (string): Lock file location
- `-ForceEvaluate` (bool): Force re-evaluation of all projects
- `-Project2ProjectTimeOut` (int): Project-to-project timeout (ms)

**Behavior**:
1. Discover projects (recursively if `-Recursive`)
2. Parse dependencies from each project/packages.config
3. Build dependency graph across all projects
4. Detect version conflicts and resolve
5. Download packages in parallel (unless `-DisableParallelProcessing`)
6. Extract to packages directory
7. Generate/update lock file if `-UseLockFile`
8. Report summary (installed, updated, conflicts)

**Lock File** (packages.lock.json):
```json
{
  "version": 1,
  "dependencies": {
    "net8.0": {
      "Newtonsoft.Json": {
        "type": "Direct",
        "requested": "[13.0.1, )",
        "resolved": "13.0.3",
        "contentHash": "sha512-..."
      }
    }
  }
}
```

**Progress Output**:
```
Restoring packages for MySolution.sln...
  Analyzing projects (3 found)... ✓
  Resolving dependencies... ✓
  Downloading packages (4 packages, 12.4 MB)...
    Newtonsoft.Json 13.0.3 [████████████] 100%
    Serilog 3.1.1 [████████████] 100%
    ...
Successfully restored 4 packages to ./packages
```

---

### Command: search

**Synopsis**: Searches package sources for packages

```bash
gonuget search <query> [options]
```

**Arguments**:
- `query` (string, optional): Search query (if empty, lists popular packages)

**Flags**:
- `-Source, -s` (string, repeatable): Package source(s)
- `-Take, -t` (int): Number of results to return (default: 20, max: 1000)
- `-Skip` (int): Number of results to skip (pagination)
- `-Prerelease` (bool): Include prerelease packages
- `-Format` (string): Output format (table, json, simple) (default: table)

**Behavior**:
1. Query each specified source (in parallel)
2. Aggregate and deduplicate results
3. Sort by relevance (download count, exact match, etc.)
4. Format output according to `-Format`

**Output (table format)**:
```
┌─────────────────────┬─────────┬──────────────────────────────┐
│ Package             │ Version │ Description                   │
├─────────────────────┼─────────┼──────────────────────────────┤
│ Newtonsoft.Json     │ 13.0.3  │ JSON framework for .NET       │
│ Serilog             │ 3.1.1   │ Diagnostic logging library    │
│ AutoMapper          │ 12.0.1  │ Object-object mapper          │
└─────────────────────┴─────────┴──────────────────────────────┘
```

**Output (JSON format)**:
```json
{
  "totalHits": 234,
  "data": [
    {
      "id": "Newtonsoft.Json",
      "version": "13.0.3",
      "description": "JSON framework for .NET",
      "authors": ["James Newton-King"],
      "totalDownloads": 2841234567,
      "verified": true,
      "tags": ["json", "serialization"]
    }
  ]
}
```

---

### Command: sign

**Synopsis**: Signs a NuGet package with a code signing certificate

```bash
gonuget sign <package> [options]
```

**Arguments**:
- `package` (string, required): Path to .nupkg file to sign

**Flags**:
- `-CertificatePath` (string): Path to certificate file (.pfx, .p12)
- `-CertificatePassword` (string): Certificate password
- `-CertificateStoreLocation` (string): Store location (CurrentUser, LocalMachine)
- `-CertificateStoreName` (string): Store name (My, Root, etc.)
- `-CertificateSubjectName` (string): Certificate subject name
- `-CertificateFingerprint` (string): Certificate SHA256 fingerprint
- `-HashAlgorithm` (string): Signature hash algorithm (SHA256, SHA384, SHA512) (default: SHA256)
- `-Timestamper` (string): RFC 3161 timestamp server URL (default: http://timestamp.digicert.com)
- `-TimestampHashAlgorithm` (string): Timestamp hash algorithm (default: SHA256)
- `-OutputDirectory` (string): Output directory (default: overwrite in place)
- `-Overwrite` (bool): Overwrite existing signature

**Behavior**:
1. Load certificate from file or system store
2. Validate certificate is valid for code signing
3. Read package, create PKCS#7 signature
4. Timestamp signature via RFC 3161 server
5. Embed signature in package (in `.signature.p7s` in package root)
6. Write signed package

**Certificate Validation**:
- Must have Code Signing or Authenticode extended key usage
- Must be within validity period
- Certificate chain must be trusted

**Output**:
```
Signing MyPackage.1.0.0.nupkg...
  Loading certificate... ✓
  Creating signature (SHA256)... ✓
  Timestamping via http://timestamp.digicert.com... ✓
  Writing signed package... ✓
Successfully signed MyPackage.1.0.0.nupkg
```

---

### Command: list source

**Synopsis**: Lists all configured package sources

```bash
gonuget list source [options]
```

**Flags**:
- `--configfile` (string): Specific config file to read
- `--format` (string): Output format (detailed, simple, json) (default: detailed)

**Example**:
```bash
# List all sources
gonuget list source

# List sources from specific config file
gonuget list source --configfile ~/custom-nuget.config

# List sources in JSON format
gonuget list source --format json
```

**Output (detailed format)**:
```
Registered Sources:
  1. nuget.org [Enabled]
     https://api.nuget.org/v3/index.json
  2. MyFeed [Disabled]
     https://myfeed.com/nuget/v3/index.json
```

**dotnet nuget Parity**:
This command matches `dotnet nuget list source` behavior exactly.

---

### Command: add source

**Synopsis**: Adds a new package source

```bash
gonuget add source <source-url> [options]
```

**Arguments**:
- `source-url` (string, required): Source URL (e.g., https://api.nuget.org/v3/index.json)

**Flags**:
- `--name` (string, required): Source name
- `--username` (string): Authentication username
- `--password` (string): Authentication password
- `--store-password-in-clear-text` (bool): Store password unencrypted (not recommended)
- `--valid-authentication-types` (string): Comma-separated auth types (basic, negotiate, kerberos)
- `--configfile` (string): Specific config file to modify

**Example**:
```bash
# Add public source
gonuget add source https://myfeed.com/nuget/v3/index.json --name MyFeed

# Add authenticated source
gonuget add source https://private.feed.com/v3/index.json \
  --name PrivateFeed \
  --username user \
  --password pass

# Add to specific config file
gonuget add source https://myfeed.com/v3/index.json \
  --name MyFeed \
  --configfile ~/custom-nuget.config
```

**dotnet nuget Parity**:
This command matches `dotnet nuget add source` behavior exactly.

---

### Command: remove source

**Synopsis**: Removes a package source

```bash
gonuget remove source [options]
```

**Flags**:
- `--name` (string, required): Source name to remove
- `--configfile` (string): Specific config file to modify

**Example**:
```bash
# Remove source by name
gonuget remove source --name MyFeed

# Remove from specific config file
gonuget remove source --name MyFeed --configfile ~/custom-nuget.config
```

**dotnet nuget Parity**:
This command matches `dotnet nuget remove source` behavior exactly.

---

### Command: update source

**Synopsis**: Updates an existing package source

```bash
gonuget update source [options]
```

**Flags**:
- `--name` (string, required): Source name to update
- `--source` (string): New source URL
- `--username` (string): Authentication username
- `--password` (string): Authentication password
- `--store-password-in-clear-text` (bool): Store password unencrypted (not recommended)
- `--valid-authentication-types` (string): Comma-separated auth types
- `--configfile` (string): Specific config file to modify

**Example**:
```bash
# Update source URL
gonuget update source --name MyFeed --source https://new.feed.com/v3/index.json

# Update authentication
gonuget update source --name PrivateFeed --username newuser --password newpass
```

**dotnet nuget Parity**:
This command matches `dotnet nuget update source` behavior exactly.

---

### Command: enable source

**Synopsis**: Enables a disabled package source

```bash
gonuget enable source [options]
```

**Flags**:
- `--name` (string, required): Source name to enable
- `--configfile` (string): Specific config file to modify

**Example**:
```bash
# Enable source
gonuget enable source --name MyFeed

# Enable in specific config file
gonuget enable source --name MyFeed --configfile ~/custom-nuget.config
```

**dotnet nuget Parity**:
This command matches `dotnet nuget enable source` behavior exactly.

---

### Command: disable source

**Synopsis**: Disables a package source without removing it

```bash
gonuget disable source [options]
```

**Flags**:
- `--name` (string, required): Source name to disable
- `--configfile` (string): Specific config file to modify

**Example**:
```bash
# Disable source
gonuget disable source --name MyFeed

# Disable in specific config file
gonuget disable source --name MyFeed --configfile ~/custom-nuget.config
```

**dotnet nuget Parity**:
This command matches `dotnet nuget disable source` behavior exactly.

---

### Command: trusted-signers

**Synopsis**: Manages trusted package signers

```bash
gonuget trusted-signers <action> [options]
```

**Actions**:
- `list`: List all trusted signers
- `add`: Add a trusted signer (author, repository, or certificate)
- `remove`: Remove a trusted signer
- `sync`: Sync repository signers from a package

**Flags (add)**:
- `-Name, -n` (string, required): Signer name
- `-ServiceIndex` (string): Repository service index URL (for repository signers)
- `-CertificateFingerprint` (string): Certificate SHA256 fingerprint
- `-FingerprintAlgorithm` (string): Fingerprint algorithm (SHA256, SHA384, SHA512)
- `-AllowUntrustedRoot` (bool): Allow certificates with untrusted root
- `-Author` (bool): Trust as author signer
- `-Repository` (bool): Trust as repository signer
- `-Owners` (string, repeatable): Repository owner names to trust

**Flags (sync)**:
- Package path (argument): Path to package to sync signers from

**Behavior**:
Manages signature validation configuration for package integrity.

**Example**:
```bash
# Add trusted author certificate
gonuget trusted-signers add -Name MyCompany \
  -CertificateFingerprint 1234567890ABCDEF... \
  -Author

# Add trusted repository
gonuget trusted-signers add -Name NuGetOrg \
  -ServiceIndex https://api.nuget.org/v3/index.json \
  -Repository

# List trusted signers
gonuget trusted-signers list

# Sync from package
gonuget trusted-signers sync MyPackage.1.0.0.nupkg
```

---

### Command: verify

**Synopsis**: Verifies package integrity and signatures

```bash
gonuget verify <package> [options]
```

**Arguments**:
- `package` (string, required): Path to .nupkg file

**Flags**:
- `-Signatures` (bool): Verify package signatures (default: true)
- `-CertificateFingerprint` (string, repeatable): Expected certificate fingerprint(s)
- `-All` (bool): Verify all validation rules

**Verification Steps**:
1. **Package Integrity**:
   - ZIP structure valid
   - Required files present (.nuspec, [Content_Types].xml)
   - OPC compliance

2. **Signature Validation** (if `-Signatures`):
   - Primary signature present
   - Signature cryptographically valid
   - Certificate chain trusted
   - Timestamp valid
   - Certificate purpose correct (code signing)

3. **Content Hash** (if present):
   - Verify SHA512 content hash in signature

4. **Certificate Fingerprint** (if `-CertificateFingerprint`):
   - Match against expected fingerprint(s)

**Output**:
```
Verifying MyPackage.1.0.0.nupkg...
  Package structure... ✓
  Signature present... ✓
  Signature valid... ✓
  Certificate trusted... ✓
  Timestamp valid... ✓
  Certificate fingerprint matches... ✓
Package verification passed.
```

**Exit Codes**:
- 0: Verification passed
- 1: Package invalid
- 2: Signature missing
- 3: Signature invalid
- 4: Certificate not trusted
- 5: Timestamp invalid or expired
- 6: Certificate fingerprint mismatch

---

## Configuration Management

### Configuration Files

**Locations** (in precedence order):
1. `--ConfigFile` flag
2. `$NUGET_CONFIG` environment variable
3. `.gonuget.yaml` in current directory
4. `.nuget/gonuget.yaml` in current directory
5. `gonuget.yaml` in project root (walk up to find .git)
6. `NuGet.config` in project root
7. `~/.nuget/gonuget.yaml` (user-level)
8. `~/.nuget/NuGet.config` (user-level)
9. System-level NuGet.config (platform-specific)

**gonuget.yaml Format**:
```yaml
# Package sources
sources:
  - name: nuget.org
    url: https://api.nuget.org/v3/index.json
    enabled: true
  - name: private-feed
    url: https://private.feed.com/v3/index.json
    enabled: true
    credentials:
      username: user
      password: encrypted:AQAAANCM...

# Global settings
settings:
  globalPackagesFolder: ~/.nuget/packages
  repositoryPath: ./packages
  defaultPushSource: https://api.nuget.org/v3/index.json
  signatureValidationMode: require

# HTTP settings
http:
  proxy: http://proxy:8080
  timeout: 300
  userAgent: gonuget/1.0.0

# Telemetry
telemetry:
  enabled: true
  level: anonymous

# Logging
logging:
  level: info
  format: text
  output: stderr
```

**NuGet.config Compatibility**:
Read and write standard NuGet.config XML format for maximum compatibility with nuget.exe and dotnet CLI.

### Credential Management

**Storage Options**:
1. **OS Keychain** (preferred):
   - Windows: Windows Credential Manager
   - macOS: Keychain
   - Linux: Secret Service API (GNOME Keyring, KWallet)

2. **Encrypted File** (fallback):
   - `~/.nuget/credentials.json` encrypted with user-specific key

3. **Environment Variables**:
   - `NUGET_CREDENTIALS_<SOURCE>` where `<SOURCE>` is normalized source name
   - Format: `username:password` or `Bearer <token>`

**External Credential Providers**:
External credential provider executables (compatible with NuGet Credential Provider protocol):
```bash
# Credential provider executables in ~/.nuget/CredentialProviders/
CredentialProvider.Microsoft.exe       # Azure Artifacts
credentialprovider-awscodeartifact    # AWS CodeArtifact
credentialprovider-jfrog              # JFrog Artifactory
```

See [Extensibility](#extensibility) section for credential provider protocol specification.

---

## Output & Formatting

### Console Abstraction

**Features**:
- Color support detection (TTY, TERM environment variable)
- Progress bars for long operations
- Spinners for indeterminate operations
- Table formatting for list outputs
- JSON output mode for scripting
- Quiet mode (errors only)
- Verbose mode (debug information)

**Color Scheme**:
```go
const (
    ColorSuccess = color.FgGreen
    ColorError   = color.FgRed
    ColorWarning = color.FgYellow
    ColorInfo    = color.FgCyan
    ColorDebug   = color.FgWhite
    ColorHeader  = color.Bold | color.FgWhite
)
```

**Verbosity Levels**:
- `Quiet`: Errors only
- `Normal`: Errors, warnings, key operations (default)
- `Detailed`: Above + progress details
- `Diagnostic`: Above + HTTP requests, cache hits, timing

### Progress Indicators

**Download Progress**:
```
Downloading Newtonsoft.Json.13.0.3.nupkg
[████████████████████████        ] 73% (2.4 MB/s) ETA: 3s
```

**Indeterminate Progress**:
```
Resolving dependencies... ⠋
```

**Multi-Package Progress**:
```
Restoring packages (4 total)...
  Newtonsoft.Json 13.0.3    [████████████] 100% ✓
  Serilog 3.1.1             [████████████] 100% ✓
  AutoMapper 12.0.1         [██████      ]  60%
  FluentValidation 11.8.1   [            ]   0%
```

### Structured Output

**JSON Mode** (`--format json` or `-o json`):
```json
{
  "success": true,
  "command": "install",
  "packages": [
    {
      "id": "Newtonsoft.Json",
      "version": "13.0.3",
      "framework": "net8.0",
      "path": "./packages/Newtonsoft.Json.13.0.3"
    }
  ],
  "duration": 2.34,
  "timestamp": "2025-10-25T12:34:56Z"
}
```

**Machine-Readable Output** (for CI/CD):
```
::set-output name=package_count::4
::set-output name=duration::2.34
```

---

## Authentication & Security

### Authentication Methods

1. **API Key**:
   - Header: `X-NuGet-ApiKey: <api-key>`
   - Stored in NuGet.config or OS keychain
   - Used for package push/delete operations

2. **Bearer Token**:
   - Header: `Authorization: Bearer <token>`
   - For OAuth2/Azure AD authentication
   - Obtained via credential providers

3. **Basic Authentication**:
   - Header: `Authorization: Basic <base64(username:password)>`
   - For private feeds with username/password
   - Credentials stored securely or obtained via credential providers

4. **Client Certificates**:
   - Mutual TLS authentication
   - Certificate selection by thumbprint or subject
   - Configured per-source in NuGet.config

5. **External Credential Providers**:
   - Delegated authentication to external executables
   - Compatible with NuGet Credential Provider protocol
   - Discovers providers in `~/.nuget/CredentialProviders/` or `$NUGET_CREDENTIALPROVIDERS_PATH`
   - Supports interactive and non-interactive authentication
   - Examples: Azure Artifacts, AWS CodeArtifact, JFrog Artifactory

### Security Features

**TLS Configuration**:
- Require TLS 1.2+ by default
- Certificate validation (can be disabled with `--insecure` flag and warning)
- Certificate pinning for known sources
- System certificate store integration

**Package Integrity**:
- SHA512 content hash validation
- PKCS#7 signature verification
- Certificate chain validation
- Timestamp verification (package signed within cert validity)

**Credential Security**:
- Never log credentials
- Encrypt stored credentials
- Memory zeroing after use
- Avoid credential exposure in errors

---

## Error Handling

### Error Taxonomy

**Network Errors**:
- Connection refused
- DNS resolution failed
- Timeout
- TLS handshake failed
- HTTP 4xx/5xx errors

**Package Errors**:
- Package not found
- Version not found
- Invalid package format
- Corrupted package
- Signature invalid

**Dependency Errors**:
- Dependency resolution failed
- Version conflict
- Circular dependency
- Missing dependency

**Configuration Errors**:
- Invalid config file
- Source not found
- Authentication failed

**Filesystem Errors**:
- Permission denied
- Disk full
- Path too long (Windows)

### Error Output Format

**Standard Error**:
```
Error: Package 'NonExistentPackage' not found in any source.

Searched sources:
  • https://api.nuget.org/v3/index.json
  • https://private.feed.com/v3/index.json

Suggestions:
  - Check package ID spelling
  - Verify package exists: gonuget search NonExistentPackage
  - Add additional sources: gonuget sources add --help

Run 'gonuget --help' for usage information.
```

**Verbose Error** (with `--verbose`):
```
Error: Package 'NonExistentPackage' not found in any source.

Details:
  Package ID: NonExistentPackage
  Version constraint: [1.0.0, 2.0.0)
  Target framework: net8.0

Source query results:
  • https://api.nuget.org/v3/index.json
    Response: 404 Not Found
    Duration: 234ms

  • https://private.feed.com/v3/index.json
    Response: 404 Not Found
    Duration: 456ms

Stack trace:
  at resolver.Resolve() (resolver.go:123)
  at client.InstallPackage() (client.go:456)
  at commands.Install() (install.go:78)
```

### Retry Logic

**Transient Errors** (auto-retry):
- HTTP 429 (Rate Limit) - respect Retry-After header
- HTTP 503 (Service Unavailable)
- Timeout errors
- Connection reset

**Retry Policy**:
- Max retries: 3
- Backoff: Exponential (1s, 2s, 4s)
- Circuit breaker: Open after 5 consecutive failures

---

## Performance & Optimization

### Performance Targets

- **Startup Time**: < 50ms (cold start)
- **Search Latency**: < 500ms (network-bound)
- **Package Resolution**: < 2s for typical project (10 dependencies)
- **Download Speed**: Network-limited (HTTP/2 multiplexing)
- **Memory Usage**: < 50MB for typical operations

### Optimization Strategies

**Parallel Operations**:
- Concurrent package downloads (default: 4 parallel)
- Parallel dependency resolution
- Parallel source queries

**Caching**:
- HTTP response cache (ETag-based)
- In-memory package metadata cache (LRU)
- Disk cache for downloaded packages

**HTTP Optimizations**:
- HTTP/2 connection reuse
- HTTP/3 support with QUIC
- Connection pooling
- Compression (gzip, br)

**Lazy Loading**:
- Load credentials only when needed
- Defer credential provider discovery until authentication required
- Stream large packages

---

## Cross-Platform Support

### Platform-Specific Features

**Windows**:
- Windows Credential Manager integration
- Windows Certificate Store (CryptoAPI)
- MSBuild integration for project packaging (REQUIRED v1.0)
- MSBuild discovery via Visual Studio setup API
- Support for MSBuild 14.0, 15.0, 16.0, 17.0+

**macOS**:
- Keychain integration
- Security.framework for certificates
- Xcode project support (future)

**Linux**:
- Secret Service API (libsecret)
- systemd credential storage
- Desktop notifications (notify-send)

### Path Handling

**Conventions**:
- Always use forward slashes internally
- Convert to OS-specific on filesystem operations
- Handle long paths on Windows (\\?\)

**Package Paths**:
```
Windows:  C:\Users\User\.nuget\packages
macOS:    /Users/user/.nuget/packages
Linux:    /home/user/.nuget/packages
```

---

## Testing Strategy

### Test Categories

1. **Unit Tests**:
   - Command parsing and validation
   - Output formatting
   - Configuration management
   - Error handling

2. **Integration Tests**:
   - End-to-end command execution
   - Real package operations (with test feed)
   - Multi-source scenarios

3. **CLI Interop Tests** (C# Bridge):
   - **Primary**: Compare gonuget vs `dotnet nuget` behavior (all platforms)
   - **Secondary**: Compare gonuget vs `nuget.exe` behavior (Windows only, for compatibility verification)
   - Validate output parity (stdout, stderr, exit codes)
   - Test config file compatibility (NuGet.config round-tripping)
   - Verify identical results for identical inputs
   - Test command structure: `<verb> <noun>` pattern
   - Test flag names: kebab-case (`--configfile`, `--name`, `--source`)
   - Verify config subcommands: `config get`, `config set`, `config list`
   - Verify source commands: `list source`, `add source`, `remove source`, `update source`, `enable source`, `disable source`

4. **Performance Tests**:
   - Benchmark startup time
   - Benchmark resolution time
   - Memory profiling

5. **Platform Tests**:
   - Windows-specific features
   - macOS-specific features
   - Linux-specific features

### Test Fixtures

**Mock Server**:
- In-memory NuGet V3 server for fast tests
- Configurable responses (errors, delays)
- Package repository with test packages

**CLI Interop Bridge**:
- C# executable that invokes both `dotnet nuget` and `gonuget` (primary comparison)
- On Windows, also compares with `nuget.exe` for compatibility verification
- Compares outputs, exit codes, and file system changes
- Returns JSON comparison report with detailed diffs
- Runs on all platforms (.NET 9.0 SDK required)
- Uses JSON-RPC protocol for command execution
- Validates command structure (`<verb> <noun>` pattern)
- Validates flag naming (kebab-case)

**Test Packages**:
- Simple packages (no dependencies)
- Complex packages (deep dependency trees)
- Packages with various frameworks
- Signed packages
- Invalid/corrupt packages

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-2)

**Goals**: CLI framework, basic commands, config management

**Deliverables**:
- CLI application structure (cobra)
- Configuration loading (viper + NuGet.config XML)
- Console abstraction with colors and progress
- Commands: `help`, `version`, `config` (get/set/list), `list source`, `add source`, `remove source`, `enable source`, `disable source`, `update source`
- **Total Commands**: 9/21 (43%)

**Command Structure**:
- `<verb> <noun>` pattern (e.g., `add source`, not `sources add`)
- Kebab-case flags (e.g., `--configfile`, not `-ConfigFile`)

**Tests**:
- Unit tests for config and CLI parsing
- CLI interop tests comparing with `dotnet nuget` commands

---

### Phase 2: Package Management (Weeks 3-5)

**Goals**: Package search, add package, restore with CPM support

**Deliverables**:
- Commands: `search`, `install`, `add package`, `restore`
- Package metadata fetching
- Package download with progress
- Project file manipulation (.csproj parsing and modification)
- **Central Package Management (CPM)** - Full support:
  - Directory.Packages.props detection and manipulation
  - PackageVersion management
  - VersionOverride support
  - Multi-project CPM scenarios
- Simple extraction (no dependency resolution)
- Cache integration
- **Total Commands**: 13/21 (62%)

**Tests**:
- Integration tests with test feed
- CLI interop tests for search, install, add package, restore
- CPM interop tests validating parity with `dotnet add package`

---

### Phase 3: Dependency Resolution (Weeks 6-7)

**Goals**: Full dependency resolution and restore

**Deliverables**:
- Commands: `restore`
- Dependency graph resolution
- Version conflict detection
- packages.config support
- Lock file generation
- **Total Commands**: 12/21 (57%)

**Tests**:
- Complex dependency scenarios
- CLI interop tests for restore

---

### Phase 4: Package Creation (Weeks 8-9)

**Goals**: Pack, push, spec

**Deliverables**:
- Commands: `pack`, `push`
- .nuspec parsing and generation
- OPC-compliant package creation
- File pattern matching
- Metadata extraction
- **Total Commands**: 14/21 (67%)

**Tests**:
- Package creation validation
- CLI interop tests for pack and push

---

### Phase 5: Signing & Security (Weeks 10-11)

**Goals**: Signing, verification, trust management

**Deliverables**:
- Commands: `sign`, `verify`, `trusted-signers`
- PKCS#7 signature creation/validation
- RFC 3161 timestamping
- Certificate store integration
- Trust configuration
- **Total Commands**: 17/21 (81%)

**Tests**:
- Signature validation scenarios
- CLI interop tests for sign and verify

---

### Phase 6: Advanced Features (Weeks 12-13)

**Goals**: Update, locals, remaining commands

**Deliverables**:
- Commands: `update`, `locals`, `add`, `init`, `delete`, `setapikey`, `spec`, `client-certs`, `list` (deprecated)
- Self-update mechanism
- Cache management
- API key storage
- **Total Commands**: 21/21 (100%)

**Tests**:
- Full command suite testing
- Complete CLI interop test coverage

---

### Phase 7: Polish & Optimization (Weeks 14-15)

**Goals**: Performance, UX, documentation

**Deliverables**:
- Performance optimization (profiling, benchmarks)
- Better error messages
- Shell completions (bash, zsh, fish, PowerShell)
- Man pages
- Comprehensive documentation

**Tests**: Performance benchmarks, interop parity tests

---

### Phase 8: Platform-Specific (Week 16)

**Goals**: Platform-specific features and installers

**Deliverables**:
- Windows installer (MSI or Chocolatey)
- macOS installer (Homebrew)
- Linux packages (deb, rpm, snap, flatpak)
- Windows Credential Manager integration
- Keychain integration

**Tests**: Platform-specific tests on CI

---

## Quality Requirements

### Code Quality

**Standards**:
- Go 1.25.2+ idioms
- 90%+ test coverage
- Zero linter warnings (golangci-lint)
- Godoc comments for all public APIs
- Examples in documentation

**Review Process**:
- All changes require code review
- Automated tests must pass
- Benchmark regressions require justification

### Performance Requirements

**Benchmarks** (must meet or exceed):
- Startup: < 50ms
- `gonuget search json`: < 1s
- `gonuget install Newtonsoft.Json`: < 3s (cold cache)
- `gonuget restore` (10 packages): < 10s (cold cache)
- Memory: < 100MB peak for typical operations

### Reliability Requirements

**Error Handling**:
- No panics in production code
- All errors wrapped with context
- Graceful degradation on non-critical errors
- Clear actionable error messages

**Stability**:
- Automated crash reporting (opt-in)
- Recovery from transient network errors
- Safe concurrent operations

### Security Requirements

**Audits**:
- Annual security audit
- Dependency vulnerability scanning (govulncheck)
- SAST tools in CI (gosec)

**Best Practices**:
- Principle of least privilege
- Secure defaults
- No secrets in logs or errors
- Encrypted credential storage

---

## Compatibility & Parity

### NuGet.config Compatibility

**Required**:
- Read/write standard NuGet.config XML
- Support all standard sections: `<packageSources>`, `<apikeys>`, `<config>`, etc.
- Respect NuGet.config hierarchy and merging
- Support `clear` elements

**Example**:
```xml
<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" protocolVersion="3" />
    <add key="private" value="https://private.feed.com/v3/index.json" />
  </packageSources>
  <apikeys>
    <add key="https://api.nuget.org/v3/index.json" value="[Encrypted]" />
  </apikeys>
  <config>
    <add key="globalPackagesFolder" value="~/.nuget/packages" />
  </config>
</configuration>
```

### Behavioral Parity

**Primary Target**: `dotnet nuget` commands (cross-platform, modern NuGet CLI)
**Secondary Target**: `nuget.exe` (Windows only, legacy compatibility)

**Critical Behaviors**:
- Version resolution algorithm (same results as `dotnet nuget`)
- Dependency conflict resolution
- Framework compatibility matching
- Package extraction layout
- Lock file format
- Error codes
- Command structure: `<verb> <noun>` pattern (e.g., `add source`, not `sources add`)
- Flag naming: kebab-case (e.g., `--configfile`, not `-ConfigFile`)
- Config subcommands: `get`, `set`, `list` (not single command with `-Set` flag)

**Testing**:
- CLI interop test suite comparing gonuget vs `dotnet nuget` commands (all platforms)
- Windows-specific tests comparing gonuget vs `nuget.exe` (for compatibility verification)
- Identical inputs → identical outputs
- Edge cases and corner cases
- Command structure validation
- Flag name validation

**Reference Implementations**:
- [dotnet/sdk](https://github.com/dotnet/sdk) - **PRIMARY**: Official .NET SDK with `dotnet nuget` commands
- [NuGet.Client](https://github.com/NuGet/NuGet.Client) - Secondary: nuget.exe implementation
- [dotnet/docs](https://github.com/dotnet/docs) - Official NuGet documentation

---

## Extensibility

### Credential Providers

gonuget implements the NuGet Credential Provider protocol for authentication extensibility. This provides exact compatibility with existing NuGet credential providers.

**Provider Discovery**:
- `~/.nuget/CredentialProviders/` (or `%LOCALAPPDATA%\NuGet\CredentialProviders\` on Windows)
- `$NUGET_CREDENTIALPROVIDERS_PATH` environment variable (semicolon-separated paths)
- Pattern: Executables matching `CredentialProvider*.exe` (Windows) or `credentialprovider-*` (Unix)

**Provider Protocol** (stdin/stdout JSON):

**Request Format**:
```json
{
  "Uri": "https://private.feed.com/v3/index.json",
  "IsRetry": false,
  "NonInteractive": false,
  "Verbosity": "normal"
}
```

**Response Format**:
```json
{
  "Username": "user@example.com",
  "Password": "token_or_password",
  "Message": "Authenticated successfully",
  "AuthTypes": ["basic", "negotiate"]
}
```

**Exit Codes**:
- `0`: Success - credentials provided
- `1`: Provider not applicable for this URI
- `2`: Failure - provider applicable but couldn't get credentials

**Execution Flow**:
1. gonuget discovers all credential provider executables
2. For each authentication challenge, gonuget calls providers sequentially
3. Provider receives JSON request on stdin
4. Provider prompts user (if interactive) or retrieves credentials from OS keychain
5. Provider writes JSON response to stdout and exits
6. gonuget uses first successful provider's credentials

**Environment Variables Passed to Providers**:
- `NUGET_CREDENTIALPROVIDER_SESSIONID`: Unique session ID
- `NUGET_CREDENTIALPROVIDER_PARENTPROCESSID`: gonuget's process ID

**Example Providers**:
- `CredentialProvider.Microsoft.exe`: Azure Artifacts authentication
- `docker-credential-ecr-login`: AWS CodeArtifact authentication (Docker-style)
- Custom providers for OAuth2, SAML, etc.

**Implementation Note**: gonuget supports the same credential provider protocol as nuget.exe and dotnet CLI, ensuring existing credential providers work without modification.

---

## Documentation

### User Documentation

**Formats**:
- Man pages (`man gonuget`, `man gonuget-install`)
- Online documentation (static site)
- Built-in help (`gonuget help install`)
- Examples for every command

**Languages**:
- English (primary)
- Czech (cs)
- German (de)
- Spanish (es)
- French (fr)
- Italian (it)
- Japanese (ja)
- Korean (ko)
- Polish (pl)
- Brazilian Portuguese (pt-BR)
- Russian (ru)
- Turkish (tr)
- Simplified Chinese (zh-Hans)
- Traditional Chinese (zh-Hant)

**Total**: 14 languages (100% parity with nuget.exe)

### Developer Documentation

**Topics**:
- Architecture overview
- Adding new commands
- Credential provider development
- Contributing guide
- API reference (godoc)

---

## Distribution

### Binary Distribution

**Release Artifacts**:
- Standalone binaries (Linux, macOS, Windows)
- Checksums (SHA256)
- GPG signatures
- Release notes

**Platforms**:
- linux-amd64, linux-arm64, linux-386
- darwin-amd64 (Intel), darwin-arm64 (Apple Silicon)
- windows-amd64, windows-386, windows-arm64

### Package Managers

**Installation Methods**:
```bash
# Homebrew (macOS/Linux)
brew install gonuget

# Chocolatey (Windows)
choco install gonuget

# Snap (Linux)
snap install gonuget

# APT (Debian/Ubuntu)
apt install gonuget

# DNF/YUM (Fedora/RHEL)
dnf install gonuget

# AUR (Arch Linux)
yay -S gonuget

# Direct download
curl -sSL https://get.gonuget.org | sh
```

---

## Success Metrics

### User Adoption

**Goals** (Year 1):
- 10,000+ downloads
- 100+ GitHub stars
- 10+ external contributors
- Featured in Go Weekly / .NET newsletter

### Performance Benchmarks

**vs nuget.exe**:
- Startup: 10x faster (5ms vs 50ms)
- Search: 2x faster
- Restore: 1.5x faster (parallel downloads)
- Memory: 50% lower

### Reliability

**Targets**:
- 99.9% success rate for package operations (network permitting)
- < 10 crashes per 10,000 operations
- < 5% support request rate (clear documentation)

---

## Risk Mitigation

### Technical Risks

**Risk**: MSBuild integration complexity for .NET project packing
**Mitigation**: ~~Focus on .nuspec first, MSBuild integration in later phase~~ **REMOVED** - MSBuild integration is REQUIRED for v1.0 to achieve 100% parity with nuget.exe. Full MSBuild support including project file parsing, building, and property substitution must be implemented.

**Risk**: Platform-specific certificate store differences
**Mitigation**: Abstract certificate store with platform-specific implementations

**Risk**: NuGet protocol changes/versioning
**Mitigation**: Automated tests against live feeds, version detection

### Compatibility Risks

**Risk**: Subtle behavioral differences from nuget.exe
**Mitigation**: Extensive interop testing, parity test suite

**Risk**: Breaking changes in gonuget library
**Mitigation**: Semantic versioning, deprecation warnings

---

## Conclusion

This design provides a roadmap for a production-ready gonuget CLI tool with 100% functional parity to nuget.exe. The phased implementation approach ensures steady progress while maintaining quality. The credential provider compatibility ensures seamless integration with existing NuGet authentication workflows and corporate infrastructure.

**Next Steps**:
1. Review and approve design
2. Set up project structure (cmd/gonuget)
3. Begin Phase 1 implementation
4. Create initial test suite
5. Set up CI/CD pipeline for multi-platform builds

**Estimated Timeline**: 16 weeks (4 months) to feature-complete v1.0
**Team Size**: 2-3 developers + 1 technical writer
**Dependencies**: gonuget library completion (M8)

---

**Document Version**: 1.0
**Last Updated**: 2025-10-25
**Status**: Awaiting Approval
