package frameworks

// GetNearest finds the nearest compatible framework from a list.
//
// Given a target framework and a list of available frameworks,
// returns the most compatible one, preferring:
// 1. Exact match
// 2. Same framework, nearest lower version
// 3. Compatible framework with highest precedence
//
// Returns nil if no compatible framework found.
func GetNearest(target *NuGetFramework, available []*NuGetFramework) *NuGetFramework {
	if target == nil || len(available) == 0 {
		return nil
	}

	var best *NuGetFramework
	var bestScore int

	for _, fw := range available {
		if !fw.IsCompatible(target) {
			continue
		}

		score := calculateCompatibilityScore(fw, target)
		if best == nil || score > bestScore {
			best = fw
			bestScore = score
		}
	}

	return best
}

// calculateCompatibilityScore calculates how well a framework matches the target.
// Higher score = better match.
func calculateCompatibilityScore(fw, target *NuGetFramework) int {
	score := 0

	// Exact match gets highest score
	if fw.Framework == target.Framework && fw.Version.Compare(target.Version) == 0 {
		return 1000
	}

	// Same framework family - strongly prefer when close in version
	if fw.Framework == target.Framework {
		score += 800

		// Closer version gets significant bonus for same framework
		versionDiff := target.Version.Compare(fw.Version)
		if versionDiff >= 0 {
			// Target version >= package version (good)
			// For .NET Framework, don't give huge bonuses to older versions
			if target.Framework == ".NETFramework" && versionDiff > 0 {
				// Older .NET Framework versions get less bonus
				score += 50
			} else {
				switch {
				case versionDiff == 0:
					// Same version
					score += 150
				case versionDiff <= 2:
					// Very close versions get bonus
					score += 120 - (versionDiff * 30)
				default:
					// Distant versions
					score += 50 - (versionDiff * 5)
				}
			}
		}
		return score
	}

	// Cross-framework compatibility
	// Prefer .NET Standard for maximum compatibility, especially for .NET Framework targets
	if fw.Framework == ".NETStandard" {
		baseScore := 700
		// Give extra bonus for .NET Framework targets
		if target.Framework == ".NETFramework" {
			baseScore = 850
		}
		score += baseScore
		// Higher .NET Standard versions are better
		score += fw.Version.Major*20 + fw.Version.Minor
		return score
	}

	// Framework precedence for other frameworks
	precedence := GetFrameworkPrecedence(fw.Framework)
	score += precedence * 10

	return score
}
