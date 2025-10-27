# CLI M2.2: Advanced Package Management Features

**Status:** Phase 2 - Package Management (Advanced Features)
**Commands:** `gonuget add package` (advanced scenarios)
**Parity Target:** `dotnet add package` (100% feature parity)

## Overview

This guide covers M2.2 Advanced Features (Chunks 14-19) that extend the basic add package command with framework-specific references, multi-TFM support, solution-level operations, and CLI interop testing. These features ensure gonuget achieves 100% parity with `dotnet add package`.

**Prerequisites:**
- ✅ M2.1 Complete (Chunks 1-10): Basic add package and restore
- ✅ Chunks 11-13 Complete: Central Package Management (CPM)

**Reference Implementations:**
- Primary: `/Users/brandon/src/sdk/src/Cli/dotnet/` - dotnet SDK implementation
- Secondary: `/Users/brandon/src/NuGet.Client/` - NuGet.Client libraries
- Docs: `/Users/brandon/src/docs/` - Official NuGet documentation

---

## Chunk 14: Framework-Specific References (Conditional ItemGroups)

**Objective**: Support adding packages to specific target frameworks using conditional ItemGroups.

**Time Estimate**: 4 hours

**Prerequisites**: Chunks 1-13 complete

### Background

When a project targets multiple frameworks (e.g., `<TargetFrameworks>net8.0;net48</TargetFrameworks>`), packages may need to be added only for specific frameworks. This requires:

1. Conditional `<ItemGroup>` elements with `Condition` attributes
2. Framework compatibility validation
3. Proper TFM (Target Framework Moniker) parsing

**Example:**
```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>

  <!-- Package for all frameworks -->
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>

  <!-- Package only for .NET 8 -->
  <ItemGroup Condition="'$(TargetFramework)' == 'net8.0'">
    <PackageReference Include="System.Text.Json" Version="8.0.0" />
  </ItemGroup>

  <!-- Package only for .NET Framework 4.8 -->
  <ItemGroup Condition="'$(TargetFramework)' == 'net48'">
    <PackageReference Include="System.Configuration.ConfigurationManager" Version="8.0.0" />
  </ItemGroup>
</Project>
```

### Reference Implementation

**dotnet SDK**: Automatically handles framework-specific references
- Location: `/Users/brandon/src/sdk/src/Tasks/Microsoft.NET.Build.Tasks/`
- MSBuild handles conditional evaluation

**NuGet.Client**: Framework compatibility logic
- Location: `/Users/brandon/src/NuGet.Client/src/NuGet.Frameworks/`
- `FrameworkReducer.cs` - Framework compatibility logic

### Files to Modify

**cmd/gonuget/project/xml.go**:
```go
// ItemGroup represents an <ItemGroup> element containing package references or other items.
type ItemGroup struct {
	Condition         string              `xml:"Condition,attr,omitempty"`  // Already exists
	PackageReferences []PackageReference  `xml:"PackageReference,omitempty"`
	ProjectReferences []Reference         `xml:"ProjectReference,omitempty"`
	References        []AssemblyReference `xml:"Reference,omitempty"`
}
```

**cmd/gonuget/project/project.go** - Modify `AddOrUpdatePackageReference`:
```go
// AddOrUpdatePackageReference adds a new PackageReference or updates an existing one.
// If frameworks is non-empty, adds to a conditional ItemGroup for those frameworks.
// Returns true if an existing PackageReference was updated, false if a new one was added.
func (p *Project) AddOrUpdatePackageReference(id, version string, frameworks []string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("package ID cannot be empty")
	}

	// M2.2 Chunk 14: Support framework-specific references
	var condition string
	if len(frameworks) > 0 {
		// Build condition string: '$(TargetFramework)' == 'net8.0'
		// For multiple frameworks: '$(TargetFramework)' == 'net8.0' OR '$(TargetFramework)' == 'net48'
		condition = buildFrameworkCondition(frameworks)

		// Validate framework compatibility with package
		for _, fw := range frameworks {
			if err := validateFrameworkCompatibility(p, id, fw); err != nil {
				return false, fmt.Errorf("package %s not compatible with framework %s: %w", id, fw, err)
			}
		}
	}

	// Find existing PackageReference in matching ItemGroup
	for _, ig := range p.Root.ItemGroups {
		// Match condition (empty condition matches unconditional ItemGroup)
		if normalizeCondition(ig.Condition) != normalizeCondition(condition) {
			continue
		}

		for i := range ig.PackageReferences {
			pr := &ig.PackageReferences[i]
			if strings.EqualFold(pr.Include, id) {
				// Update existing reference
				if version != "" {
					pr.Version = version
				}
				p.modified = true
				return true, nil
			}
		}
	}

	// Not found, add new PackageReference
	itemGroup := p.findOrCreateItemGroup(condition)
	itemGroup.PackageReferences = append(itemGroup.PackageReferences, PackageReference{
		Include: id,
		Version: version,
	})

	p.modified = true
	return false, nil
}

// buildFrameworkCondition builds an MSBuild condition string for framework filtering
func buildFrameworkCondition(frameworks []string) string {
	if len(frameworks) == 0 {
		return ""
	}

	if len(frameworks) == 1 {
		return fmt.Sprintf("'$(TargetFramework)' == '%s'", frameworks[0])
	}

	// Multiple frameworks: OR conditions
	conditions := make([]string, len(frameworks))
	for i, fw := range frameworks {
		conditions[i] = fmt.Sprintf("'$(TargetFramework)' == '%s'", fw)
	}
	return strings.Join(conditions, " OR ")
}

// normalizeCondition normalizes condition strings for comparison (trim whitespace, case-insensitive)
func normalizeCondition(condition string) string {
	// Normalize whitespace and case for comparison
	condition = strings.TrimSpace(condition)
	condition = strings.ToLower(condition)
	// Normalize quotes (both single and double quotes)
	condition = strings.ReplaceAll(condition, "\"", "'")
	return condition
}

// validateFrameworkCompatibility checks if a package is compatible with a target framework.
// This performs basic TFM validation. Full package compatibility is validated during restore.
func validateFrameworkCompatibility(p *Project, packageID, framework string) error {
	// Parse and validate the target framework moniker
	_, err := frameworks.Parse(framework)
	if err != nil {
		return fmt.Errorf("invalid target framework '%s': %w", framework, err)
	}

	// Design Note: Package-to-framework compatibility validation is intentionally deferred
	// to the restore phase. This matches dotnet behavior (see sdk/src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs).
	//
	// Rationale:
	// 1. The restore.Restorer already downloads packages and parses nuspecs
	// 2. Restore uses frameworks.FrameworkReducer for compatibility checks
	// 3. Restore provides detailed error messages when incompatible
	// 4. Pre-validation would duplicate work and slow down add command
	// 5. Users get immediate feedback since restore runs by default (unless --no-restore)
	//
	// If --no-restore is used, users won't see compatibility errors until they run restore.
	// This is acceptable and matches dotnet behavior.
	return nil
}

// findOrCreateItemGroup finds an ItemGroup with the given condition or creates a new one
func (p *Project) findOrCreateItemGroup(condition string) *ItemGroup {
	// Find existing ItemGroup with matching condition
	normalizedCondition := normalizeCondition(condition)
	for i := range p.Root.ItemGroups {
		ig := &p.Root.ItemGroups[i]
		if normalizeCondition(ig.Condition) == normalizedCondition {
			return ig
		}
	}

	// Create new ItemGroup with condition
	itemGroup := ItemGroup{
		Condition:         condition,
		PackageReferences: []PackageReference{},
	}
	p.Root.ItemGroups = append(p.Root.ItemGroups, itemGroup)
	return &p.Root.ItemGroups[len(p.Root.ItemGroups)-1]
}
```

