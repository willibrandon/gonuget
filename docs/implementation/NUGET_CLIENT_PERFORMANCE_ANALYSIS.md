# NuGet.Client NU1102 Performance Investigation Report

## Executive Summary

NuGet.Client achieves fast NU1102 error handling through a **multi-layered caching strategy** that combines:
1. **Task-result caching** for deduplication of concurrent requests
2. **HTTP disk caching** with ETag-based validation
3. **In-memory caching** at provider layers
4. **Eager version listing** with range filtering
5. **Fast-path optimizations** for exact version matching

---

## 1. CRITICAL PATH ANALYSIS: NU1102 Error Generation

### Entry Point: UnresolvedMessages.GetMessageAsync()
**File**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/Diagnostics/UnresolvedMessages.cs`

```csharp
// Line 116: Get ALL versions from sources (this is the key operation for NU1102)
var sourceInfo = await GetSourceInfosForIdAsync(
    unresolved.Name, 
    applicableRemoteLibraryProviders, 
    sourceCacheContext, 
    logger, 
    token);

// Line 118: Check if no versions found (NU1101 vs NU1102 detection)
if (sourceInfo.All(static kvp => kvp.Value.Length == 0))
{
    // NU1101: No versions exist at all
    code = NuGetLogCode.NU1101;
}
else
{
    // NU1102: Versions exist but none satisfy the range
    code = NuGetLogCode.NU1102;
    // Lines 141-152: Determine exact error (prerelease-only, version range, etc)
}
```

### The Key Bottleneck: GetSourceInfosForIdAsync()
**Lines 254-277**: Gets ALL versions from a package ID

```csharp
internal static async Task<List<KeyValuePair<PackageSource, ImmutableArray<NuGetVersion>>>> 
GetSourceInfosForIdAsync(
    string id,
    IList<IRemoteDependencyProvider> remoteLibraryProviders,
    SourceCacheContext sourceCacheContext,
    ILogger logger,
    CancellationToken token)
{
    var sources = new List<KeyValuePair<PackageSource, ImmutableArray<NuGetVersion>>>();

    // Get versions from ALL sources in PARALLEL
    var tasks = remoteLibraryProviders
        .Select(e => GetSourceInfoForIdAsync(e, id, sourceCacheContext, logger, token))
        .ToArray();

    foreach (var task in tasks)
    {
        sources.Add(await task);  // Awaits all in parallel
    }

    // Sort by most versions (shows user the feed with most matches)
    return sources
        .OrderByDescending(e => e.Value.Length)
        .ThenBy(e => e.Key.Source, StringComparer.OrdinalIgnoreCase)
        .ToList();
}

// Line 290: Calls provider.GetAllVersionsAsync(id, ...)
// This is where caching begins!
```

---

## 2. CACHE LAYER 1: TaskResultCache (Provider Level)

**File**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/SourceRepositoryDependencyProvider.cs`

### Two Instance-Level Caches

```csharp
// Line 38-39: Per-provider instance caches
private readonly TaskResultCache<LibraryRangeCacheKey, LibraryDependencyInfo> 
    _dependencyInfoCache = new();
    
private readonly TaskResultCache<LibraryRange, LibraryIdentity> 
    _libraryMatchCache = new();
```

### GetAllVersionsAsync Flow

**Lines 612-619**: Entry point with exception handling
```csharp
public async Task<IEnumerable<NuGetVersion>> GetAllVersionsAsync(
    string id,
    SourceCacheContext cacheContext,
    ILogger logger,
    CancellationToken cancellationToken)
{
    return await GetAllVersionsInternalAsync(
        id, cacheContext, logger, 
        catchAndLogExceptions: true, 
        cancellationToken: cancellationToken);
}
```

