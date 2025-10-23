namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the generate_cache_paths operation containing the generated cache paths.
/// </summary>
public class GenerateCachePathsResponse
{
    /// <summary>
    /// The hash-based folder name.
    /// </summary>
    public string BaseFolderName { get; set; } = string.Empty;

    /// <summary>
    /// The full path to the cache file.
    /// </summary>
    public string CacheFile { get; set; } = string.Empty;

    /// <summary>
    /// The full path to the temporary file during atomic writes.
    /// </summary>
    public string NewFile { get; set; } = string.Empty;
}
