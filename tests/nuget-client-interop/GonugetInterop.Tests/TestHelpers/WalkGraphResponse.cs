namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the walk_graph action containing the dependency graph.
/// </summary>
public sealed class WalkGraphResponse
{
    /// <summary>
    /// Root node of the dependency graph.
    /// </summary>
    public GraphNodeData RootNode { get; set; } = null!;
}
