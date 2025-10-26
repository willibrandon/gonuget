using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response model for executing a pair of commands (dotnet nuget vs gonuget) and comparing their outputs.
/// </summary>
public class ExecuteCommandPairResponse
{
    /// <summary>
    /// Gets or sets the exit code from the dotnet nuget command.
    /// </summary>
    [JsonPropertyName("dotnetExitCode")]
    public int DotnetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the standard output from the dotnet nuget command.
    /// </summary>
    [JsonPropertyName("dotnetStdOut")]
    public string DotnetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the dotnet nuget command.
    /// </summary>
    [JsonPropertyName("dotnetStdErr")]
    public string DotnetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets a value indicating whether the dotnet nuget command succeeded.
    /// </summary>
    [JsonPropertyName("dotnetSuccess")]
    public bool DotnetSuccess { get; set; }

    /// <summary>
    /// Gets or sets the exit code from the gonuget command.
    /// </summary>
    [JsonPropertyName("gonugetExitCode")]
    public int GonugetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the standard output from the gonuget command.
    /// </summary>
    [JsonPropertyName("gonugetStdOut")]
    public string GonugetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the gonuget command.
    /// </summary>
    [JsonPropertyName("gonugetStdErr")]
    public string GonugetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets a value indicating whether the gonuget command succeeded.
    /// </summary>
    [JsonPropertyName("gonugetSuccess")]
    public bool GonugetSuccess { get; set; }

    /// <summary>
    /// Gets or sets the normalized standard output from the dotnet nuget command.
    /// </summary>
    [JsonPropertyName("normalizedDotnetStdOut")]
    public string NormalizedDotnetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the normalized standard output from the gonuget command.
    /// </summary>
    [JsonPropertyName("normalizedGonugetStdOut")]
    public string NormalizedGonugetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets a value indicating whether the normalized outputs match between dotnet nuget and gonuget.
    /// </summary>
    [JsonPropertyName("outputMatches")]
    public bool OutputMatches { get; set; }
}
