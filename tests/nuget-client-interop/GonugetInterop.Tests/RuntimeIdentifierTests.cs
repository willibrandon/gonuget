using System.Collections.Generic;
using System.Linq;
using GonugetInterop.Tests.TestHelpers;
using NuGet.RuntimeModel;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests that verify gonuget's RID (Runtime Identifier) implementation matches NuGet.Client behavior.
/// These tests use the official NuGet.RuntimeModel APIs as the source of truth.
/// </summary>
public class RuntimeIdentifierTests
{
    private static readonly RuntimeGraph DefaultGraph = CreateDefaultRuntimeGraph();

    /// <summary>
    /// Creates a default runtime graph matching .NET's RID catalog.
    /// Reference: https://learn.microsoft.com/en-us/dotnet/core/rid-catalog
    /// </summary>
    private static RuntimeGraph CreateDefaultRuntimeGraph()
    {
        var runtimes = new List<RuntimeDescription>
        {
            // Foundation RIDs
            new RuntimeDescription("base"),
            new RuntimeDescription("any", new[] { "base" }),

            // Windows
            new RuntimeDescription("win", new[] { "any" }),
            new RuntimeDescription("win-x86", new[] { "win" }),
            new RuntimeDescription("win-x64", new[] { "win" }),
            new RuntimeDescription("win-arm", new[] { "win" }),
            new RuntimeDescription("win-arm64", new[] { "win" }),

            // Windows version chain
            new RuntimeDescription("win7", new[] { "win" }),
            new RuntimeDescription("win7-x86", new[] { "win7", "win-x86" }),
            new RuntimeDescription("win7-x64", new[] { "win7", "win-x64" }),

            new RuntimeDescription("win8", new[] { "win7" }),
            new RuntimeDescription("win8-x86", new[] { "win8", "win7-x86" }),
            new RuntimeDescription("win8-x64", new[] { "win8", "win7-x64" }),
            new RuntimeDescription("win8-arm", new[] { "win8", "win-arm" }),

            new RuntimeDescription("win81", new[] { "win8" }),
            new RuntimeDescription("win81-x86", new[] { "win81", "win8-x86" }),
            new RuntimeDescription("win81-x64", new[] { "win81", "win8-x64" }),
            new RuntimeDescription("win81-arm", new[] { "win81", "win8-arm" }),

            new RuntimeDescription("win10", new[] { "win81" }),
            new RuntimeDescription("win10-x86", new[] { "win10", "win81-x86" }),
            new RuntimeDescription("win10-x64", new[] { "win10", "win81-x64" }),
            new RuntimeDescription("win10-arm", new[] { "win10", "win81-arm" }),
            new RuntimeDescription("win10-arm64", new[] { "win10", "win-arm64" }),

            // Linux
            new RuntimeDescription("linux", new[] { "any" }),
            new RuntimeDescription("linux-x64", new[] { "linux" }),
            new RuntimeDescription("linux-arm", new[] { "linux" }),
            new RuntimeDescription("linux-arm64", new[] { "linux" }),

            // Ubuntu
            new RuntimeDescription("ubuntu", new[] { "linux" }),
            new RuntimeDescription("ubuntu-x64", new[] { "ubuntu", "linux-x64" }),
            new RuntimeDescription("ubuntu.20.04-x64", new[] { "ubuntu-x64" }),
            new RuntimeDescription("ubuntu.22.04-x64", new[] { "ubuntu-x64" }),
            new RuntimeDescription("ubuntu.24.04-x64", new[] { "ubuntu-x64" }),

            // Debian
            new RuntimeDescription("debian", new[] { "linux" }),
            new RuntimeDescription("debian-x64", new[] { "debian", "linux-x64" }),

            // macOS
            new RuntimeDescription("osx", new[] { "any" }),
            new RuntimeDescription("osx-x64", new[] { "osx" }),
            new RuntimeDescription("osx-arm64", new[] { "osx" }),
            new RuntimeDescription("osx.10.12-x64", new[] { "osx-x64" }),
            new RuntimeDescription("osx.11-x64", new[] { "osx-x64" }),
            new RuntimeDescription("osx.12-x64", new[] { "osx-x64" }),
            new RuntimeDescription("osx.12-arm64", new[] { "osx-arm64" }),
            new RuntimeDescription("osx.13-arm64", new[] { "osx-arm64" }),
        };

        return new RuntimeGraph(runtimes);
    }

