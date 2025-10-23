package frameworks

import (
	"fmt"
	"sort"
	"strings"
)

// FrameworkNameProvider provides framework name mappings and formatting.
// Reference: DefaultFrameworkNameProvider in NuGet.Client
type FrameworkNameProvider interface {
	// TryGetShortIdentifier gets the short framework identifier.
	TryGetShortIdentifier(identifier string) (string, bool)

	// GetVersionString formats a version for the given framework.
	GetVersionString(framework string, version FrameworkVersion) string

	// TryGetShortProfile gets the short profile name.
	TryGetShortProfile(framework, profile string) (string, bool)

	// TryGetPortableFrameworks parses a portable profile string into frameworks.
	TryGetPortableFrameworks(profile string, includeOptional bool) ([]*NuGetFramework, bool)

	// TryGetPortableProfile gets the profile number for a set of frameworks.
	TryGetPortableProfile(frameworks []*NuGetFramework) (int, bool)
}

type defaultFrameworkNameProvider struct {
	identifierMappings     map[string]string
	profileMappings        map[string]string
	portableProfiles       map[string][]string // profile -> framework list
	portableProfileNumbers map[string]int      // sorted framework key -> profile number
}

var defaultProvider *defaultFrameworkNameProvider

// DefaultFrameworkNameProvider returns the default framework name provider.
func DefaultFrameworkNameProvider() FrameworkNameProvider {
	if defaultProvider == nil {
		defaultProvider = newDefaultProvider()
	}
	return defaultProvider
}

func newDefaultProvider() *defaultFrameworkNameProvider {
	p := &defaultFrameworkNameProvider{
		identifierMappings: map[string]string{
			".NETFramework":           "net",
			".NETStandard":            "netstandard",
			".NETCoreApp":             "netcoreapp",
			".NETPortable":            "portable",
			".NETMicroFramework":      "netmf",
			"Silverlight":             "sl",
			"Windows":                 "win",
			"WindowsPhone":            "wp",
			"WindowsPhoneApp":         "wpa",
			"DNX":                     "dnx",
			"DNXCore":                 "dnxcore",
			"UAP":                     "uap",
			"Tizen":                   "tizen",
			"MonoAndroid":             "monoandroid",
			"MonoTouch":               "monotouch",
			"MonoMac":                 "monomac",
			"Xamarin.iOS":             "xamarinios",
			"Xamarin.Mac":             "xamarinmac",
			"Xamarin.PlayStation3":    "xamarinpsthree",
			"Xamarin.PlayStation4":    "xamarinpsthreefour",
			"Xamarin.PlayStationVita": "xamarinpsvita",
			"Xamarin.TVOS":            "xamarintvos",
			"Xamarin.WatchOS":         "xamarinwatchos",
			"NetCore":                 "netcore",
		},
		profileMappings: map[string]string{
			"Client": "client",
			"Full":   "",
		},
		portableProfiles:       make(map[string][]string),
		portableProfileNumbers: make(map[string]int),
	}

	// Populate portable profile mappings
	// Reference: DefaultPortableFrameworkMappings.cs
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

	for profileNum, frameworks := range profiles {
		profileKey := fmt.Sprintf("Profile%d", profileNum)
		p.portableProfiles[profileKey] = frameworks

		// Create sorted key for reverse lookup
		sortedFrameworks := make([]string, len(frameworks))
		copy(sortedFrameworks, frameworks)
		sort.Strings(sortedFrameworks)
		key := strings.Join(sortedFrameworks, "+")
		p.portableProfileNumbers[key] = profileNum
	}

	return p
}

func (p *defaultFrameworkNameProvider) TryGetShortIdentifier(identifier string) (string, bool) {
	short, ok := p.identifierMappings[identifier]
	return short, ok
}

