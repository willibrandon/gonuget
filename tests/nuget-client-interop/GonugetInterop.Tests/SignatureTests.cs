using System;
using System.IO;
using System.Linq;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Packaging.Signing;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests signature interop between gonuget and NuGet.Client.
/// Validates that gonuget-created signatures can be verified by NuGet.Client
/// and vice versa.
/// </summary>
public class SignatureTests : IDisposable
{
    private readonly string _tempDir;
    private readonly X509Certificate2 _testCert;
    private readonly string _certPath;
    private readonly string _keyPath;
    private readonly string _pfxPath;

    public SignatureTests()
    {
        // Create temp directory for test artifacts
        _tempDir = Path.Combine(Path.GetTempPath(), $"gonuget-interop-{Guid.NewGuid()}");
        Directory.CreateDirectory(_tempDir);

        // Create test certificate
        _testCert = TestCertificates.CreateTestCodeSigningCertificate("CN=Gonuget Test Signing");

        // Export certificate and key for gonuget
        _certPath = Path.Combine(_tempDir, "test-cert.pem");
        _keyPath = Path.Combine(_tempDir, "test-key.pem");
        _pfxPath = Path.Combine(_tempDir, "test-cert.pfx");

        TestCertificates.ExportCertificateToPem(_testCert, _certPath);
        TestCertificates.ExportPrivateKeyToPem(_testCert, _keyPath);
        TestCertificates.ExportToPfx(_testCert, _pfxPath, "test");
    }

    public void Dispose()
    {
        _testCert?.Dispose();
        if (Directory.Exists(_tempDir))
        {
            Directory.Delete(_tempDir, recursive: true);
        }
    }

    #region Gonuget → NuGet.Client Signature Creation Tests

