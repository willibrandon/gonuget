using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Commands;
using NuGet.Common;
using NuGet.Configuration;
using NuGet.DependencyResolver;
using NuGet.Frameworks;
using NuGet.LibraryModel;
using NuGet.Packaging.Core;
using NuGet.Protocol.Core.Types;
using NuGet.Versioning;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Dependency resolution tests comparing gonuget's resolver against NuGet.Client.
/// Validates graph walking, disposition states, and parent chain tracking.
/// </summary>
public sealed class ResolverTests
{
    private readonly SourceRepository _sourceRepository;
    private readonly SourceCacheContext _cacheContext;

    public ResolverTests()
    {
        // Setup NuGet.org V3 repository for NuGet.Client
        var packageSource = new PackageSource("https://api.nuget.org/v3/index.json");
        _sourceRepository = new SourceRepository(packageSource, Repository.Provider.GetCoreV3());
        _cacheContext = new SourceCacheContext();
    }

    /// <summary>
    /// Helper method to create a RemoteDependencyWalker for NuGet.Client comparison tests.
    /// </summary>
    private RemoteDependencyWalker CreateRemoteDependencyWalker()
    {
        var packageSourceMapping = PackageSourceMapping.GetPackageSourceMapping(NullSettings.Instance);
        var remoteWalkerContext = new RemoteWalkContext(
            _cacheContext,
            packageSourceMapping,
            NullLogger.Instance);

        var dependencyProvider = new SourceRepositoryDependencyProvider(
            _sourceRepository,
            NullLogger.Instance,
            _cacheContext,
            ignoreFailedSources: false,
            ignoreWarning: false);
        remoteWalkerContext.RemoteLibraryProviders.Add(dependencyProvider);

        return new RemoteDependencyWalker(remoteWalkerContext);
    }

    /// <summary>
    /// Test 1: Simple package with no dependencies - verifies that gonuget builds the same
    /// graph structure as NuGet.Client's RemoteDependencyWalker.
    /// </summary>
    [Fact]
    public async Task WalkGraph_SimplePackageNoDependencies_MatchesNuGetClient()
    {
        // Arrange
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client
        var dependencyInfoResource = await _sourceRepository.GetResourceAsync<DependencyInfoResource>();
        var resolvedPackage = await dependencyInfoResource.ResolvePackage(
            new PackageIdentity(packageId, NuGetVersion.Parse("13.0.1")),
            targetFramework,
            _cacheContext,
            NuGet.Common.NullLogger.Instance,
            CancellationToken.None);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert: Compare root node
        Assert.NotNull(resolvedPackage);

        // Find root node in flat array (depth 0)
        var rootNode = gonugetResult.Nodes.First(n => n.Depth == 0);
        Assert.Equal(resolvedPackage.Id, rootNode.PackageId);
        Assert.Equal(resolvedPackage.Version.ToString(), rootNode.Version);
        Assert.Equal(0, rootNode.Depth);

        // Newtonsoft.Json 13.0.1 has no dependencies
        var childNodes = gonugetResult.Nodes.Where(n => n.Depth == 1).ToList();
        Assert.Empty(childNodes);
    }

