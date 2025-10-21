# Milestone 3: Package Operations - Continued 3 (Chunks 11-14)

**Status**: Not Started
**Chunks**: 11-14 (Asset Selection & Extraction)
**Estimated Time**: 6 hours

---

## M3.11: Asset Selection - Pattern Engine

**Estimated Time**: 2 hours
**Dependencies**: M1.4 (frameworks)

### Overview

Implement the pattern-based asset selection engine that matches package files to target frameworks and runtime identifiers using property-based pattern definitions.

### Files to Create/Modify

- `packaging/assets/patterns.go` - Pattern definition and matching
- `packaging/assets/conventions.go` - Managed code conventions
- `packaging/assets/patterns_test.go` - Pattern matching tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/ContentModel/PatternSet.cs`
- `NuGet.Packaging/ContentModel/PatternDefinition.cs`
- `NuGet.Packaging/ContentModel/ManagedCodeConventions.cs` (549+ lines)

**Pattern Definition** (PatternDefinition.cs):
```csharp
public class PatternDefinition {
    public string Pattern { get; }
    public IDictionary<string, object> Defaults { get; }
    public IDictionary<string, ContentPropertyDefinition> Table { get; }

    // Example: "lib/{tfm}/{assembly}"
    // Extracts tfm and assembly properties from path
}
```

**Pattern Matching** (ManagedCodeConventions.cs:480-510):
```csharp
private static bool TargetFrameworkName_CompatibilityTest(object criteria, object available) {
    var criteriaFrameworkName = criteria as NuGetFramework;
    var availableFrameworkName = available as NuGetFramework;

    if (Object.Equals(AnyFramework.AnyFramework, availableFrameworkName)) {
        return true;
    }

    return NuGetFrameworkUtility.IsCompatibleWithFallbackCheck(criteriaFrameworkName, availableFrameworkName);
}
```

### Implementation Details

**1. Pattern Structures**:

```go
// packaging/assets/patterns.go

package assets

import (
    "fmt"
    "path"
    "regexp"
    "strings"

    "github.com/willibrandon/gonuget/frameworks"
)

// PatternDefinition defines a file path pattern with placeholders
type PatternDefinition struct {
    // Pattern is the path pattern with placeholders
    // Example: "lib/{tfm}/{assembly}"
    Pattern string

    // Defaults provides default values for properties
    Defaults map[string]interface{}

    // PropertyTable maps property names to their definitions
    PropertyTable map[string]PropertyDefinition
}

// PropertyDefinition defines how a property is parsed and compared
type PropertyDefinition struct {
    // Name is the property name (e.g., "tfm", "rid", "assembly")
    Name string

    // Parser converts string to typed value
    Parser PropertyParser

    // CompatibilityTest checks if criterion is compatible with available value
    CompatibilityTest CompatibilityTestFunc

    // CompareTest determines which of two available values is nearer to criterion
    // Returns: -1 if first is nearer, 1 if second is nearer, 0 if equal
    CompareTest CompareTestFunc
}

// PropertyParser converts a string value to a typed value
type PropertyParser func(value string) (interface{}, error)

// CompatibilityTestFunc tests if a criterion is compatible with an available value
type CompatibilityTestFunc func(criterion, available interface{}) bool

// CompareTestFunc compares two available values against a criterion
type CompareTestFunc func(criterion, available1, available2 interface{}) int

// PatternMatch represents a matched pattern with extracted properties
type PatternMatch struct {
    Pattern    *PatternDefinition
    Path       string
    Properties map[string]interface{}
}

// PatternSet groups multiple patterns for a specific asset type
type PatternSet struct {
    // GroupPatterns match directory-level patterns
    GroupPatterns []*PatternDefinition

    // PathPatterns match file-level patterns
    PathPatterns []*PatternDefinition

    // Properties available for this pattern set
    Properties map[string]PropertyDefinition
}
```

**2. Pattern Matching Engine**:

```go
// MatchPattern matches a file path against a pattern definition
func MatchPattern(filePath string, pattern *PatternDefinition) (*PatternMatch, error) {
    // Normalize path
    normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
    normalizedPath = strings.ToLower(normalizedPath)

    // Convert pattern to regex
    regex, propertyNames := patternToRegex(pattern.Pattern)

    // Match path
    matches := regex.FindStringSubmatch(normalizedPath)
    if matches == nil {
        return nil, fmt.Errorf("path does not match pattern")
    }

    // Extract properties
    properties := make(map[string]interface{})

    // Apply defaults
    for key, value := range pattern.Defaults {
        properties[key] = value
    }

    // Extract matched properties
    for i, propName := range propertyNames {
        if i+1 < len(matches) {
            value := matches[i+1]

            // Parse property value
            if propDef, ok := pattern.PropertyTable[propName]; ok {
                if propDef.Parser != nil {
                    parsed, err := propDef.Parser(value)
                    if err != nil {
                        return nil, fmt.Errorf("parse property %s: %w", propName, err)
                    }
                    properties[propName] = parsed
                } else {
                    properties[propName] = value
                }
            } else {
                properties[propName] = value
            }
        }
    }

    return &PatternMatch{
        Pattern:    pattern,
        Path:       filePath,
        Properties: properties,
    }, nil
}

func patternToRegex(pattern string) (*regexp.Regexp, []string) {
    // Convert pattern with {property} placeholders to regex
    // Example: "lib/{tfm}/{assembly}" -> "lib/([^/]+)/([^/]+)"

    var propertyNames []string
    regexPattern := pattern

    // Find all {property} placeholders
    placeholderRegex := regexp.MustCompile(`\{([^}?]+)\??}`)
    matches := placeholderRegex.FindAllStringSubmatch(pattern, -1)

    for _, match := range matches {
        propName := match[1]
        propertyNames = append(propertyNames, propName)

        // Replace {property} with capture group
        // {property?} means optional
        if strings.HasSuffix(match[0], "?}") {
            regexPattern = strings.Replace(regexPattern, match[0], "([^/]*)", 1)
        } else {
            regexPattern = strings.Replace(regexPattern, match[0], "([^/]+)", 1)
        }
    }

    // Escape other regex special characters
    regexPattern = "^" + regexPattern + "$"

    return regexp.MustCompile(strings.ToLower(regexPattern)), propertyNames
}

