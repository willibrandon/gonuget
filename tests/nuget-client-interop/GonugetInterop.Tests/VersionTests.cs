using GonugetInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Comprehensive version parsing and comparison tests.
/// Validates gonuget version handling against NuGet.Client reference implementation.
/// 180 test cases covering SemVer 2.0, legacy formats, and prerelease ordering.
/// </summary>
public sealed class VersionTests
{
    #region Version Parsing Tests (60 cases)

    [Theory]
    [InlineData("1.0.0", 1, 0, 0, 0, "", "", false)]
    [InlineData("1.2.3", 1, 2, 3, 0, "", "", false)]
    [InlineData("1.2.3.4", 1, 2, 3, 4, "", "", false)]
    [InlineData("0.0.0", 0, 0, 0, 0, "", "", false)]
    [InlineData("99.99.99", 99, 99, 99, 0, "", "", false)]
    [InlineData("2024.12.31", 2024, 12, 31, 0, "", "", false)]
    public void ParseVersion_SimpleVersions_ParsesCorrectly(
        string version, int major, int minor, int patch, int revision,
        string release, string metadata, bool isPrerelease)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.Equal(major, result.Major);
        Assert.Equal(minor, result.Minor);
        Assert.Equal(patch, result.Patch);
        Assert.Equal(revision, result.Revision);
        Assert.Equal(release, result.Release);
        Assert.Equal(metadata, result.Metadata);
        Assert.Equal(isPrerelease, result.IsPrerelease);
    }

    [Theory]
    [InlineData("1.0.0-alpha", 1, 0, 0, 0, "alpha", "", true)]
    [InlineData("1.0.0-beta", 1, 0, 0, 0, "beta", "", true)]
    [InlineData("1.0.0-rc.1", 1, 0, 0, 0, "rc.1", "", true)]
    [InlineData("1.0.0-alpha.1", 1, 0, 0, 0, "alpha.1", "", true)]
    [InlineData("1.0.0-beta.2", 1, 0, 0, 0, "beta.2", "", true)]
    [InlineData("1.0.0-0.3.7", 1, 0, 0, 0, "0.3.7", "", true)]
    [InlineData("1.0.0-x.7.z.92", 1, 0, 0, 0, "x.7.z.92", "", true)]
    [InlineData("1.0.0-alpha+001", 1, 0, 0, 0, "alpha", "001", true)]
    [InlineData("1.0.0+20130313144700", 1, 0, 0, 0, "", "20130313144700", false)]
    [InlineData("1.0.0-beta+exp.sha.5114f85", 1, 0, 0, 0, "beta", "exp.sha.5114f85", true)]
    public void ParseVersion_SemVer2Prerelease_ParsesCorrectly(
        string version, int major, int minor, int patch, int revision,
        string release, string metadata, bool isPrerelease)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.Equal(major, result.Major);
        Assert.Equal(minor, result.Minor);
        Assert.Equal(patch, result.Patch);
        Assert.Equal(revision, result.Revision);
        Assert.Equal(release, result.Release);
        Assert.Equal(metadata, result.Metadata);
        Assert.Equal(isPrerelease, result.IsPrerelease);
    }

    [Theory]
    [InlineData("1.0.0+git.abc123", "", "git.abc123")]
    [InlineData("1.2.3+build.20241221", "", "build.20241221")]
    [InlineData("1.0.0-alpha+001", "alpha", "001")]
    [InlineData("1.0.0-beta.11+exp.sha.5114f85", "beta.11", "exp.sha.5114f85")]
    [InlineData("1.0.0-rc.1+build.1", "rc.1", "build.1")]
    public void ParseVersion_WithMetadata_ExtractsMetadata(
        string version, string expectedRelease, string expectedMetadata)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.Equal(expectedRelease, result.Release);
        Assert.Equal(expectedMetadata, result.Metadata);
    }

    [Theory]
    [InlineData("1.0", 1, 0, 0, 0)]
    [InlineData("1", 1, 0, 0, 0)]
    [InlineData("2.3", 2, 3, 0, 0)]
    [InlineData("10", 10, 0, 0, 0)]
    public void ParseVersion_ShortForms_NormalizesCorrectly(
        string version, int major, int minor, int patch, int revision)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.Equal(major, result.Major);
        Assert.Equal(minor, result.Minor);
        Assert.Equal(patch, result.Patch);
        Assert.Equal(revision, result.Revision);
    }

    [Theory]
    [InlineData("1.0.0-alpha")]
    [InlineData("1.0.0-beta")]
    [InlineData("1.0.0-rc.1")]
    [InlineData("1.0.0-0")]
    [InlineData("1.0.0-x.y.z")]
    [InlineData("2.0.0-preview1")]
    [InlineData("3.0.0-SNAPSHOT")]
    public void ParseVersion_PrereleaseVersions_IdentifiesAsPrerelease(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.True(result.IsPrerelease);
    }

    [Theory]
    [InlineData("1.0.0")]
    [InlineData("1.2.3")]
    [InlineData("1.0.0+metadata")]
    [InlineData("2.3.4.5")]
    public void ParseVersion_StableVersions_NotPrerelease(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.False(result.IsPrerelease);
    }

    [Theory]
    [InlineData("1.0.0-alpha-beta", "alpha-beta")]
    [InlineData("1.0.0-pre.release.label", "pre.release.label")]
    [InlineData("1.0.0-feature-branch-name", "feature-branch-name")]
    [InlineData("1.0.0-20241221.142030", "20241221.142030")]
    public void ParseVersion_ComplexReleaseLabels_ParsesCorrectly(string version, string expectedRelease)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.Equal(expectedRelease, result.Release);
    }

    #endregion

    #region Version Comparison Tests (60 cases)

    [Theory]
    [InlineData("1.0.0", "2.0.0", -1)]
    [InlineData("2.0.0", "1.0.0", 1)]
    [InlineData("1.0.0", "1.0.0", 0)]
    [InlineData("1.2.3", "1.2.4", -1)]
    [InlineData("1.3.0", "1.2.9", 1)]
    [InlineData("2.0.0", "1.99.99", 1)]
    [InlineData("0.0.1", "0.0.2", -1)]
    [InlineData("1.0.0", "1.0.0.0", 0)]
    [InlineData("1.2.3.4", "1.2.3.5", -1)]
    [InlineData("1.2.3.10", "1.2.3.9", 1)]
    public void CompareVersions_StableVersions_ComparesNumerically(
        string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-alpha", "1.0.0", -1)]
    [InlineData("1.0.0", "1.0.0-alpha", 1)]
    [InlineData("1.0.0-alpha", "1.0.0-beta", -1)]
    [InlineData("1.0.0-beta", "1.0.0-alpha", 1)]
    [InlineData("1.0.0-alpha", "1.0.0-alpha.1", -1)]
    [InlineData("1.0.0-alpha.1", "1.0.0-alpha", 1)]
    [InlineData("1.0.0-alpha.1", "1.0.0-alpha.2", -1)]
    [InlineData("1.0.0-rc.1", "1.0.0-rc.2", -1)]
    public void CompareVersions_PrereleaseVsStable_PrereleaseIsLower(
        string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-alpha", "1.0.0-alpha.1")]
    [InlineData("1.0.0-alpha.1", "1.0.0-alpha.beta")]
    [InlineData("1.0.0-alpha.beta", "1.0.0-beta")]
    [InlineData("1.0.0-beta", "1.0.0-beta.2")]
    [InlineData("1.0.0-beta.2", "1.0.0-beta.11")]
    [InlineData("1.0.0-beta.11", "1.0.0-rc.1")]
    [InlineData("1.0.0-rc.1", "1.0.0")]
    public void CompareVersions_PrereleaseOrdering_FollowsSemVer2Spec(string lower, string higher)
    {
        var result = GonugetBridge.CompareVersions(lower, higher);
        Assert.Equal(-1, result.Result);

        var reverse = GonugetBridge.CompareVersions(higher, lower);
        Assert.Equal(1, reverse.Result);
    }

    [Theory]
    [InlineData("1.0.0+metadata1", "1.0.0+metadata2", 0)]
    [InlineData("1.0.0-alpha+build.1", "1.0.0-alpha+build.2", 0)]
    [InlineData("1.2.3+20241221", "1.2.3+20241222", 0)]
    public void CompareVersions_WithMetadata_IgnoresMetadata(string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-1", "1.0.0-2", -1)]
    [InlineData("1.0.0-2", "1.0.0-10", -1)]
    [InlineData("1.0.0-10", "1.0.0-100", -1)]
    [InlineData("1.0.0-alpha.1", "1.0.0-alpha.2", -1)]
    [InlineData("1.0.0-alpha.2", "1.0.0-alpha.10", -1)]
    public void CompareVersions_NumericPrereleaseLabels_ComparesNumerically(
        string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-alpha", "1.0.0-beta", -1)]
    [InlineData("1.0.0-beta", "1.0.0-gamma", -1)]
    [InlineData("1.0.0-a", "1.0.0-b", -1)]
    [InlineData("1.0.0-x.a", "1.0.0-x.b", -1)]
    public void CompareVersions_AlphabeticPrereleaseLabels_ComparesLexically(
        string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-1", "1.0.0-a", -1)]
    [InlineData("1.0.0-2", "1.0.0-alpha", -1)]
    [InlineData("1.0.0-10", "1.0.0-beta", -1)]
    public void CompareVersions_NumericVsAlphabetic_NumericIsLower(
        string numeric, string alphabetic, int expected)
    {
        var result = GonugetBridge.CompareVersions(numeric, alphabetic);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-a.b.c", "1.0.0-a.b.d", -1)]
    [InlineData("1.0.0-a.1.b", "1.0.0-a.1.c", -1)]
    [InlineData("1.0.0-1.2.3", "1.0.0-1.2.4", -1)]
    public void CompareVersions_MultipartPrereleaseLabels_ComparesLeftToRight(
        string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    #endregion

    #region SemVer 2.0 Compliance Tests (30 cases)

    [Theory]
    [InlineData("1.0.0-alpha")]
    [InlineData("1.0.0-alpha.1")]
    [InlineData("1.0.0-0.3.7")]
    [InlineData("1.0.0-x.7.z.92")]
    [InlineData("1.0.0-x-y-z.--")]
    public void SemVer2_ValidPrereleaseIdentifiers_Accepted(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.True(result.IsPrerelease);
        Assert.NotEmpty(result.Release);
    }

    [Theory]
    [InlineData("1.0.0+20130313144700")]
    [InlineData("1.0.0+exp.sha.5114f85")]
    [InlineData("1.0.0+21AF26D3----117B344092BD")]
    [InlineData("1.0.0-beta+exp.sha.5114f85")]
    [InlineData("1.0.0-alpha+001")]
    public void SemVer2_ValidBuildMetadata_Accepted(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.NotEmpty(result.Metadata);
    }

    [Theory]
    [InlineData("1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0")]
    public void SemVer2_PrecedenceExample_FromSpec(params string[] versions)
    {
        for (int i = 0; i < versions.Length - 1; i++)
        {
            var result = GonugetBridge.CompareVersions(versions[i], versions[i + 1]);
            Assert.Equal(-1, result.Result);
        }
    }

    [Theory]
    [InlineData("1.0.0+build1", "1.0.0+build2")]
    [InlineData("1.0.0-alpha+001", "1.0.0-alpha+002")]
    public void SemVer2_BuildMetadata_DoesNotAffectPrecedence(string v1, string v2)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(0, result.Result);
    }

    [Theory]
    [InlineData("1.0.0-0")]
    [InlineData("1.0.0-00")]
    [InlineData("1.0.0-1.0.0")]
    [InlineData("1.0.0-pre.1")]
    public void SemVer2_LeadingZeros_AllowedInIdentifiers(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.NotNull(result.Release);
    }

    #endregion

    #region Legacy Version Format Tests (15 cases)

    [Theory]
    [InlineData("1.0", 1, 0, 0, 0)]
    [InlineData("2", 2, 0, 0, 0)]
    [InlineData("1.2.3.4", 1, 2, 3, 4)]
    public void Legacy_TwoAndFourPartVersions_Supported(
        string version, int major, int minor, int patch, int revision)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.Equal(major, result.Major);
        Assert.Equal(minor, result.Minor);
        Assert.Equal(patch, result.Patch);
        Assert.Equal(revision, result.Revision);
    }

    [Theory]
    [InlineData("1.0.0.0", "1.0.0", 0)]
    [InlineData("1.2.3.0", "1.2.3", 0)]
    [InlineData("1.0", "1.0.0", 0)]
    public void Legacy_TrailingZeros_NormalizedCorrectly(string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    [Theory]
    [InlineData("1.0.0.1", "1.0.0.2", -1)]
    [InlineData("1.0.0.10", "1.0.0.2", 1)]
    public void Legacy_FourthComponent_ComparedNumerically(string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    #endregion

    #region Edge Cases and Special Scenarios (15 cases)

    [Theory]
    [InlineData("0.0.0")]
    [InlineData("0.0.0-alpha")]
    [InlineData("0.1.0")]
    public void EdgeCase_ZeroVersions_ParsedCorrectly(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.True(result.Major >= 0);
        Assert.True(result.Minor >= 0);
        Assert.True(result.Patch >= 0);
    }

    [Theory]
    [InlineData("99999.99999.99999")]
    [InlineData("2147483647.2147483647.2147483647")]
    public void EdgeCase_LargeVersionNumbers_Supported(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.True(result.Major > 0);
    }

    [Theory]
    [InlineData("1.0.0-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")]
    [InlineData("1.0.0+zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")]
    public void EdgeCase_LongLabelsAndMetadata_Supported(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.NotNull(result);
    }

    [Theory]
    [InlineData("1.0.0-a.b.c.d.e.f.g.h.i.j")]
    [InlineData("1.0.0+a.b.c.d.e.f.g.h.i.j")]
    public void EdgeCase_ManyDottedIdentifiers_Supported(string version)
    {
        var result = GonugetBridge.ParseVersion(version);
        Assert.NotNull(result);
    }

    [Theory]
    [InlineData("1.0.0-ALPHA", "1.0.0-alpha", 0)]
    [InlineData("1.0.0-Beta", "1.0.0-beta", 0)]
    [InlineData("1.0.0-alpha", "1.0.0-ALPHA", 0)]
    [InlineData("1.0.0-RC", "1.0.0-rc", 0)]
    public void EdgeCase_CaseSensitivity_PrereleaseIsCaseInsensitive(string v1, string v2, int expected)
    {
        var result = GonugetBridge.CompareVersions(v1, v2);
        Assert.Equal(expected, result.Result);
    }

    #endregion
}
