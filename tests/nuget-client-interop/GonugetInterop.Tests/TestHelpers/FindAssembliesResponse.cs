using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from find_runtime_assemblies or find_compile_assemblies operations.
/// Contains the list of matched content items with their properties.
/// </summary>
public class FindAssembliesResponse
{
    /// <summary>
    /// List of content items matching the specified criteria.
    /// Each item contains the path and extracted properties.
    /// </summary>
    public ContentItemData[] Items { get; set; } = Array.Empty<ContentItemData>();
}
