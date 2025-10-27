package frameworks

import "testing"

// TestCalculateCompatibilityScore_EdgeCases tests uncovered paths in calculateCompatibilityScore.
func TestCalculateCompatibilityScore_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		fw       *NuGetFramework
		target   *NuGetFramework
		minScore int // Minimum expected score
	}{
		{
			name: "Same framework, .NET Framework older version",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 5},
			},
			target: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			minScore: 800, // Same framework base score
		},
		{
			name: "Same framework, version diff <= 2",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 6, Minor: 0},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 7, Minor: 0},
			},
			minScore: 800, // Same framework base score
		},
		{
			name: "Same framework, distant versions",
			fw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 8, Minor: 0},
			},
			minScore: 800, // Same framework base score
		},
		{
			name: ".NET Standard → .NET Framework (cross-framework)",
			fw: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 0},
			},
			target: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 7, Build: 2},
			},
			minScore: 850, // Cross-framework with .NET Framework target bonus
		},
		{
			name: ".NET Standard → .NET Core (cross-framework)",
			fw: &NuGetFramework{
				Framework: ".NETStandard",
				Version:   FrameworkVersion{Major: 2, Minor: 1},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 3, Minor: 1},
			},
			minScore: 700, // Base cross-framework score
		},
		{
			name: "Different frameworks (fallback to precedence)",
			fw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 8, Minor: 0},
			},
			minScore: 0, // Just precedence-based scoring
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateCompatibilityScore(tt.fw, tt.target)
			if score < tt.minScore {
				t.Errorf("calculateCompatibilityScore() = %v, want >= %v", score, tt.minScore)
			}
		})
	}
}

// TestIsCompatible_NonNetStandard tests compatibility paths for non-.NET Standard frameworks.
func TestIsCompatible_NonNetStandard(t *testing.T) {
	tests := []struct {
		name       string
		packageFw  *NuGetFramework
		target     *NuGetFramework
		compatible bool
	}{
		{
			name: ".NET Framework 4.8 → .NET Framework 4.8",
			packageFw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			target: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			compatible: true,
		},
		{
			name: ".NET Framework 4.8 → .NET Framework 4.5 (incompatible)",
			packageFw: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 8},
			},
			target: &NuGetFramework{
				Framework: ".NETFramework",
				Version:   FrameworkVersion{Major: 4, Minor: 5},
			},
			compatible: false,
		},
		{
			name: ".NET Core 3.1 → .NET Core 3.1",
			packageFw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 3, Minor: 1},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 3, Minor: 1},
			},
			compatible: true,
		},
		{
			name: ".NET Core 3.1 → .NET 5.0",
			packageFw: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 3, Minor: 1},
			},
			target: &NuGetFramework{
				Framework: ".NETCoreApp",
				Version:   FrameworkVersion{Major: 5, Minor: 0},
			},
			compatible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.packageFw.IsCompatible(tt.target)
			if got != tt.compatible {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.compatible)
			}
		})
	}
}

// TestIsNetStandardCompatibleWithCoreApp_EdgeCases tests edge cases for .NET Standard → .NET Core compatibility.
func TestIsNetStandardCompatibleWithCoreApp_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		nsVersion   FrameworkVersion
		coreVersion FrameworkVersion
		compatible  bool
	}{
		{
			name:        ".NET Standard 1.3 → .NET Core 1.0",
			nsVersion:   FrameworkVersion{Major: 1, Minor: 3},
			coreVersion: FrameworkVersion{Major: 1, Minor: 0},
			compatible:  true,
		},
		{
			name:        ".NET Standard 1.5 → .NET Core 1.0",
			nsVersion:   FrameworkVersion{Major: 1, Minor: 5},
			coreVersion: FrameworkVersion{Major: 1, Minor: 0},
			compatible:  true,
		},
		{
			name:        ".NET Standard 2.0 → .NET Core 2.0",
			nsVersion:   FrameworkVersion{Major: 2, Minor: 0},
			coreVersion: FrameworkVersion{Major: 2, Minor: 0},
			compatible:  true,
		},
		{
			name:        ".NET Standard 9.9 (unknown) → .NET Core 8.0 (incompatible)",
			nsVersion:   FrameworkVersion{Major: 9, Minor: 9},
			coreVersion: FrameworkVersion{Major: 8, Minor: 0},
			compatible:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNetStandardCompatibleWithCoreApp(tt.nsVersion, tt.coreVersion)
			if got != tt.compatible {
				t.Errorf("isNetStandardCompatibleWithCoreApp() = %v, want %v", got, tt.compatible)
			}
		})
	}
}

// TestParseCompactVersion_EdgeCases tests edge cases for compact version parsing.
func TestParseCompactVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantBuild int
		wantRev   int
		wantErr   bool
	}{
		{
			name:      "Two digit version (net45)",
			input:     "45",
			wantMajor: 4,
			wantMinor: 5,
			wantBuild: 0,
			wantRev:   0,
			wantErr:   false,
		},
		{
			name:      "Three digit version (net472)",
			input:     "472",
			wantMajor: 4,
			wantMinor: 7,
			wantBuild: 2,
			wantRev:   0,
			wantErr:   false,
		},
		{
			name:      "Four digit version (net4721)",
			input:     "4721",
			wantMajor: 4,
			wantMinor: 7,
			wantBuild: 2,
			wantRev:   1,
			wantErr:   false,
		},
		{
			name:      "Single digit (net8 for PCL frameworks)",
			input:     "8",
			wantMajor: 8,
			wantMinor: 0,
			wantBuild: 0,
			wantRev:   0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCompactVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCompactVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.wantMajor {
					t.Errorf("Major = %v, want %v", got.Major, tt.wantMajor)
				}
				if got.Minor != tt.wantMinor {
					t.Errorf("Minor = %v, want %v", got.Minor, tt.wantMinor)
				}
				if got.Build != tt.wantBuild {
					t.Errorf("Build = %v, want %v", got.Build, tt.wantBuild)
				}
				if got.Revision != tt.wantRev {
					t.Errorf("Revision = %v, want %v", got.Revision, tt.wantRev)
				}
			}
		})
	}
}
