# Feature Specification: CLI Command Structure Restructure

**Feature Branch**: `001-cli-command-restructure`
**Created**: 2025-10-31
**Status**: Draft
**Input**: Restructure gonuget CLI to adopt modern noun-first command hierarchy matching dotnet CLI standards

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Package Management with Noun-First Commands (Priority: P1)

As a developer using gonuget, I want to manage packages using intuitive noun-first commands (`gonuget package add`, `gonuget package list`) so that the command structure feels consistent with modern CLI tools I already use (kubectl, docker, aws).

**Why this priority**: This is the core user interaction pattern that will be used most frequently. Getting package commands right is essential for user adoption and satisfaction.

**Independent Test**: Can be fully tested by executing package add, list, remove, and search commands and verifying they work with the new noun-first structure while old verb-first forms are rejected with helpful error messages.

**Acceptance Scenarios**:

1. **Given** I want to add a package, **When** I run `gonuget package add Newtonsoft.Json`, **Then** the package is added to my project successfully
2. **Given** I want to list packages, **When** I run `gonuget package list`, **Then** I see all referenced packages displayed
3. **Given** I try the old syntax, **When** I run `gonuget add package Newtonsoft.Json`, **Then** I see an error message suggesting `gonuget package add` instead
4. **Given** I want to search for packages, **When** I run `gonuget package search Serilog`, **Then** I see search results from configured sources
5. **Given** I want to remove a package, **When** I run `gonuget package remove Newtonsoft.Json`, **Then** the package is removed from my project

---

### User Story 2 - Source Management with Noun-First Commands (Priority: P1)

As a developer managing package sources, I want to configure sources using noun-first commands (`gonuget source add`, `gonuget source list`) so that source operations feel natural and consistent with the package namespace.

**Why this priority**: Source configuration is a critical setup task that must work flawlessly. Users need confidence that source management follows the same patterns as package management.

**Independent Test**: Can be fully tested by executing source add, list, remove, enable, disable, and update commands with the new noun-first structure and verifying NuGet.config is modified correctly.

**Acceptance Scenarios**:

1. **Given** I want to add a source, **When** I run `gonuget source add https://api.nuget.org/v3/index.json --name nuget.org`, **Then** the source is added to my NuGet.config
2. **Given** I want to list sources, **When** I run `gonuget source list`, **Then** I see all configured sources with their status
3. **Given** I try the old syntax, **When** I run `gonuget add source https://example.com`, **Then** I see an error message suggesting `gonuget source add` instead
4. **Given** I have a disabled source, **When** I run `gonuget source enable nuget.org`, **Then** the source becomes enabled in NuGet.config
5. **Given** I want to remove a source, **When** I run `gonuget source remove nuget.org`, **Then** the source is removed from NuGet.config

---

### User Story 3 - Helpful Error Messages for Migration (Priority: P2)

As a developer familiar with the old gonuget command structure, I want clear error messages when I use the old syntax so that I can quickly learn the new noun-first pattern without frustration.

**Why this priority**: This reduces migration friction and improves user experience during the transition. Users should feel guided, not punished, when using old syntax.

**Independent Test**: Can be fully tested by attempting all old verb-first commands and verifying each produces a specific, helpful error message with the correct new syntax.

**Acceptance Scenarios**:

1. **Given** I type an old package command, **When** I run `gonuget add package Serilog`, **Then** I see "Error: the verb-first form is not supported. Try: gonuget package add"
2. **Given** I type an old source command, **When** I run `gonuget list source`, **Then** I see "Error: the verb-first form is not supported. Try: gonuget source list"
3. **Given** I make a typo, **When** I run `gonuget pakage add`, **Then** I see suggestions for similar commands (package)
4. **Given** I want help, **When** I run `gonuget help`, **Then** I see only the new noun-first commands listed (package, source, restore, config, version)

---

### User Story 4 - JSON Output for Automation (Priority: P2)

As a developer writing automation scripts, I want structured JSON output from list and search commands so that I can reliably parse results without fragile text parsing.

**Why this priority**: JSON output enables reliable automation and integration with other tools. This is a key differentiator for modern CLI tools.

**Independent Test**: Can be fully tested by running commands with `--format json` and validating the JSON schema, field presence, and data types.

**Acceptance Scenarios**:

1. **Given** I want JSON output, **When** I run `gonuget package list --format json`, **Then** I receive valid JSON with schemaVersion, packages array, and metadata
2. **Given** I want JSON search results, **When** I run `gonuget package search Serilog --format json`, **Then** I receive valid JSON with search results array
3. **Given** JSON output is enabled, **When** there are warnings, **Then** warnings appear in stderr, not mixed with JSON stdout
4. **Given** I want consistent JSON, **When** I use `--format json` on any list/search command, **Then** all outputs include a schemaVersion field

---

### User Story 5 - Shell Completion for Productivity (Priority: P3)

As a developer using the command line frequently, I want shell completion for gonuget commands so that I can work faster and make fewer typos.

**Why this priority**: Shell completion is a nice-to-have that improves daily productivity for frequent users, but is not essential for basic functionality.

