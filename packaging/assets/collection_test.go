package assets

import (
	"slices"
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
)

func TestSelectionCriteriaBuilder(t *testing.T) {
	tests := []struct {
		name     string
		build    func(b *SelectionCriteriaBuilder) *SelectionCriteria
		wantLen  int
		validate func(*testing.T, *SelectionCriteria)
	}{
		{
			name: "single entry",
			build: func(b *SelectionCriteriaBuilder) *SelectionCriteria {
				return b.Add("tfm", "net6.0").Build()
			},
			wantLen: 1,
			validate: func(t *testing.T, c *SelectionCriteria) {
				if len(c.Entries) != 1 {
					t.Fatalf("want 1 entry, got %d", len(c.Entries))
				}
				if c.Entries[0].Properties["tfm"] != "net6.0" {
					t.Errorf("want tfm=net6.0, got %v", c.Entries[0].Properties["tfm"])
				}
			},
		},
		{
			name: "multiple entries with NextEntry",
			build: func(b *SelectionCriteriaBuilder) *SelectionCriteria {
				return b.Add("tfm", "net6.0").Add("rid", "win-x64").
					NextEntry().
					Add("tfm", "net6.0").Add("rid", nil).
					Build()
			},
			wantLen: 2,
			validate: func(t *testing.T, c *SelectionCriteria) {
				if len(c.Entries) != 2 {
					t.Fatalf("want 2 entries, got %d", len(c.Entries))
				}
				// First entry should have RID
				if c.Entries[0].Properties["rid"] != "win-x64" {
					t.Errorf("first entry: want rid=win-x64, got %v", c.Entries[0].Properties["rid"])
				}
				// Second entry should have nil RID
				if c.Entries[1].Properties["rid"] != nil {
					t.Errorf("second entry: want rid=nil, got %v", c.Entries[1].Properties["rid"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSelectionCriteriaBuilder(make(map[string]*PropertyDefinition))
			criteria := tt.build(builder)

			if len(criteria.Entries) != tt.wantLen {
				t.Errorf("want %d entries, got %d", tt.wantLen, len(criteria.Entries))
			}

			if tt.validate != nil {
				tt.validate(t, criteria)
			}
		})
	}
}

func TestForFramework(t *testing.T) {
	fw, _ := frameworks.ParseFramework("net6.0")
	properties := make(map[string]*PropertyDefinition)

	criteria := ForFramework(fw, properties)

	if len(criteria.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(criteria.Entries))
	}

	entry := criteria.Entries[0]
	if entry.Properties["tfm"] != fw {
		t.Errorf("want tfm=%v, got %v", fw, entry.Properties["tfm"])
	}
	if entry.Properties["rid"] != nil {
		t.Errorf("want rid=nil, got %v", entry.Properties["rid"])
	}
}

func TestForFrameworkAndRuntime(t *testing.T) {
	fw, _ := frameworks.ParseFramework("net6.0")
	properties := make(map[string]*PropertyDefinition)

	t.Run("with RID", func(t *testing.T) {
		criteria := ForFrameworkAndRuntime(fw, "win-x64", properties)

		if len(criteria.Entries) != 2 {
			t.Fatalf("want 2 entries (RID + fallback), got %d", len(criteria.Entries))
		}

		// First entry: RID-specific
		if criteria.Entries[0].Properties["rid"] != "win-x64" {
			t.Errorf("first entry: want rid=win-x64, got %v", criteria.Entries[0].Properties["rid"])
		}

		// Second entry: RID-agnostic fallback
		if criteria.Entries[1].Properties["rid"] != nil {
			t.Errorf("second entry: want rid=nil, got %v", criteria.Entries[1].Properties["rid"])
		}
	})

	t.Run("without RID", func(t *testing.T) {
		criteria := ForFrameworkAndRuntime(fw, "", properties)

		if len(criteria.Entries) != 1 {
			t.Fatalf("want 1 entry (fallback only), got %d", len(criteria.Entries))
		}

		// Should only have RID-agnostic entry
		if criteria.Entries[0].Properties["rid"] != nil {
			t.Errorf("want rid=nil, got %v", criteria.Entries[0].Properties["rid"])
		}
	})
}

func TestContentItemCollection_PopulateItemGroups(t *testing.T) {
	conventions := NewManagedCodeConventions()

	tests := []struct {
		name       string
		paths      []string
		patternSet *PatternSet
		wantGroups int
	}{
		{
			name: "lib paths",
			paths: []string{
				"lib/net6.0/MyLib.dll",
				"lib/net7.0/MyLib.dll",
				"lib/netstandard2.1/StandardLib.dll",
			},
			patternSet: conventions.RuntimeAssemblies,
			wantGroups: 3, // 3 different frameworks
		},
		{
			name: "mixed lib and ref",
			paths: []string{
				"lib/net6.0/MyLib.dll",
				"ref/net6.0/MyLib.dll",
				"content/readme.txt",
			},
			patternSet: conventions.RuntimeAssemblies,
			wantGroups: 1, // only lib/ matches for runtime assemblies
		},
		{
			name:       "no matches",
			paths:      []string{"content/readme.txt", "build/project.props"},
			patternSet: conventions.RuntimeAssemblies,
			wantGroups: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collection := NewContentItemCollection(tt.paths)
			groups := collection.PopulateItemGroups(tt.patternSet)

			if len(groups) != tt.wantGroups {
				t.Errorf("want %d groups, got %d", tt.wantGroups, len(groups))
			}
		})
	}
}

func TestContentItemCollection_FindBestItemGroup(t *testing.T) {
	conventions := NewManagedCodeConventions()

	tests := []struct {
		name        string
		paths       []string
		framework   string
		patternSets []*PatternSet
		wantGroup   bool
		wantItems   int
	}{
		{
			name: "find net6.0 runtime assemblies",
			paths: []string{
				"lib/net6.0/MyLib.dll",
				"lib/net7.0/MyLib.dll",
			},
			framework:   "net6.0",
			patternSets: []*PatternSet{conventions.RuntimeAssemblies},
			wantGroup:   true,
			wantItems:   1,
		},
		{
			name: "find compatible netstandard2.1 for net6.0",
			paths: []string{
				"lib/netstandard2.1/StandardLib.dll",
			},
			framework:   "net6.0",
			patternSets: []*PatternSet{conventions.RuntimeAssemblies},
			wantGroup:   true,
			wantItems:   1,
		},
		{
			name: "ref takes precedence over lib for compile",
			paths: []string{
				"lib/net6.0/MyLib.dll",
				"ref/net6.0/MyLib.dll",
			},
			framework: "net6.0",
			patternSets: []*PatternSet{
				conventions.CompileRefAssemblies,
				conventions.CompileLibAssemblies,
			},
			wantGroup: true,
			wantItems: 1, // Should pick ref/, not lib/
		},
		{
			name: "no matching framework",
			paths: []string{
				"lib/net7.0/MyLib.dll",
			},
			framework:   "net45",
			patternSets: []*PatternSet{conventions.RuntimeAssemblies},
			wantGroup:   false,
		},
		{
			name: "selects nearest framework among multiple compatible",
			paths: []string{
				"lib/net45/MyLib.dll",
				"lib/net46/MyLib.dll",
				"lib/net47/MyLib.dll",
			},
			framework:   "net48",
			patternSets: []*PatternSet{conventions.RuntimeAssemblies},
			wantGroup:   true,
			wantItems:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collection := NewContentItemCollection(tt.paths)
			fw, err := frameworks.ParseFramework(tt.framework)
			if err != nil {
				t.Fatalf("failed to parse framework %s: %v", tt.framework, err)
			}

			criteria := ForFramework(fw, conventions.Properties)
			group := collection.FindBestItemGroup(criteria, tt.patternSets...)

			if tt.wantGroup {
				if group == nil {
					t.Fatal("want group, got nil")
				}
				if len(group.Items) != tt.wantItems {
					t.Errorf("want %d items, got %d", tt.wantItems, len(group.Items))
				}
			} else if group != nil {
				t.Errorf("want nil group, got %v", group)
			}
		})
	}
}

