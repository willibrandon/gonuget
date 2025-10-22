package assets

import (
	"encoding/json"
	"fmt"
	"sync"
)

// RuntimeGraph represents the RID compatibility graph.
// Reference: NuGet.RuntimeModel/RuntimeGraph.cs
type RuntimeGraph struct {
	// Runtimes maps RID to its description
	Runtimes map[string]*RuntimeDescription

	// Supports maps profile name to compatibility profiles
	// Used for portable app compatibility (e.g., "net46.app" profile)
	Supports map[string]*CompatibilityProfile

	// Caches for performance (NuGet.Client uses ConcurrentDictionary)
	expandCache map[string][]string
	compatCache map[compatKey]bool
	cacheMutex  sync.RWMutex
}

// CompatibilityProfile defines framework/RID combinations for portable apps.
// Example: "net46.app" profile might support net46 on win/win-x86/win-x64
type CompatibilityProfile struct {
	Name            string
	RestoreContexts []*FrameworkRuntimePair // Framework + optional RID pairs
}

// FrameworkRuntimePair represents a framework with an optional RID.
// Used in compatibility profiles to define supported combinations.
type FrameworkRuntimePair struct {
	Framework string // e.g., "net46", "netstandard2.0"
	RID       string // Empty string means RID-agnostic, otherwise specific RID
}

type compatKey struct {
	target string
	pkg    string
}

// RuntimeDescription describes a runtime and its compatible runtimes.
// Reference: NuGet.RuntimeModel/RuntimeDescription.cs
type RuntimeDescription struct {
	RID                 string
	Imports             []string                         // Compatible RIDs (less specific) - JSON key is "#import"
	RuntimeDependencies map[string]*RuntimeDependencySet // Package ID -> RID-specific dependencies
}

// RuntimeDependencySet represents RID-specific package dependencies.
// Example: win10-x64 might require additional native dependencies.
type RuntimeDependencySet struct {
	ID           string                               // Package ID that has RID-specific dependencies
	Dependencies map[string]*RuntimePackageDependency // Dependency ID -> version constraint
}

// RuntimePackageDependency represents a single RID-specific dependency.
type RuntimePackageDependency struct {
	ID           string
	VersionRange string // e.g., "1.0.0", "[1.0.0,2.0.0)"
}

// NewRuntimeGraph creates an empty runtime graph.
func NewRuntimeGraph() *RuntimeGraph {
	return &RuntimeGraph{
		Runtimes:    make(map[string]*RuntimeDescription),
		Supports:    make(map[string]*CompatibilityProfile),
		expandCache: make(map[string][]string),
		compatCache: make(map[compatKey]bool),
	}
}

// LoadDefaultRuntimeGraph loads the default .NET RID graph.
// Reference: https://learn.microsoft.com/en-us/dotnet/core/rid-catalog
func LoadDefaultRuntimeGraph() *RuntimeGraph {
	graph := NewRuntimeGraph()

	// Foundation RIDs (CRITICAL: must include these for official compatibility)
	graph.AddRuntime("base", nil)             // Root of all RID inheritance
	graph.AddRuntime("any", []string{"base"}) // Platform-agnostic fallback

	// Windows (all OS RIDs import 'any')
	graph.AddRuntime("win", []string{"any"})
	graph.AddRuntime("win-x86", []string{"win"})
	graph.AddRuntime("win-x64", []string{"win"})
	graph.AddRuntime("win-arm", []string{"win"})
	graph.AddRuntime("win-arm64", []string{"win"})

	// Windows version chain (win7 → win8 → win81 → win10)
	graph.AddRuntime("win7", []string{"win"})
	graph.AddRuntime("win7-x86", []string{"win7", "win-x86"})
	graph.AddRuntime("win7-x64", []string{"win7", "win-x64"})

	graph.AddRuntime("win8", []string{"win7"})
	graph.AddRuntime("win8-x86", []string{"win8", "win7-x86"})
	graph.AddRuntime("win8-x64", []string{"win8", "win7-x64"})
	graph.AddRuntime("win8-arm", []string{"win8", "win-arm"})

	graph.AddRuntime("win81", []string{"win8"})
	graph.AddRuntime("win81-x86", []string{"win81", "win8-x86"})
	graph.AddRuntime("win81-x64", []string{"win81", "win8-x64"})
	graph.AddRuntime("win81-arm", []string{"win81", "win8-arm"})

	graph.AddRuntime("win10", []string{"win81"})
	graph.AddRuntime("win10-x86", []string{"win10", "win81-x86"})
	graph.AddRuntime("win10-x64", []string{"win10", "win81-x64"})
	graph.AddRuntime("win10-arm", []string{"win10", "win81-arm"})
	graph.AddRuntime("win10-arm64", []string{"win10", "win-arm64"})

	// Linux (all OS RIDs import 'any')
	graph.AddRuntime("linux", []string{"any"})
	graph.AddRuntime("linux-x64", []string{"linux"})
	graph.AddRuntime("linux-arm", []string{"linux"})
	graph.AddRuntime("linux-arm64", []string{"linux"})

	// Ubuntu
	graph.AddRuntime("ubuntu", []string{"linux"})
	graph.AddRuntime("ubuntu-x64", []string{"ubuntu", "linux-x64"})
	graph.AddRuntime("ubuntu.20.04-x64", []string{"ubuntu-x64"})
	graph.AddRuntime("ubuntu.22.04-x64", []string{"ubuntu-x64"})
	graph.AddRuntime("ubuntu.24.04-x64", []string{"ubuntu-x64"})

	// Debian
	graph.AddRuntime("debian", []string{"linux"})
	graph.AddRuntime("debian-x64", []string{"debian", "linux-x64"})

	// macOS (all OS RIDs import 'any')
	graph.AddRuntime("osx", []string{"any"})
	graph.AddRuntime("osx-x64", []string{"osx"})
	graph.AddRuntime("osx-arm64", []string{"osx"})
	graph.AddRuntime("osx.10.12-x64", []string{"osx-x64"})
	graph.AddRuntime("osx.11-x64", []string{"osx-x64"})
	graph.AddRuntime("osx.12-x64", []string{"osx-x64"})
	graph.AddRuntime("osx.12-arm64", []string{"osx-arm64"})
	graph.AddRuntime("osx.13-arm64", []string{"osx-arm64"})

	return graph
}

