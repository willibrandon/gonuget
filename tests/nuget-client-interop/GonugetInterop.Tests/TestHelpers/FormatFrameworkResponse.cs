namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from format_framework bridge action.
/// Contains the formatted short folder name that matches NuGet.Client's GetShortFolderName() output.
/// </summary>
public sealed record FormatFrameworkResponse
{
    /// <summary>
    /// The short folder name representation of the framework.
    /// Examples: "net6.0", "net48", "netstandard2.1", "net6.0-windows10.0", "portable-net45+win8"
    /// </summary>
    public required string ShortFolderName { get; init; }
}
