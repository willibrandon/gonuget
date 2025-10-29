# NuGet.Client Performance Optimizations - File Reference Guide

## All Relevant File Paths and Line Numbers

### 1. CRITICAL PATH: NU1102 Error Generation

**UnresolvedMessages.cs** (Error Logging)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/Diagnostics/UnresolvedMessages.cs`
- **Key Lines**:
  - 116: `GetSourceInfosForIdAsync()` call - fetches ALL versions
  - 118-152: NU1101 vs NU1102 vs NU1103 differentiation logic
  - 254-277: `GetSourceInfosForIdAsync()` implementation - parallel fetch from all sources
  - 290: `GetAllVersionsAsync()` call - entry point to caching system

### 2. LAYER 1: Provider-Level Task Deduplication

**SourceRepositoryDependencyProvider.cs** (Provider Cache)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/SourceRepositoryDependencyProvider.cs`
- **Key Lines**:
  - 38-39: Two instance-level TaskResultCache fields
    - `_dependencyInfoCache`: Caches LibraryDependencyInfo
    - `_libraryMatchCache`: Caches LibraryIdentity matches
  - 176-229: `FindLibraryAsync()` - uses `_libraryMatchCache.GetOrAddAsync()`
  - 207-211: Double-checked locking pattern with per-key cache
  - 239-270: **EXACT VERSION FAST-PATH** - optimization we're missing
    - Checks `DoesPackageExistAsync()` before full version listing
    - Only lists all versions if exact match fails
  - 612-619: `GetAllVersionsAsync()` - delegates to RemoteV3
  - 621-661: `GetAllVersionsInternalAsync()` - throttle management

### 3. LAYER 2: Remote V3 Package-Level Cache

**RemoteV3FindPackageByIdResource.cs** (Per-Source, Per-ID Cache)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/RemoteRepositories/RemoteV3FindPackageByIdResource.cs`
- **Key Lines**:
  - 29: **PRIMARY OPTIMIZATION** - Per-ID TaskResultCache (case-insensitive)
    ```csharp
    private readonly TaskResultCache<string, IEnumerable<RemoteSourceDependencyInfo>> 
        _packageVersionsCache = new(StringComparer.OrdinalIgnoreCase);
    ```
  - 86-125: `GetAllVersionsAsync()` - uses EnsurePackagesAsync()
  - 369-414: `DoesPackageExistAsync()` - exact version check
  - 416-425: `GetPackageInfoAsync()` - looks up specific version
  - 427-438: **CACHE IMPLEMENTATION** - `EnsurePackagesAsync()`
    ```csharp
    private Task<IEnumerable<RemoteSourceDependencyInfo>> EnsurePackagesAsync(...)
    {
        return _packageVersionsCache.GetOrAddAsync(
            id,  // Key: just the package ID
            cacheContext.RefreshMemoryCache,
            static state => state.caller.FindPackagesByIdAsyncCore(...),
            ..., cancellationToken);
    }
    ```
  - 440-452: `FindPackagesByIdAsyncCore()` - calls DependencyInfoResource

### 4. LAYER 3: DependencyInfo Resource (Registration Index)

**DependencyInfoResourceV3.cs** (V3 Protocol Handler)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/Resources/DependencyInfoResourceV3.cs`
- **Key Lines**:
  - 63-92: `ResolvePackage()` - single package with version range filtering
  - 102-133: `ResolvePackages()` - all packages from registration index
  - 142-159: **OVERLOAD FOR REMOTE** - `ResolvePackages()` with RemoteSourceDependencyInfo
    ```csharp
    public override Task<IEnumerable<RemoteSourceDependencyInfo>> ResolvePackages(
        string packageId, 
        SourceCacheContext cacheContext, 
        ILogger log, 
        CancellationToken token)
    {
        var uri = _regResource.GetUri(packageId);
        return ResolverMetadataClient.GetDependencies(
            _client,  // HttpSource - disk cached!
            uri,
            packageId, 
            VersionRange.All,
            cacheContext, 
            log, 
            token);
    }
    ```

### 5. LAYER 4: Metadata Parsing with Range Filtering

