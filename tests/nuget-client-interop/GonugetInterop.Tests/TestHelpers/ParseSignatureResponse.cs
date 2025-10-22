namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the parse_signature operation containing signature metadata.
/// </summary>
public class ParseSignatureResponse
{
    /// <summary>
    /// The signature type (e.g., "Author", "Repository").
    /// </summary>
    public string Type { get; set; } = "";

    /// <summary>
    /// The hash algorithm used (e.g., "SHA256", "SHA512").
    /// </summary>
    public string HashAlgorithm { get; set; } = "";

    /// <summary>
    /// The hex-encoded hash of the signer certificate.
    /// </summary>
    public string SignerCertHash { get; set; } = "";

    /// <summary>
    /// The number of timestamps in the signature.
    /// </summary>
    public int TimestampCount { get; set; }

    /// <summary>
    /// The timestamp times in RFC3339 format.
    /// </summary>
    public string[]? TimestampTimes { get; set; }

    /// <summary>
    /// The total number of certificates in the signature.
    /// </summary>
    public int Certificates { get; set; }
}
