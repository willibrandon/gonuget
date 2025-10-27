# gonuget CLI

A fast, native NuGet package manager written in Go.

## Features

- **15-17x faster** than dotnet nuget for CLI operations
- **30-35% less memory** per command invocation
- **Zero runtime overhead** - native binary, no .NET runtime required
- **100% compatible** with NuGet.config and dotnet project files
- **Cross-platform** - Windows, macOS, Linux

## Installation

### From Source

```bash
git clone https://github.com/willibrandon/gonuget
cd gonuget
make build
./gonuget --version
```

### Pre-built Binaries

Coming soon.

## Quick Start

```bash
# Add a package source
gonuget add source https://api.nuget.org/v3/index.json --name "NuGet.org"

# List configured sources
gonuget list source

# Add a package to a project
gonuget add package Newtonsoft.Json --version 13.0.3

# Get configuration value
gonuget config get repositoryPath
```

## Commands

### Configuration

```bash
# Get a configuration value
gonuget config get <key> [--configfile <path>]

# Set a configuration value
gonuget config set <key> <value> [--configfile <path>]

# List all configuration
gonuget config list [--configfile <path>]
```

### Source Management

```bash
# Add a package source
gonuget add source <URL> --name <name> [options]

# List package sources
gonuget list source [--format Detailed|Short]

# Remove a package source
gonuget remove source --name <name>

# Enable a package source
gonuget enable source --name <name>

# Disable a package source
gonuget disable source --name <name>

# Update a package source
gonuget update source --name <name> --source <URL>
```

### Package Management

```bash
# Add a package reference to a project
gonuget add package <PACKAGE_ID> [options]
  --version <VERSION>           Specific version to add
  --framework <FRAMEWORK>       Target framework
  --no-restore                  Don't restore after adding
  --source <SOURCE>             Package source URL
  --prerelease                  Include prerelease versions
  --project <PATH>              Project file path
```

## Examples

### Configure NuGet Sources

```bash
# Add the official NuGet.org source
gonuget add source https://api.nuget.org/v3/index.json --name "NuGet.org"

# Add a private feed with credentials
gonuget add source https://pkgs.dev.azure.com/org/_packaging/feed/nuget/v3/index.json \
  --name "MyFeed" \
  --username user \
  --password $PAT

# List all sources
gonuget list source --format Detailed
```

### Work with Packages

```bash
# Add latest stable version
gonuget add package Newtonsoft.Json

# Add specific version
gonuget add package Newtonsoft.Json --version 13.0.3

# Add latest prerelease
gonuget add package Newtonsoft.Json --prerelease

# Add without triggering restore
gonuget add package Newtonsoft.Json --no-restore

# Add to specific project
gonuget add package Newtonsoft.Json --project ./MyApp/MyApp.csproj
```

### Configuration Management

```bash
# Set global packages folder
gonuget config set globalPackagesFolder /path/to/packages

# Get current repository path
gonuget config get repositoryPath

# List all configuration
gonuget config list

# Use specific config file
gonuget config get repositoryPath --configfile ./custom.config
```

## Performance

gonuget is designed for speed:

| Command | gonuget | dotnet nuget | Speedup |
|---------|---------|--------------|---------|
| version | ~6.5ms | ~101ms | **15x faster** |
| config get | ~6.5ms | ~112ms | **17x faster** |
| list source | ~6.6ms | ~117ms | **17x faster** |
| add source | ~7.0ms | N/A | N/A |

**Why is it faster?**
- Native compilation (no runtime initialization)
- Zero startup overhead (no JIT, no assembly loading)
- Efficient execution (direct syscalls)
- Minimal dependencies

See [benchmarks/README.md](benchmarks/README.md) for detailed performance analysis.

## Command Structure

gonuget follows the same patterns as dotnet:

```bash
# Source management (matches dotnet nuget)
gonuget add source <URL>      # dotnet nuget add source <URL>
gonuget list source           # dotnet nuget list source
gonuget remove source <name>  # dotnet nuget remove source <name>

# Package management (matches dotnet add)
gonuget add package <ID>      # dotnet add package <ID>
```

All flags use kebab-case naming:
- `--configfile` not `--ConfigFile`
- `--name` not `--Name`
- `--store-password-in-clear-text` not `--StorePasswordInClearText`

## Configuration Files

gonuget uses standard NuGet.config files and is fully compatible with:
- `%APPDATA%\NuGet\NuGet.config` (Windows)
- `~/.nuget/NuGet/NuGet.config` (macOS/Linux)
- `./NuGet.config` (project-local)

Configuration hierarchy follows NuGet behavior:
1. Closest NuGet.config to current directory
2. Parent directories (walking up the tree)
3. User-level config
4. Machine-level config

## Project Files

gonuget works directly with .NET project files:
- `.csproj` (C# projects)
- `.fsproj` (F# projects)
- `.vbproj` (VB.NET projects)

Changes are written in a format compatible with:
- Visual Studio
- Visual Studio Code
- dotnet CLI
- MSBuild

## Development

### Building

```bash
make build              # Build all binaries
make test               # Run all tests
make test-go-unit       # Run unit tests only
make bench              # Run benchmarks
```

### Testing

```bash
# Run all tests
make test

# Run CLI tests only
go test ./cmd/gonuget/... -v

# Run with coverage
go test ./cmd/gonuget/commands -cover

# Run benchmarks
go test -tags=benchmark -bench=. ./cmd/gonuget
```

### Project Structure

```
cmd/gonuget/
├── main.go              # Entry point
├── cli/                 # CLI framework
│   ├── app.go           # Root command
│   └── version.go       # Version info
├── commands/            # Command implementations
│   ├── add.go           # Parent add command
│   ├── add_package.go   # Add package subcommand
│   ├── source_*.go      # Source management
│   ├── config.go        # Config command
│   └── version.go       # Version command
├── config/              # NuGet.config handling
├── output/              # Console output
├── project/             # Project file handling
└── benchmarks/          # Performance benchmarks
```

## Contributing

See [../../CONTRIBUTING.md](../../CONTRIBUTING.md) for development guidelines.

## License

See [../../LICENSE](../../LICENSE)
