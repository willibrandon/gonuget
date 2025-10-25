package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/willibrandon/gonuget/core/resolver"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
)

// WalkGraphHandler walks the dependency graph for a package.
type WalkGraphHandler struct{}

func (h *WalkGraphHandler) ErrorCode() string { return "WALK_001" }

func (h *WalkGraphHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req WalkGraphRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.PackageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}
	if req.VersionRange == "" {
		return nil, fmt.Errorf("versionRange is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}

	// Create real NuGet V3 metadata client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)

	// Wrap V3 client to implement resolver.PackageMetadataClient
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// Create walker
	walker := resolver.NewDependencyWalker(client, req.Sources, req.TargetFramework)

	// Walk the graph
	rootNode, err := walker.Walk(
		context.Background(),
		req.PackageID,
		req.VersionRange,
		req.TargetFramework,
	)
	if err != nil {
		return nil, fmt.Errorf("walk graph: %w", err)
	}

	// Detect conflicts and downgrades
	conflictDetector := &resolver.ConflictDetector{}
	_, downgrades := conflictDetector.DetectFromGraph(rootNode)

	// Collect all nodes in flat array format
	nodes := make([]GraphNodeData, 0)
	collectNodesFlat(rootNode, &nodes)

	// Collect cycles (package IDs that form circular dependencies)
	cycles := make([]string, 0)
	collectCycles(rootNode, &cycles)

	// Convert downgrades to response format
	downgradeInfos := make([]DowngradeInfo, len(downgrades))
	for i, dw := range downgrades {
		downgradeInfos[i] = DowngradeInfo{
			PackageID:   dw.PackageID,
			FromVersion: dw.CurrentVersion,
			ToVersion:   dw.TargetVersion,
		}
	}

	// Build response with flat arrays
	resp := WalkGraphResponse{
		Nodes:      nodes,
		Cycles:     cycles,
		Downgrades: downgradeInfos,
	}

	return resp, nil
}

// ResolveConflictsHandler resolves version conflicts in a dependency set.
type ResolveConflictsHandler struct{}

func (h *ResolveConflictsHandler) ErrorCode() string { return "RESOLVE_001" }

func (h *ResolveConflictsHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ResolveConflictsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if len(req.PackageIDs) == 0 {
		return nil, fmt.Errorf("packageIds is required")
	}
	if len(req.VersionRanges) != len(req.PackageIDs) {
		return nil, fmt.Errorf("versionRanges must match packageIds length")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}

	// Use default source (NuGet.org V3) if not provided
	sources := []string{"https://api.nuget.org/v3/"}

	// Create real NuGet V3 metadata client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)

	// Wrap V3 client to implement resolver.PackageMetadataClient
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// For now, resolve each package individually and combine results
	// In practice, we'd want to walk all packages together to detect cross-package conflicts
	allPackages := make(map[string]ResolvedPackage) // Use map to deduplicate

	for i, packageID := range req.PackageIDs {
		versionRange := req.VersionRanges[i]

		// Create resolver
		r := resolver.NewResolver(client, sources, req.TargetFramework)

		// Resolve the package
		result, err := r.Resolve(context.Background(), packageID, versionRange)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", packageID, err)
		}

		// Add all resolved packages to the result
		for _, pkg := range result.Packages {
			// Calculate depth (0 for root, traverse OuterEdge chain for others)
			depth := 0
			// In the current Resolver implementation, we don't track depth in Packages
			// For now, root packages have depth 0, all transitive deps have depth > 0
			// We can infer: if packageID matches input, depth is 0
			if pkg.ID == packageID {
				depth = 0
			} else {
				depth = 1 // Simplified: all transitive deps get depth 1+
			}

			// Use package (don't try to pick "latest" - just use what Resolver selected)
			// The Resolver already ran conflict resolution and picked the correct version
			if _, found := allPackages[pkg.ID]; !found {
				allPackages[pkg.ID] = ResolvedPackage{
					PackageID: pkg.ID,
					Version:   pkg.Version,
					Depth:     depth,
				}
			}
		}
	}

	// Convert map to slice
	packages := make([]ResolvedPackage, 0, len(allPackages))
	for _, pkg := range allPackages {
		packages = append(packages, pkg)
	}

	return ResolveConflictsResponse{Packages: packages}, nil
}

