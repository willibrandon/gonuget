using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response model for executing 'help' commands on both dotnet nuget and gonuget.
/// </summary>
public class ExecuteHelpResponse
{
    /// <summary>
    /// Gets or sets the exit code from the dotnet nuget help command.
    /// </summary>
    [JsonPropertyName("dotnetExitCode")]
    public int DotnetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the exit code from the gonuget help command.
    /// </summary>
    [JsonPropertyName("gonugetExitCode")]
    public int GonugetExitCode { get; set; }

    /// <summary>
    /// Gets or sets the standard output from the dotnet nuget help command.
    /// </summary>
    [JsonPropertyName("dotnetStdOut")]
    public string DotnetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard output from the gonuget help command.
    /// </summary>
    [JsonPropertyName("gonugetStdOut")]
    public string GonugetStdOut { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the dotnet nuget help command.
    /// </summary>
    [JsonPropertyName("dotnetStdErr")]
    public string DotnetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the standard error from the gonuget help command.
    /// </summary>
    [JsonPropertyName("gonugetStdErr")]
    public string GonugetStdErr { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets a value indicating whether the exit codes match between dotnet nuget and gonuget.
    /// </summary>
    [JsonPropertyName("exitCodesMatch")]
    public bool ExitCodesMatch { get; set; }

    /// <summary>
    /// Gets or sets a value indicating whether both outputs show a list of commands.
    /// </summary>
    [JsonPropertyName("bothShowCommands")]
    public bool BothShowCommands { get; set; }

    /// <summary>
    /// Gets or sets a value indicating whether both outputs show usage information.
    /// </summary>
    [JsonPropertyName("bothShowUsage")]
    public bool BothShowUsage { get; set; }

    /// <summary>
    /// Gets or sets a value indicating whether the output formats are similar between dotnet nuget and gonuget.
    /// </summary>
    [JsonPropertyName("outputFormatSimilar")]
    public bool OutputFormatSimilar { get; set; }
}
