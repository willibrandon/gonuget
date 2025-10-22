using GonugetInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests for NuGet framework (TFM) parsing and compatibility checking.
/// Validates gonuget's framework handling against NuGet.Client behavior.
/// </summary>
public sealed class FrameworkTests
{
    // ============================================================================
    // Basic Framework Parsing Tests (15 tests)
    // ============================================================================

    [Theory]
    [InlineData("net8.0", ".NETCoreApp", "8.0")]
    [InlineData("net7.0", ".NETCoreApp", "7.0")]
    [InlineData("net6.0", ".NETCoreApp", "6.0")]
    [InlineData("net5.0", ".NETCoreApp", "5.0")]
    [InlineData("netcoreapp3.1", ".NETCoreApp", "3.1")]
    public void ParseFramework_NetCoreApp_Success(string tfm, string expectedFramework, string expectedVersion)
    {
        var result = GonugetBridge.ParseFramework(tfm);
        Assert.Equal(expectedFramework, result.Identifier);
        Assert.Equal(expectedVersion, result.Version);
    }

    [Theory]
    [InlineData("netstandard2.1", ".NETStandard", "2.1")]
    [InlineData("netstandard2.0", ".NETStandard", "2.0")]
    [InlineData("netstandard1.6", ".NETStandard", "1.6")]
    [InlineData("netstandard1.0", ".NETStandard", "1.0")]
    public void ParseFramework_NetStandard_Success(string tfm, string expectedFramework, string expectedVersion)
    {
        var result = GonugetBridge.ParseFramework(tfm);
        Assert.Equal(expectedFramework, result.Identifier);
        Assert.Equal(expectedVersion, result.Version);
    }

    [Theory]
    [InlineData("net48", ".NETFramework", "4.8")]
    [InlineData("net472", ".NETFramework", "4.7.2")]
    [InlineData("net471", ".NETFramework", "4.7.1")]
    [InlineData("net47", ".NETFramework", "4.7")]
    [InlineData("net462", ".NETFramework", "4.6.2")]
    [InlineData("net461", ".NETFramework", "4.6.1")]
    public void ParseFramework_NetFrameworkCompact_Success(string tfm, string expectedFramework, string expectedVersion)
    {
        var result = GonugetBridge.ParseFramework(tfm);
        Assert.Equal(expectedFramework, result.Identifier);
        Assert.Equal(expectedVersion, result.Version);
    }

    // ============================================================================
    // Platform-Specific Framework Parsing Tests (8 tests)
    // ============================================================================

    [Theory]
    [InlineData("net6.0-windows", ".NETCoreApp", "6.0", "windows")]
    [InlineData("net7.0-android", ".NETCoreApp", "7.0", "android")]
    [InlineData("net8.0-ios", ".NETCoreApp", "8.0", "ios")]
    [InlineData("net6.0-macos", ".NETCoreApp", "6.0", "macos")]
    public void ParseFramework_PlatformSpecific_Success(
        string tfm,
        string expectedFramework,
        string expectedVersion,
        string expectedPlatform)
    {
        var result = GonugetBridge.ParseFramework(tfm);
        Assert.Equal(expectedFramework, result.Identifier);
        Assert.Equal(expectedVersion, result.Version);
        Assert.Equal(expectedPlatform, result.Platform);
    }

    [Theory]
    [InlineData("net6.0-windows7.0", ".NETCoreApp", "6.0", "windows")]
    [InlineData("net7.0-android31.0", ".NETCoreApp", "7.0", "android")]
    [InlineData("net8.0-ios15.0", ".NETCoreApp", "8.0", "ios")]
    [InlineData("net6.0-macos11.0", ".NETCoreApp", "6.0", "macos")]
    public void ParseFramework_PlatformWithVersion_Success(
        string tfm,
        string expectedFramework,
        string expectedVersion,
        string expectedPlatform)
    {
        var result = GonugetBridge.ParseFramework(tfm);
        Assert.Equal(expectedFramework, result.Identifier);
        Assert.Equal(expectedVersion, result.Version);
        Assert.Equal(expectedPlatform, result.Platform);
    }

