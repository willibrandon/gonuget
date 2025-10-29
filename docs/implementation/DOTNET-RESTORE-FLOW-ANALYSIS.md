# dotnet restore Flow Analysis

This document describes the complete flow of how `dotnet restore` works internally, from the dotnet SDK CLI through MSBuild to NuGet.Client.

## 1. DOTNET RESTORE COMMAND (SDK Entry Point)

### Location
- `/Users/brandon/src/sdk/src/Cli/dotnet/Commands/Restore/RestoreCommand.cs`
- `/Users/brandon/src/sdk/src/Cli/dotnet/Commands/Restore/RestoringCommand.cs`

### Flow

1. User runs: `dotnet restore`
2. RestoreCommand.Run() is invoked
3. RestoreCommand.FromArgs() parses arguments using RestoreCommandParser
4. RestoreCommand.FromParseResult() creates a CommandBase:
   - For virtual projects: Creates VirtualProjectBuildingCommand
   - For physical projects: Creates MSBuildForwardingApp via CreateForwarding()

### RestoringCommand (Wrapper for restore+build operations)

- When other commands (build, publish, etc.) call restore, they use RestoringCommand
- RestoringCommand determines if:
  - Inline restore (single MSBuild invocation with -restore flag)
  - Separate restore (two MSBuild invocations - one for restore, one for build)
- Decision based on whether properties exclude from restore (like TargetFramework)
- Optimization: Disables default items during restore to avoid globbing:
  - EnableDefaultCompileItems=false
  - EnableDefaultEmbeddedResourceItems=false
  - EnableDefaultNoneItems=false

### Key Code

```csharp
// RestoringCommand.cs - Execute()
public override int Execute()
{
    int exitCode;
    if (SeparateRestoreCommand != null)
    {
        exitCode = SeparateRestoreCommand.Execute();
        if (exitCode != 0)
            return exitCode;
    }
    exitCode = base.Execute();
    if (AdvertiseWorkloadUpdates)
        WorkloadManifestUpdater.AdvertiseWorkloadUpdates();
    return exitCode;
}
```

## 2. MSBUILD FORWARDING (SDK -> MSBuild)

### Location

- /Users/brandon/src/sdk/src/Cli/dotnet/Commands/MSBuild/MSBuildForwardingApp.cs

### Flow

1. RestoreCommand.CreateForwarding() creates MSBuildForwardingApp instance
2. MSBuildForwardingApp wraps MSBuild arguments with CLI options
3. On Execute():
   - Can run out-of-process (default) or in-process
   - Passes -restore flag to MSBuild
   - Passes restore-specific properties via -rp (RestoreProperty) flags
4. Invokes MSBuild.exe with:
   - Target: "Restore" (if separate restore) or default targets with -restore flag
   - Properties including NuGet settings
   - Telemetry loggers

### Key Code

```csharp
// MSBuildForwardingApp.cs - Execute()
public override int Execute()
{
    if (_forwardingAppWithoutLogging.ExecuteMSBuildOutOfProc)
    {
        ProcessStartInfo startInfo = GetProcessStartInfo();
        exitCode = startInfo.Execute();  // Out-of-process
    }
    else
    {
        exitCode = _forwardingAppWithoutLogging.ExecuteInProc(arguments);  // In-process
    }
    return exitCode;
}
```

## 3. MSBUILD RESTORE TARGETS

### Location

- /Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks/NuGet.targets

### File Structure

NuGet.targets is a 1580-line MSBuild targets file that orchestrates the entire restore process.

### Task Imports

