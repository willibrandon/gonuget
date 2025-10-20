package version

import "testing"

func TestNuGetVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  *NuGetVersion
		expected string
	}{
		{
			name: "simple version",
			version: &NuGetVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			expected: "1.0.0",
		},
		{
			name: "version with prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				ReleaseLabels: []string{"beta", "1"},
			},
			expected: "1.2.3-beta.1",
		},
		{
			name: "version with metadata",
			version: &NuGetVersion{
				Major:    1,
				Minor:    0,
				Patch:    0,
				Metadata: "20241019",
			},
			expected: "1.0.0+20241019",
		},
		{
			name: "legacy 4-part version",
			version: &NuGetVersion{
				Major:           2,
				Minor:           5,
				Patch:           3,
				Revision:        1,
				IsLegacyVersion: true,
			},
			expected: "2.5.3.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
