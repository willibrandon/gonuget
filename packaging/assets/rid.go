package assets

import (
	"fmt"
	"strings"
)

// RuntimeIdentifier represents a parsed RID.
// Reference: NuGet.RuntimeModel/RuntimeDescription.cs
type RuntimeIdentifier struct {
	// RID is the raw RID string
	RID string

	// OS is the operating system component (e.g., "win", "linux", "osx")
	OS string

	// Version is the OS version component (e.g., "10.12" in "osx.10.12-x64")
	Version string

	// Architecture is the architecture component (e.g., "x64", "arm64")
	Architecture string

	// Qualifiers are additional qualifiers beyond architecture
	Qualifiers []string
}

// ParseRID parses a runtime identifier string.
// Format: <os>.<version>-<architecture>-<qualifiers>
// Examples: "win10-x64", "linux-x64", "osx.10.12-x64"
func ParseRID(rid string) (*RuntimeIdentifier, error) {
	if rid == "" {
		return nil, fmt.Errorf("RID cannot be empty")
	}

	r := &RuntimeIdentifier{
		RID: rid,
	}

	// Split by hyphen
	parts := strings.Split(rid, "-")

	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid RID format")
	}

	// First part is OS (potentially with version)
	osPart := parts[0]
	if strings.Contains(osPart, ".") {
		// OS with version: "osx.10.12"
		osParts := strings.SplitN(osPart, ".", 2)
		r.OS = osParts[0]
		r.Version = osParts[1]
	} else {
		r.OS = osPart
	}

	// Remaining parts are architecture and qualifiers
	if len(parts) > 1 {
		r.Architecture = parts[1]
	}

	if len(parts) > 2 {
		r.Qualifiers = parts[2:]
	}

	return r, nil
}

// String returns the RID string.
func (r *RuntimeIdentifier) String() string {
	return r.RID
}

// IsCompatible checks if this RID is compatible with another RID using the graph.
func (r *RuntimeIdentifier) IsCompatible(other *RuntimeIdentifier, graph *RuntimeGraph) bool {
	// Exact match
	if r.RID == other.RID {
		return true
	}

	// Check graph for compatibility
	if graph != nil {
		return graph.AreCompatible(r.RID, other.RID)
	}

	// Fallback: OS and architecture must match
	return r.OS == other.OS && r.Architecture == other.Architecture
}
