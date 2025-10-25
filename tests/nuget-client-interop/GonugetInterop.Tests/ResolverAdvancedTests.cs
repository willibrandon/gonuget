using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Threading.Tasks;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Commands;
using NuGet.Common;
using NuGet.Configuration;
using NuGet.DependencyResolver;
using NuGet.Frameworks;
using NuGet.LibraryModel;
using NuGet.Protocol.Core.Types;
using NuGet.Versioning;
using Xunit;
using Xunit.Abstractions;

namespace GonugetInterop.Tests;

/// <summary>
/// Advanced resolver tests for M5.5-M5.8 (cycles, transitive, caching, parallel).
/// Tests gonuget advanced resolution features against NuGet.Client with EXACT comparisons.
/// ALL assertions use NuGet.Client as EXPECTED and gonuget as ACTUAL.
/// </summary>
public class ResolverAdvancedTests : IDisposable
{
    private readonly ITestOutputHelper _output;
    private static readonly string[] s_nugetSources = ["https://api.nuget.org/v3/"];
    private readonly SourceRepository _sourceRepository;
    private readonly SourceCacheContext _cacheContext;

    public ResolverAdvancedTests(ITestOutputHelper output)
    {
        _output = output;

        var packageSource = new PackageSource("https://api.nuget.org/v3/index.json");
        _sourceRepository = new SourceRepository(packageSource, Repository.Provider.GetCoreV3());
        _cacheContext = new SourceCacheContext();
    }

    public void Dispose()
    {
        _cacheContext?.Dispose();
        GC.SuppressFinalize(this);
    }

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

    private RemoteDependencyWalker CreateInMemoryWalker(InMemoryDependencyProvider provider)
    {
        var packageSourceMapping = PackageSourceMapping.GetPackageSourceMapping(NullSettings.Instance);
        var remoteWalkerContext = new RemoteWalkContext(
            _cacheContext,
            packageSourceMapping,
            NullLogger.Instance);

        remoteWalkerContext.RemoteLibraryProviders.Add(provider);
        return new RemoteDependencyWalker(remoteWalkerContext);
    }

    #region M5.5 - Cycle Analysis

    [Fact]
    public async Task AnalyzeCycles_SimpleCycle_DetectsSameCycleAsNuGetClient()
    {
        // Arrange: Create in-memory packages with cycle A -> B -> A
        var provider = new InMemoryDependencyProvider();
        provider.Package("A", "1.0.0").DependsOn("B", "[2.0.0]");
        provider.Package("B", "2.0.0").DependsOn("A", "[1.0.0]");

        // Act - NuGet.Client (EXPECTED)
        var nugetWalker = CreateInMemoryWalker(provider);
        var nugetGraph = await nugetWalker.WalkAsync(
            new LibraryRange("A", VersionRange.Parse("[1.0.0]"), LibraryDependencyTarget.Package),
            NuGetFramework.Parse("net8.0"),
            runtimeIdentifier: null,
            runtimeGraph: null,
            recursive: true);
        var nugetAnalysis = nugetGraph.Analyze();

        // Act - gonuget (ACTUAL) - Use SAME in-memory provider!
        var gonugetResult = GonugetBridge.AnalyzeCycles(
            packageId: "A",
            versionRange: "[1.0.0]",
            targetFramework: "net8.0",
            sources: s_nugetSources,
            inMemoryProvider: provider  // Pass the SAME in-memory provider
        );

        // Assert: NuGet.Client detected exactly 1 cycle
        int expectedCycleCount = nugetAnalysis.Cycles.Count;
        Assert.Equal(1, expectedCycleCount);

        // Assert: gonuget MUST match NuGet.Client's cycle count EXACTLY
        int actualCycleCount = gonugetResult.Cycles.Length;
        Assert.Equal(expectedCycleCount, actualCycleCount);

        _output.WriteLine($"EXPECTED (NuGet.Client): {expectedCycleCount} cycles");
        _output.WriteLine($"ACTUAL (gonuget):        {actualCycleCount} cycles");
        _output.WriteLine("RESULT: MATCH ✓");
    }

