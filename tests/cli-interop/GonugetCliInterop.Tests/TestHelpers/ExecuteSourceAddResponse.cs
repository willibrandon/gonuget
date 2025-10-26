using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response model for executing 'add source' commands on both dotnet nuget and gonuget.
/// </summary>
public class ExecuteSourceAddResponse
{
    /// <summary>
    /// Gets or sets the exit code from the dotnet nuget add source command.
    /// </summary>
    [JsonPropertyName("dotnetExitCode")]
    public int DotnetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the exit code from the gonuget add source command.
    /// </summary>
    [JsonPropertyName("gonugetExitCode")]
    public int GonugetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the standard output from the dotnet nuget add source command.
    /// </summary>
    [JsonPropertyName("dotnetStdOut")]
    public string DotnetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard output from the gonuget add source command.
    /// </summary>
    [JsonPropertyName("gonugetStdOut")]
    public string GonugetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the dotnet nuget add source command.
    /// </summary>
    [JsonPropertyName("dotnetStdErr")]
    public string DotnetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the gonuget add source command.
    /// </summary>
    [JsonPropertyName("gonugetStdErr")]
    public string GonugetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets a value indicating whether the outputs match between dotnet nuget and gonuget.
    /// </summary>
    [JsonPropertyName("outputMatches")]
    public bool OutputMatches { get; set; }
}
