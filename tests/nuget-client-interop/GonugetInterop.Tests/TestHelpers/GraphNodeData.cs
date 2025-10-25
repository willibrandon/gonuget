using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a node in the dependency graph.
/// Matches gonuget's GraphNode structure.
/// </summary>
public sealed class GraphNodeData
{
    /// <summary>
    /// Unique key for this node (packageID|version).
    /// </summary>
    public string Key { get; set; } = string.Empty;

    /// <summary>
    /// Package ID.
    /// </summary>
    public string PackageId { get; set; } = string.Empty;

    /// <summary>
    /// Package version.
    /// </summary>
    public string Version { get; set; } = string.Empty;

    /// <summary>
    /// Node disposition state: Acceptable, Rejected, Accepted, PotentiallyDowngraded, or Cycle.
    /// </summary>
    public string Disposition { get; set; } = string.Empty;

    /// <summary>
    /// Distance from root node (0 for root).
    /// </summary>
    public int Depth { get; set; }

    /// <summary>
    /// Child nodes (dependencies).
    /// </summary>
    public List<GraphNodeData> InnerNodes { get; set; } = new();

    /// <summary>
    /// Edge from parent to this node (null for root).
    /// </summary>
    public GraphEdgeData? OuterEdge { get; set; }
}