func TestGetLibItems(t *testing.T) {
	conventions := NewManagedCodeConventions()

	tests := []struct {
		name      string
		files     []string
		framework string
		wantCount int
		wantPaths []string
	}{
		{
			name: "select net6.0 runtime assemblies",
			files: []string{
				"lib/net6.0/MyLib.dll",
				"lib/net6.0/MyLib.xml",
				"ref/net6.0/RefLib.dll",
			},
			framework: "net6.0",
			wantCount: 1,
			wantPaths: []string{"lib/net6.0/MyLib.dll"},
		},
		{
			name: "filter to DLL/EXE only",
			files: []string{
				"lib/net6.0/MyLib.dll",
				"lib/net6.0/MyLib.exe",
				"lib/net6.0/MyLib.xml",
				"lib/net6.0/MyLib.pdb",
			},
			framework: "net6.0",
			wantCount: 2,
		},
		{
			name: "no matching framework",
			files: []string{
				"lib/net7.0/MyLib.dll",
			},
			framework: "net45",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := frameworks.ParseFramework(tt.framework)
			if err != nil {
				t.Fatalf("failed to parse framework: %v", err)
			}

			items := GetLibItems(tt.files, fw, conventions)

			if len(items) != tt.wantCount {
				t.Errorf("want %d items, got %d", tt.wantCount, len(items))
			}

			if tt.wantPaths != nil {
				for _, wantPath := range tt.wantPaths {
					if !slices.Contains(items, wantPath) {
						t.Errorf("want path %s, not found in %v", wantPath, items)
					}
				}
			}
		})
	}
}

