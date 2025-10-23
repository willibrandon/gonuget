package frameworks

import "testing"

// TestGetShortFolderName tests the GetShortFolderName method.
func TestGetShortFolderName(t *testing.T) {
	tests := []struct {
		name     string
		fw       *NuGetFramework
		expected string
	}{
		// .NET Framework compact versions
		{
			name: "net48",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			expected: "net48",
		},
		{
			name: "net472",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			},
			expected: "net472",
		},
		{
			name: "net471",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 7, Build: 1},
			},
			expected: "net471",
		},
		{
			name: "net47",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 7},
			},
			expected: "net47",
		},
		{
			name: "net463",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 6, Build: 3},
			},
			expected: "net463",
		},
		{
			name: "net462",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 6, Build: 2},
			},
			expected: "net462",
		},
		{
			name: "net461",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 6, Build: 1},
			},
			expected: "net461",
		},
		{
			name: "net46",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 6},
			},
			expected: "net46",
		},
		{
			name: "net452",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 5, Build: 2},
			},
			expected: "net452",
		},
		{
			name: "net451",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 5, Build: 1},
			},
			expected: "net451",
		},
		{
			name: "net45",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 5},
			},
			expected: "net45",
		},
		{
			name: "net403",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 0, Build: 3},
			},
			expected: "net403",
		},
		{
			name: "net40",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 0},
			},
			expected: "net40",
		},
		{
			name: "net35",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 3, Minor: 5},
			},
			expected: "net35",
		},
		{
			name: "net20",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			expected: "net20",
		},
		{
			name: "net11",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 1, Minor: 1},
			},
			expected: "net11",
		},

		// .NET Standard
		{
			name: "netstandard2.1",
			fw: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			expected: "netstandard2.1",
		},
		{
			name: "netstandard2.0",
			fw: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			expected: "netstandard2.0",
		},
		{
			name: "netstandard1.6",
			fw: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 1, Minor: 6},
			},
			expected: "netstandard1.6",
		},

		// .NET Core
		{
			name: "netcoreapp3.1",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 3, Minor: 1},
			},
			expected: "netcoreapp3.1",
		},
		{
			name: "netcoreapp2.1",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			expected: "netcoreapp2.1",
		},

		// .NET 5+
		{
			name: "net8.0",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 8, Minor: 0},
			},
			expected: "net8.0",
		},
		{
			name: "net7.0",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 7, Minor: 0},
			},
			expected: "net7.0",
		},
		{
			name: "net6.0",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 6, Minor: 0},
			},
			expected: "net6.0",
		},
		{
			name: "net5.0",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 5, Minor: 0},
			},
			expected: "net5.0",
		},

		// .NET 5+ with platforms
		{
			name: "net6.0-windows",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 6, Minor: 0},
				Platform:  "windows",
			},
			expected: "net6.0-windows",
		},
		{
			name: "net6.0-windows10.0",
			fw: &NuGetFramework{
				Framework:       ".NETCoreApp",
				Version:         FrameworkVersion{Major: 6, Minor: 0},
				Platform:        "windows",
				PlatformVersion: FrameworkVersion{Major: 10, Minor: 0},
			},
			expected: "net6.0-windows10.0",
		},
		{
			name: "net6.0-android",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 6, Minor: 0},
				Platform:  "android",
			},
			expected: "net6.0-android",
		},
		{
			name: "net6.0-android31.0",
			fw: &NuGetFramework{
				Framework:       ".NETCoreApp",
				Version:         FrameworkVersion{Major: 6, Minor: 0},
				Platform:        "android",
				PlatformVersion: FrameworkVersion{Major: 31, Minor: 0},
			},
			expected: "net6.0-android31.0",
		},

		// .NET Framework with profiles
		{
			name: "net40-client",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 0},
				Profile:   "Client",
			},
			expected: "net40-client",
		},

		// Special frameworks
		{
			name: "any",
			fw: &NuGetFramework{
				Framework: "Any",
			},
			expected: "any",
		},
	}

	provider := DefaultFrameworkNameProvider()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fw.GetShortFolderName(provider)
			if got != tt.expected {
				t.Errorf("GetShortFolderName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestGetShortFolderName_PCL tests PCL formatting.
// NuGet.Client expands profile numbers to framework lists.
func TestGetShortFolderName_PCL(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{
			name:     "Profile7",
			profile:  "Profile7",
			expected: "portable-net45+win8", // Expanded from Profile7
		},
		{
			name:     "Profile259",
			profile:  "Profile259",
			expected: "portable-net45+win8+wp8+wpa81", // Expanded from Profile259
		},
		{
			name:     "Profile111",
			profile:  "Profile111",
			expected: "portable-net45+win8+wpa81", // Expanded from Profile111
		},
	}

	provider := DefaultFrameworkNameProvider()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := &NuGetFramework{
				Framework: ".NETPortable",
				Profile:   tt.profile,
			}
			got := fw.GetShortFolderName(provider)
			if got != tt.expected {
				t.Errorf("GetShortFolderName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestFormat tests the format() method (which calls GetShortFolderName).
func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		tfm      string
		expected string
	}{
		{"net8.0", "net8.0", "net8.0"},
		{"net48", "net48", "net48"},
		{"netstandard2.1", "netstandard2.1", "netstandard2.1"},
		{"net6.0-windows", "net6.0-windows", "net6.0-windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := ParseFramework(tt.tfm)
			if err != nil {
				t.Fatalf("ParseFramework() error = %v", err)
			}
			got := fw.format()
			if got != tt.expected {
				t.Errorf("format() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestParsePCL_ProfileResolution tests PCL profile number resolution.
// When parsing framework lists, they are stored as-is (not converted to profile numbers).
func TestParsePCL_ProfileResolution(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedProfile string
	}{
		{
			name:            "Profile7 by name",
			input:           "portable-Profile7",
			expectedProfile: "Profile7",
		},
		{
			name:            "Profile259 by name",
			input:           "portable-Profile259",
			expectedProfile: "Profile259",
		},
		{
			name:            "Profile7 by frameworks",
			input:           "portable-net45+win8",
			expectedProfile: "net45+win8", // Stored as framework list, not converted to Profile7
		},
		{
			name:            "Profile259 by frameworks",
			input:           "portable-net45+win8+wpa81+wp8",
			expectedProfile: "net45+win8+wp8+wpa81", // Sorted alphabetically
		},
		{
			name:            "Profile111 by frameworks",
			input:           "portable-net45+win8+wpa81",
			expectedProfile: "net45+win8+wpa81", // Sorted alphabetically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := ParseFramework(tt.input)
			if err != nil {
				t.Fatalf("ParseFramework() error = %v", err)
			}
			if fw.Framework != ".NETPortable" {
				t.Errorf("Framework = %v, want .NETPortable", fw.Framework)
			}
			if fw.Profile != tt.expectedProfile {
				t.Errorf("Profile = %v, want %v", fw.Profile, tt.expectedProfile)
			}
		})
	}
}

// TestParsePCL_CustomProfile tests PCL with custom (unrecognized) profiles.
func TestParsePCL_CustomProfile(t *testing.T) {
	input := "portable-netstandard1.0+netcoreapp2.0"
	fw, err := ParseFramework(input)
	if err != nil {
		t.Fatalf("ParseFramework() error = %v", err)
	}

	if fw.Framework != ".NETPortable" {
		t.Errorf("Framework = %v, want .NETPortable", fw.Framework)
	}

	// Custom profile should be stored as-is
	if fw.Profile == "" {
		t.Error("Profile should not be empty for custom PCL")
	}
}

// TestRoundTrip tests that parsing and formatting are consistent.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		tfm  string
	}{
		{"net8.0", "net8.0"},
		{"net6.0", "net6.0"},
		{"net48", "net48"},
		{"net472", "net472"},
		{"netstandard2.1", "netstandard2.1"},
		{"netstandard2.0", "netstandard2.0"},
		{"netcoreapp3.1", "netcoreapp3.1"},
		{"net6.0-windows", "net6.0-windows"},
		{"portable-profile7", "portable-profile7"},
		{"portable-profile259", "portable-profile259"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := ParseFramework(tt.tfm)
			if err != nil {
				t.Fatalf("ParseFramework() error = %v", err)
			}

			// Clear originalString to force format() to generate
			fw.originalString = ""
			got := fw.String()

			if got != tt.tfm {
				t.Errorf("Round trip: got %v, want %v", got, tt.tfm)
			}
		})
	}
}
