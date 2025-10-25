package resolver

import (
	"context"
	"testing"
)

func TestDependencyWalker_SimpleDependency(t *testing.T) {
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

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	if node.Item.ID != "A" {
		t.Errorf("Expected root A, got %s", node.Item.ID)
	}

	if len(node.InnerNodes) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(node.InnerNodes))
	}

	if node.InnerNodes[0].Item.ID != "B" {
		t.Errorf("Expected child B, got %s", node.InnerNodes[0].Item.ID)
	}

	// Verify Disposition
	if node.Disposition != DispositionAcceptable {
		t.Errorf("Expected root Disposition=Acceptable, got %v", node.Disposition)
	}
}

func TestDependencyWalker_CycleDetection(t *testing.T) {
	// A -> B -> A (cycle)
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
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "A", VersionRange: "[1.0.0]"}, // Cycle!
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() should not error on cycle, got: %v", err)
	}

	// Find the cycle node
	var cycleNode *GraphNode
	for _, child := range node.InnerNodes {
		if child.Item != nil && child.Item.ID == "B" {
			for _, grandchild := range child.InnerNodes {
				if grandchild.Disposition == DispositionCycle {
					cycleNode = grandchild
					break
				}
			}
		}
	}

	if cycleNode == nil {
		t.Fatal("Expected to find cycle node with DispositionCycle")
	}

	if cycleNode.Disposition != DispositionCycle {
		t.Errorf("Expected Disposition=Cycle, got %v", cycleNode.Disposition)
	}
}

func TestDependencyWalker_SuppressParent(t *testing.T) {
	// A -> B (with SuppressParent=All, should not walk B's dependencies)
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{
						ID:             "B",
						VersionRange:   "[1.0.0]",
						SuppressParent: LibraryIncludeFlagsAll, // PrivateAssets="All"
					},
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "C", VersionRange: "[1.0.0]"},
				},
			},
			"C|1.0.0": {
				ID:           "C",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should have B as child
	if len(node.InnerNodes) != 1 {
		t.Fatalf("Expected 1 child (B), got %d", len(node.InnerNodes))
	}

	// But B should have NO children (suppressed)
	bNode := node.InnerNodes[0]
	if len(bNode.InnerNodes) != 0 {
		t.Errorf("Expected B to have 0 children (suppressed), got %d", len(bNode.InnerNodes))
	}
}

func TestDependencyWalker_MultipleDependencies(t *testing.T) {
	// A -> B, C, D
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},
					{ID: "C", VersionRange: "[1.0.0]"},
					{ID: "D", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"C|1.0.0": {ID: "C", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	if len(node.InnerNodes) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(node.InnerNodes))
	}

	// Verify all children are present
	ids := make(map[string]bool)
	for _, child := range node.InnerNodes {
		ids[child.Item.ID] = true
	}

	for _, expected := range []string{"B", "C", "D"} {
		if !ids[expected] {
			t.Errorf("Expected child %s not found", expected)
		}
	}
}

func TestDependencyWalker_DeepDependencies(t *testing.T) {
	// A -> B -> C -> D
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:           "A",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "B", VersionRange: "[1.0.0]"}},
			},
			"B|1.0.0": {
				ID:           "B",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "C", VersionRange: "[1.0.0]"}},
			},
			"C|1.0.0": {
				ID:           "C",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "D", VersionRange: "[1.0.0]"}},
			},
			"D|1.0.0": {
				ID:           "D",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Verify depth
	if node.Depth != 0 {
		t.Errorf("Expected root depth 0, got %d", node.Depth)
	}

	// Navigate to D and verify depth
	b := node.InnerNodes[0]
	if b.Depth != 1 {
		t.Errorf("Expected B depth 1, got %d", b.Depth)
	}

	c := b.InnerNodes[0]
	if c.Depth != 2 {
		t.Errorf("Expected C depth 2, got %d", c.Depth)
	}

	d := c.InnerNodes[0]
	if d.Depth != 3 {
		t.Errorf("Expected D depth 3, got %d", d.Depth)
	}
}

func TestDependencyWalker_MissingPackage(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "Missing", VersionRange: "[1.0.0]"},
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should still succeed but have no children (missing package is skipped)
	if len(node.InnerNodes) != 0 {
		t.Errorf("Expected 0 children when dependency is missing, got %d", len(node.InnerNodes))
	}
}