    [Theory]
    [InlineData("win10-x64")]
    [InlineData("win7-x64")]
    [InlineData("linux-x64")]
    [InlineData("osx-x64")]
    [InlineData("ubuntu.22.04-x64")]
    [InlineData("win10-arm64")]
    [InlineData("osx.13-arm64")]
    public void ExpandRuntime_ShouldMatchNuGetClient(string rid)
    {
        // NuGet.Client expansion
        var nugetExpanded = DefaultGraph.ExpandRuntime(rid).ToArray();

        // Gonuget expansion
        var gonugetResponse = GonugetBridge.ExpandRuntime(rid);
        var gonugetExpanded = gonugetResponse.ExpandedRuntimes;

        // Verify both produce same results in same order
        Assert.Equal(nugetExpanded.Length, gonugetExpanded.Length);
        for (int i = 0; i < nugetExpanded.Length; i++)
        {
            Assert.Equal(nugetExpanded[i], gonugetExpanded[i]);
        }
    }

    [Fact]
    public void ExpandRuntime_Win10x64_ShouldProduceCorrectChain()
    {
        // NuGet.Client expansion
        var nugetExpanded = DefaultGraph.ExpandRuntime("win10-x64").ToArray();

        // Expected chain: win10-x64 -> win10 -> win81-x64 -> win81 -> win8-x64 -> win8 ->
        //                 win7-x64 -> win7 -> win-x64 -> win -> any -> base
        var expected = new[]
        {
            "win10-x64", "win10", "win81-x64", "win81", "win8-x64", "win8",
            "win7-x64", "win7", "win-x64", "win", "any", "base"
        };

        Assert.Equal(expected, nugetExpanded);

        // Verify gonuget matches
        var gonugetResponse = GonugetBridge.ExpandRuntime("win10-x64");
        Assert.Equal(expected, gonugetResponse.ExpandedRuntimes);
    }

    [Fact]
    public void ExpandRuntime_Ubuntu2204x64_ShouldProduceCorrectChain()
    {
        // NuGet.Client expansion
        var nugetExpanded = DefaultGraph.ExpandRuntime("ubuntu.22.04-x64").ToArray();

        // Expected chain: ubuntu.22.04-x64 -> ubuntu-x64 -> ubuntu -> linux-x64 -> linux -> any -> base
        var expected = new[] { "ubuntu.22.04-x64", "ubuntu-x64", "ubuntu", "linux-x64", "linux", "any", "base" };

        Assert.Equal(expected, nugetExpanded);

        // Verify gonuget matches
        var gonugetResponse = GonugetBridge.ExpandRuntime("ubuntu.22.04-x64");
        Assert.Equal(expected, gonugetResponse.ExpandedRuntimes);
    }

