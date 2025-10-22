# Milestone 3: Package Operations - Continued 3 (Chunks 11-14)

**Status**: Not Started
**Chunks**: 11-14 (Asset Selection & Extraction)
**Estimated Time**: 6 hours

---

## M3.11: Asset Selection - Pattern Engine

**Estimated Time**: 3 hours
**Dependencies**: M1.9 (framework parsing), M1.11 (framework compatibility)

### Overview

Implement the pattern-based asset selection engine that matches package files to target frameworks and runtime identifiers. This system uses property-based pattern definitions with a sophisticated parsing infrastructure that mirrors NuGet.Client's ContentModel system.

**Key Concepts**:
- **Patterns**: Path templates with property placeholders (e.g., `lib/{tfm}/{assembly}`)
- **PatternTable**: Token replacement table for aliasing (e.g., `"any"` → specific framework)
- **PatternExpression**: Compiled pattern with optimized matching logic
- **ContentPropertyDefinition**: Property parser with compatibility and comparison tests
- **PatternSet**: Collection of group and path patterns for an asset type

### Files to Create/Modify

- `packaging/assets/pattern.go` - Pattern definition and PatternSet
- `packaging/assets/patterntable.go` - Token replacement tables
- `packaging/assets/expression.go` - Pattern expression (compiled pattern)
- `packaging/assets/property.go` - Property definition with parsers
- `packaging/assets/conventions.go` - Managed code conventions
- `packaging/assets/pattern_test.go` - Pattern tests
- `packaging/assets/expression_test.go` - Expression matching tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/ContentModel/ContentQueryDefinition.cs` - **PatternSet and PatternDefinition**
- `NuGet.Packaging/ContentModel/ContentPropertyDefinition.cs` - Property definitions
- `NuGet.Packaging/ContentModel/PatternTable.cs` - Token replacement
- `NuGet.Packaging/ContentModel/PatternTableEntry.cs` - Table entries
- `NuGet.Packaging/ContentModel/Infrastructure/Parser.cs` - Pattern expressions
- `NuGet.Packaging/ContentModel/ManagedCodeConventions.cs` - Conventions (549 lines)

**Critical Correction**: The guide previously referenced `PatternSet.cs` and `PatternDefinition.cs` as separate files. These classes are **actually defined in `ContentQueryDefinition.cs`**.

### Architecture Overview

```
PatternSet
  ├─ GroupPatterns []PatternDefinition   (directory-level matching)
  ├─ PathPatterns []PatternDefinition    (file-level matching)
  ├─ GroupExpressions []PatternExpression (compiled patterns)
  ├─ PathExpressions []PatternExpression
  └─ PropertyDefinitions map[string]PropertyDefinition

PatternDefinition
  ├─ Pattern string                      ("lib/{tfm}/{assembly}")
  ├─ Defaults map[string]interface{}     (default property values)
  ├─ Table *PatternTable                 (token replacement)
  └─ PreserveRawValues bool              (performance optimization)

PatternExpression
  ├─ Segments []Segment                  (literal + token segments)
  ├─ Defaults map[string]interface{}
  └─ Table *PatternTable

PropertyDefinition
  ├─ Name string
  ├─ Parser func(value, table, matchOnly)
  ├─ CompatibilityTest func(criterion, available) bool
  ├─ CompareTest func(criterion, a1, a2) int
  ├─ FileExtensions []string
  └─ AllowSubFolders bool
```

### Implementation Details

**1. Pattern Definition and PatternSet** (`packaging/assets/pattern.go`):

```go
package assets

import (
	"github.com/willibrandon/gonuget/frameworks"
)

// PatternDefinition defines a file path pattern with property placeholders.
// Reference: ContentQueryDefinition.cs PatternDefinition
type PatternDefinition struct {
	// Pattern is the path template with property placeholders.
	// Example: "lib/{tfm}/{assembly}"
	// Optional properties end with '?': "lib/{any?}"
	Pattern string

	// Defaults provides default property values.
	// Example: {"tfm": &frameworks.NuGetFramework{...}}
	Defaults map[string]interface{}

	// Table is the token replacement table.
	// Maps property values to replacements (e.g., "any" -> DotNet framework)
	Table *PatternTable

	// PreserveRawValues indicates whether to preserve unparsed string values.
	// Performance optimization for grouping operations.
	PreserveRawValues bool
}

// PatternSet groups multiple patterns for a specific asset type.
// Reference: ContentQueryDefinition.cs PatternSet
type PatternSet struct {
	// GroupPatterns match directory-level patterns.
	// Used for grouping items by common properties.
	GroupPatterns []*PatternDefinition

	// PathPatterns match individual file paths.
	// Used for selecting specific assets.
	PathPatterns []*PatternDefinition

	// GroupExpressions are compiled group patterns (for performance).
	GroupExpressions []*PatternExpression

	// PathExpressions are compiled path patterns (for performance).
	PathExpressions []*PatternExpression

	// PropertyDefinitions available for this pattern set.
	PropertyDefinitions map[string]*PropertyDefinition
}

// NewPatternSet creates a pattern set with compiled expressions.
func NewPatternSet(
	properties map[string]*PropertyDefinition,
	groupPatterns []*PatternDefinition,
	pathPatterns []*PatternDefinition,
) *PatternSet {
	ps := &PatternSet{
		GroupPatterns:       groupPatterns,
		PathPatterns:        pathPatterns,
		PropertyDefinitions: properties,
	}

	// Compile expressions
	ps.GroupExpressions = make([]*PatternExpression, len(groupPatterns))
	for i, pattern := range groupPatterns {
		ps.GroupExpressions[i] = NewPatternExpression(pattern)
	}

	ps.PathExpressions = make([]*PatternExpression, len(pathPatterns))
	for i, pattern := range pathPatterns {
		ps.PathExpressions[i] = NewPatternExpression(pattern)
	}

	return ps
}