    [Fact]
    public void GonugetAuthorSignature_SHA256_CreatesValidSignature()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act - Create signature with gonuget
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Assert - Verify it's valid PKCS#7
        Assert.NotEmpty(signature);
        Assert.Equal(0x30, signature[0]); // SEQUENCE tag (PKCS#7 starts with SEQUENCE)
    }

    [Fact]
    public void GonugetAuthorSignature_SHA384_CreatesValidSignature()
    {
        // Arrange
        var packageHash = SHA384.HashData("test package content"u8.ToArray());

        // Act
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA384");

        // Assert
        Assert.NotEmpty(signature);
        Assert.Equal(0x30, signature[0]);
    }

    [Fact]
    public void GonugetAuthorSignature_SHA512_CreatesValidSignature()
    {
        // Arrange
        var packageHash = SHA512.HashData("test package content"u8.ToArray());

        // Act
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA512");

        // Assert
        Assert.NotEmpty(signature);
        Assert.Equal(0x30, signature[0]);
    }

    [Fact]
    public void GonugetRepositorySignature_CreatesValidSignature()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Repository",
            hashAlgorithm: "SHA256");

        // Assert
        Assert.NotEmpty(signature);
        Assert.Equal(0x30, signature[0]);
    }

    [Fact]
    public void GonugetSignature_WithPFX_CreatesValidSignature()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act - Use PFX instead of separate cert/key
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _pfxPath,
            certPassword: "test",
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Assert
        Assert.NotEmpty(signature);
        Assert.Equal(0x30, signature[0]);
    }

    [Fact]
    public void GonugetSignature_EmptyPackageHash_ThrowsError()
    {
        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.SignPackage(
                Array.Empty<byte>(),
                _certPath,
                keyPath: _keyPath,
                signatureType: "Author",
                hashAlgorithm: "SHA256"));

        Assert.Equal("SIGN_001", exception.Code);
    }

    [Fact]
    public void GonugetSignature_MissingCertFile_ThrowsError()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.SignPackage(
                packageHash,
                "/nonexistent/cert.pem",
                keyPath: _keyPath,
                signatureType: "Author",
                hashAlgorithm: "SHA256"));

        Assert.Equal("SIGN_001", exception.Code);
    }

    [Fact]
    public void GonugetSignature_InvalidSignatureType_ThrowsError()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.SignPackage(
                packageHash,
                _certPath,
                keyPath: _keyPath,
                signatureType: "Invalid",
                hashAlgorithm: "SHA256"));

        Assert.Equal("SIGN_001", exception.Code);
        Assert.Contains("signatureType", exception.Message);
    }

    [Fact]
    public void GonugetSignature_InvalidHashAlgorithm_ThrowsError()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.SignPackage(
                packageHash,
                _certPath,
                keyPath: _keyPath,
                signatureType: "Author",
                hashAlgorithm: "MD5"));

        Assert.Equal("SIGN_001", exception.Code);
        Assert.Contains("hashAlgorithm", exception.Message);
    }

    #endregion

    #region Gonuget Signature Parsing Tests

    [Fact]
    public void GonugetParseSignature_AuthorSignature_ParsesCorrectly()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.Equal("Author", parsed.Type);
        Assert.Equal("SHA256", parsed.HashAlgorithm);
        Assert.True(parsed.Certificates > 0);
        Assert.NotEmpty(parsed.SignerCertHash);
    }

    [Fact]
    public void GonugetParseSignature_RepositorySignature_ParsesCorrectly()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Repository",
            hashAlgorithm: "SHA256");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.Equal("Repository", parsed.Type);
        Assert.Equal("SHA256", parsed.HashAlgorithm);
    }

    [Fact]
    public void GonugetParseSignature_SHA384Signature_ParsesHashAlgorithm()
    {
        // Arrange
        var packageHash = SHA384.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA384");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.Equal("SHA384", parsed.HashAlgorithm);
    }

    [Fact]
    public void GonugetParseSignature_SHA512Signature_ParsesHashAlgorithm()
    {
        // Arrange
        var packageHash = SHA512.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA512");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.Equal("SHA512", parsed.HashAlgorithm);
    }

    [Fact]
    public void GonugetParseSignature_ExtractsCertificateCount()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.Equal(1, parsed.Certificates); // Only signer cert, no chain
    }

    [Fact]
    public void GonugetParseSignature_NoTimestamp_HasZeroTimestamps()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.Equal(0, parsed.TimestampCount);
        Assert.True(parsed.TimestampTimes == null || parsed.TimestampTimes.Length == 0);
    }

    [Fact]
    public void GonugetParseSignature_EmptySignature_ThrowsError()
    {
        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.ParseSignature(Array.Empty<byte>()));

        Assert.Equal("PARSE_001", exception.Code);
        Assert.Contains("signature is required", exception.Message);
    }

    [Fact]
    public void GonugetParseSignature_InvalidSignature_ThrowsError()
    {
        // Arrange - Invalid PKCS#7 data
        var invalidSignature = new byte[] { 0xFF, 0xFF, 0xFF, 0xFF };

        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.ParseSignature(invalidSignature));

        Assert.Equal("PARSE_001", exception.Code);
    }

    #endregion

    #region Gonuget Signature Verification Tests

    [Fact]
    public void GonugetVerifySignature_ValidSignature_AllowUntrusted_Succeeds()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var result = GonugetBridge.VerifySignature(
            signature,
            allowUntrustedRoot: true,
            requireTimestamp: false);

        // Assert
        Assert.True(result.Valid);
        Assert.True(result.Errors == null || result.Errors.Length == 0);
        Assert.Contains("Gonuget Test Signing", result.SignerSubject);
    }

    [Fact]
    public void GonugetVerifySignature_WithTrustedRoot_Succeeds()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        var trustedRoots = new[] { _testCert.RawData };

        // Act
        var result = GonugetBridge.VerifySignature(
            signature,
            trustedRoots: trustedRoots,
            allowUntrustedRoot: false,
            requireTimestamp: false);

        // Assert
        Assert.True(result.Valid);
    }

    [Fact]
    public void GonugetVerifySignature_UntrustedRoot_RequireTrust_Fails()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act - Don't provide trusted roots and require trust
        var result = GonugetBridge.VerifySignature(
            signature,
            allowUntrustedRoot: false,
            requireTimestamp: false);

        // Assert
        Assert.False(result.Valid);
        Assert.NotNull(result.Errors);
        Assert.NotEmpty(result.Errors);
    }

    [Fact]
    public void GonugetVerifySignature_RequireTimestamp_NoTimestamp_Fails()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var result = GonugetBridge.VerifySignature(
            signature,
            allowUntrustedRoot: true,
            requireTimestamp: true);

        // Assert
        Assert.False(result.Valid);
        Assert.NotNull(result.Errors);
        Assert.Contains(result.Errors, e => e.Contains("timestamp"));
    }

    [Fact]
    public void GonugetVerifySignature_EmptySignature_ThrowsError()
    {
        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.VerifySignature(
                Array.Empty<byte>(),
                allowUntrustedRoot: true));

        Assert.Equal("VERIFY_001", exception.Code);
    }

    [Fact]
    public void GonugetVerifySignature_InvalidSignature_ThrowsError()
    {
        // Arrange
        var invalidSignature = new byte[] { 0xFF, 0xFF, 0xFF, 0xFF };

        // Act & Assert
        var exception = Assert.Throws<GonugetException>(() =>
            GonugetBridge.VerifySignature(
                invalidSignature,
                allowUntrustedRoot: true));

        Assert.Equal("VERIFY_001", exception.Code);
    }

    #endregion

    #region Round-Trip Tests

    [Fact]
    public void RoundTrip_SignParseVerify_PreservesMetadata()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act - Sign
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act - Parse
        var parsed = GonugetBridge.ParseSignature(signature);

        // Act - Verify
        var verified = GonugetBridge.VerifySignature(
            signature,
            allowUntrustedRoot: true);

        // Assert
        Assert.Equal("Author", parsed.Type);
        Assert.Equal("SHA256", parsed.HashAlgorithm);
        Assert.True(verified.Valid);
    }

    [Fact]
    public void RoundTrip_MultipleHashAlgorithms_AllWork()
    {
        // Test all three hash algorithms in sequence
        foreach (var (hashAlgo, hashFunc) in new[]
        {
            ("SHA256", (Func<byte[], byte[]>)(data => SHA256.HashData(data))),
            ("SHA384", (Func<byte[], byte[]>)(data => SHA384.HashData(data))),
            ("SHA512", (Func<byte[], byte[]>)(data => SHA512.HashData(data)))
        })
        {
            // Arrange
            var packageHash = hashFunc("test package content"u8.ToArray());

            // Act
            var signature = GonugetBridge.SignPackage(
                packageHash,
                _certPath,
                keyPath: _keyPath,
                signatureType: "Author",
                hashAlgorithm: hashAlgo);

            var parsed = GonugetBridge.ParseSignature(signature);
            var verified = GonugetBridge.VerifySignature(signature, allowUntrustedRoot: true);

            // Assert
            Assert.Equal(hashAlgo, parsed.HashAlgorithm);
            Assert.True(verified.Valid);
        }
    }

    [Fact]
    public void RoundTrip_AuthorAndRepository_BothWork()
    {
        // Test both signature types
        foreach (var sigType in new[] { "Author", "Repository" })
        {
            // Arrange
            var packageHash = SHA256.HashData("test package content"u8.ToArray());

            // Act
            var signature = GonugetBridge.SignPackage(
                packageHash,
                _certPath,
                keyPath: _keyPath,
                signatureType: sigType,
                hashAlgorithm: "SHA256");

            var parsed = GonugetBridge.ParseSignature(signature);

            // Assert
            Assert.Equal(sigType, parsed.Type);
        }
    }

    [Fact]
    public void RoundTrip_PEMAndPFX_BothWork()
    {
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Test PEM (cert + key)
        var sigPem = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Test PFX
        var sigPfx = GonugetBridge.SignPackage(
            packageHash,
            _pfxPath,
            certPassword: "test",
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Both should be valid
        var parsedPem = GonugetBridge.ParseSignature(sigPem);
        var parsedPfx = GonugetBridge.ParseSignature(sigPfx);

        Assert.Equal("Author", parsedPem.Type);
        Assert.Equal("Author", parsedPfx.Type);
    }

    #endregion

    #region Certificate Tests

    [Fact]
    public void GonugetSignature_ExtractsSignerCertHash()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var parsed = GonugetBridge.ParseSignature(signature);

        // Assert
        Assert.NotEmpty(parsed.SignerCertHash);
        Assert.Equal(40, parsed.SignerCertHash.Length); // SHA-1 hash is 20 bytes = 40 hex chars
    }

    [Fact]
    public void GonugetSignature_SameInput_SameSignerCertHash()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());

        // Act - Sign twice with same certificate
        var sig1 = GonugetBridge.SignPackage(packageHash, _certPath, keyPath: _keyPath, signatureType: "Author", hashAlgorithm: "SHA256");
        var sig2 = GonugetBridge.SignPackage(packageHash, _certPath, keyPath: _keyPath, signatureType: "Author", hashAlgorithm: "SHA256");

        var parsed1 = GonugetBridge.ParseSignature(sig1);
        var parsed2 = GonugetBridge.ParseSignature(sig2);

        // Assert - Cert hash should be identical
        Assert.Equal(parsed2.SignerCertHash, parsed1.SignerCertHash);
    }

    [Fact]
    public void GonugetSignature_VerificationIncludesSignerSubject()
    {
        // Arrange
        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var result = GonugetBridge.VerifySignature(signature, allowUntrustedRoot: true);

        // Assert
        Assert.NotNull(result.SignerSubject);
        Assert.NotEmpty(result.SignerSubject);
        Assert.Contains("CN=Gonuget Test Signing", result.SignerSubject);
    }

    #endregion

    #region Edge Cases and Error Handling

    [Fact]
    public void GonugetSignature_VeryLargePackageHash_Works()
    {
        // Arrange - 10MB of data hashed
        var largeData = new byte[10 * 1024 * 1024];
        Random.Shared.NextBytes(largeData);
        var packageHash = SHA256.HashData(largeData);

        // Act
        var signature = GonugetBridge.SignPackage(
            packageHash,
            _certPath,
            keyPath: _keyPath,
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Assert
        Assert.NotEmpty(signature);
        var parsed = GonugetBridge.ParseSignature(signature);
        Assert.Equal("Author", parsed.Type);
    }

    [Fact]
    public void GonugetSignature_DifferentPackageHashes_DifferentSignatures()
    {
        // Arrange
        var hash1 = SHA256.HashData("content 1"u8.ToArray());
        var hash2 = SHA256.HashData("content 2"u8.ToArray());

        // Act
        var sig1 = GonugetBridge.SignPackage(hash1, _certPath, keyPath: _keyPath, signatureType: "Author", hashAlgorithm: "SHA256");
        var sig2 = GonugetBridge.SignPackage(hash2, _certPath, keyPath: _keyPath, signatureType: "Author", hashAlgorithm: "SHA256");

        // Assert - Signatures should be different (different content hashes)
        Assert.NotEqual(sig2, sig1);
    }

    [Fact]
    public void GonugetVerify_SignatureWithWarnings_ReturnsWarnings()
    {
        // Arrange - Create expired certificate
        var expiredCert = TestCertificates.CreateExpiredCertificate();
        var expiredCertPath = Path.Combine(_tempDir, "expired-cert.pfx");
        TestCertificates.ExportToPfx(expiredCert, expiredCertPath, "test");

        var packageHash = SHA256.HashData("test package content"u8.ToArray());
        var signature = GonugetBridge.SignPackage(
            packageHash,
            expiredCertPath,
            certPassword: "test",
            signatureType: "Author",
            hashAlgorithm: "SHA256");

        // Act
        var result = GonugetBridge.VerifySignature(signature, allowUntrustedRoot: true);

        // Assert - Should have warnings about expired certificate
        Assert.NotNull(result.Warnings);
        Assert.NotEmpty(result.Warnings);
    }

    #endregion
}
