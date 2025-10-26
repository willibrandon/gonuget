using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response model for executing 'version' commands on both dotnet nuget and gonuget.
/// </summary>
public class ExecuteVersionResponse
{
    /// <summary>
    /// Gets or sets the exit code from the dotnet nuget version command.
    /// </summary>
    [JsonPropertyName("dotnetExitCode")]
    public int DotnetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the exit code from the gonuget version command.
    /// </summary>
    [JsonPropertyName("gonugetExitCode")]
    public int GonugetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the standard output from the dotnet nuget version command.
    /// </summary>
    [JsonPropertyName("dotnetStdOut")]
    public string DotnetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard output from the gonuget version command.
    /// </summary>
    [JsonPropertyName("gonugetStdOut")]
    public string GonugetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the dotnet nuget version command.
    /// </summary>
    [JsonPropertyName("dotnetStdErr")]
    public string DotnetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the gonuget version command.
    /// </summary>
    [JsonPropertyName("gonugetStdErr")]
    public string GonugetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets a value indicating whether the exit codes match between dotnet nuget and gonuget.
    /// </summary>
    [JsonPropertyName("exitCodesMatch")]
    public bool ExitCodesMatch { get; set; }

    /// <summary>
    /// Gets or sets a value indicating whether the output formats are similar between dotnet nuget and gonuget.
    /// </summary>
    [JsonPropertyName("outputFormatSimilar")]
    public bool OutputFormatSimilar { get; set; }
}