**cmd/gonuget/commands/add_package.go** - Update to use framework flag:
```go
// runAddPackage already accepts frameworks parameter
// Current code:
//   frameworks := []string{}
//   if opts.Framework != "" {
//       frameworks = []string{opts.Framework}
//   }
// This continues to work with the updated AddOrUpdatePackageReference
```

### Testing

**cmd/gonuget/project/project_test.go**:
```go
func TestAddOrUpdatePackageReference_WithFramework(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	// Multi-TFM project
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	// Add package for net8.0 only
	updated, err := proj.AddOrUpdatePackageReference("System.Text.Json", "8.0.0", []string{"net8.0"})
	require.NoError(t, err)
	assert.False(t, updated)

	// Save and reload
	err = proj.Save()
	require.NoError(t, err)

	proj2, err := LoadProject(projectPath)
	require.NoError(t, err)

	// Verify conditional ItemGroup was created
	found := false
	for _, ig := range proj2.Root.ItemGroups {
		if strings.Contains(strings.ToLower(ig.Condition), "net8.0") {
			found = true
			assert.Len(t, ig.PackageReferences, 1)
			assert.Equal(t, "System.Text.Json", ig.PackageReferences[0].Include)
			assert.Equal(t, "8.0.0", ig.PackageReferences[0].Version)
			break
		}
	}
	assert.True(t, found, "Conditional ItemGroup should exist")
}

func TestBuildFrameworkCondition(t *testing.T) {
	tests := []struct {
		name       string
		frameworks []string
		want       string
	}{
		{
			name:       "No frameworks",
			frameworks: []string{},
			want:       "",
		},
		{
			name:       "Single framework",
			frameworks: []string{"net8.0"},
			want:       "'$(TargetFramework)' == 'net8.0'",
		},
		{
			name:       "Multiple frameworks",
			frameworks: []string{"net8.0", "net48"},
			want:       "'$(TargetFramework)' == 'net8.0' OR '$(TargetFramework)' == 'net48'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFrameworkCondition(tt.frameworks)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeCondition(t *testing.T) {
	tests := []struct {
		name string
		cond string
		want string
	}{
		{
			name: "Single quotes",
			cond: "'$(TargetFramework)' == 'net8.0'",
			want: "'$(targetframework)' == 'net8.0'",
		},
		{
			name: "Double quotes",
			cond: "\"$(TargetFramework)\" == \"net8.0\"",
			want: "'$(targetframework)' == 'net8.0'",
		},
		{
			name: "Extra whitespace",
			cond: "  '$(TargetFramework)'  ==  'net8.0'  ",
			want: "'$(targetframework)' == 'net8.0'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCondition(tt.cond)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

**cmd/gonuget/commands/add_package_test.go**:
```go
func TestRunAddPackage_WithFramework(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	// Multi-TFM project
	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "8.0.0",
		Framework:   "net8.0",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "System.Text.Json", opts)
	assert.NoError(t, err)

	// Verify conditional ItemGroup in project file
	content, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	contentStr := string(content)

	assert.Contains(t, contentStr, "System.Text.Json")
	assert.Contains(t, contentStr, "8.0.0")
	assert.Contains(t, contentStr, "Condition")
	assert.Contains(t, contentStr, "net8.0")
}
```

### Verification

```bash
# Build CLI
go build ./cmd/gonuget

# Test framework-specific reference
mkdir test-fw && cd test-fw
cat > test.csproj <<EOF
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>
</Project>
EOF

# Add package for net8.0 only
./gonuget add package System.Text.Json --version 8.0.0 --framework net8.0 --no-restore

# Verify conditional ItemGroup created
cat test.csproj
# Should show:
# <ItemGroup Condition="'$(TargetFramework)' == 'net8.0'">
#   <PackageReference Include="System.Text.Json" Version="8.0.0" />
# </ItemGroup>

# Compare with dotnet
dotnet add package System.Drawing.Common --framework net48
# Verify output matches
```

### Success Criteria

- ✅ `--framework` flag works correctly
- ✅ Conditional ItemGroup created with proper MSBuild syntax
- ✅ Multiple frameworks supported (OR conditions)
- ✅ Condition normalization handles quotes and whitespace
- ✅ Existing conditional ItemGroups reused when condition matches
- ✅ Package added to correct ItemGroup based on framework
- ✅ Unit tests pass with 90%+ coverage
- ✅ Manual testing shows identical output to `dotnet add package --framework`

---

## Chunk 15: Transitive Dependency Resolution

**Objective**: Integrate gonuget's existing `core/resolver` to resolve and report transitive dependencies during add package.

**Time Estimate**: 3 hours

**Prerequisites**: Chunks 1-14 complete

### Background

When adding a package, dotnet shows transitive dependencies:

```bash
$ dotnet add package Serilog.Sinks.File --version 5.0.0
  info : Adding PackageReference for package 'Serilog.Sinks.File' into project '/path/to/test.csproj'.
  info : Restoring packages for /path/to/test.csproj...
  info :   GET https://api.nuget.org/v3/...
  info : Package 'Serilog.Sinks.File' is compatible with all the specified frameworks.
  info : PackageReference for package 'Serilog.Sinks.File' version '5.0.0' added to file '/path/to/test.csproj'.
  info : Writing assets file to disk. Path: /path/to/obj/project.assets.json
  log  : Restored /path/to/test.csproj (in 2.1 sec).
```

gonuget should show similar output with transitive dependencies resolved.

### Reference Implementation

**gonuget library**: `core/resolver` package already implements dependency resolution
- `core/resolver/walker.go` - DependencyWalker for graph traversal
- `core/resolver/resolver.go` - Package resolution with conflict detection

**dotnet SDK**: Uses NuGet.Commands for restore
- Location: `/Users/brandon/src/NuGet.Client/src/NuGet.Commands/RestoreCommand/`

### Files to Modify

**cmd/gonuget/commands/add_package.go** - Enhance restore output:
```go
// Add imports
import (
	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/frameworks"
)

// Modify the restore section in runAddPackage and addPackageWithCPM
// After restore completes, show transitive dependencies:

func showTransitiveDependencies(proj *project.Project, result *restore.RestoreResult) {
	fmt.Println("info : Package dependencies:")

	// Group by direct vs transitive
	direct := make(map[string]string)
	transitive := make(map[string]string)

	// Direct dependencies from project file
	for _, ref := range proj.GetPackageReferences() {
		direct[strings.ToLower(ref.Include)] = ref.Version
	}

	// All resolved packages
	for pkgID, pkgInfo := range result.ResolvedPackages {
		pkgIDLower := strings.ToLower(pkgID)
		if _, isDirect := direct[pkgIDLower]; !isDirect {
			transitive[pkgID] = pkgInfo.Version
		}
	}

	// Display direct dependencies
	if len(direct) > 0 {
		fmt.Println("info :   Direct:")
		for id, ver := range direct {
			fmt.Printf("info :     %s (%s)\n", id, ver)
		}
	}

	// Display transitive dependencies
	if len(transitive) > 0 {
		fmt.Println("info :   Transitive:")
		for id, ver := range transitive {
			fmt.Printf("info :     %s (%s)\n", id, ver)
		}
	}

	fmt.Printf("info : Total packages: %d (Direct: %d, Transitive: %d)\n",
		len(direct)+len(transitive), len(direct), len(transitive))
}

