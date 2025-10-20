package frameworks

import "testing"

func TestParseFramework(t *testing.T) {
	tests := []struct {
		name          string
		tfm           string
		wantFramework string
		wantMajor     int
		wantMinor     int
		wantBuild     int
		wantPlatform  string
		wantErr       bool
	}{
		// .NET 5+ (maps to .NETCoreApp)
		{"net10.0", "net10.0", ".NETCoreApp", 10, 0, 0, "", false},
		{"net9.0", "net9.0", ".NETCoreApp", 9, 0, 0, "", false},
		{"net8.0", "net8.0", ".NETCoreApp", 8, 0, 0, "", false},
		{"net7.0", "net7.0", ".NETCoreApp", 7, 0, 0, "", false},
		{"net6.0", "net6.0", ".NETCoreApp", 6, 0, 0, "", false},
		{"net5.0", "net5.0", ".NETCoreApp", 5, 0, 0, "", false},

		// .NET Standard
		{"netstandard2.1", "netstandard2.1", ".NETStandard", 2, 1, 0, "", false},
		{"netstandard2.0", "netstandard2.0", ".NETStandard", 2, 0, 0, "", false},
		{"netstandard1.6", "netstandard1.6", ".NETStandard", 1, 6, 0, "", false},
		{"netstandard1.0", "netstandard1.0", ".NETStandard", 1, 0, 0, "", false},

		// .NET Core
		{"netcoreapp3.1", "netcoreapp3.1", ".NETCoreApp", 3, 1, 0, "", false},
		{"netcoreapp3.0", "netcoreapp3.0", ".NETCoreApp", 3, 0, 0, "", false},
		{"netcoreapp2.2", "netcoreapp2.2", ".NETCoreApp", 2, 2, 0, "", false},
		{"netcoreapp2.1", "netcoreapp2.1", ".NETCoreApp", 2, 1, 0, "", false},
		{"netcoreapp2.0", "netcoreapp2.0", ".NETCoreApp", 2, 0, 0, "", false},

		// .NET Framework (compact format)
		{"net481", "net481", ".NETFramework", 4, 8, 1, "", false},
		{"net48", "net48", ".NETFramework", 4, 8, 0, "", false},
		{"net472", "net472", ".NETFramework", 4, 7, 2, "", false},
		{"net471", "net471", ".NETFramework", 4, 7, 1, "", false},
		{"net47", "net47", ".NETFramework", 4, 7, 0, "", false},
		{"net463", "net463", ".NETFramework", 4, 6, 3, "", false},
		{"net462", "net462", ".NETFramework", 4, 6, 2, "", false},
		{"net461", "net461", ".NETFramework", 4, 6, 1, "", false},
		{"net46", "net46", ".NETFramework", 4, 6, 0, "", false},
		{"net452", "net452", ".NETFramework", 4, 5, 2, "", false},
		{"net451", "net451", ".NETFramework", 4, 5, 1, "", false},
		{"net45", "net45", ".NETFramework", 4, 5, 0, "", false},
		{"net403", "net403", ".NETFramework", 4, 0, 3, "", false},
		{"net40", "net40", ".NETFramework", 4, 0, 0, "", false},
		{"net35", "net35", ".NETFramework", 3, 5, 0, "", false},
		{"net20", "net20", ".NETFramework", 2, 0, 0, "", false},
		{"net11", "net11", ".NETFramework", 1, 1, 0, "", false},

		// Platform-specific (.NET 5+)
		{"net8.0-windows", "net8.0-windows", ".NETCoreApp", 8, 0, 0, "windows", false},
		{"net6.0-windows", "net6.0-windows", ".NETCoreApp", 6, 0, 0, "windows", false},
		{"net6.0-android", "net6.0-android", ".NETCoreApp", 6, 0, 0, "android", false},
		{"net6.0-ios", "net6.0-ios", ".NETCoreApp", 6, 0, 0, "ios", false},

		// Errors
		{"empty", "", "", 0, 0, 0, "", true},
		{"invalid", "invalid", "", 0, 0, 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFramework(tt.tfm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFramework() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Framework != tt.wantFramework {
				t.Errorf("Framework = %v, want %v", got.Framework, tt.wantFramework)
			}
			if got.Version.Major != tt.wantMajor {
				t.Errorf("Version.Major = %v, want %v", got.Version.Major, tt.wantMajor)
			}
			if got.Version.Minor != tt.wantMinor {
				t.Errorf("Version.Minor = %v, want %v", got.Version.Minor, tt.wantMinor)
			}
			if got.Version.Build != tt.wantBuild {
				t.Errorf("Version.Build = %v, want %v", got.Version.Build, tt.wantBuild)
			}
			if got.Platform != tt.wantPlatform {
				t.Errorf("Platform = %v, want %v", got.Platform, tt.wantPlatform)
			}
		})
	}
}

