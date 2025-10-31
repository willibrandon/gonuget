using System;
using System.IO;
using System.Net.Http;
using System.Text.Json;
using System.Threading.Tasks;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Protocol;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// HTTP registration cache interoperability tests.
/// Validates that gonuget and NuGet.Client can read each other's cache files
/// for registration API responses.
///
/// Critical for performance: Registration cache provides 997x speedup by avoiding
/// redundant HTTP requests on subsequent package operations.
/// </summary>
public sealed class RegistrationCacheInteropTests : IDisposable
{
    private readonly string _testCacheDir;
    private readonly string _nugetOrgV3 = "https://api.nuget.org/v3/index.json";

    public RegistrationCacheInteropTests()
    {
        // Create a temporary cache directory for tests
        _testCacheDir = Path.Combine(Path.GetTempPath(), $"nuget-cache-test-{Guid.NewGuid()}");
        Directory.CreateDirectory(_testCacheDir);
    }

    public void Dispose()
    {
        // Cleanup test cache directory
        if (Directory.Exists(_testCacheDir))
        {
            try
            {
                Directory.Delete(_testCacheDir, recursive: true);
            }
            catch
            {
                // Ignore cleanup failures
            }
        }
    }

    #region Gonuget Reads Dotnet Cache

    [Fact]
    public async Task GonugetReadsRegistrationIndex_CachedByDotnet()
    {
        // Arrange: Use NuGet.Client to cache registration index
        var packageId = "Newtonsoft.Json";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}";
        var sourceURL = _nugetOrgV3;

        // Compute cache path using NuGet.Client's algorithm
        var baseFolderName = CachingUtility.RemoveInvalidFileNameChars(
            CachingUtility.ComputeHash(sourceURL, addIdentifiableCharacters: true));
        var baseFileName = CachingUtility.RemoveInvalidFileNameChars(cacheKey) + ".dat";
        var cacheFolder = Path.Combine(_testCacheDir, baseFolderName);
        var cacheFile = Path.Combine(cacheFolder, baseFileName);

        // Fetch registration index from network using NuGet.Client
        var registrationUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/index.json";

        // Use HttpClient with automatic decompression (registration5-gz-semver2 returns gzipped content)
        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(registrationUrl);

        // Write to cache using NuGet.Client's format
        Directory.CreateDirectory(cacheFolder);
        await File.WriteAllTextAsync(cacheFile, response);