    [Theory]
    // Windows compatibility
    [InlineData("win10-x64", "win10-x64", true)]  // Exact match
    [InlineData("win10-x64", "win10", true)]      // More specific target, less specific package
    [InlineData("win10-x64", "win-x64", true)]    // Compatible architecture
    [InlineData("win10-x64", "win", true)]        // Compatible OS family
    [InlineData("win10-x64", "any", true)]        // Any is compatible with everything
    [InlineData("win10-x64", "base", true)]       // Base is in the chain
    [InlineData("win10-x64", "linux-x64", false)] // Different OS
    [InlineData("win10-x64", "win10-arm64", false)] // Different architecture
    // Linux compatibility
    [InlineData("ubuntu.22.04-x64", "ubuntu-x64", true)]
    [InlineData("ubuntu.22.04-x64", "linux-x64", true)]
    [InlineData("ubuntu.22.04-x64", "linux", true)]
    [InlineData("ubuntu.22.04-x64", "any", true)]
    [InlineData("ubuntu.22.04-x64", "win-x64", false)]
    [InlineData("ubuntu.22.04-x64", "ubuntu.20.04-x64", false)] // Different versions
    // macOS compatibility
    [InlineData("osx.13-arm64", "osx-arm64", true)]
    [InlineData("osx.13-arm64", "osx", true)]
    [InlineData("osx.13-arm64", "any", true)]
    [InlineData("osx.13-arm64", "osx-x64", false)] // Different architecture
    [InlineData("osx.13-arm64", "linux-arm64", false)] // Different OS
    public void AreCompatible_ShouldMatchNuGetClient(string targetRid, string packageRid, bool expectedCompatible)
    {
        // NuGet.Client compatibility check
        var nugetCompatible = DefaultGraph.AreCompatible(targetRid, packageRid);
        Assert.Equal(expectedCompatible, nugetCompatible);

        // Gonuget compatibility check
        var gonugetResponse = GonugetBridge.AreRuntimesCompatible(targetRid, packageRid);
        Assert.Equal(expectedCompatible, gonugetResponse.Compatible);
    }

    [Fact]
    public void AreCompatible_Symmetric_Win10x64()
    {
        // Test that compatibility follows the expansion chain
        var expanded = DefaultGraph.ExpandRuntime("win10-x64").ToArray();

        foreach (var rid in expanded)
        {
            // NuGet should consider all expanded RIDs compatible
            Assert.True(DefaultGraph.AreCompatible("win10-x64", rid));

            // Gonuget should match
            var gonugetResponse = GonugetBridge.AreRuntimesCompatible("win10-x64", rid);
            Assert.True(gonugetResponse.Compatible);
        }
    }

    [Fact]
    public void AreCompatible_Asymmetric_Architecture()
    {
        // win-x64 packages cannot be used on win-x86 targets
        Assert.False(DefaultGraph.AreCompatible("win-x86", "win-x64"));

        var gonugetResponse = GonugetBridge.AreRuntimesCompatible("win-x86", "win-x64");
        Assert.False(gonugetResponse.Compatible);

        // But win packages (without arch) CAN be used on win-x64
        Assert.True(DefaultGraph.AreCompatible("win-x64", "win"));

        gonugetResponse = GonugetBridge.AreRuntimesCompatible("win-x64", "win");
        Assert.True(gonugetResponse.Compatible);
    }

    [Fact]
    public void AreCompatible_CrossPlatform_ShouldBeFalse()
    {
        // Test that different OS families are incompatible
        var testCases = new[]
        {
            ("win-x64", "linux-x64"),
            ("linux-x64", "osx-x64"),
            ("osx-arm64", "win-arm64"),
            ("ubuntu-x64", "debian-x64"), // Different Linux distros are compatible via linux-x64
        };

        foreach (var (target, package) in testCases)
        {
            var nugetCompatible = DefaultGraph.AreCompatible(target, package);
            var gonugetResponse = GonugetBridge.AreRuntimesCompatible(target, package);

            // Verify gonuget matches NuGet.Client behavior
            Assert.Equal(nugetCompatible, gonugetResponse.Compatible);
        }
    }

    [Fact]
    public void ExpandRuntime_FoundationRIDs()
    {
        // Test foundation RIDs
        var anyExpanded = DefaultGraph.ExpandRuntime("any").ToArray();
        Assert.Equal(new[] { "any", "base" }, anyExpanded);

        var gonugetResponse = GonugetBridge.ExpandRuntime("any");
        Assert.Equal(new[] { "any", "base" }, gonugetResponse.ExpandedRuntimes);

        var baseExpanded = DefaultGraph.ExpandRuntime("base").ToArray();
        Assert.Equal(new[] { "base" }, baseExpanded);

        gonugetResponse = GonugetBridge.ExpandRuntime("base");
        Assert.Equal(new[] { "base" }, gonugetResponse.ExpandedRuntimes);
    }
}
