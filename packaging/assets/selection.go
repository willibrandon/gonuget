package assets

import "github.com/willibrandon/gonuget/frameworks"

// SelectionCriteriaEntry represents a single criteria entry with properties.
// Reference: ContentModel/SelectionCriteriaEntry.cs
type SelectionCriteriaEntry struct {
	Properties map[string]any
}

// SelectionCriteria contains ordered list of criteria entries for asset selection.
// Entries are tried in order, allowing fallback behavior (e.g., RID-specific → RID-agnostic).
// Reference: ContentModel/SelectionCriteria.cs
type SelectionCriteria struct {
	Entries []SelectionCriteriaEntry
}

// SelectionCriteriaBuilder builds selection criteria with fluent API.
// Reference: ContentModel/SelectionCriteriaBuilder.cs
type SelectionCriteriaBuilder struct {
	properties   map[string]*PropertyDefinition
	currentEntry SelectionCriteriaEntry
	entries      []SelectionCriteriaEntry
}

// NewSelectionCriteriaBuilder creates a new criteria builder.
func NewSelectionCriteriaBuilder(properties map[string]*PropertyDefinition) *SelectionCriteriaBuilder {
	return &SelectionCriteriaBuilder{
		properties:   properties,
		currentEntry: SelectionCriteriaEntry{Properties: make(map[string]any)},
	}
}

// Add sets a property value and returns builder for chaining.
func (b *SelectionCriteriaBuilder) Add(key string, value any) *SelectionCriteriaBuilder {
	b.currentEntry.Properties[key] = value
	return b
}

// NextEntry finalizes current entry and starts new one.
func (b *SelectionCriteriaBuilder) NextEntry() *SelectionCriteriaBuilder {
	if len(b.currentEntry.Properties) > 0 {
		b.entries = append(b.entries, b.currentEntry)
		b.currentEntry = SelectionCriteriaEntry{Properties: make(map[string]any)}
	}
	return b
}

// Build finalizes and returns the criteria.
func (b *SelectionCriteriaBuilder) Build() *SelectionCriteria {
	if len(b.currentEntry.Properties) > 0 {
		b.entries = append(b.entries, b.currentEntry)
	}
	return &SelectionCriteria{Entries: b.entries}
}

// ForFramework creates criteria for framework-only matching (no RID).
// Reference: ManagedCodeConventions.cs ForFramework (Lines 410-417)
func ForFramework(framework *frameworks.NuGetFramework, properties map[string]*PropertyDefinition) *SelectionCriteria {
	builder := NewSelectionCriteriaBuilder(properties)
	builder.Add("tfm", framework)
	builder.Add("rid", nil) // Explicitly no RID
	return builder.Build()
}

// ForFrameworkAndRuntime creates criteria with RID fallback.
// First tries RID-specific assets, then falls back to RID-agnostic assets.
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
