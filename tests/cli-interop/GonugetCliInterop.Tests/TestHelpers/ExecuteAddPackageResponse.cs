using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response from executing the 'add package' command on both dotnet and gonuget.
/// </summary>
public class ExecuteAddPackageResponse
{
    /// <summary>
    /// Exit code from dotnet add package.
    /// </summary>
    [JsonPropertyName("dotnetExitCode")]
    public int DotnetExitCode { get; set; }

    /// <summary>
    /// Standard output from dotnet add package.
    /// </summary>
    [JsonPropertyName("dotnetStdOut")]
    public string DotnetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Standard error from dotnet add package.
    /// </summary>
    [JsonPropertyName("dotnetStdErr")]
    public string DotnetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Exit code from gonuget add package.
    /// </summary>
    [JsonPropertyName("gonugetExitCode")]
    public int GonugetExitCode { get; set; }

    /// <summary>
    /// Standard output from gonuget add package.
    /// </summary>
    [JsonPropertyName("gonugetStdOut")]
    public string GonugetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Standard error from gonuget add package.
    /// </summary>
    [JsonPropertyName("gonugetStdErr")]
    public string GonugetStdErr { get; set; } = string.Empty;
}
