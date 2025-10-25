package resolver

import (
	"testing"
)

func TestConflictDetector_NoConflict(t *testing.T) {
	// Simple tree: A -> B -> C (no conflicts)
	nodeC := &GraphNode{
		Key:        "C|1.0.0",
		Item:       &PackageDependencyInfo{ID: "C", Version: "1.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      2,
	}

	nodeB := &GraphNode{
		Key:        "B|1.0.0",
		Item:       &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeC},
		Depth:      1,
	}
	nodeC.OuterNode = nodeB

	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeB},
		Depth:      0,
	}
	nodeB.OuterNode = nodeA

	detector := NewConflictDetector()
	conflicts, downgrades := detector.DetectFromGraph(nodeA)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, found %d", len(conflicts))
	}

	if len(downgrades) != 0 {
		t.Errorf("Expected no downgrades, found %d", len(downgrades))
	}
}

func TestConflictDetector_SimpleConflict(t *testing.T) {
	//     A
	//    / \
	//   B   C
	//   |   |
	//   D1  D2  (conflict on D)

	nodeD1 := &GraphNode{
		Key:        "D|1.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "1.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      2,
	}

	nodeD2 := &GraphNode{
		Key:        "D|2.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "2.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      2,
	}

	nodeB := &GraphNode{
		Key:        "B|1.0.0",
		Item:       &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeD1},
		Depth:      1,
	}
	nodeD1.OuterNode = nodeB

	nodeC := &GraphNode{
		Key:        "C|1.0.0",
		Item:       &PackageDependencyInfo{ID: "C", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeD2},
		Depth:      1,
	}
	nodeD2.OuterNode = nodeC

	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeB, nodeC},
		Depth:      0,
	}
	nodeB.OuterNode = nodeA
	nodeC.OuterNode = nodeA

	detector := NewConflictDetector()
	conflicts, _ := detector.DetectFromGraph(nodeA)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, found %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.PackageID != "D" {
		t.Errorf("Expected conflict on D, got %s", conflict.PackageID)
	}

	if len(conflict.Versions) != 2 {
		t.Errorf("Expected 2 versions in conflict, got %d", len(conflict.Versions))
	}
}

func TestConflictDetector_MultipleConflicts(t *testing.T) {
	//     A
	//    / \
	//   B   C
	//   |   |
	//   D1  D2  (conflict on D)
	//   |   |
	//   E1  E2  (conflict on E)

	nodeE1 := &GraphNode{
		Key:        "E|1.0.0",
		Item:       &PackageDependencyInfo{ID: "E", Version: "1.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      3,
	}

	nodeE2 := &GraphNode{
		Key:        "E|2.0.0",
		Item:       &PackageDependencyInfo{ID: "E", Version: "2.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      3,
	}

	nodeD1 := &GraphNode{
		Key:        "D|1.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeE1},
		Depth:      2,
	}
	nodeE1.OuterNode = nodeD1

	nodeD2 := &GraphNode{
		Key:        "D|2.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "2.0.0"},
		InnerNodes: []*GraphNode{nodeE2},
		Depth:      2,
	}
	nodeE2.OuterNode = nodeD2

	nodeB := &GraphNode{
		Key:        "B|1.0.0",
		Item:       &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeD1},
		Depth:      1,
	}
	nodeD1.OuterNode = nodeB

	nodeC := &GraphNode{
		Key:        "C|1.0.0",
		Item:       &PackageDependencyInfo{ID: "C", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeD2},
		Depth:      1,
	}
	nodeD2.OuterNode = nodeC

	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeB, nodeC},
		Depth:      0,
	}
	nodeB.OuterNode = nodeA
	nodeC.OuterNode = nodeA

	detector := NewConflictDetector()
	conflicts, _ := detector.DetectFromGraph(nodeA)

	if len(conflicts) != 2 {
		t.Fatalf("Expected 2 conflicts, found %d", len(conflicts))
	}

	// Verify both D and E have conflicts
	foundD := false
	foundE := false
	for _, conflict := range conflicts {
		if conflict.PackageID == "D" {
			foundD = true
			if len(conflict.Versions) != 2 {
				t.Errorf("Expected 2 versions for D, got %d", len(conflict.Versions))
			}
		}
		if conflict.PackageID == "E" {
			foundE = true
			if len(conflict.Versions) != 2 {
				t.Errorf("Expected 2 versions for E, got %d", len(conflict.Versions))
			}
		}
	}

	if !foundD {
		t.Error("Expected conflict on D")
	}
	if !foundE {
		t.Error("Expected conflict on E")
	}
}