// ContentItem represents a matched path with extracted properties.
// Reference: ContentModel/ContentItem.cs
type ContentItem struct {
	Path       string
	Properties map[string]interface{}
}

// Add adds or updates a property value.
func (c *ContentItem) Add(key string, value interface{}) {
	if c.Properties == nil {
		c.Properties = make(map[string]interface{})
	}
	// Don't overwrite existing properties
	if _, exists := c.Properties[key]; !exists {
		c.Properties[key] = value
	}
}
```

**2. PatternTable** (`packaging/assets/patterntable.go`):

```go
package assets

// PatternTableEntry defines a token replacement.
// Reference: ContentModel/PatternTableEntry.cs
type PatternTableEntry struct {
	PropertyName string      // Property this entry applies to (e.g., "tfm")
	Name         string      // Token name to replace (e.g., "any")
	Value        interface{} // Replacement value
}

// PatternTable is a token replacement table organized by property.
// Reference: ContentModel/PatternTable.cs
type PatternTable struct {
	// table maps propertyName -> tokenName -> replacement value
	table map[string]map[string]interface{}
}

// NewPatternTable creates a pattern table from entries.
func NewPatternTable(entries []PatternTableEntry) *PatternTable {
	pt := &PatternTable{
		table: make(map[string]map[string]interface{}),
	}

	for _, entry := range entries {
		byProp, ok := pt.table[entry.PropertyName]
		if !ok {
			byProp = make(map[string]interface{})
			pt.table[entry.PropertyName] = byProp
		}
		byProp[entry.Name] = entry.Value
	}

	return pt
}

// TryLookup attempts to find a replacement value for a token.
func (pt *PatternTable) TryLookup(propertyName, name string) (interface{}, bool) {
	if pt == nil {
		return nil, false
	}

	byProp, ok := pt.table[propertyName]
	if !ok {
		return nil, false
	}

	value, ok := byProp[name]
	return value, ok
}

// Standard pattern tables used by ManagedCodeConventions.
// Reference: ManagedCodeConventions.cs lines 51-65

// DotnetAnyTable maps "any" to DotNet framework.
var DotnetAnyTable = NewPatternTable([]PatternTableEntry{
	{
		PropertyName: "tfm",
		Name:         "any",
		Value:        frameworks.CommonFrameworks.DotNet,
	},
})

// AnyTable maps "any" to AnyFramework.
var AnyTable = NewPatternTable([]PatternTableEntry{
	{
		PropertyName: "tfm",
		Name:         "any",
		Value:        frameworks.AnyFramework,
	},
})
```

**3. Property Definition** (`packaging/assets/property.go`):

```go
package assets

// PropertyParser parses a property value from a string.
// Reference: ContentPropertyDefinition.cs Parser
//
// Parameters:
//   - value: substring to parse (no allocation)
//   - table: pattern table for token replacement
//   - matchOnly: if true, skip value actualization (performance optimization)
//
// Returns parsed value or nil if doesn't match.
type PropertyParser func(value string, table *PatternTable, matchOnly bool) interface{}

// CompatibilityTest checks if criterion is compatible with available value.
// Reference: ContentPropertyDefinition.cs CompatibilityTest
type CompatibilityTest func(criterion, available interface{}) bool

// CompareTest determines which of two available values is nearer to criterion.
// Reference: ContentPropertyDefinition.cs CompareTest
// Returns: -1 if available1 is nearer, 1 if available2 is nearer, 0 if equal
type CompareTest func(criterion, available1, available2 interface{}) int

// PropertyDefinition defines how a property is parsed and compared.
// Reference: ContentModel/ContentPropertyDefinition.cs
type PropertyDefinition struct {
	// Name is the property identifier (e.g., "tfm", "rid", "assembly")
	Name string

	// FileExtensions for file-based properties (e.g., [".dll", ".exe"])
	FileExtensions []string

	// AllowSubFolders allows file extensions to match in subdirectories
	AllowSubFolders bool

	// Parser converts string value to typed value
	Parser PropertyParser

	// CompatibilityTest checks if criterion is satisfied by available value
	CompatibilityTest CompatibilityTest

	// CompareTest finds the nearest candidate
	CompareTest CompareTest
}

// TryLookup attempts to parse a value using this property definition.
// Reference: ContentPropertyDefinition.cs TryLookup (lines 91-132)
func (pd *PropertyDefinition) TryLookup(name string, table *PatternTable, matchOnly bool) (interface{}, bool) {
	if name == "" {
		return nil, false
	}

	// Check file extensions first
	if len(pd.FileExtensions) > 0 {
		hasSlash := containsSlash(name)
		if pd.AllowSubFolders || !hasSlash {
			for _, ext := range pd.FileExtensions {
				if endsWithIgnoreCase(name, ext) {
					if matchOnly {
						return nil, true // Match found, value not needed
					}
					return name, true
				}
			}
		}
	}

	// Try parser
	if pd.Parser != nil {
		value := pd.Parser(name, table, matchOnly)
		if value != nil {
			return value, true
		}
	}

	return nil, false
}

