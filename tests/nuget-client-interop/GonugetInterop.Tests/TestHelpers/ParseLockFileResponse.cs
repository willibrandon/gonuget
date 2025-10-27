using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the parse_lock_file operation.
/// Represents a parsed project.assets.json file.
/// </summary>
public class ParseLockFileResponse
{
    /// <summary>
    /// Lock file format version (typically 3).
    /// </summary>
    public int Version { get; set; }

    /// <summary>
    /// Targets section (per-TFM resolved packages).
    /// </summary>
    public Dictionary<string, object> Targets { get; set; } = new();

    /// <summary>
    /// Libraries section (package metadata).
    /// </summary>
    public Dictionary<string, LockFileLibrary> Libraries { get; set; } = new();

    /// <summary>
    /// Top-level project dependencies per framework.
    /// </summary>
    public Dictionary<string, List<string>> ProjectFileDependencyGroups { get; set; } = new();

    /// <summary>
    /// Package folders (global package cache locations).
    /// </summary>
    public Dictionary<string, object> PackageFolders { get; set; } = new();

    /// <summary>
    /// Project metadata.
    /// </summary>
    public LockFileProject Project { get; set; } = new();
}

/// <summary>
/// Lock file library entry.
/// </summary>
public class LockFileLibrary
{
    public string Type { get; set; } = "";
    public string Path { get; set; } = "";
}

/// <summary>
/// Lock file project metadata.
/// </summary>
public class LockFileProject
{
    public string Version { get; set; } = "";
    public LockFileRestore Restore { get; set; } = new();
    public Dictionary<string, LockFileFramework> Frameworks { get; set; } = new();
}

/// <summary>
/// Lock file restore metadata.
/// </summary>
public class LockFileRestore
{
    public string ProjectUniqueName { get; set; } = "";
    public string ProjectName { get; set; } = "";
    public string ProjectPath { get; set; } = "";
    public string PackagesPath { get; set; } = "";
    public string OutputPath { get; set; } = "";
    public string ProjectStyle { get; set; } = "";
    public Dictionary<string, object> Sources { get; set; } = new();
    public List<string> FallbackFolders { get; set; } = new();
    public List<string> ConfigFilePaths { get; set; } = new();
    public List<string> OriginalTargetFrameworks { get; set; } = new();
    public Dictionary<string, LockFileRestoreFramework> Frameworks { get; set; } = new();
}

/// <summary>
/// Lock file restore framework info.
/// </summary>
public class LockFileRestoreFramework
{
    public string TargetAlias { get; set; } = "";
    public Dictionary<string, object> ProjectReferences { get; set; } = new();
}

/// <summary>
/// Lock file framework info.
/// </summary>
public class LockFileFramework
{
    public string TargetAlias { get; set; } = "";
    public Dictionary<string, LockFileDependency> Dependencies { get; set; } = new();
}

/// <summary>
/// Lock file dependency entry.
/// </summary>
public class LockFileDependency
{
    public string Target { get; set; } = "";
    public string Version { get; set; } = "";
}
