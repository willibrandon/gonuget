using System;
using System.IO;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Manages test certificates for signing tests.
/// Generates self-signed certificates in memory or exports to files.
/// </summary>
public static class TestCertificates
{
    /// <summary>
    /// Creates a self-signed code signing certificate for testing.
    /// </summary>
    public static X509Certificate2 CreateTestCodeSigningCertificate(
        string subjectName = "CN=Test Code Signing",
        int keySize = 2048,
        int validDays = 365)
    {
        using var rsa = RSA.Create(keySize);

        var request = new CertificateRequest(
            subjectName,
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1);

        // Add code signing extended key usage
        request.CertificateExtensions.Add(
            new X509EnhancedKeyUsageExtension(
                new OidCollection
                {
                    new Oid("1.3.6.1.5.5.7.3.3") // Code signing
                },
                critical: true));

        // Add key usage
        request.CertificateExtensions.Add(
            new X509KeyUsageExtension(
                X509KeyUsageFlags.DigitalSignature,
                critical: true));

        // Add subject key identifier
        request.CertificateExtensions.Add(
            new X509SubjectKeyIdentifierExtension(
                request.PublicKey,
                critical: false));

        var notBefore = DateTimeOffset.UtcNow.AddDays(-1);
        var notAfter = DateTimeOffset.UtcNow.AddDays(validDays);

        var cert = request.CreateSelfSigned(notBefore, notAfter);

        // Export and re-import to ensure private key is included
        var pfxBytes = cert.Export(X509ContentType.Pfx, "test");
        return X509CertificateLoader.LoadPkcs12(pfxBytes, "test", X509KeyStorageFlags.Exportable);
    }

    /// <summary>
    /// Exports a certificate to PEM format (certificate only, no private key).
    /// </summary>
    public static void ExportCertificateToPem(X509Certificate2 cert, string path)
    {
        var pem = PemEncoding.Write("CERTIFICATE".ToCharArray(), cert.RawData);
        File.WriteAllText(path, new string(pem));
    }

    /// <summary>
    /// Exports a certificate's private key to PEM format (PKCS#8).
    /// </summary>
    public static void ExportPrivateKeyToPem(X509Certificate2 cert, string path)
    {
        var rsa = cert.GetRSAPrivateKey()
            ?? throw new InvalidOperationException("Not an RSA certificate");

        var pkcs8 = rsa.ExportPkcs8PrivateKey();
        var pem = PemEncoding.Write("PRIVATE KEY".ToCharArray(), pkcs8);
        File.WriteAllText(path, new string(pem));
    }

    /// <summary>
    /// Exports certificate to PFX format with password.
    /// </summary>
    public static void ExportToPfx(X509Certificate2 cert, string path, string password)
    {
        var pfxBytes = cert.Export(X509ContentType.Pfx, password);
        File.WriteAllBytes(path, pfxBytes);
    }

    /// <summary>
    /// Creates an expired certificate for negative testing.
    /// </summary>
    public static X509Certificate2 CreateExpiredCertificate(string subjectName = "CN=Expired Test Cert")
    {
        using var rsa = RSA.Create(2048);

        var request = new CertificateRequest(
            subjectName,
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1);

        // Certificate expired 30 days ago
        var notBefore = DateTimeOffset.UtcNow.AddDays(-60);
        var notAfter = DateTimeOffset.UtcNow.AddDays(-30);

        var cert = request.CreateSelfSigned(notBefore, notAfter);

        var pfxBytes = cert.Export(X509ContentType.Pfx, "test");
        return X509CertificateLoader.LoadPkcs12(pfxBytes, "test", X509KeyStorageFlags.Exportable);
    }
}