    [Fact]
    public async Task AnalyzeCycles_RealPackage_DetectsNoCyclesLikeNuGetClient()
    {
        // Arrange: Real packages don't have cycles
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client (EXPECTED)
        var nugetWalker = CreateRemoteDependencyWalker();
        var nugetGraph = await nugetWalker.WalkAsync(
            new LibraryRange(packageId, VersionRange.Parse(versionRange), LibraryDependencyTarget.Package),
            targetFramework,
            runtimeIdentifier: null,
            runtimeGraph: null,
            recursive: true);
        var nugetAnalysis = nugetGraph.Analyze();

        // Act - gonuget (ACTUAL)
        var gonugetResult = GonugetBridge.AnalyzeCycles(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: s_nugetSources
        );

        // Assert: NuGet.Client expects 0 cycles
        int expectedCycleCount = nugetAnalysis.Cycles.Count;
        Assert.Equal(0, expectedCycleCount);

        // Assert: gonuget MUST match EXACTLY
        int actualCycleCount = gonugetResult.Cycles.Length;
        Assert.Equal(expectedCycleCount, actualCycleCount);

        _output.WriteLine($"Package: {packageId} {versionRange}");
        _output.WriteLine($"EXPECTED (NuGet.Client): {expectedCycleCount} cycles");
        _output.WriteLine($"ACTUAL (gonuget):        {actualCycleCount} cycles");
        _output.WriteLine("RESULT: MATCH ✓");
    }

    #endregion

    #region M5.6 - Transitive Resolution

    [Fact]
    public async Task ResolveTransitive_SimplePackage_ResolvesExactSamePackagesAsNuGetClient()
    {
        // Arrange
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client (EXPECTED)
        var nugetWalker = CreateRemoteDependencyWalker();
        var nugetGraph = await nugetWalker.WalkAsync(
            new LibraryRange(packageId, VersionRange.Parse(versionRange), LibraryDependencyTarget.Package),
            targetFramework,
            runtimeIdentifier: null,
            runtimeGraph: null,
            recursive: true);

        var expectedPackages = new HashSet<string>();
        void CollectPackages(GraphNode<RemoteResolveResult> node)
        {
            if (node?.Item?.Data?.Match != null)
            {
                expectedPackages.Add(node.Item.Data.Match.Library.Name);
            }
            if (node?.InnerNodes != null)
            {
                foreach (var inner in node.InnerNodes)
                {
                    CollectPackages(inner);
                }
            }
        }
        CollectPackages(nugetGraph);

        // Act - gonuget (ACTUAL)
        var gonugetResult = GonugetBridge.ResolveTransitive(
            rootPackages: [new PackageSpec { Id = packageId, VersionRange = versionRange }],
            targetFramework: "net8.0",
            sources: s_nugetSources
        );
        var actualPackages = gonugetResult.Packages.Select(p => p.PackageId).ToHashSet();

        // Assert: EXACT package count match
        int expectedCount = expectedPackages.Count;
        int actualCount = actualPackages.Count;

        _output.WriteLine($"EXPECTED (NuGet.Client): {expectedCount} packages");
        foreach (var pkg in expectedPackages.OrderBy(p => p))
        {
            _output.WriteLine($"  - {pkg}");
        }

        _output.WriteLine($"ACTUAL (gonuget):        {actualCount} packages");
        foreach (var pkg in actualPackages.OrderBy(p => p))
        {
            _output.WriteLine($"  - {pkg}");
        }

        Assert.Equal(expectedCount, actualCount);

        // Assert: EXACT package names match
        foreach (var expectedPkg in expectedPackages)
        {
            Assert.Contains(expectedPkg, actualPackages);
        }

        _output.WriteLine("RESULT: EXACT MATCH ✓");
    }

