namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response containing cache deduplication metrics (M5.7).
/// </summary>
public class BenchmarkCacheResponse
{
    /// <summary>
    /// Total number of requests made.
    /// </summary>
    public required int TotalRequests { get; set; }

    /// <summary>
    /// Number of actual fetches performed (should be 1 with deduplication).
    /// </summary>
    public required int ActualFetches { get; set; }

    /// <summary>
    /// Total duration in milliseconds.
    /// </summary>
    public required long DurationMs { get; set; }

    /// <summary>
    /// True if ActualFetches equals 1 (deduplication worked).
    /// </summary>
    public required bool DeduplicationWorked { get; set; }
}