func TestGetRefItems(t *testing.T) {
	conventions := NewManagedCodeConventions()

	tests := []struct {
		name      string
		files     []string
		framework string
		wantCount int
		wantPaths []string
	}{
		{
			name: "ref takes precedence over lib",
			files: []string{
				"lib/net6.0/MyLib.dll",
				"ref/net6.0/MyLib.dll",
			},
			framework: "net6.0",
			wantCount: 1,
			wantPaths: []string{"ref/net6.0/MyLib.dll"},
		},
		{
			name: "fallback to lib when no ref",
			files: []string{
				"lib/net6.0/MyLib.dll",
			},
			framework: "net6.0",
			wantCount: 1,
			wantPaths: []string{"lib/net6.0/MyLib.dll"},
		},
		{
			name: "only ref assemblies",
			files: []string{
				"ref/net6.0/MyLib.dll",
				"ref/net6.0/AnotherLib.dll",
			},
			framework: "net6.0",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := frameworks.ParseFramework(tt.framework)
			if err != nil {
				t.Fatalf("failed to parse framework: %v", err)
			}

			items := GetRefItems(tt.files, fw, conventions)

			if len(items) != tt.wantCount {
				t.Errorf("want %d items, got %d", tt.wantCount, len(items))
			}

			if tt.wantPaths != nil {
				for _, wantPath := range tt.wantPaths {
					if !slices.Contains(items, wantPath) {
						t.Errorf("want path %s, not found in %v", wantPath, items)
					}
				}
			}
		})
	}
}

func TestFilterToDllExe(t *testing.T) {
	tests := []struct {
		name      string
		paths     []string
		wantCount int
	}{
		{
			name:      "filter assemblies",
			paths:     []string{"lib.dll", "lib.exe", "lib.winmd", "lib.xml", "lib.pdb"},
			wantCount: 3,
		},
		{
			name:      "case insensitive",
			paths:     []string{"lib.DLL", "lib.EXE", "lib.WINMD"},
			wantCount: 3,
		},
		{
			name:      "no assemblies",
			paths:     []string{"readme.txt", "icon.png"},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterToDllExe(tt.paths)
			if len(filtered) != tt.wantCount {
				t.Errorf("want %d filtered, got %d", tt.wantCount, len(filtered))
			}
		})
	}
}