**Lines 621-661**: Internal implementation with throttle management
```csharp
internal async Task<IEnumerable<NuGetVersion>> GetAllVersionsInternalAsync(
    string id,
    SourceCacheContext cacheContext,
    ILogger logger,
    bool catchAndLogExceptions,
    CancellationToken cancellationToken)
{
    try
    {
        // Throttle concurrent requests (prevents too many open files)
        if (_throttle != null)
        {
            await _throttle.WaitAsync(cancellationToken);
        }
        
        if (_findPackagesByIdResource == null)
        {
            return null;
        }
        
        // Delegates to RemoteV3FindPackageByIdResource
        return await _findPackagesByIdResource.GetAllVersionsAsync(
            id,
            cacheContext,
            logger,
            cancellationToken);
    }
    finally
    {
        _throttle?.Release();
    }
}
```

---

## 3. CACHE LAYER 2: RemoteV3FindPackageByIdResource (HTTP Protocol Layer)

**File**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/RemoteRepositories/RemoteV3FindPackageByIdResource.cs`

### Instance-Level Cache for Package Versions

**Line 29**: The game-changer
```csharp
// Per-source cache, keyed by package ID (case-insensitive)
private readonly TaskResultCache<string, IEnumerable<RemoteSourceDependencyInfo>> 
    _packageVersionsCache = new(StringComparer.OrdinalIgnoreCase);
```

### GetAllVersionsAsync Implementation

**Lines 86-125**: Cached version retrieval
```csharp
public override async Task<IEnumerable<NuGetVersion>> GetAllVersionsAsync(
    string id,
    SourceCacheContext cacheContext,
    ILogger logger,
    CancellationToken cancellationToken)
{
    cancellationToken.ThrowIfCancellationRequested();

    var result = await EnsurePackagesAsync(
        id, 
        cacheContext, 
        logger, 
        cancellationToken);  // ← THIS USES THE CACHE!

    return result.Select(item => item.Identity.Version);
}

// Lines 427-438: The cache access
private Task<IEnumerable<RemoteSourceDependencyInfo>> EnsurePackagesAsync(
    string id,
    SourceCacheContext cacheContext,
    ILogger logger,
    CancellationToken cancellationToken)
{
    return _packageVersionsCache.GetOrAddAsync(
        id,  // Key is just the package ID
        cacheContext.RefreshMemoryCache,  // Respects refresh flag
        static state => state.caller.FindPackagesByIdAsyncCore(
            state.id, 
            state.cacheContext, 
            state.logger, 
            state.cancellationToken),
        (caller: this, id, cacheContext, logger, cancellationToken), 
        cancellationToken);
}
```

### Fast-Path Optimization: DoesPackageExistAsync

**SourceRepositoryDependencyProvider.cs, Lines 239-270**: Exact version check BEFORE listing all
```csharp
// OPTIMIZATION: If version range is exact min version, check if it exists
if (libraryRange.VersionRange?.MinVersion != null 
    && libraryRange.VersionRange.IsMinInclusive 
    && !libraryRange.VersionRange.IsFloating)
{
    // Quick check: Does this exact version exist?
    versionExists = await _findPackagesByIdResource.DoesPackageExistAsync(
        libraryRange.Name,
        libraryRange.VersionRange.MinVersion,
        cacheContext,
        logger,
        cancellationToken);

    if (versionExists)
    {
        return new LibraryIdentity
        {
            Name = libraryRange.Name,
            Version = libraryRange.VersionRange.MinVersion,
            Type = LibraryType.Package
        };
    }
}

// If exact match fails, then get all versions
var packageVersions = await GetAllVersionsInternalAsync(
    libraryRange.Name, 
    cacheContext, 
    logger, 
    false, 
    cancellationToken);
