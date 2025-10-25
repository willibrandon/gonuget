namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response containing the complete transitive dependency closure (M5.6).
/// </summary>
public class ResolveTransitiveResponse
{
    /// <summary>
    /// All resolved packages (roots + transitive dependencies).
    /// </summary>
    public required ResolvedPackage[] Packages { get; set; }

    /// <summary>
    /// Version conflicts detected during resolution.
    /// </summary>
    public required ConflictInfo[] Conflicts { get; set; }

    /// <summary>
    /// Version downgrades detected during resolution.
    /// </summary>
    public required DowngradeInfo[] Downgrades { get; set; }
}
