package assets

import (
	"slices"
	"testing"
)

func TestParseRID(t *testing.T) {
	tests := []struct {
		name        string
		rid         string
		wantOS      string
		wantVersion string
		wantArch    string
		wantQuals   []string
		wantErr     bool
	}{
		{
			name:     "simple os-arch",
			rid:      "win-x64",
			wantOS:   "win",
			wantArch: "x64",
		},
		{
			name:      "os-arch-qualifier",
			rid:       "linux-x64-musl",
			wantOS:    "linux",
			wantArch:  "x64",
			wantQuals: []string{"musl"},
		},
		{
			name:        "os.version-arch",
			rid:         "osx.10.12-x64",
			wantOS:      "osx",
			wantVersion: "10.12",
			wantArch:    "x64",
		},
		{
			name:        "ubuntu version",
			rid:         "ubuntu.22.04-x64",
			wantOS:      "ubuntu",
			wantVersion: "22.04",
			wantArch:    "x64",
		},
		{
			name:   "os only",
			rid:    "linux",
			wantOS: "linux",
		},
		{
			name:    "empty",
			rid:     "",
			wantErr: true,
		},
		{
			name:     "win10-x64",
			rid:      "win10-x64",
			wantOS:   "win10",
			wantArch: "x64",
		},
		{
			name:     "arm64 architecture",
			rid:      "osx-arm64",
			wantOS:   "osx",
			wantArch: "arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRID(tt.rid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got.RID != tt.rid {
				t.Errorf("RID = %v, want %v", got.RID, tt.rid)
			}
			if got.OS != tt.wantOS {
				t.Errorf("OS = %v, want %v", got.OS, tt.wantOS)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", got.Version, tt.wantVersion)
			}
			if got.Architecture != tt.wantArch {
				t.Errorf("Architecture = %v, want %v", got.Architecture, tt.wantArch)
			}
			if len(got.Qualifiers) != len(tt.wantQuals) {
				t.Errorf("Qualifiers length = %v, want %v", len(got.Qualifiers), len(tt.wantQuals))
			}
		})
	}
}

func TestRuntimeIdentifier_String(t *testing.T) {
	rid := &RuntimeIdentifier{
		RID:          "win10-x64",
		OS:           "win10",
		Architecture: "x64",
	}

	if rid.String() != "win10-x64" {
		t.Errorf("String() = %v, want win10-x64", rid.String())
	}
}

func TestNewRuntimeGraph(t *testing.T) {
	graph := NewRuntimeGraph()

	if graph == nil {
		t.Fatal("NewRuntimeGraph() returned nil")
	}
	if graph.Runtimes == nil {
		t.Error("Runtimes map is nil")
	}
	if graph.Supports == nil {
		t.Error("Supports map is nil")
	}
}

func TestRuntimeGraph_AddRuntime(t *testing.T) {
	graph := NewRuntimeGraph()

	graph.AddRuntime("win-x64", []string{"win"})
	graph.AddRuntime("win", []string{"any"})

	if len(graph.Runtimes) != 2 {
		t.Errorf("Runtimes count = %v, want 2", len(graph.Runtimes))
	}

	desc := graph.Runtimes["win-x64"]
	if desc == nil {
		t.Fatal("win-x64 not found")
	}
	if desc.RID != "win-x64" {
		t.Errorf("RID = %v, want win-x64", desc.RID)
	}
	if len(desc.Imports) != 1 || desc.Imports[0] != "win" {
		t.Errorf("Imports = %v, want [win]", desc.Imports)
	}
}

