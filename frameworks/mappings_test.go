package frameworks

import (
	"slices"
	"testing"
)

func TestFrameworkCompatibilityMap(t *testing.T) {
	// Verify .NETStandard can target .NETFramework
	compat, ok := FrameworkCompatibilityMap[".NETStandard"]
	if !ok {
		t.Fatal(".NETStandard not in compatibility map")
	}

	if !slices.Contains(compat, ".NETFramework") {
		t.Error(".NETStandard should be compatible with .NETFramework")
	}
}

func TestNetStandardCompatibilityTable(t *testing.T) {
	tests := []struct {
		nsMajor int
		nsMinor int
		minNet  FrameworkVersion
	}{
		{1, 0, FrameworkVersion{4, 5, 0, 0}},
		{1, 2, FrameworkVersion{4, 5, 1, 0}},
		{1, 3, FrameworkVersion{4, 6, 0, 0}},
		{2, 0, FrameworkVersion{4, 6, 1, 0}},
	}

	for _, tt := range tests {
		got, ok := NetStandardCompatibilityTable[versionKey{tt.nsMajor, tt.nsMinor}]
		if !ok {
			t.Errorf("netStandard %d.%d not in table", tt.nsMajor, tt.nsMinor)
			continue
		}
		if got.Compare(tt.minNet) != 0 {
			t.Errorf("netStandard %d.%d maps to %v, want %v", tt.nsMajor, tt.nsMinor, got, tt.minNet)
		}
	}
}

func TestNetStandard21NotCompatibleWithFramework(t *testing.T) {
	// .NET Standard 2.1 should NOT be in the table
	// (not compatible with any .NET Framework version)
	_, ok := NetStandardCompatibilityTable[versionKey{2, 1}]
	if ok {
		t.Error(".NET Standard 2.1 should NOT be compatible with .NET Framework")
	}
}

func TestGetFrameworkPrecedence(t *testing.T) {
	tests := []struct {
		framework string
		wantGTE   int // Greater than or equal
	}{
		{".NETStandard", 0},
		{".NETCoreApp", 0},
		{".NETFramework", 0},
		{"Unknown", -1},
	}

	for _, tt := range tests {
		got := GetFrameworkPrecedence(tt.framework)
		if got < tt.wantGTE && tt.wantGTE >= 0 {
			t.Errorf("GetFrameworkPrecedence(%s) = %d, want >= %d", tt.framework, got, tt.wantGTE)
		}
		if tt.wantGTE == -1 && got != -1 {
			t.Errorf("GetFrameworkPrecedence(%s) = %d, want -1", tt.framework, got)
		}
	}
}

func TestFrameworkToNetStandardTable_Tizen(t *testing.T) {
	tests := []struct {
		version     string
		wantNS      string
		description string
	}{
		{"3.0", "1.6", "Tizen 3.0 supports NetStandard 1.6"},
		{"4.0", "2.0", "Tizen 4.0 supports NetStandard 2.0"},
		{"6.0", "2.1", "Tizen 6.0 supports NetStandard 2.1"},
	}

	tizenMap, ok := FrameworkToNetStandardTable["Tizen"]
	if !ok {
		t.Fatal("Tizen not in FrameworkToNetStandardTable")
	}

	for _, tt := range tests {
		got, ok := tizenMap[tt.version]
		if !ok {
			t.Errorf("Tizen %s not in table", tt.version)
			continue
		}
		if got != tt.wantNS {
			t.Errorf("%s: got %s, want %s", tt.description, got, tt.wantNS)
		}
	}
}

func TestFrameworkToNetStandardTable_UAP(t *testing.T) {
	uapMap, ok := FrameworkToNetStandardTable["UAP"]
	if !ok {
		t.Fatal("UAP not in FrameworkToNetStandardTable")
	}

	// Test specific version
	if got := uapMap["10.0.15064"]; got != "2.0" {
		t.Errorf("UAP 10.0.15064 = %s, want 2.0", got)
	}

	// Test wildcard
	if got := uapMap["*"]; got != "1.4" {
		t.Errorf("UAP * = %s, want 1.4", got)
	}
}

func TestFrameworkToNetStandardTable_NetCoreApp(t *testing.T) {
	tests := []struct {
		version string
		wantNS  string
	}{
		{"1.0", "1.6"},
		{"1.1", "1.7"},
		{"2.0", "2.0"},
		{"3.0", "2.1"},
	}

	coreMap, ok := FrameworkToNetStandardTable[".NETCoreApp"]
	if !ok {
		t.Fatal(".NETCoreApp not in FrameworkToNetStandardTable")
	}

	for _, tt := range tests {
		got, ok := coreMap[tt.version]
		if !ok {
			t.Errorf(".NETCoreApp %s not in table", tt.version)
			continue
		}
		if got != tt.wantNS {
			t.Errorf(".NETCoreApp %s = %s, want %s", tt.version, got, tt.wantNS)
		}
	}
}

