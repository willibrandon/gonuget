namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the parse_version operation containing parsed version components.
/// Matches the structure of NuGet.Versioning.NuGetVersion.
/// </summary>
public class ParseVersionResponse
{
    /// <summary>
    /// The major version component.
    /// </summary>
    public int Major { get; set; }

    /// <summary>
    /// The minor version component.
    /// </summary>
    public int Minor { get; set; }

    /// <summary>
    /// The patch (build) version component.
    /// </summary>
    public int Patch { get; set; }

    /// <summary>
    /// The revision version component (for legacy 4-part versions).
    /// </summary>
    public int Revision { get; set; }

    /// <summary>
    /// The pre-release label (e.g., "beta.1", "alpha", "rc.2").
    /// </summary>
    public string Release { get; set; } = "";

    /// <summary>
    /// The build metadata (e.g., "20130313144700").
    /// </summary>
    public string Metadata { get; set; } = "";

    /// <summary>
    /// Indicates whether this is a pre-release version (has a release label).
    /// </summary>
    public bool IsPrerelease { get; set; }

    /// <summary>
    /// Indicates whether this uses the legacy 4-part version format (Major.Minor.Build.Revision).
    /// </summary>
    public bool IsLegacy { get; set; }
}
