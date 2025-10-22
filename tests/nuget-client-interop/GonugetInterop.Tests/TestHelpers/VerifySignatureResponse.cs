namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the verify_signature operation containing validation results.
/// </summary>
public class VerifySignatureResponse
{
    /// <summary>
    /// Indicates whether the signature is valid.
    /// </summary>
    public bool Valid { get; set; }

    /// <summary>
    /// Validation errors that caused the signature to be invalid.
    /// </summary>
    public string[]? Errors { get; set; }

    /// <summary>
    /// Validation warnings that don't prevent the signature from being valid.
    /// </summary>
    public string[]? Warnings { get; set; }

    /// <summary>
    /// The subject (Distinguished Name) of the signer certificate.
    /// </summary>
    public string? SignerSubject { get; set; }
}