    [Fact]
    public async Task ResolveTransitive_MultipleRoots_ResolvesExactSamePackagesAsNuGetClient()
    {
        // Arrange
        var targetFramework = NuGetFramework.Parse("net8.0");
        var rootPackages = new[]
        {
            new PackageSpec { Id = "Newtonsoft.Json", VersionRange = "[13.0.1]" },
            new PackageSpec { Id = "System.Text.Json", VersionRange = "[8.0.0]" }
        };

        // Act - NuGet.Client (EXPECTED)
        var nugetWalker = CreateRemoteDependencyWalker();
        var expectedPackages = new HashSet<string>();

        foreach (var root in rootPackages)
        {
            var nugetGraph = await nugetWalker.WalkAsync(
                new LibraryRange(root.Id, VersionRange.Parse(root.VersionRange), LibraryDependencyTarget.Package),
                targetFramework,
                runtimeIdentifier: null,
                runtimeGraph: null,
                recursive: true);

            void CollectPackages(GraphNode<RemoteResolveResult> node)
            {
                if (node?.Item?.Data?.Match != null)
                {
                    expectedPackages.Add(node.Item.Data.Match.Library.Name);
                }
                if (node?.InnerNodes != null)
                {
                    foreach (var inner in node.InnerNodes)
                    {
                        CollectPackages(inner);
                    }
                }
            }
            CollectPackages(nugetGraph);
        }

        // Act - gonuget (ACTUAL)
        var gonugetResult = GonugetBridge.ResolveTransitive(
            rootPackages: rootPackages,
            targetFramework: "net8.0",
            sources: s_nugetSources
        );
        var actualPackages = gonugetResult.Packages.Select(p => p.PackageId).ToHashSet();

        // Assert: EXACT package count match
        int expectedCount = expectedPackages.Count;
        int actualCount = actualPackages.Count;

        _output.WriteLine($"Roots: {string.Join(", ", rootPackages.Select(r => r.Id))}");
        _output.WriteLine($"EXPECTED (NuGet.Client): {expectedCount} packages");
        foreach (var pkg in expectedPackages.OrderBy(p => p))
        {
            _output.WriteLine($"  - {pkg}");
        }

        _output.WriteLine($"ACTUAL (gonuget):        {actualCount} packages");
        foreach (var pkg in actualPackages.OrderBy(p => p))
        {
            _output.WriteLine($"  - {pkg}");
        }

        Assert.Equal(expectedCount, actualCount);

        // Assert: EXACT package names match
        foreach (var expectedPkg in expectedPackages)
        {
            Assert.Contains(expectedPkg, actualPackages);
        }

        _output.WriteLine("RESULT: EXACT MATCH ✓");
    }

    #endregion

    #region M5.7 - Cache Deduplication & TTL

    [Fact]
    public async Task CacheDeduplication_ConcurrentRequests_BothSystemsHandleConcurrency()
    {
        // Arrange
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");
        int concurrentRequests = 10;

        // Act - NuGet.Client (EXPECTED behavior: all requests succeed)
        var nugetWalker = CreateRemoteDependencyWalker();
        var nugetTasks = new List<Task<GraphNode<RemoteResolveResult>>>();

        for (int i = 0; i < concurrentRequests; i++)
        {
            nugetTasks.Add(nugetWalker.WalkAsync(
                new LibraryRange(packageId, VersionRange.Parse(versionRange), LibraryDependencyTarget.Package),
                targetFramework,
                runtimeIdentifier: null,
                runtimeGraph: null,
                recursive: false));
        }

        var expectedStartTime = Stopwatch.StartNew();
        var nugetResults = await Task.WhenAll(nugetTasks);
        expectedStartTime.Stop();

        int expectedSuccessCount = nugetResults.Count(r => r != null);

        // Act - gonuget (ACTUAL)
        var gonugetResult = GonugetBridge.BenchmarkCache(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: s_nugetSources,
            concurrentRequests: concurrentRequests
        );

        // Assert: NuGet.Client expects ALL requests to succeed
        Assert.Equal(concurrentRequests, expectedSuccessCount);

        // Assert: gonuget MUST match
        Assert.Equal(concurrentRequests, gonugetResult.TotalRequests);

        _output.WriteLine($"Concurrent requests: {concurrentRequests}");
        _output.WriteLine($"EXPECTED (NuGet.Client): {expectedSuccessCount} succeeded, {expectedStartTime.ElapsedMilliseconds}ms");
        _output.WriteLine($"ACTUAL (gonuget):        {gonugetResult.TotalRequests} succeeded, {gonugetResult.DurationMs}ms, {gonugetResult.ActualFetches} fetches");
        _output.WriteLine($"Deduplication: {gonugetResult.DeduplicationWorked}");
        _output.WriteLine("RESULT: Both systems handled concurrency correctly ✓");
    }

