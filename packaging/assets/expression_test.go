package assets

import (
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
)

func TestPatternExpression_Initialize(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		expectedSegs   int
		firstSegType   string
		firstSegValue  string
		secondSegType  string
		secondSegValue string
	}{
		{
			name:           "Simple literal",
			pattern:        "lib/",
			expectedSegs:   1,
			firstSegType:   "literal",
			firstSegValue:  "lib/",
			secondSegType:  "",
			secondSegValue: "",
		},
		{
			name:           "Literal and token",
			pattern:        "lib/{tfm}",
			expectedSegs:   2,
			firstSegType:   "literal",
			firstSegValue:  "lib/",
			secondSegType:  "token",
			secondSegValue: "tfm",
		},
		{
			name:           "Multiple tokens",
			pattern:        "lib/{tfm}/{assembly}",
			expectedSegs:   4,
			firstSegType:   "literal",
			firstSegValue:  "lib/",
			secondSegType:  "token",
			secondSegValue: "tfm",
		},
		{
			name:           "Optional token",
			pattern:        "lib/{any?}",
			expectedSegs:   2,
			firstSegType:   "literal",
			firstSegValue:  "lib/",
			secondSegType:  "token",
			secondSegValue: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patternDef := &PatternDefinition{Pattern: tt.pattern}
			expr := NewPatternExpression(patternDef)

			if len(expr.segments) != tt.expectedSegs {
				t.Errorf("Expected %d segments, got %d", tt.expectedSegs, len(expr.segments))
			}

			if len(expr.segments) > 0 {
				switch seg := expr.segments[0].(type) {
				case *LiteralSegment:
					if tt.firstSegType != "literal" {
						t.Errorf("Expected first segment to be %s, got literal", tt.firstSegType)
					}
					if seg.text != tt.firstSegValue {
						t.Errorf("Expected first literal segment value %s, got %s", tt.firstSegValue, seg.text)
					}
				case *TokenSegment:
					if tt.firstSegType != "token" {
						t.Errorf("Expected first segment to be %s, got token", tt.firstSegType)
					}
					if seg.name != tt.firstSegValue {
						t.Errorf("Expected first token segment name %s, got %s", tt.firstSegValue, seg.name)
					}
				}
			}
		})
	}
}

func TestPatternExpression_Match(t *testing.T) {
	net60, _ := frameworks.ParseFramework("net6.0")

	conventions := NewManagedCodeConventions()

	tests := []struct {
		name         string
		pattern      string
		path         string
		wantMatch    bool
		wantTfm      *frameworks.NuGetFramework
		wantAssembly string
	}{
		{
			name:         "Match lib/net6.0/MyLib.dll",
			pattern:      "lib/{tfm}/{assembly}",
			path:         "lib/net6.0/MyLib.dll",
			wantMatch:    true,
			wantTfm:      net60,
			wantAssembly: "MyLib.dll",
		},
		{
			name:      "No match - wrong prefix",
			pattern:   "lib/{tfm}/{assembly}",
			path:      "ref/net6.0/MyLib.dll",
			wantMatch: false,
		},
		{
			name:      "No match - missing segment",
			pattern:   "lib/{tfm}/{assembly}",
			path:      "lib/net6.0/",
			wantMatch: false,
		},
		{
			name:      "Match with optional token - no match with trailing slash",
			pattern:   "lib/{tfm}/{any?}",
			path:      "lib/net6.0/",
			wantMatch: false, // Trailing slash means the pattern doesn't fully consume the path
		},
		{
			name:      "Case insensitive literal",
			pattern:   "lib/{tfm}/{assembly}",
			path:      "LIB/net6.0/MyLib.dll",
			wantMatch: true,
			wantTfm:   net60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patternDef := &PatternDefinition{
				Pattern: tt.pattern,
				Table:   DotnetAnyTable,
			}
			expr := NewPatternExpression(patternDef)

			item := expr.Match(tt.path, conventions.Properties)

			if tt.wantMatch && item == nil {
				t.Fatalf("Expected match, got nil")
			}

			if !tt.wantMatch && item != nil {
				t.Fatalf("Expected no match, got %v", item)
			}

			if tt.wantMatch {
				if item.Path != tt.path {
					t.Errorf("Expected path %s, got %s", tt.path, item.Path)
				}

				if tt.wantTfm != nil {
					tfm, ok := item.Properties["tfm"].(*frameworks.NuGetFramework)
					if !ok {
						t.Errorf("Expected tfm to be *frameworks.NuGetFramework, got %T", item.Properties["tfm"])
					} else if !tfm.Equals(tt.wantTfm) {
						t.Errorf("Expected tfm %v, got %v", tt.wantTfm, tfm)
					}
				}

				if tt.wantAssembly != "" {
					assembly, ok := item.Properties["assembly"].(string)
					if !ok {
						t.Errorf("Expected assembly to be string, got %T", item.Properties["assembly"])
					} else if assembly != tt.wantAssembly {
						t.Errorf("Expected assembly %s, got %s", tt.wantAssembly, assembly)
					}
				}
			}
		})
	}
}

