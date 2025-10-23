namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the validate_cache_file operation indicating whether the cache file is valid.
/// </summary>
public class ValidateCacheFileResponse
{
    /// <summary>
    /// True if the file exists and is within the TTL, false if missing or expired.
    /// </summary>
    public bool Valid { get; set; }
}