// FindBestMatch finds the best matching pattern for criterion
func FindBestMatch(criterion map[string]interface{}, matches []*PatternMatch, properties map[string]PropertyDefinition) *PatternMatch {
    if len(matches) == 0 {
        return nil
    }

    if len(matches) == 1 {
        return matches[0]
    }

    // Filter to compatible matches
    compatible := filterCompatible(criterion, matches, properties)
    if len(compatible) == 0 {
        return nil
    }

    // Find nearest match
    best := compatible[0]
    for i := 1; i < len(compatible); i++ {
        if isNearer(criterion, compatible[i], best, properties) {
            best = compatible[i]
        }
    }

    return best
}

func filterCompatible(criterion map[string]interface{}, matches []*PatternMatch, properties map[string]PropertyDefinition) []*PatternMatch {
    var compatible []*PatternMatch

    for _, match := range matches {
        isCompatible := true

        for propName, criterionValue := range criterion {
            propDef, ok := properties[propName]
            if !ok {
                continue
            }

            matchValue, ok := match.Properties[propName]
            if !ok {
                continue
            }

            // Test compatibility
            if propDef.CompatibilityTest != nil {
                if !propDef.CompatibilityTest(criterionValue, matchValue) {
                    isCompatible = false
                    break
                }
            }
        }

        if isCompatible {
            compatible = append(compatible, match)
        }
    }

    return compatible
}

func isNearer(criterion map[string]interface{}, candidate, current *PatternMatch, properties map[string]PropertyDefinition) bool {
    // Compare all properties
    for propName, criterionValue := range criterion {
        propDef, ok := properties[propName]
        if !ok || propDef.CompareTest == nil {
            continue
        }

        candidateValue := candidate.Properties[propName]
        currentValue := current.Properties[propName]

        result := propDef.CompareTest(criterionValue, candidateValue, currentValue)
        if result < 0 {
            return true // Candidate is nearer
        } else if result > 0 {
            return false // Current is nearer
        }
        // Equal, continue to next property
    }

    return false
}
```

**3. Managed Code Conventions**:

```go
// packaging/assets/conventions.go

package assets

import (
    "strings"

    "github.com/willibrandon/gonuget/frameworks"
)

// ManagedCodeConventions defines standard .NET package conventions
type ManagedCodeConventions struct {
    Properties map[string]PropertyDefinition

    // Asset pattern sets
    RuntimeAssemblies           *PatternSet
    CompileTimeAssemblies       *PatternSet
    ResourceAssemblies          *PatternSet
    NativeLibraries             *PatternSet
    MSBuildFiles                *PatternSet
    MSBuildTransitiveFiles      *PatternSet
    ContentFiles                *PatternSet
    ToolsAssemblies             *PatternSet
}

// NewManagedCodeConventions creates standard managed code conventions
// Reference: ManagedCodeConventions.cs
func NewManagedCodeConventions() *ManagedCodeConventions {
    conventions := &ManagedCodeConventions{
        Properties: make(map[string]PropertyDefinition),
    }

    // Define properties
    conventions.defineProperties()

    // Define pattern sets
    conventions.definePatternSets()

    return conventions
}

func (c *ManagedCodeConventions) defineProperties() {
    // Target Framework Moniker (TFM) property
    c.Properties["tfm"] = PropertyDefinition{
        Name:              "tfm",
        Parser:            parseTFM,
        CompatibilityTest: tfmCompatibilityTest,
        CompareTest:       tfmCompareTest,
    }

    // Runtime Identifier (RID) property
    c.Properties["rid"] = PropertyDefinition{
        Name:              "rid",
        Parser:            parseRID,
        CompatibilityTest: ridCompatibilityTest,
        CompareTest:       ridCompareTest,
    }

    // Assembly name property (simple string)
    c.Properties["assembly"] = PropertyDefinition{
        Name:   "assembly",
        Parser: func(value string) (interface{}, error) { return value, nil },
    }

    // Locale property
    c.Properties["locale"] = PropertyDefinition{
        Name:   "locale",
        Parser: func(value string) (interface{}, error) { return value, nil },
    }

    // Any property (matches anything)
    c.Properties["any"] = PropertyDefinition{
        Name:   "any",
        Parser: func(value string) (interface{}, error) { return value, nil },
    }
}

func parseTFM(value string) (interface{}, error) {
    return frameworks.ParseFramework(value)
}

func parseRID(value string) (interface{}, error) {
    // RID parsing will be implemented in M3.13
    return value, nil
}

// tfmCompatibilityTest checks if criterion TFM is compatible with available TFM
// Reference: ManagedCodeConventions.cs TargetFrameworkName_CompatibilityTest
func tfmCompatibilityTest(criterion, available interface{}) bool {
    criterionFW, ok1 := criterion.(*frameworks.NuGetFramework)
    availableFW, ok2 := available.(*frameworks.NuGetFramework)

    if !ok1 || !ok2 {
        return false
    }

    // AnyFramework is always compatible
    if availableFW.IsAny() {
        return true
    }

    // Use framework compatibility check with fallback
    return frameworks.IsCompatible(criterionFW, availableFW)
}

