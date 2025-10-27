using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response from executing both dotnet restore and gonuget restore for comparison.
/// </summary>
public class ExecuteRestoreResponse
{
    [JsonPropertyName("dotnetExitCode")]
    public int DotnetExitCode { get; set; }

    [JsonPropertyName("dotnetStdOut")]
    public string DotnetStdOut { get; set; } = string.Empty;

    [JsonPropertyName("dotnetStdErr")]
    public string DotnetStdErr { get; set; } = string.Empty;

    [JsonPropertyName("gonugetExitCode")]
    public int GonugetExitCode { get; set; }

    [JsonPropertyName("gonugetStdOut")]
    public string GonugetStdOut { get; set; } = string.Empty;

    [JsonPropertyName("gonugetStdErr")]
    public string GonugetStdErr { get; set; } = string.Empty;
}