**Independent Test**: Can be fully tested by loading shell completion scripts and verifying TAB completion works for command namespaces, verbs, source names, and project paths.

**Acceptance Scenarios**:

1. **Given** shell completion is enabled, **When** I type `gonuget <TAB>`, **Then** I see suggestions: config, package, restore, source, version
2. **Given** shell completion is enabled, **When** I type `gonuget package <TAB>`, **Then** I see suggestions: add, list, remove, search
3. **Given** shell completion is enabled, **When** I type `gonuget source <TAB>`, **Then** I see suggestions: add, disable, enable, list, remove, update
4. **Given** I'm adding a source, **When** I type `gonuget source remove <TAB>`, **Then** I see source names from my NuGet.config

---

### Edge Cases

- What happens when a user runs a top-level command that doesn't exist (e.g., `gonuget unknown`)? → Exit code 2 with "unknown command" error
- How does the system handle ambiguous typos (e.g., `gonuget pak` could be package or pack)? → Show suggestions sorted by edit distance
- What happens if there are no configured sources when running `gonuget source list`? → Exit code 0 with message "No sources configured"
- How does JSON output handle empty results (e.g., package search with zero matches)? → Exit code 0 with `{"items": [], "total": 0}`
- What happens when running commands without proper permissions to modify NuGet.config? → Exit code 1 with clear permission error message
- How are multi-line help texts displayed for complex commands? → Follow standard terminal width wrapping conventions
- What happens when TAB completion is requested in a non-supported shell? → Gracefully fall back with instructions for supported shells

## Requirements *(mandatory)*

### Functional Requirements

#### Command Structure

- **FR-001**: System MUST implement `gonuget package` as a parent command with subcommands: add, list, remove, search
- **FR-002**: System MUST implement `gonuget source` as a parent command with subcommands: add, list, remove, enable, disable, update
- **FR-003**: System MUST support top-level commands: restore, config, version (matching dotnet CLI exactly)
- **FR-004**: System MUST NOT support verb-first aliases (no `gonuget add package`, `gonuget add source`, etc.)
- **FR-005**: Subcommand Use fields MUST be verb-only (e.g., `Use: "add"` not `Use: "add package"`)
- **FR-006**: NO command MUST have Aliases fields set (hard policy: zero tolerance for aliases)

#### Error Handling

- **FR-007**: System MUST detect verb-first command patterns and show helpful error messages with correct noun-first syntax
- **FR-008**: Error messages MUST suggest the correct noun-first form (e.g., "Try: gonuget package add")
- **FR-009**: System MUST use exit code 0 for success, 1 for generic errors, 2 for invalid arguments, 3 for not found, 4 for network failures
- **FR-010**: System MUST implement `SuggestionsMinimumDistance` for typo suggestions
- **FR-011**: System MUST set `SilenceErrors = true` and use custom error handler to detect verb-first patterns before showing errors

#### Flag Consistency

- **FR-012**: System MUST support `--format` flag with values: console, json (on list/search/config commands)
- **FR-013**: System MUST support `--verbosity` flag with values: quiet, minimal, normal, detailed, diagnostic (on all commands)
- **FR-014**: System MUST support `--configfile` flag for NuGet.config path (on source/config/restore commands)
- **FR-015**: System MUST support `--what-if` / `--dry-run` flag for showing planned changes (on mutating commands)
- **FR-016**: System MUST support `--yes` / `-y` flag for suppressing confirmations (on mutating commands)
- **FR-017**: System MUST use kebab-case for all flag names (--include-transitive, not --includeTransitive)

#### JSON Output

- **FR-018**: When `--format json` is specified, system MUST output only valid JSON to stdout
- **FR-019**: When `--format json` is specified, system MUST send all human-readable text to stderr
- **FR-020**: All JSON output MUST include a `schemaVersion` field
- **FR-021**: Package list JSON MUST include: schemaVersion, project, framework, packages, warnings, elapsedMs
- **FR-022**: Package search JSON MUST include: schemaVersion, searchTerm, sources, items, total, elapsedMs
- **FR-023**: Empty search results MUST return exit code 0 with `{"items": [], "total": 0}`

#### Help Text

- **FR-024**: All commands MUST have clear Short and Long descriptions
- **FR-025**: Short descriptions MUST start with verbs (e.g., "Add a package reference...")
- **FR-026**: Examples MUST show both minimal and fully-flagged command forms
- **FR-027**: Top-level help MUST show commands in logical order
- **FR-028**: Help text MUST document top-level exceptions (restore, config, version) with rationale

#### Shell Completion

- **FR-029**: System MUST provide namespace completion (gonuget <TAB> → config package restore source version)
- **FR-030**: System MUST provide verb completion (gonuget package <TAB> → add list remove search)
- **FR-031**: System MUST provide dynamic completion for source names from NuGet.config
- **FR-032**: System MUST support completion script generation for bash, zsh, PowerShell

#### Configuration

- **FR-033**: System MUST follow configuration precedence: CLI flags > env vars > config files > defaults
- **FR-034**: System MUST support `GONUGET_*` prefixed environment variables
- **FR-035**: System MUST read `NUGET_PACKAGES` environment variable for parity (read-only)

