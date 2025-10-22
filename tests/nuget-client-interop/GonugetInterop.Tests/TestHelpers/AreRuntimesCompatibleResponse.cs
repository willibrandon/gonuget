namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from checking runtime identifier (RID) compatibility.
/// </summary>
public class AreRuntimesCompatibleResponse
{
    /// <summary>
    /// Gets or sets a value indicating whether the package RID is compatible with the target RID.
    /// Returns true if a package built for the package RID can run on a system with the target RID.
    /// Example: A package built for "win" can run on "win10-x64" (returns true).
    /// </summary>
    public bool Compatible { get; set; }
}