// AddRuntime adds a runtime to the graph.
func (g *RuntimeGraph) AddRuntime(rid string, imports []string) {
	g.Runtimes[rid] = &RuntimeDescription{
		RID:                 rid,
		Imports:             imports,
		RuntimeDependencies: make(map[string]*RuntimeDependencySet),
	}
}

// ExpandRuntime returns all compatible RIDs in priority order (nearest first).
// Uses BFS traversal to ensure correct ordering for asset selection.
// Example: win10-x64 expands to [win10-x64, win10, win-x64, win81-x64, win81, ..., win, any, base]
// Reference: NuGet.Client RuntimeGraph.cs ExpandRuntime
func (g *RuntimeGraph) ExpandRuntime(rid string) []string {
	// Check cache first (performance optimization from NuGet.Client)
	g.cacheMutex.RLock()
	if cached, ok := g.expandCache[rid]; ok {
		g.cacheMutex.RUnlock()
		return cached
	}
	g.cacheMutex.RUnlock()

	// BFS traversal (matches NuGet.Client RuntimeGraph.cs)
	var result []string
	visited := make(map[string]bool)
	queue := []string{rid}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true
		result = append(result, current)

		// Add imports to queue (BFS ensures nearest RIDs come first)
		if desc, ok := g.Runtimes[current]; ok {
			for _, importRID := range desc.Imports {
				if !visited[importRID] {
					queue = append(queue, importRID)
				}
			}
		}
	}

	// Cache result for future lookups
	g.cacheMutex.Lock()
	g.expandCache[rid] = result
	g.cacheMutex.Unlock()

	return result
}

// AreCompatible checks if targetRID is compatible with packageRID.
// Uses cached expansion for performance (NuGet.Client uses ConcurrentDictionary).
func (g *RuntimeGraph) AreCompatible(targetRID, packageRID string) bool {
	// Exact match (fast path)
	if targetRID == packageRID {
		return true
	}

	// Check cache
	key := compatKey{target: targetRID, pkg: packageRID}
	g.cacheMutex.RLock()
	if result, ok := g.compatCache[key]; ok {
		g.cacheMutex.RUnlock()
		return result
	}
	g.cacheMutex.RUnlock()

	// Expand and check
	result := false
	for _, compatRID := range g.ExpandRuntime(targetRID) {
		if compatRID == packageRID {
			result = true
			break
		}
	}

	// Cache result
	g.cacheMutex.Lock()
	g.compatCache[key] = result
	g.cacheMutex.Unlock()

	return result
}

// GetAllCompatibleRIDs returns all RIDs compatible with the target RID.
// This is a convenience wrapper around ExpandRuntime for clarity.
func (g *RuntimeGraph) GetAllCompatibleRIDs(targetRID string) []string {
	return g.ExpandRuntime(targetRID)
}

