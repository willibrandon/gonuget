package frameworks

// This file contains framework compatibility mappings.
// Extracted from NuGet.Client DefaultFrameworkMappings.cs

// FrameworkCompatibilityMap defines which frameworks are compatible with which.
var FrameworkCompatibilityMap = map[string][]string{
	".NETStandard": {
		".NETFramework",
		".NETCoreApp",
		".NETStandard",
	},
	".NETCoreApp": {
		".NETStandard",
		".NETCoreApp",
	},
	".NETFramework": {
		".NETStandard",
		".NETFramework",
	},
}

// NetStandardCompatibilityTable defines .NET Standard → .NET Framework version mappings.
// Maps .NET Standard version to minimum .NET Framework version.
//
// Source: DefaultFrameworkMappings.cs
var NetStandardCompatibilityTable = map[string]string{
	"1.0": "4.5",
	"1.1": "4.5",
	"1.2": "4.5.1",
	"1.3": "4.6",
	"1.4": "4.6.1",
	"1.5": "4.6.1",
	"1.6": "4.6.1",
	"2.0": "4.6.1",
	// 2.1 is NOT compatible with any .NET Framework version
}

// NetStandardToCoreAppTable defines .NET Standard → .NET Core version mappings.
//
// Source: DefaultFrameworkMappings.cs
var NetStandardToCoreAppTable = map[string]string{
	"1.0": "1.0",
	"1.1": "1.0",
	"1.2": "1.0",
	"1.3": "1.0",
	"1.4": "1.0",
	"1.5": "1.0",
	"1.6": "1.0", // NetCoreApp1.0 supports NetStandard1.6
	"1.7": "1.1", // NetCoreApp1.1 supports NetStandard1.7
	"2.0": "2.0", // NetCoreApp2.0 supports NetStandard2.0
	"2.1": "3.0", // NetCoreApp3.0 supports NetStandard2.1
}

// FrameworkToNetStandardTable maps various frameworks to their .NET Standard support.
//
// Source: DefaultFrameworkMappings.cs
var FrameworkToNetStandardTable = map[string]map[string]string{
	// Tizen
	"Tizen": {
		"3.0": "1.6", // Tizen3 supports NetStandard1.6
		"4.0": "2.0", // Tizen4 supports NetStandard2.0
		"6.0": "2.1", // Tizen6 supports NetStandard2.1
	},
	// UAP
	"UAP": {
		"10.0.15064": "2.0", // UAP 10.0.15064.0 supports NetStandard2.0
		"*":          "1.4", // UAP (all versions) supports NetStandard1.4
	},
	// .NET Core App
	".NETCoreApp": {
		"1.0": "1.6", // NetCoreApp1.0 supports NetStandard1.6
		"1.1": "1.7", // NetCoreApp1.1 supports NetStandard1.7
		"2.0": "2.0", // NetCoreApp2.0 supports NetStandard2.0
		"3.0": "2.1", // NetCoreApp3.0 supports NetStandard2.1
	},
	// .NET Framework
	".NETFramework": {
		"4.5":   "1.1", // net45 supports netstandard1.1
		"4.5.1": "1.2", // net451 supports netstandard1.2
		"4.5.2": "1.2", // net452 supports netstandard1.2 (inherits from 4.5.1)
		"4.6":   "1.3", // net46 supports netstandard1.3
		"4.6.1": "2.0", // net461 supports netstandard2.0
		"4.6.2": "2.0", // net462 supports netstandard2.0
		"4.6.3": "2.0", // net463 supports netstandard2.0
		"4.7":   "2.0", // net47 supports netstandard2.0 (inherits from 4.6.1)
		"4.7.1": "2.0", // net471 supports netstandard2.0 (inherits from 4.6.1)
		"4.7.2": "2.0", // net472 supports netstandard2.0 (inherits from 4.6.1)
		"4.8":   "2.0", // net48 supports netstandard2.0 (inherits from 4.6.1)
		"4.8.1": "2.0", // net481 supports netstandard2.0 (inherits from 4.6.1)
	},
	// Windows Phone
	"WindowsPhoneApp": {
		"8.1": "1.2", // wpa81 supports netstandard1.2
	},
	"WindowsPhone": {
		"8.0": "1.0", // wp8 supports netstandard1.0
		"8.1": "1.0", // wp81 supports netstandard1.0
	},
	// NetCore (legacy)
	"NetCore": {
		"4.5":   "1.1", // netcore45 supports netstandard1.1
		"4.5.1": "1.2", // netcore451 supports netstandard1.2
		"5.0":   "1.4", // netcore50 supports netstandard1.4
	},
	// DNXCore
	"DNXCore": {
		"*": "1.5", // dnxcore50 supports netstandard1.5
	},
	// Xamarin - MonoAndroid
	"MonoAndroid": {
		"*": "2.1", // All MonoAndroid versions support netstandard2.1
	},
	// Xamarin - MonoMac
	"MonoMac": {
		"*": "2.1", // All MonoMac versions support netstandard2.1
	},
	// Xamarin - MonoTouch
	"MonoTouch": {
		"*": "2.1", // All MonoTouch versions support netstandard2.1
	},
	// Xamarin - iOS
	"Xamarin.iOS": {
		"*": "2.1", // All XamarinIOs versions support netstandard2.1
	},
	// Xamarin - Mac
	"Xamarin.Mac": {
		"*": "2.1", // All XamarinMac versions support netstandard2.1
	},
	// Xamarin - PlayStation 3
	"Xamarin.PlayStation3": {
		"*": "2.0", // All XamarinPlayStation3 versions support netstandard2.0
	},
	// Xamarin - PlayStation 4
	"Xamarin.PlayStation4": {
		"*": "2.0", // All XamarinPlayStation4 versions support netstandard2.0
	},
	// Xamarin - PlayStation Vita
	"Xamarin.PlayStationVita": {
		"*": "2.0", // All XamarinPlayStationVita versions support netstandard2.0
	},
	// Xamarin - Xbox 360
	"Xamarin.Xbox360": {
		"*": "2.0", // All XamarinXbox360 versions support netstandard2.0
	},
	// Xamarin - Xbox One
	"Xamarin.XboxOne": {
		"*": "2.0", // All XamarinXboxOne versions support netstandard2.0
	},
	// Xamarin - tvOS
	"Xamarin.TVOS": {
		"*": "2.1", // All XamarinTVOS versions support netstandard2.1
	},
	// Xamarin - watchOS
	"Xamarin.WatchOS": {
		"*": "2.1", // All XamarinWatchOS versions support netstandard2.1
	},
}

// FrameworkShortNames maps short names to full framework identifiers.
var FrameworkShortNames = map[string]string{
	"net":          ".NETFramework",
	"netframework": ".NETFramework",
	"netstandard":  ".NETStandard",
	"netcoreapp":   ".NETCoreApp",
	"netcore":      "NetCore",
	"uap":          "UAP",
	"tizen":        "Tizen",
	"monoandroid":  "MonoAndroid",
	"xamarinios":   "Xamarin.iOS",
	"xamarinmac":   "Xamarin.Mac",
}

// FrameworkPrecedence defines the precedence order for framework selection.
// Higher index = higher precedence.
var FrameworkPrecedence = []string{
	".NETStandard",
	".NETCoreApp",
	".NETFramework",
}

// GetFrameworkPrecedence returns the precedence value for a framework.
// Higher value = higher precedence.
func GetFrameworkPrecedence(framework string) int {
	for i, fw := range FrameworkPrecedence {
		if fw == framework {
			return i
		}
	}
	return -1
}
