namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the parse_framework operation containing parsed framework components.
/// </summary>
public class ParseFrameworkResponse
{
    /// <summary>
    /// The framework identifier (e.g., ".NETCoreApp", ".NETFramework", ".NETStandard").
    /// </summary>
    public string Identifier { get; set; } = "";

    /// <summary>
    /// The framework version as a string (e.g., "6.0", "4.7.2", "2.1").
    /// </summary>
    public string Version { get; set; } = "";

    /// <summary>
    /// Optional profile name (e.g., "Client" for .NETFramework profiles).
    /// </summary>
    public string? Profile { get; set; }

    /// <summary>
    /// Optional platform specifier (e.g., "windows7.0" for platform-specific TFMs).
    /// </summary>
    public string? Platform { get; set; }
}