// IsCriteriaSatisfied checks if criterion is satisfied by candidate value.
// Reference: ContentPropertyDefinition.cs IsCriteriaSatisfied
func (pd *PropertyDefinition) IsCriteriaSatisfied(criteriaValue, candidateValue interface{}) bool {
	if pd.CompatibilityTest == nil {
		// Default: exact equality
		return criteriaValue == candidateValue
	}
	return pd.CompatibilityTest(criteriaValue, candidateValue)
}

// Compare compares two candidate values against a criterion.
// Reference: ContentPropertyDefinition.cs Compare (lines 161-182)
func (pd *PropertyDefinition) Compare(criteriaValue, candidateValue1, candidateValue2 interface{}) int {
	// Check if one value is more compatible than the other
	betterCoverageFromValue1 := pd.IsCriteriaSatisfied(candidateValue1, candidateValue2)
	betterCoverageFromValue2 := pd.IsCriteriaSatisfied(candidateValue2, candidateValue1)

	if betterCoverageFromValue1 && !betterCoverageFromValue2 {
		return -1
	}
	if betterCoverageFromValue2 && !betterCoverageFromValue1 {
		return 1
	}

	// Tie - use external compare test
	if pd.CompareTest != nil {
		return pd.CompareTest(criteriaValue, candidateValue1, candidateValue2)
	}

	// No tie breaker
	return 0
}

func containsSlash(s string) bool {
	for _, ch := range s {
		if ch == '/' || ch == '\\' {
			return true
		}
	}
	return false
}

func endsWithIgnoreCase(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return strings.EqualFold(s[len(s)-len(suffix):], suffix)
}
```

**4. Pattern Expression** (`packaging/assets/expression.go`):

```go
package assets

import (
	"strings"
)

// PatternExpression is a compiled pattern with optimized matching.
// Reference: ContentModel/Infrastructure/Parser.cs PatternExpression
type PatternExpression struct {
	segments []Segment
	defaults map[string]interface{}
	table    *PatternTable
}

// Segment represents a pattern segment (literal or token).
type Segment interface {
	// TryMatch attempts to match this segment against path.
	// Returns true and end index if match succeeds.
	TryMatch(item **ContentItem, path string, properties map[string]*PropertyDefinition, startIndex int) (int, bool)
}

// LiteralSegment matches exact text.
type LiteralSegment struct {
	text string
}

// TokenSegment matches a property placeholder.
type TokenSegment struct {
	name             string
	delimiter        byte
	matchOnly        bool
	table            *PatternTable
	preserveRawValue bool
}

// NewPatternExpression compiles a pattern definition into an expression.
// Reference: PatternExpression constructor (lines 18-23)
func NewPatternExpression(pattern *PatternDefinition) *PatternExpression {
	expr := &PatternExpression{
		table:    pattern.Table,
		defaults: make(map[string]interface{}),
	}

	// Copy defaults
	for k, v := range pattern.Defaults {
		expr.defaults[k] = v
	}

	// Parse pattern into segments
	expr.initialize(pattern.Pattern, pattern.PreserveRawValues)

	return expr
}

// initialize parses pattern string into literal and token segments.
// Reference: PatternExpression.Initialize (lines 25-65)
func (pe *PatternExpression) initialize(pattern string, preserveRawValues bool) {
	scanIndex := 0

	for scanIndex < len(pattern) {
		// Find next token
		beginToken := len(pattern)
		endToken := len(pattern)

		for i := scanIndex; i < len(pattern); i++ {
			ch := pattern[i]
			if beginToken == len(pattern) {
				if ch == '{' {
					beginToken = i
				}
			} else if ch == '}' {
				endToken = i
				break
			}
		}

		// Add literal segment if any
		if scanIndex != beginToken {
			pe.segments = append(pe.segments, &LiteralSegment{
				text: pattern[scanIndex:beginToken],
			})
		}

		// Add token segment if any
		if beginToken != endToken {
			var delimiter byte
			if endToken+1 < len(pattern) {
				delimiter = pattern[endToken+1]
			}

			matchOnly := pattern[endToken-1] == '?'

			beginName := beginToken + 1
			endName := endToken
			if matchOnly {
				endName--
			}

			tokenName := pattern[beginName:endName]
			pe.segments = append(pe.segments, &TokenSegment{
				name:             tokenName,
				delimiter:        delimiter,
				matchOnly:        matchOnly,
				table:            pe.table,
				preserveRawValue: preserveRawValues,
			})
		}

		scanIndex = endToken + 1
	}
}

// Match attempts to match path against this expression.
// Reference: PatternExpression.Match (lines 67-109)
func (pe *PatternExpression) Match(path string, propertyDefinitions map[string]*PropertyDefinition) *ContentItem {
	var item *ContentItem
	startIndex := 0

	for _, segment := range pe.segments {
		endIndex, ok := segment.TryMatch(&item, path, propertyDefinitions, startIndex)
		if !ok {
			return nil
		}
		startIndex = endIndex
	}

	// Check if we consumed the entire path
	if startIndex != len(path) {
		return nil
	}

	// Apply defaults
	if item == nil {
		item = &ContentItem{
			Path:       path,
			Properties: pe.defaults,
		}
	} else {
		for key, value := range pe.defaults {
			item.Add(key, value)
		}
	}

	return item
}

