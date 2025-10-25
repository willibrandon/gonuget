package resolver

import (
	"context"
	"testing"
)

func TestResolver_NoConflict(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {
				ID:           "B",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(result.Conflicts))
	}

	// Should have A and B
	if len(result.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(result.Packages))
	}
}

func TestResolver_ConflictResolution_NearestWins(t *testing.T) {
	// A -> B -> D[1.0.0] (depth 2)
	// A -> C -> D[2.0.0] (depth 2)
	// Same depth, higher version should win (2.0.0)

	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},
					{ID: "C", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[1.0.0]"},
				},
			},
			"C|1.0.0": {
				ID:      "C",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[2.0.0]"},
				},
			},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"D|2.0.0": {ID: "D", Version: "2.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	// Should detect conflict
	if len(result.Conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(result.Conflicts))
	}

	// Find resolved D version
	var dVersion string
	for _, pkg := range result.Packages {
		if pkg.ID == "D" {
			dVersion = pkg.Version
			break
		}
	}

	// Should resolve to D 2.0.0 (higher version at same depth)
	if dVersion != "2.0.0" {
		t.Errorf("Expected D 2.0.0 to win, got %s", dVersion)
	}
}

func TestResolver_ConflictResolution_DepthWins(t *testing.T) {
	// A -> D[1.0.0] (depth 1) - should win
	// A -> B -> D[2.0.0] (depth 2)

	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[1.0.0]"},
					{ID: "B", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[2.0.0]"},
				},
			},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"D|2.0.0": {ID: "D", Version: "2.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	// Find resolved D version
	var dVersion string
	for _, pkg := range result.Packages {
		if pkg.ID == "D" {
			dVersion = pkg.Version
			break
		}
	}

	// Should resolve to D 1.0.0 (nearest, depth 1)
	if dVersion != "1.0.0" {
		t.Errorf("Expected D 1.0.0 to win (nearest), got %s", dVersion)
	}
}

func TestResolver_MultipleConflicts(t *testing.T) {
	// A -> B -> D[1.0.0], E[1.0.0]
	// A -> C -> D[2.0.0], E[2.0.0]
	// Should resolve both conflicts

	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},
					{ID: "C", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[1.0.0]"},
					{ID: "E", VersionRange: "[1.0.0]"},
				},
			},
			"C|1.0.0": {
				ID:      "C",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[2.0.0]"},
					{ID: "E", VersionRange: "[2.0.0]"},
				},
			},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"D|2.0.0": {ID: "D", Version: "2.0.0", Dependencies: []PackageDependency{}},
			"E|1.0.0": {ID: "E", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"E|2.0.0": {ID: "E", Version: "2.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	// Should detect 2 conflicts (D and E)
	if len(result.Conflicts) != 2 {
		t.Errorf("Expected 2 conflicts, got %d", len(result.Conflicts))
	}

	// Both should be resolved
	if len(result.Packages) < 3 {
		t.Errorf("Expected at least 3 packages (A, D, E), got %d", len(result.Packages))
	}
}

func TestResolver_EmptyInput(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	_, err := resolver.Resolve(context.Background(), "NonExistent", "[1.0.0]")

	if err == nil {
		t.Error("Expected error for non-existent package, got nil")
	}
}

func TestConflictResolver_SingleNode(t *testing.T) {
	resolver := NewConflictResolver()

	node := &GraphNode{
		Key:   "A|1.0.0",
		Item:  &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		Depth: 0,
	}

	result := resolver.ResolveConflict([]*GraphNode{node})

	if result != node {
		t.Error("Expected single node to be returned")
	}
}

func TestConflictResolver_EmptyNodes(t *testing.T) {
	resolver := NewConflictResolver()

	result := resolver.ResolveConflict([]*GraphNode{})

	if result != nil {
		t.Error("Expected nil for empty nodes")
	}
}

func TestConflictResolver_NilItems(t *testing.T) {
	resolver := NewConflictResolver()

	nodes := []*GraphNode{
		{Key: "A|1.0.0", Item: nil, Depth: 0},
		{Key: "A|2.0.0", Item: nil, Depth: 1},
	}

	result := resolver.ResolveConflict(nodes)

	// Should return first node when items are nil
	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestConflictResolver_VersionTiebreaker(t *testing.T) {
	resolver := NewConflictResolver()

	nodes := []*GraphNode{
		{
			Key:   "A|1.0.0",
			Item:  &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
			Depth: 1,
		},
		{
			Key:   "A|2.0.0",
			Item:  &PackageDependencyInfo{ID: "A", Version: "2.0.0"},
			Depth: 1,
		},
		{
			Key:   "A|1.5.0",
			Item:  &PackageDependencyInfo{ID: "A", Version: "1.5.0"},
			Depth: 1,
		},
	}

	result := resolver.ResolveConflict(nodes)

	// Should select highest version (2.0.0) when depths are equal
	if result.Item.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", result.Item.Version)
	}
}

func TestConflictResolver_DepthPriority(t *testing.T) {
	resolver := NewConflictResolver()

	nodes := []*GraphNode{
		{
			Key:   "A|3.0.0",
			Item:  &PackageDependencyInfo{ID: "A", Version: "3.0.0"},
			Depth: 2,
		},
		{
			Key:   "A|1.0.0",
			Item:  &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
			Depth: 0,
		},
		{
			Key:   "A|2.0.0",
			Item:  &PackageDependencyInfo{ID: "A", Version: "2.0.0"},
			Depth: 1,
		},
	}

	result := resolver.ResolveConflict(nodes)

	// Should select lowest depth (0) even though version is lower
	if result.Depth != 0 {
		t.Errorf("Expected depth 0, got %d", result.Depth)
	}
	if result.Item.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0 at depth 0, got %s", result.Item.Version)
	}
}