func TestPatternExpression_Defaults(t *testing.T) {
	net45, _ := frameworks.ParseFramework("net45")

	conventions := NewManagedCodeConventions()

	patternDef := &PatternDefinition{
		Pattern: "lib/{assembly}",
		Table:   DotnetAnyTable,
		Defaults: map[string]interface{}{
			"tfm": net45,
		},
	}

	expr := NewPatternExpression(patternDef)
	item := expr.Match("lib/MyLib.dll", conventions.Properties)

	if item == nil {
		t.Fatal("Expected match, got nil")
	}

	tfm, ok := item.Properties["tfm"].(*frameworks.NuGetFramework)
	if !ok {
		t.Fatalf("Expected tfm to be *frameworks.NuGetFramework, got %T", item.Properties["tfm"])
	}

	if !tfm.Equals(net45) {
		t.Errorf("Expected default tfm net45, got %v", tfm)
	}

	assembly, ok := item.Properties["assembly"].(string)
	if !ok {
		t.Fatalf("Expected assembly to be string, got %T", item.Properties["assembly"])
	}

	if assembly != "MyLib.dll" {
		t.Errorf("Expected assembly MyLib.dll, got %s", assembly)
	}
}

func TestLiteralSegment_TryMatch(t *testing.T) {
	tests := []struct {
		name      string
		literal   string
		path      string
		start     int
		wantEnd   int
		wantMatch bool
	}{
		{
			name:      "Exact match",
			literal:   "lib/",
			path:      "lib/net6.0/",
			start:     0,
			wantEnd:   4,
			wantMatch: true,
		},
		{
			name:      "Case insensitive match",
			literal:   "lib/",
			path:      "LIB/net6.0/",
			start:     0,
			wantEnd:   4,
			wantMatch: true,
		},
		{
			name:      "No match - different text",
			literal:   "lib/",
			path:      "ref/net6.0/",
			start:     0,
			wantEnd:   0,
			wantMatch: false,
		},
		{
			name:      "No match - insufficient length",
			literal:   "lib/",
			path:      "li",
			start:     0,
			wantEnd:   0,
			wantMatch: false,
		},
		{
			name:      "Match at offset",
			literal:   "net6.0/",
			path:      "lib/net6.0/MyLib.dll",
			start:     4,
			wantEnd:   11,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := &LiteralSegment{text: tt.literal}
			var item *ContentItem

			gotEnd, gotMatch := seg.TryMatch(&item, tt.path, nil, tt.start)

			if gotMatch != tt.wantMatch {
				t.Errorf("TryMatch() match = %v, want %v", gotMatch, tt.wantMatch)
			}

			if gotMatch && gotEnd != tt.wantEnd {
				t.Errorf("TryMatch() end = %d, want %d", gotEnd, tt.wantEnd)
			}
		})
	}
}

func TestTokenSegment_TryMatch(t *testing.T) {
	conventions := NewManagedCodeConventions()

	tests := []struct {
		name      string
		tokenName string
		delimiter byte
		matchOnly bool
		path      string
		start     int
		wantEnd   int
		wantMatch bool
		wantValue interface{}
	}{
		{
			name:      "Match assembly token",
			tokenName: "assembly",
			delimiter: 0,
			matchOnly: false,
			path:      "MyLib.dll",
			start:     0,
			wantEnd:   9,
			wantMatch: true,
			wantValue: "MyLib.dll",
		},
		{
			name:      "Match with delimiter",
			tokenName: "tfm",
			delimiter: '/',
			matchOnly: false,
			path:      "net6.0/MyLib.dll",
			start:     0,
			wantEnd:   6,
			wantMatch: true,
		},
		{
			name:      "Match only mode",
			tokenName: "assembly",
			delimiter: 0,
			matchOnly: true,
			path:      "MyLib.dll",
			start:     0,
			wantEnd:   9,
			wantMatch: true,
			wantValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := &TokenSegment{
				name:      tt.tokenName,
				delimiter: tt.delimiter,
				matchOnly: tt.matchOnly,
				table:     DotnetAnyTable,
			}
			var item *ContentItem

			gotEnd, gotMatch := seg.TryMatch(&item, tt.path, conventions.Properties, tt.start)

			if gotMatch != tt.wantMatch {
				t.Errorf("TryMatch() match = %v, want %v", gotMatch, tt.wantMatch)
			}

			if gotMatch && gotEnd != tt.wantEnd {
				t.Errorf("TryMatch() end = %d, want %d", gotEnd, tt.wantEnd)
			}

			if tt.wantValue != nil && item != nil {
				if gotValue := item.Properties[tt.tokenName]; gotValue != tt.wantValue {
					t.Errorf("TryMatch() value = %v, want %v", gotValue, tt.wantValue)
				}
			}
		})
	}
}

func TestPatternSet_Creation(t *testing.T) {
	conventions := NewManagedCodeConventions()

	groupPatterns := []*PatternDefinition{
		{Pattern: "lib/{tfm}/{any?}", Table: DotnetAnyTable},
	}

	pathPatterns := []*PatternDefinition{
		{Pattern: "lib/{tfm}/{assembly}", Table: DotnetAnyTable},
	}

	ps := NewPatternSet(conventions.Properties, groupPatterns, pathPatterns)

	if len(ps.GroupExpressions) != len(groupPatterns) {
		t.Errorf("Expected %d group expressions, got %d", len(groupPatterns), len(ps.GroupExpressions))
	}

	if len(ps.PathExpressions) != len(pathPatterns) {
		t.Errorf("Expected %d path expressions, got %d", len(pathPatterns), len(ps.PathExpressions))
	}

	// Test that expressions were compiled
	if ps.GroupExpressions[0] == nil {
		t.Error("Expected group expression to be compiled")
	}

	if ps.PathExpressions[0] == nil {
		t.Error("Expected path expression to be compiled")
	}
}