func (p *defaultFrameworkNameProvider) GetVersionString(framework string, version FrameworkVersion) string {
	if version.IsEmpty() {
		return ""
	}

	// For .NET Framework, use compact format for common versions
	if framework == ".NETFramework" {
		// Map major.minor to compact string
		switch version.Major {
		case 4:
			switch version.Minor {
			case 8:
				switch version.Build {
				case 1:
					return "481"
				default:
					return "48"
				}
			case 7:
				switch version.Build {
				case 2:
					return "472"
				case 1:
					return "471"
				default:
					return "47"
				}
			case 6:
				switch version.Build {
				case 3:
					return "463"
				case 2:
					return "462"
				case 1:
					return "461"
				default:
					return "46"
				}
			case 5:
				switch version.Build {
				case 2:
					return "452"
				case 1:
					return "451"
				default:
					return "45"
				}
			case 0:
				switch version.Build {
				case 3:
					return "403"
				default:
					return "40"
				}
			}
		case 3:
			if version.Minor == 5 {
				return "35"
			}
		case 2:
			if version.Minor == 0 {
				return "20"
			}
		case 1:
			if version.Minor == 1 {
				return "11"
			}
		}
	}

	// For .NET 5+ (single digit major version >= 5), use "X.Y" format
	if framework == ".NETCoreApp" && version.Major >= 5 {
		if version.Minor == 0 {
			return fmt.Sprintf("%d.0", version.Major)
		}
		return fmt.Sprintf("%d.%d", version.Major, version.Minor)
	}

	// For .NET Standard, .NET Core, use "X.Y" format
	if framework == ".NETStandard" || framework == ".NETCoreApp" {
		if version.Minor == 0 && version.Build == 0 {
			return fmt.Sprintf("%d.0", version.Major)
		} else if version.Build == 0 {
			return fmt.Sprintf("%d.%d", version.Major, version.Minor)
		}
		return fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Build)
	}

	// For legacy PCL frameworks (Windows, WindowsPhone, etc.), use compact format
	if framework == "Windows" || framework == "WindowsPhone" || framework == "WindowsPhoneApp" ||
		framework == "Silverlight" {
		// Use compact format: 8.0 → "8", 8.1 → "81"
		if version.Build > 0 {
			return fmt.Sprintf("%d%d%d", version.Major, version.Minor, version.Build)
		} else if version.Minor > 0 {
			return fmt.Sprintf("%d%d", version.Major, version.Minor)
		}
		return fmt.Sprintf("%d", version.Major)
	}

	// Default: Major.Minor format
	if version.Minor == 0 {
		return fmt.Sprintf("%d.0", version.Major)
	}
	return fmt.Sprintf("%d.%d", version.Major, version.Minor)
}

func (p *defaultFrameworkNameProvider) TryGetShortProfile(framework, profile string) (string, bool) {
	// Check profile mappings
	if short, ok := p.profileMappings[profile]; ok {
		return short, true
	}

	// Some profiles are lowercase versions - convert to title case for lookup
	lower := strings.ToLower(profile)
	var titleCase string
	switch lower {
	case "client":
		titleCase = "Client"
	case "full":
		titleCase = "Full"
	default:
		return "", false
	}
	if short, ok := p.profileMappings[titleCase]; ok {
		return short, true
	}

	return "", false
}

func (p *defaultFrameworkNameProvider) TryGetPortableFrameworks(profile string, includeOptional bool) ([]*NuGetFramework, bool) {
	// Check if this is a known profile (e.g., "Profile259")
	if strings.HasPrefix(profile, "Profile") {
		if frameworks, ok := p.portableProfiles[profile]; ok {
			result := make([]*NuGetFramework, len(frameworks))
			for i, fw := range frameworks {
				parsed, err := ParseFramework(fw)
				if err != nil {
					return nil, false
				}
				result[i] = parsed
			}
			return result, true
		}
	}

	// Try parsing as framework list (e.g., "net45+win8+wp8+wpa81")
	parts := strings.Split(profile, "+")
	if len(parts) > 0 {
		result := make([]*NuGetFramework, len(parts))
		for i, part := range parts {
			parsed, err := ParseFramework(strings.TrimSpace(part))
			if err != nil {
				return nil, false
			}
			result[i] = parsed
		}
		return result, true
	}

	return nil, false
}

func (p *defaultFrameworkNameProvider) TryGetPortableProfile(frameworks []*NuGetFramework) (int, bool) {
	if len(frameworks) == 0 {
		return 0, false
	}

	// Format each framework and sort
	shortNames := make([]string, len(frameworks))
	for i, fw := range frameworks {
		shortNames[i] = fw.GetShortFolderName(p)
	}
	sort.Strings(shortNames)
	key := strings.Join(shortNames, "+")

	// Look up profile number
	if profileNum, ok := p.portableProfileNumbers[key]; ok {
		return profileNum, true
	}

	return 0, false
}
