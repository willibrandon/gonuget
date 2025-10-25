namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents an edge between nodes in the dependency graph.
/// Matches gonuget's GraphEdge structure.
/// </summary>
public sealed class GraphEdgeData
{
    /// <summary>
    /// Package ID of the parent node.
    /// </summary>
    public string ParentPackageId { get; set; } = string.Empty;

    /// <summary>
    /// Version of the parent node.
    /// </summary>
    public string ParentVersion { get; set; } = string.Empty;

    /// <summary>
    /// Dependency that created this edge.
    /// </summary>
    public DependencyData Dependency { get; set; } = null!;
}
