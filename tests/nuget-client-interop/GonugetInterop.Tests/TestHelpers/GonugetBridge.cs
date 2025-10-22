using System;
using System.Diagnostics;
using System.IO;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Bridge to communicate with the gonuget CLI bridge process.
/// Sends JSON requests via stdin and receives JSON responses via stdout.
/// </summary>
public static class GonugetBridge
{
    private static readonly string GonugetPath = FindGonugetExecutable();

    /// <summary>
    /// Signs a package hash using gonuget and returns the PKCS#7 signature.
    /// </summary>
    public static byte[] SignPackage(
        byte[] packageHash,
        string certPath,
        string? certPassword = null,
        string? keyPath = null,
        string signatureType = "Author",
        string hashAlgorithm = "SHA256",
        string? timestampURL = null)
    {
        var request = new
        {
            action = "sign_package",
            data = new
            {
                packageHash,
                certPath,
                certPassword,
                keyPath,
                signatureType,
                hashAlgorithm,
                timestampURL
            }
        };

        var response = Execute<SignPackageResponse>(request);
        return response.Signature;
    }

    /// <summary>
    /// Parses a signature using gonuget and returns metadata.
    /// </summary>
    public static ParseSignatureResponse ParseSignature(byte[] signature)
    {
        var request = new
        {
            action = "parse_signature",
            data = new { signature }
        };

        return Execute<ParseSignatureResponse>(request);
    }

    /// <summary>
    /// Verifies a signature using gonuget.
    /// </summary>
    public static VerifySignatureResponse VerifySignature(
        byte[] signature,
        byte[][]? trustedRoots = null,
        bool allowUntrustedRoot = false,
        bool requireTimestamp = false)
    {
        var request = new
        {
            action = "verify_signature",
            data = new
            {
                signature,
                trustedRoots = trustedRoots ?? Array.Empty<byte[]>(),
                allowUntrustedRoot,
                requireTimestamp
            }
        };

        return Execute<VerifySignatureResponse>(request);
    }

    /// <summary>
    /// Executes a request against the gonuget CLI and deserializes the response.
    /// </summary>
    private static TResponse Execute<TResponse>(object request)
    {
        var psi = new ProcessStartInfo
        {
            FileName = GonugetPath,
            RedirectStandardInput = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true
        };

        using var process = Process.Start(psi)
            ?? throw new InvalidOperationException("Failed to start gonuget process");

        // Send request as JSON
        var requestJson = JsonSerializer.Serialize(request, new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        });

        process.StandardInput.WriteLine(requestJson);
        process.StandardInput.Close();

        // Read response
        var outputJson = process.StandardOutput.ReadToEnd();
        var errorOutput = process.StandardError.ReadToEnd();

        process.WaitForExit(30000); // 30 second timeout

        if (!string.IsNullOrEmpty(errorOutput))
        {
            throw new Exception($"gonuget stderr: {errorOutput}");
        }

        // Parse response envelope
        var envelope = JsonSerializer.Deserialize<ResponseEnvelope>(outputJson, new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase
        }) ?? throw new Exception("Failed to deserialize response");

        if (!envelope.Success)
        {
            var error = envelope.Error ?? new ErrorInfo { Message = "Unknown error" };
            throw new GonugetException(error.Code, error.Message, error.Details);
        }

        // Deserialize data payload
        return JsonSerializer.Deserialize<TResponse>(
            JsonSerializer.Serialize(envelope.Data),
            new JsonSerializerOptions { PropertyNamingPolicy = JsonNamingPolicy.CamelCase }
        ) ?? throw new Exception("Failed to deserialize response data");
    }

    /// <summary>
    /// Finds the gonuget executable in the test output directory or build location.
    /// </summary>
    private static string FindGonugetExecutable()
    {
        // Check test output directory first
        var testDir = AppContext.BaseDirectory;
        var exePath = Path.Combine(testDir, "gonuget-interop-test");
        if (File.Exists(exePath))
            return exePath;

        // Check relative to repository root (for local development)
        var repoRoot = Path.GetFullPath(Path.Combine(testDir, "../../../../../"));
        exePath = Path.Combine(repoRoot, "gonuget-interop-test");
        if (File.Exists(exePath))
            return exePath;

        throw new FileNotFoundException(
            "gonuget-interop-test executable not found. " +
            "Run 'go build -o gonuget-interop-test ./cmd/nuget-interop-test' before running tests.");
    }

    // Response types
    private class ResponseEnvelope
    {
        public bool Success { get; set; }
        public object? Data { get; set; }
        public ErrorInfo? Error { get; set; }
    }

    private class ErrorInfo
    {
        public string Code { get; set; } = "";
        public string Message { get; set; } = "";
        public string? Details { get; set; }
    }

    public class SignPackageResponse
    {
        public byte[] Signature { get; set; } = Array.Empty<byte>();
    }

    public class ParseSignatureResponse
    {
        public string Type { get; set; } = "";
        public string HashAlgorithm { get; set; } = "";
        public string SignerCertHash { get; set; } = "";
        public int TimestampCount { get; set; }
        public string[]? TimestampTimes { get; set; }
        public int Certificates { get; set; }
    }

    public class VerifySignatureResponse
    {
        public bool Valid { get; set; }
        public string[]? Errors { get; set; }
        public string[]? Warnings { get; set; }
        public string? SignerSubject { get; set; }
    }
}

/// <summary>
/// Exception thrown when gonuget returns an error.
/// </summary>
public class GonugetException : Exception
{
    public string Code { get; }
    public string? Details { get; }

    public GonugetException(string code, string message, string? details = null)
        : base($"[{code}] {message}")
    {
        Code = code;
        Details = details;
    }
}