Lines 159-177 import tasks from NuGet.Build.Tasks.dll:
- RestoreTask - Main restore execution task
- WriteRestoreGraphTask - Writes dependency graph to disk
- GetRestoreProjectReferencesTask - Collects project references
- GetRestorePackageReferencesTask - Collects package references
- GetCentralPackageVersionsTask - Central package management
- GetRestorePackageDownloadsTask - Explicit package downloads
- GetRestoreFrameworkReferencesTask - Framework references
- GetRestoreNuGetAuditSuppressionsTask - Vulnerability audit suppressions
- GetRestorePrunePackageReferencesTask - Package pruning
- GetRestoreDotnetCliToolsTask - CLI tool references
- GetProjectTargetFrameworksTask - Target frameworks
- GetRestoreSolutionProjectsTask - Solution project resolution
- GetRestoreSettingsTask - NuGet settings
- And more...

### Main Targets and Flow

#### 1. Restore (Entry Point - Line 185)

```xml
<Target Name="Restore" DependsOnTargets="_GenerateRestoreGraph">
  <RestoreTask
    RestoreGraphItems="@(_RestoreGraphEntryFiltered)"
    RestoreDisableParallel="$(RestoreDisableParallel)"
    RestoreNoCache="$(RestoreNoCache)"
    RestoreNoHttpCache="$(RestoreNoHttpCache)"
    RestoreIgnoreFailedSources="$(RestoreIgnoreFailedSources)"
    RestoreRecursive="$(RestoreRecursive)"
    RestoreForce="$(RestoreForce)"
    HideWarningsAndErrors="$(HideWarningsAndErrors)"
    Interactive="$(NuGetInteractive)"
    RestoreForceEvaluate="$(RestoreForceEvaluate)"
    RestorePackagesConfig="$(RestorePackagesConfig)"
    EmbedFilesInBinlog="$(RestoreEmbedFilesInBinlog)">
  </RestoreTask>
</Target>
```

#### 2. _GenerateRestoreGraph (Line 556)

- Depends on: _FilterRestoreGraphProjectInputItems, _GetAllRestoreProjectPathItems
- Builds the complete dependency graph by:
  - Loading entry point projects
  - Filtering for supported project types
  - Walking project references recursively
  - Collecting package references for each project
  - Evaluating per-framework settings

Key sub-targets:
- _LoadRestoreGraphEntryPoints - Identifies projects to restore
- _FilterRestoreGraphProjectInputItems - Filters unsupported projects
- _GetAllRestoreProjectPathItems - Walks project references
- _GenerateRestoreGraphProjectEntry - Top-level entry points
- _GenerateProjectRestoreGraph - Recursively walks dependencies
- _GenerateRestoreProjectSpec - Creates restore spec for each project
- _GenerateProjectRestoreGraphPerFramework - Per-framework dependency collection

#### 3. Dependency Collection Targets (Lines 1072-1187)

For each project and target framework, collects:
- PackageReferences (via GetRestorePackageReferencesTask)
- ProjectReferences (via GetRestoreProjectReferencesTask)
- FrameworkReferences (via GetRestoreFrameworkReferencesTask)
- PackageDownloads (via GetRestorePackageDownloadsTask)
- CentralPackageVersions (via GetCentralPackageVersionsTask)
- NuGetAuditSuppressions (via GetRestoreNuGetAuditSuppressionsTask)
- PrunePackageReferences (via GetRestorePrunePackageReferencesTask)
- TargetFrameworkInformation (framework metadata)

Each is converted to a RestoreGraphEntry item that contains:
- Project name and path
- Package references with versions
- Framework target information
- Package sources and settings

### Restore Graph Structure

The _RestoreGraphEntry items form a complete dependency graph in an internal format that includes:
- RestoreSpec - Specification for a project to restore
- ProjectReference - Project dependency
- PackageReference - NuGet package dependency
- FrameworkReference - .NET framework dependency
- TargetFrameworkInformation - Framework compatibility data
- DotnetCliToolReference - Tool dependencies

## 4. NUGET.BUILD.TASKS.DLL TASKS

### Location

- /Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks/RestoreTask.cs
- /Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks/RestoreTaskEx.cs
- /Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks/BuildTasksUtility.cs

### RestoreTask.Execute() Flow

