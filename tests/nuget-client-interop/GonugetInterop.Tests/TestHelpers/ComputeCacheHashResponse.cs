namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the compute_cache_hash operation containing the computed hash.
/// </summary>
public class ComputeCacheHashResponse
{
    /// <summary>
    /// The computed cache hash (40-char hex + optional trailing chars).
    /// </summary>
    public string Hash { get; set; } = string.Empty;
}
