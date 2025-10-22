using System;
using System.Linq;
using GonugetInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests for NuGet ContentModel (pattern-based asset selection).
/// Validates gonuget's pattern matching and property extraction.
/// </summary>
public sealed class ContentModelTests
{
    // ============================================================================
    // Parse Asset Path Tests
    // ============================================================================

    [Theory]
    [InlineData("lib/net6.0/MyLib.dll", "net6.0", "MyLib.dll")]
    [InlineData("lib/net7.0/AnotherLib.dll", "net7.0", "AnotherLib.dll")]
    [InlineData("lib/netstandard2.1/StandardLib.dll", "netstandard2.1", "StandardLib.dll")]
    public void ParseAssetPath_LibPath_ExtractsProperties(string path, string expectedTfm, string expectedAssembly)
    {
        // Act
        var result = GonugetBridge.ParseAssetPath(path);

        // Assert
        Assert.NotNull(result.Item);
        Assert.Equal(path, result.Item.Path);
        Assert.Equal(expectedTfm, result.Item.Properties["tfm"].ToString());
        Assert.Equal(expectedAssembly, result.Item.Properties["assembly"].ToString());
    }

    [Theory]
    [InlineData("ref/net6.0/MyLib.dll", "net6.0", "MyLib.dll")]
    [InlineData("ref/netstandard2.0/RefLib.dll", "netstandard2.0", "RefLib.dll")]
    public void ParseAssetPath_RefPath_ExtractsProperties(string path, string expectedTfm, string expectedAssembly)
    {
        // Act
        var result = GonugetBridge.ParseAssetPath(path);

        // Assert
        Assert.NotNull(result.Item);
        Assert.Equal(path, result.Item.Path);
        Assert.Equal(expectedTfm, result.Item.Properties["tfm"].ToString());
        Assert.Equal(expectedAssembly, result.Item.Properties["assembly"].ToString());
    }

    [Theory]
    [InlineData("content/readme.txt")]
    [InlineData("build/project.props")]
    [InlineData("random/file.dat")]
    public void ParseAssetPath_NonMatchingPath_ReturnsNull(string path)
    {
        // Act
        var result = GonugetBridge.ParseAssetPath(path);

        // Assert - should not match
        Assert.Null(result.Item);
    }

    // ============================================================================
    // Find Runtime Assemblies Tests
    // ============================================================================

    [Fact]
    public void FindRuntimeAssemblies_MultiplePaths_MatchesLibOnly()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "lib/net7.0/MyLib.dll",
            "lib/netstandard2.1/MyLib.dll",
            "ref/net6.0/RefLib.dll", // Should not match runtime assemblies
            "content/readme.txt"      // Should not match
        };

        // Act
        var result = GonugetBridge.FindRuntimeAssemblies(paths);

        // Assert - should match only lib/ paths
        Assert.Equal(3, result.Items.Length);
        Assert.All(result.Items, item => Assert.StartsWith("lib/", item.Path));
    }

    [Fact]
    public void FindRuntimeAssemblies_MultipleFrameworks_MatchesAll()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "lib/net7.0/MyLib.dll",
            "lib/netstandard2.1/MyLib.dll"
        };

        // Act - pattern matching without targetFramework
        var result = GonugetBridge.FindRuntimeAssemblies(paths);

        // Assert - should match all lib/ paths
        Assert.Equal(3, result.Items.Length);
        Assert.Contains(result.Items, item => item.Path == "lib/net6.0/MyLib.dll");
        Assert.Contains(result.Items, item => item.Path == "lib/net7.0/MyLib.dll");
        Assert.Contains(result.Items, item => item.Path == "lib/netstandard2.1/MyLib.dll");
    }

    // ============================================================================
    // Find Compile Assemblies Tests
    // ============================================================================

    [Fact]
    public void FindCompileAssemblies_RefAndLib_MatchesBoth()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "ref/net6.0/MyLib.dll"
        };

        // Act - pattern matching without targetFramework
        var result = GonugetBridge.FindCompileAssemblies(paths);

        // Assert - should match both ref/ and lib/ patterns
        Assert.Equal(2, result.Items.Length);
        Assert.Contains(result.Items, item => item.Path == "ref/net6.0/MyLib.dll");
        Assert.Contains(result.Items, item => item.Path == "lib/net6.0/MyLib.dll");
    }

    [Fact]
    public void FindCompileAssemblies_OnlyLib_UsesLib()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "lib/net7.0/AnotherLib.dll"
        };

        // Act
        var result = GonugetBridge.FindCompileAssemblies(paths);

        // Assert - should match lib/ paths
        Assert.Equal(2, result.Items.Length);
        Assert.All(result.Items, item => Assert.StartsWith("lib/", item.Path));
    }

    // ============================================================================
    // Property Extraction Tests
    // ============================================================================

    [Theory]
    [InlineData("lib/net6.0/MyLib.dll")]
    [InlineData("lib/netstandard2.1/StandardLib.dll")]
    [InlineData("ref/net7.0/RefLib.dll")]
    public void ParseAssetPath_ExtractsTfmAndAssembly(string path)
    {
        // Act
        var result = GonugetBridge.ParseAssetPath(path);

        // Assert - should extract both properties
        Assert.NotNull(result.Item);
        Assert.True(result.Item.Properties.ContainsKey("tfm"));
        Assert.True(result.Item.Properties.ContainsKey("assembly"));

        // TFM should be a valid framework string
        var tfm = result.Item.Properties["tfm"].ToString();
        Assert.NotNull(tfm);
        Assert.NotEmpty(tfm);

        // Assembly should be the filename
        var assembly = result.Item.Properties["assembly"].ToString();
        Assert.NotNull(assembly);
        Assert.EndsWith(".dll", assembly);
    }
}
