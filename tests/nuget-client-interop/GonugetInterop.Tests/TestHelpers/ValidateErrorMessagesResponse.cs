using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from validating error message format between gonuget and NuGet.Client.
/// </summary>
public class ValidateErrorMessagesResponse
{
    /// <summary>
    /// NuGet error code (NU1101, NU1102, NU1103).
    /// </summary>
    public string ErrorCode { get; set; } = string.Empty;

    /// <summary>
    /// Error message from gonuget.
    /// </summary>
    public string GonugetMessage { get; set; } = string.Empty;

    /// <summary>
    /// Error message from NuGet.Client.
    /// </summary>
    public string NuGetClientMessage { get; set; } = string.Empty;

    /// <summary>
    /// Whether messages match (allowing for formatting tolerance).
    /// </summary>
    public bool Match { get; set; }

    /// <summary>
    /// Specific differences found between the messages.
    /// </summary>
    public List<string> Differences { get; set; } = new();
}
