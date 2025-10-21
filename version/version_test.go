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

func TestNuGetVersion_IsPrerelease(t *testing.T) {
	tests := []struct {
		name     string
		version  *NuGetVersion
		expected bool
	}{
		{
			name: "stable version - not prerelease",
			version: &NuGetVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			expected: false,
		},
		{
			name: "version with metadata only - not prerelease",
			version: &NuGetVersion{
				Major:    1,
				Minor:    0,
				Patch:    0,
				Metadata: "build123",
			},
			expected: false,
		},
		{
			name: "version with single release label - is prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         0,
				Patch:         0,
				ReleaseLabels: []string{"beta"},
			},
			expected: true,
		},
		{
			name: "version with multiple release labels - is prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         0,
				Patch:         0,
				ReleaseLabels: []string{"rc", "1"},
			},
			expected: true,
		},
		{
			name: "version with empty release labels array - not prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         0,
				Patch:         0,
				ReleaseLabels: []string{},
			},
			expected: false,
		},
		{
			name: "version with empty string in release labels - not prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         0,
				Patch:         0,
				ReleaseLabels: []string{""},
			},
			expected: false,
		},
		{
			name: "version with non-empty label after empty - is prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         0,
				Patch:         0,
				ReleaseLabels: []string{"", "beta"},
			},
			expected: true,
		},
		{
			name: "prerelease with metadata - is prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         0,
				Patch:         0,
				ReleaseLabels: []string{"alpha"},
				Metadata:      "build456",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.IsPrerelease()
			if got != tt.expected {
				t.Errorf("IsPrerelease() = %v, want %v", got, tt.expected)
			}
		})
	}
}