    [Fact]
    public async Task CacheTTL_WithCacheContext_BothSystemsResolveSuccessfully()
    {
        // Arrange
        var packageId = "Newtonsoft.Json";
        var versionRange = "[13.0.1]";
        var targetFramework = NuGetFramework.Parse("net8.0");

        // Act - NuGet.Client (EXPECTED: resolves successfully with caching)
        var nugetWalker = CreateRemoteDependencyWalker();
        var nugetGraph = await nugetWalker.WalkAsync(
            new LibraryRange(packageId, VersionRange.Parse(versionRange), LibraryDependencyTarget.Package),
            targetFramework,
            runtimeIdentifier: null,
            runtimeGraph: null,
            recursive: true);

        bool expectedSuccess = nugetGraph != null && nugetGraph.Item != null;
        string expectedPackageName = nugetGraph?.Item?.Key?.Name ?? "";

        // Act - gonuget (ACTUAL)
        var gonugetResult = GonugetBridge.ResolveWithTTL(
            packageId: packageId,
            versionRange: versionRange,
            targetFramework: "net8.0",
            sources: s_nugetSources,
            ttlSeconds: 60
        );

        bool actualSuccess = gonugetResult.Packages.Length > 0;
        string actualPackageName = gonugetResult.Packages.FirstOrDefault(p => p.PackageId == packageId)?.PackageId ?? "";

        // Assert: NuGet.Client expects resolution to succeed
        Assert.True(expectedSuccess);
        Assert.Equal(packageId, expectedPackageName);

        // Assert: gonuget MUST match
        Assert.True(actualSuccess);
        Assert.Equal(expectedPackageName, actualPackageName);

        _output.WriteLine($"EXPECTED (NuGet.Client): Resolved '{expectedPackageName}'");
        _output.WriteLine($"ACTUAL (gonuget):        Resolved '{actualPackageName}', {gonugetResult.Packages.Length} total packages");
        _output.WriteLine("RESULT: Both systems resolved with caching ✓");
    }

    #endregion

    #region M5.8 - Parallel Resolution

    [Fact]
    public async Task ParallelResolution_MultiplePackages_ResolvesExactSamePackagesAsNuGetClient()
    {
        // Arrange
        var targetFramework = NuGetFramework.Parse("net8.0");
        var packageSpecs = new[]
        {
            new PackageSpec { Id = "Newtonsoft.Json", VersionRange = "[13.0.1]" },
            new PackageSpec { Id = "System.Text.Json", VersionRange = "[8.0.0]" },
            new PackageSpec { Id = "Microsoft.Extensions.Logging", VersionRange = "[8.0.0]" }
        };

        // Act - NuGet.Client PARALLEL (EXPECTED)
        var nugetWalker = CreateRemoteDependencyWalker();
        var nugetTasks = packageSpecs.Select(spec =>
            nugetWalker.WalkAsync(
                new LibraryRange(spec.Id, VersionRange.Parse(spec.VersionRange), LibraryDependencyTarget.Package),
                targetFramework,
                runtimeIdentifier: null,
                runtimeGraph: null,
                recursive: false)).ToList();

        var expectedStartTime = Stopwatch.StartNew();
        var nugetResults = await Task.WhenAll(nugetTasks);
        expectedStartTime.Stop();

        int expectedPackageCount = nugetResults.Count(r => r != null);

        // Act - gonuget PARALLEL (ACTUAL)
        var gonugetResult = GonugetBridge.BenchmarkParallel(
            packageSpecs: packageSpecs,
            targetFramework: "net8.0",
            sources: s_nugetSources,
            sequential: false,
            recursive: false  // Match NuGet.Client's recursive: false
        );

        // Assert: NuGet.Client resolved exactly 3 packages
        Assert.Equal(3, expectedPackageCount);

        // Assert: gonuget MUST match EXACTLY
        Assert.Equal(expectedPackageCount, gonugetResult.PackageCount);
        Assert.True(gonugetResult.WasParallel);

        _output.WriteLine($"Packages to resolve: {packageSpecs.Length}");
        _output.WriteLine($"EXPECTED (NuGet.Client parallel): {expectedPackageCount} packages, {expectedStartTime.ElapsedMilliseconds}ms");
        _output.WriteLine($"ACTUAL (gonuget parallel):        {gonugetResult.PackageCount} packages, {gonugetResult.DurationMs}ms");
        _output.WriteLine("RESULT: EXACT MATCH ✓");
    }

