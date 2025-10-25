package resolver

import (
	"testing"
)

func TestCycleAnalyzer_SimpleCycle(t *testing.T) {
	// Build graph: A -> B -> (cycle to A)
	cycleNode := &GraphNode{
		Key:         "A|[1.0.0]",
		Item:        nil,
		Disposition: DispositionCycle,
		Depth:       2,
	}

	nodeB := &GraphNode{
		Key:         "B|1.0.0",
		Item:        &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes:  []*GraphNode{cycleNode},
		Disposition: DispositionAcceptable,
		Depth:       1,
	}
	cycleNode.OuterEdge = &GraphEdge{Item: nodeB.Item, OuterEdge: nil}

	nodeA := &GraphNode{
		Key:         "A|1.0.0",
		Item:        &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes:  []*GraphNode{nodeB},
		Disposition: DispositionAcceptable,
		Depth:       0,
	}
	nodeB.OuterEdge = &GraphEdge{Item: nodeA.Item, OuterEdge: nil}

	analyzer := NewCycleAnalyzer()
	reports := analyzer.AnalyzeCycles(nodeA)

	if len(reports) != 1 {
		t.Fatalf("Expected 1 cycle report, got %d", len(reports))
	}

	report := reports[0]
	if report.PackageID != "A" {
		t.Errorf("Expected cycle on A, got %s", report.PackageID)
	}

	if report.Depth != 2 {
		t.Errorf("Expected depth 2, got %d", report.Depth)
	}
}

func TestCycleAnalyzer_MultipleCycles(t *testing.T) {
	// Build graph with two separate cycles
	cycle1 := &GraphNode{
		Key:         "B|[1.0.0]",
		Disposition: DispositionCycle,
		Depth:       3,
	}

	cycle2 := &GraphNode{
		Key:         "D|[1.0.0]",
		Disposition: DispositionCycle,
		Depth:       3,
	}

	nodeB := &GraphNode{
		Key:        "B|1.0.0",
		Item:       &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes: []*GraphNode{cycle1},
		Depth:      2,
	}

	nodeD := &GraphNode{
		Key:        "D|1.0.0",
		Item:       &PackageDependencyInfo{ID: "D", Version: "1.0.0"},
		InnerNodes: []*GraphNode{cycle2},
		Depth:      2,
	}

	nodeC := &GraphNode{
		Key:        "C|1.0.0",
		Item:       &PackageDependencyInfo{ID: "C", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeB, nodeD},
		Depth:      1,
	}

	nodeA := &GraphNode{
		Key:        "A|1.0.0",
		Item:       &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes: []*GraphNode{nodeC},
		Depth:      0,
	}

	analyzer := NewCycleAnalyzer()
	reports := analyzer.AnalyzeCycles(nodeA)

	if len(reports) != 2 {
		t.Fatalf("Expected 2 cycle reports, got %d", len(reports))
	}
}

func TestCycleAnalyzer_NoCycles(t *testing.T) {
	// Build graph with no cycles: A -> B -> C
	nodeC := &GraphNode{
		Key:         "C|1.0.0",
		Item:        &PackageDependencyInfo{ID: "C", Version: "1.0.0"},
		InnerNodes:  []*GraphNode{},
		Disposition: DispositionAcceptable,
		Depth:       2,
	}

	nodeB := &GraphNode{
		Key:         "B|1.0.0",
		Item:        &PackageDependencyInfo{ID: "B", Version: "1.0.0"},
		InnerNodes:  []*GraphNode{nodeC},
		Disposition: DispositionAcceptable,
		Depth:       1,
	}

	nodeA := &GraphNode{
		Key:         "A|1.0.0",
		Item:        &PackageDependencyInfo{ID: "A", Version: "1.0.0"},
		InnerNodes:  []*GraphNode{nodeB},
		Disposition: DispositionAcceptable,
		Depth:       0,
	}

	analyzer := NewCycleAnalyzer()
	reports := analyzer.AnalyzeCycles(nodeA)

	if len(reports) != 0 {
		t.Errorf("Expected 0 cycle reports for graph without cycles, got %d", len(reports))
	}
}

func TestCycleAnalyzer_ExtractPackageID(t *testing.T) {
	analyzer := NewCycleAnalyzer()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "normal key with pipe",
			key:  "PackageA|[1.0.0]",
			want: "PackageA",
		},
		{
			name: "key without pipe",
			key:  "PackageB",
			want: "PackageB",
		},
		{
			name: "empty key",
			key:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.extractPackageID(tt.key)
			if got != tt.want {
				t.Errorf("extractPackageID(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestCycleAnalyzer_FormatDescription(t *testing.T) {
	analyzer := NewCycleAnalyzer()

	tests := []struct {
		name      string
		packageID string
		path      []string
		want      string
	}{
		{
			name:      "empty path",
			packageID: "PackageA",
			path:      []string{},
			want:      "Circular dependency on PackageA",
		},
		{
			name:      "single item path",
			packageID: "PackageB",
			path:      []string{"PackageA 1.0.0"},
			want:      "Circular dependency: PackageA 1.0.0 -> ... -> PackageB",
		},
		{
			name:      "multi-item path",
			packageID: "PackageC",
			path:      []string{"PackageA 1.0.0", "PackageB 2.0.0"},
			want:      "Circular dependency: PackageA 1.0.0 -> PackageB 2.0.0 -> ... -> PackageC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.formatCycleDescription(tt.packageID, tt.path)
			if got != tt.want {
				t.Errorf("formatCycleDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}
