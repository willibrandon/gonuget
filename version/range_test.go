package version

import "testing"

func TestParseVersionRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"inclusive both", "[1.0, 2.0]", false},
		{"exclusive both", "(1.0, 2.0)", false},
		{"mixed", "[1.0, 2.0)", false},
		{"open upper", "[1.0, )", false},
		{"open lower", "(, 2.0]", false},
		{"simple version", "1.0.0", false},
		{"empty", "", true},
		{"missing bracket", "[1.0, 2.0", true},
		{"wrong brackets", "]1.0, 2.0[", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVersionRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersionRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVersionRange_FindBestMatch_FavorLower(t *testing.T) {
	// Test case from real Serilog dependency resolution
	// Serilog.Sinks.File 5.0.0 depends on Serilog [2.10.0, )
	// When both 2.10.0 and 3.0.0 are cached, NuGet picks 2.10.0 (lower)
	versions := []*NuGetVersion{
		MustParse("2.10.0"),
		MustParse("3.0.0"),
		MustParse("4.0.0-beta"),
	}

	r, err := ParseVersionRange("[2.10.0, )")
	if err != nil {
		t.Fatalf("ParseVersionRange() error = %v", err)
	}

	best := r.FindBestMatch(versions)

	if best == nil {
		t.Fatal("FindBestMatch() = nil, want 2.10.0")
	}

	if best.String() != "2.10.0" {
		t.Errorf("FindBestMatch() = %v, want 2.10.0 (favors lower version)", best)
	}
}

func TestVersionRange_Satisfies(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		version  string
		expected bool
	}{
		// Inclusive ranges
		{"inclusive min", "[1.0, 2.0]", "1.0.0", true},
		{"inclusive max", "[1.0, 2.0]", "2.0.0", true},
		{"inclusive middle", "[1.0, 2.0]", "1.5.0", true},
		{"inclusive below", "[1.0, 2.0]", "0.9.0", false},
		{"inclusive above", "[1.0, 2.0]", "2.1.0", false},

		// Exclusive ranges
		{"exclusive min", "(1.0, 2.0)", "1.0.0", false},
		{"exclusive max", "(1.0, 2.0)", "2.0.0", false},
		{"exclusive middle", "(1.0, 2.0)", "1.5.0", true},

		// Mixed
		{"mixed min inclusive", "[1.0, 2.0)", "1.0.0", true},
		{"mixed max exclusive", "[1.0, 2.0)", "2.0.0", false},

		// Open-ended
		{"open upper", "[1.0, )", "100.0.0", true},
		{"open lower", "(, 2.0]", "0.1.0", true},

		// Simple version (>= semantics)
		{"simple satisfies", "1.0.0", "1.5.0", true},
		{"simple not satisfies", "1.0.0", "0.9.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("ParseVersionRange() error = %v", err)
			}

			v := MustParse(tt.version)
			got := r.Satisfies(v)

			if got != tt.expected {
				t.Errorf("Satisfies(%s) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestVersionRange_FindBestMatch(t *testing.T) {
	versions := []*NuGetVersion{
		MustParse("1.0.0"),
		MustParse("1.5.0"),
		MustParse("2.0.0"),
		MustParse("2.5.0"),
		MustParse("3.0.0"),
	}

	tests := []struct {
		name     string
		rangeStr string
		expected string
	}{
		{"range 1.0-2.0", "[1.0, 2.0]", "1.0.0"},         // Favor lower: minimum satisfying version
		{"range 1.0-2.0 exclusive", "[1.0, 2.0)", "1.0.0"}, // Favor lower: minimum satisfying version
		{"open upper from 2.0", "[2.0, )", "2.0.0"},      // Favor lower: minimum satisfying version
		{"open lower to 2.0", "(, 2.0]", "1.0.0"},        // Favor lower: minimum satisfying version (1.0.0 is lowest)
		{"no match", "[10.0, 20.0]", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("ParseVersionRange() error = %v", err)
			}

			got := r.FindBestMatch(versions)

			if tt.expected == "" {
				if got != nil {
					t.Errorf("FindBestMatch() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("FindBestMatch() = nil, want %s", tt.expected)
				} else if got.String() != tt.expected {
					t.Errorf("FindBestMatch() = %v, want %s", got, tt.expected)
				}
			}
		})
	}
}
