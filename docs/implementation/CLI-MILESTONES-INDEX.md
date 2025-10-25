# CLI Implementation Milestones - Index

**Project**: gonuget CLI
**Target**: 100% parity with `dotnet nuget` (cross-platform)
**Reference Implementation**: dotnet/sdk and NuGet.Client/NuGet.CommandLine.XPlat
**Total Duration**: 16 weeks (4 months)
**Prerequisites**: gonuget library M1-M8 complete
**Testing Approach**: CLI interop tests using JSON-RPC bridge (see [CLI-INTEROP-TESTING.md](./CLI-INTEROP-TESTING.md))

---

## Implementation Guide Documents

### Phase 1: Foundation (Weeks 1-2)
**Documents**:
- [CLI-M1-FOUNDATION.md](./CLI-M1-FOUNDATION.md) (Chunks 1-5, 1,706 lines)
- [CLI-M1-FOUNDATION-CONTINUED.md](./CLI-M1-FOUNDATION-CONTINUED.md) (Chunks 6-10, 1,800 lines)

- ✅ Chunk 1: Project Structure and Entry Point
- ✅ Chunk 2: Console Abstraction
- ✅ Chunk 3: Configuration Management (NuGet.config XML)
- ✅ Chunk 4: Version Command (`gonuget --version` matches `dotnet nuget --version`)
- ✅ Chunk 5: Config Command (`gonuget config` - get/set/list matches `dotnet nuget config`)
- ✅ Chunk 6: Source Commands (`list source`, `add source`, `remove source`, `enable source`, `disable source`, `update source`)
- ✅ Chunk 7: Help Command
- ✅ Chunk 8: Progress Bars and Spinners
- ✅ Chunk 9: CLI Interop Tests for Phase 1
- ✅ Chunk 10: Performance Benchmarks

**Status**: Documentation Complete - Ready for Implementation
**Commands**: 9/21 (43% - dotnet nuget parity)

---

### Phase 2: Core Operations (Weeks 3-5)
**Document**: [CLI-M2-CORE-OPERATIONS.md](./CLI-M2-CORE-OPERATIONS.md)
- Chunk 1: Search Command Infrastructure
- Chunk 2: Search Command Implementation (V3 Protocol)
- Chunk 3: Search Command (V2 Protocol Support)
- Chunk 4: List Command (delegates to search)
- Chunk 5: Install Command - Basic Structure
- Chunk 6: Install Command - Download and Extract
- Chunk 7: Install Command - Framework Compatibility
- Chunk 8: Install Command - packages.config Support
- Chunk 9: Install Command - Progress Reporting
- Chunk 10: CLI Interop Tests for Phase 2

**Status**: Not Started
**Commands**: +2 (11/21 - 52%)

---

### Phase 3: Dependency Resolution (Weeks 6-7)
**Document**: [CLI-M3-DEPENDENCY-RESOLUTION.md](./CLI-M3-DEPENDENCY-RESOLUTION.md)
- Chunk 1: Restore Command - Project Discovery
- Chunk 2: Restore Command - packages.config Restore
- Chunk 3: Restore Command - PackageReference Restore
- Chunk 4: Restore Command - Solution Restore
- Chunk 5: Restore Command - Dependency Graph Building
- Chunk 6: Restore Command - Parallel Downloads
- Chunk 7: Restore Command - Lock File Generation
- Chunk 8: Restore Command - Conflict Resolution
- Chunk 9: Restore Command - Recursive Restore
- Chunk 10: CLI Interop Tests for Phase 3

**Status**: Not Started
**Commands**: +1 (12/21 - 57%)

---

### Phase 4: Package Creation (Weeks 8-9)
**Document**: [CLI-M4-PACKAGE-CREATION.md](./CLI-M4-PACKAGE-CREATION.md)
- Chunk 1: Spec Command - nuspec Generation
- Chunk 2: Pack Command - nuspec Parsing
- Chunk 3: Pack Command - File Collection
- Chunk 4: Pack Command - Property Substitution
- Chunk 5: Pack Command - OPC Package Creation
- Chunk 6: Pack Command - Symbols Package Support
- Chunk 7: Pack Command - MSBuild Discovery (Cross-Platform)
- Chunk 8: Pack Command - MSBuild Project Parsing
- Chunk 9: Pack Command - MSBuild Property Extraction
- Chunk 10: Pack Command - Build Integration
- Chunk 11: Pack Command - Referenced Projects
- Chunk 12: Push Command - Upload Implementation
- Chunk 13: Push Command - Retry Logic
- Chunk 14: Push Command - Symbols Upload
- Chunk 15: CLI Interop Tests for Phase 4