func TestDependencyWalker_GraphEdgeChain(t *testing.T) {
	// A -> B -> C, verify edge chain
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:           "A",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "B", VersionRange: "[1.0.0]"}},
			},
			"B|1.0.0": {
				ID:           "B",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "C", VersionRange: "[1.0.0]"}},
			},
			"C|1.0.0": {
				ID:           "C",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Navigate to C
	b := node.InnerNodes[0]
	c := b.InnerNodes[0]

	// Verify edge chain: C -> B -> A
	if c.OuterEdge == nil {
		t.Fatal("Expected C to have OuterEdge")
	}

	if c.OuterEdge.Item.ID != "B" {
		t.Errorf("Expected C's OuterEdge.Item to be B, got %s", c.OuterEdge.Item.ID)
	}

	if c.OuterEdge.OuterEdge == nil {
		t.Fatal("Expected C's OuterEdge to have OuterEdge (B -> A)")
	}

	if c.OuterEdge.OuterEdge.Item.ID != "A" {
		t.Errorf("Expected B's OuterEdge.Item to be A, got %s", c.OuterEdge.OuterEdge.Item.ID)
	}
}

func TestGraphNode_PathFromRoot(t *testing.T) {
	// Create a simple graph: A -> B -> C
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:           "A",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "B", VersionRange: "[1.0.0]"}},
			},
			"B|1.0.0": {
				ID:           "B",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "C", VersionRange: "[1.0.0]"}},
			},
			"C|1.0.0": {
				ID:           "C",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")
	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Get path from C
	c := node.InnerNodes[0].InnerNodes[0]
	path := c.PathFromRoot()

	expected := []string{"A 1.0.0", "B 1.0.0", "C 1.0.0"}
	if len(path) != len(expected) {
		t.Fatalf("Expected path length %d, got %d", len(expected), len(path))
	}

	for i, p := range path {
		if p != expected[i] {
			t.Errorf("Expected path[%d] = %s, got %s", i, expected[i], p)
		}
	}
}

func TestDependencyWalker_FrameworkSpecificDependencies(t *testing.T) {
	// A with framework-specific dependencies
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				DependencyGroups: []DependencyGroup{
					{
						TargetFramework: "net8.0",
						Dependencies: []PackageDependency{
							{ID: "B", VersionRange: "[1.0.0]"},
						},
					},
					{
						TargetFramework: "net6.0",
						Dependencies: []PackageDependency{
							{ID: "C", VersionRange: "[1.0.0]"},
						},
					},
				},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"C|1.0.0": {ID: "C", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should only have B (net8.0 specific), not C (net6.0 specific)
	if len(node.InnerNodes) != 1 {
		t.Fatalf("Expected 1 child (B for net8.0), got %d", len(node.InnerNodes))
	}

	if node.InnerNodes[0].Item.ID != "B" {
		t.Errorf("Expected child B, got %s", node.InnerNodes[0].Item.ID)
	}
}

func TestDependencyWalker_ContextCancellation(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:           "A",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "B", VersionRange: "[1.0.0]"}},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := walker.Walk(ctx, "A", "[1.0.0]", "net8.0")

	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
}