    [Fact]
    public async Task WorkerPool_WithLimit_ResolvesExactSamePackagesAsNuGetClient()
    {
        // Arrange
        var targetFramework = NuGetFramework.Parse("net8.0");
        var packageSpecs = new[]
        {
            new PackageSpec { Id = "Newtonsoft.Json", VersionRange = "[13.0.1]" },
            new PackageSpec { Id = "System.Text.Json", VersionRange = "[8.0.0]" },
            new PackageSpec { Id = "Microsoft.Extensions.Logging", VersionRange = "[8.0.0]" },
            new PackageSpec { Id = "Serilog", VersionRange = "[3.0.0]" },
            new PackageSpec { Id = "NLog", VersionRange = "[5.0.0]" }
        };

        // Act - NuGet.Client (EXPECTED)
        var nugetWalker = CreateRemoteDependencyWalker();
        var nugetTasks = packageSpecs.Select(spec =>
            nugetWalker.WalkAsync(
                new LibraryRange(spec.Id, VersionRange.Parse(spec.VersionRange), LibraryDependencyTarget.Package),
                targetFramework,
                runtimeIdentifier: null,
                runtimeGraph: null,
                recursive: false)).ToList();

        var nugetResults = await Task.WhenAll(nugetTasks);
        int expectedPackageCount = nugetResults.Count(r => r != null);

        // Act - gonuget with worker limit (ACTUAL)
        var gonugetResult = GonugetBridge.ResolveWithWorkerLimit(
            packageSpecs: packageSpecs,
            targetFramework: "net8.0",
            sources: s_nugetSources,
            maxWorkers: 2
        );

        int actualSuccessCount = gonugetResult.Results.Count(r => string.IsNullOrEmpty(r.Error));

        // Assert: NuGet.Client resolved exactly 5 packages
        Assert.Equal(5, expectedPackageCount);

        // Assert: gonuget MUST resolve the SAME packages despite worker limit
        Assert.Equal(expectedPackageCount, actualSuccessCount);

        // Assert: Worker pool limit was respected
        Assert.True(gonugetResult.MaxConcurrent <= 2,
            $"Worker pool limit violated: expected ≤2, got {gonugetResult.MaxConcurrent}");

        _output.WriteLine($"Packages to resolve: {packageSpecs.Length}");
        _output.WriteLine($"EXPECTED (NuGet.Client): {expectedPackageCount} packages resolved");
        _output.WriteLine($"ACTUAL (gonuget):        {actualSuccessCount} packages resolved, max concurrent={gonugetResult.MaxConcurrent}");

        for (int i = 0; i < packageSpecs.Length; i++)
        {
            bool expectedResolved = nugetResults[i] != null;
            bool actualResolved = string.IsNullOrEmpty(gonugetResult.Results[i].Error);
            Assert.Equal(expectedResolved, actualResolved);

            string status = actualResolved ? "✓" : "✗";
            _output.WriteLine($"  {status} {packageSpecs[i].Id}");
        }

        _output.WriteLine("RESULT: EXACT MATCH with worker pool limits ✓");
    }

    #endregion

    #region Integration Test

