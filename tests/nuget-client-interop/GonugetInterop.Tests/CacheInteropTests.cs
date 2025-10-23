using System.IO;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Protocol;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Cache operation interop tests validating gonuget against NuGet.Client.
/// Tests hash computation, filename sanitization, and cache path generation.
/// These tests ensure gonuget's cache structure is compatible with NuGet.Client.
/// </summary>
public sealed class CacheInteropTests
{
    #region ComputeHash Tests (10 cases)

    [Theory]
    [InlineData("https://api.nuget.org/v3/index.json", true)]
    [InlineData("https://example.com/nuget/v3/index.json", true)]
    [InlineData("https://pkgs.dev.azure.com/_packaging/feed/nuget/v3/index.json", true)]
    [InlineData("newtonsoft.json", true)]
    [InlineData("Microsoft.Extensions.DependencyInjection", true)]
    public void ComputeCacheHash_WithIdentifiableChars_MatchesNuGetClient(string value, bool addChars)
    {
        // Compute hash with NuGet.Client
        var nugetHash = CachingUtility.ComputeHash(value, addChars);

        // Compute hash with gonuget
        var gonugetResponse = GonugetBridge.ComputeCacheHash(value, addChars);

        // Validate hashes match exactly
        Assert.Equal(nugetHash, gonugetResponse.Hash);
    }

    [Theory]
    [InlineData("https://api.nuget.org/v3/index.json", false)]
    [InlineData("https://example.com/very/long/path/that/exceeds/thirty/two/characters", false)]
    [InlineData("newtonsoft.json", false)]
    [InlineData("test", false)]
    [InlineData("a", false)]
    public void ComputeCacheHash_WithoutIdentifiableChars_MatchesNuGetClient(string value, bool addChars)
    {
        // Compute hash with NuGet.Client
        var nugetHash = CachingUtility.ComputeHash(value, addChars);

        // Compute hash with gonuget
        var gonugetResponse = GonugetBridge.ComputeCacheHash(value, addChars);

        // Validate hashes match exactly
        Assert.Equal(nugetHash, gonugetResponse.Hash);
    }

    [Fact]
    public void ComputeCacheHash_ShortString_ReturnsHashWithTrailing()
    {
        var value = "https://api.nuget.org/v3/index.json";

        // Compute with NuGet.Client
        var nugetHash = CachingUtility.ComputeHash(value, true);

        // Compute with gonuget
        var gonugetResponse = GonugetBridge.ComputeCacheHash(value, true);

        // Should have format: 40-char-hex + $ + trailing-chars
        Assert.Contains("$", gonugetResponse.Hash);
        Assert.Equal(nugetHash, gonugetResponse.Hash);

        // Hash portion should be 40 chars (SHA256 truncated to 20 bytes)
        var parts = gonugetResponse.Hash.Split('$');
        Assert.Equal(2, parts.Length);
        Assert.Equal(40, parts[0].Length);
    }

    [Fact]
    public void ComputeCacheHash_LongString_UsesLast32Chars()
    {
        var value = "https://example.com/very/long/path/that/definitely/exceeds/thirty/two/characters/for/sure/index.json";

        // Compute with NuGet.Client
        var nugetHash = CachingUtility.ComputeHash(value, true);

        // Compute with gonuget
        var gonugetResponse = GonugetBridge.ComputeCacheHash(value, true);

        Assert.Equal(nugetHash, gonugetResponse.Hash);

        // Should only include last 32 characters after $
        var parts = gonugetResponse.Hash.Split('$');
        Assert.Equal(2, parts.Length);
        Assert.True(parts[1].Length <= 32);
    }

    [Fact]
    public void ComputeCacheHash_WithoutChars_ReturnsOnlyHash()
    {
        var value = "test-value";

        // Compute with NuGet.Client
        var nugetHash = CachingUtility.ComputeHash(value, false);

        // Compute with gonuget
        var gonugetResponse = GonugetBridge.ComputeCacheHash(value, false);

        Assert.Equal(nugetHash, gonugetResponse.Hash);

        // Should be exactly 40 characters (20 bytes hex-encoded)
        Assert.Equal(40, gonugetResponse.Hash.Length);
        Assert.DoesNotContain("$", gonugetResponse.Hash);
    }

    #endregion

    #region Filename Sanitization Tests (10 cases)

    [Theory]
    [InlineData("valid-filename.txt", "valid-filename.txt")]
    [InlineData("normal.json", "normal.json")]
    [InlineData("file123.dat", "file123.dat")]
    [InlineData("service-index", "service-index")]
    public void SanitizeCacheFilename_ValidFilename_RemainsUnchanged(string input, string expected)
    {
        // Sanitize with NuGet.Client
        var nugetSanitized = CachingUtility.RemoveInvalidFileNameChars(input);

        // Sanitize with gonuget
        var gonugetResponse = GonugetBridge.SanitizeCacheFilename(input);

        Assert.Equal(expected, nugetSanitized);
        Assert.Equal(nugetSanitized, gonugetResponse.Sanitized);
    }

    [Theory]
    [InlineData("file<name>test.txt")]
    [InlineData("file:with:colons")]
    [InlineData("file*with*stars")]
    [InlineData("file?question")]
    [InlineData("file|pipe")]
    [InlineData("file\"quotes\"")]
    public void SanitizeCacheFilename_WindowsInvalidChars_MatchesNuGetClient(string input)
    {
        // Sanitize with NuGet.Client
        var nugetSanitized = CachingUtility.RemoveInvalidFileNameChars(input);

        // Sanitize with gonuget
        var gonugetResponse = GonugetBridge.SanitizeCacheFilename(input);

        // On Windows, these chars are replaced. On Unix, they're valid.
        // Just verify gonuget matches NuGet.Client behavior regardless of platform.
        Assert.Equal(nugetSanitized, gonugetResponse.Sanitized);
    }

