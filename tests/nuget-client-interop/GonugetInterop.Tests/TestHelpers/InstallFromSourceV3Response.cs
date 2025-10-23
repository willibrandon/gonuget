namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from gonuget's install_from_source_v3 action.
/// </summary>
public class InstallFromSourceV3Response
{
    /// <summary>
    /// True if package was installed (false if already existed).
    /// </summary>
    public bool Installed { get; set; }

    /// <summary>
    /// Final package directory path.
    /// </summary>
    public string PackageDirectory { get; set; } = string.Empty;

    /// <summary>
    /// Path to the .nupkg file (if saved).
    /// </summary>
    public string? NupkgPath { get; set; }

    /// <summary>
    /// Path to the .nuspec file.
    /// </summary>
    public string NuspecPath { get; set; } = string.Empty;

    /// <summary>
    /// Path to the .sha512 hash file.
    /// </summary>
    public string HashPath { get; set; } = string.Empty;

    /// <summary>
    /// Path to the .nupkg.metadata file.
    /// </summary>
    public string MetadataPath { get; set; } = string.Empty;
}
