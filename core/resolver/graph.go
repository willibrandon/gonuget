package resolver

// Disposition tracks the state of a node in the dependency graph.
// Matches NuGet.Client's Disposition enum.
type Disposition int

const (
	// DispositionAcceptable - Node is valid and can be used
	DispositionAcceptable Disposition = iota
	// DispositionRejected - Node was rejected (conflict, constraint violation)
	DispositionRejected
	// DispositionAccepted - Node was explicitly accepted
	DispositionAccepted
	// DispositionPotentiallyDowngraded - Node might cause a downgrade
	DispositionPotentiallyDowngraded
	// DispositionCycle - Node creates a circular dependency
	DispositionCycle
)

func (d Disposition) String() string {
	switch d {
	case DispositionAcceptable:
		return "Acceptable"
	case DispositionRejected:
		return "Rejected"
	case DispositionAccepted:
		return "Accepted"
	case DispositionPotentiallyDowngraded:
		return "PotentiallyDowngraded"
	case DispositionCycle:
		return "Cycle"
	default:
		return "Unknown"
	}
}

// GraphEdge represents the edge between two nodes in the dependency graph.
// Matches NuGet.Client's GraphEdge<RemoteResolveResult>.
type GraphEdge struct {
	// OuterEdge - parent edge (chain to root)
	OuterEdge *GraphEdge

	// Item - the package at this edge
	Item *PackageDependencyInfo

	// Edge - the dependency that created this edge
	Edge PackageDependency
}

// GraphNode represents a node in the dependency graph.
// Matches NuGet.Client's GraphNode<RemoteResolveResult>.
type GraphNode struct {
	// Key - unique identifier for this node (packageID|version)
	Key string

	// Item - package metadata and dependencies
	Item *PackageDependencyInfo

	// OuterNode - parent node (singular, for tree structure)
	OuterNode *GraphNode

	// InnerNodes - child nodes (dependencies)
	InnerNodes []*GraphNode

	// ParentNodes - tracks multiple parents when node is shared
	// Used when node is removed from outer node but needs parent tracking
	ParentNodes []*GraphNode

	// Disposition - state of this node
	Disposition Disposition

	// Depth - distance from root
	Depth int

	// OuterEdge - edge to this node from parent
	OuterEdge *GraphEdge
}

// PathFromRoot returns the path from root to this node
func (n *GraphNode) PathFromRoot() []string {
	if n == nil {
		return nil
	}

	path := make([]string, 0, n.Depth+1)
	current := n
	for current != nil {
		if current.Item != nil {
			path = append([]string{current.Item.String()}, path...)
		}
		current = current.OuterNode
	}
	return path
}

// AreAllParentsRejected checks if all parent nodes are rejected
func (n *GraphNode) AreAllParentsRejected() bool {
	if len(n.ParentNodes) == 0 {
		return false
	}

	for _, parent := range n.ParentNodes {
		if parent.Disposition != DispositionRejected {
			return false
		}
	}
	return true
}

// WalkerStackState represents the state of a single frame in the manual stack traversal.
// Matches NuGet.Client's GraphNodeStackState.
type WalkerStackState struct {
	// Node being processed
	Node *GraphNode

	// Dependency creation tasks (started but not yet awaited)
	DependencyTasks []*DependencyFetchTask

	// Current index in DependencyTasks
	Index int

	// OuterEdge for this frame
	OuterEdge *GraphEdge
}

// DependencyFetchTask represents an in-flight dependency fetch operation
type DependencyFetchTask struct {
	// The dependency being fetched
	Dependency PackageDependency

	// Channel for receiving the result (Go equivalent of Task<T>)
	ResultChan chan *DependencyFetchResult

	// Edge information
	InnerEdge *GraphEdge
}

// DependencyFetchResult contains the result of fetching a dependency
type DependencyFetchResult struct {
	Info  *PackageDependencyInfo
	Error error
}