**Status**: Not Started
**Commands**: +2 (14/21 - 67%)
**Critical**: MSBuild integration required for 100% parity

---

### Phase 5: Signing & Security (Weeks 10-11)
**Document**: [CLI-M5-SIGNING-SECURITY.md](./CLI-M5-SIGNING-SECURITY.md)
- Chunk 1: Sign Command - Certificate Loading (File)
- Chunk 2: Sign Command - Certificate Loading (Store - Windows)
- Chunk 3: Sign Command - Certificate Loading (Store - macOS/Linux)
- Chunk 4: Sign Command - PKCS#7 Signature Creation
- Chunk 5: Sign Command - RFC 3161 Timestamping
- Chunk 6: Verify Command - Package Integrity
- Chunk 7: Verify Command - Signature Verification
- Chunk 8: Verify Command - Certificate Chain Validation
- Chunk 9: Trusted-Signers Command - Configuration Management
- Chunk 10: Client-Certs Command - Certificate Management
- Chunk 11: CLI Interop Tests for Phase 5

**Status**: Not Started
**Commands**: +3 (17/21 - 81%)

---

### Phase 6: Advanced Features (Weeks 12-13)
**Document**: [CLI-M6-ADVANCED-FEATURES.md](./CLI-M6-ADVANCED-FEATURES.md)
- Chunk 1: Update Command - Version Discovery
- Chunk 2: Update Command - Constraint Handling (-Safe)
- Chunk 3: Update Command - File Conflict Resolution
- Chunk 4: Update Command - packages.config Update
- Chunk 5: Locals Command - Cache Location Discovery
- Chunk 6: Locals Command - Cache Clearing
- Chunk 7: Add Command - Offline Feed Support
- Chunk 8: Init Command - Feed Initialization
- Chunk 9: Delete Command - Package Removal
- Chunk 10: SetApiKey Command - Credential Storage
- Chunk 11: CLI Interop Tests for Phase 6

**Status**: Not Started
**Commands**: +4 (21/21 - 100%)

---

### Phase 7: Polish & Optimization (Weeks 14-15)
**Document**: [CLI-M7-POLISH-OPTIMIZATION.md](./CLI-M7-POLISH-OPTIMIZATION.md)
- Chunk 1: Localization Infrastructure - XLIFF Loading
- Chunk 2: Localization - String Extraction
- Chunk 3: Localization - Czech (cs) Translation
- Chunk 4: Localization - German (de) Translation
- Chunk 5: Localization - Spanish (es) Translation
- Chunk 6: Localization - French (fr) Translation
- Chunk 7: Localization - Italian (it) Translation
- Chunk 8: Localization - Japanese (ja) Translation
- Chunk 9: Localization - Korean (ko) Translation
- Chunk 10: Localization - Polish (pl) Translation
- Chunk 11: Localization - Portuguese (pt-BR) Translation
- Chunk 12: Localization - Russian (ru) Translation
- Chunk 13: Localization - Turkish (tr) Translation
- Chunk 14: Localization - Chinese Simplified (zh-Hans) Translation
- Chunk 15: Localization - Chinese Traditional (zh-Hant) Translation
- Chunk 16: Shell Completions - Bash
- Chunk 17: Shell Completions - Zsh
- Chunk 18: Shell Completions - Fish
- Chunk 19: Shell Completions - PowerShell
- Chunk 20: Performance Optimization - Profiling
- Chunk 21: Performance Optimization - Memory Reduction
- Chunk 22: Performance Optimization - Startup Time
- Chunk 23: Man Pages Generation
- Chunk 24: Documentation - User Guide
- Chunk 25: Documentation - Examples
- Chunk 26: CLI Interop Tests for Phase 7

**Status**: Not Started
**Commands**: 0 (21/21 - 100%)
**Critical**: All 14 languages required for 100% parity

---

### Phase 8: Platform-Specific (Week 16)
**Document**: [CLI-M8-PLATFORM-SPECIFIC.md](./CLI-M8-PLATFORM-SPECIFIC.md)
- Chunk 1: Windows - Credential Manager Integration
- Chunk 2: Windows - Certificate Store Integration
- Chunk 3: Windows - MSBuild Discovery via Visual Studio Setup API
- Chunk 4: Windows - Registry Access
- Chunk 5: Windows - Long Path Support
- Chunk 6: Windows - Installer (MSI)
- Chunk 7: Windows - Chocolatey Package
- Chunk 8: macOS - Keychain Integration
- Chunk 9: macOS - Security.framework Integration
- Chunk 10: macOS - Homebrew Formula
- Chunk 11: macOS - DMG Installer
- Chunk 12: macOS - Code Signing and Notarization
- Chunk 13: Linux - Secret Service API Integration
- Chunk 14: Linux - .deb Package
- Chunk 15: Linux - .rpm Package
- Chunk 16: Linux - Snap Package
- Chunk 17: Linux - Flatpak
- Chunk 18: Cross-Platform - Build Scripts
- Chunk 19: Cross-Platform - Release Automation
- Chunk 20: CLI Interop Tests for Phase 8

