using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from gonuget restore transitive operation via JSON-RPC bridge.
/// </summary>
public class RestoreTransitiveResponse
{
    /// <summary>
    /// Whether restore completed without errors.
    /// </summary>
    public bool Success { get; set; }

    /// <summary>
    /// Packages explicitly listed in project file.
    /// </summary>
    public List<RestoredPackageInfo> DirectPackages { get; set; } = new();

    /// <summary>
    /// Packages pulled in as dependencies.
    /// </summary>
    public List<RestoredPackageInfo> TransitivePackages { get; set; } = new();

    /// <summary>
    /// Packages that could not be resolved.
    /// </summary>
    public List<UnresolvedPackage> UnresolvedPackages { get; set; } = new();

    /// <summary>
    /// Path to generated project.assets.json lock file.
    /// </summary>
    public string LockFilePath { get; set; } = string.Empty;

    /// <summary>
    /// Restore execution time in milliseconds.
    /// </summary>
    public long ElapsedMs { get; set; }

    /// <summary>
    /// Error messages (NU1101, NU1102, NU1103).
    /// </summary>
    public List<string> ErrorMessages { get; set; } = new();
}
