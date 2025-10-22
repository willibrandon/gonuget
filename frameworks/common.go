package frameworks

// CommonFrameworks provides common .NET framework instances.
var CommonFrameworks = struct {
	DotNet *NuGetFramework
	Net    *NuGetFramework
}{
	// DotNet represents the .NETCoreApp framework (.NET 5+).
	DotNet: &NuGetFramework{
		Framework: ".NETCoreApp",
		Version:   FrameworkVersion{Major: 5, Minor: 0},
	},
	// Net represents the legacy .NETFramework 4.5.
	Net: &NuGetFramework{
		Framework: ".NETFramework",
		Version:   FrameworkVersion{Major: 4, Minor: 5},
	},
}

// IsCompatible checks if the package framework is compatible with the target framework.
// This is a convenience function that wraps the NuGetFramework.IsCompatible method.
func IsCompatible(pkg, target *NuGetFramework) bool {
	if pkg == nil || target == nil {
		return false
	}
	return pkg.IsCompatible(target)
}

// FrameworkReducer helps find the nearest compatible framework.
type FrameworkReducer struct{}

// NewFrameworkReducer creates a new framework reducer.
func NewFrameworkReducer() *FrameworkReducer {
	return &FrameworkReducer{}
}

// GetNearest finds the nearest compatible framework from available frameworks.
func (fr *FrameworkReducer) GetNearest(target *NuGetFramework, available []*NuGetFramework) *NuGetFramework {
	return GetNearest(target, available)
}
