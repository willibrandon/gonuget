using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from comparing two project.assets.json files (gonuget vs NuGet.Client).
/// </summary>
public class CompareProjectAssetsResponse
{
    /// <summary>
    /// Overall equality result - true if all fields match.
    /// </summary>
    public bool AreEqual { get; set; }

    /// <summary>
    /// Libraries map keys and structure match between gonuget and NuGet.Client.
    /// </summary>
    public bool LibrariesMatch { get; set; }

    /// <summary>
    /// Direct dependency groups match (ProjectFileDependencyGroups section).
    /// </summary>
    public bool ProjectFileDependencyGroupsMatch { get; set; }

    /// <summary>
    /// Package versions match across all libraries.
    /// </summary>
    public bool VersionsMatch { get; set; }

    /// <summary>
    /// Package paths match (including lowercase requirement).
    /// </summary>
    public bool PathsMatch { get; set; }

    /// <summary>
    /// Human-readable list of differences found during comparison.
    /// </summary>
    public List<string> Differences { get; set; } = new();
}
