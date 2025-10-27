package resolver

import (
	"context"
	"testing"
)

// TestResolver_UnresolvedPackageCollection verifies that the resolver collects
// unresolved packages from the graph and populates ResolutionResult.Unresolved.
// Matches NuGet.Client's RestoreTargetGraph.Unresolved collection behavior.
func TestResolver_UnresolvedPackageCollection(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "Missing1", VersionRange: "[1.0.0]"},
					{ID: "B", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {
				ID:      "B",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "Missing2", VersionRange: "[2.0.0]"},
				},
			},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	// Should have 2 resolved packages (A, B)
	if len(result.Packages) != 2 {
		t.Errorf("Expected 2 resolved packages, got %d", len(result.Packages))
	}

	// Should have 2 unresolved packages (Missing1, Missing2)
	if len(result.Unresolved) != 2 {
		t.Fatalf("Expected 2 unresolved packages, got %d", len(result.Unresolved))
	}

	// Verify unresolved packages
	unresolvedIDs := make(map[string]bool)
	for _, unresolved := range result.Unresolved {
		unresolvedIDs[unresolved.ID] = true

		// Verify error code is set
		if unresolved.ErrorCode == "" {
			t.Errorf("Expected ErrorCode to be set for %s", unresolved.ID)
		}

		// Verify message is set
		if unresolved.Message == "" {
			t.Errorf("Expected Message to be set for %s", unresolved.ID)
		}

		// Verify target framework is set
		if unresolved.TargetFramework != "net8.0" {
			t.Errorf("Expected TargetFramework net8.0, got %s", unresolved.TargetFramework)
		}
	}

	// Verify both Missing1 and Missing2 are reported
	if !unresolvedIDs["Missing1"] {
		t.Error("Expected Missing1 in unresolved collection")
	}
	if !unresolvedIDs["Missing2"] {
		t.Error("Expected Missing2 in unresolved collection")
	}
}

// TestResolver_Success verifies that ResolutionResult.Success() returns true
// when there are no unresolved packages, and false when there are.
// Matches NuGet.Client: graphs.All(g => g.Unresolved.Count == 0)
func TestResolver_Success(t *testing.T) {
	t.Run("Success with all packages resolved", func(t *testing.T) {
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

		// Should succeed - no unresolved packages
		if !result.Success() {
			t.Error("Expected Success() = true when all packages resolved")
		}

		if len(result.Unresolved) != 0 {
			t.Errorf("Expected 0 unresolved packages, got %d", len(result.Unresolved))
		}
	})

	t.Run("Failure with unresolved packages", func(t *testing.T) {
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

		resolver := NewResolver(client, []string{"source1"}, "net8.0")
		result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

		if err != nil {
			t.Fatalf("Resolve() failed: %v", err)
		}

		// Should fail - has unresolved package
		if result.Success() {
			t.Error("Expected Success() = false when unresolved packages exist")
		}

		if len(result.Unresolved) == 0 {
			t.Error("Expected unresolved packages to be collected")
		}
	})
}

// TestResolver_UnresolvedNotInPackagesList verifies that unresolved packages
// appear only in the Unresolved collection, NOT in the Packages list.
// Matches NuGet.Client behavior: unresolved != resolved.
func TestResolver_UnresolvedNotInPackagesList(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "Missing", VersionRange: "[1.0.0]"},
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

	// Should have 2 resolved packages (A, B) - NOT Missing
	if len(result.Packages) != 2 {
		t.Errorf("Expected 2 resolved packages, got %d", len(result.Packages))
	}

	// Verify Missing is NOT in Packages list
	for _, pkg := range result.Packages {
		if pkg.ID == "Missing" {
			t.Error("Unresolved package 'Missing' should not appear in Packages list")
		}
		if pkg.IsUnresolved {
			t.Errorf("Package %s in Packages list should not be marked as unresolved", pkg.ID)
		}
	}

	// Verify Missing IS in Unresolved list
	foundMissing := false
	for _, unresolved := range result.Unresolved {
		if unresolved.ID == "Missing" {
			foundMissing = true
			break
		}
	}

	if !foundMissing {
		t.Error("Expected 'Missing' to appear in Unresolved collection")
	}
}

// TestResolver_UnresolvedRootPackage verifies that when the root package
// cannot be resolved, it appears in the Unresolved collection.
func TestResolver_UnresolvedRootPackage(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	result, err := resolver.Resolve(context.Background(), "NonExistent", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() should not error, got: %v", err)
	}

	// Should have 0 resolved packages
	if len(result.Packages) != 0 {
		t.Errorf("Expected 0 resolved packages, got %d", len(result.Packages))
	}

	// Should have 1 unresolved package (the root)
	if len(result.Unresolved) != 1 {
		t.Fatalf("Expected 1 unresolved package, got %d", len(result.Unresolved))
	}

	unresolved := result.Unresolved[0]
	if unresolved.ID != "NonExistent" {
		t.Errorf("Expected unresolved ID 'NonExistent', got %s", unresolved.ID)
	}

	if unresolved.VersionRange != "[1.0.0]" {
		t.Errorf("Expected version range '[1.0.0]', got %s", unresolved.VersionRange)
	}

	// Should fail success check
	if result.Success() {
		t.Error("Expected Success() = false when root package is unresolved")
	}
}

