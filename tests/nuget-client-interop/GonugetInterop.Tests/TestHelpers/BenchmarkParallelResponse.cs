namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response containing parallel resolution metrics (M5.8).
/// </summary>
public class BenchmarkParallelResponse
{
    /// <summary>
    /// Number of packages resolved.
    /// </summary>
    public required int PackageCount { get; set; }

    /// <summary>
    /// Total duration in milliseconds.
    /// </summary>
    public required long DurationMs { get; set; }

    /// <summary>
    /// Indicates if parallel resolution was used.
    /// </summary>
    public required bool WasParallel { get; set; }
}
