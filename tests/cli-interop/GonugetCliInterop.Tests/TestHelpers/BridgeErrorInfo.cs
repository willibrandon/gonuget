using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Error information returned by the gonuget CLI interop test bridge when a command fails.
/// </summary>
internal class BridgeErrorInfo
{
    /// <summary>
    /// Gets or sets the error code identifying the type of error.
    /// </summary>
    [JsonPropertyName("code")]
    public string Code { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets the error message describing what went wrong.
    /// </summary>
    [JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;

    /// <summary>
    /// Gets or sets additional error details, such as stack traces or output.
    /// </summary>
    [JsonPropertyName("details")]
    public string? Details { get; set; }
}