// TryMatch for LiteralSegment (lines 119-136)
func (ls *LiteralSegment) TryMatch(item **ContentItem, path string, properties map[string]*PropertyDefinition, startIndex int) (int, bool) {
	if startIndex+len(ls.text) > len(path) {
		return 0, false
	}

	// Case-insensitive comparison
	pathSegment := path[startIndex : startIndex+len(ls.text)]
	if !strings.EqualFold(pathSegment, ls.text) {
		return 0, false
	}

	return startIndex + len(ls.text), true
}

// TryMatch for TokenSegment (lines 167-261)
func (ts *TokenSegment) TryMatch(item **ContentItem, path string, properties map[string]*PropertyDefinition, startIndex int) (int, bool) {
	// Find end of this token (until delimiter or end of path)
	endIndex := startIndex
	if ts.delimiter != 0 {
		for endIndex < len(path) && path[endIndex] != ts.delimiter {
			endIndex++
		}
	} else {
		endIndex = len(path)
	}

	if endIndex == startIndex && !ts.matchOnly {
		// Empty value for non-optional token
		return 0, false
	}

	// Get property definition
	propDef, ok := properties[ts.name]
	if !ok {
		// Unknown property, treat as string
		tokenValue := path[startIndex:endIndex]
		if *item == nil {
			*item = &ContentItem{
				Path:       path,
				Properties: make(map[string]interface{}),
			}
		}
		(*item).Properties[ts.name] = tokenValue
		return endIndex, true
	}

	// Try to parse value
	tokenValue := path[startIndex:endIndex]
	value, matched := propDef.TryLookup(tokenValue, ts.table, ts.matchOnly)

	if !matched {
		return 0, false
	}

	// Store value if not match-only
	if !ts.matchOnly && value != nil {
		if *item == nil {
			*item = &ContentItem{
				Path:       path,
				Properties: make(map[string]interface{}),
			}
		}

		// Store parsed value
		(*item).Properties[ts.name] = value

		// Store raw value if preserving
		if ts.preserveRawValue {
			(*item).Properties[ts.name+"_raw"] = tokenValue
		}
	}

	return endIndex, true
}
```

**5. Managed Code Conventions** (`packaging/assets/conventions.go`):

```go
package assets

import (
	"github.com/willibrandon/gonuget/frameworks"
)

// ManagedCodeConventions defines standard .NET package conventions.
// Reference: ContentModel/ManagedCodeConventions.cs
type ManagedCodeConventions struct {
	// Properties available for pattern matching
	Properties map[string]*PropertyDefinition

	// Pattern sets for different asset types
	RuntimeAssemblies      *PatternSet
	CompileRefAssemblies   *PatternSet
	CompileLibAssemblies   *PatternSet
	NativeLibraries        *PatternSet
	ResourceAssemblies     *PatternSet
	MSBuildFiles           *PatternSet
	MSBuildMultiTargeting  *PatternSet
	ContentFiles           *PatternSet
	ToolsAssemblies        *PatternSet
}

// NewManagedCodeConventions creates standard managed code conventions.
// Reference: ManagedCodeConventions constructor (lines 77-103)
func NewManagedCodeConventions() *ManagedCodeConventions {
	conventions := &ManagedCodeConventions{
		Properties: make(map[string]*PropertyDefinition),
	}

	// Define properties
	conventions.defineProperties()

	// Define pattern sets
	conventions.definePatternSets()

	return conventions
}

func (c *ManagedCodeConventions) defineProperties() {
	// Assembly property - matches .dll, .winmd, .exe files
	// Reference: ManagedCodeConventions.cs lines 25-27
	c.Properties["assembly"] = &PropertyDefinition{
		Name:           "assembly",
		Parser:         allowEmptyFolderParser,
		FileExtensions: []string{".dll", ".winmd", ".exe"},
	}

	// MSBuild property - matches .targets, .props files
	c.Properties["msbuild"] = &PropertyDefinition{
		Name:           "msbuild",
		Parser:         allowEmptyFolderParser,
		FileExtensions: []string{".targets", ".props"},
	}

	// Satellite assembly property - matches .resources.dll
	c.Properties["satelliteAssembly"] = &PropertyDefinition{
		Name:           "satelliteAssembly",
		Parser:         allowEmptyFolderParser,
		FileExtensions: []string{".resources.dll"},
	}

	// Locale property - matches any string
	c.Properties["locale"] = &PropertyDefinition{
		Name:   "locale",
		Parser: localeParser,
	}

	// Any property - matches any string
	c.Properties["any"] = &PropertyDefinition{
		Name:   "any",
		Parser: identityParser,
	}

	// Target Framework Moniker (TFM) property
	// Reference: ManagedCodeConventions.cs lines 94-98
	c.Properties["tfm"] = &PropertyDefinition{
		Name:              "tfm",
		Parser:            tfmParser,
		CompatibilityTest: tfmCompatibilityTest,
		CompareTest:       tfmCompareTest,
	}

	// Runtime Identifier (RID) property
	c.Properties["rid"] = &PropertyDefinition{
		Name:              "rid",
		Parser:            identityParser,
		CompatibilityTest: ridCompatibilityTest,
	}

	// Code language property
	c.Properties["codeLanguage"] = &PropertyDefinition{
		Name:   "codeLanguage",
		Parser: codeLanguageParser,
	}
}

// Property parsers
// Reference: ManagedCodeConventions.cs lines 256-397

func allowEmptyFolderParser(value string, table *PatternTable, matchOnly bool) interface{} {
	if matchOnly {
		return value // Return something non-nil to indicate match
	}
	return value
}

func identityParser(value string, table *PatternTable, matchOnly bool) interface{} {
	if matchOnly {
		return value
	}
	return value
}