```csharp
public override bool Execute()
{
    // 1. Setup logging
    var log = new MSBuildLogger(Log);

    // 2. Run migrations
    NuGet.Common.Migrations.MigrationRunner.Run();

    try
    {
        // 3. Execute async restore
        return ExecuteAsync(log).Result;
    }
    catch (OperationCanceledException)
    {
        log.LogError(Strings.RestoreCanceled);
        return false;
    }
}

private async Task<bool> ExecuteAsync(Common.ILogger log)
{
    // 4. Convert MSBuild items to internal wrapper
    var wrappedItems = RestoreGraphItems.Select(MSBuildUtility.WrapMSBuildItem);

    // 5. Parse dependency spec from graph items
    var dgFile = MSBuildRestoreUtility.GetDependencySpec(wrappedItems, readOnly: true);

    // 6. Call BuildTasksUtility.RestoreAsync() with the dependency graph
    var restoreSummaries = await BuildTasksUtility.RestoreAsync(
        dependencyGraphSpec: dgFile,
        interactive: Interactive,
        recursive: RestoreRecursive,
        noCache: RestoreNoCache || RestoreNoHttpCache,
        ignoreFailedSources: RestoreIgnoreFailedSources,
        disableParallel: RestoreDisableParallel,
        force: RestoreForce,
        forceEvaluate: RestoreForceEvaluate,
        hideWarningsAndErrors: HideWarningsAndErrors,
        restorePC: RestorePackagesConfig,
        log: log,
        cancellationToken: _cts.Token);

    // 7. Return results
    ProjectsRestored = restoreSummaries.Count;
    ProjectsAlreadyUpToDate = upToDate;
    ProjectsAudited = audited;
    return restoreSummaries.All(s => s.Success);
}
```

### BuildTasksUtility.RestoreAsync() (Line 116-299)

This is the core restore orchestration:

```csharp
public static async Task<List<RestoreSummary>> RestoreAsync(
    DependencyGraphSpec dependencyGraphSpec,
    bool interactive,
    bool recursive,
    bool noCache,
    bool ignoreFailedSources,
    bool disableParallel,
    bool force,
    bool forceEvaluate,
    bool hideWarningsAndErrors,
    bool restorePC,
    bool cleanupAssetsForUnsupportedProjects,
    Common.ILogger log,
    CancellationToken cancellationToken)
{
    // 1. Setup credentials service
    DefaultCredentialServiceUtility.SetupDefaultCredentialService(log, !interactive);

    // 2. Setup network
    NetworkProtocolUtility.SetConnectionLimit();

    // 3. Setup user agent for NuGet
    UserAgent.SetUserAgentString(new UserAgentStringBuilder("NuGet .NET Core MSBuild Task"));

    // 4. Setup certificate trust store
    X509TrustStore.InitializeForDotNetSdk(log);

    // 5. Create cache context
    using (var cacheContext = new SourceCacheContext())
    {
        cacheContext.NoCache = noCache;
        cacheContext.IgnoreFailedSources = ignoreFailedSources;

        // 6. Create pre-loaded request provider from dependency graph
        var providers = new List<IPreLoadedRestoreRequestProvider>();
        providers.Add(new DependencyGraphSpecRequestProvider(providerCache, dependencyGraphSpec));

        // 7. Create restore context with all settings
        var restoreContext = new RestoreArgs()
        {
            CacheContext = cacheContext,
            DisableParallel = disableParallel,
            Log = log,
            MachineWideSettings = new XPlatMachineWideSetting(),
            PreLoadedRequestProviders = providers,
            AllowNoOp = !force,
            HideWarningsAndErrors = hideWarningsAndErrors,
            RestoreForceEvaluate = forceEvaluate
        };

        // 8. Call RestoreRunner.RunAsync() to execute restore
        restoreSummaries.AddRange(
            await RestoreRunner.RunAsync(restoreContext, cancellationToken));
    }

    // 9. Cleanup unsupported projects' assets if needed
    if (cleanupAssetsForUnsupportedProjects)
    {
        // Delete .assets.json and other files for projects that no longer use PackageReference
    }

    return restoreSummaries;
}
```

