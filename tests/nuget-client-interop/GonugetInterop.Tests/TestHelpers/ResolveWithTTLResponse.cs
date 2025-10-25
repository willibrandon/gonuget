namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from package resolution with cache TTL (M5.7).
/// </summary>
public class ResolveWithTTLResponse
{
    /// <summary>
    /// Resolved packages.
    /// </summary>
    public required ResolvedPackage[] Packages { get; set; }

    /// <summary>
    /// Indicates if the result came from cache.
    /// </summary>
    public required bool WasCached { get; set; }
}
