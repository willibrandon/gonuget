using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the sign_package operation containing the generated signature.
/// </summary>
public class SignPackageResponse
{
    /// <summary>
    /// The PKCS#7 signature bytes.
    /// </summary>
    public byte[] Signature { get; set; } = Array.Empty<byte>();
}
