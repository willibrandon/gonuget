namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a circular dependency chain.
/// </summary>
public class CycleInfo
{
    /// <summary>
    /// Package IDs forming the cycle (e.g., ["A", "B", "C", "A"]).
    /// </summary>
    public required string[] PackageIds { get; set; }

    /// <summary>
    /// Number of packages in the cycle.
    /// </summary>
    public required int Length { get; set; }
}
