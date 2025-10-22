using GonugetInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Smoke tests to verify all GonugetBridge methods work end-to-end.
/// </summary>
public sealed class BridgeSmokeTests
{
    private static readonly string[] s_testAuthors = ["Test Author"];

    [Fact]
    public void CompareVersions_Works()
    {
        var result = GonugetBridge.CompareVersions("1.0.0", "2.0.0");
        Assert.Equal(-1, result.Result);
    }

    [Fact]
    public void ParseVersion_Works()
    {
        var result = GonugetBridge.ParseVersion("1.2.3-beta.1+git.abc123");
        Assert.Equal(1, result.Major);
        Assert.Equal(2, result.Minor);
        Assert.Equal(3, result.Patch);
        Assert.Equal("beta.1", result.Release);
        Assert.Equal("git.abc123", result.Metadata);
        Assert.True(result.IsPrerelease);
    }

    [Fact]
    public void CheckFrameworkCompat_Works()
    {
        var result = GonugetBridge.CheckFrameworkCompat("net6.0", "net8.0");
        Assert.True(result.Compatible);
    }

    [Fact]
    public void ParseFramework_Works()
    {
        var result = GonugetBridge.ParseFramework("net8.0");
        Assert.Equal(".NETCoreApp", result.Identifier);
        Assert.Equal("8.0", result.Version);
    }

    [Fact]
    public void BuildAndReadPackage_RoundTrip_Works()
    {
        var built = GonugetBridge.BuildPackage(
            "TestPackage",
            "1.0.0",
            s_testAuthors,
            "Test Description");

        Assert.NotEmpty(built.PackageBytes);

        var read = GonugetBridge.ReadPackage(built.PackageBytes);
        Assert.Equal("TestPackage", read.Id);
        Assert.Equal("1.0.0", read.Version);
        Assert.NotNull(read.Authors);
        Assert.Contains("Test Author", read.Authors);
    }
}
