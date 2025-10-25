# CLI Implementation Milestones - Index

**Project**: gonuget CLI
**Target**: 100% parity with nuget.exe
**Total Duration**: 16 weeks (4 months)
**Prerequisites**: gonuget library M1-M8 complete

---

## Implementation Guide Documents

### Phase 1: Foundation (Weeks 1-2)
**Documents**:
- [CLI-M1-FOUNDATION.md](./CLI-M1-FOUNDATION.md) (Chunks 1-5, 1,706 lines)
- [CLI-M1-FOUNDATION-CONTINUED.md](./CLI-M1-FOUNDATION-CONTINUED.md) (Chunks 6-10, 1,800 lines)

- ⏳ Chunk 1: Project Structure and Entry Point
- ⏳ Chunk 2: Console Abstraction
- ⏳ Chunk 3: Configuration Management (NuGet.config XML)
- ⏳ Chunk 4: Version Command
- ⏳ Chunk 5: Config Command (Reading)
- ⏳ Chunk 6: Sources Command (list, add, remove, enable, disable)
- ⏳ Chunk 7: Help Command
- ⏳ Chunk 8: Progress Bars and Spinners
- ⏳ Chunk 9: Integration Tests for Phase 1
- ⏳ Chunk 10: Performance Benchmarks

**Status**: Documentation Complete (Ready to implement - 3,506 lines of guides)
**Commands**: 0/20 (0% - guides only)

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
- Chunk 10: Integration Tests for Phase 2

**Status**: Not Started
**Commands**: +3 (5/20 - 25%)

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
- Chunk 10: Integration Tests for Phase 3

**Status**: Not Started
**Commands**: +1 (6/20 - 30%)

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
- Chunk 15: Integration Tests for Phase 4

**Status**: Not Started
**Commands**: +3 (9/20 - 45%)
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
- Chunk 11: Integration Tests for Phase 5

**Status**: Not Started
**Commands**: +4 (13/20 - 65%)

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
- Chunk 11: Integration Tests for Phase 6

**Status**: Not Started
**Commands**: +6 (19/20 - 95%)

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
- Chunk 26: Integration Tests for Phase 7

**Status**: Not Started
**Commands**: 0 (19/20 - 95%)
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
- Chunk 20: Integration Tests for Phase 8

**Status**: Not Started
**Commands**: +1 (20/20 - 100%)

---

## Progress Tracking

### Overall Progress

| Phase | Duration | Commands | Status | Document Lines | Progress |
|-------|----------|----------|--------|----------------|----------|
| 1. Foundation | Weeks 1-2 | 2/20 | In Progress | 1,706 | 50% |
| 2. Core Operations | Weeks 3-5 | +3 | Not Started | - | 0% |
| 3. Dependency Resolution | Weeks 6-7 | +1 | Not Started | - | 0% |
| 4. Package Creation | Weeks 8-9 | +3 | Not Started | - | 0% |
| 5. Signing & Security | Weeks 10-11 | +4 | Not Started | - | 0% |
| 6. Advanced Features | Weeks 12-13 | +6 | Not Started | - | 0% |
| 7. Polish & Optimization | Weeks 14-15 | 0 | Not Started | - | 0% |
| 8. Platform-Specific | Week 16 | +1 | Not Started | - | 0% |
| **TOTAL** | **16 weeks** | **20/20** | **In Progress** | **~15,000** | **6%** |

### Command Implementation Status

| # | Command | Phase | Status | Tests | Coverage |
|---|---------|-------|--------|-------|----------|
| 1 | help | 1 | Planned | - | - |
| 2 | version | 1 | ✅ Complete | ✅ | 95% |
| 3 | config | 1 | ✅ Complete | ✅ | 90% |
| 4 | sources | 1 | Planned | - | - |
| 5 | search | 2 | Not Started | - | - |
| 6 | list | 2 | Not Started | - | - |
| 7 | install | 2 | Not Started | - | - |
| 8 | restore | 3 | Not Started | - | - |
| 9 | spec | 4 | Not Started | - | - |
| 10 | pack | 4 | Not Started | - | - |
| 11 | push | 4 | Not Started | - | - |
| 12 | sign | 5 | Not Started | - | - |
| 13 | verify | 5 | Not Started | - | - |
| 14 | trusted-signers | 5 | Not Started | - | - |
| 15 | client-certs | 5 | Not Started | - | - |
| 16 | update | 6 | Not Started | - | - |
| 17 | locals | 6 | Not Started | - | - |
| 18 | add | 6 | Not Started | - | - |
| 19 | init | 6 | Not Started | - | - |
| 20 | delete | 6 | Not Started | - | - |
| 21 | setapikey | 6 | Not Started | - | - |

### Acceptance Criteria

- [ ] All 20 commands implemented
- [ ] 100% NuGet.Client interop tests passing
- [ ] Startup time < 50ms (P50)
- [ ] All 14 languages supported
- [ ] 90%+ test coverage
- [ ] Zero linter warnings
- [ ] MSBuild integration complete (Windows, Linux, macOS)
- [ ] All platform-specific features implemented
- [ ] Security audit passed
- [ ] Documentation complete

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

### Testing
- Unit test implementation
- Test execution commands

### Commit
- Git commit with conventional commit message
- Summary of changes
```

### Verification at Each Chunk

Every chunk ends with:
1. **Verification**: Manual testing to ensure functionality
2. **Testing**: Automated tests with coverage check
3. **Commit**: Git commit with clear message

This ensures:
- Incremental progress
- Working software at each step
- Clear rollback points
- Testable deliverables

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

1. **Complete CLI-M1-FOUNDATION.md** (Chunks 6-10)
2. **Create CLI-M2-CORE-OPERATIONS.md**
3. **Create CLI-M3-DEPENDENCY-RESOLUTION.md**
4. **Create CLI-M4-PACKAGE-CREATION.md** (Critical: MSBuild)
5. **Create CLI-M5-SIGNING-SECURITY.md**
6. **Create CLI-M6-ADVANCED-FEATURES.md**
7. **Create CLI-M7-POLISH-OPTIMIZATION.md** (Critical: Localization)
8. **Create CLI-M8-PLATFORM-SPECIFIC.md**

---

**Last Updated**: 2025-10-25
**Status**: Phase 1 in progress (6% complete overall)
**Target**: 100% parity with nuget.exe
**Timeline**: 16 weeks to feature-complete v1.0
