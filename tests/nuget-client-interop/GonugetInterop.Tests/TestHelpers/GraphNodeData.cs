namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a node in the dependency graph.
/// Matches gonuget's GraphNode structure.
/// </summary>
public sealed class GraphNodeData
{
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
    /// Package IDs of dependencies (just IDs, not full node data).
    /// </summary>
    public string[] Dependencies { get; set; } = [];
}