// tfmCompareTest determines which TFM is nearest
// Reference: ManagedCodeConventions.cs TargetFrameworkName_NearestCompareTest
func tfmCompareTest(criterion, available1, available2 interface{}) int {
    criterionFW, ok := criterion.(*frameworks.NuGetFramework)
    if !ok {
        return 0
    }

    fw1, ok1 := available1.(*frameworks.NuGetFramework)
    fw2, ok2 := available2.(*frameworks.NuGetFramework)

    if !ok1 || !ok2 {
        return 0
    }

    // Use framework reducer to find nearest
    reducer := frameworks.NewFrameworkReducer()
    nearest := reducer.GetNearest(criterionFW, []*frameworks.NuGetFramework{fw1, fw2})

    if nearest == nil {
        return 0
    }

    if nearest == fw1 {
        return -1
    } else if nearest == fw2 {
        return 1
    }

    return 0
}

func ridCompatibilityTest(criterion, available interface{}) bool {
    // RID compatibility will be implemented in M3.13
    // For now, exact match
    return criterion == available
}

func ridCompareTest(criterion, available1, available2 interface{}) int {
    // RID comparison will be implemented in M3.13
    return 0
}

func (c *ManagedCodeConventions) definePatternSets() {
    // Runtime assemblies (lib/ folder)
    // Reference: ManagedCodeConventions.cs RuntimeAssemblies
    c.RuntimeAssemblies = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "runtimes/{rid}/lib/{tfm}/{any?}",
                PropertyTable: c.Properties,
            },
            {
                Pattern:       "lib/{tfm}/{any?}",
                PropertyTable: c.Properties,
            },
            {
                Pattern: "lib/{assembly?}",
                Defaults: map[string]interface{}{
                    "tfm": &frameworks.NuGetFramework{Framework: ".NETFramework", Version: frameworks.Version{Major: 0}},
                },
                PropertyTable: c.Properties,
            },
        },
        PathPatterns: []*PatternDefinition{
            {
                Pattern:       "runtimes/{rid}/lib/{tfm}/{assembly}",
                PropertyTable: c.Properties,
            },
            {
                Pattern:       "lib/{tfm}/{assembly}",
                PropertyTable: c.Properties,
            },
            {
                Pattern: "lib/{assembly}",
                Defaults: map[string]interface{}{
                    "tfm": &frameworks.NuGetFramework{Framework: ".NETFramework", Version: frameworks.Version{Major: 0}},
                },
                PropertyTable: c.Properties,
            },
        },
    }

    // Compile-time assemblies (ref/ folder)
    c.CompileTimeAssemblies = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "ref/{tfm}/{any?}",
                PropertyTable: c.Properties,
            },
        },
        PathPatterns: []*PatternDefinition{
            {
                Pattern:       "ref/{tfm}/{assembly}",
                PropertyTable: c.Properties,
            },
        },
    }

    // Resource assemblies (satellite assemblies)
    c.ResourceAssemblies = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "runtimes/{rid}/lib/{tfm}/{locale}/{any?}",
                PropertyTable: c.Properties,
            },
            {
                Pattern:       "lib/{tfm}/{locale}/{any?}",
                PropertyTable: c.Properties,
            },
        },
        PathPatterns: []*PatternDefinition{
            {
                Pattern:       "runtimes/{rid}/lib/{tfm}/{locale}/{assembly}",
                PropertyTable: c.Properties,
            },
            {
                Pattern:       "lib/{tfm}/{locale}/{assembly}",
                PropertyTable: c.Properties,
            },
        },
    }

    // Native libraries
    c.NativeLibraries = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "runtimes/{rid}/native/{any?}",
                PropertyTable: c.Properties,
            },
        },
        PathPatterns: []*PatternDefinition{
            {
                Pattern:       "runtimes/{rid}/native/{any}",
                PropertyTable: c.Properties,
            },
        },
    }

    // MSBuild files (build/ folder)
    c.MSBuildFiles = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "build/{tfm}/{any?}",
                PropertyTable: c.Properties,
            },
            {
                Pattern: "build/{any?}",
                Defaults: map[string]interface{}{
                    "tfm": &frameworks.NuGetFramework{Framework: "any", Version: frameworks.Version{Major: 0}},
                },
                PropertyTable: c.Properties,
            },
        },
    }

    // MSBuild transitive files (buildTransitive/ folder)
    c.MSBuildTransitiveFiles = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "buildTransitive/{tfm}/{any?}",
                PropertyTable: c.Properties,
            },
        },
    }

    // Content files
    c.ContentFiles = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "contentFiles/{any}/{tfm}/{any?}",
                PropertyTable: c.Properties,
            },
        },
    }

    // Tools assemblies
    c.ToolsAssemblies = &PatternSet{
        Properties: c.Properties,
        GroupPatterns: []*PatternDefinition{
            {
                Pattern:       "tools/{any?}",
                PropertyTable: c.Properties,
            },
        },
    }
}
```

### Verification Steps

```bash
# 1. Run pattern matching tests
go test ./packaging/assets -v -run TestPatternMatch

# 2. Test TFM compatibility
go test ./packaging/assets -v -run TestTFMCompatibility

# 3. Test pattern sets
go test ./packaging/assets -v -run TestPatternSets

# 4. Test with real package paths
go test ./packaging/assets -v -run TestRealPackagePaths

# 5. Check test coverage
go test ./packaging/assets -cover
```

### Acceptance Criteria

- [ ] Pattern definition with placeholders ({tfm}, {rid}, {assembly})
- [ ] Pattern to regex conversion
- [ ] Property extraction from paths
- [ ] Property parsing (TFM, RID, strings)
- [ ] Compatibility testing per property
- [ ] Nearest match selection
- [ ] Managed code conventions (lib/, ref/, runtimes/)
- [ ] Pattern sets for all asset types
- [ ] Default value support
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement asset selection pattern engine

Add pattern-based asset selection:
- Pattern definitions with property placeholders
- Regex-based pattern matching
- Property extraction and parsing
- TFM compatibility and comparison
- Managed code conventions (lib/, ref/, build/, runtimes/)
- Pattern sets for all asset types

Reference: NuGet.Packaging/ContentModel/PatternSet.cs
Reference: ManagedCodeConventions.cs
```