// Update runAddPackage restore section:
if !opts.NoRestore {
	// ... existing restore code ...

	if err := lockFile.Save(assetsPath); err != nil {
		return fmt.Errorf("failed to save project.assets.json: %w", err)
	}

	// Show transitive dependencies
	showTransitiveDependencies(proj, result)

	fmt.Println("info : Package added successfully")
}
```

### Testing

**cmd/gonuget/commands/add_package_test.go**:
```go
func TestRunAddPackage_ShowsTransitiveDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "5.0.0",
		NoRestore:   false, // Allow restore
	}

	err = runAddPackage(context.Background(), "Serilog.Sinks.File", opts)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)

	// Verify transitive dependency output
	assert.Contains(t, output, "Package dependencies:")
	assert.Contains(t, output, "Direct:")
	assert.Contains(t, output, "Serilog.Sinks.File")
	assert.Contains(t, output, "Transitive:")
	// Serilog.Sinks.File depends on Serilog
	assert.Contains(t, output, "Serilog")
}
```

### Verification

```bash
# Test with a package that has transitive dependencies
./gonuget add package Serilog.Sinks.File --version 5.0.0

# Expected output:
# info : Adding PackageReference for package 'Serilog.Sinks.File' into project '/path/to/test.csproj'.
# info : Restored 3 packages to /Users/.../.nuget/packages/
# info : Package dependencies:
# info :   Direct:
# info :     Serilog.Sinks.File (5.0.0)
# info :   Transitive:
# info :     Serilog (2.12.0)
# info : Total packages: 2 (Direct: 1, Transitive: 1)
# info : Package added successfully

# Compare with dotnet
dotnet add package Serilog.Sinks.File --version 5.0.0
```

### Success Criteria

- ✅ Transitive dependencies resolved using `core/resolver`
- ✅ Dependencies grouped into Direct and Transitive
- ✅ Output shows package hierarchy
- ✅ Total package count displayed
- ✅ Integration test verifies transitive resolution
- ✅ Performance acceptable (< 5 seconds for typical packages)

---

## Chunk 16: Multi-TFM Project Support

**Objective**: Handle projects with multiple target frameworks (`<TargetFrameworks>`).

**Time Estimate**: 3 hours

**Prerequisites**: Chunks 1-15 complete

### Background

Projects can target multiple frameworks:

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48;netstandard2.0</TargetFrameworks>
  </PropertyGroup>
</Project>
```

When adding a package to a multi-TFM project:
1. Validate package compatibility with ALL target frameworks
2. Add to global ItemGroup if compatible with all
3. Add to conditional ItemGroups if only compatible with some frameworks

### Reference Implementation

**gonuget library**:
- `frameworks/` package - TFM parsing and compatibility
- `frameworks/reducer.go` - Framework reduction and compatibility logic

**NuGet.Client**:
- Location: `/Users/brandon/src/NuGet.Client/src/NuGet.Frameworks/`
- `FrameworkReducer.cs` - Framework compatibility

### Files to Modify

**cmd/gonuget/project/project.go** - Add multi-TFM support:
```go
import (
	"github.com/willibrandon/gonuget/frameworks"
)

// GetTargetFrameworks returns the list of target frameworks for the project.
// Returns single framework from TargetFramework or multiple from TargetFrameworks.
func (p *Project) GetTargetFrameworks() []string {
	for _, pg := range p.Root.Properties {
		// TargetFrameworks (plural) - multiple frameworks
		if pg.TargetFrameworks != "" {
			return strings.Split(pg.TargetFrameworks, ";")
		}

		// TargetFramework (singular) - single framework
		if pg.TargetFramework != "" {
			return []string{pg.TargetFramework}
		}
	}
	return []string{}
}

// IsMultiTargeting returns true if project targets multiple frameworks.
func (p *Project) IsMultiTargeting() bool {
	return len(p.GetTargetFrameworks()) > 1
}

// AddOrUpdatePackageReference (enhanced for multi-TFM)
func (p *Project) AddOrUpdatePackageReference(id, version string, frameworks []string) (bool, error) {
	// ... existing validation ...

	// If no specific frameworks requested but project is multi-targeting,
	// validate compatibility with ALL project frameworks
	if len(frameworks) == 0 && p.IsMultiTargeting() {
		projectFrameworks := p.GetTargetFrameworks()

		// Check if package is compatible with all frameworks
		compatible, incompatible := validateMultiFrameworkCompatibility(id, version, projectFrameworks)

		if len(incompatible) > 0 {
			// Package not compatible with all frameworks
			// Offer to add only to compatible frameworks
			return false, fmt.Errorf(
				"package %s is not compatible with all target frameworks.\n"+
				"Compatible: %v\n"+
				"Incompatible: %v\n"+
				"Use --framework flag to add for specific frameworks only.",
				id, compatible, incompatible)
		}
	}

	// ... rest of existing logic ...
}

// validateMultiFrameworkCompatibility validates target framework monikers for a multi-TFM project.
// Returns lists of valid and invalid TFMs. Package compatibility is checked during restore.
func validateMultiFrameworkCompatibility(packageID, version string, targetFrameworks []string) (compatible, incompatible []string) {
	for _, tfm := range targetFrameworks {
		// Validate TFM syntax
		_, err := frameworks.Parse(tfm)
		if err != nil {
			// Invalid TFM syntax
			incompatible = append(incompatible, tfm)
			continue
		}

		// TFM is valid
		compatible = append(compatible, tfm)
	}

	// Design Note: We only validate TFM syntax here, not package-to-framework compatibility.
	// Package compatibility validation happens during restore (matches dotnet behavior).
	//
	// The restore.Restorer will:
	// - Download package and parse nuspec
	// - Check nuspec dependency groups using frameworks.FrameworkReducer
	// - Report detailed compatibility errors if package doesn't support the framework
	//
	// This avoids duplicate work and provides better error messages.
	return compatible, incompatible
}
```

**cmd/gonuget/commands/add_package.go** - Handle multi-TFM messages:
```go
func runAddPackage(ctx context.Context, packageID string, opts *AddPackageOptions) error {
	// ... existing project loading ...

	// Check if multi-targeting
	if proj.IsMultiTargeting() && opts.Framework == "" {
		frameworks := proj.GetTargetFrameworks()
		fmt.Printf("info : Project targets multiple frameworks: %v\n", frameworks)
		fmt.Println("info : Package will be added for all frameworks if compatible")
	}

	// ... rest of existing logic ...
}
```

### Testing

**cmd/gonuget/project/project_test.go**:
```go
func TestGetTargetFrameworks_Single(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	frameworks := proj.GetTargetFrameworks()
	assert.Equal(t, []string{"net8.0"}, frameworks)
	assert.False(t, proj.IsMultiTargeting())
}

func TestGetTargetFrameworks_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48;netstandard2.0</TargetFrameworks>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	frameworks := proj.GetTargetFrameworks()
	assert.Equal(t, []string{"net8.0", "net48", "netstandard2.0"}, frameworks)
	assert.True(t, proj.IsMultiTargeting())
}

func TestAddOrUpdatePackageReference_MultiTFM_AllCompatible(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	// Add package compatible with all frameworks
	updated, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", nil)
	require.NoError(t, err)
	assert.False(t, updated)

	// Should be added to global ItemGroup (no condition)
	found := false
	for _, ig := range proj.Root.ItemGroups {
		if ig.Condition == "" {
			for _, pr := range ig.PackageReferences {
				if pr.Include == "Newtonsoft.Json" {
					found = true
					break
				}
			}
		}
	}
	assert.True(t, found, "Package should be in global ItemGroup")
}
```

