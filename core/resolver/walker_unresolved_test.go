package resolver

import (
	"context"
	"testing"
)

// TestDependencyWalker_UnresolvedRootPackage verifies that when the root package
// doesn't exist, the walker creates an unresolved node and continues instead of failing.
// Matches NuGet.Client's ResolverUtility.CreateUnresolvedResult behavior.
func TestDependencyWalker_UnresolvedRootPackage(t *testing.T) {
	// Empty packages map - nothing exists
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "NonExistent", "[1.0.0]", "net8.0", true)

	// Should NOT error - should create unresolved node instead
	if err != nil {
		t.Fatalf("Walk() should not error on missing package, got: %v", err)
	}

	// Root node should be created
	if node == nil {
		t.Fatal("Expected root node to be created for unresolved package")
	}

	// Root node should be marked as unresolved
	if node.Item == nil {
		t.Fatal("Expected root node to have Item (unresolved package)")
	}

	if !node.Item.IsUnresolved {
		t.Error("Expected root node to be marked as unresolved")
	}

	// Verify package ID and version range are preserved
	if node.Item.ID != "NonExistent" {
		t.Errorf("Expected package ID 'NonExistent', got %s", node.Item.ID)
	}

	if node.Item.Version != "[1.0.0]" {
		t.Errorf("Expected version range '[1.0.0]' preserved, got %s", node.Item.Version)
	}

	// Should have no dependencies
	if len(node.Item.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies for unresolved package, got %d", len(node.Item.Dependencies))
	}
}

// TestDependencyWalker_UnresolvedTransitiveDependency verifies that when a transitive
// dependency doesn't exist, the walker creates an unresolved node and continues walking.
// Matches NuGet.Client behavior: complete graph walk, report all errors together.
func TestDependencyWalker_UnresolvedTransitiveDependency(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"}, // B exists
					{ID: "C", VersionRange: "[1.0.0]"}, // C does NOT exist
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "D", VersionRange: "[1.0.0]"}, // D does NOT exist
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0", true)

	// Should NOT error - should continue walking
	if err != nil {
		t.Fatalf("Walk() should not error on missing dependencies, got: %v", err)
	}

	// Should have 2 children (B and C)
	if len(node.InnerNodes) != 2 {
		t.Fatalf("Expected 2 children (B and C), got %d", len(node.InnerNodes))
	}

	// Find B and C nodes
	var bNode, cNode *GraphNode
	for _, child := range node.InnerNodes {
		switch child.Item.ID {
		case "B":
			bNode = child
		case "C":
			cNode = child
		}
	}

	// Verify B exists and is resolved
	if bNode == nil {
		t.Fatal("Expected to find B node")
	}
	if bNode.Item.IsUnresolved {
		t.Error("Expected B to be resolved (not unresolved)")
	}

	// Verify C exists and is unresolved
	if cNode == nil {
		t.Fatal("Expected to find C node (unresolved)")
	}
	if !cNode.Item.IsUnresolved {
		t.Error("Expected C to be marked as unresolved")
	}
	if cNode.Item.Version != "[1.0.0]" {
		t.Errorf("Expected C version range '[1.0.0]', got %s", cNode.Item.Version)
	}

	// Verify B has child D (unresolved)
	if len(bNode.InnerNodes) != 1 {
		t.Fatalf("Expected B to have 1 child (D), got %d", len(bNode.InnerNodes))
	}

	dNode := bNode.InnerNodes[0]
	if dNode.Item.ID != "D" {
		t.Errorf("Expected D node, got %s", dNode.Item.ID)
	}
	if !dNode.Item.IsUnresolved {
		t.Error("Expected D to be marked as unresolved")
	}
}

// TestDependencyWalker_MultipleUnresolvedPackages verifies that the walker collects
// multiple unresolved packages and continues walking the entire graph.
// Matches NuGet.Client: never fail during walk, collect all errors, report together.
func TestDependencyWalker_MultipleUnresolvedPackages(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "Missing1", VersionRange: "[1.0.0]"},
					{ID: "Missing2", VersionRange: "[2.0.0]"},
					{ID: "Missing3", VersionRange: "[3.0.0]"},
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0", true)

	// Should NOT error
	if err != nil {
		t.Fatalf("Walk() should not error, got: %v", err)
	}

	// Should have 3 children (all unresolved)
	if len(node.InnerNodes) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(node.InnerNodes))
	}

	// All children should be unresolved
	for i, child := range node.InnerNodes {
		if !child.Item.IsUnresolved {
			t.Errorf("Child %d (%s) should be unresolved", i, child.Item.ID)
		}
	}

	// Verify IDs are correct
	expectedIDs := map[string]bool{
		"Missing1": false,
		"Missing2": false,
		"Missing3": false,
	}

	for _, child := range node.InnerNodes {
		if _, exists := expectedIDs[child.Item.ID]; exists {
			expectedIDs[child.Item.ID] = true
		} else {
			t.Errorf("Unexpected child ID: %s", child.Item.ID)
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected to find child with ID %s", id)
		}
	}
}

// TestDependencyWalker_UnresolvedWithResolvedSiblings verifies that unresolved
// packages don't prevent siblings from being resolved correctly.
func TestDependencyWalker_UnresolvedWithResolvedSiblings(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},       // Exists
					{ID: "Missing", VersionRange: "[1.0.0]"}, // Missing
					{ID: "C", VersionRange: "[1.0.0]"},       // Exists
				},
			},
			"B|1.0.0": {
				ID:           "B",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
			"C|1.0.0": {
				ID:           "C",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0", true)

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should have 3 children
	if len(node.InnerNodes) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(node.InnerNodes))
	}

	// Count resolved vs unresolved
	resolvedCount := 0
	unresolvedCount := 0

	for _, child := range node.InnerNodes {
		if child.Item.IsUnresolved {
			unresolvedCount++
		} else {
			resolvedCount++
		}
	}

	if resolvedCount != 2 {
		t.Errorf("Expected 2 resolved children (B, C), got %d", resolvedCount)
	}

	if unresolvedCount != 1 {
		t.Errorf("Expected 1 unresolved child (Missing), got %d", unresolvedCount)
	}
}

// TestDependencyWalker_UnresolvedDoesNotAffectCycleDetection verifies that
// unresolved packages don't interfere with cycle detection logic.
func TestDependencyWalker_UnresolvedDoesNotAffectCycleDetection(t *testing.T) {
	// A -> B -> A (cycle)
	// A -> C (unresolved)
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},
					{ID: "C", VersionRange: "[1.0.0]"}, // Unresolved
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "A", VersionRange: "[1.0.0]"}, // Cycle
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0", true)

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should have 2 children (B and C)
	if len(node.InnerNodes) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(node.InnerNodes))
	}

	// Find B node
	var bNode *GraphNode
	for _, child := range node.InnerNodes {
		if child.Item.ID == "B" && !child.Item.IsUnresolved {
			bNode = child
			break
		}
	}

	if bNode == nil {
		t.Fatal("Expected to find resolved B node")
	}

	// B should have cycle child
	if len(bNode.InnerNodes) != 1 {
		t.Fatalf("Expected B to have 1 child (cycle), got %d", len(bNode.InnerNodes))
	}

	cycleNode := bNode.InnerNodes[0]
	if cycleNode.Disposition != DispositionCycle {
		t.Errorf("Expected cycle disposition, got %v", cycleNode.Disposition)
	}
}