---

## M3.12: Asset Selection - Framework Resolution

**Estimated Time**: 1.5 hours
**Dependencies**: M3.11, M1.4

### Overview

Implement framework-based asset selection that uses the pattern engine and framework reducer to select the most appropriate assets for a target framework.

### Files to Create/Modify

- `packaging/assets/selector.go` - Asset selection logic
- `packaging/assets/selector_test.go` - Asset selection tests
- `packaging/reader.go` - Add asset selection methods

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/LockFileBuilder.cs` GetLibItems, GetRefItems
- `NuGet.Frameworks/FrameworkReducer.cs` GetNearest

### Implementation Details

**1. Asset Selector**:

```go
// packaging/assets/selector.go

package assets

import (
    "strings"

    "github.com/willibrandon/gonuget/frameworks"
)

// AssetSelector selects package assets based on target framework
type AssetSelector struct {
    conventions *ManagedCodeConventions
}

// NewAssetSelector creates a new asset selector
func NewAssetSelector() *AssetSelector {
    return &AssetSelector{
        conventions: NewManagedCodeConventions(),
    }
}

// SelectionCriteria defines asset selection criteria
type SelectionCriteria struct {
    TargetFramework  *frameworks.NuGetFramework
    RuntimeIdentifier string
}

// AssetGroup represents a group of assets for a specific framework/RID
type AssetGroup struct {
    TargetFramework  *frameworks.NuGetFramework
    RuntimeIdentifier string
    Items            []string
}

// SelectRuntimeAssemblies selects runtime assemblies (lib/ folder) for target framework
func (s *AssetSelector) SelectRuntimeAssemblies(files []string, criteria SelectionCriteria) *AssetGroup {
    return s.selectAssets(files, s.conventions.RuntimeAssemblies, criteria)
}

// SelectCompileAssemblies selects compile-time assemblies (ref/ folder)
func (s *AssetSelector) SelectCompileAssemblies(files []string, criteria SelectionCriteria) *AssetGroup {
    return s.selectAssets(files, s.conventions.CompileTimeAssemblies, criteria)
}

// SelectNativeLibraries selects native libraries for RID
func (s *AssetSelector) SelectNativeLibraries(files []string, criteria SelectionCriteria) *AssetGroup {
    return s.selectAssets(files, s.conventions.NativeLibraries, criteria)
}

// SelectBuildFiles selects MSBuild files
func (s *AssetSelector) SelectBuildFiles(files []string, criteria SelectionCriteria) *AssetGroup {
    return s.selectAssets(files, s.conventions.MSBuildFiles, criteria)
}

func (s *AssetSelector) selectAssets(files []string, patternSet *PatternSet, criteria SelectionCriteria) *AssetGroup {
    // Build criterion map
    criterionMap := make(map[string]interface{})
    if criteria.TargetFramework != nil {
        criterionMap["tfm"] = criteria.TargetFramework
    }
    if criteria.RuntimeIdentifier != "" {
        criterionMap["rid"] = criteria.RuntimeIdentifier
    }

    // Match all files against patterns
    var matches []*PatternMatch
    for _, file := range files {
        for _, pattern := range patternSet.PathPatterns {
            match, err := MatchPattern(file, pattern)
            if err == nil {
                matches = append(matches, match)
            }
        }
    }

    if len(matches) == 0 {
        return &AssetGroup{
            TargetFramework: criteria.TargetFramework,
            RuntimeIdentifier: criteria.RuntimeIdentifier,
            Items: []string{},
        }
    }

    // Group matches by framework
    groups := groupMatchesByFramework(matches)

    // Find best matching group
    bestGroup := selectBestGroup(groups, criterionMap, patternSet.Properties)

    if bestGroup == nil {
        return &AssetGroup{
            TargetFramework: criteria.TargetFramework,
            RuntimeIdentifier: criteria.RuntimeIdentifier,
            Items: []string{},
        }
    }

    return bestGroup
}

func groupMatchesByFramework(matches []*PatternMatch) map[string][]*PatternMatch {
    groups := make(map[string][]*PatternMatch)

    for _, match := range matches {
        // Get framework from match
        var groupKey string
        if tfm, ok := match.Properties["tfm"]; ok {
            if fw, ok := tfm.(*frameworks.NuGetFramework); ok {
                groupKey = fw.GetShortFolderName()
            }
        }

        if groupKey == "" {
            groupKey = "any"
        }

        groups[groupKey] = append(groups[groupKey], match)
    }

    return groups
}

func selectBestGroup(groups map[string][]*PatternMatch, criterion map[string]interface{}, properties map[string]PropertyDefinition) *AssetGroup {
    if len(groups) == 0 {
        return nil
    }

    // Get criterion framework
    var criterionFW *frameworks.NuGetFramework
    if tfm, ok := criterion["tfm"]; ok {
        criterionFW, _ = tfm.(*frameworks.NuGetFramework)
    }

    // Collect framework keys
    var frameworks []*frameworks.NuGetFramework
    groupByFramework := make(map[string][]*PatternMatch)

    for key, matches := range groups {
        // Extract framework from first match
        if len(matches) > 0 {
            if tfm, ok := matches[0].Properties["tfm"]; ok {
                if fw, ok := tfm.(*frameworks.NuGetFramework); ok {
                    frameworks = append(frameworks, fw)
                    groupByFramework[fw.GetShortFolderName()] = matches
                }
            }
        }
    }

    // Find nearest framework
    var nearestFW *frameworks.NuGetFramework
    if criterionFW != nil {
        reducer := frameworks.NewFrameworkReducer()
        nearestFW = reducer.GetNearest(criterionFW, frameworks)
    } else if len(frameworks) > 0 {
        nearestFW = frameworks[0]
    }

    if nearestFW == nil {
        return nil
    }

    // Get matches for nearest framework
    matches := groupByFramework[nearestFW.GetShortFolderName()]
    if len(matches) == 0 {
        return nil
    }

    // Extract file paths
    var items []string
    for _, match := range matches {
        items = append(items, match.Path)
    }

    return &AssetGroup{
        TargetFramework: nearestFW,
        Items:           items,
    }
}

