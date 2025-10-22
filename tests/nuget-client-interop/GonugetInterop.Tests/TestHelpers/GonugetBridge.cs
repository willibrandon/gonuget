using System;
using System.Collections.Generic;
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
    /// Compares two NuGet version strings and returns the comparison result.
    /// </summary>
    /// <param name="version1">The first version string to compare.</param>
    /// <param name="version2">The second version string to compare.</param>
    /// <returns>-1 if version1 &lt; version2, 0 if equal, 1 if version1 &gt; version2.</returns>
    public static CompareVersionsResponse CompareVersions(string version1, string version2)
    {
        var request = new
        {
            action = "compare_versions",
            data = new { version1, version2 }
        };

        return Execute<CompareVersionsResponse>(request);
    }

    /// <summary>
    /// Parses a NuGet version string into its components.
    /// </summary>
    /// <param name="version">The version string to parse (e.g., "1.0.0-beta.1+git.abc123").</param>
    /// <returns>Parsed version components including major, minor, patch, release label, and metadata.</returns>
    public static ParseVersionResponse ParseVersion(string version)
    {
        var request = new
        {
            action = "parse_version",
            data = new { version }
        };

        return Execute<ParseVersionResponse>(request);
    }

    /// <summary>
    /// Checks if a package framework is compatible with a project framework.
    /// </summary>
    /// <param name="packageFramework">The framework the package supports (e.g., "net6.0").</param>
    /// <param name="projectFramework">The project's target framework (e.g., "net8.0").</param>
    /// <returns>Compatibility result indicating if the project can use the package.</returns>
    public static CheckFrameworkCompatResponse CheckFrameworkCompat(
        string packageFramework,
        string projectFramework)
    {
        var request = new
        {
            action = "check_framework_compat",
            data = new { packageFramework, projectFramework }
        };

        return Execute<CheckFrameworkCompatResponse>(request);
    }

    /// <summary>
    /// Parses a framework identifier (TFM) into its components.
    /// </summary>
    /// <param name="framework">The framework identifier to parse (e.g., "net8.0", "netstandard2.1").</param>
    /// <returns>Parsed framework components including identifier, version, profile, and platform.</returns>
    public static ParseFrameworkResponse ParseFramework(string framework)
    {
        var request = new
        {
            action = "parse_framework",
            data = new { framework }
        };

        return Execute<ParseFrameworkResponse>(request);
    }

    /// <summary>
    /// Reads package metadata and structure from a .nupkg file.
    /// </summary>
    /// <param name="packageBytes">The complete package file as a byte array (ZIP format).</param>
    /// <returns>Package metadata including ID, version, authors, dependencies, and signature information.</returns>
    public static ReadPackageResponse ReadPackage(byte[] packageBytes)
    {
        var request = new
        {
            action = "read_package",
            data = new { packageBytes }
        };

        return Execute<ReadPackageResponse>(request);
    }

    /// <summary>
    /// Builds a minimal NuGet package from metadata and files.
    /// </summary>
    /// <param name="id">The package identifier (e.g., "MyPackage").</param>
    /// <param name="version">The package version (e.g., "1.0.0").</param>
    /// <param name="authors">Optional package authors.</param>
    /// <param name="description">Optional package description.</param>
    /// <param name="files">Optional files to include (relative path -> content bytes).</param>
    /// <returns>The complete package as a byte array (ZIP format).</returns>
    public static BuildPackageResponse BuildPackage(
        string id,
        string version,
        string[]? authors = null,
        string? description = null,
        Dictionary<string, byte[]>? files = null)
    {
        var request = new
        {
            action = "build_package",
            data = new
            {
                id,
                version,
                authors = authors ?? Array.Empty<string>(),
                description = description ?? "",
                files = files ?? new Dictionary<string, byte[]>()
            }
        };

        return Execute<BuildPackageResponse>(request);
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
        var requestJson = JsonSerializer.Serialize(request, s_serializerOptions);

        process.StandardInput.WriteLine(requestJson);
        process.StandardInput.Close();

        // Read response
        var outputJson = process.StandardOutput.ReadToEnd();
        var errorOutput = process.StandardError.ReadToEnd();

        process.WaitForExit(30000); // 30 second timeout

        if (!string.IsNullOrEmpty(errorOutput))
        {
            throw new InvalidOperationException($"gonuget stderr: {errorOutput}");
        }

        // Parse response envelope
        var envelope = JsonSerializer.Deserialize<ResponseEnvelope>(outputJson, s_deserializerOptions)
            ?? throw new InvalidOperationException("Failed to deserialize response");

        if (!envelope.Success)
        {
            var error = envelope.Error ?? new ErrorInfo { Message = "Unknown error" };
            throw new GonugetException(error.Code, error.Message, error.Details);
        }

        // Deserialize data payload
        return JsonSerializer.Deserialize<TResponse>(
            JsonSerializer.Serialize(envelope.Data, s_serializerOptions),
            s_deserializerOptions
        ) ?? throw new InvalidOperationException("Failed to deserialize response data");
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

    // Internal response envelope types
    private sealed class ResponseEnvelope
    {
        public bool Success { get; set; }
        public object? Data { get; set; }
        public ErrorInfo? Error { get; set; }
    }

    private sealed class ErrorInfo
    {
        public string Code { get; set; } = "";
        public string Message { get; set; } = "";
        public string? Details { get; set; }
    }

    // Cached JSON serializer options for performance
    private static readonly JsonSerializerOptions s_serializerOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private static readonly JsonSerializerOptions s_deserializerOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase
    };
}