func TestRuntimeGraph_ExpandRuntime(t *testing.T) {
	graph := NewRuntimeGraph()
	graph.AddRuntime("base", nil)
	graph.AddRuntime("any", []string{"base"})
	graph.AddRuntime("win", []string{"any"})
	graph.AddRuntime("win-x64", []string{"win"})
	graph.AddRuntime("win10-x64", []string{"win10", "win-x64"})
	graph.AddRuntime("win10", []string{"win"})

	tests := []struct {
		name string
		rid  string
		want []string
	}{
		{
			name: "base",
			rid:  "base",
			want: []string{"base"},
		},
		{
			name: "any",
			rid:  "any",
			want: []string{"any", "base"},
		},
		{
			name: "win",
			rid:  "win",
			want: []string{"win", "any", "base"},
		},
		{
			name: "win-x64",
			rid:  "win-x64",
			want: []string{"win-x64", "win", "any", "base"},
		},
		{
			name: "win10-x64 BFS order",
			rid:  "win10-x64",
			want: []string{"win10-x64", "win10", "win-x64", "win", "any", "base"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.ExpandRuntime(tt.rid)

			if len(got) != len(tt.want) {
				t.Fatalf("ExpandRuntime() length = %v, want %v\nGot: %v\nWant: %v",
					len(got), len(tt.want), got, tt.want)
			}

			for i, rid := range tt.want {
				if got[i] != rid {
					t.Errorf("ExpandRuntime()[%d] = %v, want %v", i, got[i], rid)
				}
			}
		})
	}
}

func TestRuntimeGraph_ExpandRuntime_Caching(t *testing.T) {
	graph := NewRuntimeGraph()
	graph.AddRuntime("base", nil)
	graph.AddRuntime("any", []string{"base"})
	graph.AddRuntime("win", []string{"any"})

	// First call - should populate cache
	result1 := graph.ExpandRuntime("win")

	// Second call - should use cache
	result2 := graph.ExpandRuntime("win")

	if len(result1) != len(result2) {
		t.Errorf("Cached result differs in length: %v vs %v", len(result1), len(result2))
	}

	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("Cached result differs at index %d: %v vs %v", i, result1[i], result2[i])
		}
	}
}

func TestRuntimeGraph_AreCompatible(t *testing.T) {
	graph := NewRuntimeGraph()
	graph.AddRuntime("base", nil)
	graph.AddRuntime("any", []string{"base"})
	graph.AddRuntime("win", []string{"any"})
	graph.AddRuntime("win-x64", []string{"win"})
	graph.AddRuntime("win10", []string{"win"})
	graph.AddRuntime("win10-x64", []string{"win10", "win-x64"})

	tests := []struct {
		name      string
		targetRID string
		pkgRID    string
		want      bool
	}{
		{
			name:      "exact match",
			targetRID: "win10-x64",
			pkgRID:    "win10-x64",
			want:      true,
		},
		{
			name:      "compatible - win10-x64 can use win10",
			targetRID: "win10-x64",
			pkgRID:    "win10",
			want:      true,
		},
		{
			name:      "compatible - win10-x64 can use win-x64",
			targetRID: "win10-x64",
			pkgRID:    "win-x64",
			want:      true,
		},
		{
			name:      "compatible - win10-x64 can use win",
			targetRID: "win10-x64",
			pkgRID:    "win",
			want:      true,
		},
		{
			name:      "compatible - win10-x64 can use any",
			targetRID: "win10-x64",
			pkgRID:    "any",
			want:      true,
		},
		{
			name:      "compatible - win10-x64 can use base",
			targetRID: "win10-x64",
			pkgRID:    "base",
			want:      true,
		},
		{
			name:      "incompatible - win10 cannot use win10-x64",
			targetRID: "win10",
			pkgRID:    "win10-x64",
			want:      false,
		},
		{
			name:      "incompatible - win cannot use win-x64",
			targetRID: "win",
			pkgRID:    "win-x64",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.AreCompatible(tt.targetRID, tt.pkgRID)
			if got != tt.want {
				t.Errorf("AreCompatible(%v, %v) = %v, want %v",
					tt.targetRID, tt.pkgRID, got, tt.want)
			}
		})
	}
}