**ResolverMetadataClient.cs** (JSON Parsing + Version Filtering)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/DependencyInfo/ResolverMetadataClient.cs`
- **Key Lines**:
  - 26-58: `GetDependencies()` - **EARLY FILTERING**
    - Loads registration ranges
    - Filters versions: `if (range.Satisfies(version))`
    - Returns only matching versions
  - 66-108: `ProcessPackageVersion()` - parses individual version entry
  - 114-169: `GetRegistrationInfo()` - parses with framework filtering

### 6. LAYER 5: Smart Range-Based Loading

**RegistrationUtility.cs** (HTTP Request Optimization)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/DependencyInfo/RegistrationUtility.cs`
- **Key Lines**:
  - 31-104: **CRITICAL OPTIMIZATION** - `LoadRanges()`
    - 40: Case-insensitive ID for cache keys
    - 44-57: **Load registration INDEX** with caching
      ```csharp
      var index = await httpSource.GetAsync(
          new HttpSourceCachedRequest(
              registrationUri.OriginalString,
              $"list_{packageIdLowerCase}_index",  // ← Cache key
              httpSourceCacheContext)
          { IgnoreNotFounds = true },  // ← Cache 404s!
          ...);
      ```
    - 59-62: Return empty if 404 (cached)
    - 72-98: **SMART RANGE FILTERING** - only load relevant pages
      ```csharp
      if (range.DoesRangeSatisfy(lower, upper))
      {
          // Load this page
          rangeTasks.Add(httpSource.GetAsync(
              new HttpSourceCachedRequest(
                  rangeUri,
                  $"list_{packageIdLowerCase}_range_{lower}-{upper}",
                  httpSourceCacheContext)
              ...));
      }
      // Skip pages outside range!
      ```

### 7. LAYER 6: HTTP Disk Cache with File Locking

**HttpSource.cs** (Core HTTP Caching)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/HttpSource/HttpSource.cs`
- **Key Lines**:
  - 67-207: **COMPLETE CACHING IMPLEMENTATION** - `GetAsync<T>()`
    - 75-79: Initialize cache result with path calculation
    - 81: **FILE LOCKING** - `ExecuteWithFileLockedAsync()` for thread-safe access
    - 85-116: **CACHE READ** - Try disk first
      - 85: `TryReadCacheFile()` checks file age
      - 88-104: Cache hit path
      - 90: Log "CACHE" message
      - 95: Content validation
    - 118-139: HTTP request factory setup
    - 141-195: **CACHE WRITE** - On cache miss
      - 143-148: Handle 404 (IgnoreNotFounds)
      - 150-156: Handle 204 No Content
      - 160-174: Write to disk cache
      - Atomic: Write to `.dat-new`, then rename
  - 245-250: Stream processing variants

**HttpCacheUtility.cs** (Cache File Management)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/HttpSource/HttpCacheUtility.cs`
- **Key Lines**:
  - 17-49: **CACHE PATH INITIALIZATION**
    ```csharp
    public static HttpCacheResult InitializeHttpCacheResult(...)
    {
        // ~/.nuget/v3-cache/{sourceHash}/{cacheKey}.dat
        var baseFolderName = CachingUtility.ComputeHash(sourceUri.OriginalString);
        var baseFileName = cacheKey + ".dat";  // "list_packageid_index.dat"
        var cacheFolder = Path.Combine(httpCacheDirectory, baseFolderName);
        var cacheFile = Path.Combine(cacheFolder, baseFileName);
        var newCacheFile = cacheFile + "-new";  // Atomic write
    }
    ```
  - 51-127: `CreateCacheFileAsync()` - **ATOMIC WRITE PATTERN**
    - 68-82: Write to `.dat-new` file
    - 85-89: Content validation
    - 92-101: Delete old file (if not in use)
    - 111-116: Atomic rename to final name
    - 121-126: Open final file for reading

### 8. LAYER 7: Service Index Caching (40-Minute TTL)

**ServiceIndexResourceV3Provider.cs** (Service Index Cache)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Protocol/Providers/ServiceIndexResourceV3Provider.cs`
- **Key Lines**:
  - 24: **STATIC CACHE** - `DefaultCacheDuration = TimeSpan.FromMinutes(40)`
  - 25: **CONCURRENT CACHE** - `ConcurrentDictionary<string, ServiceIndexCacheInfo>`
  - 47-99: `TryCreate()` - **IN-MEMORY CACHING**
    - 58-63: Check cache validity with TTL
    - 65-89: **DOUBLE-CHECKED LOCKING** with semaphore
    - 73: Fetch if not in cache or expired
    - 76-83: Cache result (even if null)
    - 93-96: Return from cache

### 9. LAYER 8: Per-Walk Operation Cache

**RemoteWalkContext.cs** (Resolution-Level Caching)
- **Path**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.DependencyResolver.Core/Remote/RemoteWalkContext.cs`
- **Key Lines**:
  - 28-29: **TWO HIGH-LEVEL CACHES**
    ```csharp
    FindLibraryEntryCache = new TaskResultCache<
        LibraryRangeCacheKey, 
        GraphItem<RemoteResolveResult>>();
    
    ResolvePackageLibraryMatchCache = new TaskResultCache<
        LibraryRange, 
        Tuple<LibraryRange, RemoteMatch>>();
    ```
  - 49-51: Per-walk instance caches (live for entire restore)

### 10. THE CORE: TaskResultCache Implementation