func TestConflictDetector_DowngradeWarning(t *testing.T) {
	// Tree with downgrade node
	nodeB := &GraphNode{
		Key:         "B|1.0.0",
		Item:        &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes:  []*GraphNode{},
		Disposition: DispositionPotentiallyDowngraded,
		Depth:       1,
	}

	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeB},
		Depth:      0,
	}
	nodeB.OuterNode = nodeA

	detector := NewConflictDetector()
	_, downgrades := detector.DetectFromGraph(nodeA)

	if len(downgrades) != 1 {
		t.Fatalf("Expected 1 downgrade, found %d", len(downgrades))
	}

	downgrade := downgrades[0]
	if downgrade.PackageID != "B" {
		t.Errorf("Expected downgrade on B, got %s", downgrade.PackageID)
	}

	if downgrade.TargetVersion != "1.0.0" {
		t.Errorf("Expected target version 1.0.0, got %s", downgrade.TargetVersion)
	}
}

func TestConflictDetector_NilNode(t *testing.T) {
	detector := NewConflictDetector()
	conflicts, downgrades := detector.DetectFromGraph(nil)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts for nil node, found %d", len(conflicts))
	}

	if len(downgrades) != 0 {
		t.Errorf("Expected no downgrades for nil node, found %d", len(downgrades))
	}
}

func TestConflictDetector_EmptyGraph(t *testing.T) {
	// Node with no children
	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      0,
	}

	detector := NewConflictDetector()
	conflicts, downgrades := detector.DetectFromGraph(nodeA)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts for single node, found %d", len(conflicts))
	}

	if len(downgrades) != 0 {
		t.Errorf("Expected no downgrades for single node, found %d", len(downgrades))
	}
}

func TestConflictDetector_ConflictPaths(t *testing.T) {
	//     A
	//    / \
	//   B   C
	//   |   |
	//   D1  D2  (conflict on D)

	nodeD1 := &GraphNode{
		Key:        "D|1.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "1.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      2,
	}

	nodeD2 := &GraphNode{
		Key:        "D|2.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "2.0.0"},
		InnerNodes: []*GraphNode{},
		Depth:      2,
	}

	nodeB := &GraphNode{
		Key:        "B|1.0.0",
		Item:       &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeD1},
		Depth:      1,
	}
	nodeD1.OuterNode = nodeB

	nodeC := &GraphNode{
		Key:        "C|1.0.0",
		Item:       &PackageDependencyInfo{ID: "C", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeD2},
		Depth:      1,
	}
	nodeD2.OuterNode = nodeC

	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeB, nodeC},
		Depth:      0,
	}
	nodeB.OuterNode = nodeA
	nodeC.OuterNode = nodeA

	detector := NewConflictDetector()
	conflicts, _ := detector.DetectFromGraph(nodeA)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, found %d", len(conflicts))
	}

	conflict := conflicts[0]
	if len(conflict.Paths) != 2 {
		t.Fatalf("Expected 2 paths, got %d", len(conflict.Paths))
	}

	// Verify paths are tracked
	for i, path := range conflict.Paths {
		if len(path) == 0 {
			t.Errorf("Path %d is empty", i)
		}
	}
}
