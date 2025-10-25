package resolver

import (
	"context"
	"testing"
)

func TestTransitiveResolver_MultipleRoots(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "C", VersionRange: "[1.0.0]"},
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

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	transitiveResolver := NewTransitiveResolver(resolver)

	roots := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
		{ID: "B", VersionRange: "[1.0.0]"},
	}

	result, err := transitiveResolver.ResolveMultipleRoots(context.Background(), roots)

	if err != nil {
		t.Fatalf("ResolveMultipleRoots() failed: %v", err)
	}

	// Should have A, B, C (C deduplicated)
	if len(result.Packages) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(result.Packages))
	}

	// Verify C appears only once
	cCount := 0
	for _, pkg := range result.Packages {
		if pkg.ID == "C" {
			cCount++
		}
	}

	if cCount != 1 {
		t.Errorf("Expected C once, got %d times", cCount)
	}
}

func TestTransitiveResolver_WithConflicts(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "C", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "C", VersionRange: "[2.0.0]"},
				},
			},
			"C|1.0.0": {ID: "C", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"C|2.0.0": {ID: "C", Version: "2.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	transitiveResolver := NewTransitiveResolver(resolver)

	roots := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
		{ID: "B", VersionRange: "[1.0.0]"},
	}

	result, err := transitiveResolver.ResolveMultipleRoots(context.Background(), roots)

	if err != nil {
		t.Fatalf("ResolveMultipleRoots() failed: %v", err)
	}

	// Should detect conflict on C
	if len(result.Conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(result.Conflicts))
	}

	// Find resolved C version
	var cVersion string
	for _, pkg := range result.Packages {
		if pkg.ID == "C" {
			cVersion = pkg.Version
			break
		}
	}

	// Should resolve to higher version
	if cVersion != "2.0.0" {
		t.Errorf("Expected C 2.0.0, got %s", cVersion)
	}
}

func TestTransitiveResolver_SinglePackage(t *testing.T) {
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
	transitiveResolver := NewTransitiveResolver(resolver)

	result, err := transitiveResolver.ResolveTransitive(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("ResolveTransitive() failed: %v", err)
	}

	// Should have A and B
	if len(result.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(result.Packages))
	}

	// Verify no conflicts
	if len(result.Conflicts) != 0 {
		t.Errorf("Expected 0 conflicts, got %d", len(result.Conflicts))
	}
}

func TestTransitiveResolver_ExcludesSyntheticRoot(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:           "A",
				Version:      "1.0.0",
				Dependencies: []PackageDependency{},
			},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	transitiveResolver := NewTransitiveResolver(resolver)

	roots := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
	}

	result, err := transitiveResolver.ResolveMultipleRoots(context.Background(), roots)

	if err != nil {
		t.Fatalf("ResolveMultipleRoots() failed: %v", err)
	}

	// Verify synthetic root is not in packages
	for _, pkg := range result.Packages {
		if pkg.ID == "__project__" {
			t.Error("Synthetic root should not appear in result packages")
		}
	}

	// Should have only A
	if len(result.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(result.Packages))
	}
}
