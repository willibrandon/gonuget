package packaging

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/willibrandon/gonuget/version"
)

const (
	// MaxPackageIDLength is the maximum allowed package ID length
	// Reference: PackageIdValidator.cs
	MaxPackageIDLength = 100
)

var (
	// Package ID pattern: must start with letter or underscore,
	// can contain letters, digits, periods, hyphens, underscores
	packageIDPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9._-]*$`)
)

// ValidatePackageID validates a package ID
// Reference: PackageIdValidator.cs IsValidPackageId
func ValidatePackageID(id string) error {
	if id == "" {
		return fmt.Errorf("package ID cannot be empty")
	}

	if len(id) > MaxPackageIDLength {
		return fmt.Errorf("package ID cannot exceed %d characters", MaxPackageIDLength)
	}

	// Must start with letter or underscore
	firstChar := rune(id[0])
	if !unicode.IsLetter(firstChar) && firstChar != '_' {
		return fmt.Errorf("package ID must start with a letter or underscore")
	}

	// Check pattern
	if !packageIDPattern.MatchString(id) {
		return fmt.Errorf("package ID contains invalid characters (only letters, digits, '.', '-', '_' allowed)")
	}

	return nil
}

// ValidateDependencies validates all dependency groups
// Reference: PackageBuilder.cs ValidateDependencies
func ValidateDependencies(packageID string, packageVersion *version.NuGetVersion, groups []PackageDependencyGroup) error {
	for _, group := range groups {
		// Check for duplicate dependencies in the same group
		seen := make(map[string]bool)
		for _, dep := range group.Dependencies {
			depKey := strings.ToLower(dep.ID)
			if seen[depKey] {
				return fmt.Errorf("duplicate dependency %q in group for %s", dep.ID, group.TargetFramework.String())
			}
			seen[depKey] = true

			// Validate dependency version range
			if err := validateDependencyVersion(dep); err != nil {
				return fmt.Errorf("invalid dependency %q: %w", dep.ID, err)
			}

			// Check for self-dependency
			if strings.EqualFold(dep.ID, packageID) {
				return fmt.Errorf("package cannot depend on itself")
			}
		}
	}

	return nil
}

// validateDependencyVersion validates a dependency version range
// Reference: Manifest.cs ValidateDependencyVersion
func validateDependencyVersion(dep PackageDependency) error {
	if dep.VersionRange == nil {
		return nil
	}

	vr := dep.VersionRange

	// If both min and max are set
	if vr.MinVersion != nil && vr.MaxVersion != nil {
		// If both exclusive and versions are equal, invalid
		if !vr.MinInclusive && !vr.MaxInclusive && vr.MinVersion.Equals(vr.MaxVersion) {
			return fmt.Errorf("version range (exclusive) cannot have equal min and max versions")
		}

		// Max must be >= Min
		if vr.MaxVersion.Compare(vr.MinVersion) < 0 {
			return fmt.Errorf("max version must be greater than or equal to min version")
		}
	}

	return nil
}

// ValidateFiles validates all files in the package
func ValidateFiles(files []PackageFile) error {
	if len(files) == 0 {
		return fmt.Errorf("package must contain at least one file")
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, file := range files {
		normalized := strings.ToLower(normalizePackagePath(file.TargetPath))
		if seen[normalized] {
			return fmt.Errorf("duplicate file path: %s", file.TargetPath)
		}
		seen[normalized] = true

		// Validate path
		if err := ValidatePackagePath(file.TargetPath); err != nil {
			return fmt.Errorf("invalid file path %q: %w", file.TargetPath, err)
		}
	}

	return nil
}

// ValidateLicense validates license metadata
// Reference: PackageBuilder.cs ValidateLicenseFile
func ValidateLicense(metadata PackageMetadata, files []PackageFile) error {
	// If RequireLicenseAcceptance is true, must have license
	if metadata.RequireLicenseAcceptance {
		if metadata.LicenseURL == nil && metadata.LicenseMetadata == nil {
			return fmt.Errorf("requireLicenseAcceptance requires either licenseUrl or license metadata")
		}
	}

	// If both licenseUrl and license metadata, they must match or one must be null
	if metadata.LicenseURL != nil && metadata.LicenseMetadata != nil {
		return fmt.Errorf("cannot specify both licenseUrl and license metadata")
	}

	// If license is a file, verify it exists
	if metadata.LicenseMetadata != nil && metadata.LicenseMetadata.Type == "file" {
		licenseFile := metadata.LicenseMetadata.Text
		if !fileExists(files, licenseFile) {
			return fmt.Errorf("license file %q specified but not found in package", licenseFile)
		}
	}

	return nil
}

func fileExists(files []PackageFile, targetPath string) bool {
	normalized := strings.ToLower(normalizePackagePath(targetPath))
	for _, file := range files {
		if strings.ToLower(normalizePackagePath(file.TargetPath)) == normalized {
			return true
		}
	}
	return false
}

// ValidateIcon validates icon file reference
// Reference: PackageBuilder.cs ValidateIconFile
func ValidateIcon(metadata PackageMetadata, files []PackageFile) error {
	if metadata.Icon == "" {
		return nil
	}

	if !fileExists(files, metadata.Icon) {
		return fmt.Errorf("icon file %q specified but not found in package", metadata.Icon)
	}

	// Icon should be in a specific folder or root
	// NuGet recommends icon/ folder or root
	normalized := strings.ToLower(metadata.Icon)
	if !strings.HasPrefix(normalized, "icon/") && strings.Contains(normalized, "/") {
		return fmt.Errorf("icon file should be in 'icon/' folder or at package root")
	}

	return nil
}

// ValidateReadme validates readme file reference
// Reference: PackageBuilder.cs ValidateReadmeFile
func ValidateReadme(metadata PackageMetadata, files []PackageFile) error {
	if metadata.Readme == "" {
		return nil
	}

	if !fileExists(files, metadata.Readme) {
		return fmt.Errorf("readme file %q specified but not found in package", metadata.Readme)
	}

	return nil
}

// ValidateFrameworkReferences validates framework reference groups
func ValidateFrameworkReferences(groups []PackageFrameworkReferenceGroup) error {
	for _, group := range groups {
		if group.TargetFramework == nil {
			return fmt.Errorf("framework reference group must have a target framework")
		}

		if len(group.References) == 0 {
			return fmt.Errorf("framework reference group for %s has no references", group.TargetFramework.String())
		}

		// Check for duplicates
		seen := make(map[string]bool)
		for _, ref := range group.References {
			refKey := strings.ToLower(ref)
			if seen[refKey] {
				return fmt.Errorf("duplicate framework reference %q in group for %s", ref, group.TargetFramework.String())
			}
			seen[refKey] = true
		}
	}

	return nil
}
