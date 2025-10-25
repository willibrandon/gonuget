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

	// Step 4: Resolve conflicts
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
	}, nil
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
	*packages = append(*packages, node.Item)

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