func TestRuntimeGraph_AreCompatible_Caching(t *testing.T) {
	graph := NewRuntimeGraph()
	graph.AddRuntime("base", nil)
	graph.AddRuntime("any", []string{"base"})
	graph.AddRuntime("win", []string{"any"})
	graph.AddRuntime("win-x64", []string{"win"})

	// First call - should populate cache
	result1 := graph.AreCompatible("win-x64", "win")

	// Second call - should use cache
	result2 := graph.AreCompatible("win-x64", "win")

	if result1 != result2 {
		t.Errorf("Cached result differs: %v vs %v", result1, result2)
	}
}

func TestLoadDefaultRuntimeGraph(t *testing.T) {
	graph := LoadDefaultRuntimeGraph()

	if graph == nil {
		t.Fatal("LoadDefaultRuntimeGraph() returned nil")
	}

	// Check foundation RIDs
	tests := []struct {
		name string
		rid  string
	}{
		{"base", "base"},
		{"any", "any"},
		{"win", "win"},
		{"win-x64", "win-x64"},
		{"win10-x64", "win10-x64"},
		{"linux", "linux"},
		{"linux-x64", "linux-x64"},
		{"ubuntu-x64", "ubuntu-x64"},
		{"osx", "osx"},
		{"osx-x64", "osx-x64"},
		{"osx-arm64", "osx-arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := graph.Runtimes[tt.rid]; !ok {
				t.Errorf("RID %v not found in default graph", tt.rid)
			}
		})
	}

	// Verify win10-x64 expansion includes expected RIDs
	expansion := graph.ExpandRuntime("win10-x64")
	expectedRIDs := []string{"win10-x64", "win10", "win-x64", "win81-x64", "win", "any", "base"}

	for _, expectedRID := range expectedRIDs {
		if !slices.Contains(expansion, expectedRID) {
			t.Errorf("Expected RID %v not found in win10-x64 expansion: %v", expectedRID, expansion)
		}
	}
}

func TestRuntimeGraph_GetAllCompatibleRIDs(t *testing.T) {
	graph := LoadDefaultRuntimeGraph()

	rids := graph.GetAllCompatibleRIDs("win10-x64")

	if len(rids) == 0 {
		t.Error("GetAllCompatibleRIDs returned empty slice")
	}

	// Should start with the requested RID
	if rids[0] != "win10-x64" {
		t.Errorf("First RID = %v, want win10-x64", rids[0])
	}

	// Should end with foundation RIDs
	lastRID := rids[len(rids)-1]
	if lastRID != "base" {
		t.Errorf("Last RID = %v, want base", lastRID)
	}
}

func TestRuntimeGraph_FindRuntimeDependencies(t *testing.T) {
	graph := NewRuntimeGraph()
	graph.AddRuntime("base", nil)
	graph.AddRuntime("any", []string{"base"})
	graph.AddRuntime("win", []string{"any"})
	graph.AddRuntime("win-x64", []string{"win"})

	// Add runtime dependencies
	desc := graph.Runtimes["win-x64"]
	desc.RuntimeDependencies["MyPackage"] = &RuntimeDependencySet{
		ID: "MyPackage",
		Dependencies: map[string]*RuntimePackageDependency{
			"NativeLib": {
				ID:           "NativeLib",
				VersionRange: "1.0.0",
			},
		},
	}

	// Should find dependencies for exact RID
	deps := graph.FindRuntimeDependencies("win-x64", "MyPackage")
	if len(deps) != 1 {
		t.Fatalf("FindRuntimeDependencies() returned %v deps, want 1", len(deps))
	}
	if deps[0].ID != "NativeLib" {
		t.Errorf("Dependency ID = %v, want NativeLib", deps[0].ID)
	}

	// Should return nil for package without dependencies
	deps = graph.FindRuntimeDependencies("win-x64", "OtherPackage")
	if deps != nil {
		t.Errorf("FindRuntimeDependencies() = %v, want nil", deps)
	}
}

