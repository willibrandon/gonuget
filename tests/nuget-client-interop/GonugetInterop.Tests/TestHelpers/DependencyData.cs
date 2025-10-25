namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a package dependency.
/// Matches gonuget's PackageDependency structure.
/// </summary>
public sealed class DependencyData
{
    /// <summary>
    /// Package ID of the dependency.
    /// </summary>
    public string Id { get; set; } = string.Empty;

    /// <summary>
    /// Version range for the dependency.
    /// </summary>
    public string VersionRange { get; set; } = string.Empty;

    /// <summary>
    /// Target framework (empty for all frameworks).
    /// </summary>
    public string TargetFramework { get; set; } = string.Empty;
}
