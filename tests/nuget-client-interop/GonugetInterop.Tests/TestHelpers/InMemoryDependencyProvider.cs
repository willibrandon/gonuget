using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using NuGet.Common;
using NuGet.Configuration;
using NuGet.DependencyResolver;
using NuGet.Frameworks;
using NuGet.LibraryModel;
using NuGet.Packaging;
using NuGet.Packaging.Core;
using NuGet.Protocol.Core.Types;
using NuGet.Versioning;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// In-memory dependency provider for testing cycle detection and resolution.
/// Matches NuGet.Client's DependencyProvider test harness pattern.
/// </summary>
public class InMemoryDependencyProvider : IRemoteDependencyProvider
{
    private readonly Dictionary<string, List<PackageInfo>> _packages = new();

    public bool IsHttp => false;
    public PackageSource Source => new PackageSource("InMemory");
    public SourceRepository SourceRepository => null!;

    public class PackageInfo
    {
        public required string Id { get; set; } = string.Empty;
        public required NuGetVersion Version { get; set; } = new NuGetVersion("1.0.0");
        public List<PackageDependency> Dependencies { get; set; } = new();
    }

    /// <summary>
    /// Fluent API to add a package.
    /// </summary>
    public PackageBuilder Package(string id, string version)
    {
        return new PackageBuilder(this, id, version);
    }

    public class PackageBuilder
    {
        private readonly InMemoryDependencyProvider _provider;
        private readonly PackageInfo _package;

        public PackageBuilder(InMemoryDependencyProvider provider, string id, string version)
        {
            _provider = provider;
            _package = new PackageInfo
            {
                Id = id,
                Version = NuGetVersion.Parse(version)
            };

            if (!_provider._packages.ContainsKey(id))
            {
                _provider._packages[id] = new List<PackageInfo>();
            }
            _provider._packages[id].Add(_package);
        }

        public PackageBuilder DependsOn(string id, string versionRange)
        {
            _package.Dependencies.Add(new PackageDependency(
                id,
                VersionRange.Parse(versionRange)));
            return this;
        }
    }

    public Task<LibraryIdentity> FindLibraryAsync(
        LibraryRange libraryRange,
        NuGetFramework targetFramework,
        SourceCacheContext cacheContext,
        ILogger logger,
        CancellationToken cancellationToken)
    {
        if (_packages.TryGetValue(libraryRange.Name, out var versions))
        {
            var match = versions.FirstOrDefault(p =>
                libraryRange.VersionRange!.Satisfies(p.Version));

            if (match != null)
            {
                return Task.FromResult(new LibraryIdentity(
                    match.Id,
                    match.Version,
                    LibraryType.Package));
            }
        }

        return Task.FromResult<LibraryIdentity>(null!);
    }

    public Task<LibraryDependencyInfo> GetDependenciesAsync(
        LibraryIdentity library,
        NuGetFramework targetFramework,
        SourceCacheContext cacheContext,
        ILogger logger,
        CancellationToken cancellationToken)
    {
        if (_packages.TryGetValue(library.Name, out var versions))
        {
            var match = versions.FirstOrDefault(p => p.Version == library.Version);
            if (match != null)
            {
                var dependencies = match.Dependencies.Select(d => new LibraryDependency
                {
                    LibraryRange = new LibraryRange(
                        d.Id,
                        d.VersionRange,
                        LibraryDependencyTarget.Package)
                }).ToList();

                return Task.FromResult(new LibraryDependencyInfo(
                    library,
                    resolved: true,
                    framework: targetFramework,
                    dependencies: dependencies));
            }
        }

        return Task.FromResult<LibraryDependencyInfo>(null!);
    }

    public Task CopyToAsync(
        LibraryIdentity identity,
        System.IO.Stream stream,
        SourceCacheContext cacheContext,
        ILogger logger,
        CancellationToken cancellationToken)
    {
        return Task.CompletedTask;
    }

    public Task<IEnumerable<NuGetVersion>> GetAllVersionsAsync(
        string id,
        SourceCacheContext cacheContext,
        ILogger logger,
        CancellationToken cancellationToken)
    {
        if (_packages.TryGetValue(id, out var versions))
        {
            return Task.FromResult(versions.Select(p => p.Version));
        }

        return Task.FromResult(Enumerable.Empty<NuGetVersion>());
    }

    public Task<IPackageDownloader> GetPackageDownloaderAsync(
        PackageIdentity packageIdentity,
        SourceCacheContext cacheContext,
        ILogger logger,
        CancellationToken cancellationToken)
    {
        return Task.FromResult<IPackageDownloader>(null!);
    }

    /// <summary>
    /// Gets all packages for serialization across the CLI bridge.
    /// </summary>
    public IEnumerable<PackageInfo> GetAllPackages()
    {
        return _packages.Values.SelectMany(versions => versions);
    }
}
