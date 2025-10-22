using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the build_package operation containing the created package.
/// </summary>
public class BuildPackageResponse
{
    /// <summary>
    /// The complete NuGet package as a byte array (ZIP format).
    /// </summary>
    public byte[] PackageBytes { get; set; } = Array.Empty<byte>();
}
