namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Specifies a package ID and version range for resolution.
/// </summary>
public class PackageSpec
{
    /// <summary>
    /// Package identifier.
    /// </summary>
    public required string Id { get; set; }

    /// <summary>
    /// Version constraint (e.g., "[1.0.0]", "[1.0.0,2.0.0)").
    /// </summary>
    public required string VersionRange { get; set; }
}