**cmd/gonuget/commands/add_package_test.go**:
```go
func TestRunAddPackage_MultiTFM(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test.csproj")

	projectContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>
</Project>`
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	opts := &AddPackageOptions{
		ProjectPath: projectPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify package added to global ItemGroup
	content, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	contentStr := string(content)

	assert.Contains(t, contentStr, "Newtonsoft.Json")
	assert.Contains(t, contentStr, "13.0.3")
	// Should NOT have Condition attribute for compatible package
	assert.NotContains(t, contentStr, "Condition")
}
```

### Verification

```bash
# Create multi-TFM project
mkdir test-multitfm && cd test-multitfm
cat > test.csproj <<EOF
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48;netstandard2.0</TargetFrameworks>
  </PropertyGroup>
</Project>
EOF

# Add package compatible with all frameworks
./gonuget add package Newtonsoft.Json --version 13.0.3 --no-restore

# Verify added to global ItemGroup (no condition)
cat test.csproj

# Try framework-specific add
./gonuget add package System.Text.Json --version 8.0.0 --framework net8.0 --no-restore

# Verify conditional ItemGroup created
cat test.csproj

# Compare with dotnet
dotnet add package Newtonsoft.Json --version 13.0.3
dotnet add package System.Text.Json --version 8.0.0 --framework net8.0
```

### Success Criteria

- ✅ `GetTargetFrameworks()` correctly parses TargetFramework and TargetFrameworks
- ✅ `IsMultiTargeting()` detects multi-TFM projects
- ✅ Compatible packages added to global ItemGroup
- ✅ Informative messages for multi-TFM projects
- ✅ Error handling for incompatible packages
- ✅ Unit tests pass with 90%+ coverage
- ✅ Manual testing shows correct behavior

---

## Chunk 17: Solution File Support

**Objective**: Support adding packages to all projects in a solution file (.sln).

**Time Estimate**: 5 hours

**Prerequisites**: Chunks 1-16 complete

### Background

dotnet supports operating on solution files:

```bash
# Add package to all projects in solution
dotnet add MySolution.sln package Newtonsoft.Json --version 13.0.3

# Add package to specific project in solution
dotnet add MyProject/MyProject.csproj package Newtonsoft.Json
```

gonuget should support the same patterns.

### Reference Implementation

**dotnet SDK**: Solution parsing
- Location: `/Users/brandon/src/sdk/src/Cli/Microsoft.DotNet.Cli.Utils/`
- Solution file parsing and project discovery

**NuGet.Client**: Solution-level operations
- Location: `/Users/brandon/src/NuGet.Client/src/NuGet.CommandLine/`

### Files to Create

**cmd/gonuget/solution/solution.go** (NEW):
```go
package solution

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Solution represents a .NET solution file (.sln)
type Solution struct {
	Path     string
	Projects []ProjectReference
}

// ProjectReference represents a project in a solution
type ProjectReference struct {
	Name         string // Project name
	Path         string // Relative path to project file
	AbsolutePath string // Absolute path to project file
	GUID         string // Project GUID
	TypeGUID     string // Project type GUID
}

// LoadSolution loads a solution file from disk
func LoadSolution(path string) (*Solution, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open solution file: %w", err)
	}
	defer file.Close()

	solution := &Solution{
		Path:     path,
		Projects: []ProjectReference{},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse project lines:
		// Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "MyProject", "MyProject\MyProject.csproj", "{12345678-1234-1234-1234-123456789ABC}"
		if strings.HasPrefix(line, "Project(") {
			proj, err := parseProjectLine(line, filepath.Dir(path))
			if err != nil {
				// Skip invalid project lines
				continue
			}
			solution.Projects = append(solution.Projects, proj)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read solution file: %w", err)
	}

	return solution, nil
}

// parseProjectLine parses a solution project line
func parseProjectLine(line, solutionDir string) (ProjectReference, error) {
	// Project("{TypeGUID}") = "Name", "RelativePath", "{ProjectGUID}"

	// Extract type GUID
	typeGUIDStart := strings.Index(line, "{")
	typeGUIDEnd := strings.Index(line, "}")
	if typeGUIDStart == -1 || typeGUIDEnd == -1 {
		return ProjectReference{}, fmt.Errorf("invalid project line format")
	}
	typeGUID := line[typeGUIDStart+1 : typeGUIDEnd]

	// Extract name
	nameStart := strings.Index(line, "\"")
	if nameStart == -1 {
		return ProjectReference{}, fmt.Errorf("invalid project line format")
	}
	nameEnd := strings.Index(line[nameStart+1:], "\"")
	if nameEnd == -1 {
		return ProjectReference{}, fmt.Errorf("invalid project line format")
	}
	name := line[nameStart+1 : nameStart+1+nameEnd]

	// Extract path
	pathStart := strings.Index(line[nameStart+nameEnd+2:], "\"")
	if pathStart == -1 {
		return ProjectReference{}, fmt.Errorf("invalid project line format")
	}
	pathStart += nameStart + nameEnd + 2
	pathEnd := strings.Index(line[pathStart+1:], "\"")
	if pathEnd == -1 {
		return ProjectReference{}, fmt.Errorf("invalid project line format")
	}
	relativePath := line[pathStart+1 : pathStart+1+pathEnd]

	// Extract project GUID
	guidStart := strings.LastIndex(line, "{")
	guidEnd := strings.LastIndex(line, "}")
	if guidStart == -1 || guidEnd == -1 {
		return ProjectReference{}, fmt.Errorf("invalid project line format")
	}
	guid := line[guidStart+1 : guidEnd]

	// Convert to absolute path (handle both / and \ separators)
	relativePath = strings.ReplaceAll(relativePath, "\\", string(filepath.Separator))
	absolutePath := filepath.Join(solutionDir, relativePath)

	return ProjectReference{
		Name:         name,
		Path:         relativePath,
		AbsolutePath: absolutePath,
		GUID:         guid,
		TypeGUID:     typeGUID,
	}, nil
}

// GetProjectFiles returns absolute paths to all project files in the solution
func (s *Solution) GetProjectFiles() []string {
	paths := make([]string, len(s.Projects))
	for i, proj := range s.Projects {
		paths[i] = proj.AbsolutePath
	}
	return paths
}

