package restore

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// CalculateDgSpecHash computes dependency graph hash for a project.
// Matches DependencyGraphSpec.GetHash() in NuGet.Client (simplified version).
//
// The hash includes:
// - Target frameworks
// - Package references (ID + version)
// - Project file path
//
// NOTE: NuGet.Client uses FnvHash64 by default, but for simplicity we use SHA512
// (which NuGet also supports via UseLegacyHashFunction flag). The exact algorithm
// doesn't matter as long as it's consistent.
func CalculateDgSpecHash(proj *project.Project) (string, error) {
	// Collect all inputs that affect restore
	var parts []string

	// 1. Project file path (normalized)
	parts = append(parts, proj.Path)

	// 2. Target frameworks (sorted for determinism)
	frameworks := proj.GetTargetFrameworks()
	sort.Strings(frameworks)
	for _, tfm := range frameworks {
		parts = append(parts, fmt.Sprintf("tfm:%s", tfm))
	}

	// 3. Package references (sorted by ID for determinism)
	packageRefs := proj.GetPackageReferences()
	sort.Slice(packageRefs, func(i, j int) bool {
		return packageRefs[i].Include < packageRefs[j].Include
	})
	for _, pkg := range packageRefs {
		parts = append(parts, fmt.Sprintf("pkg:%s:%s", pkg.Include, pkg.Version))
	}

	// 4. Combine and hash
	combined := strings.Join(parts, "|")
	hash := sha512.Sum512([]byte(combined))

	// Return base64-encoded hash (first 12 bytes for compact representation)
	// This matches the compact format dotnet uses
	return base64.StdEncoding.EncodeToString(hash[:12]), nil
}