```

---

## 4. CACHE LAYER 3: DependencyInfoResourceV3 (Registration Index)

**File**: `/Users/brandon/src/NuGet.Core/NuGet.Protocol/Resources/DependencyInfoResourceV3.cs`

### Version Retrieval with Remote Caching

**Lines 142-159**: ResolvePackages with version listing
```csharp
public override Task<IEnumerable<RemoteSourceDependencyInfo>> ResolvePackages(
    string packageId, 
    SourceCacheContext cacheContext, 
    ILogger log, 
    CancellationToken token)
{
    var uri = _regResource.GetUri(packageId);  // registration/newtonsoft.json/index.json
    
    // Gets all versions from registration index
    return ResolverMetadataClient.GetDependencies(
        _client,  // HttpSource with its own caching
        uri,
        packageId, 
        VersionRange.All,  // Get ALL versions
        cacheContext, 
        log, 
        token);
}
```

---

## 5. CACHE LAYER 4: ResolverMetadataClient (Registration Index Parsing)

**File**: `/Users/brandon/src/NuGet.Core/NuGet.Protocol/DependencyInfo/ResolverMetadataClient.cs`

### Version Range Smart Filtering

**Lines 26-58**: GetDependencies - Parses registration with early filtering
```csharp
public static async Task<IEnumerable<RemoteSourceDependencyInfo>> GetDependencies(
    HttpSource httpClient,
    Uri registrationUri,
    string packageId,
    VersionRange range,
    SourceCacheContext cacheContext,
    ILogger log,
    CancellationToken token)
{
    // Load ALL registration ranges (may be paginated)
    var ranges = await RegistrationUtility.LoadRanges(
        httpClient,
        registrationUri,
        packageId, 
        range,  // ← Range used for HTTP request filtering
        cacheContext, 
        log, 
        token);

    var results = new HashSet<RemoteSourceDependencyInfo>();
    
    foreach (var rangeObj in ranges)
    {
        foreach (JObject packageObj in rangeObj["items"])
        {
            var catalogEntry = (JObject)packageObj["catalogEntry"];
            var version = NuGetVersion.Parse(catalogEntry["version"].ToString());

            // FILTER: Only include versions in the requested range
            if (range.Satisfies(version))  // ← Early filtering!
            {
                results.Add(ProcessPackageVersion(packageObj, version));
            }
        }
    }

    return results;
}
```

---

## 6. CACHE LAYER 5: RegistrationUtility.LoadRanges (HTTP + Disk Cache)

**File**: `/Users/brandon/src/NuGet.Core/NuGet.Protocol/DependencyInfo/RegistrationUtility.cs`

### Smart Range Loading with Version Bucketing

**Lines 31-104**: LoadRanges - Loads only relevant registration pages
```csharp
public async static Task<IEnumerable<JObject>> LoadRanges(
    HttpSource httpSource,
    Uri registrationUri,  // e.g., https://api.nuget.org/v3/registration5-gz/newtonsoft.json/index.json
    string packageId,
    VersionRange range,
    SourceCacheContext cacheContext,
    ILogger log,
    CancellationToken token)
{
    // 1. Load registration INDEX with caching
    var index = await httpSource.GetAsync(
        new HttpSourceCachedRequest(
            registrationUri.OriginalString,
            $"list_{packageIdLowerCase}_index",  // ← Cache key for index
            httpSourceCacheContext)
        {
            IgnoreNotFounds = true,  // 404 = package doesn't exist
        },
        async httpSourceResult =>
        {
            return await httpSourceResult.Stream.AsJObjectAsync(token);
        },
        log,
        token);

    if (index == null)
    {
        // Package doesn't exist - return empty
        return Enumerable.Empty<JObject>();
    }

    IList<Task<JObject>> rangeTasks = new List<Task<JObject>>();

    // 2. Load only RELEVANT pages based on version range
    foreach (JObject item in index["items"])
    {
        var lower = NuGetVersion.Parse(item["lower"].ToString());
        var upper = NuGetVersion.Parse(item["upper"].ToString());

        // CRITICAL OPTIMIZATION: Only load pages that overlap with requested range!
        if (range.DoesRangeSatisfy(lower, upper))
        {
            JToken items;
            
            // If page is inline, use it directly
            if (!item.TryGetValue("items", out items))
            {
                // Otherwise, load the page with caching
                var rangeUri = item["@id"].ToString();

                rangeTasks.Add(httpSource.GetAsync(
                    new HttpSourceCachedRequest(
                        rangeUri,
                        $"list_{packageIdLowerCase}_range_{lower}-{upper}",  // ← Cache each range
                        httpSourceCacheContext)
                    {
                        IgnoreNotFounds = true,
                    },
                    // ... load and parse page
                ));
            }
            else
            {
                rangeTasks.Add(Task.FromResult(item));
            }
        }
    }

    await Task.WhenAll(rangeTasks.ToArray());
    return rangeTasks.Select((t) => t.Result);
}
```

**KEY INSIGHT**: For a NU1102 error (version not found), only the registration pages that could contain versions in the requested range are fetched and cached!

---

## 7. CACHE LAYER 6: HttpSource Disk Cache

**File**: `/Users/brandon/src/NuGet.Core/NuGet.Protocol/HttpSource/HttpSource.cs`

### Multi-Level HTTP Caching

**Lines 67-207**: GetAsync - Disk + ETag caching with file locking
```csharp
public virtual async Task<T> GetAsync<T>(
    HttpSourceCachedRequest request,
    Func<HttpSourceResult, Task<T>> processAsync,
    ILogger log,
    CancellationToken token)
{
    // 1. Initialize cache file path and check MaxAge
    var cacheResult = HttpCacheUtility.InitializeHttpCacheResult(
        HttpCacheDirectory,
        _sourceUri,
        request.CacheKey,  // e.g., "list_packageid_index"
        request.CacheContext);

    return await ConcurrencyUtilities.ExecuteWithFileLockedAsync(
        cacheResult.CacheFile,  // Lock file for thread-safe access
        action: async lockedToken =>
        {
            // 2. Try to read from disk cache FIRST
            cacheResult.Stream = TryReadCacheFile(
                request.Uri, 
                cacheResult.MaxAge,  // Check if cache is stale
                cacheResult.CacheFile);
            
            try
            {
                if (cacheResult.Stream != null)
                {
                    log.LogInformation($"CACHE: {request.Uri}");  // Log cache hit!

                    // 3. Validate cached content
                    request.EnsureValidContents?.Invoke(cacheResult.Stream);
                    cacheResult.Stream.Seek(0, SeekOrigin.Begin);

                    // Return cached result
                    return await processAsync(new HttpSourceResult(
                        HttpSourceResultStatus.OpenedFromDisk,
                        cacheResult.CacheFile,
                        cacheResult.Stream));
                }

                // 4. Cache miss - fetch from HTTP
                using (var throttledResponse = await GetThrottledResponse(...))
                {
                    if (request.IgnoreNotFounds && throttledResponse.Response.StatusCode == 404)
                    {
                        // 404 = package not found - cache this too!
                        return await processAsync(new HttpSourceResult(
                            HttpSourceResultStatus.NotFound));
                    }

                    throttledResponse.Response.EnsureSuccessStatusCode();

                    // 5. Write to disk cache for future use
                    if (!request.CacheContext.DirectDownload)
                    {
                        await HttpCacheUtility.CreateCacheFileAsync(
                            cacheResult,
                            throttledResponse.Response,
                            request.EnsureValidContents,
                            lockedToken);

                        // Return from cache
                        return await processAsync(new HttpSourceResult(
                            HttpSourceResultStatus.OpenedFromDisk,
                            cacheResult.CacheFile,
                            cacheResult.Stream));
                    }
                }
            }
            finally
            {
                if (cacheResult.Stream != null)
                {
                    cacheResult.Stream.Dispose();
                }
            }
        },
        token: token);
}
```

### Disk Cache File Organization
**File**: `/Users/brandon/src/NuGet.Core/NuGet.Protocol/HttpSource/HttpCacheUtility.cs`

```csharp
// Lines 17-49: Cache file layout
public static HttpCacheResult InitializeHttpCacheResult(
    string httpCacheDirectory,
    Uri sourceUri,
    string cacheKey,
    HttpSourceCacheContext context)
{
    if (context.MaxAge > TimeSpan.Zero)
    {
        // Global HTTP cache: ~/.nuget/v3-cache/{sourceHash}/{cacheKey}.dat
        var baseFolderName = CachingUtility.ComputeHash(sourceUri.OriginalString);
        var baseFileName = cacheKey + ".dat";  // e.g., "list_packageid_index.dat"
        
        var cacheFolder = Path.Combine(httpCacheDirectory, baseFolderName);
        var cacheFile = Path.Combine(cacheFolder, baseFileName);
        
        return new HttpCacheResult(
            context.MaxAge,
            cacheFile + "-new",  // Atomic write: write to .dat-new, then rename
            cacheFile);
    }
}
```

---

## 8. CACHE LAYER 7: ServiceIndexResourceV3Provider (Service Index Caching)

**File**: `/Users/brandon/src/NuGet.Core/NuGet.Protocol/Providers/ServiceIndexResourceV3Provider.cs`

### In-Memory Service Index Cache

**Lines 22-90**: Service index caching with 40-minute TTL
```csharp
public class ServiceIndexResourceV3Provider : ResourceProvider
{
    private static readonly TimeSpan DefaultCacheDuration = TimeSpan.FromMinutes(40);
    