func TestFrameworkToNetStandardTable_NetFramework(t *testing.T) {
	tests := []struct {
		version string
		wantNS  string
	}{
		{"4.5", "1.1"},
		{"4.5.1", "1.2"},
		{"4.5.2", "1.2"},
		{"4.6", "1.3"},
		{"4.6.1", "2.0"},
		{"4.6.2", "2.0"},
		{"4.6.3", "2.0"},
		{"4.7", "2.0"},
		{"4.7.1", "2.0"},
		{"4.7.2", "2.0"},
		{"4.8", "2.0"},
		{"4.8.1", "2.0"},
	}

	fxMap, ok := FrameworkToNetStandardTable[".NETFramework"]
	if !ok {
		t.Fatal(".NETFramework not in FrameworkToNetStandardTable")
	}

	for _, tt := range tests {
		got, ok := fxMap[tt.version]
		if !ok {
			t.Errorf(".NETFramework %s not in table", tt.version)
			continue
		}
		if got != tt.wantNS {
			t.Errorf(".NETFramework %s = %s, want %s", tt.version, got, tt.wantNS)
		}
	}
}

func TestFrameworkToNetStandardTable_WindowsPhone(t *testing.T) {
	// Test WindowsPhoneApp
	wpaMap, ok := FrameworkToNetStandardTable["WindowsPhoneApp"]
	if !ok {
		t.Fatal("WindowsPhoneApp not in FrameworkToNetStandardTable")
	}
	if got := wpaMap["8.1"]; got != "1.2" {
		t.Errorf("WindowsPhoneApp 8.1 = %s, want 1.2", got)
	}

	// Test WindowsPhone
	wpMap, ok := FrameworkToNetStandardTable["WindowsPhone"]
	if !ok {
		t.Fatal("WindowsPhone not in FrameworkToNetStandardTable")
	}
	if got := wpMap["8.0"]; got != "1.0" {
		t.Errorf("WindowsPhone 8.0 = %s, want 1.0", got)
	}
	if got := wpMap["8.1"]; got != "1.0" {
		t.Errorf("WindowsPhone 8.1 = %s, want 1.0", got)
	}
}

func TestFrameworkToNetStandardTable_LegacyNetCore(t *testing.T) {
	tests := []struct {
		version string
		wantNS  string
	}{
		{"4.5", "1.1"},
		{"4.5.1", "1.2"},
		{"5.0", "1.4"},
	}

	netcoreMap, ok := FrameworkToNetStandardTable["NetCore"]
	if !ok {
		t.Fatal("NetCore not in FrameworkToNetStandardTable")
	}

	for _, tt := range tests {
		got, ok := netcoreMap[tt.version]
		if !ok {
			t.Errorf("NetCore %s not in table", tt.version)
			continue
		}
		if got != tt.wantNS {
			t.Errorf("NetCore %s = %s, want %s", tt.version, got, tt.wantNS)
		}
	}
}

func TestFrameworkToNetStandardTable_DNXCore(t *testing.T) {
	dnxMap, ok := FrameworkToNetStandardTable["DNXCore"]
	if !ok {
		t.Fatal("DNXCore not in FrameworkToNetStandardTable")
	}

	if got := dnxMap["*"]; got != "1.5" {
		t.Errorf("DNXCore * = %s, want 1.5", got)
	}
}

func TestFrameworkToNetStandardTable_Xamarin_NS21(t *testing.T) {
	// Test all Xamarin frameworks that support NetStandard 2.1
	frameworks := []string{
		"MonoAndroid",
		"MonoMac",
		"MonoTouch",
		"Xamarin.iOS",
		"Xamarin.Mac",
		"Xamarin.TVOS",
		"Xamarin.WatchOS",
	}

	for _, fw := range frameworks {
		fwMap, ok := FrameworkToNetStandardTable[fw]
		if !ok {
			t.Errorf("%s not in FrameworkToNetStandardTable", fw)
			continue
		}
		if got := fwMap["*"]; got != "2.1" {
			t.Errorf("%s * = %s, want 2.1", fw, got)
		}
	}
}

func TestFrameworkToNetStandardTable_Xamarin_NS20(t *testing.T) {
	// Test all Xamarin frameworks that support NetStandard 2.0
	frameworks := []string{
		"Xamarin.PlayStation3",
		"Xamarin.PlayStation4",
		"Xamarin.PlayStationVita",
		"Xamarin.Xbox360",
		"Xamarin.XboxOne",
	}

	for _, fw := range frameworks {
		fwMap, ok := FrameworkToNetStandardTable[fw]
		if !ok {
			t.Errorf("%s not in FrameworkToNetStandardTable", fw)
			continue
		}
		if got := fwMap["*"]; got != "2.0" {
			t.Errorf("%s * = %s, want 2.0", fw, got)
		}
	}
}

func TestFrameworkShortNames(t *testing.T) {
	tests := []struct {
		short string
		full  string
	}{
		{"net", ".NETFramework"},
		{"netframework", ".NETFramework"},
		{"netstandard", ".NETStandard"},
		{"netcoreapp", ".NETCoreApp"},
		{"netcore", "NetCore"},
		{"uap", "UAP"},
		{"tizen", "Tizen"},
		{"monoandroid", "MonoAndroid"},
		{"xamarinios", "Xamarin.iOS"},
		{"xamarinmac", "Xamarin.Mac"},
	}

	for _, tt := range tests {
		got, ok := FrameworkShortNames[tt.short]
		if !ok {
			t.Errorf("Short name %s not in FrameworkShortNames", tt.short)
			continue
		}
		if got != tt.full {
			t.Errorf("FrameworkShortNames[%s] = %s, want %s", tt.short, got, tt.full)
		}
	}
}