func TestDependencyWalker_CachingWorks(t *testing.T) {
	callCount := 0
	client := &mockPackageMetadataClientWithCounter{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:           "A",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{{ID: "B", VersionRange: "[1.0.0]"}},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
		callCount: &callCount,
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	// First walk
	_, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")
	if err != nil {
		t.Fatalf("First Walk() failed: %v", err)
	}

	firstCallCount := callCount

	// Second walk - should use cache
	_, err = walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")
	if err != nil {
		t.Fatalf("Second Walk() failed: %v", err)
	}

	if callCount != firstCallCount {
		t.Errorf("Expected caching to avoid additional calls, got %d calls for second walk", callCount-firstCallCount)
	}
}

func TestDisposition_String(t *testing.T) {
	tests := []struct {
		d        Disposition
		expected string
	}{
		{DispositionAcceptable, "Acceptable"},
		{DispositionRejected, "Rejected"},
		{DispositionAccepted, "Accepted"},
		{DispositionPotentiallyDowngraded, "PotentiallyDowngraded"},
		{DispositionCycle, "Cycle"},
		{Disposition(999), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.d.String(); got != tt.expected {
			t.Errorf("Disposition(%d).String() = %s, want %s", tt.d, got, tt.expected)
		}
	}
}

func TestGraphNode_AreAllParentsRejected(t *testing.T) {
	// No parents
	node1 := &GraphNode{ParentNodes: []*GraphNode{}}
	if node1.AreAllParentsRejected() {
		t.Error("Expected false when no parents, got true")
	}

	// All parents rejected
	node2 := &GraphNode{
		ParentNodes: []*GraphNode{
			{Disposition: DispositionRejected},
			{Disposition: DispositionRejected},
		},
	}
	if !node2.AreAllParentsRejected() {
		t.Error("Expected true when all parents rejected, got false")
	}

	// Mix of rejected and accepted
	node3 := &GraphNode{
		ParentNodes: []*GraphNode{
			{Disposition: DispositionRejected},
			{Disposition: DispositionAcceptable},
		},
	}
	if node3.AreAllParentsRejected() {
		t.Error("Expected false when not all parents rejected, got true")
	}
}

func TestDependencyWalker_RootPackageNotFound(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	_, err := walker.Walk(context.Background(), "Missing", "[1.0.0]", "net8.0")

	if err == nil {
		t.Error("Expected error when root package not found, got nil")
	}
}

func TestDependencyWalker_EmptyFrameworkGroup(t *testing.T) {
	// Package with empty framework group (should use fallback)
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				DependencyGroups: []DependencyGroup{
					{
						TargetFramework: "",
						Dependencies: []PackageDependency{
							{ID: "B", VersionRange: "[1.0.0]"},
						},
					},
				},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should use empty framework group and get B
	if len(node.InnerNodes) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(node.InnerNodes))
	}

	if node.InnerNodes[0].Item.ID != "B" {
		t.Errorf("Expected child B, got %s", node.InnerNodes[0].Item.ID)
	}
}

func TestGraphNode_PathFromRoot_Nil(t *testing.T) {
	var node *GraphNode = nil
	path := node.PathFromRoot()

	if path != nil {
		t.Errorf("Expected nil path from nil node, got %v", path)
	}
}

func TestDependencyWalker_CalculateDependencyResult(t *testing.T) {
	walker := NewDependencyWalker(nil, []string{}, "net8.0")

	// Test direct cycle (current key matches dependency)
	result, _ := walker.calculateDependencyResult(
		nil,
		PackageDependency{ID: "A", VersionRange: "[1.0.0]"},
		"A|1.0.0",
	)
	if result != DependencyResultCycle {
		t.Errorf("Expected DependencyResultCycle for direct cycle, got %v", result)
	}

	// Test cycle through parent chain
	edge := &GraphEdge{
		OuterEdge: nil,
		Item: &PackageDependencyInfo{
			ID:      "A",
			Version: "1.0.0",
		},
		Edge: PackageDependency{ID: "B", VersionRange: "[1.0.0]"},
	}

	result, _ = walker.calculateDependencyResult(
		edge,
		PackageDependency{ID: "A", VersionRange: "[1.0.0]"},
		"B|1.0.0",
	)
	if result != DependencyResultCycle {
		t.Errorf("Expected DependencyResultCycle for parent chain cycle, got %v", result)
	}

	// Test acceptable (no cycle, no downgrade)
	result, _ = walker.calculateDependencyResult(
		edge,
		PackageDependency{ID: "C", VersionRange: "[1.0.0]"},
		"B|1.0.0",
	)
	if result != DependencyResultAcceptable {
		t.Errorf("Expected DependencyResultAcceptable, got %v", result)
	}

	// Test with nil edge item
	edgeWithNilItem := &GraphEdge{
		OuterEdge: edge,
		Item:      nil,
		Edge:      PackageDependency{ID: "X", VersionRange: "[1.0.0]"},
	}

	result, _ = walker.calculateDependencyResult(
		edgeWithNilItem,
		PackageDependency{ID: "A", VersionRange: "[1.0.0]"},
		"Y|1.0.0",
	)
	// Should skip nil item and check outer edge
	if result != DependencyResultCycle {
		t.Errorf("Expected DependencyResultCycle after skipping nil item, got %v", result)
	}
}