// GetLibItems is a helper to get runtime assemblies with DLL/EXE filtering
func (s *AssetSelector) GetLibItems(files []string, targetFramework *frameworks.NuGetFramework) []string {
    criteria := SelectionCriteria{
        TargetFramework: targetFramework,
    }

    group := s.SelectRuntimeAssemblies(files, criteria)
    if group == nil {
        return []string{}
    }

    // Filter to DLL/EXE files
    var assemblies []string
    for _, item := range group.Items {
        ext := strings.ToLower(strings.TrimPrefix(path.Ext(item), "."))
        if ext == "dll" || ext == "exe" || ext == "winmd" {
            assemblies = append(assemblies, item)
        }
    }

    return assemblies
}

// GetRefItems is a helper to get compile-time assemblies
func (s *AssetSelector) GetRefItems(files []string, targetFramework *frameworks.NuGetFramework) []string {
    criteria := SelectionCriteria{
        TargetFramework: targetFramework,
    }

    group := s.SelectCompileAssemblies(files, criteria)
    if group == nil {
        return []string{}
    }

    return group.Items
}
```

**2. Add to PackageReader**:

```go
// packaging/reader.go additions

import (
    "github.com/willibrandon/gonuget/packaging/assets"
)

// SelectLibItems selects runtime assemblies for a target framework
func (r *PackageReader) SelectLibItems(targetFramework *frameworks.NuGetFramework) ([]string, error) {
    files := r.GetPackageFiles()

    selector := assets.NewAssetSelector()
    return selector.GetLibItems(files, targetFramework), nil
}

// SelectRefItems selects compile-time assemblies for a target framework
func (r *PackageReader) SelectRefItems(targetFramework *frameworks.NuGetFramework) ([]string, error) {
    files := r.GetPackageFiles()

    selector := assets.NewAssetSelector()
    return selector.GetRefItems(files, targetFramework), nil
}

// SelectBestLibItems selects the best runtime assemblies using framework reducer
func (r *PackageReader) SelectBestLibItems(targetFramework *frameworks.NuGetFramework) ([]string, error) {
    return r.SelectLibItems(targetFramework)
}
```

### Verification Steps

```bash
# 1. Run asset selection tests
go test ./packaging/assets -v -run TestAssetSelection

# 2. Test framework selection
go test ./packaging/assets -v -run TestFrameworkSelection

# 3. Test with real packages
go test ./packaging/assets -v -run TestSelectFromRealPackage

# 4. Test edge cases
go test ./packaging/assets -v -run TestAssetSelectionEdgeCases

# 5. Check test coverage
go test ./packaging/assets -cover
```

### Acceptance Criteria

- [ ] Select runtime assemblies for target framework
- [ ] Select compile-time assemblies for target framework
- [ ] Select native libraries for RID
- [ ] Select MSBuild files
- [ ] Use framework reducer for nearest match
- [ ] Group assets by framework
- [ ] Filter to DLL/EXE files
- [ ] Handle missing assets gracefully
- [ ] Integration with PackageReader
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement framework-based asset selection

Add asset selection for target frameworks:
- Select runtime assemblies (lib/ folder)
- Select compile-time assemblies (ref/ folder)
- Select native libraries by RID
- Select MSBuild files
- Use FrameworkReducer for nearest match
- Integration with PackageReader

Reference: LockFileBuilder.cs GetLibItems
```

---

## M3.13: Asset Selection - RID Resolution

**Estimated Time**: 1.5 hours
**Dependencies**: M3.12

### Overview

Implement Runtime Identifier (RID) resolution for platform-specific asset selection using the RID graph.

### Files to Create/Modify

- `packaging/assets/rid.go` - RID parsing and resolution
- `packaging/assets/ridgraph.go` - RID graph implementation
- `packaging/assets/rid_test.go` - RID tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.RuntimeModel/RuntimeGraph.cs`
- `NuGet.RuntimeModel/RuntimeDescription.cs`
- RID Catalog: https://learn.microsoft.com/en-us/dotnet/core/rid-catalog

**RID Format**: `<os>.<version>-<architecture>-<additional qualifiers>`
Examples: `win10-x64`, `linux-x64`, `osx.10.12-x64`

### Implementation Details

**1. RID Structure**:

```go
// packaging/assets/rid.go

package assets

import (
    "fmt"
    "strings"
)

// RuntimeIdentifier represents a parsed RID
type RuntimeIdentifier struct {
    // Raw RID string
    RID string

    // Parsed components
    OS           string
    Version      string
    Architecture string
    Qualifiers   []string
}

// ParseRID parses a runtime identifier string
func ParseRID(rid string) (*RuntimeIdentifier, error) {
    if rid == "" {
        return nil, fmt.Errorf("RID cannot be empty")
    }

    r := &RuntimeIdentifier{
        RID: rid,
    }

    // Split by hyphen
    parts := strings.Split(rid, "-")

    if len(parts) == 0 {
        return nil, fmt.Errorf("invalid RID format")
    }

    // First part is OS (potentially with version)
    osPart := parts[0]
    if strings.Contains(osPart, ".") {
        // OS with version: "osx.10.12"
        osParts := strings.SplitN(osPart, ".", 2)
        r.OS = osParts[0]
        r.Version = osParts[1]
    } else {
        r.OS = osPart
    }

    // Remaining parts are architecture and qualifiers
    if len(parts) > 1 {
        r.Architecture = parts[1]
    }

    if len(parts) > 2 {
        r.Qualifiers = parts[2:]
    }

    return r, nil
}

// String returns the RID string
func (r *RuntimeIdentifier) String() string {
    return r.RID
}