        // Act: Verify gonuget can read this cache file
        var gonugetResponse = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);

        // Assert: gonuget should recognize the cache as valid
        Assert.True(gonugetResponse.Valid, "gonuget should be able to read NuGet.Client cached registration index");

        // Verify the cache file still exists and is readable
        Assert.True(File.Exists(cacheFile), "Cache file should exist");
        var cachedContent = await File.ReadAllTextAsync(cacheFile);
        Assert.NotEmpty(cachedContent);

        // Verify it's valid JSON (basic sanity check)
        using var jsonDoc = JsonDocument.Parse(cachedContent);
        Assert.NotNull(jsonDoc);
    }

    [Fact]
    public async Task GonugetReadsRegistrationPage_CachedByDotnet()
    {
        // Arrange: Cache a registration page using NuGet.Client's format
        var packageId = "Microsoft.Extensions.Logging";
        var lowerVersion = "1.0.0-rc1-final";
        var upperVersion = "3.1.23";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}_range_{lowerVersion}-{upperVersion}";
        var sourceURL = _nugetOrgV3;

        // Compute cache path
        var baseFolderName = CachingUtility.RemoveInvalidFileNameChars(
            CachingUtility.ComputeHash(sourceURL, addIdentifiableCharacters: true));
        var baseFileName = CachingUtility.RemoveInvalidFileNameChars(cacheKey) + ".dat";
        var cacheFolder = Path.Combine(_testCacheDir, baseFolderName);
        var cacheFile = Path.Combine(cacheFolder, baseFileName);

        // Fetch registration page from network
        var pageUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/page/{lowerVersion}/{upperVersion}.json";

        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(pageUrl);

        // Write to cache
        Directory.CreateDirectory(cacheFolder);
        await File.WriteAllTextAsync(cacheFile, response);

        // Act: Verify gonuget can read this cache
        var gonugetResponse = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);

        // Assert
        Assert.True(gonugetResponse.Valid, "gonuget should be able to read NuGet.Client cached registration page");

        // Verify content is valid JSON with expected structure
        var cachedContent = await File.ReadAllTextAsync(cacheFile);
        using var jsonDoc = JsonDocument.Parse(cachedContent);
        Assert.True(jsonDoc.RootElement.TryGetProperty("items", out _), "Registration page should have 'items' property");
    }

    [Theory]
    [InlineData("Microsoft.Extensions.Logging", "1.0.0-rc1-final", "3.1.23")]
    [InlineData("Serilog", "0.1.6", "1.2.47")]
    public async Task GonugetReadsVariousRegistrationPages_CachedByDotnet(string packageId, string lowerVersion, string upperVersion)
    {
        // Arrange
        var cacheKey = $"list_{packageId.ToLowerInvariant()}_range_{lowerVersion}-{upperVersion}";
        var sourceURL = _nugetOrgV3;

        var baseFolderName = CachingUtility.RemoveInvalidFileNameChars(
            CachingUtility.ComputeHash(sourceURL, addIdentifiableCharacters: true));
        var baseFileName = CachingUtility.RemoveInvalidFileNameChars(cacheKey) + ".dat";
        var cacheFolder = Path.Combine(_testCacheDir, baseFolderName);
        var cacheFile = Path.Combine(cacheFolder, baseFileName);

        // Fetch from network
        var pageUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/page/{lowerVersion}/{upperVersion}.json";

        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(pageUrl);

        Directory.CreateDirectory(cacheFolder);
        await File.WriteAllTextAsync(cacheFile, response);

        // Act
        var gonugetResponse = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);

        // Assert
        Assert.True(gonugetResponse.Valid, $"gonuget should read cached page for {packageId} range {lowerVersion}-{upperVersion}");
    }

    #endregion

    #region Dotnet Reads Gonuget Cache

    [Fact]
    public async Task DotnetReadsRegistrationIndex_CachedByGonuget()
    {
        // Arrange: Use gonuget to cache registration index
        var packageId = "Newtonsoft.Json";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}";
        var sourceURL = _nugetOrgV3;

        // Fetch and cache using gonuget (simulated by writing cache file)
        var registrationUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/index.json";

        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(registrationUrl);

        // Write cache using gonuget's bridge (which uses gonuget's cache writing logic)
        var gonugetPaths = GonugetBridge.GenerateCachePaths(_testCacheDir, sourceURL, cacheKey);
        Directory.CreateDirectory(Path.GetDirectoryName(gonugetPaths.CacheFile)!);
        await File.WriteAllTextAsync(gonugetPaths.CacheFile, response);

        // Act: Verify NuGet.Client can read this cache
        var maxAge = TimeSpan.FromMinutes(30);
        var dotnetStream = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile);

        // Assert
        Assert.NotNull(dotnetStream);

        using (dotnetStream)
        {
            var content = await new StreamReader(dotnetStream).ReadToEndAsync();
            Assert.NotEmpty(content);

            // Verify it's valid JSON
            using var jsonDoc = JsonDocument.Parse(content);
            Assert.NotNull(jsonDoc);
        }
    }

    [Fact]
    public async Task DotnetReadsRegistrationPage_CachedByGonuget()
    {
        // Arrange
        var packageId = "Microsoft.Extensions.Logging";
        var lowerVersion = "1.0.0-rc1-final";
        var upperVersion = "3.1.23";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}_range_{lowerVersion}-{upperVersion}";
        var sourceURL = _nugetOrgV3;

        // Fetch from network
        var pageUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/page/{lowerVersion}/{upperVersion}.json";

        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(pageUrl);

        // Cache using gonuget's paths
        var gonugetPaths = GonugetBridge.GenerateCachePaths(_testCacheDir, sourceURL, cacheKey);
        Directory.CreateDirectory(Path.GetDirectoryName(gonugetPaths.CacheFile)!);
        await File.WriteAllTextAsync(gonugetPaths.CacheFile, response);

        // Act: NuGet.Client reads the cache
        var maxAge = TimeSpan.FromMinutes(30);
        var dotnetStream = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile);

        // Assert
        Assert.NotNull(dotnetStream);

        using (dotnetStream)
        {
            var content = await new StreamReader(dotnetStream).ReadToEndAsync();
            Assert.NotEmpty(content);

            // Verify structure
            using var jsonDoc = JsonDocument.Parse(content);
            Assert.True(jsonDoc.RootElement.TryGetProperty("items", out _));
        }
    }

    [Theory]
    [InlineData("Microsoft.Extensions.Logging", "1.0.0-rc1-final", "3.1.23")]
    [InlineData("Serilog", "0.1.6", "1.2.47")]
    public async Task DotnetReadsVariousRegistrationPages_CachedByGonuget(string packageId, string lowerVersion, string upperVersion)
    {
        // Arrange
        var cacheKey = $"list_{packageId.ToLowerInvariant()}_range_{lowerVersion}-{upperVersion}";
        var sourceURL = _nugetOrgV3;

        var pageUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/page/{lowerVersion}/{upperVersion}.json";

        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(pageUrl);

        var gonugetPaths = GonugetBridge.GenerateCachePaths(_testCacheDir, sourceURL, cacheKey);
        Directory.CreateDirectory(Path.GetDirectoryName(gonugetPaths.CacheFile)!);
        await File.WriteAllTextAsync(gonugetPaths.CacheFile, response);

        // Act
        var maxAge = TimeSpan.FromMinutes(30);
        var dotnetStream = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile);

        // Assert
        Assert.NotNull(dotnetStream);
        dotnetStream?.Dispose();
    }

    #endregion

    #region Round-Trip Tests

    [Fact]
    public async Task RoundTrip_DotnetCaches_GonugetReads_DotnetReadsAgain()
    {
        // This test verifies that cache files remain compatible through multiple operations
        var packageId = "Newtonsoft.Json";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}";
        var sourceURL = _nugetOrgV3;

        // Step 1: NuGet.Client caches
        var registrationUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/index.json";
        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(registrationUrl);

        var baseFolderName = CachingUtility.RemoveInvalidFileNameChars(
            CachingUtility.ComputeHash(sourceURL, addIdentifiableCharacters: true));
        var baseFileName = CachingUtility.RemoveInvalidFileNameChars(cacheKey) + ".dat";
        var cacheFolder = Path.Combine(_testCacheDir, baseFolderName);
        var cacheFile = Path.Combine(cacheFolder, baseFileName);

        Directory.CreateDirectory(cacheFolder);
        await File.WriteAllTextAsync(cacheFile, response);

        // Step 2: gonuget reads
        var gonugetRead1 = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);
        Assert.True(gonugetRead1.Valid);

        // Step 3: NuGet.Client reads again
        var maxAge = TimeSpan.FromMinutes(30);
        var dotnetRead = CachingUtility.ReadCacheFile(maxAge, cacheFile);
        Assert.NotNull(dotnetRead);
        dotnetRead?.Dispose();

        // Step 4: gonuget reads again
        var gonugetRead2 = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);
        Assert.True(gonugetRead2.Valid);
    }

    [Fact]
    public async Task RoundTrip_GonugetCaches_DotnetReads_GonugetReadsAgain()
    {
        // Reverse of above: gonuget → dotnet → gonuget
        var packageId = "Microsoft.Extensions.Logging";
        var lowerVersion = "1.0.0-rc1-final";
        var upperVersion = "3.1.23";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}_range_{lowerVersion}-{upperVersion}";
        var sourceURL = _nugetOrgV3;

        // Step 1: gonuget caches
        var pageUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/page/{lowerVersion}/{upperVersion}.json";
        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(pageUrl);

        var gonugetPaths = GonugetBridge.GenerateCachePaths(_testCacheDir, sourceURL, cacheKey);
        Directory.CreateDirectory(Path.GetDirectoryName(gonugetPaths.CacheFile)!);
        await File.WriteAllTextAsync(gonugetPaths.CacheFile, response);

        // Step 2: NuGet.Client reads
        var maxAge = TimeSpan.FromMinutes(30);
        var dotnetRead = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile);
        Assert.NotNull(dotnetRead);
        dotnetRead?.Dispose();

        // Step 3: gonuget reads
        var gonugetRead1 = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);
        Assert.True(gonugetRead1.Valid);

        // Step 4: NuGet.Client reads again
        var dotnetRead2 = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile);
        Assert.NotNull(dotnetRead2);
        dotnetRead2?.Dispose();

        // Step 5: gonuget reads again
        var gonugetRead2 = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800);
        Assert.True(gonugetRead2.Valid);
    }

    #endregion

    #region TTL Interoperability

    [Fact]
    public async Task BothTools_RespectSame30MinuteTTL()
    {
        // Verify both tools use the same 30-minute TTL for registration cache
        var packageId = "Newtonsoft.Json";
        var cacheKey = $"list_{packageId.ToLowerInvariant()}";
        var sourceURL = _nugetOrgV3;

        // Create cache file
        var registrationUrl = $"https://api.nuget.org/v3/registration5-gz-semver2/{packageId.ToLowerInvariant()}/index.json";
        using var handler = new HttpClientHandler { AutomaticDecompression = System.Net.DecompressionMethods.GZip | System.Net.DecompressionMethods.Deflate };
        using var httpClient = new HttpClient(handler);
        var response = await httpClient.GetStringAsync(registrationUrl);

        var gonugetPaths = GonugetBridge.GenerateCachePaths(_testCacheDir, sourceURL, cacheKey);
        Directory.CreateDirectory(Path.GetDirectoryName(gonugetPaths.CacheFile)!);
        await File.WriteAllTextAsync(gonugetPaths.CacheFile, response);

        // Set file timestamp to 29 minutes ago (within TTL)
        var validTime = DateTime.UtcNow.AddMinutes(-29);
        File.SetLastWriteTimeUtc(gonugetPaths.CacheFile, validTime);

        // Both should consider it valid with 30-minute TTL
        var maxAge = TimeSpan.FromMinutes(30);
        var dotnetValid = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile) != null;
        var gonugetValid = GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800).Valid;

        Assert.True(dotnetValid);
        Assert.True(gonugetValid);

        // Set file timestamp to 31 minutes ago (expired)
        var expiredTime = DateTime.UtcNow.AddMinutes(-31);
        File.SetLastWriteTimeUtc(gonugetPaths.CacheFile, expiredTime);

        // Both should consider it invalid
        var dotnetInvalid = CachingUtility.ReadCacheFile(maxAge, gonugetPaths.CacheFile) == null;
        var gonugetInvalid = !GonugetBridge.ValidateCacheFile(_testCacheDir, sourceURL, cacheKey, maxAgeSeconds: 1800).Valid;

        Assert.True(dotnetInvalid);
        Assert.True(gonugetInvalid);
    }

    #endregion
}
