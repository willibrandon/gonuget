namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the resolve_latest_version operation.
/// </summary>
public class ResolveLatestVersionResponse
{
    /// <summary>
    /// The resolved latest version string.
    /// </summary>
    public string Version { get; set; } = "";
}
