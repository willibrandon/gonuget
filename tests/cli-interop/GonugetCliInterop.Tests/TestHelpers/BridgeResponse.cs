using System.Text.Json.Serialization;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Response wrapper for JSON-RPC communication with the gonuget CLI interop test bridge.
/// </summary>
/// <typeparam name="T">The type of data returned in the response.</typeparam>
internal class BridgeResponse<T>
{
    /// <summary>
    /// Gets or sets a value indicating whether the command execution was successful.
    /// </summary>
    [JsonPropertyName("success")]
    public bool Success { get; set; }

    /// <summary>
    /// Gets or sets the response data when the command succeeds.
    /// </summary>
    [JsonPropertyName("data")]
    public T? Data { get; set; }

    /// <summary>
    /// Gets or sets the error information when the command fails.
    /// </summary>
    [JsonPropertyName("error")]
    public BridgeErrorInfo? Error { get; set; }
}