**Status**: Not Started
**Commands**: 0 (21/21 - 100%)

---

## Progress Tracking

### Overall Progress

| Phase | Duration | Commands | Status | Document Lines | Progress |
|-------|----------|----------|--------|----------------|----------|
| 1. Foundation | Weeks 1-2 | 9/21 (43%) | Documentation Complete | 3,506 | 100% |
| 2. Core Operations | Weeks 3-5 | +2 (11/21 - 52%) | Not Started | - | 0% |
| 3. Dependency Resolution | Weeks 6-7 | +1 (12/21 - 57%) | Not Started | - | 0% |
| 4. Package Creation | Weeks 8-9 | +2 (14/21 - 67%) | Not Started | - | 0% |
| 5. Signing & Security | Weeks 10-11 | +3 (17/21 - 81%) | Not Started | - | 0% |
| 6. Advanced Features | Weeks 12-13 | +4 (21/21 - 100%) | Not Started | - | 0% |
| 7. Polish & Optimization | Weeks 14-15 | 0 (21/21 - 100%) | Not Started | - | 0% |
| 8. Platform-Specific | Week 16 | 0 (21/21 - 100%) | Not Started | - | 0% |
| **TOTAL** | **16 weeks** | **21/21** | **Phase 1 Complete** | **~15,000** | **43%** |

### Command Implementation Status

| # | Command | dotnet nuget Equivalent | Phase | Status | CLI Interop | Coverage |
|---|---------|------------------------|-------|--------|-------------|----------|
| 1 | help / --help | `dotnet nuget --help` | 1 | Documentation Complete | Planned | - |
| 2 | --version | `dotnet nuget --version` | 1 | Documentation Complete | Planned | - |
| 3 | config | `dotnet nuget config` | 1 | Documentation Complete | Planned | - |
| 4 | list source | `dotnet nuget list source` | 1 | Documentation Complete | Planned | - |
| 5 | add source | `dotnet nuget add source` | 1 | Documentation Complete | Planned | - |
| 6 | remove source | `dotnet nuget remove source` | 1 | Documentation Complete | Planned | - |
| 7 | enable source | `dotnet nuget enable source` | 1 | Documentation Complete | Planned | - |
| 8 | disable source | `dotnet nuget disable source` | 1 | Documentation Complete | Planned | - |
| 9 | update source | `dotnet nuget update source` | 1 | Documentation Complete | Planned | - |
| 10 | list | Package search/list | 2 | Not Started | - | - |
| 11 | install | Package installation | 2 | Not Started | - | - |
| 12 | restore | `dotnet restore` (similar) | 3 | Not Started | - | - |
| 13 | pack | `dotnet pack` (similar) | 4 | Not Started | - | - |
| 14 | push | `dotnet nuget push` | 4 | Not Started | - | - |
| 15 | sign | `dotnet nuget sign` | 5 | Not Started | - | - |
| 16 | verify | `dotnet nuget verify` | 5 | Not Started | - | - |
| 17 | trust | `dotnet nuget trust` | 5 | Not Started | - | - |
| 18 | locals | `dotnet nuget locals` | 6 | Not Started | - | - |
| 19 | delete | `dotnet nuget delete` | 6 | Not Started | - | - |
| 20 | add (package) | Add to local feed | 6 | Not Started | - | - |
| 21 | init | Initialize local feed | 6 | Not Started | - | - |

### Acceptance Criteria

- [ ] All 21 commands implemented
- [ ] 100% CLI interop tests passing (output matches `dotnet nuget`)
- [ ] 100% library interop tests passing (NuGet.Client parity)
- [ ] Startup time < 50ms (P50)
- [ ] All 14 languages supported
- [ ] 90%+ test coverage
- [ ] Zero linter warnings
- [ ] MSBuild integration complete (Windows, Linux, macOS)
- [ ] All platform-specific features implemented
- [ ] Security audit passed
- [ ] Documentation complete
- [ ] Cross-platform validated (Windows, macOS, Linux)

---

## Document Structure

Each implementation guide follows this structure:

### Per-Chunk Format

