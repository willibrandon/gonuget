namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// A resolved package after conflict resolution.
/// </summary>
public sealed class ResolvedPackage
{
    /// <summary>
    /// Package ID.
    /// </summary>
    public string PackageId { get; set; } = string.Empty;

    /// <summary>
    /// Selected version after conflict resolution.
    /// </summary>
    public string Version { get; set; } = string.Empty;

    /// <summary>
    /// Depth in the dependency graph (0 = root).
    /// </summary>
    public int Depth { get; set; }
}
