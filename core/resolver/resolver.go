package resolver

import (
	"context"
	"fmt"
)

// Resolver provides high-level resolution API.
type Resolver struct {
	walker           *DependencyWalker
	conflictDetector *ConflictDetector
	conflictResolver *ConflictResolver
	parallelResolver *ParallelResolver
	targetFramework  string
}

// NewResolver creates a new resolver.
func NewResolver(client PackageMetadataClient, sources []string, targetFramework string) *Resolver {
	r := &Resolver{
		walker:           NewDependencyWalker(client, sources, targetFramework),
		conflictDetector: NewConflictDetector(),
		conflictResolver: NewConflictResolver(),
		targetFramework:  targetFramework,
	}

	// Add parallel resolver
	r.parallelResolver = NewParallelResolver(r, 10)

	return r
}

// Resolve performs complete dependency resolution with conflict resolution.
func (r *Resolver) Resolve(
	ctx context.Context,
	packageID string,
	versionRange string,
) (*ResolutionResult, error) {
	return r.resolveWithRecursive(ctx, packageID, versionRange, true)
}

// ResolveNonRecursive resolves a package without transitive dependencies.
// Matches NuGet.Client's WalkAsync with recursive: false.
func (r *Resolver) ResolveNonRecursive(
	ctx context.Context,
	packageID string,
	versionRange string,
) (*ResolutionResult, error) {
	return r.resolveWithRecursive(ctx, packageID, versionRange, false)
}

// resolveWithRecursive internal helper for resolution with configurable recursion.
func (r *Resolver) resolveWithRecursive(
	ctx context.Context,
	packageID string,
	versionRange string,
	recursive bool,
) (*ResolutionResult, error) {
	// Step 1: Walk dependency graph
	rootNode, err := r.walker.Walk(ctx, packageID, versionRange, r.targetFramework, recursive)
	if err != nil {
		return nil, fmt.Errorf("walk dependencies: %w", err)
	}

	// Step 2: Detect conflicts and downgrades
	conflicts, downgrades := r.conflictDetector.DetectFromGraph(rootNode)

	// Step 3: Analyze cycles
	cycleAnalyzer := NewCycleAnalyzer()
	cycles := cycleAnalyzer.AnalyzeCycles(rootNode)

	// Step 4: Collect unresolved packages with enhanced diagnostics
	// Matches NuGet.Client's RestoreTargetGraph.Create which populates Unresolved collection
	unresolved := r.collectUnresolvedPackages(ctx, rootNode)

	// Step 5: Resolve conflicts
	resolvedPackages := make([]*PackageDependencyInfo, 0)

	if len(conflicts) > 0 {
		// Group all nodes by package ID
		nodesByID := make(map[string][]*GraphNode)
		r.collectAllNodes(rootNode, nodesByID)

		// Resolve each conflict
		for _, nodes := range nodesByID {
			winner := r.conflictResolver.ResolveConflict(nodes)
			if winner != nil && winner.Item != nil {
				resolvedPackages = append(resolvedPackages, winner.Item)
			}
		}
	} else {
		// No conflicts - flatten graph
		resolvedPackages = r.flattenGraph(rootNode)
	}

	return &ResolutionResult{
		Packages:   resolvedPackages,
		Conflicts:  conflicts,
		Downgrades: downgrades,
		Cycles:     cycles,
		Unresolved: unresolved,
	}, nil
}

// collectUnresolvedPackages traverses the graph and collects all unresolved packages
// with enhanced diagnostics (error codes, available versions, nearest version).
// Matches NuGet.Client's RestoreTargetGraph.Create which populates the Unresolved collection.
func (r *Resolver) collectUnresolvedPackages(ctx context.Context, node *GraphNode) []UnresolvedPackage {
	visited := make(map[string]bool)
	unresolved := make([]UnresolvedPackage, 0)

	r.collectUnresolvedRecursive(ctx, node, visited, &unresolved)

	return unresolved
}

func (r *Resolver) collectUnresolvedRecursive(
	ctx context.Context,
	node *GraphNode,
	visited map[string]bool,
	unresolved *[]UnresolvedPackage,
) {
	if node == nil || node.Item == nil {
		return
	}

	key := node.Item.Key()
	if visited[key] {
		return
	}
	visited[key] = true

	// If this package is unresolved, diagnose and add to the list
	if node.Item.IsUnresolved {
		unresolvedPkg := r.diagnoseUnresolvedPackage(ctx, node.Item.ID, node.Item.Version)
		*unresolved = append(*unresolved, unresolvedPkg)
	}

	// Continue traversing children
	for _, child := range node.InnerNodes {
		r.collectUnresolvedRecursive(ctx, child, visited, unresolved)
	}
}

