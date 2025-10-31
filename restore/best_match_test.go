package restore

import (
	"testing"

	"github.com/willibrandon/gonuget/version"
)

// TestGetBestMatch verifies that getBestMatch matches NuGet.Client's GetBestMatch algorithm.
// Test cases are based on examples from UnresolvedMessages.cs in NuGet.Client.
func TestGetBestMatch(t *testing.T) {
	tests := []struct {
		name            string
		versionRange    string
		available       []string
		expectedNearest string
	}{
		{
			name:            "Range [1.0.0, ) with versions below lower bound",
			versionRange:    "[1.0.0,)",
			available:       []string{"0.7.0", "0.9.0"},
			expectedNearest: "0.9.0", // Highest available (below bound)
		},
		{
			name:            "Range (0.5.0, 1.0.0) with versions at bounds",
			versionRange:    "(0.5.0, 1.0.0)",
			available:       []string{"0.1.0", "1.0.0"},
			expectedNearest: "1.0.0", // Closest to upper bound
		},
		{
			name:            "Range (, 1.0.0) with versions above upper bound",
			versionRange:    "(, 1.0.0)",
			available:       []string{"2.0.0", "3.0.0"},
			expectedNearest: "2.0.0", // First above upper bound
		},
		{
			name:            "Range [2.0.0, ) with exact match available",
			versionRange:    "[2.0.0,)",
			available:       []string{"1.0.0", "2.0.0", "3.0.0"},
			expectedNearest: "2.0.0", // Exact match at lower bound
		},
		{
			name:            "Range [1.0.0, 2.0.0] with versions inside and outside",
			versionRange:    "[1.0.0, 2.0.0]",
			available:       []string{"0.5.0", "1.5.0", "2.5.0"},
			expectedNearest: "1.5.0", // First version above MinVersion (pivot)
		},
		{
			name:            "Simple version 1.0.0 (implicit [1.0.0,))",
			versionRange:    "1.0.0",
			available:       []string{"0.5.0", "0.9.0", "1.5.0"},
			expectedNearest: "1.5.0", // First above lower bound
		},
		{
			name:            "Range with only lower bound",
			versionRange:    "[1.0.0,)",
			available:       []string{"1.0.0", "1.5.0", "2.0.0"},
			expectedNearest: "1.0.0", // Exact match at lower bound
		},
		{
			name:            "Range with only upper bound",
			versionRange:    "(,2.0.0]",
			available:       []string{"1.0.0", "3.0.0", "4.0.0"},
			expectedNearest: "3.0.0", // First above upper bound
		},
		{
			name:            "Empty available versions",
			versionRange:    "[1.0.0,)",
			available:       []string{},
			expectedNearest: "", // No versions available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse version range
			vr, err := version.ParseVersionRange(tt.versionRange)
			if err != nil {
				t.Fatalf("Failed to parse version range %q: %v", tt.versionRange, err)
			}

			// Call getBestMatch
			nearest := getBestMatch(tt.available, vr)

			// Verify result
			if nearest != tt.expectedNearest {
				t.Errorf("getBestMatch() = %q, want %q\nRange: %s\nAvailable: %v",
					nearest, tt.expectedNearest, tt.versionRange, tt.available)
			}
		})
	}
}

// TestGetBestMatch_NilRange tests getBestMatch with nil range (should return highest version)
func TestGetBestMatch_NilRange(t *testing.T) {
	available := []string{"1.0.0", "2.0.0", "3.0.0"}
	nearest := getBestMatch(available, nil)

	expected := "3.0.0"
	if nearest != expected {
		t.Errorf("getBestMatch(nil range) = %q, want %q", nearest, expected)
	}
}