```markdown
## Chunk N: [Feature Name]

**Objective**: Clear, measurable objective

**Prerequisites**: What must be complete before starting

**Files to create/modify**: List of files

### Step N.1: [Sub-step name]
- Code implementation
- Detailed instructions

### Step N.2: [Sub-step name]
- Continue implementation

### Verification
- Manual testing steps
- Expected outputs
- Compare with `dotnet nuget` output

### CLI Interop Testing
- Add handler to `cmd/gonuget-cli-interop-test/handlers_*.go`
- Add C# test to `tests/cli-interop/GonugetCliInterop.Tests/`
- Validate output matches `dotnet nuget`
- Test execution commands

### Unit Testing
- Unit test implementation
- Test execution commands

### Commit
- Git commit with conventional commit message
- Summary of changes
```

### Verification at Each Chunk

Every chunk ends with:
1. **Verification**: Manual testing comparing with `dotnet nuget`
2. **CLI Interop Testing**: Automated C# tests via JSON-RPC bridge
3. **Unit Testing**: Go unit tests with coverage check
4. **Commit**: Git commit with clear message

This ensures:
- Incremental progress
- Working software at each step
- Clear rollback points
- Testable deliverables
- Cross-platform compatibility via interop tests

---

## Usage Instructions

1. **Start with Phase 1**: Begin with CLI-M1-FOUNDATION.md
2. **Follow chunks sequentially**: Each chunk builds on previous ones
3. **Verify before proceeding**: Run verification and tests after each chunk
4. **Commit frequently**: Commit after each chunk completion
5. **Track progress**: Update this index as you complete chunks

**For AI Coding Assistants**:
- Each chunk is bite-sized (typically 100-300 lines of code)
- Clear objectives prevent wandering
- Verification steps ensure correctness
- Tests provide immediate feedback
- Commits create safe checkpoints

---

## Critical Path Items

### Must-Have for v1.0 (100% Parity)

1. **MSBuild Integration** (Phase 4, Chunks 7-11):
   - Cross-platform MSBuild discovery
   - Project file parsing
   - Property extraction and substitution
   - Build integration
   - Referenced project handling

2. **Localization** (Phase 7, Chunks 1-15):
   - All 14 languages
   - XLIFF format matching nuget.exe
   - Locale detection
   - String extraction workflow

3. **Credential Providers** (Phase 5, Chunk 10):
   - Discovery mechanism
   - stdin/stdout JSON protocol
   - Environment variable passing
   - Compatible with Azure Artifacts, AWS CodeArtifact

4. **Platform-Specific Features** (Phase 8):
   - Windows: Credential Manager, Certificate Store
   - macOS: Keychain
   - Linux: Secret Service API

---

## Dependencies

### External Packages

```go
require (
    github.com/spf13/cobra v1.8.0           // CLI framework
    github.com/spf13/viper v1.18.2          // Configuration
    github.com/fatih/color v1.16.0          // Colored output
    github.com/schollz/progressbar/v3 v3.14.1 // Progress bars
    github.com/olekukonko/tablewriter v0.0.5  // Table formatting
    github.com/zalando/go-keyring v0.2.3      // OS keychain
)
```

### Internal Packages (gonuget library)

All gonuget library milestones (M1-M8) must be complete:
- version (M1)
- frameworks (M2)
- packaging (M3)
- protocol/v2, protocol/v3 (M4, M5)
- resolver (M6)
- cache (M7)
- auth (M8)

---

## Next Steps

1. ✅ **CLI-M1-FOUNDATION.md** (Chunks 1-5) - Documentation Complete
2. ✅ **CLI-M1-FOUNDATION-CONTINUED.md** (Chunks 6-10) - Documentation Complete
3. **Implement Phase 1** (Chunks 1-10) following the updated documentation
4. **Update CLI-M2-CORE-OPERATIONS.md** with CLI interop tests and dotnet nuget parity
5. **Update CLI-M3-DEPENDENCY-RESOLUTION.md** with CLI interop tests
6. **Update CLI-M4-PACKAGE-CREATION.md** (Critical: MSBuild) with CLI interop tests
7. **Update CLI-M5-SIGNING-SECURITY.md** with CLI interop tests
8. **Update CLI-M6-ADVANCED-FEATURES.md** with CLI interop tests
9. **Update CLI-M7-POLISH-OPTIMIZATION.md** (Critical: Localization)
10. **Update CLI-M8-PLATFORM-SPECIFIC.md**

---

**Last Updated**: 2025-01-25
**Status**: Phase 1 Documentation Complete (43% of commands documented)
**Implementation Status**: Ready to implement Phase 1 (9/21 commands)
**Target**: 100% parity with `dotnet nuget` (cross-platform)
**Reference**: dotnet/sdk and NuGet.Client/NuGet.CommandLine.XPlat
**Timeline**: 16 weeks to feature-complete v1.0
