package resolver

import (
	"sort"

	"github.com/willibrandon/gonuget/version"
)

// ConflictResolver resolves version conflicts using nearest-wins.
type ConflictResolver struct{}

// NewConflictResolver creates a new conflict resolver.
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// ResolveConflict resolves a conflict by selecting the nearest (lowest depth) version.
// If depths are equal, selects highest version (matches NuGet.Client).
func (cr *ConflictResolver) ResolveConflict(nodes []*GraphNode) *GraphNode {
	if len(nodes) == 0 {
		return nil
	}

	if len(nodes) == 1 {
		return nodes[0]
	}

	// Sort by depth (ascending), then version (descending)
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Depth != nodes[j].Depth {
			return nodes[i].Depth < nodes[j].Depth // Lower depth wins
		}

		// Same depth - use version comparison (higher wins)
		if nodes[i].Item == nil || nodes[j].Item == nil {
			return false
		}

		vi, _ := version.Parse(nodes[i].Item.Version)
		vj, _ := version.Parse(nodes[j].Item.Version)

		return vi.Compare(vj) > 0 // Higher version wins
	})

	return nodes[0]
}
