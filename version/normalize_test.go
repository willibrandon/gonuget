package version

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Basic normalization
		{"simple version", "1.0.0", "1.0.0", false},
		{"leading zeros", "1.01.1", "1.1.1", false},
		{"single digit", "1", "1.0.0", false},
		{"two digits", "1.2", "1.2.0", false},

		// Legacy versions
		{"legacy 4-part", "1.0.0.0", "1.0.0.0", false},
		{"legacy with values", "2.5.3.1", "2.5.3.1", false},

		// Prerelease
		{"with prerelease", "1.0.0-beta", "1.0.0-beta", false},
		{"prerelease multi", "1.0.0-beta.1", "1.0.0-beta.1", false},

		// Metadata
		{"with metadata", "1.0.0+build", "1.0.0+build", false},
		{"full version", "1.0.0-rc.1+build.123", "1.0.0-rc.1+build.123", false},

		// Invalid
		{"empty string", "", "", true},
		{"invalid format", "abc", "", true},
		{"negative", "-1.0.0", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("Normalize() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMustNormalize(t *testing.T) {
	// Should not panic
	result := MustNormalize("1.0.0")
	if result != "1.0.0" {
		t.Errorf("MustNormalize() = %v, want 1.0.0", result)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNormalize() should panic on invalid version")
		}
	}()
	MustNormalize("invalid")
}

func TestNormalizeOrOriginal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid version", "1.0.0", "1.0.0"},
		{"leading zeros", "1.01.1", "1.1.1"},
		{"invalid returns original", "invalid", "invalid"},
		{"empty returns original", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeOrOriginal(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeOrOriginal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNuGetVersion_ToNormalizedString(t *testing.T) {
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
			name: "with prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				ReleaseLabels: []string{"beta", "1"},
			},
			expected: "1.2.3-beta.1",
		},
		{
			name: "with metadata",
			version: &NuGetVersion{
				Major:    1,
				Minor:    0,
				Patch:    0,
				Metadata: "build.123",
			},
			expected: "1.0.0+build.123",
		},
		{
			name: "legacy version",
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
			got := tt.version.ToNormalizedString()
			if got != tt.expected {
				t.Errorf("ToNormalizedString() = %v, want %v", got, tt.expected)
			}
		})
	}
}