func localeParser(value string, table *PatternTable, matchOnly bool) interface{} {
	// Locale validation would go here
	if matchOnly {
		return value
	}
	return value
}

func codeLanguageParser(value string, table *PatternTable, matchOnly bool) interface{} {
	// Code language parsing (cs, vb, fs, etc.)
	if matchOnly {
		return value
	}
	return strings.ToLower(value)
}

func tfmParser(value string, table *PatternTable, matchOnly bool) interface{} {
	// Check table first for aliases
	if table != nil {
		if replacement, ok := table.TryLookup("tfm", value); ok {
			return replacement
		}
	}

	// Parse framework
	if matchOnly {
		// Don't parse, just check validity
		_, err := frameworks.ParseFramework(value)
		if err != nil {
			return nil
		}
		return value
	}

	fw, err := frameworks.ParseFramework(value)
	if err != nil {
		return nil
	}
	return fw
}

// Compatibility and comparison tests
// Reference: ManagedCodeConventions.cs lines 400-527

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

	return frameworks.IsCompatible(criterionFW, availableFW)
}

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

	if nearest.Equals(fw1) {
		return -1
	} else if nearest.Equals(fw2) {
		return 1
	}

	return 0
}

func ridCompatibilityTest(criterion, available interface{}) bool {
	// RID compatibility will be implemented in M3.13
	// For now, exact match
	return criterion == available
}

func (c *ManagedCodeConventions) definePatternSets() {
	// Define pattern sets for each asset type
	// Reference: ManagedCodeConventions.cs ManagedCodePatterns (lines 478-606)

	// RuntimeAssemblies: lib/ folder
	c.RuntimeAssemblies = NewPatternSet(
		c.Properties,
		[]*PatternDefinition{
			{
				Pattern: "runtimes/{rid}/lib/{tfm}/{any?}",
				Table:   DotnetAnyTable,
			},
			{
				Pattern: "lib/{tfm}/{any?}",
				Table:   DotnetAnyTable,
			},
			{
				Pattern: "lib/{assembly?}",
				Table:   DotnetAnyTable,
				Defaults: map[string]interface{}{
					"tfm": frameworks.CommonFrameworks.Net,
				},
			},
		},
		[]*PatternDefinition{
			{
				Pattern: "runtimes/{rid}/lib/{tfm}/{assembly}",
				Table:   DotnetAnyTable,
			},
			{
				Pattern: "lib/{tfm}/{assembly}",
				Table:   DotnetAnyTable,
			},
			{
				Pattern: "lib/{assembly}",
				Table:   DotnetAnyTable,
				Defaults: map[string]interface{}{
					"tfm": frameworks.CommonFrameworks.Net,
				},
			},
		},
	)

	// CompileRefAssemblies: ref/ folder
	c.CompileRefAssemblies = NewPatternSet(
		c.Properties,
		[]*PatternDefinition{
			{
				Pattern: "ref/{tfm}/{any?}",
				Table:   DotnetAnyTable,
			},
		},
		[]*PatternDefinition{
			{
				Pattern: "ref/{tfm}/{assembly}",
				Table:   DotnetAnyTable,
			},
		},
	)

	// CompileLibAssemblies: lib/ folder for compile
	c.CompileLibAssemblies = NewPatternSet(
		c.Properties,
		[]*PatternDefinition{
			{
				Pattern: "lib/{tfm}/{any?}",
				Table:   DotnetAnyTable,
			},
			{
				Pattern: "lib/{assembly?}",
				Table:   DotnetAnyTable,
				Defaults: map[string]interface{}{
					"tfm": frameworks.CommonFrameworks.Net,
				},
			},
		},
		[]*PatternDefinition{
			{
				Pattern: "lib/{tfm}/{assembly}",
				Table:   DotnetAnyTable,
			},
			{
				Pattern: "lib/{assembly}",
				Table:   DotnetAnyTable,
				Defaults: map[string]interface{}{
					"tfm": frameworks.CommonFrameworks.Net,
				},
			},
		},
	)

	// Additional pattern sets...
	// (NativeLibraries, ResourceAssemblies, MSBuildFiles, etc.)
}
```

### Verification Steps

```bash
# 1. Test pattern table
go test ./packaging/assets -v -run TestPatternTable

# 2. Test property definitions
go test ./packaging/assets -v -run TestPropertyDefinition

# 3. Test pattern expression parsing
go test ./packaging/assets -v -run TestPatternExpression

# 4. Test pattern matching
go test ./packaging/assets -v -run TestPatternMatch

# 5. Test managed code conventions
go test ./packaging/assets -v -run TestManagedCodeConventions

# 6. Test with real package paths
go test ./packaging/assets -v -run TestRealPackagePaths

# 7. Check test coverage
go test ./packaging/assets -cover
```

### Acceptance Criteria

- [ ] PatternDefinition with pattern, defaults, and table
- [ ] PatternTable for token replacement
- [ ] PropertyDefinition with parser, compatibility test, and compare test
- [ ] PatternExpression compilation (literal and token segments)
- [ ] Pattern matching with property extraction
- [ ] TFM compatibility and comparison
- [ ] ManagedCodeConventions with all pattern sets
- [ ] RuntimeAssemblies patterns (lib/, runtimes/)
- [ ] CompileRefAssemblies patterns (ref/)
- [ ] MSBuildFiles patterns (build/)
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement asset selection pattern engine

Add NuGet.Client-compatible pattern system:
- PatternDefinition with placeholders and defaults
- PatternTable for token replacement (e.g., "any" -> DotNet)
- PropertyDefinition with parsers and compatibility tests
- PatternExpression with compiled segment matching
- Managed code conventions (lib/, ref/, build/, runtimes/)
- TFM compatibility and nearest-match selection

Reference: NuGet.Packaging/ContentModel/
- ContentQueryDefinition.cs (PatternSet, PatternDefinition)
- ContentPropertyDefinition.cs
- PatternTable.cs
- Infrastructure/Parser.cs (PatternExpression)
- ManagedCodeConventions.cs

Chunk: M3.11
Status: ✓ Complete
```

