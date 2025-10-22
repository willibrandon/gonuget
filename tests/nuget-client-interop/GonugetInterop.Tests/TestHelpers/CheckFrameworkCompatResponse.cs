namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the check_framework_compat operation containing compatibility result.
/// </summary>
public class CheckFrameworkCompatResponse
{
    /// <summary>
    /// Indicates whether the package framework is compatible with the project framework.
    /// True means the project can use the package.
    /// </summary>
    public bool Compatible { get; set; }

    /// <summary>
    /// The reason for incompatibility. Empty if compatible.
    /// </summary>
    public string? Reason { get; set; }
}
