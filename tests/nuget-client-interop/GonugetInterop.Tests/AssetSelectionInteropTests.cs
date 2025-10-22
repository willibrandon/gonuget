using System.Collections.Generic;
using System.Linq;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Client;
using NuGet.ContentModel;
using NuGet.Frameworks;
using NuGet.RuntimeModel;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Interop tests comparing NuGet.Client and gonuget asset selection behavior.
/// These tests verify that gonuget's ContentModel implementation produces
/// identical results to NuGet.Client's FindBestItemGroup logic.
/// </summary>
public class AssetSelectionInteropTests
{
    [Fact]
    public void FindRuntimeAssemblies_SelectsExactFrameworkMatch()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "lib/net7.0/MyLib.dll",
            "lib/netstandard2.1/StandardLib.dll"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net6.0"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "net6.0");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Contains("lib/net6.0/MyLib.dll", nugetPaths);
    }

    [Fact]
    public void FindRuntimeAssemblies_SelectsCompatibleFramework()
    {
        // Arrange - netstandard2.1 is compatible with net6.0
        var paths = new[]
        {
            "lib/netstandard2.1/StandardLib.dll",
            "lib/netstandard2.0/StandardLib.dll"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net6.0"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "net6.0");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Contains("lib/netstandard2.1/StandardLib.dll", nugetPaths);
    }

    [Fact]
    public void FindRuntimeAssemblies_NoMatchReturnsEmpty()
    {
        // Arrange - net7.0 is not compatible with net45
        var paths = new[] { "lib/net7.0/MyLib.dll" };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net45"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "net45");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Empty(nugetPaths);
        Assert.Empty(gonugetPaths);
    }

    [Fact]
    public void FindCompileAssemblies_RefTakesPrecedenceOverLib()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "ref/net6.0/MyLib.dll"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net6.0"));
        var group = collection.FindBestItemGroup(
            criteria,
            conventions.Patterns.CompileRefAssemblies,
            conventions.Patterns.CompileLibAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindCompileAssemblies(paths, "net6.0");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Contains("ref/net6.0/MyLib.dll", nugetPaths);
    }

    [Fact]
    public void FindCompileAssemblies_FallbackToLibWhenNoRef()
    {
        // Arrange
        var paths = new[] { "lib/net6.0/MyLib.dll" };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net6.0"));
        var group = collection.FindBestItemGroup(
            criteria,
            conventions.Patterns.CompileRefAssemblies,
            conventions.Patterns.CompileLibAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindCompileAssemblies(paths, "net6.0");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Contains("lib/net6.0/MyLib.dll", nugetPaths);
    }

    [Fact]
    public void FindRuntimeAssemblies_RuntimeAgnosticFallback()
    {
        // Arrange - lib/netcore50 should be selected over runtimes/aot/lib/netcore50
        // when using framework-only criteria (no RID)
        var paths = new[]
        {
            "runtimes/aot/lib/netcore50/System.Reflection.Emit.dll",
            "lib/netcore50/System.Reflection.Emit.dll"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(
            new RuntimeGraph(new List<CompatibilityProfile> { new("netcore50.app") }));
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("netcore50"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "netcore50");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Equal("lib/netcore50/System.Reflection.Emit.dll", nugetPaths[0]);
    }

    [Fact]
    public void FindRuntimeAssemblies_MultipleFrameworksSelectsNearest()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net45/MyLib.dll",
            "lib/net46/MyLib.dll",
            "lib/net47/MyLib.dll"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net47"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "net47");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Contains("lib/net47/MyLib.dll", nugetPaths);
    }

    [Fact]
    public void FindRuntimeAssemblies_FiltersToAssembliesOnly()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "lib/net6.0/MyLib.xml",
            "lib/net6.0/MyLib.pdb"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net6.0"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "net6.0");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Single(nugetPaths);
        Assert.Contains("lib/net6.0/MyLib.dll", nugetPaths);
    }

    [Fact]
    public void FindRuntimeAssemblies_IncludesExeAndWinmd()
    {
        // Arrange
        var paths = new[]
        {
            "lib/net6.0/MyLib.dll",
            "lib/net6.0/MyTool.exe",
            "lib/net6.0/MyComponent.winmd"
        };

        // Act - NuGet.Client
        var conventions = new ManagedCodeConventions(new RuntimeGraph());
        var collection = new ContentItemCollection();
        collection.Load(paths);
        var criteria = conventions.Criteria.ForFramework(NuGetFramework.Parse("net6.0"));
        var group = collection.FindBestItemGroup(criteria, conventions.Patterns.RuntimeAssemblies);
        var nugetPaths = group?.Items.Select(i => i.Path).OrderBy(p => p).ToArray() ?? System.Array.Empty<string>();

        // Act - gonuget
        var gonugetResponse = GonugetBridge.FindRuntimeAssemblies(paths, "net6.0");
        var gonugetPaths = gonugetResponse.Items.Select(i => i.Path).OrderBy(p => p).ToArray();

        // Assert
        Assert.Equal(nugetPaths, gonugetPaths);
        Assert.Equal(3, nugetPaths.Length);
    }
}
