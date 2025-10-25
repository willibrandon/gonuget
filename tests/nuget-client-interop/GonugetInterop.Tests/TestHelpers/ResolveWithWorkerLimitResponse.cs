namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from parallel resolution with worker pool limits (M5.8).
/// </summary>
public class ResolveWithWorkerLimitResponse
{
    /// <summary>
    /// Resolution results for each package.
    /// </summary>
    public required ResolveResult[] Results { get; set; }

    /// <summary>
    /// Maximum concurrent operations observed.
    /// </summary>
    public required int MaxConcurrent { get; set; }
}
