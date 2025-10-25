namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the walk_graph action containing the dependency graph.
/// </summary>
public sealed class WalkGraphResponse
{
    /// <summary>
    /// All nodes in the graph as a flat array.
    /// </summary>
    public GraphNodeData[] Nodes { get; set; } = [];

    /// <summary>
    /// Package IDs that created cycles (e.g., "A -> B -> A").
    /// </summary>
    public string[] Cycles { get; set; } = [];

    /// <summary>
    /// Detected downgrade warnings.
    /// </summary>
    public DowngradeInfo[] Downgrades { get; set; } = [];
}