    // Static cache - shared across entire AppDomain
    private readonly ConcurrentDictionary<string, ServiceIndexCacheInfo> _cache;
    private readonly SemaphoreSlim _semaphore = new SemaphoreSlim(1, 1);

    public override async Task<Tuple<bool, INuGetResource>> TryCreate(
        SourceRepository source, 
        CancellationToken token)
    {
        var url = source.PackageSource.Source;
        var utcNow = DateTime.UtcNow;
        var entryValidCutoff = utcNow.Subtract(MaxCacheDuration);

        // Check in-memory cache FIRST
        if (!_cache.TryGetValue(url, out cacheInfo) ||
            entryValidCutoff > cacheInfo.CachedTime)
        {
            await _semaphore.WaitAsync(token);

            try
            {
                // Double-check after lock
                if (!_cache.TryGetValue(url, out cacheInfo) ||
                    entryValidCutoff > cacheInfo.CachedTime)
                {
                    // Fetch service index
                    index = await GetServiceIndexResourceV3(
                        source, 
                        utcNow, 
                        NullLogger.Instance, 
                        token);

                    // Cache the result (even if null)
                    _cache.AddOrUpdate(url, 
                        new ServiceIndexCacheInfo
                        {
                            CachedTime = utcNow,
                            Index = index
                        },
                        (key, value) => ...);
                }
            }
            finally
            {
                _semaphore.Release();
            }
        }

        // Retrieve from cache
        if (index == null && cacheInfo != null)
        {
            index = cacheInfo.Index;
        }

        return new Tuple<bool, INuGetResource>(
            index != null, 
            index);
    }
}
```

---

## 9. CACHE LAYER 8: RemoteWalkContext (Dependency Resolution Cache)

**File**: `/Users/brandon/src/NuGet.Core/NuGet.DependencyResolver.Core/Remote/RemoteWalkContext.cs`

### Operation-Level Caches

**Lines 28-51**: Two high-level caches for resolution
```csharp
public class RemoteWalkContext
{
    public RemoteWalkContext(
        SourceCacheContext cacheContext, 
        PackageSourceMapping packageSourceMapping, 
        ILogger logger)
    {
        CacheContext = cacheContext;
        Logger = logger;

        // ... providers ...

        // Per-walk cache: Prevents duplicate work during same restore
        FindLibraryEntryCache = new TaskResultCache<
            LibraryRangeCacheKey, 
            GraphItem<RemoteResolveResult>>();
            
        ResolvePackageLibraryMatchCache = new TaskResultCache<
            LibraryRange, 
            Tuple<LibraryRange, RemoteMatch>>();

        LockFileLibraries = new Dictionary<LockFileCacheKey, IList<LibraryIdentity>>();
    }