// FindRuntimeDependencies finds RID-specific dependencies for a package.
// Returns the dependencies for the most specific compatible RID.
// Reference: NuGet.Client RuntimeGraph.cs FindRuntimeDependencies
func (g *RuntimeGraph) FindRuntimeDependencies(runtimeName, packageID string) []*RuntimePackageDependency {
	// Expand RID in priority order (nearest first)
	for _, expandedRID := range g.ExpandRuntime(runtimeName) {
		if desc, ok := g.Runtimes[expandedRID]; ok {
			if depSet, ok := desc.RuntimeDependencies[packageID]; ok {
				// Found dependencies for this RID
				var deps []*RuntimePackageDependency
				for _, dep := range depSet.Dependencies {
					deps = append(deps, dep)
				}
				return deps
			}
		}
	}

	return nil // No RID-specific dependencies
}

// LoadFromJSON loads a runtime graph from JSON.
// Reference: NuGet.Client JsonRuntimeFormat.cs
func LoadFromJSON(data []byte) (*RuntimeGraph, error) {
	// Parse raw JSON to handle dynamic structure
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal runtime graph: %w", err)
	}

	graph := NewRuntimeGraph()

	// Parse "runtimes" section
	if runtimesData, ok := raw["runtimes"]; ok {
		var runtimes map[string]json.RawMessage
		if err := json.Unmarshal(runtimesData, &runtimes); err != nil {
			return nil, fmt.Errorf("unmarshal runtimes: %w", err)
		}

		for rid, ridData := range runtimes {
			// Parse as map to handle both #import and package dependencies
			var ridMap map[string]json.RawMessage
			if err := json.Unmarshal(ridData, &ridMap); err != nil {
				return nil, fmt.Errorf("unmarshal rid %s: %w", rid, err)
			}

			desc := &RuntimeDescription{
				RID:                 rid,
				RuntimeDependencies: make(map[string]*RuntimeDependencySet),
			}

			// Parse #import (inheritance)
			if importsData, ok := ridMap["#import"]; ok {
				var imports []string
				if err := json.Unmarshal(importsData, &imports); err != nil {
					return nil, fmt.Errorf("unmarshal imports for %s: %w", rid, err)
				}
				desc.Imports = imports
			}

			// Parse package-specific dependencies (any key that's not #import)
			for key, val := range ridMap {
				if key == "#import" {
					continue
				}
				// key is package ID, val is map of dependency ID -> version
				packageID := key
				var deps map[string]string
				if err := json.Unmarshal(val, &deps); err != nil {
					return nil, fmt.Errorf("unmarshal dependencies for %s/%s: %w", rid, packageID, err)
				}

				depSet := &RuntimeDependencySet{
					ID:           packageID,
					Dependencies: make(map[string]*RuntimePackageDependency),
				}

				for depID, version := range deps {
					depSet.Dependencies[depID] = &RuntimePackageDependency{
						ID:           depID,
						VersionRange: version,
					}
				}

				desc.RuntimeDependencies[packageID] = depSet
			}

			graph.Runtimes[rid] = desc
		}
	}

	// Parse "supports" section (compatibility profiles)
	if supportsData, ok := raw["supports"]; ok {
		var supports map[string]json.RawMessage
		if err := json.Unmarshal(supportsData, &supports); err != nil {
			return nil, fmt.Errorf("unmarshal supports: %w", err)
		}

		for profileName, profileData := range supports {
			// Parse as map of framework -> RID(s)
			var profileMap map[string]json.RawMessage
			if err := json.Unmarshal(profileData, &profileMap); err != nil {
				return nil, fmt.Errorf("unmarshal profile %s: %w", profileName, err)
			}

			profile := &CompatibilityProfile{
				Name:            profileName,
				RestoreContexts: make([]*FrameworkRuntimePair, 0),
			}

			for framework, ridsData := range profileMap {
				// Value can be string (single RID) or array (multiple RIDs)
				var rids []string
				// Try array first
				if err := json.Unmarshal(ridsData, &rids); err != nil {
					// Try single string
					var singleRID string
					if err2 := json.Unmarshal(ridsData, &singleRID); err2 != nil {
						return nil, fmt.Errorf("unmarshal RIDs for %s/%s: %w", profileName, framework, err)
					}
					rids = []string{singleRID}
				}

				for _, rid := range rids {
					profile.RestoreContexts = append(profile.RestoreContexts, &FrameworkRuntimePair{
						Framework: framework,
						RID:       rid,
					})
				}
			}

			graph.Supports[profileName] = profile
		}
	}

	return graph, nil
}