---
## M3.12: Asset Selection - Framework Resolution

**Estimated Time**: 2.5 hours
**Dependencies**: M3.11, M1.4 (FrameworkReducer)

### Overview

Implement framework-based asset selection using ContentModel's `FindBestItemGroup` pattern. This matches NuGet.Client's approach in `ContentItemCollection` and `LockFileUtils`, NOT the simplified pattern shown in earlier guides.

**Critical Architecture Notes**:
1. **NO explicit GetLibItems/GetRefItems methods** - NuGet.Client uses `FindBestItemGroup`
2. Asset selection is performed by `ContentItemCollection.FindBestItemGroup()` in NuGet.Packaging
3. `LockFileUtils.GetLockFileItems()` orchestrates pattern matching and group selection
4. Framework resolution uses `PropertyDefinition.Compare()` with `FrameworkReducer.GetNearest()`

### Files to Create/Modify

- `packaging/assets/collection.go` - ContentItemCollection with FindBestItemGroup
- `packaging/assets/selection.go` - SelectionCriteria and SelectionCriteriaBuilder
- `packaging/assets/collection_test.go` - Asset selection tests
- `packaging/reader.go` - Add GetContentItems and selection helpers

### Reference Implementation

**NuGet.Client Reference (CORRECTED)**:
- `NuGet.Packaging/ContentModel/ContentItemCollection.cs` - **FindBestItemGroup** (Lines 136-241)
- `NuGet.Commands/RestoreCommand/Utility/LockFileUtils.cs` - **GetLockFileItems** (Lines 663-713)
- `NuGet.Commands/RestoreCommand/LockFileBuilder.cs` - **CreateLockFile** orchestration
- `NuGet.Frameworks/FrameworkReducer.cs` - **GetNearest** (Lines 42-67)
- `NuGet.Packaging/ContentModel/ManagedCodeConventions.cs` - Criteria building (Lines 377-405)

### Implementation Details

**Architecture**: Following NuGet.Client's ContentModel pattern:
1. `ContentItemCollection` - Stores assets and performs matching
2. `PopulateItemGroups` - Groups assets by framework/RID properties
3. `FindBestItemGroup` - Selects best matching group using property comparison
4. `SelectionCriteria` - Defines ordered criteria for matching

**1. Selection Criteria** (`packaging/assets/selection.go`):

```go
package assets

import "github.com/willibrandon/gonuget/frameworks"

// SelectionCriteriaEntry represents a single criteria entry with properties
// Reference: ContentModel/SelectionCriteriaEntry.cs
type SelectionCriteriaEntry struct {
	Properties map[string]interface{}
}

// SelectionCriteria contains ordered list of criteria entries
// Reference: ContentModel/SelectionCriteria.cs
type SelectionCriteria struct {
	Entries []SelectionCriteriaEntry
}

// SelectionCriteriaBuilder builds selection criteria with fluent API
// Reference: ContentModel/SelectionCriteriaBuilder.cs
type SelectionCriteriaBuilder struct {
	properties   map[string]*PropertyDefinition
	currentEntry SelectionCriteriaEntry
	entries      []SelectionCriteriaEntry
}

// NewSelectionCriteriaBuilder creates a new criteria builder
func NewSelectionCriteriaBuilder(properties map[string]*PropertyDefinition) *SelectionCriteriaBuilder {
	return &SelectionCriteriaBuilder{
		properties:   properties,
		currentEntry: SelectionCriteriaEntry{Properties: make(map[string]interface{})},
	}
}

// Add sets a property value and returns builder for chaining
func (b *SelectionCriteriaBuilder) Add(key string, value interface{}) *SelectionCriteriaBuilder {
	b.currentEntry.Properties[key] = value
	return b
}

// NextEntry finalizes current entry and starts new one
func (b *SelectionCriteriaBuilder) NextEntry() *SelectionCriteriaBuilder {
	if len(b.currentEntry.Properties) > 0 {
		b.entries = append(b.entries, b.currentEntry)
		b.currentEntry = SelectionCriteriaEntry{Properties: make(map[string]interface{})}
	}
	return b
}

// Build finalizes and returns the criteria
func (b *SelectionCriteriaBuilder) Build() *SelectionCriteria {
	if len(b.currentEntry.Properties) > 0 {
		b.entries = append(b.entries, b.currentEntry)
	}
	return &SelectionCriteria{Entries: b.entries}
}

// ForFramework creates criteria for framework-only matching (no RID)
// Reference: ManagedCodeConventions.cs ForFramework (Lines 410-417)
func ForFramework(framework *frameworks.NuGetFramework, properties map[string]*PropertyDefinition) *SelectionCriteria {
	builder := NewSelectionCriteriaBuilder(properties)
	builder.Add("tfm", framework)
	builder.Add("rid", nil) // Explicitly no RID
	return builder.Build()
}

// ForFrameworkAndRuntime creates criteria with RID fallback
// Reference: ManagedCodeConventions.cs ForFrameworkAndRuntime (Lines 377-405)
func ForFrameworkAndRuntime(framework *frameworks.NuGetFramework, runtimeIdentifier string, properties map[string]*PropertyDefinition) *SelectionCriteria {
	builder := NewSelectionCriteriaBuilder(properties)

	if runtimeIdentifier != "" {
		// First try: RID-specific assets
		builder.Add("tfm", framework)
		builder.Add("rid", runtimeIdentifier)
		builder.NextEntry()
	}

	// Fallback: RID-agnostic assets
	builder.Add("tfm", framework)
	builder.Add("rid", nil)

	return builder.Build()
}
```