// Mock client with counter
type mockPackageMetadataClientWithCounter struct {
	packages  map[string]*PackageDependencyInfo
	callCount *int
}

func (m *mockPackageMetadataClientWithCounter) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*PackageDependencyInfo, error) {
	*m.callCount++
	result := make([]*PackageDependencyInfo, 0)
	for _, pkg := range m.packages {
		if pkg.ID == packageID {
			result = append(result, pkg)
		}
	}
	return result, nil
}

// Mock client
type mockPackageMetadataClient struct {
	packages map[string]*PackageDependencyInfo
}

func (m *mockPackageMetadataClient) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*PackageDependencyInfo, error) {
	result := make([]*PackageDependencyInfo, 0)
	for _, pkg := range m.packages {
		if pkg.ID == packageID {
			result = append(result, pkg)
		}
	}
	return result, nil
}

func TestDependencyWalker_MultipleVersions(t *testing.T) {
	// Test version range matching with multiple versions available
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0,2.0.0)"},
				},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"B|1.5.0": {ID: "B", Version: "1.5.0", Dependencies: []PackageDependency{}},
			"B|2.0.0": {ID: "B", Version: "2.0.0", Dependencies: []PackageDependency{}},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should select highest version in range [1.0.0, 2.0.0) which is 1.5.0
	if len(node.InnerNodes) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(node.InnerNodes))
	}

	if node.InnerNodes[0].Item.Version != "1.5.0" {
		t.Errorf("Expected B version 1.5.0 (highest in range), got %s", node.InnerNodes[0].Item.Version)
	}
}

func TestDependencyWalker_InvalidVersionRange(t *testing.T) {
	// Test error handling for invalid version range
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "invalid-range"},
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	_, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err == nil {
		t.Error("Expected error for invalid version range, got nil")
	}
}

func TestDependencyWalker_NoMatchingFramework(t *testing.T) {
	// Test when no framework group matches (should return nil)
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				DependencyGroups: []DependencyGroup{
					{
						TargetFramework: "net6.0",
						Dependencies: []PackageDependency{
							{ID: "B", VersionRange: "[1.0.0]"},
						},
					},
				},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Should have no children (no matching framework group)
	if len(node.InnerNodes) != 0 {
		t.Errorf("Expected 0 children when framework doesn't match, got %d", len(node.InnerNodes))
	}
}

func TestDependencyWalker_StackTraversalOrder(t *testing.T) {
	// Test that stack-based traversal produces correct depth-first order
	// A -> B, C
	// B -> D
	// C -> E
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
					{ID: "E", VersionRange: "[1.0.0]"},
				},
			},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"E|1.0.0": {ID: "E", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")

	node, err := walker.Walk(context.Background(), "A", "[1.0.0]", "net8.0")

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Verify structure: A should have B and C as children
	if len(node.InnerNodes) != 2 {
		t.Fatalf("Expected 2 children for A, got %d", len(node.InnerNodes))
	}

	// B should have D as child
	b := node.InnerNodes[0]
	if b.Item.ID != "B" {
		t.Errorf("Expected first child to be B, got %s", b.Item.ID)
	}
	if len(b.InnerNodes) != 1 || b.InnerNodes[0].Item.ID != "D" {
		t.Error("Expected B to have child D")
	}

	// C should have E as child
	c := node.InnerNodes[1]
	if c.Item.ID != "C" {
		t.Errorf("Expected second child to be C, got %s", c.Item.ID)
	}
	if len(c.InnerNodes) != 1 || c.InnerNodes[0].Item.ID != "E" {
		t.Error("Expected C to have child E")
	}

	// Verify depths are correct
	if b.Depth != 1 {
		t.Errorf("Expected B depth 1, got %d", b.Depth)
	}
	if c.Depth != 1 {
		t.Errorf("Expected C depth 1, got %d", c.Depth)
	}
	if b.InnerNodes[0].Depth != 2 {
		t.Errorf("Expected D depth 2, got %d", b.InnerNodes[0].Depth)
	}
	if c.InnerNodes[0].Depth != 2 {
		t.Errorf("Expected E depth 2, got %d", c.InnerNodes[0].Depth)
	}
}
