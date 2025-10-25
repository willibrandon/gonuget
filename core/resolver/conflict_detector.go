package resolver

// ConflictDetector detects version conflicts in dependency graphs.
// Operates during and after traversal (inline + post-processing).
type ConflictDetector struct{}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectFromGraph analyzes a completed graph for conflicts and downgrades.
func (cd *ConflictDetector) DetectFromGraph(root *GraphNode) ([]VersionConflict, []DowngradeWarning) {
	conflicts := make([]VersionConflict, 0)
	downgrades := make([]DowngradeWarning, 0)

	// Collect all nodes by package ID
	nodesByID := make(map[string][]*GraphNode)
	cd.collectNodes(root, nodesByID)

	// Find conflicts (multiple versions of same package)
	for packageID, nodes := range nodesByID {
		if len(nodes) <= 1 {
			continue
		}

		// Multiple versions - conflict
		versions := make([]string, 0, len(nodes))
		paths := make([][]string, 0, len(nodes))

		for _, node := range nodes {
			if node.Item != nil {
				versions = append(versions, node.Item.Version)
				paths = append(paths, node.PathFromRoot())
			}
		}

		if len(versions) > 1 {
			conflicts = append(conflicts, VersionConflict{
				PackageID: packageID,
				Versions:  versions,
				Paths:     paths,
			})
		}
	}

	// Find downgrades (nodes marked DispositionPotentiallyDowngraded)
	cd.collectDowngrades(root, &downgrades)

	return conflicts, downgrades
}

// collectNodes recursively collects all nodes by package ID.
func (cd *ConflictDetector) collectNodes(node *GraphNode, nodesByID map[string][]*GraphNode) {
	if node == nil {
		return
	}

	if node.Item != nil {
		nodesByID[node.Item.ID] = append(nodesByID[node.Item.ID], node)
	}

	for _, child := range node.InnerNodes {
		cd.collectNodes(child, nodesByID)
	}
}

// collectDowngrades finds all downgrade warnings.
func (cd *ConflictDetector) collectDowngrades(node *GraphNode, downgrades *[]DowngradeWarning) {
	if node == nil {
		return
	}

	if node.Disposition == DispositionPotentiallyDowngraded && node.Item != nil {
		// Find what version it would downgrade from
		// This requires walking parent chain to find existing version
		*downgrades = append(*downgrades, DowngradeWarning{
			PackageID:      node.Item.ID,
			TargetVersion:  node.Item.Version,
			CurrentVersion: "", // Would need parent chain analysis
			Path:           node.PathFromRoot(),
		})
	}

	for _, child := range node.InnerNodes {
		cd.collectDowngrades(child, downgrades)
	}
}
