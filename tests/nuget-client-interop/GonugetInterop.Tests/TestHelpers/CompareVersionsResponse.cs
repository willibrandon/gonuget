namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Response from the compare_versions operation containing the comparison result.
/// </summary>
public class CompareVersionsResponse
{
    /// <summary>
    /// The comparison result: -1 if version1 &lt; version2, 0 if equal, 1 if version1 &gt; version2.
    /// </summary>
    public int Result { get; set; }
}
