package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int // -1, 0, 1
	}{
		// Basic comparisons
		{"equal", "1.0.0", "1.0.0", 0},
		{"major less", "1.0.0", "2.0.0", -1},
		{"major greater", "2.0.0", "1.0.0", 1},
		{"minor less", "1.0.0", "1.1.0", -1},
		{"minor greater", "1.1.0", "1.0.0", 1},
		{"patch less", "1.0.0", "1.0.1", -1},
		{"patch greater", "1.0.1", "1.0.0", 1},

		// Prerelease comparisons
		{"release > prerelease", "1.0.0", "1.0.0-beta", 1},
		{"prerelease < release", "1.0.0-beta", "1.0.0", -1},
		{"prerelease alpha < beta", "1.0.0-alpha", "1.0.0-beta", -1},
		{"prerelease beta > alpha", "1.0.0-beta", "1.0.0-alpha", 1},

		// Numeric vs alphanumeric labels
		{"numeric < alphanumeric", "1.0.0-1", "1.0.0-alpha", -1},
		{"alphanumeric > numeric", "1.0.0-alpha", "1.0.0-1", 1},

		// Multiple labels
		{"shorter label list", "1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"longer label list", "1.0.0-alpha.1", "1.0.0-alpha", 1},
		{"equal multiple labels", "1.0.0-alpha.1", "1.0.0-alpha.1", 0},

		// Metadata ignored
		{"metadata ignored 1", "1.0.0+a", "1.0.0+b", 0},
		{"metadata ignored 2", "1.0.0+build", "1.0.0", 0},

		// Legacy versions
		{"legacy equal", "1.0.0.0", "1.0.0.0", 0},
		{"legacy revision", "1.0.0.0", "1.0.0.1", -1},
		{"legacy vs semver", "1.0.0.1", "1.0.0", 0}, // revision ignored when comparing to semver
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1 := MustParse(tt.v1)
			v2 := MustParse(tt.v2)

			got := v1.Compare(v2)
			if got != tt.expected {
				t.Errorf("Compare(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.expected)
			}
		})
	}
}

func TestEquals(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "2.0.0", false},
		{"1.0.0+a", "1.0.0+b", true}, // metadata ignored
	}

	for _, tt := range tests {
		v1 := MustParse(tt.v1)
		v2 := MustParse(tt.v2)

		got := v1.Equals(v2)
		if got != tt.expected {
			t.Errorf("Equals(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.0.0", "2.0.0", true},
		{"2.0.0", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		v1 := MustParse(tt.v1)
		v2 := MustParse(tt.v2)

		got := v1.LessThan(v2)
		if got != tt.expected {
			t.Errorf("LessThan(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"2.0.0", "1.0.0", true},
		{"1.0.0", "2.0.0", false},
		{"1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		v1 := MustParse(tt.v1)
		v2 := MustParse(tt.v2)

		got := v1.GreaterThan(v2)
		if got != tt.expected {
			t.Errorf("GreaterThan(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

// Benchmark version comparison
func BenchmarkCompare(b *testing.B) {
	v1 := MustParse("1.2.3-beta.1")
	v2 := MustParse("1.2.3-beta.2")

	b.ResetTimer()
	for b.Loop() {
		_ = v1.Compare(v2)
	}
}

// Benchmark simple version comparison (no prerelease labels)
func BenchmarkCompare_Simple(b *testing.B) {
	v1 := MustParse("1.2.3")
	v2 := MustParse("1.2.4")

	b.ResetTimer()
	for b.Loop() {
		_ = v1.Compare(v2)
	}
}
