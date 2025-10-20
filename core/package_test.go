package core

import (
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

func TestPackageIdentity_Equals(t *testing.T) {
	tests := []struct {
		name   string
		p1     PackageIdentity
		p2     PackageIdentity
		equals bool
	}{
		{
			name:   "same ID and version",
			p1:     NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			p2:     NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			equals: true,
		},
		{
			name:   "case-insensitive ID",
			p1:     NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			p2:     NewPackageIdentity("newtonsoft.json", version.MustParse("13.0.1")),
			equals: true,
		},
		{
			name:   "different version",
			p1:     NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			p2:     NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.2")),
			equals: false,
		},
		{
			name:   "different ID",
			p1:     NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			p2:     NewPackageIdentity("Other.Package", version.MustParse("13.0.1")),
			equals: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p1.Equals(tt.p2)
			if got != tt.equals {
				t.Errorf("Equals() = %v, want %v", got, tt.equals)
			}
		})
	}
}

func TestPackageIdentity_String(t *testing.T) {
	p := NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1"))
	got := p.String()
	want := "Newtonsoft.Json 13.0.1"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPackageMetadata_GetDependenciesForFramework(t *testing.T) {
	// Setup test package metadata with multiple dependency groups
	meta := &PackageMetadata{
		Identity: NewPackageIdentity("TestPackage", version.MustParse("1.0.0")),
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: frameworks.MustParseFramework("net45"),
				Dependencies: []PackageDependency{
					{
						ID:           "Newtonsoft.Json",
						VersionRange: version.MustParseRange("[12.0.1, )"),
					},
				},
			},
			{
				TargetFramework: frameworks.MustParseFramework("netstandard2.0"),
				Dependencies: []PackageDependency{
					{
						ID:           "System.Text.Json",
						VersionRange: version.MustParseRange("[6.0.0, )"),
					},
				},
			},
			{
				TargetFramework: frameworks.MustParseFramework("net6.0"),
				Dependencies: []PackageDependency{
					{
						ID:           "System.Text.Json",
						VersionRange: version.MustParseRange("[7.0.0, )"),
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		target         string
		expectDepCount int
		expectDepID    string
	}{
		{
			name:           "exact match net6.0",
			target:         "net6.0",
			expectDepCount: 1,
			expectDepID:    "System.Text.Json",
		},
		{
			name:           "net8.0 picks net6.0 (nearest compatible)",
			target:         "net8.0",
			expectDepCount: 1,
			expectDepID:    "System.Text.Json",
		},
		{
			name:           "net48 picks netstandard2.0 (cross-framework compatible)",
			target:         "net48",
			expectDepCount: 1,
			expectDepID:    "System.Text.Json",
		},
		{
			name:           "exact match net45",
			target:         "net45",
			expectDepCount: 1,
			expectDepID:    "Newtonsoft.Json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := frameworks.MustParseFramework(tt.target)
			deps := meta.GetDependenciesForFramework(target)

			if len(deps) != tt.expectDepCount {
				t.Errorf("GetDependenciesForFramework() returned %d dependencies, want %d", len(deps), tt.expectDepCount)
				return
			}

			if tt.expectDepCount > 0 && deps[0].ID != tt.expectDepID {
				t.Errorf("GetDependenciesForFramework() returned dependency %q, want %q", deps[0].ID, tt.expectDepID)
			}
		})
	}
}

func TestPackageMetadata_GetDependenciesForFramework_NoCompatible(t *testing.T) {
	meta := &PackageMetadata{
		Identity: NewPackageIdentity("TestPackage", version.MustParse("1.0.0")),
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: frameworks.MustParseFramework("net6.0"),
				Dependencies: []PackageDependency{
					{
						ID:           "System.Text.Json",
						VersionRange: version.MustParseRange("[7.0.0, )"),
					},
				},
			},
		},
	}

	// net45 is not compatible with net6.0
	target := frameworks.MustParseFramework("net45")
	deps := meta.GetDependenciesForFramework(target)

	if deps != nil {
		t.Errorf("GetDependenciesForFramework() = %v, want nil (no compatible framework)", deps)
	}
}

func TestPackageMetadata_GetDependenciesForFramework_NilTarget(t *testing.T) {
	meta := &PackageMetadata{
		Identity: NewPackageIdentity("TestPackage", version.MustParse("1.0.0")),
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: frameworks.MustParseFramework("net6.0"),
				Dependencies: []PackageDependency{
					{
						ID:           "System.Text.Json",
						VersionRange: version.MustParseRange("[7.0.0, )"),
					},
				},
			},
		},
	}

	deps := meta.GetDependenciesForFramework(nil)
	if deps != nil {
		t.Errorf("GetDependenciesForFramework(nil) = %v, want nil", deps)
	}
}

func TestPackageMetadata_GetDependenciesForFramework_EmptyGroups(t *testing.T) {
	meta := &PackageMetadata{
		Identity:         NewPackageIdentity("TestPackage", version.MustParse("1.0.0")),
		DependencyGroups: []PackageDependencyGroup{},
	}

	target := frameworks.MustParseFramework("net6.0")
	deps := meta.GetDependenciesForFramework(target)

	if deps != nil {
		t.Errorf("GetDependenciesForFramework() with empty groups = %v, want nil", deps)
	}
}

// Benchmark package identity operations
func BenchmarkPackageIdentity_Equals(b *testing.B) {
	p1 := NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1"))
	p2 := NewPackageIdentity("newtonsoft.json", version.MustParse("13.0.1"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p1.Equals(p2)
	}
}

func BenchmarkGetDependenciesForFramework(b *testing.B) {
	meta := &PackageMetadata{
		Identity: NewPackageIdentity("TestPackage", version.MustParse("1.0.0")),
		DependencyGroups: []PackageDependencyGroup{
			{
				TargetFramework: frameworks.MustParseFramework("net45"),
				Dependencies: []PackageDependency{
					{ID: "Dep1", VersionRange: version.MustParseRange("[1.0.0, )")},
				},
			},
			{
				TargetFramework: frameworks.MustParseFramework("netstandard2.0"),
				Dependencies: []PackageDependency{
					{ID: "Dep2", VersionRange: version.MustParseRange("[2.0.0, )")},
				},
			},
			{
				TargetFramework: frameworks.MustParseFramework("net6.0"),
				Dependencies: []PackageDependency{
					{ID: "Dep3", VersionRange: version.MustParseRange("[3.0.0, )")},
				},
			},
		},
	}

	target := frameworks.MustParseFramework("net8.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = meta.GetDependenciesForFramework(target)
	}
}