**TaskResultCache.cs** (Per-Key Locking Magic)
- **Path**: `/Users/brandon/src/NuGet.Client/build/Shared/TaskResultCache.cs`
- **Key Lines**:
  - 14-19: **CLASS DEFINITION** - Generic, per-key locking
  - 25: **CACHE DICTIONARY** - Stores Tasks, not results
  - 30: **PER-KEY LOCKS** - ConcurrentDictionary of lock objects
  - 36-59: **CONSTRUCTORS** - Initialize with capacity/comparer
  - 74-119: **GETORАДDASYNC** - Core implementation
    - 76: Fast path: return cached task immediately
    - 90-92: **FAST PATH** - Cache hit, return task
    - 103: **PER-KEY LOCK** - GetOrAdd() pattern for locks
    - 105-118: **DOUBLE-CHECKED LOCKING**
      ```csharp
      lock (lockObject)
      {
          if (!refresh && _cache.TryGetValue(key, out value))
          {
              return value;  // Second check
          }
          
          // First caller does work, returns cached TASK
          return _cache[key] = valueFactory(state)
              .ContinueWith(
                  static task => task.GetAwaiter().GetResult(),
                  cancellationToken,
                  TaskContinuationOptions.RunContinuationsAsynchronously,
                  TaskScheduler.Default);
      }
      ```
    - **KEY INSIGHT**: Returns `Task<TValue>`, not `TValue`
      - Multiple callers await same task
      - Only one HTTP request made
      - Others get same result with minimal overhead

### 11. Negative Result (404) Handling

**Multiple Locations**:
- `RegistrationUtility.cs`, line 50: `IgnoreNotFounds = true`
- `HttpSource.cs`, lines 143-148: Handle 404 response
- `HttpCacheUtility.cs`: Cache 404 responses to disk

---

## Quick Reference: Where Each Optimization Happens

| Optimization | File | Method | Lines |
|---|---|---|---|
| **Entry Point** | UnresolvedMessages.cs | GetMessageAsync | 116-152 |
| **Provider Cache** | SourceRepositoryDependencyProvider.cs | FindLibraryAsync | 207-211 |
| **Per-ID Cache** | RemoteV3FindPackageByIdResource.cs | EnsurePackagesAsync | 427-438 |
| **Fast-Path Check** | SourceRepositoryDependencyProvider.cs | FindLibraryCoreAsync | 239-270 |
| **Registration Index** | DependencyInfoResourceV3.cs | ResolvePackages | 142-159 |
| **Range Filtering** | ResolverMetadataClient.cs | GetDependencies | 26-58 |
| **Smart Loading** | RegistrationUtility.cs | LoadRanges | 72-98 |
| **HTTP Cache Read** | HttpSource.cs | GetAsync | 85-116 |
| **HTTP Cache Write** | HttpCacheUtility.cs | CreateCacheFileAsync | 51-127 |
| **Service Index Cache** | ServiceIndexResourceV3Provider.cs | TryCreate | 47-99 |
| **Task Deduplication** | TaskResultCache.cs | GetOrAddAsync | 74-119 |

---

## Implementation Priority for gonuget

1. **Start with TaskResultCache** (biggest bang for buck)
   - Implement per-key locking
   - Use as base for all caches

2. **Then HTTP Disk Cache**
   - File locking pattern
   - Cache path structure
   - Atomic writes

3. **Then Service Index Cache**
   - 40-minute TTL
   - Static/shared across operations

4. **Consider Range Filtering**
   - DoesRangeSatisfy() checks
   - Skip unnecessary pages

5. **Consider Fast-Path**
   - Exact version check first
   - Only list all if needed

---

## All Files You Need to Reference

```
NuGet.Client Source Tree (for reference):
├── src/NuGet.Core/
│   ├── NuGet.Commands/
│   │   └── RestoreCommand/
│   │       ├── SourceRepositoryDependencyProvider.cs ← KEY
│   │       └── Diagnostics/
│   │           └── UnresolvedMessages.cs ← ENTRY POINT
│   └── NuGet.Protocol/
│       ├── RemoteRepositories/
│       │   └── RemoteV3FindPackageByIdResource.cs ← CACHE #2
│       ├── Resources/
│       │   ├── DependencyInfoResourceV3.cs ← CACHE #3
│       │   ├── FindPackageByIdResource.cs
│       │   └── DependencyInfoResource.cs
│       ├── DependencyInfo/
│       │   ├── ResolverMetadataClient.cs ← FILTERING
│       │   └── RegistrationUtility.cs ← SMART LOADING
│       ├── HttpSource/
│       │   ├── HttpSource.cs ← DISK CACHE
│       │   └── HttpCacheUtility.cs ← CACHE MANAGEMENT
│       └── Providers/
│           └── ServiceIndexResourceV3Provider.cs ← SERVICE INDEX
├── src/NuGet.Core/NuGet.DependencyResolver.Core/
│   └── Remote/
│       └── RemoteWalkContext.cs ← WALK CACHE
└── build/Shared/
    └── TaskResultCache.cs ← CORE PATTERN
```