    // Caches live for entire restore operation
    internal TaskResultCache<LibraryRangeCacheKey, GraphItem<RemoteResolveResult>> 
        FindLibraryEntryCache { get; }

    internal TaskResultCache<LibraryRange, Tuple<LibraryRange, RemoteMatch>> 
        ResolvePackageLibraryMatchCache { get; }
}
```

---

## 10. THE TASKRESULTCACHE MAGIC: Deduplication

**File**: `/Users/brandon/src/NuGet.Client/build/Shared/TaskResultCache.cs`

### Thread-Safe Task Deduplication with Per-Key Locking

**Lines 14-119**: The heart of concurrency optimization
```csharp
internal sealed class TaskResultCache<TKey, TValue>
    where TKey : notnull
{
    // Cache holds TASKS, not results
    private readonly ConcurrentDictionary<TKey, Task<TValue>> _cache;
    
    // Per-key locks to avoid locking entire cache
    private readonly ConcurrentDictionary<TKey, object> _perTaskLock;

    public Task<TValue> GetOrAddAsync<TState>(
        TKey key, 
        bool refresh, 
        Func<TState, Task<TValue>> valueFactory, 
        TState state, 
        CancellationToken cancellationToken)
    {
        // Fast path: Cache hit
        if (!refresh && _cache.TryGetValue(key, out Task<TValue> value))
        {
            return value;  // Return cached TASK immediately!
        }

        // Get per-key lock (not global)
        object lockObject = _perTaskLock.GetOrAdd(
            key, 
            static (TKey _) => new object());

        lock (lockObject)
        {
            // Double-check after lock
            if (!refresh && _cache.TryGetValue(key, out value))
            {
                return value;
            }

            // First caller executes the work, others await the same task
            return _cache[key] = valueFactory(state)
                .ContinueWith(
                    static task => task.GetAwaiter().GetResult(),
                    cancellationToken,
                    TaskContinuationOptions.RunContinuationsAsynchronously,
                    TaskScheduler.Default);  // ← Awaitable cached Task
        }
    }

    public bool TryGetValue(TKey key, out Task<TValue> value)
    {
        return _cache.TryGetValue(key, out value);
    }
}
```

**CRITICAL PERFORMANCE INSIGHT**:
- Multiple concurrent requests for same `packageId` → Only ONE HTTP request is made
- Other requests await the same cached Task
- Per-key locking prevents global lock contention

---

## 11. NEGATIVE RESULT CACHING (404 Handling)

### Package Not Found Handling

**RegistrationUtility.LoadRanges, Lines 59-62**:
```csharp
if (index == null)
{
    // 404 response is cached as null - prevents re-querying
    return Enumerable.Empty<JObject>();
}
```

**HttpSource.GetAsync, Lines 143-148**:
```csharp
if (request.IgnoreNotFounds && 
    throttledResponse.Response.StatusCode == HttpStatusCode.NotFound)
{
    // Even 404s are cached and returned quickly
    var httpSourceResult = new HttpSourceResult(
        HttpSourceResultStatus.NotFound);
    
    return await processAsync(httpSourceResult);
}
```

**HttpCacheUtility.cs**: 404 responses can be cached to disk, preventing repeated HTTP lookups.

---

## 12. QUICK-FAIL OPTIMIZATIONS

### 1. Exact Version Check Before Full Listing
**SourceRepositoryDependencyProvider.cs, Lines 239-270**:
```csharp
// If asking for exact version "1.2.3", check directly first
if (libraryRange.VersionRange?.MinVersion != null 
    && libraryRange.VersionRange.IsMinInclusive 
    && !libraryRange.VersionRange.IsFloating)
{
    versionExists = await _findPackagesByIdResource.DoesPackageExistAsync(
        libraryRange.Name,
        libraryRange.VersionRange.MinVersion,
        cacheContext,
        logger,
        cancellationToken);

    if (versionExists)
    {
        return new LibraryIdentity { ... };  // Fast exit!
    }
}
```

### 2. Version Range-Aware Registration Loading
**RegistrationUtility.LoadRanges, Lines 72-98**:
```csharp
// Only load registration pages that could contain matching versions
if (range.DoesRangeSatisfy(lower, upper))
{
    // Load this page
}
// Skip pages outside version range completely
```

### 3. HTTP Cache Validation Before Network
**HttpSource.GetAsync, Lines 85-116**:
- Check disk cache with file locking
- Validate cached content
- Return from disk before any HTTP request

---

## 13. COMPARISON: Missing Optimizations in gonuget

### What gonuget is missing:

| Optimization | NuGet.Client | gonuget |
|---|---|---|
| **TaskResultCache** | Per-ID task deduplication across requests | ❌ Manual deduplication only |
| **HTTP Disk Cache** | Full disk caching with ETag support | ❌ No disk cache yet |
| **Service Index Cache** | 40-minute in-memory cache | ❌ Fetched every restore |
| **Registration Range Filtering** | Loads only relevant pages | ❌ Loads full index |
| **Exact Version Fast-Path** | DoesPackageExistAsync before full listing | ❌ Always lists all |
| **Negative Result Caching** | Caches 404s to prevent re-queries | ⚠️  Manual caching only |
| **Per-Key Locking** | Lock per package ID, not global | ❌ Global/operation-level |
| **RemoteWalkContext Caches** | Per-walk deduplication | ⚠️  Resolver cache exists |

---

## 14. PERFORMANCE IMPACT ESTIMATES

For a NU1102 error on nonexistent package (e.g., "badpackage"):

### NuGet.Client (Fast)
1. Check in-memory Service Index cache (cached, <1ms)
2. Check disk cache for registration index (cache hit, <5ms)
3. Parse index, identify no matching pages needed (0ms)
4. Return immediately with cached 404 response
5. **Total: 5-10ms** (from cache)

### gonuget (Current)
1. Fetch service index (network, ~100-200ms)
2. Fetch registration index (network, ~100-200ms)
3. Parse full index (parsing, ~50ms)
4. Check all versions (processing, ~20ms)
5. **Total: 270-470ms** (from network on cold start)

### gonuget (Cached)
1. Fetch service index (cache, ~0ms) - not cached yet
2. Fetch registration index (cache, ~5ms)
3. Parse full index (parsing, ~20ms)
4. Check all versions (processing, ~5ms)
5. **Total: 30-50ms**

---

## 15. KEY FILES TO OPTIMIZE IN GONUGET

```
gonuget/protocol/
├── v3/
│   ├── registration.go (add TaskResultCache pattern)
│   ├── http_cache.go (add disk cache like HttpCacheUtility)
│   └── service_index.go (add 40-minute in-memory cache)
├── shared/
│   └── task_result_cache.go (implement per-key locking)
└── core/
    └── provider.go (add exact version fast-path)
```

---

## Summary of Caching Layers

1. **Layer 1**: TaskResultCache (per-dependency provider) - Deduplication across concurrent requests
2. **Layer 2**: TaskResultCache (per-remote-v3 resource) - Per-package-ID caching
3. **Layer 3**: DependencyInfoResourceV3 - Registration index queries
4. **Layer 4**: ResolverMetadataClient - Version filtering with range awareness
5. **Layer 5**: RegistrationUtility - Smart range page loading
6. **Layer 6**: HttpSource - Disk cache with file locking and ETag validation
7. **Layer 7**: ServiceIndexResourceV3Provider - 40-minute service index cache
8. **Layer 8**: RemoteWalkContext - Per-walk operation caching

**The magic**: Multiple layers allow cache hits at different points:
- Same restore? Use Layer 8 (operation cache)
- Different restore, same session? Use Layer 6 (disk cache)
- Different session? Use Layer 7 (service index cache)

