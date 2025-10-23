package frameworks

import (
	"fmt"
	"testing"
)

// TestDefaultFrameworkNameProvider_TryGetShortIdentifier tests identifier mappings.
func TestDefaultFrameworkNameProvider_TryGetShortIdentifier(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	tests := []struct {
		name       string
		identifier string
		want       string
		wantOk     bool
	}{
		{".NETFramework", ".NETFramework", "net", true},
		{".NETStandard", ".NETStandard", "netstandard", true},
		{".NETCoreApp", ".NETCoreApp", "netcoreapp", true},
		{".NETPortable", ".NETPortable", "portable", true},
		{"UAP", "UAP", "uap", true},
		{"Xamarin.iOS", "Xamarin.iOS", "xamarinios", true},
		{"Unknown", "Unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := provider.TryGetShortIdentifier(tt.identifier)
			if ok != tt.wantOk {
				t.Errorf("TryGetShortIdentifier() ok = %v, want %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("TryGetShortIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDefaultFrameworkNameProvider_GetVersionString tests version formatting.
func TestDefaultFrameworkNameProvider_GetVersionString(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	tests := []struct {
		name      string
		framework string
		version   FrameworkVersion
		want      string
	}{
		// .NET Framework compact format
		{"net4.8", ".NETFramework", FrameworkVersion{Major: 4, Minor: 8}, "48"},
		{"net4.7.2", ".NETFramework", FrameworkVersion{Major: 4, Minor: 7, Build: 2}, "472"},
		{"net4.7.1", ".NETFramework", FrameworkVersion{Major: 4, Minor: 7, Build: 1}, "471"},
		{"net4.7", ".NETFramework", FrameworkVersion{Major: 4, Minor: 7}, "47"},
		{"net4.6.3", ".NETFramework", FrameworkVersion{Major: 4, Minor: 6, Build: 3}, "463"},
		{"net4.6.2", ".NETFramework", FrameworkVersion{Major: 4, Minor: 6, Build: 2}, "462"},
		{"net4.6.1", ".NETFramework", FrameworkVersion{Major: 4, Minor: 6, Build: 1}, "461"},
		{"net4.6", ".NETFramework", FrameworkVersion{Major: 4, Minor: 6}, "46"},
		{"net4.5.2", ".NETFramework", FrameworkVersion{Major: 4, Minor: 5, Build: 2}, "452"},
		{"net4.5.1", ".NETFramework", FrameworkVersion{Major: 4, Minor: 5, Build: 1}, "451"},
		{"net4.5", ".NETFramework", FrameworkVersion{Major: 4, Minor: 5}, "45"},
		{"net4.0.3", ".NETFramework", FrameworkVersion{Major: 4, Minor: 0, Build: 3}, "403"},
		{"net4.0", ".NETFramework", FrameworkVersion{Major: 4, Minor: 0}, "40"},
		{"net3.5", ".NETFramework", FrameworkVersion{Major: 3, Minor: 5}, "35"},
		{"net2.0", ".NETFramework", FrameworkVersion{Major: 2, Minor: 0}, "20"},
		{"net1.1", ".NETFramework", FrameworkVersion{Major: 1, Minor: 1}, "11"},

		// .NET 5+ (X.Y format)
		{"net8.0", ".NETCoreApp", FrameworkVersion{Major: 8, Minor: 0}, "8.0"},
		{"net7.0", ".NETCoreApp", FrameworkVersion{Major: 7, Minor: 0}, "7.0"},
		{"net6.0", ".NETCoreApp", FrameworkVersion{Major: 6, Minor: 0}, "6.0"},
		{"net5.0", ".NETCoreApp", FrameworkVersion{Major: 5, Minor: 0}, "5.0"},
		{"net6.1", ".NETCoreApp", FrameworkVersion{Major: 6, Minor: 1}, "6.1"},

		// .NET Standard (X.Y format)
		{"netstandard2.1", ".NETStandard", FrameworkVersion{Major: 2, Minor: 1}, "2.1"},
		{"netstandard2.0", ".NETStandard", FrameworkVersion{Major: 2, Minor: 0}, "2.0"},
		{"netstandard1.6", ".NETStandard", FrameworkVersion{Major: 1, Minor: 6}, "1.6"},

		// .NET Core (X.Y format)
		{"netcoreapp3.1", ".NETCoreApp", FrameworkVersion{Major: 3, Minor: 1}, "3.1"},
		{"netcoreapp2.1", ".NETCoreApp", FrameworkVersion{Major: 2, Minor: 1}, "2.1"},

		// Empty version
		{"empty", ".NETFramework", FrameworkVersion{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.GetVersionString(tt.framework, tt.version)
			if got != tt.want {
				t.Errorf("GetVersionString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDefaultFrameworkNameProvider_TryGetShortProfile tests profile mappings.
func TestDefaultFrameworkNameProvider_TryGetShortProfile(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	tests := []struct {
		name      string
		framework string
		profile   string
		want      string
		wantOk    bool
	}{
		{"Client", ".NETFramework", "Client", "client", true},
		{"Full", ".NETFramework", "Full", "", true}, // Full maps to empty string
		{"client lowercase", ".NETFramework", "client", "client", true},
		{"Unknown", ".NETFramework", "Unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := provider.TryGetShortProfile(tt.framework, tt.profile)
			if ok != tt.wantOk {
				t.Errorf("TryGetShortProfile() ok = %v, want %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("TryGetShortProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDefaultFrameworkNameProvider_TryGetPortableFrameworks tests portable profile parsing.
func TestDefaultFrameworkNameProvider_TryGetPortableFrameworks(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	tests := []struct {
		name          string
		profile       string
		wantOk        bool
		wantCount     int
		wantFirstName string
	}{
		{
			name:          "Profile7",
			profile:       "Profile7",
			wantOk:        true,
			wantCount:     2,
			wantFirstName: "net45",
		},
		{
			name:          "Profile259",
			profile:       "Profile259",
			wantOk:        true,
			wantCount:     4,
			wantFirstName: "net45",
		},
		{
			name:          "Profile111",
			profile:       "Profile111",
			wantOk:        true,
			wantCount:     3,
			wantFirstName: "net45",
		},
		{
			name:          "Framework list",
			profile:       "net45+win8",
			wantOk:        true,
			wantCount:     2,
			wantFirstName: "net45",
		},
		{
			name:    "Unknown profile",
			profile: "ProfileUnknown",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := provider.TryGetPortableFrameworks(tt.profile, false)
			if ok != tt.wantOk {
				t.Errorf("TryGetPortableFrameworks() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("TryGetPortableFrameworks() count = %v, want %v", len(got), tt.wantCount)
			}
			if len(got) > 0 && got[0].format() != tt.wantFirstName {
				t.Errorf("TryGetPortableFrameworks() first = %v, want %v", got[0].format(), tt.wantFirstName)
			}
		})
	}
}

// TestDefaultFrameworkNameProvider_TryGetPortableProfile tests profile number resolution.
func TestDefaultFrameworkNameProvider_TryGetPortableProfile(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	tests := []struct {
		name        string
		frameworks  []string
		wantProfile int
		wantOk      bool
	}{
		{
			name:        "Profile7 (net45+win8)",
			frameworks:  []string{"net45", "win8"},
			wantProfile: 7,
			wantOk:      true,
		},
		{
			name:        "Profile259 (net45+win8+wpa81+wp8)",
			frameworks:  []string{"net45", "win8", "wpa81", "wp8"},
			wantProfile: 259,
			wantOk:      true,
		},
		{
			name:        "Profile111 (net45+win8+wpa81)",
			frameworks:  []string{"net45", "win8", "wpa81"},
			wantProfile: 111,
			wantOk:      true,
		},
		{
			name:        "Profile7 (order doesn't matter)",
			frameworks:  []string{"win8", "net45"},
			wantProfile: 7,
			wantOk:      true,
		},
		{
			name:       "Unknown combination",
			frameworks: []string{"netstandard2.0", "netcoreapp3.1"},
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse frameworks
			fws := make([]*NuGetFramework, len(tt.frameworks))
			for i, tfm := range tt.frameworks {
				fw, err := ParseFramework(tfm)
				if err != nil {
					t.Fatalf("ParseFramework(%s) error = %v", tfm, err)
				}
				fws[i] = fw
			}

			got, ok := provider.TryGetPortableProfile(fws)
			if ok != tt.wantOk {
				t.Errorf("TryGetPortableProfile() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if got != tt.wantProfile {
				t.Errorf("TryGetPortableProfile() = %v, want %v", got, tt.wantProfile)
			}
		})
	}
}

// TestPortableProfiles tests all standard portable profiles.
func TestPortableProfiles(t *testing.T) {
	provider := DefaultFrameworkNameProvider()

	profiles := map[int][]string{
		7:   {"net45", "win8"},
		31:  {"win81", "wp81"},
		32:  {"win81", "wpa81"},
		44:  {"net451", "win81"},
		49:  {"net45", "wp8"},
		78:  {"net45", "win8", "wp8"},
		84:  {"wp81", "wpa81"},
		111: {"net45", "win8", "wpa81"},
		151: {"net451", "win81", "wpa81"},
		157: {"win81", "wp81", "wpa81"},
		259: {"net45", "win8", "wpa81", "wp8"},
	}

	for profileNum, expectedFws := range profiles {
		t.Run(fmt.Sprintf("Profile%d", profileNum), func(t *testing.T) {
			profileKey := fmt.Sprintf("Profile%d", profileNum)

			// Test TryGetPortableFrameworks
			frameworks, ok := provider.TryGetPortableFrameworks(profileKey, false)
			if !ok {
				t.Errorf("TryGetPortableFrameworks(%s) failed", profileKey)
				return
			}
			if len(frameworks) != len(expectedFws) {
				t.Errorf("TryGetPortableFrameworks(%s) count = %v, want %v", profileKey, len(frameworks), len(expectedFws))
			}

			// Test TryGetPortableProfile (reverse lookup)
			gotProfile, ok := provider.TryGetPortableProfile(frameworks)
			if !ok {
				t.Errorf("TryGetPortableProfile failed for %s", profileKey)
				return
			}
			if gotProfile != profileNum {
				t.Errorf("TryGetPortableProfile() = %v, want %v", gotProfile, profileNum)
			}
		})
	}
}
