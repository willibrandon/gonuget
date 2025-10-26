# Product Requirements Document: gonuget CLI

**Document Version**: 1.1
**Last Updated**: 2025-10-25
**Status**: Draft
**Product**: gonuget CLI Tool
**Target Release**: v1.0
**Revision**: Updated to target dotnet nuget compatibility

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Product Vision](#product-vision)
3. [Goals and Objectives](#goals-and-objectives)
4. [Target Audience](#target-audience)
5. [Functional Requirements](#functional-requirements)
6. [Non-Functional Requirements](#non-functional-requirements)
7. [User Stories](#user-stories)
8. [Command Requirements](#command-requirements)
9. [Platform Requirements](#platform-requirements)
10. [Integration Requirements](#integration-requirements)
11. [Performance Requirements](#performance-requirements)
12. [Security Requirements](#security-requirements)
13. [Usability Requirements](#usability-requirements)
14. [Compatibility Requirements](#compatibility-requirements)
15. [Success Metrics](#success-metrics)
16. [Dependencies and Constraints](#dependencies-and-constraints)
17. [Acceptance Criteria](#acceptance-criteria)
18. [Out of Scope](#out-of-scope)
19. [Risks and Mitigation](#risks-and-mitigation)
20. [Appendix](#appendix)

---

## Executive Summary

gonuget CLI is a production-ready command-line tool providing 100% functional parity with `dotnet nuget` commands and `nuget.exe`. Built as a native Go binary, it offers superior startup performance, cross-platform support, and modern CLI user experience while maintaining complete compatibility with existing NuGet workflows, configuration files, and credential providers.

**Key Value Propositions**:
- **Drop-in Replacement**: Works identically to `dotnet nuget` and `nuget.exe` in all scenarios
- **Native Performance**: 10x faster startup (5ms vs 50ms)
- **Single Binary**: No .NET SDK/Framework dependency
- **Modern UX**: Colored output, progress indicators, structured logging
- **Universal Platform Support**: Windows, Linux, macOS with single codebase
- **Unified CLI**: Single tool combining `dotnet nuget` and `nuget.exe` functionality

---

## Product Vision

Create the definitive cross-platform NuGet CLI tool that sets the standard for package management performance and user experience while maintaining complete backward compatibility with the NuGet ecosystem.

**Strategic Positioning**:
- Primary CLI for Go developers needing NuGet packages
- Performance-focused alternative to nuget.exe for CI/CD pipelines
- Cross-platform solution for teams working across Windows, Linux, and macOS
- Foundation for future NuGet tooling innovation

---

## Goals and Objectives

### Primary Goals

1. **Complete Parity**: 100% functional compatibility with `dotnet nuget` and `nuget.exe` commands
2. **Superior Performance**: Sub-50ms startup time, faster package operations
3. **Platform Independence**: Identical behavior across Windows, Linux, macOS
4. **Production Quality**: Zero-downtime deployments, <0.01% crash rate
5. **Seamless Migration**: Drop-in replacement requiring zero configuration changes

### Secondary Goals

6. **Modern UX**: Improved error messages, progress indicators, colored output
7. **Developer Productivity**: Faster feedback loops, better debugging tools
8. **Enterprise Ready**: Audit logging, telemetry, compliance features
9. **Community Building**: Open-source contribution model, plugin ecosystem

### Success Criteria

- [ ] All `dotnet nuget` and `nuget.exe` commands implemented with identical behavior
- [ ] 100% of CLI interop tests passing (dotnet nuget comparison)
- [ ] 100% of NuGet.Client interop tests passing (library-level)
- [ ] Startup time < 50ms (vs dotnet nuget ~100-200ms, nuget.exe ~50-100ms)
- [ ] Package restore 1.5x faster than `dotnet restore` (parallel downloads)
- [ ] Zero breaking changes from `dotnet nuget` / `nuget.exe` in CI/CD scenarios
- [ ] 10,000+ downloads in first 6 months
- [ ] 95% user satisfaction rating

---

## Target Audience

### Primary Users

**1. Go Developers**
- Need: Access NuGet packages from Go applications
- Pain: No native Go tools for NuGet, must install .NET runtime
- Value: Native binary, familiar CLI patterns

**2. DevOps Engineers**
- Need: Fast, reliable package restoration in CI/CD
- Pain: `dotnet nuget` / `nuget.exe` startup time, .NET SDK/Framework dependency
- Value: 10x faster startup, single binary deployment

**3. Cross-Platform Teams**
- Need: Consistent tooling across Windows, Linux, macOS
- Pain: `nuget.exe` Windows-only, `dotnet nuget` requires .NET SDK
- Value: True cross-platform binary with identical behavior, no SDK requirement

### Secondary Users

**4. Enterprise IT**
- Need: Auditable, secure package management
- Pain: Limited logging and telemetry in nuget.exe
- Value: Built-in audit logs, telemetry, compliance reporting

**5. Package Publishers**
- Need: Reliable package publishing workflow
- Pain: nuget.exe crash on large packages, slow uploads
- Value: Streaming uploads, better error handling

---

## Functional Requirements

### FR-1: Command Completeness

**Requirement**: gonuget SHALL implement all 21 commands matching `dotnet nuget` with identical syntax and behavior.

**Commands (Priority Order)**:

**Foundation Commands** (P0 - Must Have):
1. `help` - Display help information
2. `version` - Display version information (via `--version` flag)
3. `config` - Manage configuration (subcommands: get, set, list)

**Source Management Commands** (P0 - Must Have):
4. `list source` - List configured package sources
5. `add source` - Add a new package source
6. `remove source` - Remove a package source
7. `update source` - Update an existing source
8. `enable source` - Enable a package source
9. `disable source` - Disable a package source

**Package Operations Commands** (P0 - Must Have):
10. `search` - Search package feeds
11. `install` - Install packages
12. `restore` - Restore project dependencies
13. `pack` - Create NuGet packages
14. `push` - Publish packages to feeds

**Secondary Commands** (P1 - Should Have):
15. `list` - List packages (deprecated, delegates to search)
16. `delete` - Remove packages from feeds
17. `update` - Update packages
18. `add` - Add packages to offline feeds
19. `init` - Initialize offline feeds
20. `locals` - Manage local caches
21. `setapikey` - Store API keys

**Advanced Commands** (P2 - Nice to Have):
22. `sign` - Sign packages with certificates
23. `verify` - Verify package signatures
24. `trusted-signers` - Manage trusted signers
25. `client-certs` - Manage client certificates
26. `spec` - Generate .nuspec files

**Command Structure**:
- **Pattern**: `<verb> <noun>` (e.g., `add source`, not `sources add`)
- **Flags**: kebab-case (e.g., `--configfile`, `--name`, `--source`)
- **Config subcommands**: `get`, `set`, `list` (not single command with `-Set` flag)
- **Total**: 21 core commands for 100% `dotnet nuget` parity

**Deprecated/Out of Scope**:
- `mirror` - Deprecated in NuGet 3.2+, not implemented

**Acceptance Criteria**:
- [ ] All P0 commands functional and tested
- [ ] All P1 commands functional and tested
- [ ] All P2 commands functional and tested
- [ ] Command help matches nuget.exe format
- [ ] Exit codes match nuget.exe behavior

---

### FR-2: Configuration Management

**Requirement**: gonuget SHALL read and write NuGet.config files with identical semantics to nuget.exe.

**Details**:
- Read standard NuGet.config XML format
- Support configuration hierarchy (machine, user, project)
- Respect configuration merging rules
- Support `<clear />` elements
- Support encrypted sections

**Configuration Sections**:
- `<packageSources>` - Package source URLs
- `<apikeys>` - API keys (encrypted)
- `<config>` - Global settings
- `<packageSourceCredentials>` - Source credentials
- `<trustedSigners>` - Trusted signers
- `<packageSourceMapping>` - Source routing rules

**Acceptance Criteria**:
- [ ] Reads all NuGet.config sections
- [ ] Writes valid NuGet.config XML
- [ ] Respects configuration hierarchy
- [ ] Encrypts sensitive values (API keys, passwords)
- [ ] Compatible with nuget.exe and dotnet CLI

---

### FR-3: Package Operations

**Requirement**: gonuget SHALL support all package lifecycle operations with byte-for-byte identical output to nuget.exe.

**Operations**:

**3.1 Package Creation** (`pack`):
- Parse .nuspec files (XML)
- Parse project files (.csproj, .vbproj, .fsproj)
- Support Go project packaging (custom gonuget.yaml)
- Apply property substitutions (`$version$`, `$id$`, etc.)
- Collect files based on glob patterns
- Create OPC-compliant ZIP archives
- Generate symbols packages (.symbols.nupkg, .snupkg)
- Validate package structure and metadata

**3.2 Package Publishing** (`push`):
- Upload packages via HTTP PUT/POST
- Support NuGet V3 PackagePublish resource
- Support NuGet V2 upload endpoint
- Retry failed uploads with exponential backoff
- Push symbols packages to symbol servers
- Skip duplicates with `--skip-duplicate`

**3.3 Package Installation** (`install`, `restore`):
- Resolve dependencies with target framework compatibility
- Download packages from V2/V3 feeds
- Verify package signatures (if signature validation enabled)
- Extract to packages directory
- Generate packages.config
- Generate lock files (packages.lock.json)
- Run install.ps1 scripts (optional, security gated)

**3.4 Package Signing** (`sign`):
- Create PKCS#7 signatures
- Timestamp via RFC 3161 servers
- Support certificate stores (Windows) and files (cross-platform)
- Embed signatures in package

**3.5 Package Verification** (`verify`):
- Validate ZIP structure
- Verify PKCS#7 signatures
- Validate certificate chains
- Check timestamp validity

**Acceptance Criteria**:
- [ ] Packages created by gonuget readable by nuget.exe
- [ ] Packages created by nuget.exe readable by gonuget
- [ ] Signatures interoperable with nuget.exe
- [ ] Lock files compatible with dotnet CLI
- [ ] Package structure follows OPC spec

---

### FR-4: Dependency Resolution

**Requirement**: gonuget SHALL resolve package dependencies identically to nuget.exe using the same algorithm and conflict resolution rules.

**Details**:
- Support packages.config dependency format
- Support PackageReference dependency format (project.json, .csproj)
- Implement nearest-wins dependency resolution
- Detect and report circular dependencies
- Handle version conflicts with clear error messages
- Support dependency version constraints (Lowest, Highest, HighestMinor, HighestPatch)
- Respect framework compatibility rules

**Algorithm**: Implement RemoteDependencyWalker algorithm matching NuGet.Client

**Acceptance Criteria**:
- [ ] Identical resolution results for sample projects
- [ ] Conflict resolution matches nuget.exe
- [ ] Lock files match dotnet CLI output
- [ ] Framework compatibility rules identical

---

### FR-5: Authentication

**Requirement**: gonuget SHALL support all authentication methods available in nuget.exe with exact protocol compatibility.

**Authentication Methods**:

**5.1 API Key Authentication**:
- Store API keys in NuGet.config (encrypted)
- Store API keys in OS keychain
- Send `X-NuGet-ApiKey` header
- Support source-specific and default API keys

**5.2 Basic Authentication**:
- Prompt for username/password
- Store credentials in OS keychain
- Send `Authorization: Basic` header

**5.3 Bearer Token Authentication**:
- Obtain tokens via credential providers
- Send `Authorization: Bearer` header
- Support token refresh

**5.4 Client Certificate Authentication**:
- Load certificates from system stores (Windows)
- Load certificates from PEM files (Linux/macOS)
- Select certificates by thumbprint or subject
- Mutual TLS handshake

**5.5 External Credential Providers**:
- Discover providers in `~/.nuget/CredentialProviders/`
- Discover providers via `$NUGET_CREDENTIALPROVIDERS_PATH`
- Execute providers with JSON request on stdin
- Parse JSON response from stdout
- Support exit codes (0=success, 1=not applicable, 2=failure)
- Pass environment variables (`NUGET_CREDENTIALPROVIDER_SESSIONID`, etc.)
- Support interactive and non-interactive modes
- Cache credentials per-session

**Acceptance Criteria**:
- [ ] API keys stored/retrieved identically to nuget.exe
- [ ] Basic auth prompts functional
- [ ] Client certificates functional on all platforms
- [ ] Existing credential providers work without modification
- [ ] `CredentialProvider.Microsoft.exe` compatibility verified

---

### FR-6: Protocol Support

**Requirement**: gonuget SHALL support NuGet V2 and V3 protocols with automatic detection and fallback.

**6.1 NuGet V3 Protocol**:
- Service index discovery (`index.json`)
- Registration resource (package metadata)
- Package download resource
- Search resource
- Autocomplete resource
- PackagePublish resource (push)

**6.2 NuGet V2 Protocol**:
- OData feed discovery
- XML/Atom parsing
- FindPackagesById operation
- Search operation
- Package download
- Package publish

**6.3 Protocol Detection**:
- Attempt V3 first (check for JSON content-type)
- Fall back to V2 (check for XML/Atom content-type)
- Remember protocol per-source (cache)
- Report protocol version in verbose mode

**Acceptance Criteria**:
- [ ] Works with nuget.org (V3)
- [ ] Works with legacy V2 feeds
- [ ] Auto-detection functional
- [ ] Falls back gracefully on protocol errors

---

### FR-7: Output and Formatting

**Requirement**: gonuget SHALL provide superior output formatting while maintaining compatibility with nuget.exe output parsing.

**7.1 Output Modes**:
- **Normal**: Human-readable with colors and progress
- **Quiet**: Errors only
- **Detailed**: Include debug information
- **JSON**: Machine-readable structured output
- **NonInteractive**: No prompts, suitable for CI/CD

**7.2 Output Features**:
- Colored output (disable if not TTY)
- Progress bars for downloads
- Spinners for indeterminate operations
- Table formatting for list/search results
- Unicode support (emoji indicators: ✓, ✗, ⚠)

**7.3 Compatibility**:
- JSON output for CI/CD parsing
- Machine-readable output format (e.g., `::set-output`)
- Identical exit codes to nuget.exe

**Acceptance Criteria**:
- [ ] Colors disable in non-TTY environments
- [ ] Progress bars functional
- [ ] JSON output parseable
- [ ] Exit codes match nuget.exe

---

## Non-Functional Requirements

### NFR-1: Performance

**Startup Time**:
- **Requirement**: gonuget SHALL start in < 50ms (cold start, P50)
- **Baseline**: nuget.exe ~50-100ms
- **Target**: 10x improvement over nuget.exe

**Command Execution**:
- **search**: < 1s (network-bound)
- **install** (single package, cold cache): < 5s
- **restore** (10 packages, cold cache): < 15s
- **pack**: < 2s for typical project

**Resource Usage**:
- **Memory**: < 100MB peak for typical operations
- **CPU**: < 50% single core utilization (except during parallel downloads)

**Acceptance Criteria**:
- [ ] Startup time benchmarks passing
- [ ] Command execution within targets
- [ ] Memory profiling < 100MB
- [ ] No performance regressions vs previous versions

---

### NFR-2: Reliability

**Uptime/Availability**:
- **Requirement**: gonuget SHALL have < 0.01% crash rate
- **Recovery**: Graceful handling of network errors, malformed responses
- **Data Integrity**: Never corrupt NuGet.config or packages

**Error Handling**:
- All errors SHALL include actionable error messages
- Network errors SHALL retry automatically (3 attempts, exponential backoff)
- Transient errors SHALL be distinguished from permanent failures

**Acceptance Criteria**:
- [ ] Crash rate < 0.01% in production telemetry
- [ ] Zero reported data corruption issues
- [ ] All errors have test coverage

---

### NFR-3: Security

**Package Integrity**:
- Verify SHA512 content hashes
- Verify PKCS#7 signatures (if signature validation enabled)
- Validate certificate chains
- Reject packages with invalid signatures (if required)

**Credential Security**:
- Encrypt API keys using OS-specific mechanisms:
  - Windows: DPAPI (Data Protection API)
  - macOS: Keychain
  - Linux: Secret Service API (libsecret)
- Never log credentials
- Zero credentials in error messages
- Memory zeroing after credential use

**Network Security**:
- Require TLS 1.2+ by default
- Certificate validation (reject invalid certificates)
- Support certificate pinning for corporate environments
- Warn on HTTP (non-HTTPS) feeds

**Acceptance Criteria**:
- [ ] SAST tools (gosec) passing
- [ ] Dependency vulnerability scanning clean
- [ ] Security audit completed
- [ ] No credentials in logs verified

---

### NFR-4: Maintainability

**Code Quality**:
- 90%+ test coverage
- Zero linter warnings (golangci-lint)
- Godoc comments for all public APIs
- Consistent code style

**Documentation**:
- User documentation for all commands
- Developer documentation for architecture
- API reference (godoc)
- Troubleshooting guide

**Acceptance Criteria**:
- [ ] Test coverage > 90%
- [ ] Linter passing
- [ ] Documentation complete
- [ ] No "TODO" comments in production code

---

### NFR-5: Usability

**Error Messages**:
- Clear, actionable error messages
- Suggest fixes when possible
- Include relevant context (package ID, version, source)
- No stack traces in normal mode (only in verbose)

**Help System**:
- `gonuget help` lists all commands
- `gonuget help <command>` shows detailed command help
- `gonuget <command> --help` shows command usage
- Examples for common operations

**Interactive Features**:
- Colored output for better readability
- Progress indicators for long operations
- Confirmation prompts for destructive operations
- Interactive credential prompts

**Acceptance Criteria**:
- [ ] Help text for all commands
- [ ] Error messages user-tested
- [ ] Progress bars functional
- [ ] Interactive prompts work correctly

---

## User Stories

### US-1: Developer Installing Packages

**As a** Go developer
**I want to** install NuGet packages for my project
**So that** I can use .NET libraries from Go

**Acceptance Criteria**:
- `gonuget install Newtonsoft.Json -Framework net8.0` installs package
- Dependencies resolved automatically
- Packages extracted to `./packages/` by default
- Progress bar shows download progress
- Success message displayed

---

### US-2: DevOps Engineer Restoring Packages

**As a** DevOps engineer
**I want to** restore packages in CI/CD pipeline
**So that** builds complete quickly and reliably

**Acceptance Criteria**:
- `gonuget restore MySolution.sln` restores all projects
- Parallel downloads (4 concurrent)
- Respects lock file if present
- Exit code 0 on success, non-zero on failure
- JSON output for parsing

---

### US-3: Package Author Publishing

**As a** package author
**I want to** publish my package to nuget.org
**So that** others can consume my library

**Acceptance Criteria**:
- `gonuget pack MyPackage.nuspec` creates .nupkg
- `gonuget push MyPackage.1.0.0.nupkg -ApiKey <key>` uploads
- Progress bar shows upload progress
- Symbols package automatically pushed
- Success message with package URL

---

### US-4: Enterprise User with Private Feed

**As an** enterprise developer
**I want to** authenticate to corporate NuGet feed
**So that** I can access internal packages

**Acceptance Criteria**:
- `gonuget add source https://corp.feed.com/v3/index.json --name Corporate`
- Credential provider automatically invoked on 401
- Credentials cached for session
- Works with Azure Artifacts, AWS CodeArtifact, JFrog

---

### US-5: Security Engineer Verifying Packages

**As a** security engineer
**I want to** verify package signatures
**So that** I ensure packages haven't been tampered with

**Acceptance Criteria**:
- `gonuget verify MyPackage.1.0.0.nupkg` checks signature
- Certificate chain validated
- Timestamp checked
- Clear success/failure message
- Exit code indicates verification result

---

### US-6: Team Lead Managing Configuration

**As a** team lead
**I want to** configure package sources for my team
**So that** everyone uses the same feeds

**Acceptance Criteria**:
- `gonuget config list` shows current settings
- `gonuget list source` shows configured sources
- `gonuget add source <url> --name <name>` adds source to NuGet.config
- Configuration hierarchy respected (machine > user > project)

---

## Command Requirements

### Detailed Command Specifications

Each command SHALL meet the following detailed requirements. See [CLI-DESIGN.md](../design/CLI-DESIGN.md) for full command specifications.

#### CR-1: add package

**Synopsis**: `gonuget add package <PACKAGE_ID> [options]`

**Requirements**:
- Adds a NuGet package reference to a project file (.csproj)
- Auto-discovers project file in current directory if not specified
- Supports `--version` to specify package version (default: latest stable)
- Supports `--framework` to add framework-specific references
- Supports `--no-restore` to skip automatic package restore after add
- Supports `--source` to specify package sources for version resolution
- Supports `--prerelease` to include prerelease versions when resolving latest
- Parses and modifies .csproj XML with UTF-8 BOM preservation
- **Central Package Management (CPM)** - Full support (REQUIRED):
  - Detects `ManagePackageVersionsCentrally` property in project
  - Loads and parses `Directory.Packages.props`
  - When CPM enabled:
    - Adds/updates `<PackageVersion Include="..." Version="..." />` in Directory.Packages.props
    - Adds `<PackageReference Include="..." />` WITHOUT version attribute to .csproj
  - When CPM not enabled:
    - Adds/updates `<PackageReference Include="..." Version="..." />` to .csproj
  - Supports `VersionOverride` attribute for CPM projects
  - Compatible with multi-project CPM scenarios
- Preserves XML formatting and structure
- Automatically restores packages after add (unless `--no-restore`)
- **100% parity with `dotnet add package`**

**Priority**: P0 (REQUIRED for dotnet parity)
**Dependencies**: project file manipulation, version resolution, protocol (for latest version lookup), CPM support

**Reference Implementation**:
- dotnet/sdk: `src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs`
- Includes SetCentralVersion() method for CPM support

**Acceptance Criteria**:
- [ ] Adds package reference to .csproj correctly
- [ ] Resolves latest version when not specified
- [ ] CPM detection functional
- [ ] Directory.Packages.props manipulation correct
- [ ] PackageVersion entries added correctly for CPM projects
- [ ] PackageReference without version for CPM projects
- [ ] PackageReference with version for non-CPM projects
- [ ] CLI interop tests pass comparing with `dotnet add package`
- [ ] CPM interop tests pass (100% parity)

---

#### CR-2: config

**Synopsis**:
- `gonuget config get <key> [options]`
- `gonuget config set [options]`
- `gonuget config list [options]`

**Requirements**:
- **get**: Get a configuration value by key
- **set**: Set one or more configuration values (requires `--set key=value` flag)
- **list**: List all configuration values
- Supports `--as-path` to resolve paths (get subcommand)
- Supports `--configfile` to specify config file
- Supports `--format` for list output (detailed, simple, json)
- Reads/writes NuGet.config XML
- Respects configuration hierarchy

**dotnet nuget Parity**:
- Matches `dotnet nuget config get`
- Matches `dotnet nuget config set`
- Matches `dotnet nuget config list`

**Priority**: P0
**Dependencies**: configuration library

---

#### CR-4: delete

**Synopsis**: `gonuget delete <id> <version> [apikey]`

**Requirements**:
- Deletes package from remote feed
- Supports V3 PackagePublish DELETE
- Supports V2 DELETE endpoint
- Requires API key authentication
- Confirmation prompt (unless `--non-interactive`)

**Priority**: P1
**Dependencies**: authentication, HTTP client

---

#### CR-5: help

**Synopsis**: `gonuget help [command]`

**Requirements**:
- Shows command list if no command specified
- Shows detailed help for specific command
- Supports `-All` to show all command help
- Supports `-Markdown` to generate documentation
- Help text matches nuget.exe format

**Priority**: P0
**Dependencies**: None (built-in)

---

#### CR-6: locals

**Synopsis**: `gonuget locals <resource> [options]`

**Requirements**:
- Manages local caches
- Resources: http-cache, global-packages, temp, plugins-cache, all
- Supports `-List` to show cache locations and sizes
- Supports `-Clear` to delete cache contents
- Requires confirmation for destructive operations

**Priority**: P1
**Dependencies**: file system utilities

---

#### CR-7: pack

**Synopsis**: `gonuget pack [<nuspec>|<project>] [options]`

**Requirements**:
- Creates .nupkg from .nuspec file
- Creates .nupkg from project file (.csproj, .vbproj, .fsproj) **WITH FULL MSBUILD INTEGRATION**
- **MSBuild Integration** (REQUIRED for 100% parity):
  - Parse MSBuild project files
  - Extract metadata from MSBuild properties
  - Support `-MSBuildPath` and `-MSBuildVersion` flags
  - Support `-Build` flag to build project before packing
  - Support `-IncludeReferencedProjects` flag
  - MSBuild property substitution
  - PackagesDirectory and SolutionDirectory resolution
  - Integration with NuGet.Build.Tasks
- Supports Go project packaging (gonuget.yaml)
- Applies property substitutions
- Collects files based on glob patterns
- Creates OPC-compliant ZIP
- Supports `-Symbols` for symbols packages
- Validates package structure
- Reports package size and file count

**Priority**: P0
**Dependencies**: packaging library, XML parser, MSBuild libraries, YAML parser (for Go)

---

#### CR-8: push

**Synopsis**: `gonuget push <package> [apikey] [options]`

**Requirements**:
- Uploads package to remote feed
- Supports V3 PackagePublish resource
- Supports V2 upload endpoint
- Shows upload progress bar
- Retries on transient failures
- Pushes symbols package if present (unless `--no-symbols`)
- Supports `--skip-duplicate` to succeed if exists
- Returns package URL on success

**Priority**: P0
**Dependencies**: authentication, HTTP client

---

#### CR-9: restore

**Synopsis**: `gonuget restore [<solution>|<project>|<packages.config>] [options]`

**Requirements**:
- Restores packages for solution, project, or packages.config
- Resolves dependencies
- Downloads packages in parallel
- Generates/updates lock file (packages.lock.json)
- Supports `-Recursive` for subdirectories
- Supports `-Force` to re-download
- Supports `-UseLockFile` to use/generate lock file
- Supports `-LockedMode` to require lock file
- Shows progress for each package

**Priority**: P0
**Dependencies**: resolver, packaging, protocol

---

#### CR-10: search

**Synopsis**: `gonuget search <query> [options]`

**Requirements**:
- Searches package feeds
- Supports multiple sources
- Supports `-Take` to limit results
- Supports `-Skip` for pagination
- Supports `-Prerelease` to include prerelease
- Formats output as table (or JSON with `--format json`)
- Shows download counts, verified badge
- Sorts by relevance

**Priority**: P1
**Dependencies**: protocol

---

#### CR-11: sign

**Synopsis**: `gonuget sign <package> [options]`

**Requirements**:
- Creates PKCS#7 signature
- Timestamps via RFC 3161 server
- Supports certificate from file or store
- Supports hash algorithms: SHA256, SHA384, SHA512
- Embeds signature in package
- Supports `--overwrite` for re-signing
- Validates certificate purpose (code signing)

**Priority**: P2
**Dependencies**: packaging/signatures

---

#### CR-12: Source Management Commands

**Synopsis**:
- `gonuget list source [options]`
- `gonuget add source <url> [options]`
- `gonuget remove source [options]`
- `gonuget update source [options]`
- `gonuget enable source [options]`
- `gonuget disable source [options]`

**Requirements**:

**list source**:
- Lists all configured package sources
- Supports `--configfile` to specify config file
- Supports `--format` (detailed, simple, json)
- Shows source name, URL, enabled status

**add source**:
- Adds a new package source to NuGet.config
- Requires source URL as positional argument
- Requires `--name` flag for source name
- Supports `--username` and `--password` for authentication
- Supports `--configfile` to specify config file

**remove source**:
- Removes a package source from NuGet.config
- Requires `--name` flag for source name to remove
- Supports `--configfile` to specify config file

**update source**:
- Updates an existing package source
- Requires `--name` flag for source name
- Supports `--source` to update URL
- Supports `--username` and `--password` to update credentials
- Supports `--configfile` to specify config file

**enable source**:
- Enables a disabled package source
- Requires `--name` flag for source name
- Supports `--configfile` to specify config file

**disable source**:
- Disables a package source without removing it
- Requires `--name` flag for source name
- Supports `--configfile` to specify config file

**dotnet nuget Parity**:
- Matches `dotnet nuget list source`
- Matches `dotnet nuget add source`
- Matches `dotnet nuget remove source`
- Matches `dotnet nuget update source`
- Matches `dotnet nuget enable source`
- Matches `dotnet nuget disable source`

**Command Structure**:
- Pattern: `<verb> <noun>` (e.g., `add source`, not `sources add`)
- Flag naming: kebab-case (e.g., `--configfile`, `--name`, `--source`)

**Priority**: P0
**Dependencies**: configuration

---

#### CR-13: trusted-signers

**Synopsis**: `gonuget trusted-signers <action> [options]`

**Requirements**:
- Actions: list, add, remove, sync
- Manages trusted signers configuration
- Supports author signers
- Supports repository signers
- Stores in NuGet.config
- Validates certificate fingerprints

**Priority**: P2
**Dependencies**: packaging/signatures, configuration

---

#### CR-14: verify

**Synopsis**: `gonuget verify <package> [options]`

**Requirements**:
- Verifies package integrity
- Validates ZIP structure
- Verifies PKCS#7 signature
- Validates certificate chain
- Checks timestamp validity
- Supports `--certificate-fingerprint` to match expected
- Reports detailed results

**Priority**: P2
**Dependencies**: packaging/signatures

---

## Platform Requirements

### PR-1: Windows Support

**Requirements**:
- Compile native Windows executable (PE format)
- Support Windows 10+ (64-bit)
- Integrate with Windows Certificate Store
- Integrate with Windows Credential Manager
- **MSBuild Integration** (REQUIRED for 100% parity):
  - Discover MSBuild installations via Visual Studio Setup API
  - Support MSBuild 14.0, 15.0, 16.0, 17.0+
  - Invoke MSBuild for project building
  - Parse MSBuild project files
  - Extract MSBuild properties and metadata
- Support Windows paths (backslashes, drive letters)
- Support long paths (\\?\)
- Handle Windows line endings (CRLF)

**Distribution**:
- Standalone .exe
- Chocolatey package
- Windows installer (MSI or NSIS)
- Scoop manifest

**Acceptance Criteria**:
- [ ] Runs on Windows 10, 11, Server 2019, 2022
- [ ] Certificate store integration functional
- [ ] Credential Manager integration functional
- [ ] Passes all tests on Windows

---

### PR-2: Linux Support

**Requirements**:
- Compile native Linux executable (ELF format)
- Support major distributions (Ubuntu, Debian, RHEL, Fedora, Arch)
- Integrate with Secret Service API (libsecret)
- Support PEM certificate files
- Support Linux paths (forward slashes)
- Handle Unix line endings (LF)

**Distribution**:
- Standalone binary
- .deb package (Debian/Ubuntu)
- .rpm package (RHEL/Fedora)
- Snap package
- Flatpak
- AUR package (Arch)

**Acceptance Criteria**:
- [ ] Runs on Ubuntu 20.04+, Debian 11+, RHEL 8+
- [ ] Secret Service integration functional
- [ ] Passes all tests on Linux

---

### PR-3: macOS Support

**Requirements**:
- Compile native macOS executable (Mach-O format)
- Support macOS 11+ (Intel and Apple Silicon)
- Integrate with macOS Keychain
- Support certificate from Keychain
- Support macOS paths
- Handle Unix line endings (LF)

**Distribution**:
- Standalone binary
- Homebrew formula
- DMG installer
- Signed and notarized binary

**Acceptance Criteria**:
- [ ] Runs on macOS 11+, both Intel and Apple Silicon
- [ ] Keychain integration functional
- [ ] Passes all tests on macOS
- [ ] Binary signed and notarized

---

## Integration Requirements

### IR-1: NuGet.config Compatibility

**Requirement**: gonuget SHALL read and write NuGet.config files with 100% compatibility with nuget.exe and dotnet CLI.

**Details**:
- Identical XML structure
- Same configuration hierarchy
- Same encryption for sensitive values
- Same default values

**Test Strategy**: Round-trip test (gonuget → nuget.exe → gonuget)

**Acceptance Criteria**:
- [ ] Read NuGet.config created by nuget.exe
- [ ] Write NuGet.config readable by nuget.exe
- [ ] Round-trip preserves all values

---

### IR-2: Lock File Compatibility

**Requirement**: gonuget SHALL generate packages.lock.json files compatible with dotnet CLI.

**Details**:
- Identical JSON structure
- Same dependency resolution results
- Same content hashes

**Test Strategy**: Compare lock files for identical projects

**Acceptance Criteria**:
- [ ] Lock files generated by gonuget work with dotnet restore
- [ ] Lock files generated by dotnet work with gonuget restore

---

### IR-3: Credential Provider Compatibility

**Requirement**: gonuget SHALL support existing NuGet credential providers without modification.

**Details**:
- Same discovery mechanism
- Same protocol (stdin/stdout JSON)
- Same environment variables
- Same exit codes

**Test Providers**:
- CredentialProvider.Microsoft.exe (Azure Artifacts)
- AWS CodeArtifact credential provider
- JFrog CLI credential provider

**Acceptance Criteria**:
- [ ] Azure Artifacts authentication works
- [ ] AWS CodeArtifact authentication works
- [ ] Custom providers work

---

### IR-4: CI/CD Integration

**Requirement**: gonuget SHALL work as drop-in replacement in CI/CD pipelines.

**Supported CI/CD Systems**:
- Azure Pipelines
- GitHub Actions
- GitLab CI
- Jenkins
- CircleCI
- Travis CI

**Requirements**:
- Identical exit codes
- Machine-readable output (JSON, GitHub Actions format)
- Non-interactive mode
- Environment variable support

**Acceptance Criteria**:
- [ ] Works in Azure Pipelines
- [ ] Works in GitHub Actions
- [ ] Works in GitLab CI
- [ ] JSON output parseable

---

## Performance Requirements

### Detailed Performance Targets

**Startup Time** (Cold Start, P50):
- gonuget: < 50ms
- nuget.exe baseline: ~50-100ms
- Target: 10x improvement

**Command Execution Time** (P50, Network Operations):
- `search json`: < 1s
- `install Newtonsoft.Json` (cold cache): < 5s
- `restore` (10 packages, cold cache): < 15s
- `pack` (typical project): < 2s
- `push` (1MB package): < 5s (network-dependent)

**Resource Usage**:
- Memory (peak): < 100MB for typical operations
- Memory (typical): < 50MB
- CPU: < 50% single core (except parallel downloads)
- Disk I/O: Optimized with buffering

**Scalability**:
- Handle solutions with 100+ projects
- Handle 1000+ packages in feed
- Handle packages up to 500MB

**Acceptance Criteria**:
- [ ] All performance targets met in benchmark suite
- [ ] No performance regressions between versions
- [ ] Memory profiling clean (no leaks)

---

## Security Requirements

### Detailed Security Controls

**SR-1: Input Validation**

**Requirement**: ALL user inputs SHALL be validated before processing.

**Validation Rules**:
- Package IDs: alphanumeric, dots, dashes, underscores only
- Versions: valid SemVer or legacy version format
- URLs: valid HTTP/HTTPS URLs, reject file:// and other schemes
- File paths: sanitize, prevent directory traversal
- Command arguments: reject malicious patterns

**Acceptance Criteria**:
- [ ] All inputs validated
- [ ] Directory traversal attacks prevented
- [ ] Command injection prevented

---

**SR-2: Package Signature Verification**

**Requirement**: gonuget SHALL verify package signatures when signature validation is enabled.

**Verification Steps**:
1. Validate PKCS#7 signature structure
2. Verify certificate chain to trusted root
3. Check certificate purpose (code signing)
4. Validate timestamp (within certificate validity)
5. Verify content hash matches

**Modes**:
- `accept` (default): Accept signed and unsigned packages
- `require`: Reject unsigned packages

**Acceptance Criteria**:
- [ ] Signature verification functional
- [ ] Invalid signatures rejected
- [ ] Trusted signers respected

---

**SR-3: Credential Protection**

**Requirement**: Credentials SHALL be protected using OS-specific secure storage.

**Storage Mechanisms**:
- Windows: DPAPI (Data Protection API)
- macOS: Keychain
- Linux: Secret Service API (GNOME Keyring, KWallet)
- Fallback: Encrypted file with user-specific key

**Protection Rules**:
- Never log credentials
- Never include credentials in error messages
- Zero memory after use
- Encrypt in NuGet.config

**Acceptance Criteria**:
- [ ] Credentials stored securely on all platforms
- [ ] No credentials in logs
- [ ] No credentials in error messages
- [ ] Memory zeroing verified

---

**SR-4: Network Security**

**Requirement**: All network communication SHALL use secure protocols and validate certificates.

**TLS Requirements**:
- Require TLS 1.2+ by default
- Validate server certificates
- Support certificate pinning
- Support custom CA certificates

**HTTP Requirements**:
- Warn on HTTP (non-HTTPS) feeds
- Require explicit opt-in for HTTP with `--allow-insecure`
- User-Agent header identifies client
- Timeout protection

**Acceptance Criteria**:
- [ ] TLS 1.2+ enforced
- [ ] Certificate validation functional
- [ ] HTTP feeds require opt-in
- [ ] User-Agent header correct

---

## Usability Requirements

### UR-1: Error Messages

**Requirement**: Error messages SHALL be clear, actionable, and user-friendly.

**Error Message Format**:
```
Error: Package 'NonExistentPackage' not found in any source.

Searched sources:
  • https://api.nuget.org/v3/index.json (404)
  • https://private.feed.com/v3/index.json (404)

Suggestions:
  - Check package ID spelling
  - Verify package exists: gonuget search NonExistentPackage
  - Add additional sources: gonuget sources add --help
```

**Requirements**:
- Start with "Error:" prefix
- Include relevant context (package ID, version, source)
- Provide actionable suggestions
- No stack traces in normal mode
- Include error code for documentation lookup

**Acceptance Criteria**:
- [ ] All error messages follow format
- [ ] User testing validates clarity
- [ ] Error codes documented

---

### UR-2: Help System

**Requirement**: Help system SHALL provide comprehensive usage information.

**Help Levels**:
1. **Command List**: `gonuget help` or `gonuget --help`
2. **Command Help**: `gonuget help <command>` or `gonuget <command> --help`
3. **Verbose Help**: `gonuget help <command> --verbose`

**Help Content**:
- Synopsis with syntax
- Description
- Options with descriptions
- Examples (at least 2 per command)
- Related commands

**Acceptance Criteria**:
- [ ] Help for all commands
- [ ] Examples for all commands
- [ ] Help text reviewed by technical writer

---

### UR-3: Progress Indicators

**Requirement**: Long-running operations SHALL show progress.

**Indicators**:
- **Determinate**: Progress bar with percentage (downloads)
- **Indeterminate**: Spinner (resolution, search)
- **Multi-Operation**: Multiple progress bars (parallel downloads)

**Requirements**:
- Update at least every 100ms
- Show current operation
- Show ETA for downloads
- Show transfer speed
- Clear on completion

**Acceptance Criteria**:
- [ ] Progress bars functional
- [ ] Performance not degraded
- [ ] Works in all terminal types

---

### UR-4: Interactive Prompts

**Requirement**: Prompts SHALL be clear and support both interactive and non-interactive modes.

**Prompt Types**:
- Confirmation (yes/no)
- Credential input (username/password)
- Choice selection

**Non-Interactive Mode**:
- Use defaults or fail fast
- No prompts in CI/CD
- `--non-interactive` flag disables all prompts

**Acceptance Criteria**:
- [ ] Prompts clear and intuitive
- [ ] Non-interactive mode works
- [ ] Credential prompts secure (password hidden)

---

### UR-5: Localization

**Requirement**: gonuget SHALL support all languages supported by nuget.exe for 100% parity.

**Supported Languages** (14 total):
1. English (en) - Primary
2. Czech (cs)
3. German (de)
4. Spanish (es)
5. French (fr)
6. Italian (it)
7. Japanese (ja)
8. Korean (ko)
9. Polish (pl)
10. Brazilian Portuguese (pt-BR)
11. Russian (ru)
12. Turkish (tr)
13. Simplified Chinese (zh-Hans)
14. Traditional Chinese (zh-Hant)

**Localized Content**:
- All command help text
- All error messages
- All user-facing strings
- Date and number formatting per locale
- Progress indicators and status messages

**Locale Detection**:
- Respect system locale (LC_ALL, LANG environment variables)
- Support `--locale` flag to override
- Fallback to English if translation missing

**Implementation**:
- Use XLIFF (.xlf) format matching nuget.exe
- String extraction and management workflow
- Translation verification tests

**Acceptance Criteria**:
- [ ] All 14 languages fully supported
- [ ] Locale auto-detection functional
- [ ] All user-facing strings localized
- [ ] Translation completeness: 100% for all languages
- [ ] No English fallback strings in non-English locales (except for untranslated technical terms)

---

## Compatibility Requirements

### CR-1: Command Compatibility

**Requirement**: gonuget commands SHALL be 100% compatible with nuget.exe commands.

**Compatibility Matrix**:

| Command | gonuget v1.0 | dotnet nuget | nuget.exe 6.x | Parity Status |
|---------|--------------|--------------|---------------|---------------|
| help | ✓ | ✓ | ✓ | Required |
| version | ✓ | ✓ | ✓ | Required |
| config get | ✓ | ✓ | ✗ (single cmd) | Required |
| config set | ✓ | ✓ | ✗ (single cmd) | Required |
| config list | ✓ | ✓ | ✗ (single cmd) | Required |
| list source | ✓ | ✓ | ✗ (sources list) | Required |
| add source | ✓ | ✓ | ✗ (sources add) | Required |
| remove source | ✓ | ✓ | ✗ (sources remove) | Required |
| update source | ✓ | ✓ | ✗ (sources update) | Required |
| enable source | ✓ | ✓ | ✗ (sources enable) | Required |
| disable source | ✓ | ✓ | ✗ (sources disable) | Required |
| search | ✓ | ✓ | ✓ | Required |
| install | ✓ | ✗ | ✓ | Required |
| restore | ✓ | ✓ | ✓ | Required |
| pack | ✓ | ✓ | ✓ | Required |
| push | ✓ | ✓ | ✓ | Required |
| delete | ✓ | ✓ | ✓ | Required |
| add | ✓ | ✓ | ✓ | Required |
| init | ✓ | ✗ | ✓ | Required |
| list | ✓ (delegates) | ✗ | ✓ (deprecated) | Required |
| locals | ✓ | ✓ | ✓ | Required |
| setapikey | ✓ | ✗ | ✓ | Required |
| update | ✓ | ✗ | ✓ | Required |
| sign | ✓ | ✓ | ✓ | Required |
| verify | ✓ | ✓ | ✓ | Required |
| trusted-signers | ✓ | ✓ | ✓ | Required |
| client-certs | ✓ | ✗ | ✓ | Required |
| spec | ✓ | ✗ | ✓ | Required |
| mirror | ✗ | ✗ | ✗ (deprecated) | Out of Scope |

**Notes**:
- **Primary Target**: `dotnet nuget` commands (cross-platform)
- **Secondary Target**: `nuget.exe` compatibility (Windows legacy)
- ✗ in dotnet nuget column indicates command not available in `dotnet nuget`, but may be in `nuget.exe`
- Command structure follows `dotnet nuget` pattern: `<verb> <noun>` with kebab-case flags

**Acceptance Criteria**:
- [ ] All commands listed as "Required" implemented
- [ ] Interop tests passing for all commands
- [ ] Behavior identical to nuget.exe

---

### CR-2: Configuration Compatibility

**Requirement**: NuGet.config files SHALL be interchangeable between gonuget and nuget.exe.

**Test Scenarios**:
1. Create config with gonuget, read with nuget.exe
2. Create config with nuget.exe, read with gonuget
3. Round-trip (gonuget → nuget.exe → gonuget)

**Acceptance Criteria**:
- [ ] All test scenarios pass
- [ ] No data loss in round-trip
- [ ] Encrypted values preserved

---

### CR-3: Package Compatibility

**Requirement**: Packages created by gonuget SHALL be readable by nuget.exe and vice versa.

**Test Scenarios**:
1. Pack with gonuget, install with nuget.exe
2. Pack with nuget.exe, install with gonuget
3. Sign with gonuget, verify with nuget.exe
4. Sign with nuget.exe, verify with gonuget

**Acceptance Criteria**:
- [ ] All test scenarios pass
- [ ] OPC compliance verified
- [ ] Signatures interoperable

---

## Success Metrics

### Primary Metrics

**Adoption**:
- 10,000+ downloads in first 6 months
- 100+ GitHub stars in first 3 months
- 10+ external contributors

**Quality**:
- 100% of interop tests passing
- < 0.01% crash rate
- < 5% support request rate (clear documentation)

**Performance**:
- Startup time < 50ms (P50)
- 1.5x faster package restore vs nuget.exe
- 10x faster startup vs nuget.exe

**User Satisfaction**:
- 95% satisfaction rating (survey)
- 80% would recommend to colleague
- 4.5+ star rating on package managers

### Secondary Metrics

**Community**:
- 50+ issues filed (engagement)
- 20+ pull requests merged
- 5+ blog posts/articles mentioning gonuget

**Enterprise**:
- 10+ enterprise users (Fortune 500)
- 5+ case studies
- Featured in .NET newsletter

---

## Dependencies and Constraints

### Dependencies

**Internal**:
- gonuget library (M1-M8 complete)
- Version package
- Frameworks package
- Packaging package
- Protocol packages (V2, V3)
- Resolver package
- Cache package
- Auth package

**External**:
- Go 1.25.2+
- cobra (CLI framework)
- viper (configuration)
- go-keyring (credential storage)
- tablewriter (output formatting)
- progressbar (progress indicators)
- color (colored output)

**Platform**:
- Windows: DPAPI, Certificate Store API
- macOS: Keychain, Security.framework
- Linux: libsecret

### Constraints

**Technical**:
- Single binary (no external dependencies)
- No .NET runtime required
- Pure Go implementation
- Maximum binary size: 50MB

**Business**:
- Open source (MIT license)
- No telemetry without explicit opt-in
- No breaking changes from nuget.exe
- Maintain backward compatibility

**Timeline**:
- 16 weeks to v1.0 feature complete
- 4 weeks beta period
- 2 weeks release candidate
- Total: 22 weeks to GA

---

## Acceptance Criteria

### Phase 1: Foundation (Weeks 1-2)

- [ ] CLI framework (cobra) integrated
- [ ] Configuration loading functional (NuGet.config XML)
- [ ] Console output with colors and progress
- [ ] Commands: help, version
- [ ] Config subcommands: get, set, list
- [ ] Source commands: list source, add source, remove source, update source, enable source, disable source
- [ ] **Total Commands**: 9/21 (43%)
- [ ] Command structure: `<verb> <noun>` pattern
- [ ] Flag naming: kebab-case (--configfile, --name, --source)
- [ ] Unit tests for config and CLI parsing
- [ ] CLI interop tests comparing with `dotnet nuget` commands
- [ ] 80%+ test coverage

### Phase 2: Core Operations (Weeks 3-5)

- [ ] Commands: search, install, list
- [ ] Package metadata fetching (V2, V3)
- [ ] Package download with progress
- [ ] Simple extraction (no dependency resolution)
- [ ] Cache integration
- [ ] Integration tests with test feed
- [ ] 85%+ test coverage

### Phase 3: Dependency Resolution (Weeks 6-7)

- [ ] Commands: restore
- [ ] Dependency graph resolution
- [ ] Version conflict detection
- [ ] packages.config support
- [ ] Lock file generation
- [ ] Complex dependency tests
- [ ] 90%+ test coverage

### Phase 4: Package Creation (Weeks 8-9)

- [ ] Commands: pack, push, spec
- [ ] .nuspec parsing and generation
- [ ] **MSBuild Integration** (REQUIRED):
  - [ ] MSBuild project file parsing (.csproj, .vbproj, .fsproj)
  - [ ] MSBuild discovery and invocation
  - [ ] MSBuild property extraction and substitution
  - [ ] `-Build` flag support
  - [ ] `-IncludeReferencedProjects` support
  - [ ] `-MSBuildPath` and `-MSBuildVersion` flags
- [ ] OPC-compliant package creation
- [ ] File pattern matching
- [ ] Metadata extraction
- [ ] Package creation validation tests
- [ ] 90%+ test coverage

### Phase 5: Signing & Security (Weeks 10-11)

- [ ] Commands: sign, verify, trusted-signers, client-certs
- [ ] PKCS#7 signature creation/validation
- [ ] RFC 3161 timestamping
- [ ] Certificate store integration (all platforms)
- [ ] Trust configuration
- [ ] Signature validation tests
- [ ] 90%+ test coverage

### Phase 6: Advanced Features (Weeks 12-13)

- [ ] Commands: update, locals, add, init, delete, setapikey
- [ ] Self-update mechanism
- [ ] Cache management
- [ ] API key storage
- [ ] Full command suite tests
- [ ] 90%+ test coverage

### Phase 7: Polish & Optimization (Weeks 14-15)

- [ ] Performance optimization (profiling, benchmarks)
- [ ] Error messages improved
- [ ] **Localization** (REQUIRED):
  - [ ] XLIFF (.xlf) infrastructure
  - [ ] All 14 languages implemented (cs, de, es, fr, it, ja, ko, pl, pt-BR, ru, tr, zh-Hans, zh-Hant)
  - [ ] Locale detection and switching
  - [ ] Translation completeness verification
- [ ] Shell completions (bash, zsh, fish, PowerShell)
- [ ] Man pages
- [ ] Documentation complete
- [ ] Performance benchmarks meet targets
- [ ] Interop parity tests 100% passing

### Phase 8: Platform-Specific (Week 16)

- [ ] Windows installer (MSI/Chocolatey)
- [ ] Windows-specific MSBuild integration complete
- [ ] macOS installer (Homebrew)
- [ ] Linux packages (deb, rpm, snap)
- [ ] Windows Credential Manager integration
- [ ] macOS Keychain integration
- [ ] Linux Secret Service integration
- [ ] Platform-specific tests on CI
- [ ] Binaries signed (macOS notarized)

### Release Criteria

**v1.0 GA**:
- [ ] All acceptance criteria met
- [ ] 100% of interop tests passing
- [ ] Zero P0/P1 bugs
- [ ] Documentation complete
- [ ] Security audit passed
- [ ] Performance benchmarks met
- [ ] All platforms supported
- [ ] Release notes finalized

---

## Out of Scope

### Explicitly Out of Scope

**For v1.0**:
1. **mirror command** - Deprecated in nuget.exe 3.2+
2. **install.ps1 script execution** - Security risk, Package Manager Console feature (Visual Studio only), not part of nuget.exe CLI
3. **Visual Studio integration** - IDE plugin in future release
4. **GUI** - CLI only for v1.0
5. **Plugin system for commands** - Only credential providers in v1.0 (matches nuget.exe)

**Deferred to Future Releases**:
- NuGet v4 protocol (if/when released)
- Package curation/moderation features
- Advanced analytics and reporting
- Web UI for package browsing

**IMPORTANT NOTE**: MSBuild integration and localization (13 languages) are REQUIRED for v1.0, NOT out of scope. These are core features of nuget.exe and must be included for 100% parity.

---

## Risks and Mitigation

### Technical Risks

**Risk**: NuGet protocol changes break compatibility
**Likelihood**: Medium
**Impact**: High
**Mitigation**:
- Monitor NuGet.Client releases
- Automated tests against live feeds
- Version detection and fallback
- Community early warning system

---

**Risk**: Platform-specific certificate store differences
**Likelihood**: High
**Impact**: Medium
**Mitigation**:
- Abstract certificate store interface
- Platform-specific implementations
- Extensive platform testing
- Fallback to PEM files

---

**Risk**: Credential provider incompatibilities
**Likelihood**: Medium
**Impact**: High
**Mitigation**:
- Test with major providers early
- Document protocol precisely
- Provide reference implementation
- Work with provider maintainers

---

**Risk**: Performance targets not met
**Likelihood**: Low
**Impact**: Medium
**Mitigation**:
- Profile early and often
- Benchmark against baselines
- Optimize hot paths
- Accept some targets may be network-bound

---

### Business Risks

**Risk**: Low adoption due to ecosystem lock-in
**Likelihood**: Medium
**Impact**: High
**Mitigation**:
- Focus on Go developers first (clear value)
- Demonstrate CI/CD performance wins
- Partner with package feed providers
- Active community building

---

**Risk**: nuget.exe introduces breaking changes
**Likelihood**: Low
**Impact**: High
**Mitigation**:
- Version compatibility matrix
- Explicit version support policy
- Rapid response to NuGet updates
- Communication with NuGet team

---

## Appendix

### A. Glossary

**Terms**:
- **NuGet**: .NET package manager
- **nuget.exe**: Official NuGet command-line tool
- **gonuget**: This project (Go implementation of NuGet CLI)
- **V2/V3**: NuGet protocol versions
- **OPC**: Open Packaging Conventions (ZIP-based package format)
- **PKCS#7**: Cryptographic signature format
- **RFC 3161**: Timestamping protocol
- **TFM**: Target Framework Moniker (e.g., net8.0, netstandard2.0)
- **packages.config**: Legacy NuGet dependency format (XML)
- **PackageReference**: Modern NuGet dependency format (MSBuild)
- **Credential Provider**: External executable providing authentication

---

### B. References

**NuGet Documentation**:
- [NuGet CLI Reference](https://learn.microsoft.com/nuget/reference/nuget-exe-cli-reference)
- [dotnet nuget commands](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-nuget)
- [NuGet V3 Protocol](https://learn.microsoft.com/nuget/api/overview)
- [NuGet Package Versioning](https://learn.microsoft.com/nuget/concepts/package-versioning)
- [NuGet Package Signing](https://learn.microsoft.com/nuget/create-packages/sign-a-package)
- [NuGet Credential Providers](https://learn.microsoft.com/nuget/reference/extensibility/nuget-credential-providers-for-visual-studio)

**Standards**:
- [SemVer 2.0](https://semver.org/)
- [OPC Specification](https://www.ecma-international.org/publications-and-standards/standards/ecma-376/)
- [PKCS#7 / CMS](https://tools.ietf.org/html/rfc5652)
- [RFC 3161 Timestamping](https://tools.ietf.org/html/rfc3161)

**Reference Implementations**:
- [dotnet/sdk](https://github.com/dotnet/sdk) - Official .NET SDK with `dotnet nuget` commands (PRIMARY REFERENCE)
- [NuGet.Client](https://github.com/NuGet/NuGet.Client) - nuget.exe implementation
- [dotnet/docs](https://github.com/dotnet/docs) - Official NuGet documentation

**Related Projects**:
- [dotnet CLI](https://github.com/dotnet/cli) - .NET CLI with NuGet integration
- [gonuget library](../implementation/) - Go NuGet library

---

### C. Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-25 | - | Initial PRD |
| 1.1 | 2025-10-25 | - | Updated to target `dotnet nuget` compatibility in addition to `nuget.exe`; added dotnet/sdk and dotnet/docs as reference implementations |

---

**Document Status**: Draft
**Next Review**: After design approval
**Approval Required**: Project Lead, Technical Lead

---

**END OF DOCUMENT**
