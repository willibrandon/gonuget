namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Information about a detected downgrade.
/// </summary>
public sealed class DowngradeInfo
{
    /// <summary>
    /// Package ID being downgraded.
    /// </summary>
    public string PackageId { get; set; } = string.Empty;

    /// <summary>
    /// Current (higher) version.
    /// </summary>
    public string FromVersion { get; set; } = string.Empty;

    /// <summary>
    /// Target (lower) version causing downgrade.
    /// </summary>
    public string ToVersion { get; set; } = string.Empty;
}
