package assets

import (
	"strings"

	"github.com/willibrandon/gonuget/frameworks"
)

// ManagedCodeConventions defines standard .NET package conventions.
// Reference: ContentModel/ManagedCodeConventions.cs
type ManagedCodeConventions struct {
	// Properties available for pattern matching
	Properties map[string]*PropertyDefinition

	// Pattern sets for different asset types
	RuntimeAssemblies     *PatternSet
	CompileRefAssemblies  *PatternSet
	CompileLibAssemblies  *PatternSet
	NativeLibraries       *PatternSet
	ResourceAssemblies    *PatternSet
	MSBuildFiles          *PatternSet
	MSBuildMultiTargeting *PatternSet
	ContentFiles          *PatternSet
	ToolsAssemblies       *PatternSet
}

// NewManagedCodeConventions creates standard managed code conventions.
// Reference: ManagedCodeConventions constructor
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
	// Reference: ManagedCodeConventions.cs
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
	// Reference: ManagedCodeConventions.cs
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
// Reference: ManagedCodeConventions.cs

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
// Reference: ManagedCodeConventions.cs

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

	// Check if the available framework is compatible with the criterion (target)
	// This asks: "Can a package targeting availableFW be used in a project targeting criterionFW?"
	return frameworks.IsCompatible(availableFW, criterionFW)
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
	// Nil criterion means RID-agnostic, compatible with anything
	if criterion == nil {
		return available == nil
	}

	// Nil available means RID-agnostic, only compatible with nil criterion
	if available == nil {
		return criterion == nil
	}

	// Both are strings - use default RID graph for compatibility
	criterionStr, ok1 := criterion.(string)
	availableStr, ok2 := available.(string)

	if !ok1 || !ok2 {
		return criterion == available
	}

	// Load default RID graph for compatibility checking
	graph := LoadDefaultRuntimeGraph()
	return graph.AreCompatible(criterionStr, availableStr)
}

func (c *ManagedCodeConventions) definePatternSets() {
	// Define pattern sets for each asset type
	// Reference: ManagedCodeConventions.cs ManagedCodePatterns

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

	// Additional pattern sets (stubs for now, to be filled in later chunks)
	c.NativeLibraries = NewPatternSet(c.Properties, nil, nil)
	c.ResourceAssemblies = NewPatternSet(c.Properties, nil, nil)
	c.MSBuildFiles = NewPatternSet(c.Properties, nil, nil)
	c.MSBuildMultiTargeting = NewPatternSet(c.Properties, nil, nil)
	c.ContentFiles = NewPatternSet(c.Properties, nil, nil)
	c.ToolsAssemblies = NewPatternSet(c.Properties, nil, nil)
}