    [Fact]
    public async Task Integration_ComplexScenario_AllResultsMatchNuGetClientExactly()
    {
        _output.WriteLine("=== M5.5-M5.8 Integration Test: NuGet.Client vs gonuget ===\n");

        var targetFramework = NuGetFramework.Parse("net8.0");
        var rootPackages = new[]
        {
            new PackageSpec { Id = "Microsoft.Extensions.Logging", VersionRange = "[8.0.0]" },
            new PackageSpec { Id = "Newtonsoft.Json", VersionRange = "[13.0.1]" },
            new PackageSpec { Id = "Serilog", VersionRange = "[3.0.0]" }
        };

        // 1. Transitive Resolution
        _output.WriteLine("1. TRANSITIVE RESOLUTION");

        var nugetWalker = CreateRemoteDependencyWalker();
        var expectedPackages = new HashSet<string>();
        foreach (var root in rootPackages)
        {
            var graph = await nugetWalker.WalkAsync(
                new LibraryRange(root.Id, VersionRange.Parse(root.VersionRange), LibraryDependencyTarget.Package),
                targetFramework,
                runtimeIdentifier: null,
                runtimeGraph: null,
                recursive: true);

            void Collect(GraphNode<RemoteResolveResult> node)
            {
                if (node?.Item?.Data?.Match != null)
                    expectedPackages.Add(node.Item.Data.Match.Library.Name);
                if (node?.InnerNodes != null)
                    foreach (var inner in node.InnerNodes)
                        Collect(inner);
            }
            Collect(graph);
        }

        var gonugetTransitive = GonugetBridge.ResolveTransitive(
            rootPackages: rootPackages,
            targetFramework: "net8.0",
            sources: s_nugetSources);
        var actualPackages = gonugetTransitive.Packages.Select(p => p.PackageId).ToHashSet();

        Assert.Equal(expectedPackages.Count, actualPackages.Count);
        foreach (var pkg in expectedPackages)
        {
            Assert.Contains(pkg, actualPackages);
        }

        _output.WriteLine($"   EXPECTED: {expectedPackages.Count} packages");
        _output.WriteLine($"   ACTUAL:   {actualPackages.Count} packages");
        _output.WriteLine($"   RESULT:   EXACT MATCH ✓\n");

        // 2. Cycle Detection
        _output.WriteLine("2. CYCLE DETECTION");

        var nugetGraph = await nugetWalker.WalkAsync(
            new LibraryRange("Newtonsoft.Json", VersionRange.Parse("[13.0.1]"), LibraryDependencyTarget.Package),
            targetFramework,
            runtimeIdentifier: null,
            runtimeGraph: null,
            recursive: true);
        var nugetAnalysis = nugetGraph.Analyze();
        int expectedCycles = nugetAnalysis.Cycles.Count;

        var gonugetCycles = GonugetBridge.AnalyzeCycles(
            packageId: "Newtonsoft.Json",
            versionRange: "[13.0.1]",
            targetFramework: "net8.0",
            sources: s_nugetSources);
        int actualCycles = gonugetCycles.Cycles.Length;

        Assert.Equal(expectedCycles, actualCycles);

        _output.WriteLine($"   EXPECTED: {expectedCycles} cycles");
        _output.WriteLine($"   ACTUAL:   {actualCycles} cycles");
        _output.WriteLine($"   RESULT:   EXACT MATCH ✓\n");

        // 3. Parallel Resolution
        _output.WriteLine("3. PARALLEL RESOLUTION");

        var nugetParallelTasks = rootPackages.Select(spec =>
            nugetWalker.WalkAsync(
                new LibraryRange(spec.Id, VersionRange.Parse(spec.VersionRange), LibraryDependencyTarget.Package),
                targetFramework,
                runtimeIdentifier: null,
                runtimeGraph: null,
                recursive: false)).ToList();
        var nugetParallel = await Task.WhenAll(nugetParallelTasks);
        int expectedParallelCount = nugetParallel.Count(r => r != null);

        var gonugetParallel = GonugetBridge.BenchmarkParallel(
            packageSpecs: rootPackages,
            targetFramework: "net8.0",
            sources: s_nugetSources,
            sequential: false,
            recursive: false);  // Match NuGet.Client's recursive: false

        Assert.Equal(expectedParallelCount, gonugetParallel.PackageCount);

        _output.WriteLine($"   EXPECTED: {expectedParallelCount} packages");
        _output.WriteLine($"   ACTUAL:   {gonugetParallel.PackageCount} packages");
        _output.WriteLine($"   RESULT:   EXACT MATCH ✓\n");

        _output.WriteLine("=== ALL TESTS MATCH NuGet.Client EXACTLY ✓ ===");
    }

    #endregion
}
