using System.Collections.Generic;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Represents a content item with its path and extracted properties.
/// Matches the structure returned by gonuget's asset selection engine.
/// </summary>
public class ContentItemData
{
    /// <summary>
    /// The file path of the content item (e.g., "lib/net6.0/MyLib.dll").
    /// </summary>
    public string Path { get; set; } = "";

    /// <summary>
    /// Dictionary of extracted properties from the path.
    /// Keys include: "tfm", "assembly", "rid", "locale", "codeLanguage", etc.
    /// Values can be strings or complex objects (serialized as JSON).
    /// </summary>
    public Dictionary<string, object> Properties { get; set; } = new();
}
