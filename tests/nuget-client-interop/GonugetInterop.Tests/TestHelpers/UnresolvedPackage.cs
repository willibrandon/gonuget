using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a package that could not be found or resolved.
/// </summary>
public class UnresolvedPackage
{
    /// <summary>
    /// NuGet package identifier.
    /// </summary>
    public string PackageId { get; set; } = string.Empty;

    /// <summary>
    /// Requested version or version range.
    /// </summary>
    public string VersionRange { get; set; } = string.Empty;

    /// <summary>
    /// Target framework moniker.
    /// </summary>
    public string TargetFramework { get; set; } = string.Empty;

    /// <summary>
    /// NuGet error code (NU1101, NU1102, NU1103).
    /// </summary>
    public string ErrorCode { get; set; } = string.Empty;

    /// <summary>
    /// Full error message.
    /// </summary>
    public string Message { get; set; } = string.Empty;

    /// <summary>
    /// Package sources queried.
    /// </summary>
    public List<string> Sources { get; set; } = new();

    /// <summary>
    /// Versions found (for NU1102/NU1103).
    /// </summary>
    public List<string> AvailableVersions { get; set; } = new();

    /// <summary>
    /// Closest version match (for NU1102/NU1103).
    /// </summary>
    public string NearestVersion { get; set; } = string.Empty;
}