// collectNodesFlat recursively collects all nodes into a flat array.
func collectNodesFlat(node *resolver.GraphNode, nodes *[]GraphNodeData) {
	if node == nil {
		return
	}

	// Collect dependency IDs
	deps := make([]string, 0, len(node.InnerNodes))
	for _, child := range node.InnerNodes {
		if child.Item != nil {
			deps = append(deps, child.Item.ID)
		}
	}

	// Add current node
	nodeData := GraphNodeData{
		PackageID:    "",
		Version:      "",
		Disposition:  node.Disposition.String(),
		Depth:        node.Depth,
		Dependencies: deps,
	}

	if node.Item != nil {
		nodeData.PackageID = node.Item.ID
		nodeData.Version = node.Item.Version
	}

	*nodes = append(*nodes, nodeData)

	// Recursively collect children
	for _, child := range node.InnerNodes {
		collectNodesFlat(child, nodes)
	}
}

// collectCycles collects package IDs that have Cycle disposition.
func collectCycles(node *resolver.GraphNode, cycles *[]string) {
	if node == nil {
		return
	}

	// Check if this node is a cycle
	if node.Disposition == resolver.DispositionCycle && node.Item != nil {
		*cycles = append(*cycles, node.Item.ID)
	}

	// Recursively check children
	for _, child := range node.InnerNodes {
		collectCycles(child, cycles)
	}
}

// v3MetadataClientAdapter adapts v3.MetadataClient to resolver.PackageMetadataClient.
type v3MetadataClientAdapter struct {
	v3Client *v3.MetadataClient
}

// GetPackageMetadata implements resolver.PackageMetadataClient by fetching from NuGet V3 API.
func (a *v3MetadataClientAdapter) GetPackageMetadata(ctx context.Context, source string, packageID string) ([]*resolver.PackageDependencyInfo, error) {
	// Fetch registration index from V3 API
	index, err := a.v3Client.GetPackageMetadata(ctx, source, packageID)
	if err != nil {
		return nil, err
	}

	// Convert all versions to PackageDependencyInfo
	var packages []*resolver.PackageDependencyInfo
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry == nil {
				continue
			}

			pkg := &resolver.PackageDependencyInfo{
				ID:               leaf.CatalogEntry.PackageID,
				Version:          leaf.CatalogEntry.Version,
				DependencyGroups: make([]resolver.DependencyGroup, 0, len(leaf.CatalogEntry.DependencyGroups)),
			}

			// Convert dependency groups
			for _, v3Group := range leaf.CatalogEntry.DependencyGroups {
				group := resolver.DependencyGroup{
					TargetFramework: normalizeFramework(v3Group.TargetFramework),
					Dependencies:    make([]resolver.PackageDependency, 0, len(v3Group.Dependencies)),
				}

				// Convert dependencies
				for _, v3Dep := range v3Group.Dependencies {
					dep := resolver.PackageDependency{
						ID:              v3Dep.ID,
						VersionRange:    v3Dep.Range,
						TargetFramework: group.TargetFramework,
					}
					group.Dependencies = append(group.Dependencies, dep)
				}

				pkg.DependencyGroups = append(pkg.DependencyGroups, group)
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// normalizeFramework normalizes framework strings to match NuGet.Client format.
func normalizeFramework(fw string) string {
	if fw == "" {
		return ""
	}
	// V3 API returns frameworks like ".NETCoreApp3.1" but we need "netcoreapp3.1"
	fw = strings.ToLower(fw)
	fw = strings.TrimPrefix(fw, ".")
	fw = strings.ReplaceAll(fw, " ", "")
	return fw
}