// IsCompatible checks if this RID is compatible with another
func (r *RuntimeIdentifier) IsCompatible(other *RuntimeIdentifier, graph *RuntimeGraph) bool {
    // Exact match
    if r.RID == other.RID {
        return true
    }

    // Check graph for compatibility
    if graph != nil {
        return graph.AreCompatible(r.RID, other.RID)
    }

    // Fallback: OS and architecture must match
    return r.OS == other.OS && r.Architecture == other.Architecture
}
```

**2. RID Graph**:

```go
// packaging/assets/ridgraph.go

package assets

import (
    "encoding/json"
    "fmt"
)

// RuntimeGraph represents the RID compatibility graph
type RuntimeGraph struct {
    // Runtimes maps RID to its description
    Runtimes map[string]*RuntimeDescription
}

// RuntimeDescription describes a runtime and its compatible runtimes
type RuntimeDescription struct {
    RID     string
    Imports []string // Compatible RIDs (less specific)
}

// NewRuntimeGraph creates an empty runtime graph
func NewRuntimeGraph() *RuntimeGraph {
    return &RuntimeGraph{
        Runtimes: make(map[string]*RuntimeDescription),
    }
}

// LoadDefaultRuntimeGraph loads the default .NET RID graph
func LoadDefaultRuntimeGraph() *RuntimeGraph {
    graph := NewRuntimeGraph()

    // Add common RIDs
    // Reference: https://learn.microsoft.com/en-us/dotnet/core/rid-catalog

    // Windows
    graph.AddRuntime("win", nil)
    graph.AddRuntime("win-x86", []string{"win"})
    graph.AddRuntime("win-x64", []string{"win"})
    graph.AddRuntime("win-arm64", []string{"win"})
    graph.AddRuntime("win10", []string{"win"})
    graph.AddRuntime("win10-x64", []string{"win10", "win-x64"})
    graph.AddRuntime("win10-x86", []string{"win10", "win-x86"})
    graph.AddRuntime("win10-arm64", []string{"win10", "win-arm64"})

    // Linux
    graph.AddRuntime("linux", nil)
    graph.AddRuntime("linux-x64", []string{"linux"})
    graph.AddRuntime("linux-arm", []string{"linux"})
    graph.AddRuntime("linux-arm64", []string{"linux"})

    // Ubuntu
    graph.AddRuntime("ubuntu", []string{"linux"})
    graph.AddRuntime("ubuntu-x64", []string{"ubuntu", "linux-x64"})
    graph.AddRuntime("ubuntu.20.04-x64", []string{"ubuntu-x64", "ubuntu", "linux-x64"})
    graph.AddRuntime("ubuntu.22.04-x64", []string{"ubuntu-x64", "ubuntu", "linux-x64"})

    // macOS
    graph.AddRuntime("osx", nil)
    graph.AddRuntime("osx-x64", []string{"osx"})
    graph.AddRuntime("osx-arm64", []string{"osx"})
    graph.AddRuntime("osx.10.12-x64", []string{"osx-x64", "osx"})
    graph.AddRuntime("osx.11-x64", []string{"osx-x64", "osx"})
    graph.AddRuntime("osx.12-arm64", []string{"osx-arm64", "osx"})

    return graph
}

// AddRuntime adds a runtime to the graph
func (g *RuntimeGraph) AddRuntime(rid string, imports []string) {
    g.Runtimes[rid] = &RuntimeDescription{
        RID:     rid,
        Imports: imports,
    }
}

// AreCompatible checks if targetRID is compatible with packageRID
func (g *RuntimeGraph) AreCompatible(targetRID, packageRID string) bool {
    // Exact match
    if targetRID == packageRID {
        return true
    }

    // Get target runtime
    target, ok := g.Runtimes[targetRID]
    if !ok {
        return false
    }

    // Check if packageRID is in target's imports (transitively)
    return g.isInImports(target, packageRID, make(map[string]bool))
}

func (g *RuntimeGraph) isInImports(runtime *RuntimeDescription, searchRID string, visited map[string]bool) bool {
    // Avoid cycles
    if visited[runtime.RID] {
        return false
    }
    visited[runtime.RID] = true

    // Check direct imports
    for _, importRID := range runtime.Imports {
        if importRID == searchRID {
            return true
        }

        // Check transitively
        if importRuntime, ok := g.Runtimes[importRID]; ok {
            if g.isInImports(importRuntime, searchRID, visited) {
                return true
            }
        }
    }

    return false
}

// GetAllCompatibleRIDs returns all RIDs compatible with the target RID
func (g *RuntimeGraph) GetAllCompatibleRIDs(targetRID string) []string {
    var compatible []string

    target, ok := g.Runtimes[targetRID]
    if !ok {
        return compatible
    }

    visited := make(map[string]bool)
    g.collectImports(target, &compatible, visited)

    return compatible
}

func (g *RuntimeGraph) collectImports(runtime *RuntimeDescription, result *[]string, visited map[string]bool) {
    if visited[runtime.RID] {
        return
    }
    visited[runtime.RID] = true

    *result = append(*result, runtime.RID)

    for _, importRID := range runtime.Imports {
        if importRuntime, ok := g.Runtimes[importRID]; ok {
            g.collectImports(importRuntime, result, visited)
        }
    }
}

// LoadFromJSON loads a runtime graph from JSON
func LoadFromJSON(data []byte) (*RuntimeGraph, error) {
    var graphData struct {
        Runtimes map[string]struct {
            Imports []string `json:"#import"`
        } `json:"runtimes"`
    }

    if err := json.Unmarshal(data, &graphData); err != nil {
        return nil, fmt.Errorf("unmarshal runtime graph: %w", err)
    }

    graph := NewRuntimeGraph()
    for rid, data := range graphData.Runtimes {
        graph.AddRuntime(rid, data.Imports)
    }

    return graph, nil
}
```

**3. Update Asset Selector**:

```go
// packaging/assets/selector.go updates

