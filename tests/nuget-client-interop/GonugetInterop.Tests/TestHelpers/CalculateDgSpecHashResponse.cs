namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the calculate_dgspec_hash operation containing the computed dgSpecHash.
/// </summary>
public class CalculateDgSpecHashResponse
{
    /// <summary>
    /// The base64-encoded FNV-1a hash of the dependency graph specification.
    /// </summary>
    public string Hash { get; set; } = string.Empty;
}