// FindSolutionFile searches for a .sln file in the given directory
func FindSolutionFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var solutions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sln") {
			solutions = append(solutions, filepath.Join(dir, entry.Name()))
		}
	}

	if len(solutions) == 0 {
		return "", fmt.Errorf("no solution file found in directory")
	}

	if len(solutions) > 1 {
		return "", fmt.Errorf("multiple solution files found: %v", solutions)
	}

	return solutions[0], nil
}
```

**cmd/gonuget/solution/solution_test.go** (NEW):
```go
package solution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSolution(t *testing.T) {
	tmpDir := t.TempDir()
	slnPath := filepath.Join(tmpDir, "test.sln")

	slnContent := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project1", "Project1\Project1.csproj", "{12345678-1234-1234-1234-123456789ABC}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project2", "Project2\Project2.csproj", "{87654321-4321-4321-4321-CBA987654321}"
EndProject
Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
		Release|Any CPU = Release|Any CPU
	EndGlobalSection
EndGlobal`

	err := os.WriteFile(slnPath, []byte(slnContent), 0644)
	require.NoError(t, err)

	sln, err := LoadSolution(slnPath)
	require.NoError(t, err)
	assert.NotNil(t, sln)
	assert.Equal(t, slnPath, sln.Path)
	assert.Len(t, sln.Projects, 2)

	// Verify first project
	assert.Equal(t, "Project1", sln.Projects[0].Name)
	assert.Contains(t, sln.Projects[0].Path, "Project1.csproj")
	assert.Equal(t, "12345678-1234-1234-1234-123456789ABC", sln.Projects[0].GUID)

	// Verify second project
	assert.Equal(t, "Project2", sln.Projects[1].Name)
	assert.Contains(t, sln.Projects[1].Path, "Project2.csproj")
	assert.Equal(t, "87654321-4321-4321-4321-CBA987654321", sln.Projects[1].GUID)
}

func TestFindSolutionFile(t *testing.T) {
	tmpDir := t.TempDir()
	slnPath := filepath.Join(tmpDir, "test.sln")

	err := os.WriteFile(slnPath, []byte("solution content"), 0644)
	require.NoError(t, err)

	found, err := FindSolutionFile(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, slnPath, found)
}

func TestFindSolutionFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindSolutionFile(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no solution file found")
}

func TestFindSolutionFile_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "test1.sln"), []byte("sln1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "test2.sln"), []byte("sln2"), 0644)
	require.NoError(t, err)

	_, err = FindSolutionFile(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple solution files found")
}
```

### Files to Modify

**cmd/gonuget/commands/add_package.go** - Add solution support:
```go
import (
	"github.com/willibrandon/gonuget/cmd/gonuget/solution"
)

func runAddPackage(ctx context.Context, packageID string, opts *AddPackageOptions) error {
	projectPath := opts.ProjectPath

	// 1. Determine if input is solution or project
	if projectPath == "" {
		// Try to find project or solution in current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try solution first
		slnPath, err := solution.FindSolutionFile(currentDir)
		if err == nil {
			// Found solution, process all projects
			return addPackageToSolution(ctx, slnPath, packageID, opts)
		}

		// Fall back to project file
		foundPath, err := project.FindProjectFile(currentDir)
		if err != nil {
			return fmt.Errorf("failed to find project or solution file: %w", err)
		}
		projectPath = foundPath
	} else if strings.HasSuffix(projectPath, ".sln") {
		// Explicit solution file
		return addPackageToSolution(ctx, projectPath, packageID, opts)
	}

	// Single project file (existing logic continues)
	// ... rest of existing runAddPackage logic ...
}

// addPackageToSolution adds a package to all projects in a solution
func addPackageToSolution(ctx context.Context, slnPath string, packageID string, opts *AddPackageOptions) error {
	// Load solution
	sln, err := solution.LoadSolution(slnPath)
	if err != nil {
		return fmt.Errorf("failed to load solution: %w", err)
	}

	fmt.Printf("info : Adding package '%s' to solution '%s'\n", packageID, filepath.Base(slnPath))
	fmt.Printf("info : Found %d projects in solution\n", len(sln.Projects))

	// Track success/failure
	var successCount, failCount int
	var errors []error

	// Add to each project
	for _, proj := range sln.Projects {
		fmt.Printf("info : Adding to project '%s'...\n", proj.Name)

		// Create options for this project
		projOpts := *opts // Copy options
		projOpts.ProjectPath = proj.AbsolutePath

		// Add to project
		if err := runAddPackageToProject(ctx, packageID, &projOpts); err != nil {
			fmt.Printf("warn : Failed to add to project '%s': %v\n", proj.Name, err)
			failCount++
			errors = append(errors, fmt.Errorf("%s: %w", proj.Name, err))
		} else {
			fmt.Printf("info : Successfully added to project '%s'\n", proj.Name)
			successCount++
		}
	}

	// Summary
	fmt.Printf("\ninfo : Summary: %d succeeded, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("failed to add package to %d project(s)", failCount)
	}

	return nil
}

// runAddPackageToProject is the extracted single-project logic
func runAddPackageToProject(ctx context.Context, packageID string, opts *AddPackageOptions) error {
	// This is the existing runAddPackage logic for a single project
	// Move the current runAddPackage implementation here
	// ... (existing single-project logic) ...
}
```

### Testing

**cmd/gonuget/commands/add_package_test.go**:
```go
func TestRunAddPackage_Solution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create solution file
	slnPath := filepath.Join(tmpDir, "test.sln")
	slnContent := `Microsoft Visual Studio Solution File, Format Version 12.00
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project1", "Project1\Project1.csproj", "{12345678-1234-1234-1234-123456789ABC}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project2", "Project2\Project2.csproj", "{87654321-4321-4321-4321-CBA987654321}"
EndProject
Global
EndGlobal`
	err := os.WriteFile(slnPath, []byte(slnContent), 0644)
	require.NoError(t, err)

	// Create project directories and files
	proj1Dir := filepath.Join(tmpDir, "Project1")
	err = os.Mkdir(proj1Dir, 0755)
	require.NoError(t, err)
	proj1Path := filepath.Join(proj1Dir, "Project1.csproj")
	proj1Content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(proj1Path, []byte(proj1Content), 0644)
	require.NoError(t, err)

	proj2Dir := filepath.Join(tmpDir, "Project2")
	err = os.Mkdir(proj2Dir, 0755)
	require.NoError(t, err)
	proj2Path := filepath.Join(proj2Dir, "Project2.csproj")
	proj2Content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	err = os.WriteFile(proj2Path, []byte(proj2Content), 0644)
	require.NoError(t, err)

	// Add package to solution
	opts := &AddPackageOptions{
		ProjectPath: slnPath,
		Version:     "13.0.3",
		NoRestore:   true,
	}

	err = runAddPackage(context.Background(), "Newtonsoft.Json", opts)
	assert.NoError(t, err)

	// Verify package added to both projects
	proj1Data, err := os.ReadFile(proj1Path)
	require.NoError(t, err)
	assert.Contains(t, string(proj1Data), "Newtonsoft.Json")
	assert.Contains(t, string(proj1Data), "13.0.3")

	proj2Data, err := os.ReadFile(proj2Path)
	require.NoError(t, err)
	assert.Contains(t, string(proj2Data), "Newtonsoft.Json")
	assert.Contains(t, string(proj2Data), "13.0.3")
}
```

### Verification

```bash
# Create solution with multiple projects
mkdir test-sln && cd test-sln

# Create solution file
cat > test.sln <<'EOF'
Microsoft Visual Studio Solution File, Format Version 12.00
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project1", "Project1\Project1.csproj", "{12345678-1234-1234-1234-123456789ABC}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project2", "Project2\Project2.csproj", "{87654321-4321-4321-4321-CBA987654321}"
EndProject
Global
EndGlobal
EOF

# Create projects
mkdir Project1 Project2
cat > Project1/Project1.csproj <<EOF
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>
EOF

cat > Project2/Project2.csproj <<EOF
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>
EOF

# Add package to solution
./gonuget add test.sln package Newtonsoft.Json --version 13.0.3 --no-restore

# Verify package added to both projects
cat Project1/Project1.csproj
cat Project2/Project2.csproj

# Compare with dotnet
dotnet add test.sln package Serilog --version 3.1.1
```

### Success Criteria

- ✅ Solution file parsing works correctly
- ✅ Package added to all projects in solution
- ✅ Success/failure summary displayed
- ✅ Error handling for missing/invalid projects
- ✅ .sln extension detected and handled
- ✅ Unit tests pass with 90%+ coverage
- ✅ Manual testing shows identical behavior to `dotnet add`

---

## Chunk 18: CLI Interop Tests for CPM

**Objective**: Create CLI interop tests that validate gonuget's CPM implementation against `dotnet add package`.

**Time Estimate**: 4 hours

**Prerequisites**: Chunks 11-13 complete (CPM implementation)

### Background

CLI interop tests ensure 100% parity with dotnet by:
1. Running both `gonuget` and `dotnet` with identical inputs
2. Comparing outputs (XML files, console output, exit codes)
3. Validating XML formatting, structure, and content

**Test Infrastructure**: Already exists at `tests/cli-interop/`

### Reference Implementation

**Existing CLI Interop Tests**:
- Location: `/Users/brandon/src/gonuget/tests/cli-interop/GonugetCliInterop.Tests/`
- Pattern: `AddPackageTests.cs`, `RestoreTests.cs`

### Files to Create

**tests/cli-interop/GonugetCliInterop.Tests/AddPackageCpmTests.cs** (NEW):
```csharp
using System;
using System.IO;
using System.Xml.Linq;
using Xunit;
using Xunit.Abstractions;

namespace GonugetCliInterop.Tests
{
    /// <summary>
    /// CLI interop tests for Central Package Management (CPM) scenarios.
    /// These tests validate that gonuget add package behaves identically to dotnet add package
    /// when working with CPM-enabled projects.
    /// </summary>
    public class AddPackageCpmTests : CliInteropTestBase
    {
        public AddPackageCpmTests(ITestOutputHelper output) : base(output)
        {
        }

        [Fact]
        public void AddPackage_CpmEnabled_UpdatesDirectoryPackagesProps()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");
            var dppPath = Path.Combine(testDir, "Directory.Packages.props");

            // Create CPM-enabled project
            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            // Create Directory.Packages.props
            File.WriteAllText(dppPath, @"
<?xml version=""1.0"" encoding=""utf-8""?>
<Project>
  <ItemGroup>
  </ItemGroup>
</Project>");

            // Act
            var dotnetResult = RunDotnet($"add {csprojPath} package Newtonsoft.Json --version 13.0.3 --no-restore");

            // Save dotnet outputs
            var dotnetDpp = File.ReadAllText(dppPath);
            var dotnetCsproj = File.ReadAllText(csprojPath);

            // Reset files
            RestoreFiles(testDir);

            var gonugetResult = RunGonuget($"add package Newtonsoft.Json --version 13.0.3 --project {csprojPath} --no-restore");

            // Save gonuget outputs
            var gonugetDpp = File.ReadAllText(dppPath);
            var gonugetCsproj = File.ReadAllText(csprojPath);

            // Assert
            Assert.Equal(0, dotnetResult.ExitCode);
            Assert.Equal(0, gonugetResult.ExitCode);

            // Compare Directory.Packages.props
            AssertXmlEqual(dotnetDpp, gonugetDpp, "Directory.Packages.props");
            Assert.Contains("<PackageVersion Include=\"Newtonsoft.Json\" Version=\"13.0.3\"", gonugetDpp);

            // Compare .csproj
            AssertXmlEqual(dotnetCsproj, gonugetCsproj, "test.csproj");
        }

        [Fact]
        public void AddPackage_CpmEnabled_AddsPackageReferenceWithoutVersion()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");
            var dppPath = Path.Combine(testDir, "Directory.Packages.props");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            File.WriteAllText(dppPath, @"
<?xml version=""1.0"" encoding=""utf-8""?>
<Project>
  <ItemGroup>
  </ItemGroup>
</Project>");

            // Act
            var gonugetResult = RunGonuget($"add package Newtonsoft.Json --version 13.0.3 --project {csprojPath} --no-restore");

            // Assert
            Assert.Equal(0, gonugetResult.ExitCode);

            var csprojContent = File.ReadAllText(csprojPath);
            var csprojXml = XDocument.Parse(csprojContent);

            // Find PackageReference
            var packageRef = csprojXml.Descendants("PackageReference")
                .FirstOrDefault(e => e.Attribute("Include")?.Value == "Newtonsoft.Json");

            Assert.NotNull(packageRef);

            // CRITICAL: Version attribute must NOT exist in CPM mode
            Assert.Null(packageRef.Attribute("Version"));
        }

        [Fact]
        public void AddPackage_CpmEnabled_UpdatesExistingPackageVersion()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");
            var dppPath = Path.Combine(testDir, "Directory.Packages.props");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            // Directory.Packages.props with existing package at old version
            File.WriteAllText(dppPath, @"
<?xml version=""1.0"" encoding=""utf-8""?>
<Project>
  <ItemGroup>
    <PackageVersion Include=""Newtonsoft.Json"" Version=""12.0.0"" />
  </ItemGroup>
</Project>");

            // Act
            var dotnetResult = RunDotnet($"add {csprojPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
            var dotnetDpp = File.ReadAllText(dppPath);

            RestoreFiles(testDir);

            var gonugetResult = RunGonuget($"add package Newtonsoft.Json --version 13.0.3 --project {csprojPath} --no-restore");
            var gonugetDpp = File.ReadAllText(dppPath);

            // Assert
            Assert.Equal(0, dotnetResult.ExitCode);
            Assert.Equal(0, gonugetResult.ExitCode);

            AssertXmlEqual(dotnetDpp, gonugetDpp, "Directory.Packages.props");

            // Version should be updated
            Assert.Contains("13.0.3", gonugetDpp);
            Assert.DoesNotContain("12.0.0", gonugetDpp);
        }

        [Fact]
        public void AddPackage_CpmEnabled_ReusesExistingVersion()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");
            var dppPath = Path.Combine(testDir, "Directory.Packages.props");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            // Directory.Packages.props with existing version
            File.WriteAllText(dppPath, @"
<?xml version=""1.0"" encoding=""utf-8""?>
<Project>
  <ItemGroup>
    <PackageVersion Include=""Serilog"" Version=""3.1.1"" />
  </ItemGroup>
</Project>");

            // Act - Add without specifying version
            var dotnetResult = RunDotnet($"add {csprojPath} package Serilog --no-restore");
            var dotnetDpp = File.ReadAllText(dppPath);
            var dotnetCsproj = File.ReadAllText(csprojPath);

            RestoreFiles(testDir);

            var gonugetResult = RunGonuget($"add package Serilog --project {csprojPath} --no-restore");
            var gonugetDpp = File.ReadAllText(dppPath);
            var gonugetCsproj = File.ReadAllText(csprojPath);

            // Assert
            Assert.Equal(0, dotnetResult.ExitCode);
            Assert.Equal(0, gonugetResult.ExitCode);

            // Version should remain unchanged
            AssertXmlEqual(dotnetDpp, gonugetDpp, "Directory.Packages.props");
            Assert.Contains("3.1.1", gonugetDpp);

            // PackageReference should be added to csproj
            AssertXmlEqual(dotnetCsproj, gonugetCsproj, "test.csproj");
            Assert.Contains("Serilog", gonugetCsproj);
        }

        [Fact]
        public void AddPackage_CpmEnabled_MissingPropsFile_ReturnsError()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");

            // CPM enabled but NO Directory.Packages.props
            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            // Act
            var dotnetResult = RunDotnet($"add {csprojPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
            var gonugetResult = RunGonuget($"add package Newtonsoft.Json --version 13.0.3 --project {csprojPath} --no-restore");

            // Assert
            // Both should fail
            Assert.NotEqual(0, dotnetResult.ExitCode);
            Assert.NotEqual(0, gonugetResult.ExitCode);

            // Error messages should mention Directory.Packages.props
            Assert.Contains("Directory.Packages.props", gonugetResult.StdErr);
        }

        [Fact]
        public void AddPackage_CpmEnabled_XmlFormatMatches()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");
            var dppPath = Path.Combine(testDir, "Directory.Packages.props");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            File.WriteAllText(dppPath, @"
<?xml version=""1.0"" encoding=""utf-8""?>
<Project>
  <ItemGroup>
  </ItemGroup>
</Project>");

            // Act
            var dotnetResult = RunDotnet($"add {csprojPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
            var dotnetDpp = File.ReadAllBytes(dppPath);
            var dotnetCsproj = File.ReadAllBytes(csprojPath);

            RestoreFiles(testDir);

            var gonugetResult = RunGonuget($"add package Newtonsoft.Json --version 13.0.3 --project {csprojPath} --no-restore");
            var gonugetDpp = File.ReadAllBytes(dppPath);
            var gonugetCsproj = File.ReadAllBytes(csprojPath);

            // Assert XML formatting
            // Check UTF-8 BOM
            Assert.True(HasUtf8Bom(dotnetDpp));
            Assert.True(HasUtf8Bom(gonugetDpp));

            // Check indentation (2 spaces)
            var dotnetDppStr = Encoding.UTF8.GetString(dotnetDpp);
            var gonugetDppStr = Encoding.UTF8.GetString(gonugetDpp);

            AssertIndentationMatches(dotnetDppStr, gonugetDppStr);
        }

        [Fact]
        public void AddPackage_CpmEnabled_CaseInsensitiveMatching()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");
            var dppPath = Path.Combine(testDir, "Directory.Packages.props");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>");

            // Existing package with different case
            File.WriteAllText(dppPath, @"
<?xml version=""1.0"" encoding=""utf-8""?>
<Project>
  <ItemGroup>
    <PackageVersion Include=""Newtonsoft.Json"" Version=""12.0.0"" />
  </ItemGroup>
</Project>");

            // Act - Update with different case
            var gonugetResult = RunGonuget($"add package NEWTONSOFT.JSON --version 13.0.3 --project {csprojPath} --no-restore");

            // Assert
            Assert.Equal(0, gonugetResult.ExitCode);

            var dppContent = File.ReadAllText(dppPath);

            // Should update existing entry (case-insensitive match)
            Assert.Contains("13.0.3", dppContent);
            Assert.DoesNotContain("12.0.0", dppContent);

            // Original case should be preserved
            Assert.Contains("Newtonsoft.Json", dppContent);
        }

        #region Helper Methods

        private void RestoreFiles(string testDir)
        {
            var backupDir = Path.Combine(testDir, ".backup");
            if (Directory.Exists(backupDir))
            {
                foreach (var file in Directory.GetFiles(backupDir))
                {
                    var fileName = Path.GetFileName(file);
                    File.Copy(file, Path.Combine(testDir, fileName), overwrite: true);
                }
            }
        }

        private bool HasUtf8Bom(byte[] data)
        {
            return data.Length >= 3 &&
                   data[0] == 0xEF &&
                   data[1] == 0xBB &&
                   data[2] == 0xBF;
        }

        private void AssertIndentationMatches(string dotnetXml, string gonugetXml)
        {
            // Both should use 2-space indentation
            var dotnetLines = dotnetXml.Split('\n');
            var gonugetLines = gonugetXml.Split('\n');

            for (int i = 0; i < Math.Min(dotnetLines.Length, gonugetLines.Length); i++)
            {
                var dotnetIndent = GetIndentation(dotnetLines[i]);
                var gonugetIndent = GetIndentation(gonugetLines[i]);

                Assert.Equal(dotnetIndent, gonugetIndent,
                    $"Indentation mismatch on line {i + 1}");
            }
        }

        private int GetIndentation(string line)
        {
            int count = 0;
            foreach (char c in line)
            {
                if (c == ' ') count++;
                else if (c == '\t') count += 4; // Count tabs as 4 spaces
                else break;
            }
            return count;
        }

        #endregion
    }
}
```

### Verification

```bash
# Build and run CPM interop tests
cd tests/cli-interop
make build

# Run CPM tests
dotnet test --filter "FullyQualifiedName~AddPackageCpmTests"

# Expected output:
# Total tests: 7
# Passed: 7
# Failed: 0
```

### Success Criteria

- ✅ All 7 CPM interop tests pass
- ✅ XML output matches dotnet exactly
- ✅ UTF-8 BOM present in output files
- ✅ Indentation matches (2 spaces)
- ✅ PackageReference has NO Version attribute in CPM mode
- ✅ Directory.Packages.props correctly updated
- ✅ Case-insensitive matching works
- ✅ Error handling matches dotnet

---

## Chunk 19: CLI Interop Tests for Advanced Features

**Objective**: Test all M2.2 advanced features (Chunks 14-17) for parity with dotnet.

**Time Estimate**: 4 hours

**Prerequisites**: Chunks 14-18 complete

### Files to Create

**tests/cli-interop/GonugetCliInterop.Tests/AddPackageAdvancedTests.cs** (NEW):
```csharp
using System;
using System.IO;
using System.Linq;
using System.Xml.Linq;
using Xunit;
using Xunit.Abstractions;

namespace GonugetCliInterop.Tests
{
    /// <summary>
    /// CLI interop tests for advanced add package features:
    /// - Framework-specific references
    /// - Multi-TFM projects
    /// - Solution-level operations
    /// </summary>
    public class AddPackageAdvancedTests : CliInteropTestBase
    {
        public AddPackageAdvancedTests(ITestOutputHelper output) : base(output)
        {
        }

        [Fact]
        public void AddPackage_FrameworkSpecific_CreatesConditionalItemGroup()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48</TargetFrameworks>
  </PropertyGroup>
</Project>");

            // Act
            var dotnetResult = RunDotnet($"add {csprojPath} package System.Text.Json --version 8.0.0 --framework net8.0 --no-restore");
            var dotnetCsproj = File.ReadAllText(csprojPath);

            RestoreFile(csprojPath);

            var gonugetResult = RunGonuget($"add package System.Text.Json --version 8.0.0 --framework net8.0 --project {csprojPath} --no-restore");
            var gonugetCsproj = File.ReadAllText(csprojPath);

            // Assert
            Assert.Equal(0, dotnetResult.ExitCode);
            Assert.Equal(0, gonugetResult.ExitCode);

            AssertXmlEqual(dotnetCsproj, gonugetCsproj, "test.csproj");

            // Verify conditional ItemGroup exists
            var xml = XDocument.Parse(gonugetCsproj);
            var conditionalItemGroup = xml.Descendants("ItemGroup")
                .FirstOrDefault(e => e.Attribute("Condition")?.Value.Contains("net8.0") == true);

            Assert.NotNull(conditionalItemGroup);

            var packageRef = conditionalItemGroup.Descendants("PackageReference")
                .FirstOrDefault(e => e.Attribute("Include")?.Value == "System.Text.Json");

            Assert.NotNull(packageRef);
            Assert.Equal("8.0.0", packageRef.Attribute("Version")?.Value);
        }

        [Fact]
        public void AddPackage_MultiTFM_AddsToGlobalItemGroup()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net48;netstandard2.0</TargetFrameworks>
  </PropertyGroup>
</Project>");

            // Act - Add package compatible with all frameworks
            var dotnetResult = RunDotnet($"add {csprojPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
            var dotnetCsproj = File.ReadAllText(csprojPath);

            RestoreFile(csprojPath);

            var gonugetResult = RunGonuget($"add package Newtonsoft.Json --version 13.0.3 --project {csprojPath} --no-restore");
            var gonugetCsproj = File.ReadAllText(csprojPath);

            // Assert
            Assert.Equal(0, dotnetResult.ExitCode);
            Assert.Equal(0, gonugetResult.ExitCode);

            AssertXmlEqual(dotnetCsproj, gonugetCsproj, "test.csproj");

            // Should be in global ItemGroup (no Condition)
            var xml = XDocument.Parse(gonugetCsproj);
            var globalItemGroup = xml.Descendants("ItemGroup")
                .FirstOrDefault(e => e.Attribute("Condition") == null);

            Assert.NotNull(globalItemGroup);

            var packageRef = globalItemGroup.Descendants("PackageReference")
                .FirstOrDefault(e => e.Attribute("Include")?.Value == "Newtonsoft.Json");

            Assert.NotNull(packageRef);
        }

        [Fact]
        public void AddPackage_Solution_AddsToAllProjects()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var slnPath = Path.Combine(testDir, "test.sln");

            // Create solution
            File.WriteAllText(slnPath, @"
Microsoft Visual Studio Solution File, Format Version 12.00
Project(""{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}"") = ""Project1"", ""Project1\Project1.csproj"", ""{12345678-1234-1234-1234-123456789ABC}""
EndProject
Project(""{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}"") = ""Project2"", ""Project2\Project2.csproj"", ""{87654321-4321-4321-4321-CBA987654321}""
EndProject
Global
EndGlobal");

            // Create projects
            var proj1Dir = Path.Combine(testDir, "Project1");
            Directory.CreateDirectory(proj1Dir);
            var proj1Path = Path.Combine(proj1Dir, "Project1.csproj");
            File.WriteAllText(proj1Path, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>");

            var proj2Dir = Path.Combine(testDir, "Project2");
            Directory.CreateDirectory(proj2Dir);
            var proj2Path = Path.Combine(proj2Dir, "Project2.csproj");
            File.WriteAllText(proj2Path, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>");

            // Act
            var dotnetResult = RunDotnet($"add {slnPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
            var dotnetProj1 = File.ReadAllText(proj1Path);
            var dotnetProj2 = File.ReadAllText(proj2Path);

            RestoreFiles(testDir);

            var gonugetResult = RunGonuget($"add {slnPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
            var gonugetProj1 = File.ReadAllText(proj1Path);
            var gonugetProj2 = File.ReadAllText(proj2Path);

            // Assert
            Assert.Equal(0, dotnetResult.ExitCode);
            Assert.Equal(0, gonugetResult.ExitCode);

            // Both projects should have the package
            AssertXmlEqual(dotnetProj1, gonugetProj1, "Project1.csproj");
            AssertXmlEqual(dotnetProj2, gonugetProj2, "Project2.csproj");

            Assert.Contains("Newtonsoft.Json", gonugetProj1);
            Assert.Contains("13.0.3", gonugetProj1);
            Assert.Contains("Newtonsoft.Json", gonugetProj2);
            Assert.Contains("13.0.3", gonugetProj2);
        }

        [Fact]
        public void AddPackage_FrameworkSpecific_MultipleFrameworks()
        {
            // Arrange
            var testDir = CreateTestDirectory();
            var csprojPath = Path.Combine(testDir, "test.csproj");

            File.WriteAllText(csprojPath, @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFrameworks>net8.0;net7.0;net6.0</TargetFrameworks>
  </PropertyGroup>
</Project>");

            // Act - Add to multiple frameworks
            var gonugetResult1 = RunGonuget($"add package System.Text.Json --version 8.0.0 --framework net8.0 --project {csprojPath} --no-restore");
            var gonugetResult2 = RunGonuget($"add package System.Text.Json --version 7.0.0 --framework net7.0 --project {csprojPath} --no-restore");

            // Assert
            Assert.Equal(0, gonugetResult1.ExitCode);
            Assert.Equal(0, gonugetResult2.ExitCode);

            var csprojContent = File.ReadAllText(csprojPath);
            var xml = XDocument.Parse(csprojContent);

            // Should have 2 conditional ItemGroups
            var net8ItemGroup = xml.Descendants("ItemGroup")
                .FirstOrDefault(e => e.Attribute("Condition")?.Value.Contains("net8.0") == true);
            var net7ItemGroup = xml.Descendants("ItemGroup")
                .FirstOrDefault(e => e.Attribute("Condition")?.Value.Contains("net7.0") == true);

            Assert.NotNull(net8ItemGroup);
            Assert.NotNull(net7ItemGroup);

            // Each should have System.Text.Json with different versions
            var net8Pkg = net8ItemGroup.Descendants("PackageReference")
                .FirstOrDefault(e => e.Attribute("Include")?.Value == "System.Text.Json");
            Assert.Equal("8.0.0", net8Pkg?.Attribute("Version")?.Value);

            var net7Pkg = net7ItemGroup.Descendants("PackageReference")
                .FirstOrDefault(e => e.Attribute("Include")?.Value == "System.Text.Json");
            Assert.Equal("7.0.0", net7Pkg?.Attribute("Version")?.Value);
        }
    }
}
```

### Verification

```bash
# Build and run advanced feature tests
cd tests/cli-interop
make build

# Run all advanced tests
dotnet test --filter "FullyQualifiedName~AddPackageAdvancedTests"

# Expected output:
# Total tests: 5
# Passed: 5
# Failed: 0
```

### Success Criteria

- ✅ All 5 advanced feature interop tests pass
- ✅ Framework-specific references create correct conditional ItemGroups
- ✅ Multi-TFM projects handle package compatibility correctly
- ✅ Solution-level operations add packages to all projects
- ✅ Multiple framework-specific references work correctly
- ✅ XML output matches dotnet exactly

---

## M2.2 Success Criteria

**All Chunks 14-19 Complete:**

✅ **Framework-Specific References (Chunk 14)**:
- `--framework` flag works
- Conditional ItemGroups created with proper MSBuild syntax
- Multiple frameworks supported (OR conditions)

✅ **Transitive Dependency Resolution (Chunk 15)**:
- Transitive dependencies resolved and displayed
- Direct vs transitive categorization
- Integration with core/resolver

✅ **Multi-TFM Project Support (Chunk 16)**:
- Multi-targeting detected
- Package compatibility validated across frameworks
- Appropriate ItemGroup selection (global vs conditional)

✅ **Solution File Support (Chunk 17)**:
- .sln file parsing
- Package added to all projects in solution
- Success/failure summary

✅ **CLI Interop Tests for CPM (Chunk 18)**:
- 7 CPM interop tests passing
- XML formatting matches dotnet
- UTF-8 BOM verification

✅ **CLI Interop Tests for Advanced Features (Chunk 19)**:
- 5 advanced feature interop tests passing
- Framework-specific, multi-TFM, and solution tests
- 100% parity with dotnet

**Overall M2.2 Achievement**: 100% feature parity with `dotnet add package` for all advanced scenarios.

---

## Next Steps

After completing M2.2 Chunks 14-19:

1. **Run full CLI test suite**: `make test-cli`
2. **Run all CLI interop tests**: `make test-cli-interop`
3. **Update CLI milestones**: Mark M2.2 as complete
4. **Performance benchmarks**: Compare against dotnet (should be 15-17x faster)
5. **Move to Phase 3**: Dependency Resolution (restore enhancements)

**Total M2 Implementation Time**: 12-15 hours for Chunks 14-19
**Expected Completion**: All add package features at 100% parity with dotnet