// SelectRuntimeAssembliesWithRID selects runtime assemblies for TFM and RID
func (s *AssetSelector) SelectRuntimeAssembliesWithRID(files []string, criteria SelectionCriteria) *AssetGroup {
    // First try RID-specific runtimes/ folder
    if criteria.RuntimeIdentifier != "" {
        ridGroup := s.selectRIDSpecificAssets(files, criteria)
        if ridGroup != nil && len(ridGroup.Items) > 0 {
            return ridGroup
        }
    }

    // Fall back to lib/ folder (RID-agnostic)
    return s.SelectRuntimeAssemblies(files, criteria)
}

func (s *AssetSelector) selectRIDSpecificAssets(files []string, criteria SelectionCriteria) *AssetGroup {
    // Match runtimes/{rid}/lib/{tfm}/ pattern
    var matches []*PatternMatch

    for _, file := range files {
        if !strings.HasPrefix(strings.ToLower(file), "runtimes/") {
            continue
        }

        for _, pattern := range s.conventions.RuntimeAssemblies.PathPatterns {
            if !strings.Contains(pattern.Pattern, "{rid}") {
                continue
            }

            match, err := MatchPattern(file, pattern)
            if err == nil {
                // Filter by RID
                if ridValue, ok := match.Properties["rid"]; ok {
                    if ridStr, ok := ridValue.(string); ok && ridStr == criteria.RuntimeIdentifier {
                        matches = append(matches, match)
                    }
                }
            }
        }
    }

    if len(matches) == 0 {
        return nil
    }

    // Extract items
    var items []string
    for _, match := range matches {
        items = append(items, match.Path)
    }

    return &AssetGroup{
        TargetFramework:   criteria.TargetFramework,
        RuntimeIdentifier: criteria.RuntimeIdentifier,
        Items:             items,
    }
}
```

### Verification Steps

```bash
# 1. Run RID parsing tests
go test ./packaging/assets -v -run TestParseRID

# 2. Test RID graph
go test ./packaging/assets -v -run TestRuntimeGraph

# 3. Test RID compatibility
go test ./packaging/assets -v -run TestRIDCompatibility

# 4. Test RID-specific asset selection
go test ./packaging/assets -v -run TestSelectWithRID

# 5. Check test coverage
go test ./packaging/assets -cover
```

### Acceptance Criteria

- [ ] Parse RID strings (os-arch, os.version-arch)
- [ ] Build RID graph with compatibility imports
- [ ] Check RID compatibility
- [ ] Get all compatible RIDs for target
- [ ] Load default .NET RID graph
- [ ] Select RID-specific assets (runtimes/ folder)
- [ ] Fall back to RID-agnostic assets
- [ ] Support common RIDs (win, linux, osx)
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement RID resolution and selection

Add Runtime Identifier support:
- RID parsing (os-arch, os.version-arch)
- RID graph with compatibility imports
- RID compatibility checking
- Default .NET RID graph
- RID-specific asset selection (runtimes/ folder)

Reference: NuGet.RuntimeModel/RuntimeGraph.cs
Reference: https://learn.microsoft.com/en-us/dotnet/core/rid-catalog
```

---

## M3.14: Package Extraction

**Estimated Time**: 1 hour
**Dependencies**: M3.1, M3.2, M3.3, M3.12

### Overview

Implement complete package extraction with asset selection, file filtering, and installation directory structure creation.

### Files to Create/Modify

- `packaging/extractor.go` - Package extraction implementation
- `packaging/extractor_test.go` - Extraction tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageExtractor.cs` (300+ lines)
- `NuGet.Packaging/PackagePathResolver.cs`

### Implementation Details

**1. Extraction Options**:

```go
// packaging/extractor.go

package packaging

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/packaging/assets"
)

// ExtractionOptions configures package extraction
type ExtractionOptions struct {
    // TargetFramework for asset selection
    TargetFramework *frameworks.NuGetFramework

    // RuntimeIdentifier for RID-specific assets
    RuntimeIdentifier string

    // ExtractNuspec controls whether to extract .nuspec
    ExtractNuspec bool

    // ExtractFiles controls whether to extract package files
    ExtractFiles bool

    // ExtractNupkg controls whether to save .nupkg
    ExtractNupkg bool

    // FileFilter filters which files to extract (nil = all)
    FileFilter func(path string) bool

    // Logger for extraction progress
    Logger Logger
}

// Logger interface for extraction logging
type Logger interface {
    Info(format string, args ...interface{})
    Warning(format string, args ...interface{})
    Error(format string, args ...interface{})
}

// DefaultExtractionOptions returns default extraction options
func DefaultExtractionOptions(targetFramework *frameworks.NuGetFramework) ExtractionOptions {
    return ExtractionOptions{
        TargetFramework: targetFramework,
        ExtractNuspec:   true,
        ExtractFiles:    true,
        ExtractNupkg:    false,
    }
}

