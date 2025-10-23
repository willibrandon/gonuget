namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the sanitize_cache_filename operation containing the sanitized filename.
/// </summary>
public class SanitizeCacheFilenameResponse
{
    /// <summary>
    /// The sanitized filename with invalid chars replaced and collapsed.
    /// </summary>
    public string Sanitized { get; set; } = string.Empty;
}
