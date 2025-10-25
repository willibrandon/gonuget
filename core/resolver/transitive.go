package resolver

import (
	"context"
	"fmt"
)

// TransitiveResolver resolves the complete transitive closure of dependencies
type TransitiveResolver struct {
	resolver *Resolver
}

// NewTransitiveResolver creates a new transitive resolver
func NewTransitiveResolver(resolver *Resolver) *TransitiveResolver {
	return &TransitiveResolver{
		resolver: resolver,
	}
}

// ResolveTransitive resolves all transitive dependencies for a package.
// Returns a flattened list of all unique packages in the dependency graph.
func (tr *TransitiveResolver) ResolveTransitive(
	ctx context.Context,
	packageID string,
	versionRange string,
) (*ResolutionResult, error) {
	// Use full resolver which handles conflicts, cycles, downgrades
	return tr.resolver.Resolve(ctx, packageID, versionRange)
}

// ResolveMultipleRoots resolves transitive dependencies for multiple root packages.
// Used for projects with multiple direct dependencies.
func (tr *TransitiveResolver) ResolveMultipleRoots(
	ctx context.Context,
	roots []PackageDependency,
) (*ResolutionResult, error) {
	// Create synthetic root info
	syntheticRoot := &PackageDependencyInfo{
		ID:           "__project__",
		Version:      "1.0.0",
		Dependencies: roots,
	}

	// Create synthetic root node manually (don't fetch from client)
	rootNode := &GraphNode{
		Key:         syntheticRoot.Key(),
		Item:        syntheticRoot,
		OuterNode:   nil,
		InnerNodes:  make([]*GraphNode, 0),
		ParentNodes: make([]*GraphNode, 0),
		Disposition: DispositionAcceptable,
		Depth:       0,
		OuterEdge:   nil,
	}

	// Walk each root dependency
	for _, dep := range roots {
		childNode, err := tr.resolver.walker.Walk(
			ctx,
			dep.ID,
			dep.VersionRange,
			tr.resolver.targetFramework,
		)
		if err != nil {
			return nil, fmt.Errorf("walk dependency %s: %w", dep.ID, err)
		}
		rootNode.InnerNodes = append(rootNode.InnerNodes, childNode)
		childNode.OuterNode = rootNode
	}

	// Detect conflicts, cycles, downgrades
	conflicts, downgrades := tr.resolver.conflictDetector.DetectFromGraph(rootNode)
	cycleAnalyzer := NewCycleAnalyzer()
	cycles := cycleAnalyzer.AnalyzeCycles(rootNode)

	// Resolve conflicts
	resolvedPackages := make([]*PackageDependencyInfo, 0)
	if len(conflicts) > 0 {
		nodesByID := make(map[string][]*GraphNode)
		tr.resolver.collectAllNodes(rootNode, nodesByID)
		for packageID, nodes := range nodesByID {
			// Skip synthetic root
			if packageID == "__project__" {
				continue
			}
			winner := tr.resolver.conflictResolver.ResolveConflict(nodes)
			if winner != nil && winner.Item != nil {
				resolvedPackages = append(resolvedPackages, winner.Item)
			}
		}
	} else {
		resolvedPackages = tr.flattenGraphExcludingRoot(rootNode)
	}

	return &ResolutionResult{
		Packages:   resolvedPackages,
		Conflicts:  conflicts,
		Downgrades: downgrades,
		Cycles:     cycles,
	}, nil
}

// flattenGraphExcludingRoot flattens graph but excludes synthetic root
func (tr *TransitiveResolver) flattenGraphExcludingRoot(root *GraphNode) []*PackageDependencyInfo {
	visited := make(map[string]bool)
	packages := make([]*PackageDependencyInfo, 0)

	// Start from children (skip synthetic root)
	for _, child := range root.InnerNodes {
		tr.flattenRecursive(child, visited, &packages)
	}

	return packages
}

// flattenRecursive recursively flattens graph
func (tr *TransitiveResolver) flattenRecursive(
	node *GraphNode,
	visited map[string]bool,
	packages *[]*PackageDependencyInfo,
) {
	if node == nil || node.Item == nil {
		return
	}

	// Skip cycle and downgrade nodes (they don't have Item)
	if node.Disposition == DispositionCycle || node.Disposition == DispositionPotentiallyDowngraded {
		if node.Item == nil {
			return
		}
	}

	key := node.Item.Key()
	if visited[key] {
		return
	}

	visited[key] = true
	*packages = append(*packages, node.Item)

	for _, child := range node.InnerNodes {
		tr.flattenRecursive(child, visited, packages)
	}
}