// ExtractPackage extracts a package to a destination directory
func ExtractPackage(reader *PackageReader, destDir string, opts ExtractionOptions) error {
    // Get package identity
    identity, err := reader.GetIdentity()
    if err != nil {
        return fmt.Errorf("get package identity: %w", err)
    }

    // Create destination directory
    if err := os.MkdirAll(destDir, 0755); err != nil {
        return fmt.Errorf("create destination directory: %w", err)
    }

    var extractedFiles []string

    // Extract .nuspec if requested
    if opts.ExtractNuspec {
        nuspecPath := filepath.Join(destDir, identity.ID+".nuspec")
        if err := reader.ExtractFile(identity.ID+".nuspec", nuspecPath); err != nil {
            // Try finding nuspec
            nuspecFile, err := reader.GetNuspecFile()
            if err != nil {
                return fmt.Errorf("get nuspec file: %w", err)
            }

            if err := reader.ExtractFile(nuspecFile.Name, nuspecPath); err != nil {
                return fmt.Errorf("extract nuspec: %w", err)
            }
        }

        extractedFiles = append(extractedFiles, nuspecPath)
        if opts.Logger != nil {
            opts.Logger.Info("Extracted %s", nuspecPath)
        }
    }

    // Extract package files if requested
    if opts.ExtractFiles {
        files := reader.GetPackageFiles()

        // Apply file filter
        if opts.FileFilter != nil {
            var filtered []string
            for _, file := range files {
                if opts.FileFilter(file) {
                    filtered = append(filtered, file)
                }
            }
            files = filtered
        }

        // Extract files
        for _, file := range files {
            destPath := filepath.Join(destDir, filepath.FromSlash(file))

            if err := reader.ExtractFile(file, destPath); err != nil {
                if opts.Logger != nil {
                    opts.Logger.Warning("Failed to extract %s: %v", file, err)
                }
                continue
            }

            extractedFiles = append(extractedFiles, destPath)
        }

        if opts.Logger != nil {
            opts.Logger.Info("Extracted %d files", len(files))
        }
    }

    if opts.Logger != nil {
        opts.Logger.Info("Package extraction complete: %s %s", identity.ID, identity.Version.String())
    }

    return nil
}

// ExtractLibItems extracts only runtime assemblies for target framework
func ExtractLibItems(reader *PackageReader, destDir string, targetFramework *frameworks.NuGetFramework) error {
    // Select lib items
    libItems, err := reader.SelectLibItems(targetFramework)
    if err != nil {
        return fmt.Errorf("select lib items: %w", err)
    }

    // Create file filter
    libItemsSet := make(map[string]bool)
    for _, item := range libItems {
        libItemsSet[item] = true
    }

    opts := ExtractionOptions{
        TargetFramework: targetFramework,
        ExtractNuspec:   false,
        ExtractFiles:    true,
        ExtractNupkg:    false,
        FileFilter: func(path string) bool {
            return libItemsSet[path]
        },
    }

    return ExtractPackage(reader, destDir, opts)
}
```

**2. Package Path Resolver**:

```go
// PackagePathResolver resolves package installation paths
type PackagePathResolver struct {
    rootDirectory string
}

// NewPackagePathResolver creates a new path resolver
func NewPackagePathResolver(rootDir string) *PackagePathResolver {
    return &PackagePathResolver{
        rootDirectory: rootDir,
    }
}

// GetInstallPath returns the installation directory for a package
// Format: {root}/{id}/{version}/
func (r *PackagePathResolver) GetInstallPath(identity *PackageIdentity) string {
    return filepath.Join(r.rootDirectory, identity.ID, identity.Version.String())
}

// GetPackageFileName returns the .nupkg file name
func (r *PackagePathResolver) GetPackageFileName(identity *PackageIdentity) string {
    return fmt.Sprintf("%s.%s.nupkg", identity.ID, identity.Version.String())
}

// GetManifestFileName returns the .nuspec file name
func (r *PackagePathResolver) GetManifestFileName(identity *PackageIdentity) string {
    return fmt.Sprintf("%s.nuspec", identity.ID)
}

// GetPackageFilePath returns the full path to the .nupkg file
func (r *PackagePathResolver) GetPackageFilePath(identity *PackageIdentity) string {
    installPath := r.GetInstallPath(identity)
    return filepath.Join(installPath, r.GetPackageFileName(identity))
}
```

**3. Extraction Helper**:

```go
// ExtractAndInstall extracts a package using standard NuGet directory structure
func ExtractAndInstall(packagePath, packagesRoot string, targetFramework *frameworks.NuGetFramework) (*PackageIdentity, error) {
    // Open package
    reader, err := OpenPackage(packagePath)
    if err != nil {
        return nil, fmt.Errorf("open package: %w", err)
    }
    defer reader.Close()

    // Get identity
    identity, err := reader.GetIdentity()
    if err != nil {
        return nil, fmt.Errorf("get identity: %w", err)
    }

    // Determine install path
    resolver := NewPackagePathResolver(packagesRoot)
    installPath := resolver.GetInstallPath(identity)

    // Extract package
    opts := DefaultExtractionOptions(targetFramework)
    opts.ExtractNupkg = true

    if err := ExtractPackage(reader, installPath, opts); err != nil {
        return nil, fmt.Errorf("extract package: %w", err)
    }

    return identity, nil
}
```

### Verification Steps

```bash
# 1. Run extraction tests
go test ./packaging -v -run TestExtraction

# 2. Test with target framework filtering
go test ./packaging -v -run TestExtractLibItems

# 3. Test path resolver
go test ./packaging -v -run TestPackagePathResolver

# 4. Test with real packages
go test ./packaging -v -run TestExtractRealPackage

# 5. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Extract complete package to directory
- [ ] Extract nuspec with correct filename
- [ ] Extract package files maintaining structure
- [ ] Extract lib items with framework filtering
- [ ] Apply custom file filters
- [ ] Resolve installation paths (id/version/)
- [ ] Support extraction options
- [ ] Handle extraction errors gracefully
- [ ] Optional logging support
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement package extraction

Add package extraction with:
- Extract to installation directory
- Framework-based asset filtering
- Nuspec extraction
- Custom file filters
- Package path resolver (id/version/ structure)
- Extraction options configuration

Reference: NuGet.Packaging/PackageExtractor.cs
```

---

## Summary - All M3 Chunks Complete

**Total Time for All M3 Files**: 32 hours
**Total Files Created**: 28
**Total Lines of Code**: ~5,400

**Milestone 3 Complete**: All 14 chunks implemented
- Chunks 1-4: Package Reader & Builder Core
- Chunks 5-7: OPC Compliance & Validation
- Chunks 8-10: Package Signatures
- Chunks 11-14: Asset Selection & Extraction

**Next Milestone**: M4 - Infrastructure & Resilience (Not in this file set)