    // ============================================================================
    // .NET Standard Compatibility Tests (12 tests)
    // ============================================================================

    [Theory]
    [InlineData("netstandard2.0", "net6.0", true)]
    [InlineData("netstandard2.0", "net7.0", true)]
    [InlineData("netstandard2.0", "net8.0", true)]
    [InlineData("netstandard2.1", "net5.0", true)]
    [InlineData("netstandard2.1", "net6.0", true)]
    [InlineData("netstandard1.6", "netcoreapp3.1", true)]
    public void Compatibility_NetStandard_ToNetCoreApp_Compatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    [Theory]
    [InlineData("netstandard2.0", "net461", true)]
    [InlineData("netstandard2.0", "net462", true)]
    [InlineData("netstandard2.0", "net47", true)]
    [InlineData("netstandard2.0", "net48", true)]
    [InlineData("netstandard1.6", "net461", true)]
    [InlineData("netstandard1.0", "net45", true)]
    public void Compatibility_NetStandard_ToNetFramework_Compatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    // ============================================================================
    // .NET Standard 2.1 Special Cases (4 tests)
    // ============================================================================

    [Theory]
    [InlineData("netstandard2.1", "net48", false)]
    [InlineData("netstandard2.1", "net472", false)]
    [InlineData("netstandard2.1", "net47", false)]
    [InlineData("netstandard2.1", "net461", false)]
    public void Compatibility_NetStandard21_NotCompatible_WithNetFramework(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    // ============================================================================
    // .NET Core Compatibility Tests (6 tests)
    // ============================================================================

    [Theory]
    [InlineData("netcoreapp3.1", "net5.0", true)]
    [InlineData("netcoreapp3.1", "net6.0", true)]
    [InlineData("netcoreapp2.1", "netcoreapp3.1", true)]
    [InlineData("net5.0", "net6.0", true)]
    [InlineData("net6.0", "net7.0", true)]
    [InlineData("net7.0", "net8.0", true)]
    public void Compatibility_NetCoreApp_ForwardCompatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    // ============================================================================
    // .NET Framework Compatibility Tests (5 tests)
    // ============================================================================

    [Theory]
    [InlineData("net461", "net462", true)]
    [InlineData("net461", "net47", true)]
    [InlineData("net461", "net48", true)]
    [InlineData("net47", "net472", true)]
    [InlineData("net472", "net48", true)]
    public void Compatibility_NetFramework_ForwardCompatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    // ============================================================================
    // Incompatibility Tests (6 tests)
    // ============================================================================

    [Theory]
    [InlineData("net6.0", "net5.0", false)]
    [InlineData("net7.0", "net6.0", false)]
    [InlineData("netcoreapp3.1", "netcoreapp2.1", false)]
    public void Compatibility_NetCoreApp_NotBackwardCompatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    [Theory]
    [InlineData("net48", "net47", false)]
    [InlineData("net472", "net461", false)]
    [InlineData("net47", "net462", false)]
    public void Compatibility_NetFramework_NotBackwardCompatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }

    // ============================================================================
    // Cross-Platform Incompatibility Tests (4 tests)
    // ============================================================================

    [Theory]
    [InlineData("net48", "net6.0", false)]
    [InlineData("net48", "netcoreapp3.1", false)]
    [InlineData("net6.0", "net48", false)]
    [InlineData("netcoreapp3.1", "net461", false)]
    public void Compatibility_CrossPlatform_Incompatible(
        string packageFramework,
        string projectFramework,
        bool expectedCompatible)
    {
        var result = GonugetBridge.CheckFrameworkCompat(packageFramework, projectFramework);
        Assert.Equal(expectedCompatible, result.Compatible);
    }
}