// diagnoseUnresolvedPackage queries sources to determine the specific error type and build diagnostics.
// Matches NuGet.Client's UnresolvedMessages.GetMessagesAsync behavior.
func (r *Resolver) diagnoseUnresolvedPackage(ctx context.Context, packageID string, versionRange string) UnresolvedPackage {
	// Get sources from walker
	sources := r.walker.sources

	// Try to get all available versions from all sources
	availableVersions := make([]string, 0)
	for _, source := range sources {
		// Query source for all versions of this package
		packages, err := r.walker.client.GetPackageMetadata(ctx, source, packageID, "0.0.0") // Query all versions
		if err != nil {
			continue // Source may not have this package
		}

		for _, pkg := range packages {
			if pkg.ID == packageID {
				availableVersions = append(availableVersions, pkg.Version)
			}
		}
	}

	// Determine error code based on what we found
	var errorCode NuGetErrorCode
	var message string
	var nearestVersion string

	if len(availableVersions) == 0 {
		// NU1101: Package doesn't exist at all
		errorCode = NU1101
		message = fmt.Sprintf("Unable to find package '%s'. No packages exist with this id in source(s): %s",
			packageID, formatSources(sources))
	} else {
		// Package exists but version doesn't match
		// TODO: Check if only prerelease versions exist (NU1103)
		// For now, use NU1102 (version mismatch)
		errorCode = NU1102

		// Find nearest version (for now, just use first available)
		if len(availableVersions) > 0 {
			nearestVersion = availableVersions[0]
		}

		message = fmt.Sprintf("Unable to find package '%s' with version (%s)",
			packageID, versionRange)
		if nearestVersion != "" {
			message += fmt.Sprintf("\n  - Found %d version(s) in source(s) [ Nearest version: %s ]",
				len(availableVersions), nearestVersion)
		}
	}

	return UnresolvedPackage{
		ID:                packageID,
		VersionRange:      versionRange,
		TargetFramework:   r.targetFramework,
		ErrorCode:         string(errorCode),
		Message:           message,
		Sources:           sources,
		AvailableVersions: availableVersions,
		NearestVersion:    nearestVersion,
	}
}

// formatSources creates a comma-separated list of source URLs for error messages.
func formatSources(sources []string) string {
	if len(sources) == 0 {
		return "(no sources configured)"
	}
	if len(sources) == 1 {
		return sources[0]
	}
	// Join with commas
	result := ""
	for i, source := range sources {
		if i > 0 {
			result += ", "
		}
		result += source
	}
	return result
}

// collectAllNodes collects all nodes from graph by package ID.
func (r *Resolver) collectAllNodes(node *GraphNode, nodesByID map[string][]*GraphNode) {
	if node == nil {
		return
	}

	if node.Item != nil {
		nodesByID[node.Item.ID] = append(nodesByID[node.Item.ID], node)
	}

	for _, child := range node.InnerNodes {
		r.collectAllNodes(child, nodesByID)
	}
}

// flattenGraph creates flat list of packages (no conflicts).
func (r *Resolver) flattenGraph(node *GraphNode) []*PackageDependencyInfo {
	visited := make(map[string]bool)
	packages := make([]*PackageDependencyInfo, 0)

	r.flattenGraphRecursive(node, visited, &packages)

	return packages
}

func (r *Resolver) flattenGraphRecursive(
	node *GraphNode,
	visited map[string]bool,
	packages *[]*PackageDependencyInfo,
) {
	if node == nil || node.Item == nil {
		return
	}

	key := node.Item.Key()
	if visited[key] {
		return
	}

	visited[key] = true

	// Only add resolved packages (skip unresolved ones)
	// Matches NuGet.Client: unresolved packages go in Unresolved collection, not Packages
	if !node.Item.IsUnresolved {
		*packages = append(*packages, node.Item)
	}

	for _, child := range node.InnerNodes {
		r.flattenGraphRecursive(child, visited, packages)
	}
}

// ResolveProject resolves transitive dependencies for a project with multiple direct dependencies.
func (r *Resolver) ResolveProject(
	ctx context.Context,
	dependencies []PackageDependency,
) (*ResolutionResult, error) {
	transitiveResolver := NewTransitiveResolver(r)
	return transitiveResolver.ResolveMultipleRoots(ctx, dependencies)
}

// ResolveMultiple resolves multiple packages in parallel.
func (r *Resolver) ResolveMultiple(
	ctx context.Context,
	packages []PackageDependency,
) ([]*ResolutionResult, error) {
	return r.parallelResolver.ResolveMultiplePackages(ctx, packages)
}

// ResolveBatch resolves packages in batches for better resource control.
func (r *Resolver) ResolveBatch(
	ctx context.Context,
	packages []PackageDependency,
	batchSize int,
) ([]*ResolutionResult, error) {
	return r.parallelResolver.BatchResolve(ctx, packages, batchSize)
}

// ReplaceParallelResolver replaces the parallel resolver with a custom one.
// This is useful for controlling worker pool limits.
func (r *Resolver) ReplaceParallelResolver(pr *ParallelResolver) {
	r.parallelResolver = pr
}