// TestResolver_UnresolvedPreservesVersionRange verifies that the requested
// version range is preserved in the UnresolvedPackage, matching NuGet.Client's
// LibraryRange.VersionRange field.
func TestResolver_UnresolvedPreservesVersionRange(t *testing.T) {
	testCases := []struct {
		name         string
		versionRange string
	}{
		{"Exact version", "[1.0.0]"},
		{"Minimum inclusive", "1.0.0"},
		{"Range exclusive max", "[1.0.0,2.0.0)"},
		{"Range inclusive max", "[1.0.0,2.0.0]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockPackageMetadataClient{
				packages: map[string]*PackageDependencyInfo{
					"A|1.0.0": {
						ID:      "A",
						Version: "1.0.0",
						Dependencies: []PackageDependency{
							{ID: "Missing", VersionRange: tc.versionRange},
						},
					},
				},
			}

			resolver := NewResolver(client, []string{"source1"}, "net8.0")
			result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

			if err != nil {
				t.Fatalf("Resolve() failed: %v", err)
			}

			if len(result.Unresolved) != 1 {
				t.Fatalf("Expected 1 unresolved package, got %d", len(result.Unresolved))
			}

			if result.Unresolved[0].VersionRange != tc.versionRange {
				t.Errorf("Expected version range %q, got %q",
					tc.versionRange, result.Unresolved[0].VersionRange)
			}
		})
	}
}

// TestResolver_EnhancedDiagnostics_NU1101 verifies that NU1101 error code is used
// when a package doesn't exist at all (no versions found).
func TestResolver_EnhancedDiagnostics_NU1101(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "DoesNotExist", VersionRange: "[1.0.0]"},
				},
			},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	// Should have 1 unresolved package
	if len(result.Unresolved) != 1 {
		t.Fatalf("Expected 1 unresolved package, got %d", len(result.Unresolved))
	}

	unresolved := result.Unresolved[0]

	// Should use NU1101 (package doesn't exist)
	if unresolved.ErrorCode != string(NU1101) {
		t.Errorf("Expected error code NU1101, got %s", unresolved.ErrorCode)
	}

	// Should have sources populated
	if len(unresolved.Sources) == 0 {
		t.Error("Expected sources to be populated")
	}

	// Should have message with package name
	if !contains(unresolved.Message, "DoesNotExist") {
		t.Errorf("Expected message to contain package name, got: %s", unresolved.Message)
	}

	// Should have no available versions
	if len(unresolved.AvailableVersions) != 0 {
		t.Errorf("Expected 0 available versions for NU1101, got %d", len(unresolved.AvailableVersions))
	}
}

// TestResolver_EnhancedDiagnostics_NU1102 verifies that NU1102 error code is used
// when a package exists but no version matches the requested range.
func TestResolver_EnhancedDiagnostics_NU1102(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[5.0.0]"}, // Requires 5.0.0 exactly
				},
			},
			// B exists but only has versions 1.0.0, 2.0.0, 3.0.0 (no 5.0.0)
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"B|2.0.0": {ID: "B", Version: "2.0.0", Dependencies: []PackageDependency{}},
			"B|3.0.0": {ID: "B", Version: "3.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	result, err := resolver.Resolve(context.Background(), "A", "[1.0.0]")

	if err != nil {
		t.Fatalf("Resolve() failed: %v", err)
	}

	// Should have 1 unresolved package (B)
	if len(result.Unresolved) != 1 {
		t.Fatalf("Expected 1 unresolved package, got %d", len(result.Unresolved))
	}

	unresolved := result.Unresolved[0]

	// Should use NU1102 (version doesn't match)
	if unresolved.ErrorCode != string(NU1102) {
		t.Errorf("Expected error code NU1102, got %s", unresolved.ErrorCode)
	}

	// Should have available versions
	if len(unresolved.AvailableVersions) != 3 {
		t.Errorf("Expected 3 available versions, got %d", len(unresolved.AvailableVersions))
	}

	// Should have nearest version populated
	if unresolved.NearestVersion == "" {
		t.Error("Expected nearest version to be populated for NU1102")
	}

	// Should have sources populated
	if len(unresolved.Sources) == 0 {
		t.Error("Expected sources to be populated")
	}

	// Message should mention the version range
	if !contains(unresolved.Message, "[5.0.0]") {
		t.Errorf("Expected message to contain version range, got: %s", unresolved.Message)
	}

	// Message should mention available versions
	if !contains(unresolved.Message, "Found") || !contains(unresolved.Message, "version(s)") {
		t.Errorf("Expected message to mention available versions, got: %s", unresolved.Message)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