    /// <summary>
    /// Test 2: Package with dependencies - verifies gonuget walks the full dependency tree
    /// matching NuGet.Client's behavior.
    /// </summary>
    [Fact]
    public async Task WalkGraph_PackageWithDependencies_MatchesNuGetClient()
    {
        // Arrange
        var packageId = "Microsoft.Extensions.Logging";
        var versionRange = "[8.0.0]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client
        var dependencyInfoResource = await _sourceRepository.GetResourceAsync<DependencyInfoResource>();
        var resolvedPackage = await dependencyInfoResource.ResolvePackage(
            new PackageIdentity(packageId, NuGetVersion.Parse("8.0.0")),
            targetFramework,
            _cacheContext,
            NullLogger.Instance,
            CancellationToken.None);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert: Compare root node
        Assert.NotNull(resolvedPackage);

        // Find root node in flat array (depth 0)
        var rootNode = gonugetResult.Nodes.First(n => n.Depth == 0);
        Assert.Equal(resolvedPackage.Id, rootNode.PackageId);
        Assert.Equal(resolvedPackage.Version.ToString(), rootNode.Version);
        Assert.Equal(0, rootNode.Depth);

        // Verify dependencies exist
        var nugetDependencies = resolvedPackage.Dependencies.ToList();

        if (nugetDependencies.Count != 0)
        {
            // gonuget should have at least one dependency
            var childNodes = gonugetResult.Nodes.Where(n => n.Depth == 1).ToList();
            Assert.NotEmpty(childNodes);

            // Verify first child has correct depth
            var firstChild = childNodes.First();
            Assert.Equal(1, firstChild.Depth);

            // Verify root node lists this child in its dependencies
            Assert.Contains(firstChild.PackageId, rootNode.Dependencies);
        }
    }

