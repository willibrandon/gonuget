namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response containing detected circular dependencies (M5.5).
/// </summary>
public class AnalyzeCyclesResponse
{
    /// <summary>
    /// Array of detected circular dependency chains.
    /// </summary>
    public required CycleInfo[] Cycles { get; set; }
}
