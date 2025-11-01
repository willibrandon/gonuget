using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
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
    /// Formats a framework to its short folder name representation.
    /// This matches NuGet.Client's GetShortFolderName() behavior.
    /// </summary>
    /// <param name="framework">The framework identifier to format (e.g., "net6.0-windows", "portable-net45+win8").</param>
    /// <returns>The short folder name (e.g., "net6.0-windows10.0", "portable-net45+win8").</returns>
    public static FormatFrameworkResponse FormatFramework(string framework)
    {
        var request = new
        {
            action = "format_framework",
            data = new { framework }
        };

        return Execute<FormatFrameworkResponse>(request);
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
    /// Finds runtime assemblies matching package paths for a target framework.
    /// </summary>
    /// <param name="paths">Package file paths (e.g., "lib/net6.0/MyLib.dll").</param>
    /// <param name="targetFramework">Target framework (e.g., "net8.0").</param>
    /// <returns>List of matched content items with their properties.</returns>
    public static FindAssembliesResponse FindRuntimeAssemblies(
        string[] paths,
        string? targetFramework = null)
    {
        var request = new
        {
            action = "find_runtime_assemblies",
            data = new { paths, targetFramework }
        };

        return Execute<FindAssembliesResponse>(request);
    }

    /// <summary>
    /// Finds compile reference assemblies matching package paths for a target framework.
    /// </summary>
    /// <param name="paths">Package file paths (e.g., "ref/net6.0/MyLib.dll").</param>
    /// <param name="targetFramework">Target framework (e.g., "net8.0").</param>
    /// <returns>List of matched content items with their properties.</returns>
    public static FindAssembliesResponse FindCompileAssemblies(
        string[] paths,
        string? targetFramework = null)
    {
        var request = new
        {
            action = "find_compile_assemblies",
            data = new { paths, targetFramework }
        };

        return Execute<FindAssembliesResponse>(request);
    }

    /// <summary>
    /// Parses a single asset path and extracts its properties (tfm, assembly, rid, etc.).
    /// </summary>
    /// <param name="path">The asset path to parse (e.g., "lib/net6.0/MyLib.dll").</param>
    /// <returns>Parsed asset properties including framework, assembly name, runtime ID, etc.</returns>
    public static ParseAssetPathResponse ParseAssetPath(string path)
    {
        var request = new
        {
            action = "parse_asset_path",
            data = new { path }
        };

        return Execute<ParseAssetPathResponse>(request);
    }

    /// <summary>
    /// Expands a runtime identifier to all compatible RIDs in priority order.
    /// </summary>
    /// <param name="rid">The runtime identifier to expand (e.g., "win10-x64").</param>
    /// <returns>Array of compatible RIDs in priority order (nearest first).</returns>
    public static ExpandRuntimeResponse ExpandRuntime(string rid)
    {
        var request = new
        {
            action = "expand_runtime",
            data = new { rid }
        };

        return Execute<ExpandRuntimeResponse>(request);
    }

    /// <summary>
    /// Checks if two runtime identifiers are compatible.
    /// </summary>
    /// <param name="targetRid">The target runtime (criteria).</param>
    /// <param name="packageRid">The package runtime (provided).</param>
    /// <returns>True if the package RID is compatible with the target RID.</returns>
    public static AreRuntimesCompatibleResponse AreRuntimesCompatible(string targetRid, string packageRid)
    {
        var request = new
        {
            action = "are_runtimes_compatible",
            data = new { targetRid, packageRid }
        };

        return Execute<AreRuntimesCompatibleResponse>(request);
    }

    /// <summary>
    /// Computes a cache hash for a given value using gonuget's algorithm.
    /// </summary>
    /// <param name="value">The string to hash (usually a URL or package ID).</param>
    /// <param name="addIdentifiableCharacters">Whether to append trailing portion for readability.</param>
    /// <returns>The computed cache hash (40-char hex + optional trailing chars).</returns>
    public static ComputeCacheHashResponse ComputeCacheHash(string value, bool addIdentifiableCharacters = true)
    {
        var request = new
        {
            action = "compute_cache_hash",
            data = new { value, addIdentifiableCharacters }
        };

        return Execute<ComputeCacheHashResponse>(request);
    }

    /// <summary>
    /// Sanitizes a filename by removing invalid characters using gonuget's algorithm.
    /// </summary>
    /// <param name="value">The filename or path to sanitize.</param>
    /// <returns>The sanitized filename with invalid chars replaced and collapsed.</returns>
    public static SanitizeCacheFilenameResponse SanitizeCacheFilename(string value)
    {
        var request = new
        {
            action = "sanitize_cache_filename",
            data = new { value }
        };

        return Execute<SanitizeCacheFilenameResponse>(request);
    }

    /// <summary>
    /// Generates cache file paths for a source URL and cache key using gonuget's algorithm.
    /// </summary>
    /// <param name="cacheDirectory">The root cache directory path.</param>
    /// <param name="sourceURL">The source URL to hash for the folder name.</param>
    /// <param name="cacheKey">The cache key for the file name.</param>
    /// <returns>The generated cache paths (base folder name, cache file, new file).</returns>
    public static GenerateCachePathsResponse GenerateCachePaths(string cacheDirectory, string sourceURL, string cacheKey)
    {
        var request = new
        {
            action = "generate_cache_paths",
            data = new { cacheDirectory, sourceURL, cacheKey }
        };

        return Execute<GenerateCachePathsResponse>(request);
    }

    /// <summary>
    /// Validates whether a cache file is still valid based on its age and the maximum allowed age.
    /// Matches NuGet.Client's CachingUtility.ReadCacheFile() TTL validation logic.
    /// </summary>
    /// <param name="cacheDirectory">The root cache directory path.</param>
    /// <param name="sourceURL">The source URL for the cached resource.</param>
    /// <param name="cacheKey">The cache key for the file.</param>
    /// <param name="maxAgeSeconds">The maximum age in seconds before the cache is considered expired.</param>
    /// <returns>True if the file exists and is within the TTL, false if missing or expired.</returns>
    public static ValidateCacheFileResponse ValidateCacheFile(string cacheDirectory, string sourceURL, string cacheKey, long maxAgeSeconds)
    {
        var request = new
        {
            action = "validate_cache_file",
            data = new { cacheDirectory, sourceURL, cacheKey, maxAgeSeconds }
        };

        return Execute<ValidateCacheFileResponse>(request);
    }

    /// <summary>
    /// Calculates the dgSpecHash for a project file.
    /// </summary>
    /// <param name="projectPath">The absolute path to the project file (.csproj/.fsproj/.vbproj).</param>
    /// <returns>The calculated dgSpecHash.</returns>
    public static CalculateDgSpecHashResponse CalculateDgSpecHash(string projectPath)
    {
        var request = new
        {
            action = "calculate_dgspec_hash",
            data = new { projectPath }
        };

        return Execute<CalculateDgSpecHashResponse>(request);
    }

    /// <summary>
    /// Verifies a project.nuget.cache file.
    /// </summary>
    /// <param name="cachePath">The absolute path to the project.nuget.cache file.</param>
    /// <param name="currentHash">The expected dgSpecHash to compare against.</param>
    /// <returns>Cache validation results.</returns>
    public static VerifyProjectCacheFileResponse VerifyProjectCacheFile(string cachePath, string currentHash)
    {
        var request = new
        {
            action = "verify_project_cache_file",
            data = new { cachePath, currentHash }
        };

        return Execute<VerifyProjectCacheFileResponse>(request);
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
    /// Extracts a package using V2 (packages.config) layout.
    /// </summary>
    public static ExtractPackageV2Response ExtractPackageV2(
        byte[] packageBytes,
        string installPath,
        int packageSaveMode = 7, // Default: Nuspec | Nupkg | Files
        bool useSideBySideLayout = true,
        int xmlDocFileSaveMode = 0) // Default: None
    {
        var request = new
        {
            action = "extract_package_v2",
            data = new
            {
                packageBytes,
                installPath,
                packageSaveMode,
                useSideBySideLayout,
                xmlDocFileSaveMode
            }
        };

        return Execute<ExtractPackageV2Response>(request);
    }

    /// <summary>
    /// Installs a package using V3 (PackageReference) layout.
    /// </summary>
    public static InstallFromSourceV3Response InstallFromSourceV3(
        byte[] packageBytes,
        string id,
        string version,
        string globalPackagesFolder,
        int packageSaveMode = 7, // Default: Nuspec | Nupkg | Files
        int xmlDocFileSaveMode = 0) // Default: None
    {
        var request = new
        {
            action = "install_from_source_v3",
            data = new
            {
                packageBytes,
                id,
                version,
                globalPackagesFolder,
                packageSaveMode,
                xmlDocFileSaveMode
            }
        };

        return Execute<InstallFromSourceV3Response>(request);
    }

    /// <summary>
    /// Walks the dependency graph for a package using gonuget's resolver.
    /// </summary>
    public static WalkGraphResponse WalkGraph(
        string packageId,
        string versionRange,
        string targetFramework,
        string[] sources)
    {
        var request = new
        {
            action = "walk_graph",
            data = new
            {
                packageId,
                versionRange,
                targetFramework,
                sources
            }
        };

        return Execute<WalkGraphResponse>(request);
    }

    /// <summary>
    /// Resolves version conflicts in a dependency graph.
    /// </summary>
    public static ResolveConflictsResponse ResolveConflicts(
        string[] packageIds,
        string[] versionRanges,
        string targetFramework)
    {
        var request = new
        {
            action = "resolve_conflicts",
            data = new
            {
                packageIds,
                versionRanges,
                targetFramework
            }
        };

        return Execute<ResolveConflictsResponse>(request);
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

    /// <summary>
    /// Analyzes dependency graph for circular dependencies (M5.5).
    /// </summary>
    public static AnalyzeCyclesResponse AnalyzeCycles(
        string packageId,
        string versionRange,
        string targetFramework,
        string[] sources,
        InMemoryDependencyProvider? inMemoryProvider = null)
    {
        var data = new Dictionary<string, object>
        {
            ["packageId"] = packageId,
            ["versionRange"] = versionRange,
            ["targetFramework"] = targetFramework,
            ["sources"] = sources
        };

        // Add in-memory packages if provider is given
        if (inMemoryProvider != null)
        {
            data["inMemoryPackages"] = SerializeInMemoryPackages(inMemoryProvider);
        }

        var request = new
        {
            action = "analyze_cycles",
            data
        };

        return Execute<AnalyzeCyclesResponse>(request);
    }

    /// <summary>
    /// Serializes in-memory packages to JSON format for the CLI bridge.
    /// </summary>
    private static object[] SerializeInMemoryPackages(InMemoryDependencyProvider provider)
    {
        return provider.GetAllPackages()
            .Select(pkg => new
            {
                id = pkg.Id,
                version = pkg.Version.ToString(),
                dependencies = pkg.Dependencies.Select(d => new
                {
                    id = d.Id,
                    versionRange = d.VersionRange.ToString()
                }).ToArray()
            })
            .ToArray();
    }

    /// <summary>
    /// Resolves transitive dependencies for multiple root packages (M5.6).
    /// </summary>
    public static ResolveTransitiveResponse ResolveTransitive(
        PackageSpec[] rootPackages,
        string targetFramework,
        string[] sources)
    {
        var request = new
        {
            action = "resolve_transitive",
            data = new
            {
                rootPackages,
                targetFramework,
                sources
            }
        };

        return Execute<ResolveTransitiveResponse>(request);
    }

    /// <summary>
    /// Benchmarks cache deduplication with concurrent requests (M5.7).
    /// </summary>
    public static BenchmarkCacheResponse BenchmarkCache(
        string packageId,
        string versionRange,
        string targetFramework,
        string[] sources,
        int concurrentRequests)
    {
        var request = new
        {
            action = "benchmark_cache",
            data = new
            {
                packageId,
                versionRange,
                targetFramework,
                sources,
                concurrentRequests
            }
        };

        return Execute<BenchmarkCacheResponse>(request);
    }

    /// <summary>
    /// Resolves package with cache TTL (M5.7).
    /// </summary>
    public static ResolveWithTTLResponse ResolveWithTTL(
        string packageId,
        string versionRange,
        string targetFramework,
        string[] sources,
        int ttlSeconds)
    {
        var request = new
        {
            action = "resolve_with_ttl",
            data = new
            {
                packageId,
                versionRange,
                targetFramework,
                sources,
                ttlSeconds
            }
        };

        return Execute<ResolveWithTTLResponse>(request);
    }

    /// <summary>
    /// Benchmarks parallel resolution performance (M5.8).
    /// </summary>
    public static BenchmarkParallelResponse BenchmarkParallel(
        PackageSpec[] packageSpecs,
        string targetFramework,
        string[] sources,
        bool sequential = false,
        bool recursive = true)
    {
        var request = new
        {
            action = "benchmark_parallel",
            data = new
            {
                packageSpecs,
                targetFramework,
                sources,
                sequential,
                recursive
            }
        };

        return Execute<BenchmarkParallelResponse>(request);
    }

    /// <summary>
    /// Resolves packages with worker pool limits (M5.8).
    /// </summary>
    public static ResolveWithWorkerLimitResponse ResolveWithWorkerLimit(
        PackageSpec[] packageSpecs,
        string targetFramework,
        string[] sources,
        int maxWorkers)
    {
        var request = new
        {
            action = "resolve_with_worker_limit",
            data = new
            {
                packageSpecs,
                targetFramework,
                sources,
                maxWorkers
            }
        };

        return Execute<ResolveWithWorkerLimitResponse>(request);
    }

    /// <summary>
    /// Resolves the latest version of a package.
    /// </summary>
    public static ResolveLatestVersionResponse ResolveLatestVersion(
        string packageId,
        string source = "https://api.nuget.org/v3/index.json",
        bool prerelease = false)
    {
        var request = new
        {
            action = "resolve_latest_version",
            data = new
            {
                packageId,
                source,
                prerelease
            }
        };

        return Execute<ResolveLatestVersionResponse>(request);
    }

    /// <summary>
    /// Parses a project.assets.json lock file.
    /// </summary>
    public static ParseLockFileResponse ParseLockFile(string lockFilePath)
    {
        var request = new
        {
            action = "parse_lock_file",
            data = new { lockFilePath }
        };

        return Execute<ParseLockFileResponse>(request);
    }

    /// <summary>
    /// Restores direct dependencies for a project.
    /// </summary>
    public static RestoreDirectDependenciesResponse RestoreDirectDependencies(
        string projectPath,
        string packagesFolder,
        string[] sources,
        bool noCache = false,
        bool force = false)
    {
        var request = new
        {
            action = "restore_direct_dependencies",
            data = new
            {
                projectPath,
                packagesFolder,
                sources,
                noCache,
                force
            }
        };

        return Execute<RestoreDirectDependenciesResponse>(request);
    }

    /// <summary>
    /// Restores a project with full transitive dependency resolution and categorization.
    /// </summary>
    /// <param name="projectPath">Absolute path to .csproj file</param>
    /// <param name="packagesFolder">Optional custom packages folder path</param>
    /// <param name="sources">Optional list of package sources</param>
    /// <param name="noCache">Disable cache usage</param>
    /// <param name="force">Force re-download of packages</param>
    /// <returns>Restore result with direct and transitive packages categorized</returns>
    public static RestoreTransitiveResponse RestoreTransitive(
        string projectPath,
        string? packagesFolder = null,
        string[]? sources = null,
        bool noCache = false,
        bool force = false)
    {
        var request = new
        {
            action = "restore_transitive",
            data = new
            {
                projectPath,
                packagesFolder,
                sources = sources ?? Array.Empty<string>(),
                noCache,
                force
            }
        };

        return Execute<RestoreTransitiveResponse>(request);
    }

    /// <summary>
    /// Compares two project.assets.json files semantically (gonuget vs NuGet.Client).
    /// </summary>
    /// <param name="gonugetLockFilePath">Path to gonuget-generated project.assets.json</param>
    /// <param name="nugetLockFilePath">Path to NuGet.Client-generated project.assets.json</param>
    /// <returns>Comparison result with detailed differences if files don't match</returns>
    public static CompareProjectAssetsResponse CompareProjectAssets(
        string gonugetLockFilePath,
        string nugetLockFilePath)
    {
        var request = new
        {
            action = "compare_project_assets",
            data = new
            {
                gonugetLockFilePath,
                nugetLockFilePath
            }
        };

        return Execute<CompareProjectAssetsResponse>(request);
    }

    /// <summary>
    /// Validates error message format between gonuget and NuGet.Client.
    /// </summary>
    /// <param name="gonugetError">Error message from gonuget</param>
    /// <param name="nugetError">Error message from NuGet.Client</param>
    /// <returns>Validation result with differences if messages don't match</returns>
    public static ValidateErrorMessagesResponse ValidateErrorMessages(
        string gonugetError,
        string nugetError)
    {
        var request = new
        {
            action = "validate_error_messages",
            data = new
            {
                gonugetError,
                nugetError
            }
        };

        return Execute<ValidateErrorMessagesResponse>(request);
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
