namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a version conflict between dependencies.
/// </summary>
public class ConflictInfo
{
    /// <summary>
    /// Package ID with conflicting versions.
    /// </summary>
    public required string PackageId { get; set; }

    /// <summary>
    /// Conflicting version requirements.
    /// </summary>
    public required string[] Versions { get; set; }

    /// <summary>
    /// Resolved version that was selected.
    /// </summary>
    public required string WinnerVersion { get; set; }
}
