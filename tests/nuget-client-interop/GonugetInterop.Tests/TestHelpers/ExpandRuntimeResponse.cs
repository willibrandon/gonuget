using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from expanding a runtime identifier (RID) to all compatible RIDs.
/// </summary>
public class ExpandRuntimeResponse
{
    /// <summary>
    /// Gets or sets the array of expanded runtime identifiers in priority order.
    /// The first element is the original RID, followed by compatible RIDs in nearest-first order.
    /// Example: For "win10-x64", returns ["win10-x64", "win10", "win-x64", "win81-x64", ..., "any", "base"].
    /// </summary>
    public string[] ExpandedRuntimes { get; set; } = Array.Empty<string>();
}
