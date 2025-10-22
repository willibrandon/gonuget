package assets

import (
	"strings"
)

// PropertyParser parses a property value from a string.
// Reference: ContentPropertyDefinition.cs Parser
//
// Parameters:
//   - value: substring to parse (no allocation)
//   - table: pattern table for token replacement
//   - matchOnly: if true, skip value actualization (performance optimization)
//
// Returns parsed value or nil if doesn't match.
type PropertyParser func(value string, table *PatternTable, matchOnly bool) any

// CompatibilityTest checks if criterion is compatible with available value.
// Reference: ContentPropertyDefinition.cs CompatibilityTest
type CompatibilityTest func(criterion, available any) bool

// CompareTest determines which of two available values is nearer to criterion.
// Reference: ContentPropertyDefinition.cs CompareTest
// Returns: -1 if available1 is nearer, 1 if available2 is nearer, 0 if equal
type CompareTest func(criterion, available1, available2 any) int

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
// Reference: ContentPropertyDefinition.cs TryLookup
func (pd *PropertyDefinition) TryLookup(name string, table *PatternTable, matchOnly bool) (any, bool) {
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
func (pd *PropertyDefinition) IsCriteriaSatisfied(criteriaValue, candidateValue any) bool {
	if pd.CompatibilityTest == nil {
		// Default: exact equality
		return criteriaValue == candidateValue
	}
	return pd.CompatibilityTest(criteriaValue, candidateValue)
}

// Compare compares two candidate values against a criterion.
// Reference: ContentPropertyDefinition.cs Compare
func (pd *PropertyDefinition) Compare(criteriaValue, candidateValue1, candidateValue2 any) int {
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