## 5. RESTORE RUNNER (Core Orchestration)

### Location

- /Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/RestoreRunner.cs

### Flow

```csharp
public static async Task<IReadOnlyList<RestoreSummary>> RunAsync(
    RestoreArgs restoreContext,
    CancellationToken token)
{
    // 1. Get restore requests from dependency graph
    var requests = await GetRequests(restoreContext);

    // 2. Run requests (parallel execution)
    return await RunAsync(requests, restoreContext, token);
}

private static async Task<IReadOnlyList<RestoreSummary>> RunAsync(
    IEnumerable<RestoreSummaryRequest> restoreRequests,
    RestoreArgs restoreArgs,
    CancellationToken token)
{
    // 1. Determine max task count for parallel execution
    int maxTasks = GetMaxTaskCount(restoreArgs);

    // 2. Queue all requests
    var requests = new Queue<RestoreSummaryRequest>(restoreRequests);
    var restoreTasks = new List<Task<RestoreSummary>>(maxTasks);

    // 3. Execute in parallel (respecting maxTasks limit)
    while (requests.Count > 0)
    {
        // Throttle if we have maxTasks running
        if (restoreTasks.Count == maxTasks)
        {
            var restoreSummary = await CompleteTaskAsync(restoreTasks);
            restoreSummaries.Add(restoreSummary);
        }

        var request = requests.Dequeue();
        var task = Task.Run(() => ExecuteAndCommitAsync(request, restoreArgs.ProgressReporter, token), token);
        restoreTasks.Add(task);
    }

    // 4. Wait for remaining tasks to complete
    while (restoreTasks.Count > 0)
    {
        var restoreSummary = await CompleteTaskAsync(restoreTasks);
        restoreSummaries.Add(restoreSummary);
    }

    return restoreSummaries;
}

private static async Task<RestoreSummary> ExecuteAndCommitAsync(
    RestoreSummaryRequest summaryRequest,
    IRestoreProgressReporter progressReporter,
    CancellationToken token)
{
    // 1. Execute restore
    RestoreResultPair result = await ExecuteAsync(summaryRequest, token);

    // 2. Commit results (write assets file, etc.)
    return await CommitAsync(result, progressReporter, token);
}

private static async Task<RestoreResultPair> ExecuteAsync(
    RestoreSummaryRequest summaryRequest,
    CancellationToken token)
{
    var request = summaryRequest.Request;

    // 1. Create RestoreCommand instance
    var command = new RestoreCommand(request);

    // 2. Execute the restore command (see section 6)
    var result = await command.ExecuteAsync(token);

    return new RestoreResultPair(summaryRequest, result);
}
```

## 6. RESTORE COMMAND (Core Restoration Logic)

### Location

- /Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/RestoreCommand.cs

### Responsibilities

- Dependency resolution (using DependencyResolver)
- Lock file management (packages.lock.json)
- Assets file generation (project.assets.json)
- MSBuild props/targets file generation (project.csproj.nuget.g.props/targets)
- NuGet audit (vulnerability scanning)
- Package pruning (removing unused packages)

### Key Methods

1. **ExecuteAsync(CancellationToken token)**
   - Main entry point
   - Orchestrates entire restore process
   - Returns RestoreResult

2. **Major Steps:**
   - Evaluate lock file if present
   - Generate restore graph (dependency tree)
   - Resolve dependencies for each target framework
   - Validate resolved packages
   - Generate assets file
   - Write MSBuild props/targets
   - Run NuGet audit if enabled
   - Return results

## 7. COMPLETE FLOW DIAGRAM

