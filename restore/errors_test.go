package restore

import (
	"strings"
	"testing"
)

func TestNuGetError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NuGetError
		contains []string // Strings that should be in the error message
	}{
		{
			name: "NU1101 basic",
			err: &NuGetError{
				Code:    "NU1101",
				Message: "Unable to find package 'NonExistent'.",
				Sources: []string{"https://api.nuget.org/v3/index.json"},
			},
			contains: []string{"NU1101", "NonExistent", "https://api.nuget.org"},
		},
		{
			name: "NU1102 with package and constraint",
			err: &NuGetError{
				Code:       "NU1102",
				Message:    "Unable to find package 'TestPackage' with version (>= 2.0.0)",
				Sources:    []string{"https://api.nuget.org/v3/index.json"},
				PackageID:  "TestPackage",
				Constraint: ">= 2.0.0",
			},
			contains: []string{"NU1102", "TestPackage", ">= 2.0.0"},
		},
		{
			name: "Error with multiple sources",
			err: &NuGetError{
				Code:    "NU1101",
				Message: "Package not found",
				Sources: []string{"https://source1.com", "https://source2.com"},
			},
			contains: []string{"NU1101", "source1", "source2"},
		},
		{
			name: "NU1103 error",
			err: &NuGetError{
				Code:    "NU1103",
				Message: "Package dependency could not be resolved",
			},
			contains: []string{"NU1103"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, substr := range tt.contains {
				if !strings.Contains(errMsg, substr) {
					t.Errorf("Error message should contain %q, got: %s", substr, errMsg)
				}
			}
		})
	}
}

func TestNuGetError_FormatError(t *testing.T) {
	err := &NuGetError{
		Code:    "NU1101",
		Message: "Test error",
		Sources: []string{"https://test.com"},
	}

	// Test colorized output
	colorized := err.FormatError(true)
	if !strings.Contains(colorized, "NU1101") {
		t.Errorf("Colorized error should contain error code")
	}

	// Test non-colorized output
	plain := err.FormatError(false)
	if !strings.Contains(plain, "NU1101") {
		t.Errorf("Plain error should contain error code")
	}

	// Both should contain the core message
	for _, msg := range []string{colorized, plain} {
		if !strings.Contains(msg, "Test error") {
			t.Errorf("Error message should contain 'Test error', got: %s", msg)
		}
	}
}

func TestFormatVersionConstraintForDisplay(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		want       string
	}{
		{
			name:       "exact version",
			constraint: "[1.0.0]",
			want:       "= 1.0.0",
		},
		{
			name:       "minimum version",
			constraint: "1.0.0",
			want:       ">= 1.0.0",
		},
		{
			name:       "version range",
			constraint: "[1.0.0, 2.0.0)",
			want:       ">= 1.0.0 && < 2.0.0",
		},
		{
			name:       "complex range",
			constraint: "(1.0.0, 2.0.0]",
			want:       "> 1.0.0 && <= 2.0.0",
		},
		{
			name:       "open-ended upper inclusive",
			constraint: "[1.0.0,)",
			want:       ">= 1.0.0",
		},
		{
			name:       "open-ended upper exclusive",
			constraint: "(1.0.0,)",
			want:       "> 1.0.0",
		},
		{
			name:       "open-ended lower inclusive",
			constraint: "(,2.0.0]",
			want:       "<= 2.0.0",
		},
		{
			name:       "open-ended lower exclusive",
			constraint: "(,2.0.0)",
			want:       "< 2.0.0",
		},
		{
			name:       "malformed single bracket",
			constraint: "(malformed",
			want:       "(malformed", // Fallback to original
		},
		{
			name:       "whitespace handling",
			constraint: "  [1.0.0, 2.0.0]  ",
			want:       ">= 1.0.0 && <= 2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersionConstraintForDisplay(tt.constraint)
			if got != tt.want {
				t.Errorf("formatVersionConstraintForDisplay(%q) = %q, want %q", tt.constraint, got, tt.want)
			}
		})
	}
}

func TestNewPackageNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		packageID   string
		version     string
		sources     []string
		wantCode    []string // Acceptable error codes
	}{
		{
			name:        "basic package not found",
			projectPath: "/tmp/test.csproj",
			packageID:   "TestPackage",
			version:     "1.0.0",
			sources:     []string{"https://api.nuget.org/v3/index.json"},
			wantCode:    []string{"NU1101", "NU1102"},
		},
		{
			name:        "multiple sources",
			projectPath: "/tmp/test.csproj",
			packageID:   "AnotherPackage",
			version:     "2.0.0",
			sources:     []string{"https://api.nuget.org/v3/index.json", "https://www.nuget.org/api/v2"},
			wantCode:    []string{"NU1101", "NU1102"},
		},
		{
			name:        "empty sources",
			projectPath: "/tmp/test.csproj",
			packageID:   "SomePackage",
			version:     "1.5.0",
			sources:     []string{},
			wantCode:    []string{"NU1101", "NU1102"},
		},
		{
			name:        "nuget.org source",
			projectPath: "/home/user/project.csproj",
			packageID:   "Package123",
			version:     "3.0.0",
			sources:     []string{"https://www.nuget.org/api/v2/"},
			wantCode:    []string{"NU1101", "NU1102"},
		},
		{
			name:        "v2 source",
			projectPath: "/tmp/test.csproj",
			packageID:   "TestPkg",
			version:     "1.0.0",
			sources:     []string{"https://example.com/nuget/v2"},
			wantCode:    []string{"NU1101", "NU1102"},
		},
		{
			name:        "custom source",
			projectPath: "/tmp/test.csproj",
			packageID:   "CustomPkg",
			version:     "2.0.0",
			sources:     []string{"https://mycompany.pkgs.visualstudio.com/_packaging/feed/nuget/v3/index.json"},
			wantCode:    []string{"NU1101", "NU1102"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewPackageNotFoundError(tt.projectPath, tt.packageID, tt.version, tt.sources)

			// Check if error code is one of the expected codes
			found := false
			for _, code := range tt.wantCode {
				if err.Code == code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected error code to be one of %v, got %s", tt.wantCode, err.Code)
			}

			if err.ProjectPath != tt.projectPath {
				t.Errorf("Expected project path %s, got %s", tt.projectPath, err.ProjectPath)
			}

			errMsg := err.Error()
			if !strings.Contains(errMsg, tt.packageID) {
				t.Errorf("Error message should contain package ID %s, got: %s", tt.packageID, errMsg)
			}
		})
	}
}