    /// <summary>
    /// Test 3: Verify disposition states match NuGet.Client's GraphNode.Disposition.
    /// Uses RemoteDependencyWalker to build a full graph with disposition states.
    /// </summary>
    [Fact]
    public async Task WalkGraph_DispositionStates_MatchesNuGetClientConventions()
    {
        // Arrange
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client: Use RemoteDependencyWalker to build graph with disposition states
        var packageSourceMapping = PackageSourceMapping.GetPackageSourceMapping(NullSettings.Instance);
        var remoteWalkerContext = new RemoteWalkContext(
            _cacheContext,
            packageSourceMapping,
            NullLogger.Instance);

        // Add remote dependency provider for NuGet.org
        var dependencyProvider = new SourceRepositoryDependencyProvider(
            _sourceRepository,
            NullLogger.Instance,
            _cacheContext,
            ignoreFailedSources: false,
            ignoreWarning: false);
        remoteWalkerContext.RemoteLibraryProviders.Add(dependencyProvider);

        var remoteWalker = new RemoteDependencyWalker(remoteWalkerContext);

        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse("13.0.1")),
            LibraryDependencyTarget.Package);

        var nugetGraphNode = await remoteWalker.WalkAsync(
            libraryRange,
            targetFramework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert: Compare disposition states
        Assert.NotNull(nugetGraphNode);
        Assert.NotEmpty(gonugetResult.Nodes);

        // Find root node in flat array (depth 0)
        var rootNode = gonugetResult.Nodes.First(n => n.Depth == 0);

        // Compare root node disposition
        var expectedRootDisposition = nugetGraphNode.Disposition.ToString();
        Assert.Equal(expectedRootDisposition, rootNode.Disposition);

        // Compare child nodes disposition states
        var nugetChildren = nugetGraphNode.InnerNodes.ToList();
        var gonugetChildren = gonugetResult.Nodes.Where(n => n.Depth == 1).ToList();

        // Verify all child nodes have matching disposition states
        foreach (var gonugetChild in gonugetChildren)
        {
            // Find matching node in NuGet.Client graph by package ID
            var matchingNugetNode = nugetChildren.FirstOrDefault(n =>
                n.Item?.Key?.Name == gonugetChild.PackageId);

            Assert.NotNull(matchingNugetNode);

            var expectedDisposition = matchingNugetNode.Disposition.ToString();
            Assert.True(
                expectedDisposition == gonugetChild.Disposition,
                $"Disposition mismatch for {gonugetChild.PackageId}: expected {expectedDisposition}, got {gonugetChild.Disposition}");
        }
    }

    /// <summary>
    /// Test 4 (M5.1): Simple package should have "Acceptable" disposition state.
    /// Validates that gonuget's disposition states match NuGet.Client exactly.
    /// </summary>
    [Fact]
    public async Task WalkGraph_SimplePackage_DispositionAcceptable()
    {
        // Arrange
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client
        var remoteWalker = CreateRemoteDependencyWalker();
        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse("13.0.1")),
            LibraryDependencyTarget.Package);

        var nugetGraphNode = await remoteWalker.WalkAsync(
            libraryRange,
            targetFramework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert - Compare disposition
        Assert.NotNull(nugetGraphNode);
        var rootNode = gonugetResult.Nodes.First(n => n.Depth == 0);

        var expectedDisposition = nugetGraphNode.Disposition.ToString();
        Assert.Equal(expectedDisposition, rootNode.Disposition);
        Assert.Equal(nugetGraphNode.Item.Key.Name, rootNode.PackageId);
        Assert.Equal(nugetGraphNode.Item.Key.Version.ToString(), rootNode.Version);
    }

    /// <summary>
    /// Test 5 (M5.1): Transitive dependencies should have correct depth values.
    /// Validates depth tracking matches NuGet.Client across the full dependency tree.
    /// </summary>
    [Fact]
    public async Task WalkGraph_TransitiveDependencies_CorrectDepth()
    {
        // Arrange
        var packageId = "Microsoft.Extensions.Logging";
        var versionRange = "[8.0.0]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client
        var remoteWalker = CreateRemoteDependencyWalker();
        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse("8.0.0")),
            LibraryDependencyTarget.Package);

        var nugetGraphNode = await remoteWalker.WalkAsync(
            libraryRange,
            targetFramework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert - Compare root
        Assert.NotNull(nugetGraphNode);
        var rootNode = gonugetResult.Nodes.First(n => n.Depth == 0);
        Assert.Equal(0, rootNode.Depth);

        // Compare direct dependencies depth
        var nugetDirectDeps = nugetGraphNode.InnerNodes.ToList();
        var gonugetDirectDeps = gonugetResult.Nodes.Where(n => n.Depth == 1).ToList();

        // Verify all NuGet.Client deps are present in gonuget with correct depth
        foreach (var nugetDep in nugetDirectDeps)
        {
            var gonugetDep = gonugetDirectDeps.FirstOrDefault(g =>
                g.PackageId == nugetDep.Item?.Key?.Name);

            Assert.NotNull(gonugetDep);
            Assert.Equal(1, gonugetDep.Depth);
            Assert.Contains(gonugetDep.PackageId, rootNode.Dependencies);
        }
    }

    /// <summary>
    /// Test 6 (M5.2): Framework-specific dependencies should vary by target framework.
    /// Validates framework-specific dependency selection matches NuGet.Client.
    /// </summary>
    [Fact]
    public async Task WalkGraph_FrameworkSpecific_DifferentDependencies()
    {
        // Arrange
        var packageId = "Microsoft.Extensions.Configuration";
        var version = "8.0.0";
        var net6Framework = NuGetFramework.Parse("net6.0");
        var net8Framework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client with net6.0
        var remoteWalker = CreateRemoteDependencyWalker();
        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse(version)),
            LibraryDependencyTarget.Package);

        var nugetNet6Graph = await remoteWalker.WalkAsync(
            libraryRange,
            net6Framework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        var nugetNet8Graph = await remoteWalker.WalkAsync(
            libraryRange,
            net8Framework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget with both frameworks
        var gonugetNet6Result = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: $"[{version}]",
            targetFramework: "net6.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        var gonugetNet8Result = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: $"[{version}]",
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert - Both should resolve same root package
        Assert.NotNull(nugetNet6Graph);
        Assert.NotNull(nugetNet8Graph);

        var net6Root = gonugetNet6Result.Nodes.First(n => n.Depth == 0);
        var net8Root = gonugetNet8Result.Nodes.First(n => n.Depth == 0);

        Assert.Equal(nugetNet6Graph.Item.Key.Name, net6Root.PackageId);
        Assert.Equal(nugetNet8Graph.Item.Key.Name, net8Root.PackageId);
        Assert.Equal(version, net6Root.Version);
        Assert.Equal(version, net8Root.Version);

        // Verify dependency counts match between frameworks
        var nugetNet6Deps = nugetNet6Graph.InnerNodes.Count;
        var nugetNet8Deps = nugetNet8Graph.InnerNodes.Count;
        var gonugetNet6Deps = gonugetNet6Result.Nodes.Count(n => n.Depth == 1);
        var gonugetNet8Deps = gonugetNet8Result.Nodes.Count(n => n.Depth == 1);

        Assert.Equal(nugetNet6Deps, gonugetNet6Deps);
        Assert.Equal(nugetNet8Deps, gonugetNet8Deps);
    }

    /// <summary>
    /// Test 7 (M5.3): Package graph should detect cycles matching NuGet.Client.
    /// Validates cycle detection disposition matches NuGet.Client exactly.
    /// </summary>
    [Fact]
    public async Task WalkGraph_PackageWithCycle_DetectsCycle()
    {
        // Arrange - Use a package that's known to work
        var packageId = "Microsoft.AspNetCore.App.Ref";
        var version = "8.0.0";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client
        var remoteWalker = CreateRemoteDependencyWalker();
        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse(version)),
            LibraryDependencyTarget.Package);

        var nugetGraph = await remoteWalker.WalkAsync(
            libraryRange,
            targetFramework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: $"[{version}]",
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert - Verify both handle the package the same way
        Assert.NotNull(nugetGraph);
        Assert.NotNull(gonugetResult.Cycles);
        Assert.NotEmpty(gonugetResult.Nodes);

        var rootNode = gonugetResult.Nodes.First(n => n.Depth == 0);
        Assert.Equal(nugetGraph.Item.Key.Name, rootNode.PackageId);
        Assert.Equal(nugetGraph.Item.Key.Version.ToString(), rootNode.Version);

        // Check if NuGet.Client detected cycles (Disposition == Cycle)
        var nugetHasCycles = CountCycleNodes(nugetGraph) > 0;
        var gonugetHasCycles = gonugetResult.Cycles.Length > 0 ||
                               gonugetResult.Nodes.Any(n => n.Disposition == "Cycle");

        // Both should agree on whether cycles exist
        Assert.Equal(nugetHasCycles, gonugetHasCycles);
    }

    private static int CountCycleNodes(GraphNode<RemoteResolveResult> node)
    {
        if (node == null) return 0;

        int count = node.Disposition == Disposition.Cycle ? 1 : 0;
        foreach (var child in node.InnerNodes)
        {
            count += CountCycleNodes(child);
        }
        return count;
    }

    /// <summary>
    /// Test 8 (M5.3): Downgrade detection should match NuGet.Client's PotentiallyDowngraded disposition.
    /// Validates downgrade warnings match NuGet.Client exactly.
    /// </summary>
    [Fact]
    public async Task WalkGraph_ConflictWithDowngrade_DetectsDowngrade()
    {
        // Arrange
        var packageId = "Microsoft.Extensions.Logging";
        var version = "8.0.0";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client
        var remoteWalker = CreateRemoteDependencyWalker();
        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse(version)),
            LibraryDependencyTarget.Package);

        var nugetGraph = await remoteWalker.WalkAsync(
            libraryRange,
            targetFramework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget
        var gonugetResult = GonugetBridge.WalkGraph(
            packageId: packageId,
            versionRange: $"[{version}]",
            targetFramework: "net8.0",
            sources: ["https://api.nuget.org/v3/"]
        );

        // Assert - Downgrades array should exist
        Assert.NotNull(gonugetResult.Downgrades);

        // Count PotentiallyDowngraded nodes in NuGet.Client graph
        var nugetDowngrades = CountDowngradeNodes(nugetGraph);
        var gonugetDowngrades = gonugetResult.Downgrades.Length;

        // Both should agree on downgrade count
        Assert.Equal(nugetDowngrades, gonugetDowngrades);

        // Verify structure of downgrade info
        foreach (var downgrade in gonugetResult.Downgrades)
        {
            Assert.NotEmpty(downgrade.PackageId);
            Assert.NotEmpty(downgrade.FromVersion);
            Assert.NotEmpty(downgrade.ToVersion);
        }
    }

    private static int CountDowngradeNodes(GraphNode<RemoteResolveResult> node)
    {
        if (node == null) return 0;

        int count = node.Disposition == Disposition.PotentiallyDowngraded ? 1 : 0;
        foreach (var child in node.InnerNodes)
        {
            count += CountDowngradeNodes(child);
        }
        return count;
    }

    /// <summary>
    /// Test 9 (M5.4): Conflict resolution should match NuGet.Client's nearest-wins behavior.
    /// Validates that resolved packages match NuGet.Client's Accepted nodes.
    /// </summary>
    [Fact]
    public async Task ResolveConflicts_NearestWins_SelectsClosestVersion()
    {
        // Arrange - Package with transitive dependencies that may have conflicts
        var packageId = "Microsoft.Extensions.Logging";
        var version = "8.0.0";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client: Walk graph to see conflict resolution
        var remoteWalker = CreateRemoteDependencyWalker();
        var libraryRange = new LibraryRange(
            packageId,
            new VersionRange(NuGetVersion.Parse(version)),
            LibraryDependencyTarget.Package);

        var nugetGraph = await remoteWalker.WalkAsync(
            libraryRange,
            targetFramework,
            runtimeIdentifier: string.Empty,
            runtimeGraph: null,
            recursive: true);

        // Act - gonuget: Use ResolveConflicts API
        var gonugetResult = GonugetBridge.ResolveConflicts(
            packageIds: [packageId],
            versionRanges: [$"[{version}]"],
            targetFramework: "net8.0"
        );

        // Assert - Verify structure
        Assert.NotNull(gonugetResult.Packages);
        Assert.NotEmpty(gonugetResult.Packages);

        // Run conflict resolution on NuGet.Client graph using Analyze()
        var analyzeResult = nugetGraph.Analyze();

        // Collect all Accepted/Acceptable nodes from NuGet.Client (conflict resolution winners)
        var nugetResolvedPackages = new Dictionary<string, string>();
        CollectResolvedPackages(nugetGraph, nugetResolvedPackages);

        // Verify each gonuget package has required properties
        foreach (var package in gonugetResult.Packages)
        {
            Assert.NotEmpty(package.PackageId);
            Assert.NotEmpty(package.Version);
            Assert.True(package.Depth >= 0);
        }

        // Verify no duplicate package IDs in resolved set
        var packageIdCounts = gonugetResult.Packages.GroupBy(p => p.PackageId);
        foreach (var group in packageIdCounts)
        {
            Assert.Single(group); // Each package ID should appear exactly once
        }

        // Compare with NuGet.Client: verify gonuget resolved the same packages
        Assert.NotEmpty(nugetResolvedPackages);
        foreach (var nugetPkg in nugetResolvedPackages)
        {
            var gonugetPkg = gonugetResult.Packages.FirstOrDefault(p => p.PackageId == nugetPkg.Key);
            Assert.NotNull(gonugetPkg);
            Assert.Equal(nugetPkg.Value, gonugetPkg.Version);
        }

        // Verify gonuget didn't resolve extra packages that NuGet.Client rejected
        foreach (var gonugetPkg in gonugetResult.Packages)
        {
            Assert.True(
                nugetResolvedPackages.ContainsKey(gonugetPkg.PackageId),
                $"gonuget resolved {gonugetPkg.PackageId} but NuGet.Client didn't accept it");
        }
    }

    private static void CollectResolvedPackages(GraphNode<RemoteResolveResult> node, Dictionary<string, string> resolved)
    {
        if (node == null) return;

        // After conflict resolution, winning nodes are marked Accepted
        // Nodes without conflicts remain Acceptable
        // Both represent packages that should be in the final resolution
        if ((node.Disposition == Disposition.Accepted || node.Disposition == Disposition.Acceptable)
            && node.Item?.Key?.Name != null)
        {
            var packageId = node.Item.Key.Name;
            var version = node.Item.Key.Version?.ToString() ?? "";
            if (!resolved.ContainsKey(packageId))
            {
                resolved[packageId] = version;
            }
        }

        foreach (var child in node.InnerNodes)
        {
            CollectResolvedPackages(child, resolved);
        }
    }
}
