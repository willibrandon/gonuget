namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the verify_project_cache_file operation containing cache validation results.
/// </summary>
public class VerifyProjectCacheFileResponse
{
    /// <summary>
    /// True if the cache file exists and dgSpecHash matches.
    /// </summary>
    public bool Valid { get; set; }

    /// <summary>
    /// The cache file format version.
    /// </summary>
    public int Version { get; set; }

    /// <summary>
    /// The hash stored in the cache file.
    /// </summary>
    public string DgSpecHash { get; set; } = string.Empty;

    /// <summary>
    /// Indicates whether the restore was successful.
    /// </summary>
    public bool Success { get; set; }

    /// <summary>
    /// The project path stored in the cache.
    /// </summary>
    public string ProjectFilePath { get; set; } = string.Empty;

    /// <summary>
    /// The number of expected package files.
    /// </summary>
    public int ExpectedPackageFilesCount { get; set; }
}