**2. ContentItemCollection and FindBestItemGroup** (`packaging/assets/collection.go`):

```go
package assets

import "github.com/willibrandon/gonuget/frameworks"

// ContentItemCollection manages package assets and performs selection
// Reference: ContentModel/ContentItemCollection.cs
type ContentItemCollection struct {
	Assets []*ContentItem
}

// NewContentItemCollection creates a collection from file paths
func NewContentItemCollection(paths []string) *ContentItemCollection {
	assets := make([]*ContentItem, len(paths))
	for i, path := range paths {
		assets[i] = &ContentItem{Path: path, Properties: make(map[string]interface{})}
	}
	return &ContentItemCollection{Assets: assets}
}

// PopulateItemGroups groups assets by their properties
// Reference: ContentItemCollection.cs PopulateItemGroups (Lines 84-134)
func (c *ContentItemCollection) PopulateItemGroups(patternSet *PatternSet) []*ContentItemGroup {
	if len(c.Assets) == 0 {
		return nil
	}

	groupAssets := make(map[string]*ContentItemGroup)

	for _, asset := range c.Assets {
		// Try each group pattern
		for _, groupExpr := range patternSet.GroupExpressions {
			item := groupExpr.Match(asset.Path, patternSet.Properties)
			if item != nil {
				// Create group key from properties
				groupKey := buildGroupKey(item.Properties)

				if _, exists := groupAssets[groupKey]; !exists {
					groupAssets[groupKey] = &ContentItemGroup{
						Properties: item.Properties,
						Items:      []*ContentItem{},
					}
				}

				// Find matching items using path patterns
				for _, pathExpr := range patternSet.PathExpressions {
					pathItem := pathExpr.Match(asset.Path, patternSet.Properties)
					if pathItem != nil {
						groupAssets[groupKey].Items = append(groupAssets[groupKey].Items, pathItem)
						break
					}
				}
				break
			}
		}
	}

	// Convert map to slice
	groups := make([]*ContentItemGroup, 0, len(groupAssets))
	for _, group := range groupAssets {
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}

// FindBestItemGroup selects the best matching group for criteria
// Reference: ContentItemCollection.cs FindBestItemGroup (Lines 136-241)
func (c *ContentItemCollection) FindBestItemGroup(criteria *SelectionCriteria, patternSets ...*PatternSet) *ContentItemGroup {
	for _, patternSet := range patternSets {
		groups := c.PopulateItemGroups(patternSet)

		// Try each criteria entry in order
		for _, criteriaEntry := range criteria.Entries {
			var bestGroup *ContentItemGroup
			bestAmbiguity := false

			for _, itemGroup := range groups {
				groupIsValid := true

				// Check if group satisfies all criteria properties
				for key, criteriaValue := range criteriaEntry.Properties {
					if criteriaValue == nil {
						// Criteria requires property to NOT exist
						if _, exists := itemGroup.Properties[key]; exists {
							groupIsValid = false
							break
						}
					} else {
						// Criteria requires property to exist and be compatible
						itemValue, exists := itemGroup.Properties[key]
						if !exists {
							groupIsValid = false
							break
						}

						propDef, hasDef := patternSet.Properties[key]
						if !hasDef {
							groupIsValid = false
							break
						}

						// Use property definition's compatibility test
						if !propDef.IsCriteriaSatisfied(criteriaValue, itemValue) {
							groupIsValid = false
							break
						}
					}
				}

				if groupIsValid {
					if bestGroup == nil {
						bestGroup = itemGroup
					} else {
						// Compare groups to find better match
						groupComparison := 0

						for key, criteriaValue := range criteriaEntry.Properties {
							if criteriaValue == nil {
								continue
							}

							bestGroupValue := bestGroup.Properties[key]
							itemGroupValue := itemGroup.Properties[key]
							propDef := patternSet.Properties[key]

							groupComparison = propDef.Compare(criteriaValue, bestGroupValue, itemGroupValue)
							if groupComparison != 0 {
								break
							}
						}

						if groupComparison > 0 {
							// itemGroup is better
							bestGroup = itemGroup
							bestAmbiguity = false
						} else if groupComparison == 0 {
							// Ambiguous - equal match
							bestAmbiguity = true
						}
					}
				}
			}

			if bestGroup != nil {
				return bestGroup
			}
		}
	}

	return nil
}

// ContentItemGroup represents assets grouped by properties
// Reference: ContentModel/ContentItemGroup.cs
type ContentItemGroup struct {
	Properties map[string]interface{}
	Items      []*ContentItem
}

func buildGroupKey(properties map[string]interface{}) string {
	// Build stable key from properties
	key := ""
	if tfm, ok := properties["tfm"]; ok {
		if fw, ok := tfm.(*frameworks.NuGetFramework); ok {
			key += fw.GetShortFolderName() + "|"
		}
	}
	if rid, ok := properties["rid"]; ok {
		if ridStr, ok := rid.(string); ok {
			key += ridStr
		}
	}
	return key
}
```

