using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the restore_direct_dependencies operation.
/// </summary>
public class RestoreDirectDependenciesResponse
{
    /// <summary>
    /// Indicates whether the restore operation succeeded.
    /// </summary>
    public bool Success { get; set; }

    /// <summary>
    /// Path to the generated project.assets.json lock file.
    /// </summary>
    public string LockFilePath { get; set; } = "";

    /// <summary>
    /// Restore elapsed time in milliseconds.
    /// </summary>
    public long ElapsedMs { get; set; }

    /// <summary>
    /// List of installed packages (format: "PackageId/Version").
    /// </summary>
    public List<string> InstalledPackages { get; set; } = new();
}
