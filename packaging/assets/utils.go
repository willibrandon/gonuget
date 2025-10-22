package assets

import (
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/frameworks"
)

// GetLockFileItems selects assets using criteria and patterns.
// Returns paths of items from the best matching group.
// Reference: LockFileUtils.cs GetLockFileItems (Lines 663-713)
func GetLockFileItems(criteria *SelectionCriteria, collection *ContentItemCollection, patternSets ...*PatternSet) []string {
	group := collection.FindBestItemGroup(criteria, patternSets...)
	if group == nil {
		return []string{}
	}

	paths := make([]string, len(group.Items))
	for i, item := range group.Items {
		paths[i] = item.Path
	}
	return paths
}

// FilterToDllExe filters paths to DLL/EXE/WINMD files.
func FilterToDllExe(paths []string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".dll" || ext == ".exe" || ext == ".winmd" {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

// GetLibItems gets runtime assemblies for target framework.
// Uses RuntimeAssemblies pattern set and filters to DLL/EXE/WINMD files.
// Reference: LockFileUtils.cs CreateLockFileTargetLibrary (Lines 184-190)
func GetLibItems(files []string, targetFramework *frameworks.NuGetFramework, conventions *ManagedCodeConventions) []string {
	collection := NewContentItemCollection(files)
	criteria := ForFramework(targetFramework, conventions.Properties)

	paths := GetLockFileItems(criteria, collection, conventions.RuntimeAssemblies)
	return FilterToDllExe(paths)
}

// GetRefItems gets compile-time reference assemblies.
// Tries CompileRefAssemblies first (ref/ folder), then CompileLibAssemblies (lib/ folder) as fallback.
// Reference: LockFileUtils.cs CreateLockFileTargetLibrary (Lines 177-183)
func GetRefItems(files []string, targetFramework *frameworks.NuGetFramework, conventions *ManagedCodeConventions) []string {
	collection := NewContentItemCollection(files)
	criteria := ForFramework(targetFramework, conventions.Properties)

	// Compile: ref takes precedence over lib
	return GetLockFileItems(criteria, collection, conventions.CompileRefAssemblies, conventions.CompileLibAssemblies)
}