func TestParseFramework_PCL(t *testing.T) {
	fw, err := ParseFramework("portable-net45+win8")
	if err != nil {
		t.Fatalf("ParseFramework() error = %v", err)
	}

	if fw.Framework != ".NETPortable" {
		t.Errorf("Framework = %v, want .NETPortable", fw.Framework)
	}

	if fw.Profile == "" {
		t.Error("Profile should not be empty for PCL")
	}
}

func TestMustParseFramework(t *testing.T) {
	// Should not panic
	fw := MustParseFramework("net8.0")
	if fw.Version.Major != 8 {
		t.Errorf("Major = %v, want 8", fw.Version.Major)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseFramework should panic on invalid TFM")
		}
	}()
	MustParseFramework("invalid")
}

func TestNuGetFramework_String(t *testing.T) {
	tests := []struct {
		name     string
		tfm      string
		expected string
	}{
		{"net8.0", "net8.0", "net8.0"},
		{"netstandard2.1", "netstandard2.1", "netstandard2.1"},
		{"net6.0-windows", "net6.0-windows", "net6.0-windows"},
		{"portable", "portable-net45+win8", "portable-net45+win8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := MustParseFramework(tt.tfm)
			got := fw.String()
			if got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseFramework_PlatformVersions(t *testing.T) {
	tests := []struct {
		name              string
		tfm               string
		wantPlatform      string
		wantPlatformMajor int
		wantPlatformMinor int
		wantErr           bool
	}{
		{"android with version", "net6.0-android31.0", "android", 31, 0, false},
		{"ios with version", "net6.0-ios15.0", "ios", 15, 0, false},
		{"windows no version", "net6.0-windows", "windows", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFramework(tt.tfm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFramework() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Platform != tt.wantPlatform {
				t.Errorf("Platform = %v, want %v", got.Platform, tt.wantPlatform)
			}
			if got.PlatformVersion.Major != tt.wantPlatformMajor {
				t.Errorf("PlatformVersion.Major = %v, want %v", got.PlatformVersion.Major, tt.wantPlatformMajor)
			}
			if got.PlatformVersion.Minor != tt.wantPlatformMinor {
				t.Errorf("PlatformVersion.Minor = %v, want %v", got.PlatformVersion.Minor, tt.wantPlatformMinor)
			}
		})
	}
}

func TestParseFramework_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		tfm     string
		wantErr bool
	}{
		{"whitespace", "  net8.0  ", false},
		{"no version", "net", true},
		{"invalid compact version - 1 digit", "net4", true},
		{"invalid version format", "net8.a", true},
		{"three part version", "netstandard1.6.1", false},
		{"four digit compact version", "net4721", false}, // 4.7.2.1 - valid per NuGet.Client
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFramework(tt.tfm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFramework() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseFramework_FourDigitVersion(t *testing.T) {
	// Test 4-digit compact version support (per NuGet.Client)
	fw, err := ParseFramework("net4721")
	if err != nil {
		t.Fatalf("ParseFramework() error = %v", err)
	}

	if fw.Framework != ".NETFramework" {
		t.Errorf("Framework = %v, want .NETFramework", fw.Framework)
	}
	if fw.Version.Major != 4 {
		t.Errorf("Version.Major = %v, want 4", fw.Version.Major)
	}
	if fw.Version.Minor != 7 {
		t.Errorf("Version.Minor = %v, want 7", fw.Version.Minor)
	}
	if fw.Version.Build != 2 {
		t.Errorf("Version.Build = %v, want 2", fw.Version.Build)
	}
	if fw.Version.Revision != 1 {
		t.Errorf("Version.Revision = %v, want 1", fw.Version.Revision)
	}
}
