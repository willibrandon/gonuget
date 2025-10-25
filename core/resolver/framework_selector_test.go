package resolver

import (
	"testing"
)

// TestFrameworkSelector_ExactMatch tests exact framework match
func TestFrameworkSelector_ExactMatch(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "net8.0",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
		{
			TargetFramework: "net6.0",
			Dependencies: []PackageDependency{
				{ID: "PackageB", VersionRange: "[2.0.0]"},
			},
		},
	}

	deps := selector.SelectDependencies(groups, "net8.0")

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].ID != "PackageA" {
		t.Errorf("Expected PackageA, got %s", deps[0].ID)
	}
}

// TestFrameworkSelector_NearestCompatible tests that net8.0 selects net6.0 over netstandard2.0
func TestFrameworkSelector_NearestCompatible(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "netstandard2.0",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
		{
			TargetFramework: "net6.0",
			Dependencies: []PackageDependency{
				{ID: "PackageB", VersionRange: "[2.0.0]"},
			},
		},
	}

	// net8.0 should select net6.0 as nearest compatible (over netstandard2.0)
	deps := selector.SelectDependencies(groups, "net8.0")

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].ID != "PackageB" {
		t.Errorf("Expected PackageB from net6.0 group, got %s", deps[0].ID)
	}
}

// TestFrameworkSelector_FallbackToUntargeted tests fallback to empty TargetFramework
func TestFrameworkSelector_FallbackToUntargeted(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "", // Untargeted group
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
		{
			TargetFramework: "net461",
			Dependencies: []PackageDependency{
				{ID: "PackageB", VersionRange: "[2.0.0]"},
			},
		},
	}

	// net8.0 is incompatible with net461, should fall back to untargeted group
	deps := selector.SelectDependencies(groups, "net8.0")

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].ID != "PackageA" {
		t.Errorf("Expected PackageA from untargeted group, got %s", deps[0].ID)
	}
}

// TestFrameworkSelector_EmptyGroups tests empty dependency groups
func TestFrameworkSelector_EmptyGroups(t *testing.T) {
	selector := NewFrameworkSelector()

	deps := selector.SelectDependencies([]DependencyGroup{}, "net8.0")

	if deps != nil {
		t.Errorf("Expected nil for empty groups, got %v", deps)
	}
}

// TestFrameworkSelector_NoCompatibleGroups tests when no groups are compatible
func TestFrameworkSelector_NoCompatibleGroups(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "net461",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
		{
			TargetFramework: "net462",
			Dependencies: []PackageDependency{
				{ID: "PackageB", VersionRange: "[2.0.0]"},
			},
		},
	}

	// net8.0 is incompatible with net461/net462 (no fallback group)
	deps := selector.SelectDependencies(groups, "net8.0")

	if deps != nil {
		t.Errorf("Expected nil when no compatible groups, got %v", deps)
	}
}

// TestFrameworkSelector_SingleGroup tests single dependency group
func TestFrameworkSelector_SingleGroup(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "net6.0",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
	}

	deps := selector.SelectDependencies(groups, "net8.0")

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].ID != "PackageA" {
		t.Errorf("Expected PackageA, got %s", deps[0].ID)
	}
}

// TestFrameworkSelector_InvalidTargetFramework tests invalid target framework
func TestFrameworkSelector_InvalidTargetFramework(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "net8.0",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
	}

	deps := selector.SelectDependencies(groups, "invalid-framework")

	if deps != nil {
		t.Errorf("Expected nil for invalid target framework, got %v", deps)
	}
}

// TestFrameworkSelector_InvalidGroupFramework tests invalid group framework
func TestFrameworkSelector_InvalidGroupFramework(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "invalid-framework",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
		{
			TargetFramework: "net8.0",
			Dependencies: []PackageDependency{
				{ID: "PackageB", VersionRange: "[2.0.0]"},
			},
		},
	}

	deps := selector.SelectDependencies(groups, "net8.0")

	// Should skip invalid group and select net8.0
	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].ID != "PackageB" {
		t.Errorf("Expected PackageB, got %s", deps[0].ID)
	}
}

// TestFrameworkSelector_UntargetedGroupPriority tests untargeted group priority
func TestFrameworkSelector_UntargetedGroupPriority(t *testing.T) {
	selector := NewFrameworkSelector()

	groups := []DependencyGroup{
		{
			TargetFramework: "",
			Dependencies: []PackageDependency{
				{ID: "PackageA", VersionRange: "[1.0.0]"},
			},
		},
		{
			TargetFramework: "net8.0",
			Dependencies: []PackageDependency{
				{ID: "PackageB", VersionRange: "[2.0.0]"},
			},
		},
	}

	deps := selector.SelectDependencies(groups, "net8.0")

	// Should prefer exact match over untargeted
	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].ID != "PackageB" {
		t.Errorf("Expected PackageB from net8.0 group, got %s", deps[0].ID)
	}
}
