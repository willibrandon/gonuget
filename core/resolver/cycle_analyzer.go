package resolver

import (
	"fmt"
	"strings"
)

// CycleAnalyzer provides advanced cycle analysis and reporting
type CycleAnalyzer struct{}

// NewCycleAnalyzer creates a new cycle analyzer
func NewCycleAnalyzer() *CycleAnalyzer {
	return &CycleAnalyzer{}
}

// AnalyzeCycles extracts all cycles from a graph and provides detailed reports
func (ca *CycleAnalyzer) AnalyzeCycles(root *GraphNode) []CycleReport {
	reports := make([]CycleReport, 0)

	// Find all nodes with Disposition.Cycle
	cycleNodes := ca.findCycleNodes(root)

	for _, node := range cycleNodes {
		report := ca.createCycleReport(node)
		if report != nil {
			reports = append(reports, *report)
		}
	}

	return reports
}

// findCycleNodes recursively finds all nodes marked with Disposition.Cycle
func (ca *CycleAnalyzer) findCycleNodes(node *GraphNode) []*GraphNode {
	if node == nil {
		return nil
	}

	nodes := make([]*GraphNode, 0)

	if node.Disposition == DispositionCycle {
		nodes = append(nodes, node)
	}

	for _, child := range node.InnerNodes {
		nodes = append(nodes, ca.findCycleNodes(child)...)
	}

	return nodes
}

// createCycleReport creates a detailed report for a cycle node
func (ca *CycleAnalyzer) createCycleReport(node *GraphNode) *CycleReport {
	if node == nil {
		return nil
	}

	path := node.PathFromRoot()

	// Extract package ID from key
	packageID := ca.extractPackageID(node.Key)

	return &CycleReport{
		PackageID:   packageID,
		PathToSelf:  path,
		Depth:       node.Depth,
		Description: ca.formatCycleDescription(packageID, path),
	}
}

// extractPackageID extracts package ID from node key
func (ca *CycleAnalyzer) extractPackageID(key string) string {
	// Key format: "packageID|versionRange"
	parts := strings.Split(key, "|")
	if len(parts) > 0 {
		return parts[0]
	}
	return key
}

// formatCycleDescription creates a human-readable cycle description
func (ca *CycleAnalyzer) formatCycleDescription(packageID string, path []string) string {
	if len(path) == 0 {
		return fmt.Sprintf("Circular dependency on %s", packageID)
	}

	return fmt.Sprintf("Circular dependency: %s -> ... -> %s",
		strings.Join(path, " -> "), packageID)
}
