// Package assets implements NuGet's pattern-based asset selection engine.
// This system matches package files to target frameworks and runtime identifiers
// using property-based pattern definitions.
//
// Reference: NuGet.Packaging/ContentModel/
package assets

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
