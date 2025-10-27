package version

import (
	"fmt"
	"strings"
)

// Range represents a range of acceptable versions.
//
// Syntax:
//
//	[1.0, 2.0]   - 1.0 ≤ x ≤ 2.0 (inclusive)
//	(1.0, 2.0)   - 1.0 < x < 2.0 (exclusive)
//	[1.0, 2.0)   - 1.0 ≤ x < 2.0 (mixed)
//	[1.0, )      - x ≥ 1.0 (open upper)
//	(, 2.0]      - x ≤ 2.0 (open lower)
//	1.0          - x ≥ 1.0 (implicit minimum)
type Range struct {
	MinVersion   *NuGetVersion
	MaxVersion   *NuGetVersion
	MinInclusive bool
	MaxInclusive bool
}

// ParseVersionRange parses a version range string.
func ParseVersionRange(s string) (*Range, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("version range cannot be empty")
	}

	// Check if it starts with bracket (range syntax)
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "(") {
		return parseRangeSyntax(s)
	}

	// Otherwise, it's a simple version (>= that version)
	v, err := Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid version range: %w", err)
	}

	return &Range{
		MinVersion:   v,
		MinInclusive: true,
		MaxVersion:   nil,
		MaxInclusive: false,
	}, nil
}

// MustParseRange parses a version range string and panics on error.
// Use this only when you know the range string is valid.
func MustParseRange(s string) *Range {
	r, err := ParseVersionRange(s)
	if err != nil {
		panic(err)
	}
	return r
}

// parseRangeSyntax parses bracket range syntax like [1.0, 2.0).
func parseRangeSyntax(s string) (*Range, error) {
	// Determine inclusive/exclusive from brackets
	if !strings.HasPrefix(s, "[") && !strings.HasPrefix(s, "(") {
		return nil, fmt.Errorf("range must start with [ or (")
	}
	if !strings.HasSuffix(s, "]") && !strings.HasSuffix(s, ")") {
		return nil, fmt.Errorf("range must end with ] or )")
	}

	minInclusive := strings.HasPrefix(s, "[")
	maxInclusive := strings.HasSuffix(s, "]")

	// Remove brackets
	s = s[1 : len(s)-1]

	// Split on comma
	parts := strings.Split(s, ",")

	// Handle single-part exact version syntax [1.0.0] -> [1.0.0, 1.0.0]
	var minPart, maxPart string
	switch len(parts) {
	case 1:
		minPart = strings.TrimSpace(parts[0])
		maxPart = minPart // Exact version match
	case 2:
		minPart = strings.TrimSpace(parts[0])
		maxPart = strings.TrimSpace(parts[1])
	default:
		return nil, fmt.Errorf("range must have one or two parts separated by comma")
	}

	var minVersion, maxVersion *NuGetVersion
	var err error

	// Parse min version (empty means no lower bound)
	if minPart != "" {
		minVersion, err = Parse(minPart)
		if err != nil {
			return nil, fmt.Errorf("invalid min version: %w", err)
		}
	}

	// Parse max version (empty means no upper bound)
	if maxPart != "" {
		maxVersion, err = Parse(maxPart)
		if err != nil {
			return nil, fmt.Errorf("invalid max version: %w", err)
		}
	}

	return &Range{
		MinVersion:   minVersion,
		MaxVersion:   maxVersion,
		MinInclusive: minInclusive,
		MaxInclusive: maxInclusive,
	}, nil
}

// Satisfies returns true if the version satisfies this range.
func (r *Range) Satisfies(version *NuGetVersion) bool {
	if version == nil {
		return false
	}

	// Check lower bound
	if r.MinVersion != nil {
		cmp := version.Compare(r.MinVersion)
		if r.MinInclusive {
			if cmp < 0 {
				return false
			}
		} else {
			if cmp <= 0 {
				return false
			}
		}
	}

	// Check upper bound
	if r.MaxVersion != nil {
		cmp := version.Compare(r.MaxVersion)
		if r.MaxInclusive {
			if cmp > 0 {
				return false
			}
		} else {
			if cmp >= 0 {
				return false
			}
		}
	}

	return true
}

// FindBestMatch finds the highest version that satisfies this range.
//
// Returns nil if no version satisfies the range.
func (r *Range) FindBestMatch(versions []*NuGetVersion) *NuGetVersion {
	var best *NuGetVersion

	for _, v := range versions {
		if r.Satisfies(v) {
			if best == nil || v.GreaterThan(best) {
				best = v
			}
		}
	}

	return best
}

// String returns the string representation of the range.
func (r *Range) String() string {
	minBracket := "("
	if r.MinInclusive {
		minBracket = "["
	}
	maxBracket := ")"
	if r.MaxInclusive {
		maxBracket = "]"
	}

	minStr := ""
	if r.MinVersion != nil {
		minStr = r.MinVersion.String()
	}

	maxStr := ""
	if r.MaxVersion != nil {
		maxStr = r.MaxVersion.String()
	}

	return fmt.Sprintf("%s%s, %s%s", minBracket, minStr, maxStr, maxBracket)
}
