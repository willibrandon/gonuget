package resolver

import (
	"context"
	"testing"
)

// BenchmarkDependencyWalker_SimpleGraph benchmarks a simple A -> B -> C graph
func BenchmarkDependencyWalker_SimpleGraph(b *testing.B) {
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
	ctx := context.Background()

	for b.Loop() {
		_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")
	}
}

// BenchmarkDependencyWalker_WideGraph benchmarks a graph with many dependencies at one level
// A -> B1, B2, B3, B4, B5, B6, B7, B8, B9, B10
func BenchmarkDependencyWalker_WideGraph(b *testing.B) {
	packages := map[string]*PackageDependencyInfo{
		"A|1.0.0": {
			ID:      "A",
			Version: "1.0.0",
			Dependencies: []PackageDependency{
				{ID: "B1", VersionRange: "[1.0.0]"},
				{ID: "B2", VersionRange: "[1.0.0]"},
				{ID: "B3", VersionRange: "[1.0.0]"},
				{ID: "B4", VersionRange: "[1.0.0]"},
				{ID: "B5", VersionRange: "[1.0.0]"},
				{ID: "B6", VersionRange: "[1.0.0]"},
				{ID: "B7", VersionRange: "[1.0.0]"},
				{ID: "B8", VersionRange: "[1.0.0]"},
				{ID: "B9", VersionRange: "[1.0.0]"},
				{ID: "B10", VersionRange: "[1.0.0]"},
			},
		},
	}

	// Add all B packages
	for i := 1; i <= 10; i++ {
		id := "B" + string(rune('0'+i))
		packages[id+"|1.0.0"] = &PackageDependencyInfo{
			ID:           id,
			Version:      "1.0.0",
			Dependencies: []PackageDependency{},
		}
	}

	client := &mockPackageMetadataClient{packages: packages}
	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")
	ctx := context.Background()

	for b.Loop() {
		_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")
	}
}

// BenchmarkDependencyWalker_DeepGraph benchmarks a deep linear dependency chain
// A -> B -> C -> D -> E -> F -> G -> H -> I -> J
func BenchmarkDependencyWalker_DeepGraph(b *testing.B) {
	packages := map[string]*PackageDependencyInfo{
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
			Dependencies: []PackageDependency{{ID: "E", VersionRange: "[1.0.0]"}},
		},
		"E|1.0.0": {
			ID:           "E",
			Version:      "1.0.0",
			Dependencies: []PackageDependency{{ID: "F", VersionRange: "[1.0.0]"}},
		},
		"F|1.0.0": {
			ID:           "F",
			Version:      "1.0.0",
			Dependencies: []PackageDependency{{ID: "G", VersionRange: "[1.0.0]"}},
		},
		"G|1.0.0": {
			ID:           "G",
			Version:      "1.0.0",
			Dependencies: []PackageDependency{{ID: "H", VersionRange: "[1.0.0]"}},
		},
		"H|1.0.0": {
			ID:           "H",
			Version:      "1.0.0",
			Dependencies: []PackageDependency{{ID: "I", VersionRange: "[1.0.0]"}},
		},
		"I|1.0.0": {
			ID:           "I",
			Version:      "1.0.0",
			Dependencies: []PackageDependency{{ID: "J", VersionRange: "[1.0.0]"}},
		},
		"J|1.0.0": {
			ID:           "J",
			Version:      "1.0.0",
			Dependencies: []PackageDependency{},
		},
	}

	client := &mockPackageMetadataClient{packages: packages}
	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")
	ctx := context.Background()

	for b.Loop() {
		_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")
	}
}

// BenchmarkDependencyWalker_ComplexGraph benchmarks a complex diamond-shaped graph
// This tests the stack-based approach with shared dependencies
//
//	  A
//	 / \
//	B   C
//	 \ / \
//	  D   E
//	   \ /
//	    F
func BenchmarkDependencyWalker_ComplexGraph(b *testing.B) {
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
					{ID: "D", VersionRange: "[1.0.0]"},
					{ID: "E", VersionRange: "[1.0.0]"},
				},
			},
			"D|1.0.0": {
				ID:      "D",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "F", VersionRange: "[1.0.0]"},
				},
			},
			"E|1.0.0": {
				ID:      "E",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "F", VersionRange: "[1.0.0]"},
				},
			},
			"F|1.0.0": {
				ID:           "F",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")
	ctx := context.Background()

	for b.Loop() {
		_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")
	}
}

// BenchmarkDependencyWalker_CycleDetection benchmarks cycle detection performance
func BenchmarkDependencyWalker_CycleDetection(b *testing.B) {
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
					{ID: "C", VersionRange: "[1.0.0]"},
				},
			},
			"C|1.0.0": {
				ID:      "C",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "A", VersionRange: "[1.0.0]"}, // Cycle back to A
				},
			},
		},
	}

	walker := NewDependencyWalker(client, []string{"source1"}, "net8.0")
	ctx := context.Background()

	for b.Loop() {
		_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")
	}
}

// BenchmarkDependencyWalker_CacheEffectiveness benchmarks cache hit performance
func BenchmarkDependencyWalker_CacheEffectiveness(b *testing.B) {
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
	ctx := context.Background()

	// Warm up cache
	_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")

	for b.Loop() {
		// All subsequent walks should hit cache
		_, _ = walker.Walk(ctx, "A", "[1.0.0]", "net8.0")
	}
}
