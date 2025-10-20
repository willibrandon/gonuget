package version

import "testing"

func TestParseFloatRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		behavior FloatBehavior
		wantErr  bool
	}{
		{"wildcard only", "*", FloatMajor, false},
		{"major float", "1.*", FloatMinor, false},
		{"minor float", "1.0.*", FloatPatch, false},
		{"patch float", "1.0.0.*", FloatRevision, false},
		{"prerelease float", "1.0.0-*", FloatPrerelease, false},
		{"no wildcard", "1.0.0", FloatNone, true},
		{"empty", "", FloatNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloatRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloatRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.FloatBehavior != tt.behavior {
				t.Errorf("ParseFloatRange() behavior = %v, want %v", got.FloatBehavior, tt.behavior)
			}
		})
	}
}

func TestFloatRange_Satisfies(t *testing.T) {
	tests := []struct {
		name     string
		floatStr string
		version  string
		expected bool
	}{
		// Major float
		{"major float any", "*", "5.0.0", true},
		{"major float prerelease", "*", "1.0.0-beta", true},

		// Minor float
		{"minor float match", "1.*", "1.5.0", true},
		{"minor float no match", "1.*", "2.0.0", false},
		{"minor float exact", "1.*", "1.0.0", true},

		// Patch float
		{"patch float match", "1.0.*", "1.0.5", true},
		{"patch float no match", "1.0.*", "1.1.0", false},
		{"patch float exact", "1.0.*", "1.0.0", true},

		// Revision float
		{"revision float match", "1.0.0.*", "1.0.0", true},

		// Prerelease float
		{"prerelease float match", "1.0.0-*", "1.0.0-beta", true},
		{"prerelease float match stable", "1.0.0-*", "1.0.0", true},
		{"prerelease float no match", "1.0.0-*", "1.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr, err := ParseFloatRange(tt.floatStr)
			if err != nil {
				t.Fatalf("ParseFloatRange() error = %v", err)
			}

			v := MustParse(tt.version)
			got := fr.Satisfies(v)

			if got != tt.expected {
				t.Errorf("Satisfies(%s) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestFloatRange_FindBestMatch(t *testing.T) {
	versions := []*NuGetVersion{
		MustParse("1.0.0"),
		MustParse("1.0.5"),
		MustParse("1.5.0"),
		MustParse("2.0.0"),
		MustParse("2.5.0"),
	}

	tests := []struct {
		name     string
		floatStr string
		expected string
	}{
		{"wildcard", "*", "2.5.0"},
		{"major 1", "1.*", "1.5.0"},
		{"major 2", "2.*", "2.5.0"},
		{"minor 1.0", "1.0.*", "1.0.5"},
		{"no match", "3.*", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr, err := ParseFloatRange(tt.floatStr)
			if err != nil {
				t.Fatalf("ParseFloatRange() error = %v", err)
			}

			got := fr.FindBestMatch(versions)

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

func TestFloatBehavior_String(t *testing.T) {
	tests := []struct {
		behavior FloatBehavior
		expected string
	}{
		{FloatNone, "none"},
		{FloatPrerelease, "prerelease"},
		{FloatRevision, "revision"},
		{FloatPatch, "patch"},
		{FloatMinor, "minor"},
		{FloatMajor, "major"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.behavior.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
