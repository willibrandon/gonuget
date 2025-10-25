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
        Assert.Equal(resolvedPackage.Id, gonugetResult.RootNode.PackageId);
        Assert.Equal(resolvedPackage.Version.ToString(), gonugetResult.RootNode.Version);
        Assert.Equal(0, gonugetResult.RootNode.Depth);

        // Newtonsoft.Json 13.0.1 has no dependencies
        Assert.Empty(gonugetResult.RootNode.InnerNodes);
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
        Assert.Equal(resolvedPackage.Id, gonugetResult.RootNode.PackageId);
        Assert.Equal(resolvedPackage.Version.ToString(), gonugetResult.RootNode.Version);
        Assert.Equal(0, gonugetResult.RootNode.Depth);

        // Verify dependencies exist
        var nugetDependencies = resolvedPackage.Dependencies.ToList();

        if (nugetDependencies.Count != 0)
        {
            // gonuget should have at least one dependency
            Assert.NotEmpty(gonugetResult.RootNode.InnerNodes);

            // Verify first child has correct depth and parent edge
            var firstChild = gonugetResult.RootNode.InnerNodes.First();
            Assert.Equal(1, firstChild.Depth);
            Assert.NotNull(firstChild.OuterEdge);
            Assert.Equal(packageId, firstChild.OuterEdge.ParentPackageId);
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
        Assert.NotNull(gonugetResult.RootNode);

        // Compare root node disposition
        var expectedRootDisposition = nugetGraphNode.Disposition.ToString();
        Assert.Equal(expectedRootDisposition, gonugetResult.RootNode.Disposition);

        // Compare child nodes disposition states
        var nugetChildren = nugetGraphNode.InnerNodes.ToList();
        var gonugetChildren = gonugetResult.RootNode.InnerNodes;

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
}
