namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a single package resolution result.
/// </summary>
public class ResolveResult
{
    /// <summary>
    /// Package identifier.
    /// </summary>
    public required string PackageId { get; set; }

    /// <summary>
    /// Resolved packages (root + transitive dependencies).
    /// </summary>
    public required ResolvedPackage[] Packages { get; set; }

    /// <summary>
    /// Error message if resolution failed (empty if successful).
    /// </summary>
    public string? Error { get; set; }
}