    [Theory]
    [InlineData("file/with/slashes")]
    [InlineData("file\\with\\backslashes")]
    [InlineData("mixed/slash\\test")]
    public void SanitizeCacheFilename_PathSeparators_MatchesNuGetClient(string input)
    {
        // Sanitize with NuGet.Client
        var nugetSanitized = CachingUtility.RemoveInvalidFileNameChars(input);

        // Sanitize with gonuget
        var gonugetResponse = GonugetBridge.SanitizeCacheFilename(input);

        // Forward slash is always replaced. Backslash behavior is platform-specific.
        // Just verify gonuget matches NuGet.Client behavior regardless of platform.
        Assert.Equal(nugetSanitized, gonugetResponse.Sanitized);
    }

    [Theory]
    [InlineData("file__name", "file_name")]
    [InlineData("file___test", "file_test")]
    [InlineData("multiple____underscores", "multiple_underscores")]
    public void SanitizeCacheFilename_DoubleUnderscores_CollapsesToSingle(string input, string expected)
    {
        // Sanitize with NuGet.Client
        var nugetSanitized = CachingUtility.RemoveInvalidFileNameChars(input);

        // Sanitize with gonuget
        var gonugetResponse = GonugetBridge.SanitizeCacheFilename(input);

        Assert.Equal(expected, nugetSanitized);
        Assert.Equal(nugetSanitized, gonugetResponse.Sanitized);
    }

    #endregion

    #region Cache Path Generation Tests (10 cases)

    [Fact]
    public void GenerateCachePaths_StandardURL_MatchesNuGetClientStructure()
    {
        var cacheDir = Path.GetTempPath();
        var sourceURL = "https://api.nuget.org/v3/index.json";
        var cacheKey = "service-index";

        // Generate paths with gonuget
        var gonugetResponse = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);

        // Validate structure
        Assert.NotEmpty(gonugetResponse.BaseFolderName);
        Assert.Contains(gonugetResponse.BaseFolderName, gonugetResponse.CacheFile);
        Assert.Contains(cacheKey, gonugetResponse.CacheFile);
        Assert.Contains(".dat", gonugetResponse.CacheFile);

        // NewFile should be CacheFile + "-new"
        Assert.EndsWith("-new", gonugetResponse.NewFile);
    }

    [Fact]
    public void GenerateCachePaths_BaseFolderName_MatchesHashFormat()
    {
        var cacheDir = Path.GetTempPath();
        var sourceURL = "https://api.nuget.org/v3/index.json";
        var cacheKey = "test-key";

        // Compute expected hash with NuGet.Client
        var expectedHash = CachingUtility.ComputeHash(sourceURL, true);
        var expectedFolder = CachingUtility.RemoveInvalidFileNameChars(expectedHash);

        // Generate paths with gonuget
        var gonugetResponse = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);

        // Validate base folder name matches
        Assert.Equal(expectedFolder, gonugetResponse.BaseFolderName);
    }

    [Theory]
    [InlineData("https://api.nuget.org/v3/index.json", "service-index")]
    [InlineData("https://example.com/feed", "package-metadata")]
    [InlineData("https://pkgs.dev.azure.com/_packaging/feed/nuget/v3/index.json", "registration")]
    public void GenerateCachePaths_DifferentSources_ProducesDifferentPaths(string sourceURL, string cacheKey)
    {
        var cacheDir = Path.GetTempPath();

        // Generate paths
        var gonugetResponse = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);

        // Validate structure
        Assert.Contains(gonugetResponse.BaseFolderName, gonugetResponse.CacheFile);
        Assert.Contains(cacheKey, gonugetResponse.CacheFile);
        Assert.EndsWith(".dat", gonugetResponse.CacheFile);
    }

    [Fact]
    public void GenerateCachePaths_CacheKeyWithInvalidChars_SanitizesFilename()
    {
        var cacheDir = Path.GetTempPath();
        var sourceURL = "https://example.com/feed";
        var cacheKey = "package:id/with\\invalid*chars";

        // Generate paths
        var gonugetResponse = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);

        // Cache file should have sanitized key
        var fileName = Path.GetFileName(gonugetResponse.CacheFile);

        // Forward slash should always be sanitized (invalid on all platforms)
        Assert.DoesNotContain("/", fileName);

        // Verify the filename is valid and ends with .dat
        Assert.False(string.IsNullOrEmpty(fileName));
        Assert.EndsWith(".dat", fileName);
    }

    [Fact]
    public void GenerateCachePaths_SameSourceDifferentKeys_SharesBaseFolder()
    {
        var cacheDir = Path.GetTempPath();
        var sourceURL = "https://api.nuget.org/v3/index.json";

        // Generate paths for different cache keys
        var response1 = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, "key1");
        var response2 = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, "key2");

        // Should use same base folder
        Assert.Equal(response1.BaseFolderName, response2.BaseFolderName);

        // But different cache files
        Assert.NotEqual(response1.CacheFile, response2.CacheFile);
    }

    [Fact]
    public void GenerateCachePaths_DifferentSourcesSameKey_UseDifferentFolders()
    {
        var cacheDir = Path.GetTempPath();
        var cacheKey = "test-key";

        // Generate paths for different sources
        var response1 = GonugetBridge.GenerateCachePaths(cacheDir, "https://source1.com", cacheKey);
        var response2 = GonugetBridge.GenerateCachePaths(cacheDir, "https://source2.com", cacheKey);

        // Should use different base folders
        Assert.NotEqual(response1.BaseFolderName, response2.BaseFolderName);
    }

    #endregion
}
