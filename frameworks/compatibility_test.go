package frameworks

import "testing"

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		name       string
		package_   string
		target     string
		compatible bool
	}{
		// .NET Standard → .NET 6+
		{"netstandard2.0 → net6.0", "netstandard2.0", "net6.0", true},
		{"netstandard2.1 → net6.0", "netstandard2.1", "net6.0", true},

		// .NET Standard → .NET Framework
		{"netstandard1.0 → net45", "netstandard1.0", "net45", true},
		{"netstandard2.0 → net461", "netstandard2.0", "net461", true},
		{"netstandard2.1 → net48", "netstandard2.1", "net48", false}, // 2.1 not compatible

		// .NET Standard → .NET Core
		{"netstandard2.0 → netcoreapp2.0", "netstandard2.0", "netcoreapp2.0", true},
		{"netstandard2.1 → netcoreapp3.0", "netstandard2.1", "netcoreapp3.0", true},

		// Same framework
		{"net6.0 → net6.0", "net6.0", "net6.0", true},
		{"net48 → net48", "net48", "net48", true},

		// Higher to lower (not compatible)
		{"net6.0 → net5.0", "net6.0", "net5.0", false},
		{"net48 → net45", "net48", "net45", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := MustParseFramework(tt.package_)
			target := MustParseFramework(tt.target)

			got := pkg.IsCompatible(target)
			if got != tt.compatible {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.compatible)
			}
		})
	}
}

func TestGetNearest(t *testing.T) {
	available := []*NuGetFramework{
		MustParseFramework("net45"),
		MustParseFramework("netstandard2.0"),
		MustParseFramework("net6.0"),
		MustParseFramework("netcoreapp3.1"),
	}

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{"net8.0 picks net6.0", "net8.0", "net6.0"},
		{"net48 picks netstandard2.0", "net48", "netstandard2.0"},
		{"netcoreapp3.1 exact", "netcoreapp3.1", "netcoreapp3.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := MustParseFramework(tt.target)
			got := GetNearest(target, available)

			if got == nil {
				t.Fatalf("GetNearest() = nil, want %s", tt.expected)
			}

			if got.String() != tt.expected {
				t.Errorf("GetNearest() = %s, want %s", got, tt.expected)
			}
		})
	}
}

// Benchmark compatibility checking
func BenchmarkIsCompatible(b *testing.B) {
	pkg := MustParseFramework("netstandard2.0")
	target := MustParseFramework("net6.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pkg.IsCompatible(target)
	}
}
