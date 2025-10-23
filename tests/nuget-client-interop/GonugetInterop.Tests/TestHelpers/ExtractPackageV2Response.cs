using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from gonuget's extract_package_v2 action.
/// </summary>
public class ExtractPackageV2Response
{
    /// <summary>
    /// Paths to all extracted files.
    /// </summary>
    public string[] ExtractedFiles { get; set; } = Array.Empty<string>();

    /// <summary>
    /// Number of files extracted.
    /// </summary>
    public int FileCount { get; set; }
}
