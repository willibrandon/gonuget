package assets

import (
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
)

func TestPatternTable(t *testing.T) {
	tests := []struct {
		name         string
		entries      []PatternTableEntry
		propertyName string
		tokenName    string
		wantValue    any
		wantOK       bool
	}{
		{
			name: "DotnetAnyTable lookup",
			entries: []PatternTableEntry{
				{PropertyName: "tfm", Name: "any", Value: frameworks.CommonFrameworks.DotNet},
			},
			propertyName: "tfm",
			tokenName:    "any",
			wantValue:    frameworks.CommonFrameworks.DotNet,
			wantOK:       true,
		},
		{
			name: "Missing property",
			entries: []PatternTableEntry{
				{PropertyName: "tfm", Name: "any", Value: frameworks.CommonFrameworks.DotNet},
			},
			propertyName: "rid",
			tokenName:    "any",
			wantValue:    nil,
			wantOK:       false,
		},
		{
			name: "Missing token",
			entries: []PatternTableEntry{
				{PropertyName: "tfm", Name: "any", Value: frameworks.CommonFrameworks.DotNet},
			},
			propertyName: "tfm",
			tokenName:    "missing",
			wantValue:    nil,
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewPatternTable(tt.entries)
			gotValue, gotOK := table.TryLookup(tt.propertyName, tt.tokenName)

			if gotOK != tt.wantOK {
				t.Errorf("TryLookup() ok = %v, want %v", gotOK, tt.wantOK)
			}

			if tt.wantOK && gotValue != tt.wantValue {
				t.Errorf("TryLookup() value = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestPropertyDefinition_TryLookup(t *testing.T) {
	tests := []struct {
		name      string
		propDef   *PropertyDefinition
		value     string
		table     *PatternTable
		matchOnly bool
		wantValue any
		wantOK    bool
	}{
		{
			name: "File extension match - .dll",
			propDef: &PropertyDefinition{
				Name:           "assembly",
				FileExtensions: []string{".dll", ".exe"},
			},
			value:     "MyLib.dll",
			matchOnly: false,
			wantValue: "MyLib.dll",
			wantOK:    true,
		},
		{
			name: "File extension match - match only",
			propDef: &PropertyDefinition{
				Name:           "assembly",
				FileExtensions: []string{".dll", ".exe"},
			},
			value:     "MyLib.dll",
			matchOnly: true,
			wantValue: nil,
			wantOK:    true,
		},
		{
			name: "File extension no match",
			propDef: &PropertyDefinition{
				Name:           "assembly",
				FileExtensions: []string{".dll", ".exe"},
			},
			value:     "MyLib.txt",
			matchOnly: false,
			wantValue: nil,
			wantOK:    false,
		},
		{
			name: "Parser match",
			propDef: &PropertyDefinition{
				Name: "test",
				Parser: func(value string, table *PatternTable, matchOnly bool) any {
					if value == "valid" {
						if matchOnly {
							return value
						}
						return "parsed:" + value
					}
					return nil
				},
			},
			value:     "valid",
			matchOnly: false,
			wantValue: "parsed:valid",
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOK := tt.propDef.TryLookup(tt.value, tt.table, tt.matchOnly)

			if gotOK != tt.wantOK {
				t.Errorf("TryLookup() ok = %v, want %v", gotOK, tt.wantOK)
			}

			if tt.wantOK && tt.wantValue != nil && gotValue != tt.wantValue {
				t.Errorf("TryLookup() value = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestContentItem_Add(t *testing.T) {
	item := &ContentItem{
		Path: "lib/net6.0/MyLib.dll",
	}

	// Add first property
	item.Add("tfm", "net6.0")
	if item.Properties["tfm"] != "net6.0" {
		t.Errorf("Expected tfm=net6.0, got %v", item.Properties["tfm"])
	}

	// Try to overwrite - should not change
	item.Add("tfm", "net7.0")
	if item.Properties["tfm"] != "net6.0" {
		t.Errorf("Expected tfm=net6.0 (unchanged), got %v", item.Properties["tfm"])
	}

	// Add different property
	item.Add("assembly", "MyLib.dll")
	if item.Properties["assembly"] != "MyLib.dll" {
		t.Errorf("Expected assembly=MyLib.dll, got %v", item.Properties["assembly"])
	}
}

func TestManagedCodeConventions(t *testing.T) {
	conventions := NewManagedCodeConventions()

	t.Run("Properties defined", func(t *testing.T) {
		expectedProps := []string{"assembly", "msbuild", "satelliteAssembly", "locale", "any", "tfm", "rid", "codeLanguage"}
		for _, prop := range expectedProps {
			if _, ok := conventions.Properties[prop]; !ok {
				t.Errorf("Expected property %s to be defined", prop)
			}
		}
	})

	t.Run("RuntimeAssemblies pattern set", func(t *testing.T) {
		if conventions.RuntimeAssemblies == nil {
			t.Fatal("RuntimeAssemblies pattern set is nil")
		}
		if len(conventions.RuntimeAssemblies.GroupPatterns) == 0 {
			t.Error("Expected group patterns for RuntimeAssemblies")
		}
		if len(conventions.RuntimeAssemblies.PathPatterns) == 0 {
			t.Error("Expected path patterns for RuntimeAssemblies")
		}
	})

	t.Run("CompileRefAssemblies pattern set", func(t *testing.T) {
		if conventions.CompileRefAssemblies == nil {
			t.Fatal("CompileRefAssemblies pattern set is nil")
		}
		if len(conventions.CompileRefAssemblies.PathPatterns) == 0 {
			t.Error("Expected path patterns for CompileRefAssemblies")
		}
	})
}

func TestTfmParser(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		matchOnly bool
		wantNil   bool
	}{
		{
			name:      "Valid framework - net6.0",
			value:     "net6.0",
			matchOnly: false,
			wantNil:   false,
		},
		{
			name:      "Valid framework - netstandard2.1",
			value:     "netstandard2.1",
			matchOnly: false,
			wantNil:   false,
		},
		{
			name:      "Invalid framework",
			value:     "invalid",
			matchOnly: false,
			wantNil:   true,
		},
		{
			name:      "Match only - valid",
			value:     "net6.0",
			matchOnly: true,
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tfmParser(tt.value, nil, tt.matchOnly)

			if tt.wantNil && result != nil {
				t.Errorf("Expected nil result, got %v", result)
			}

			if !tt.wantNil && result == nil {
				t.Errorf("Expected non-nil result, got nil")
			}

			if !tt.matchOnly && !tt.wantNil {
				if _, ok := result.(*frameworks.NuGetFramework); !ok {
					t.Errorf("Expected *frameworks.NuGetFramework, got %T", result)
				}
			}
		})
	}
}

func TestTfmCompatibilityTest(t *testing.T) {
	net60, _ := frameworks.ParseFramework("net6.0")
	net70, _ := frameworks.ParseFramework("net7.0")
	netstandard21, _ := frameworks.ParseFramework("netstandard2.1")

	tests := []struct {
		name       string
		criterion  any
		available  any
		wantCompat bool
	}{
		{
			name:       "net7.0 package not compatible with older net6.0 project",
			criterion:  net60,
			available:  net70,
			wantCompat: false,
		},
		{
			name:       "netstandard2.1 package available for net6.0 project",
			criterion:  net60,
			available:  netstandard21,
			wantCompat: true,
		},
		{
			name:       "AnyFramework always compatible",
			criterion:  net60,
			available:  &frameworks.AnyFramework,
			wantCompat: true,
		},
		{
			name:       "Invalid types return false",
			criterion:  "not a framework",
			available:  net60,
			wantCompat: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCompat := tfmCompatibilityTest(tt.criterion, tt.available)
			if gotCompat != tt.wantCompat {
				t.Errorf("tfmCompatibilityTest() = %v, want %v", gotCompat, tt.wantCompat)
			}
		})
	}
}

func TestTfmCompareTest(t *testing.T) {
	net60, _ := frameworks.ParseFramework("net6.0")
	net70, _ := frameworks.ParseFramework("net7.0")
	netstandard21, _ := frameworks.ParseFramework("netstandard2.1")

	tests := []struct {
		name       string
		criterion  any
		available1 any
		available2 any
		wantResult int // -1 if available1 is better, 1 if available2 is better, 0 if equal
	}{
		{
			name:       "net7.0 closer to net7.0 than net6.0",
			criterion:  net70,
			available1: net70,
			available2: net60,
			wantResult: -1,
		},
		{
			name:       "net6.0 closer to net6.0 than netstandard2.1",
			criterion:  net60,
			available1: net60,
			available2: netstandard21,
			wantResult: -1,
		},
		{
			name:       "available2 is closer",
			criterion:  net60,
			available1: netstandard21,
			available2: net60,
			wantResult: 1,
		},
		{
			name:       "Invalid criterion returns 0",
			criterion:  "not a framework",
			available1: net60,
			available2: net70,
			wantResult: 0,
		},
		{
			name:       "Invalid available returns 0",
			criterion:  net60,
			available1: "not a framework",
			available2: net70,
			wantResult: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := tfmCompareTest(tt.criterion, tt.available1, tt.available2)
			if gotResult != tt.wantResult {
				t.Errorf("tfmCompareTest() = %d, want %d", gotResult, tt.wantResult)
			}
		})
	}
}

func TestPropertyParsers(t *testing.T) {
	t.Run("allowEmptyFolderParser", func(t *testing.T) {
		result := allowEmptyFolderParser("test", nil, false)
		if result != "test" {
			t.Errorf("Expected 'test', got %v", result)
		}

		resultMatch := allowEmptyFolderParser("test", nil, true)
		if resultMatch != "test" {
			t.Errorf("Expected 'test' in match mode, got %v", resultMatch)
		}
	})

	t.Run("identityParser", func(t *testing.T) {
		result := identityParser("value", nil, false)
		if result != "value" {
			t.Errorf("Expected 'value', got %v", result)
		}

		resultMatch := identityParser("value", nil, true)
		if resultMatch != "value" {
			t.Errorf("Expected 'value' in match mode, got %v", resultMatch)
		}
	})

	t.Run("localeParser", func(t *testing.T) {
		result := localeParser("en-US", nil, false)
		if result != "en-US" {
			t.Errorf("Expected 'en-US', got %v", result)
		}

		resultMatch := localeParser("en-US", nil, true)
		if resultMatch != "en-US" {
			t.Errorf("Expected 'en-US' in match mode, got %v", resultMatch)
		}
	})

	t.Run("codeLanguageParser", func(t *testing.T) {
		result := codeLanguageParser("CS", nil, false)
		if result != "cs" {
			t.Errorf("Expected 'cs', got %v", result)
		}

		resultMatch := codeLanguageParser("VB", nil, true)
		if resultMatch != "VB" {
			t.Errorf("Expected 'VB' in match mode, got %v", resultMatch)
		}
	})

	t.Run("ridCompatibilityTest", func(t *testing.T) {
		// Exact match
		if !ridCompatibilityTest("win-x64", "win-x64") {
			t.Error("Expected exact RID match to be compatible")
		}

		// No match
		if ridCompatibilityTest("win-x64", "linux-x64") {
			t.Error("Expected different RIDs to be incompatible")
		}
	})
}

func TestPropertyDefinitionMethods(t *testing.T) {
	conventions := NewManagedCodeConventions()

	t.Run("IsCriteriaSatisfied", func(t *testing.T) {
		propDef := conventions.Properties["tfm"]
		net60, _ := frameworks.ParseFramework("net6.0")
		netstandard21, _ := frameworks.ParseFramework("netstandard2.1")

		// Compatible frameworks
		if !propDef.IsCriteriaSatisfied(net60, netstandard21) {
			t.Error("Expected netstandard2.1 to satisfy net6.0 criteria")
		}

		// Incompatible frameworks
		net70, _ := frameworks.ParseFramework("net7.0")
		if propDef.IsCriteriaSatisfied(net60, net70) {
			t.Error("Expected net7.0 to not satisfy net6.0 criteria")
		}

		// Test property without CompatibilityTest (assembly property) - always returns false
		assemblyProp := conventions.Properties["assembly"]
		if assemblyProp.IsCriteriaSatisfied("test1", "test2") {
			t.Error("Expected false for property without CompatibilityTest")
		}
	})

	t.Run("Compare", func(t *testing.T) {
		propDef := conventions.Properties["tfm"]
		net60, _ := frameworks.ParseFramework("net6.0")
		net70, _ := frameworks.ParseFramework("net7.0")
		netstandard21, _ := frameworks.ParseFramework("netstandard2.1")

		// net7.0 is closer to net7.0 than net6.0
		result := propDef.Compare(net70, net70, net60)
		if result != -1 {
			t.Errorf("Expected -1 (net7.0 closer), got %d", result)
		}

		// net6.0 is closer to net6.0 than netstandard2.1
		result = propDef.Compare(net60, net60, netstandard21)
		if result != -1 {
			t.Errorf("Expected -1 (net6.0 closer), got %d", result)
		}

		// Test reverse comparison (candidate2 better)
		result = propDef.Compare(net60, netstandard21, net60)
		if result != 1 {
			t.Errorf("Expected 1 (net6.0 closer than netstandard2.1), got %d", result)
		}

		// Test tie - both equally compatible, uses CompareTest
		net50, _ := frameworks.ParseFramework("net5.0")
		result = propDef.Compare(net60, net50, netstandard21)
		// Both are compatible with net6.0, so it should use tfmCompareTest tie breaker
		if result == 0 {
			t.Error("Expected non-zero result from CompareTest tie breaker")
		}

		// Test property without CompareTest (assembly property)
		assemblyProp := conventions.Properties["assembly"]
		result = assemblyProp.Compare("test", "value1", "value2")
		if result != 0 {
			t.Errorf("Expected 0 for property without CompareTest, got %d", result)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("containsSlash", func(t *testing.T) {
		if !containsSlash("path/to/file") {
			t.Error("Expected true for path with slash")
		}
		if containsSlash("filename") {
			t.Error("Expected false for path without slash")
		}
	})

	t.Run("endsWithIgnoreCase", func(t *testing.T) {
		if !endsWithIgnoreCase("MyLib.DLL", ".dll") {
			t.Error("Expected true for .DLL ending with .dll")
		}
		if !endsWithIgnoreCase("MyLib.dll", ".DLL") {
			t.Error("Expected true for .dll ending with .DLL")
		}
		if endsWithIgnoreCase("MyLib.exe", ".dll") {
			t.Error("Expected false for .exe ending with .dll")
		}
		if endsWithIgnoreCase("short", ".toolong") {
			t.Error("Expected false when suffix is longer than text")
		}
	})
}
