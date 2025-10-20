package version

import "fmt"

// Normalize parses a version string and returns its normalized form.
//
// Normalization converts versions to their canonical string representation,
// removing leading zeros and applying consistent formatting.
//
// Examples:
//   - "1.01.1" → "1.1.1"
//   - "1" → "1.0.0"
//   - "1.2" → "1.2.0"
//   - "1.0.0.0" → "1.0.0.0" (legacy preserved)
func Normalize(s string) (string, error) {
	v, err := Parse(s)
	if err != nil {
		return "", fmt.Errorf("cannot normalize invalid version: %w", err)
	}
	return v.ToNormalizedString(), nil
}

// MustNormalize normalizes a version string, panicking on error.
func MustNormalize(s string) string {
	normalized, err := Normalize(s)
	if err != nil {
		panic(err)
	}
	return normalized
}

// NormalizeOrOriginal attempts to normalize a version string.
// If normalization fails, returns the original string.
func NormalizeOrOriginal(s string) string {
	normalized, err := Normalize(s)
	if err != nil {
		return s
	}
	return normalized
}
