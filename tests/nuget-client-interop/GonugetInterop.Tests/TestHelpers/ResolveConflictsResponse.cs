namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the resolve_conflicts action.
/// </summary>
public sealed class ResolveConflictsResponse
{
    /// <summary>
    /// Resolved packages after conflict resolution.
    /// </summary>
    public ResolvedPackage[] Packages { get; set; } = [];
}
