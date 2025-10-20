package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *NuGetVersion
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.0.0",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				originalString: "1.0.0",
			},
		},
		{
			name:  "version with prerelease",
			input: "1.2.3-beta",
			want: &NuGetVersion{
				Major:          1,
				Minor:          2,
				Patch:          3,
				ReleaseLabels:  []string{"beta"},
				originalString: "1.2.3-beta",
			},
		},
		{
			name:  "version with multiple prerelease labels",
			input: "1.0.0-alpha.1",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				ReleaseLabels:  []string{"alpha", "1"},
				originalString: "1.0.0-alpha.1",
			},
		},
		{
			name:  "version with metadata",
			input: "1.0.0+20241019",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				Metadata:       "20241019",
				originalString: "1.0.0+20241019",
			},
		},
		{
			name:  "version with prerelease and metadata",
			input: "1.0.0-rc.1+build.123",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				ReleaseLabels:  []string{"rc", "1"},
				Metadata:       "build.123",
				originalString: "1.0.0-rc.1+build.123",
			},
		},
		{
			name:  "major.minor only",
			input: "1.0",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				originalString: "1.0",
			},
		},
		{
			name:  "major only",
			input: "1",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				originalString: "1",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "1.2.3.4.5",
			wantErr: true,
		},
		{
			name:    "invalid major",
			input:   "a.0.0",
			wantErr: true,
		},
		{
			name:    "negative version",
			input:   "1.-1.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare fields
			if got.Major != tt.want.Major {
				t.Errorf("Major = %v, want %v", got.Major, tt.want.Major)
			}
			if got.Minor != tt.want.Minor {
				t.Errorf("Minor = %v, want %v", got.Minor, tt.want.Minor)
			}
			if got.Patch != tt.want.Patch {
				t.Errorf("Patch = %v, want %v", got.Patch, tt.want.Patch)
			}
			if got.Metadata != tt.want.Metadata {
				t.Errorf("Metadata = %v, want %v", got.Metadata, tt.want.Metadata)
			}
			if len(got.ReleaseLabels) != len(tt.want.ReleaseLabels) {
				t.Errorf("ReleaseLabels length = %v, want %v", len(got.ReleaseLabels), len(tt.want.ReleaseLabels))
			}
			for i := range got.ReleaseLabels {
				if got.ReleaseLabels[i] != tt.want.ReleaseLabels[i] {
					t.Errorf("ReleaseLabels[%d] = %v, want %v", i, got.ReleaseLabels[i], tt.want.ReleaseLabels[i])
				}
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	// Should not panic
	v := MustParse("1.0.0")
	if v.Major != 1 {
		t.Errorf("MustParse() Major = %v, want 1", v.Major)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse() should panic on invalid version")
		}
	}()
	MustParse("invalid")
}

func TestParse_Legacy(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *NuGetVersion
	}{
		{
			name:  "4-part version",
			input: "1.0.0.0",
			want: &NuGetVersion{
				Major:           1,
				Minor:           0,
				Patch:           0,
				Revision:        0,
				IsLegacyVersion: true,
				originalString:  "1.0.0.0",
			},
		},
		{
			name:  "4-part with non-zero revision",
			input: "2.5.3.1",
			want: &NuGetVersion{
				Major:           2,
				Minor:           5,
				Patch:           3,
				Revision:        1,
				IsLegacyVersion: true,
				originalString:  "2.5.3.1",
			},
		},
		{
			name:  "4-part with all non-zero",
			input: "10.20.30.40",
			want: &NuGetVersion{
				Major:           10,
				Minor:           20,
				Patch:           30,
				Revision:        40,
				IsLegacyVersion: true,
				originalString:  "10.20.30.40",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			if got.Major != tt.want.Major {
				t.Errorf("Major = %v, want %v", got.Major, tt.want.Major)
			}
			if got.Minor != tt.want.Minor {
				t.Errorf("Minor = %v, want %v", got.Minor, tt.want.Minor)
			}
			if got.Patch != tt.want.Patch {
				t.Errorf("Patch = %v, want %v", got.Patch, tt.want.Patch)
			}
			if got.Revision != tt.want.Revision {
				t.Errorf("Revision = %v, want %v", got.Revision, tt.want.Revision)
			}
			if got.IsLegacyVersion != tt.want.IsLegacyVersion {
				t.Errorf("IsLegacyVersion = %v, want %v", got.IsLegacyVersion, tt.want.IsLegacyVersion)
			}
		})
	}
}

func TestParse_Legacy_String(t *testing.T) {
	// Test that legacy versions format correctly
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0.0", "1.0.0.0"},
		{"2.5.3.1", "2.5.3.1"},
		{"10.20.30.40", "10.20.30.40"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			got := v.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
