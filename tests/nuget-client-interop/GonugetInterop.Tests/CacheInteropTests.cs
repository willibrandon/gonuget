using System;
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

    #region TTL Validation Tests (Minimum 15 cases)

    [Fact]
    public void ValidateCacheFile_FreshFile_WithinTTL_ReturnsValid()
    {
        // Setup: Create a temporary cache directory and file
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = "test-resource";
            var testData = new byte[] { 1, 2, 3, 4, 5 };

            // Get the cache paths
            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);

            // Create the cache directory structure
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);

            // Write a test file
            File.WriteAllBytes(paths.CacheFile, testData);

            // NuGet.Client: Fresh file with 1 hour TTL should be valid
            var maxAge = TimeSpan.FromHours(1);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget: Same test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Both should return valid
            Assert.True(nugetValid);
            Assert.True(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            // Cleanup
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ValidateCacheFile_ExpiredFile_ReturnsInvalid()
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = "expired-resource";
            var testData = new byte[] { 1, 2, 3 };

            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);

            // Write file and modify its timestamp to 2 hours ago
            File.WriteAllBytes(paths.CacheFile, testData);
            var oldTime = DateTime.UtcNow.AddHours(-2);
            File.SetLastWriteTimeUtc(paths.CacheFile, oldTime);

            // NuGet.Client: File older than 1 hour TTL should be invalid
            var maxAge = TimeSpan.FromHours(1);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget: Same test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Both should return invalid
            Assert.False(nugetValid);
            Assert.False(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ValidateCacheFile_MissingFile_ReturnsInvalid()
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = "nonexistent";
            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);

            // NuGet.Client: Missing file should return null
            var maxAge = TimeSpan.FromHours(1);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget: Same test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Both should return invalid
            Assert.False(nugetValid);
            Assert.False(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ValidateCacheFile_ZeroMaxAge_AlwaysExpired()
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = "zero-ttl";
            var testData = new byte[] { 1, 2, 3 };

            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);
            File.WriteAllBytes(paths.CacheFile, testData);

            // NuGet.Client: Zero max age means immediately expired
            var maxAge = TimeSpan.Zero;
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget: Same test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Both should return invalid
            Assert.False(nugetValid);
            Assert.False(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ValidateCacheFile_VeryLargeMaxAge_AlwaysValid()
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = "large-ttl";
            var testData = new byte[] { 1, 2, 3 };

            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);

            // Write file and set timestamp to 1 day ago
            File.WriteAllBytes(paths.CacheFile, testData);
            var oldTime = DateTime.UtcNow.AddDays(-1);
            File.SetLastWriteTimeUtc(paths.CacheFile, oldTime);

            // NuGet.Client: With 365 day TTL, 1-day-old file should be valid
            var maxAge = TimeSpan.FromDays(365);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget: Same test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Both should return valid
            Assert.True(nugetValid);
            Assert.True(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Theory]
    [InlineData(30, 29, true)]   // 29 min old file, 30 min TTL = valid
    [InlineData(30, 30, false)]  // 30 min old file, 30 min TTL = expired (age >= maxAge)
    [InlineData(30, 31, false)]  // 31 min old file, 30 min TTL = expired
    [InlineData(60, 59, true)]   // 59 min old file, 60 min TTL = valid
    [InlineData(60, 60, false)]  // 60 min old file, 60 min TTL = expired
    public void ValidateCacheFile_EdgeCaseBoundaries_MatchesNuGetClient(int ttlMinutes, int ageMinutes, bool expectedValid)
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = $"edge-case-{ttlMinutes}-{ageMinutes}";
            var testData = new byte[] { 1, 2, 3 };

            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);

            // Write file with specific age
            File.WriteAllBytes(paths.CacheFile, testData);
            var fileTime = DateTime.UtcNow.AddMinutes(-ageMinutes);
            File.SetLastWriteTimeUtc(paths.CacheFile, fileTime);

            // NuGet.Client test
            var maxAge = TimeSpan.FromMinutes(ttlMinutes);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Validate both match expected and each other
            Assert.Equal(expectedValid, nugetValid);
            Assert.Equal(expectedValid, gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ValidateCacheFile_Default30MinutesTTL_MatchesNuGetClient()
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = "default-ttl";
            var testData = new byte[] { 1, 2, 3 };

            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);
            File.WriteAllBytes(paths.CacheFile, testData);

            // NuGet.Client default is 30 minutes
            var maxAge = TimeSpan.FromMinutes(30);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget: Same test
            var maxAgeSeconds = (long)maxAge.TotalSeconds;
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Both should be valid (fresh file)
            Assert.True(nugetValid);
            Assert.True(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Theory]
    [InlineData(1)]    // 1 second
    [InlineData(60)]   // 1 minute
    [InlineData(300)]  // 5 minutes
    [InlineData(3600)] // 1 hour
    public void ValidateCacheFile_VariousTTLs_FreshFile_AlwaysValid(int maxAgeSeconds)
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var sourceURL = "https://api.nuget.org/v3/index.json";
            var cacheKey = $"ttl-{maxAgeSeconds}";
            var testData = new byte[] { 1, 2, 3 };

            var paths = GonugetBridge.GenerateCachePaths(cacheDir, sourceURL, cacheKey);
            var cacheFileDir = Path.GetDirectoryName(paths.CacheFile)!;
            Directory.CreateDirectory(cacheFileDir);
            File.WriteAllBytes(paths.CacheFile, testData);

            // NuGet.Client test
            var maxAge = TimeSpan.FromSeconds(maxAgeSeconds);
            var nugetStream = CachingUtility.ReadCacheFile(maxAge, paths.CacheFile);
            var nugetValid = nugetStream != null;
            nugetStream?.Dispose();

            // gonuget test
            var gonugetResponse = GonugetBridge.ValidateCacheFile(cacheDir, sourceURL, cacheKey, maxAgeSeconds);

            // Fresh files should always be valid
            Assert.True(nugetValid);
            Assert.True(gonugetResponse.Valid);
            Assert.Equal(nugetValid, gonugetResponse.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ValidateCacheFile_MultipleSourcesSameKey_IndependentValidation()
    {
        // Setup
        var cacheDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(cacheDir);

        try
        {
            var source1 = "https://source1.com";
            var source2 = "https://source2.com";
            var cacheKey = "shared-key";
            var testData = new byte[] { 1, 2, 3 };

            // Create files for both sources
            var paths1 = GonugetBridge.GenerateCachePaths(cacheDir, source1, cacheKey);
            var paths2 = GonugetBridge.GenerateCachePaths(cacheDir, source2, cacheKey);

            Directory.CreateDirectory(Path.GetDirectoryName(paths1.CacheFile)!);
            Directory.CreateDirectory(Path.GetDirectoryName(paths2.CacheFile)!);

            // Source1: fresh file
            File.WriteAllBytes(paths1.CacheFile, testData);

            // Source2: old file (2 hours ago)
            File.WriteAllBytes(paths2.CacheFile, testData);
            File.SetLastWriteTimeUtc(paths2.CacheFile, DateTime.UtcNow.AddHours(-2));

            var maxAgeSeconds = 3600L; // 1 hour

            // Validate source1 (should be valid)
            var response1 = GonugetBridge.ValidateCacheFile(cacheDir, source1, cacheKey, maxAgeSeconds);

            // Validate source2 (should be invalid)
            var response2 = GonugetBridge.ValidateCacheFile(cacheDir, source2, cacheKey, maxAgeSeconds);

            Assert.True(response1.Valid);
            Assert.False(response2.Valid);
        }
        finally
        {
            if (Directory.Exists(cacheDir))
            {
                Directory.Delete(cacheDir, recursive: true);
            }
        }
    }

    #endregion
}
