package resolver

import (
	"github.com/willibrandon/gonuget/frameworks"
)

// FrameworkSelector selects the best dependency group for a target framework.
// Matches NuGet.Frameworks.FrameworkReducer behavior.
type FrameworkSelector struct {
	provider frameworks.FrameworkNameProvider
}

// NewFrameworkSelector creates a new framework selector.
func NewFrameworkSelector() *FrameworkSelector {
	return &FrameworkSelector{
		provider: frameworks.DefaultFrameworkNameProvider(),
	}
}

// SelectDependencies selects dependencies from groups based on target framework.
// Implements NuGet's framework compatibility and reduction logic.
func (fs *FrameworkSelector) SelectDependencies(
	groups []DependencyGroup,
	targetFramework string,
) []PackageDependency {
	if len(groups) == 0 {
		return nil
	}

	// Parse target framework
	target, err := frameworks.ParseFramework(targetFramework)
	if err != nil {
		return nil
	}

	// Find all compatible groups
	compatibleGroups := make([]DependencyGroup, 0)
	for _, group := range groups {
		if group.TargetFramework == "" {
			// Untargeted group is always compatible
			compatibleGroups = append(compatibleGroups, group)
			continue
		}

		groupFw, err := frameworks.ParseFramework(group.TargetFramework)
		if err != nil {
			continue
		}

		// Check if group framework is compatible with target framework
		// e.g., "Can net8.0 (target) use a package built for netstandard2.0 (groupFw)?"
		// Matches NuGetFramework.IsCompatible semantics: packageFw.IsCompatible(projectFw)
		if groupFw.IsCompatible(target) {
			compatibleGroups = append(compatibleGroups, group)
		}
	}

	if len(compatibleGroups) == 0 {
		return nil
	}

	// If only one compatible group, use it
	if len(compatibleGroups) == 1 {
		return compatibleGroups[0].Dependencies
	}

	// Find nearest (most specific) compatible framework
	nearest := fs.findNearest(compatibleGroups, target)
	if nearest != nil {
		return nearest.Dependencies
	}

	// Fall back to untargeted group
	for _, group := range compatibleGroups {
		if group.TargetFramework == "" {
			return group.Dependencies
		}
	}

	return nil
}

// findNearest finds the nearest compatible framework using FrameworkReducer.
func (fs *FrameworkSelector) findNearest(
	groups []DependencyGroup,
	target *frameworks.NuGetFramework,
) *DependencyGroup {
	// Convert groups to frameworks
	fws := make([]*frameworks.NuGetFramework, 0, len(groups))
	for _, group := range groups {
		if group.TargetFramework == "" {
			continue
		}
		fw, err := frameworks.ParseFramework(group.TargetFramework)
		if err != nil {
			continue
		}
		fws = append(fws, fw)
	}

	// Use frameworks.GetNearest to find nearest
	nearest := frameworks.GetNearest(target, fws)
	if nearest == nil {
		return nil
	}

	// Find group with nearest framework
	for i := range groups {
		if groups[i].TargetFramework == nearest.GetShortFolderName(fs.provider) {
			return &groups[i]
		}
	}

	return nil
}
