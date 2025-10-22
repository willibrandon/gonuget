namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the parse_asset_path operation.
/// Contains the parsed properties from a single asset path.
/// </summary>
public class ParseAssetPathResponse
{
    /// <summary>
    /// The parsed content item with extracted properties.
    /// Null if the path did not match any known pattern.
    /// </summary>
    public ContentItemData? Item { get; set; }
}