func TestLoadFromJSON(t *testing.T) {
	jsonData := []byte(`{
		"runtimes": {
			"win": {
				"#import": ["any"]
			},
			"win-x64": {
				"#import": ["win"]
			},
			"any": {}
		}
	}`)

	graph, err := LoadFromJSON(jsonData)
	if err != nil {
		t.Fatalf("LoadFromJSON() error = %v", err)
	}

	if len(graph.Runtimes) != 3 {
		t.Errorf("Runtimes count = %v, want 3", len(graph.Runtimes))
	}

	// Check imports
	winDesc := graph.Runtimes["win"]
	if len(winDesc.Imports) != 1 || winDesc.Imports[0] != "any" {
		t.Errorf("win imports = %v, want [any]", winDesc.Imports)
	}

	winx64Desc := graph.Runtimes["win-x64"]
	if len(winx64Desc.Imports) != 1 || winx64Desc.Imports[0] != "win" {
		t.Errorf("win-x64 imports = %v, want [win]", winx64Desc.Imports)
	}
}

func TestLoadFromJSON_WithDependencies(t *testing.T) {
	jsonData := []byte(`{
		"runtimes": {
			"win-x64": {
				"#import": ["win"],
				"MyPackage": {
					"NativeLib": "1.0.0"
				}
			}
		}
	}`)

	graph, err := LoadFromJSON(jsonData)
	if err != nil {
		t.Fatalf("LoadFromJSON() error = %v", err)
	}

	desc := graph.Runtimes["win-x64"]
	if len(desc.RuntimeDependencies) != 1 {
		t.Errorf("RuntimeDependencies count = %v, want 1", len(desc.RuntimeDependencies))
	}

	depSet := desc.RuntimeDependencies["MyPackage"]
	if depSet == nil {
		t.Fatal("MyPackage dependencies not found")
	}

	dep := depSet.Dependencies["NativeLib"]
	if dep == nil {
		t.Fatal("NativeLib dependency not found")
	}
	if dep.VersionRange != "1.0.0" {
		t.Errorf("VersionRange = %v, want 1.0.0", dep.VersionRange)
	}
}

func TestLoadFromJSON_WithSupports(t *testing.T) {
	jsonData := []byte(`{
		"supports": {
			"net46.app": {
				"net46": ["win", "win-x86", "win-x64"]
			}
		}
	}`)

	graph, err := LoadFromJSON(jsonData)
	if err != nil {
		t.Fatalf("LoadFromJSON() error = %v", err)
	}

	if len(graph.Supports) != 1 {
		t.Errorf("Supports count = %v, want 1", len(graph.Supports))
	}

	profile := graph.Supports["net46.app"]
	if profile == nil {
		t.Fatal("net46.app profile not found")
	}

	if len(profile.RestoreContexts) != 3 {
		t.Errorf("RestoreContexts count = %v, want 3", len(profile.RestoreContexts))
	}

	// Check one of the contexts
	for _, ctx := range profile.RestoreContexts {
		if ctx.Framework != "net46" {
			t.Errorf("Framework = %v, want net46", ctx.Framework)
		}
	}
}

func TestRuntimeIdentifier_IsCompatible(t *testing.T) {
	graph := LoadDefaultRuntimeGraph()

	rid1, _ := ParseRID("win10-x64")
	rid2, _ := ParseRID("win-x64")
	rid3, _ := ParseRID("linux-x64")

	// Compatible with graph
	if !rid1.IsCompatible(rid2, graph) {
		t.Error("win10-x64 should be compatible with win-x64")
	}

	// Not compatible
	if rid1.IsCompatible(rid3, graph) {
		t.Error("win10-x64 should not be compatible with linux-x64")
	}

	// Exact match
	if !rid1.IsCompatible(rid1, graph) {
		t.Error("RID should be compatible with itself")
	}
}

func TestLoadFromJSON_InvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"invalid JSON", `{invalid`},
		{"empty", ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFromJSON([]byte(tt.json))
			if err == nil {
				t.Error("LoadFromJSON() expected error for invalid JSON")
			}
		})
	}
}
