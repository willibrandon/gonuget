package assets

import (
	"github.com/willibrandon/gonuget/frameworks"
)

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
// Reference: ManagedCodeConventions.cs

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
