package restore

import (
	"context"
	"strings"
	"testing"
)

func TestResolveLatestVersion(t *testing.T) {
	tests := []struct {
		name        string
		packageID   string
		opts        *ResolveLatestVersionOptions
		expectError bool
		skipReason  string
	}{
		{
			name:      "resolve latest stable version",
			packageID: "Newtonsoft.Json",
			opts: &ResolveLatestVersionOptions{
				Source:     "https://api.nuget.org/v3/index.json",
				Prerelease: false,
			},
			expectError: false,
			skipReason:  "integration test - requires network",
		},
		{
			name:      "resolve latest prerelease version",
			packageID: "Newtonsoft.Json",
			opts: &ResolveLatestVersionOptions{
				Source:     "https://api.nuget.org/v3/index.json",
				Prerelease: true,
			},
			expectError: false,
			skipReason:  "integration test - requires network",
		},
		{
			name:      "use default source when not specified",
			packageID: "Newtonsoft.Json",
			opts: &ResolveLatestVersionOptions{
				Source:     "",
				Prerelease: false,
			},
			expectError: false,
			skipReason:  "integration test - requires network",
		},
		{
			name:      "error on nonexistent package",
			packageID: "NonExistentPackage123456789",
			opts: &ResolveLatestVersionOptions{
				Source:     "https://api.nuget.org/v3/index.json",
				Prerelease: false,
			},
			expectError: true,
			skipReason:  "integration test - requires network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() && tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			ctx := context.Background()
			version, err := ResolveLatestVersion(ctx, tt.packageID, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if version == "" {
					t.Errorf("expected version but got empty string")
				}
			}
		})
	}
}

func TestResolveLatestVersion_InvalidSource(t *testing.T) {
	ctx := context.Background()
	_, err := ResolveLatestVersion(ctx, "Newtonsoft.Json", &ResolveLatestVersionOptions{
		Source:     "https://invalid-source-that-does-not-exist.example.com/v3/index.json",
		Prerelease: false,
	})

	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestResolveLatestVersion_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := ResolveLatestVersion(ctx, "Newtonsoft.Json", &ResolveLatestVersionOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: false,
	})

	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestResolveLatestVersion_OnlyPrereleaseVersions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	ctx := context.Background()

	// Test requesting stable version when only prerelease exists
	// (This would require a package with only prerelease versions)
	// For now, we'll test that prerelease flag works correctly

	_, err := ResolveLatestVersion(ctx, "Newtonsoft.Json", &ResolveLatestVersionOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: true,
	})

	if err != nil {
		t.Errorf("unexpected error when requesting prerelease: %v", err)
	}
}

func TestResolveLatestVersion_StableWhenBothExist(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - requires network")
	}

	ctx := context.Background()

	// Request stable version (should return latest stable, not latest overall)
	version, err := ResolveLatestVersion(ctx, "Newtonsoft.Json", &ResolveLatestVersionOptions{
		Source:     "https://api.nuget.org/v3/index.json",
		Prerelease: false,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if version == "" {
		t.Error("expected version but got empty string")
	}

	// Verify it's not a prerelease version (no dash in version)
	if strings.Contains(version, "-") {
		t.Errorf("expected stable version but got prerelease: %s", version)
	}
}