### Key Entities *(include if feature involves data)*

- **PackageReference**: Represents a package dependency with attributes: ID, version, framework, type (direct/transitive)
- **PackageSource**: Represents a NuGet source with attributes: name, URL, enabled status, credentials
- **Command**: Represents a CLI command with attributes: Use field (verb-only), parent command, flags, subcommands
- **JSONSchema**: Versioned output schema with attributes: schemaVersion, data fields, metadata (warnings, elapsedMs)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users successfully execute package add/list/remove/search commands using noun-first syntax with 100% success rate
- **SC-002**: Users attempting old verb-first syntax receive helpful error messages in <50ms with correct alternative syntax
- **SC-003**: Help command shows exactly 5 top-level commands (config, package, restore, source, version) with no verb-first patterns visible
- **SC-004**: JSON output validates against documented schemas with 100% compliance (all outputs include schemaVersion)
- **SC-005**: Shell completion suggests correct commands for all namespaces in <100ms
- **SC-006**: All tests pass including golden tests for help output, JSON schemas, and exit codes
- **SC-007**: Zero aliases are registered for any command (validated via reflection test)
- **SC-008**: Configuration precedence correctly resolves values in order: flags > env > config > defaults in 100% of test cases
- **SC-009**: Empty search results return exit code 0 with valid JSON schema 100% of the time
- **SC-010**: Verb-first error detection identifies all 9 verb-first patterns (add/list/remove package/source, enable/disable/update source) with helpful messages

### User Experience Outcomes

- **SC-011**: New users unfamiliar with gonuget can discover all package and source commands via `gonuget help` and TAB completion
- **SC-012**: Users migrating from old gonuget syntax learn new commands within 1-2 errors due to helpful error messages
- **SC-013**: Automation scripts reliably parse JSON output without fragile regex/text parsing
- **SC-014**: Command structure feels consistent with other modern CLI tools (docker, kubectl, aws) based on industry standards
- **SC-015**: Documentation clearly explains noun-first rationale and top-level exceptions without requiring technical implementation knowledge

## Scope *(mandatory)*

### In Scope

- Implementing parent commands (package.go, source.go) with proper subcommand registration
- Renaming command files to match new structure (package_add.go, source_list.go, etc.)
- Creating new package commands: list, search, remove
- Updating main.go to register parent commands and top-level commands (restore, config, version)
- Implementing custom error handler for verb-first pattern detection
- Adding standard flags (--format, --verbosity, --configfile, --what-if, --yes) consistently
- Implementing JSON output contracts for list and search commands
- Creating golden tests for help output, JSON schemas, and exit codes
- Implementing shell completion for bash, zsh, PowerShell
- Writing reflection tests to enforce verb-only Use fields and zero aliases policy
- Updating all existing tests to use new noun-first command structure
- Documenting command structure in README.md with clear examples

### Out of Scope

- Changing the behavior of existing command implementations (only changing structure, not functionality)
- Adding new features beyond command restructuring (e.g., new package operations, new source capabilities)
- Migrating user configurations or scripts automatically (users adopt new syntax manually)
- Supporting verb-first aliases or deprecated commands (hard policy: no aliases)
- Implementing telemetry or analytics (explicitly excluded per project policy)
- Modifying NuGet protocol implementations (V2/V3)
- Changing package resolution or dependency graph algorithms
- Adding new output formats beyond console and JSON
- Implementing interactive command modes
- Creating GUI or web-based management tools

## Dependencies *(optional)*

### Internal Dependencies

- Cobra CLI framework (existing dependency, no changes needed)
- Existing command implementations (add_package.go, source commands, etc.)
- gonuget core packages (version, frameworks, protocol, packaging, resolver)
- Output abstraction (cmd/gonuget/output package)

### External Dependencies

- NuGet.config files (existing standard format, no changes)
- .NET project files (.csproj, .fsproj, .vbproj) (existing standard format, no changes)
- Shell environments for completion (bash, zsh, PowerShell)

## Assumptions *(optional)*

- Users are familiar with modern CLI tools (docker, kubectl, aws) and understand noun-first command patterns
- Standard shells (bash, zsh, PowerShell) are available for completion script generation
- NuGet.config files follow standard XML schema as defined by Microsoft
- .NET project files follow standard MSBuild XML schema
- Performance targets match industry-standard CLI tools (command execution <100ms, help output <50ms)
- Error messages should be single-line, actionable, and free of jargon
- JSON schemas should be stable and versioned to prevent breaking changes for automation scripts
- Users expect dotnet CLI parity for familiar commands (restore, config, version)
- gonuget will never collect telemetry (per explicit project policy)
- Exit codes follow Unix conventions (0=success, 1=error, 2=invalid usage)
- Help text should be concise, scannable, and example-driven
- Shell completion should work with standard TAB key behavior
- Environment variables use `GONUGET_` prefix for tool-specific vars, `NUGET_` for ecosystem compatibility
- Configuration precedence follows standard CLI conventions (flags override everything)

## Open Questions *(optional - remove if none)*

None - all aspects of the command restructure are well-defined based on the comprehensive implementation guide provided. The feature is ready for planning and implementation.