**3. Helper Functions for PackageReader** (`packaging/assets/utils.go`):

```go
package assets

import (
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/frameworks"
)

// GetLockFileItems selects assets using criteria and patterns
// Reference: LockFileUtils.cs GetLockFileItems (Lines 663-713)
func GetLockFileItems(criteria *SelectionCriteria, collection *ContentItemCollection, patternSets ...*PatternSet) []string {
	group := collection.FindBestItemGroup(criteria, patternSets...)
	if group == nil {
		return []string{}
	}

	paths := make([]string, len(group.Items))
	for i, item := range group.Items {
		paths[i] = item.Path
	}
	return paths
}

// FilterToDllExe filters paths to DLL/EXE/WINMD files
func FilterToDllExe(paths []string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".dll" || ext == ".exe" || ext == ".winmd" {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

// GetLibItems gets runtime assemblies for target framework
// Reference: LockFileUtils.cs CreateLockFileTargetLibrary (Lines 184-190)
func GetLibItems(files []string, targetFramework *frameworks.NuGetFramework, conventions *ManagedCodeConventions) []string {
	collection := NewContentItemCollection(files)
	criteria := ForFramework(targetFramework, conventions.Properties)

	paths := GetLockFileItems(criteria, collection, conventions.RuntimeAssemblies)
	return FilterToDllExe(paths)
}

// GetRefItems gets compile-time reference assemblies
// Reference: LockFileUtils.cs CreateLockFileTargetLibrary (Lines 177-183)
func GetRefItems(files []string, targetFramework *frameworks.NuGetFramework, conventions *ManagedCodeConventions) []string {
	collection := NewContentItemCollection(files)
	criteria := ForFramework(targetFramework, conventions.Properties)

	// Compile: ref takes precedence over lib
	return GetLockFileItems(criteria, collection, conventions.CompileRefAssemblies, conventions.CompileLibAssemblies)
}
```

### Verification Steps

```bash
# 1. Run ContentItemCollection tests
go test ./packaging/assets -v -run TestContentItemCollection

# 2. Test FindBestItemGroup selection
go test ./packaging/assets -v -run TestFindBestItemGroup

# 3. Test SelectionCriteria building
go test ./packaging/assets -v -run TestSelectionCriteria

# 4. Test framework resolution with FrameworkReducer
go test ./packaging/assets -v -run TestFrameworkResolution

# 5. Test GetLibItems/GetRefItems helpers
go test ./packaging/assets -v -run TestGetLibItems
go test ./packaging/assets -v -run TestGetRefItems

# 6. Test RID fallback behavior
go test ./packaging/assets -v -run TestRIDFallback

# 7. Run interop tests against NuGet.Client
cd tests/nuget-client-interop/GonugetInterop.Tests
dotnet test --filter "FullyQualifiedName~ContentModelTests"

# 8. Check test coverage (target: 90%+)
go test ./packaging/assets -cover
```

### Acceptance Criteria

- [ ] Implement `SelectionCriteria` with ordered entries for fallback
- [ ] Implement `SelectionCriteriaBuilder` with fluent API
- [ ] Implement `ForFramework()` helper for framework-only criteria
- [ ] Implement `ForFrameworkAndRuntime()` with RID fallback
- [ ] Implement `ContentItemCollection.PopulateItemGroups()`
- [ ] Implement `ContentItemCollection.FindBestItemGroup()` with property comparison
- [ ] Use `PropertyDefinition.Compare()` for framework selection
- [ ] Use `PropertyDefinition.IsCriteriaSatisfied()` for compatibility testing
- [ ] Implement `GetLockFileItems()` orchestration helper
- [ ] Implement `FilterToDllExe()` to filter assemblies
- [ ] Implement `GetLibItems()` for runtime assemblies
- [ ] Implement `GetRefItems()` for compile-time reference assemblies
- [ ] Handle ref/ precedence over lib/ for compile assemblies
- [ ] Handle RID-specific then RID-agnostic fallback
- [ ] Pass NuGet.Client interop tests (ContentModelTests.cs)
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement ContentModel asset selection with FindBestItemGroup

Add NuGet ContentModel asset selection matching NuGet.Client architecture:
- ContentItemCollection with PopulateItemGroups and FindBestItemGroup
- SelectionCriteria with ordered fallback entries (RID-specific → RID-agnostic)
- SelectionCriteriaBuilder with fluent API
- ForFramework and ForFrameworkAndRuntime helper functions
- PropertyDefinition.Compare for framework precedence
- PropertyDefinition.IsCriteriaSatisfied for compatibility testing
- GetLockFileItems orchestration helper
- GetLibItems for runtime assemblies (lib/ folder)
- GetRefItems for compile assemblies (ref/ precedence over lib/)
- FilterToDllExe for assembly filtering
- Integration with FrameworkReducer for nearest compatible framework

This replaces the simplified AssetSelector pattern with the actual
ContentItemCollection.FindBestItemGroup pattern used in NuGet.Client.

Reference:
- ContentItemCollection.cs (Lines 136-241) - FindBestItemGroup
- LockFileUtils.cs (Lines 663-713) - GetLockFileItems
- ManagedCodeConventions.cs (Lines 377-405) - Criteria building
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