```
User Input
    |
    v
dotnet restore command
    |
    v
RestoreCommand.Run() -> RestoringCommand
    |
    v
MSBuildForwardingApp.Execute()
    |
    v (spawns or calls in-process)
MSBuild.exe with NuGet.targets
    |
    +----> NuGet.targets: Restore target
            |
            +----> _GenerateRestoreGraph
                    |
                    +----> _LoadRestoreGraphEntryPoints (find projects)
                    +----> _FilterRestoreGraphProjectInputItems (filter types)
                    +----> _GetAllRestoreProjectPathItems (walk projects)
                    +----> _GenerateRestoreGraphProjectEntry (entry points)
                    +----> _GenerateProjectRestoreGraph (per project)
                            |
                            +----> _GenerateProjectRestoreGraphPerFramework
                                    |
                                    +----> GetRestoreProjectReferencesTask
                                    +----> GetRestorePackageReferencesTask
                                    +----> GetRestoreFrameworkReferencesTask
                                    +----> GetRestorePackageDownloadsTask
                                    +----> GetRestoreNuGetAuditSuppressionsTask
                                    +----> GetRestorePrunePackageReferencesTask
            |
            +----> RestoreTask (from NuGet.Build.Tasks.dll)
                    |
                    v
            RestoreTask.Execute()
                    |
                    +----> BuildTasksUtility.RestoreAsync()
                            |
                            +----> RestoreRunner.RunAsync()
                                    |
                                    +----> (parallel) ExecuteAndCommitAsync()
                                            |
                                            +----> RestoreCommand.ExecuteAsync()
                                                    |
                                                    +----> Resolve dependencies
                                                    +----> Generate assets.json
                                                    +----> Generate props/targets
                                                    +----> Run audit
                                            |
                                            +----> CommitAsync()
                                                    |
                                                    +----> Write files to disk
```

## 8. KEY DATA STRUCTURES

### DependencyGraphSpec

- Represents the complete project dependency graph
- Created by NuGet.targets by walking MSBuild projects
- Contains: projects, packages, frameworks, settings
- Serialized as dgspec.json file
- Input to RestoreRunner.RunAsync()

### RestoreRequest

- Represents a single project to restore
- Created from DependencyGraphSpec entries
- Contains: project spec, package sources, cache context, etc.

### RestoreSummaryRequest

- Wrapper around RestoreRequest with metadata
- Tracks input path, project path for reporting

### RestoreResult

- Represents the result of restoring a single project
- Contains: lock file, assets file, errors/warnings
- Different types: NoOpRestoreResult (cached), UpdatedRestoreResult, etc.

### RestoreSummary

- Final result reported to user
- Success/failure status, project count, audit results

## 9. KEY CONFIGURATION FILES

### NuGet.config

- Located at project root or parent directories
- Specifies package sources
- Specifies feed credentials
- Specifies repository paths

### Directory.Packages.props

- Used for Central Package Management (CPM)
- Defines versions of packages used across solution
- Auto-imported when ManagePackageVersionsCentrally=true

### packages.lock.json

- Lock file for reproducible restores
- Optional, created when RestorePackagesWithLockFile=true

### project.assets.json

- Output of restore
- Complete resolved dependency graph
- Used by build and run targets

### project.csproj.nuget.g.props / project.csproj.nuget.g.targets

- Generated MSBuild files
- Props file: sets up NuGet properties for build
- Targets file: implements package asset inclusion during build

## 10. SUMMARY

The restore flow involves three major layers:

1. **dotnet CLI (SDK)**: Parses arguments, invokes MSBuild
2. **MSBuild + NuGet.targets**: Walks projects, builds dependency graph, invokes RestoreTask
3. **NuGet.Client**: Resolves dependencies, downloads packages, generates output files

This separation allows NuGet to be:
- Used independently via NuGet.exe
- Used from Visual Studio via NuGetVS
- Used from dotnet CLI via SDK
- Used from MSBuild directly via targets file

All paths funnel through RestoreRunner -> RestoreCommand which implements the core logic.