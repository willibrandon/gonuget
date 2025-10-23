package frameworks

import "testing"

// TestFrameworkVersion_String tests version string formatting.
func TestFrameworkVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  FrameworkVersion
		expected string
	}{
		{
			name:     "Full version with revision",
			version:  FrameworkVersion{Major: 4, Minor: 7, Build: 2, Revision: 1},
			expected: "4.7.2.1",
		},
		{
			name:     "Version with build",
			version:  FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			expected: "4.7.2",
		},
		{
			name:     "Version with minor",
			version:  FrameworkVersion{Major: 6, Minor: 0},
			expected: "6.0",
		},
		{
			name:     "Major version only",
			version:  FrameworkVersion{Major: 5},
			expected: "5.0",
		},
		{
			name:     "Zero version",
			version:  FrameworkVersion{},
			expected: "0.0",
		},
		{
			name:     "All components",
			version:  FrameworkVersion{Major: 1, Minor: 2, Build: 3, Revision: 4},
			expected: "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestFrameworkVersion_Compare tests version comparison.
func TestFrameworkVersion_Compare(t *testing.T) {
	tests := []struct {
		name     string
		v1       FrameworkVersion
		v2       FrameworkVersion
		expected int
	}{
		{
			name:     "Equal versions",
			v1:       FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			v2:       FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			expected: 0,
		},
		{
			name:     "v1 major > v2 major",
			v1:       FrameworkVersion{Major: 5, Minor: 0},
			v2:       FrameworkVersion{Major: 4, Minor: 8},
			expected: 1,
		},
		{
			name:     "v1 major < v2 major",
			v1:       FrameworkVersion{Major: 4, Minor: 8},
			v2:       FrameworkVersion{Major: 5, Minor: 0},
			expected: -1,
		},
		{
			name:     "v1 minor > v2 minor",
			v1:       FrameworkVersion{Major: 4, Minor: 8},
			v2:       FrameworkVersion{Major: 4, Minor: 7},
			expected: 1,
		},
		{
			name:     "v1 minor < v2 minor",
			v1:       FrameworkVersion{Major: 4, Minor: 7},
			v2:       FrameworkVersion{Major: 4, Minor: 8},
			expected: -1,
		},
		{
			name:     "v1 build > v2 build",
			v1:       FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			v2:       FrameworkVersion{Major: 4, Minor: 7, Build: 1},
			expected: 1,
		},
		{
			name:     "v1 build < v2 build",
			v1:       FrameworkVersion{Major: 4, Minor: 7, Build: 1},
			v2:       FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			expected: -1,
		},
		{
			name:     "v1 revision > v2 revision",
			v1:       FrameworkVersion{Major: 4, Minor: 7, Build: 2, Revision: 1},
			v2:       FrameworkVersion{Major: 4, Minor: 7, Build: 2, Revision: 0},
			expected: 1,
		},
		{
			name:     "v1 revision < v2 revision",
			v1:       FrameworkVersion{Major: 4, Minor: 7, Build: 2, Revision: 0},
			v2:       FrameworkVersion{Major: 4, Minor: 7, Build: 2, Revision: 1},
			expected: -1,
		},
		{
			name:     "Zero versions equal",
			v1:       FrameworkVersion{},
			v2:       FrameworkVersion{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v1.Compare(tt.v2)
			if got != tt.expected {
				t.Errorf("Compare() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestGetShortFolderName_EdgeCases tests edge cases for GetShortFolderName.
func TestGetShortFolderName_EdgeCases(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	tests := []struct {
		name     string
		fw       *NuGetFramework
		expected string
	}{
		{
			name: "Unsupported framework",
			fw: &NuGetFramework{
				Framework: "Unsupported",
			},
			expected: "unsupported",
		},
		{
			name: "Empty framework",
			fw: &NuGetFramework{
				Framework: "",
			},
			expected: "",
		},
		{
			name: "Custom profile",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 5},
				Profile:   "CustomProfile",
			},
			expected: "net45-customprofile",
		},
		{
			name: "PCL with custom frameworks",
			fw: &NuGetFramework{
				Framework: ".NETPortable",
				Profile:   "netstandard1.0+netcoreapp2.0",
			},
			expected: "portable-netcoreapp2.0+netstandard1.0",
		},
		{
			name: "Unknown framework identifier",
			fw: &NuGetFramework{
				Framework: "CustomFramework",
				Version:   FrameworkVersion{Major: 1, Minor: 0},
			},
			expected: "customframework1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fw.GetShortFolderName(provider)
			if got != tt.expected {
				t.Errorf("GetShortFolderName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsNetStandardCompatible_EdgeCases tests .NET Standard compatibility edge cases.
func TestIsNetStandardCompatible_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		nsPackage  *NuGetFramework
		target     *NuGetFramework
		compatible bool
	}{
		{
			name: ".NET Standard 2.1 → .NET Framework 4.8 (incompatible)",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			target: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			compatible: false,
		},
		{
			name: ".NET Standard 2.0 → .NET Framework 4.6.1",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			target: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 6, Build: 1},
			},
			compatible: true,
		},
		{
			name: ".NET Standard 1.0 → .NET Core 1.0",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 1, Minor: 0},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 1, Minor: 0},
			},
			compatible: true,
		},
		{
			name: ".NET Standard 2.1 → .NET 5.0",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 5, Minor: 0},
			},
			compatible: true,
		},
		{
			name: ".NET Standard 2.0 → .NET Standard 2.1",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			target: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			compatible: true,
		},
		{
			name: ".NET Standard 2.1 → .NET Core 2.0 (incompatible)",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			compatible: false,
		},
		{
			name: ".NET Standard 1.6 → .NET Core 1.1",
			nsPackage: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 1, Minor: 6},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 1, Minor: 1},
			},
			compatible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.nsPackage.IsCompatible(tt.target)
			if got != tt.compatible {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.compatible)
			}
		})
	}
}

// TestNuGetFramework_String_EdgeCases tests NuGetFramework.String() edge cases.
func TestNuGetFramework_String_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		fw       *NuGetFramework
		expected string
	}{
		{
			name: "Platform with version",
			fw: &NuGetFramework{
				Framework:       ".NETCoreApp",
				Version:         FrameworkVersion{Major: 6, Minor: 0},
				Platform:        "windows",
				PlatformVersion: FrameworkVersion{Major: 10, Minor: 0},
			},
			expected: "net6.0-windows10.0",
		},
		{
			name: "PCL with profile",
			fw: &NuGetFramework{
				Framework: ".NETPortable",
				Profile:   "Profile7",
			},
			// String() falls back to GetShortFolderName() which expands Profile7
			expected: "portable-net45+win8",
		},
		{
			name: "Framework with profile",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 0},
				Profile:   "Client",
			},
			expected: "net40-client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fw.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
