using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the read_package operation containing package metadata and structure.
/// </summary>
public class ReadPackageResponse
{
    /// <summary>
    /// The package identifier (e.g., "Newtonsoft.Json").
    /// </summary>
    public string Id { get; set; } = "";

    /// <summary>
    /// The package version string (e.g., "13.0.1").
    /// </summary>
    public string Version { get; set; } = "";

    /// <summary>
    /// The package authors as an array of strings.
    /// </summary>
    public string[]? Authors { get; set; }

    /// <summary>
    /// The package description text.
    /// </summary>
    public string? Description { get; set; }

    /// <summary>
    /// Package dependencies formatted as "id:version" strings.
    /// </summary>
    public string[]? Dependencies { get; set; }

    /// <summary>
    /// The total number of files in the package ZIP archive.
    /// </summary>
    public int FileCount { get; set; }

    /// <summary>
    /// Indicates whether the package contains a .signature.p7s file.
    /// </summary>
    public bool HasSignature { get; set; }

    /// <summary>
    /// The signature type if present ("Author", "Repository", or empty if no signature).
    /// </summary>
    public string? SignatureType { get; set; }
}
