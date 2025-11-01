namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a successfully resolved package with path and categorization.
/// Matches the Go RestoredPackageInfo struct from protocol.go.
/// </summary>
public class RestoredPackageInfo
{
    /// <summary>
    /// NuGet package identifier.
    /// </summary>
    public string PackageId { get; set; } = string.Empty;

    /// <summary>
    /// Resolved version (exact version, not range).
    /// </summary>
    public string Version { get; set; } = string.Empty;

    /// <summary>
    /// Path to package in global packages folder.
    /// </summary>
    public string Path { get; set; } = string.Empty;

    /// <summary>
    /// True if directly referenced in project file, false if transitive.
    /// </summary>
    public bool IsDirect { get; set; }
}
