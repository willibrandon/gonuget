package packaging

import (
	"testing"

	"github.com/willibrandon/gonuget/packaging/signatures"
)

// TestGetPrimarySignature tests reading primary signature from signed packages
func TestGetPrimarySignature(t *testing.T) {
	tests := []struct {
		name         string
		packagePath  string
		wantSigned   bool
		wantErr      bool
		expectedType signatures.SignatureType
	}{
		{
			name:         "Author signed package",
			packagePath:  "testdata/TestPackage.AuthorSigned.1.0.0.nupkg",
			wantSigned:   true,
			wantErr:      false,
			expectedType: signatures.SignatureTypeAuthor,
		},
		{
			name:        "Unsigned package",
			packagePath: "testdata/TestUpdatePackage.1.0.1.nupkg",
			wantSigned:  false,
			wantErr:     true, // ErrPackageNotSigned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := OpenPackage(tt.packagePath)
			if err != nil {
				t.Skipf("Test package not found: %s", tt.packagePath)
			}
			defer func() { _ = pkg.Close() }()

			sig, err := pkg.GetPrimarySignature()

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrimarySignature() expected error, got nil")
				}
				if err != ErrPackageNotSigned {
					t.Errorf("GetPrimarySignature() expected ErrPackageNotSigned, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetPrimarySignature() error = %v", err)
			}

			if sig == nil {
				t.Fatal("GetPrimarySignature() returned nil signature")
			}

			if sig.Type != tt.expectedType {
				t.Errorf("GetPrimarySignature() type = %v, want %v", sig.Type, tt.expectedType)
			}

			t.Logf("Primary signature type: %v", sig.Type)
		})
	}
}

// TestIsAuthorSigned tests checking if package has author signature
func TestIsAuthorSigned(t *testing.T) {
	tests := []struct {
		name        string
		packagePath string
		wantSigned  bool
		wantErr     bool
	}{
		{
			name:        "Author signed package",
			packagePath: "testdata/TestPackage.AuthorSigned.1.0.0.nupkg",
			wantSigned:  true,
			wantErr:     false,
		},
		{
			name:        "Unsigned package",
			packagePath: "testdata/TestUpdatePackage.1.0.1.nupkg",
			wantSigned:  false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := OpenPackage(tt.packagePath)
			if err != nil {
				t.Skipf("Test package not found: %s", tt.packagePath)
			}
			defer func() { _ = pkg.Close() }()

			isSigned, err := pkg.IsAuthorSigned()

			if tt.wantErr {
				if err == nil {
					t.Errorf("IsAuthorSigned() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("IsAuthorSigned() error = %v", err)
			}

			if isSigned != tt.wantSigned {
				t.Errorf("IsAuthorSigned() = %v, want %v", isSigned, tt.wantSigned)
			}

			t.Logf("Is author signed: %v", isSigned)
		})
	}
}

// TestIsRepositorySigned tests checking if package has repository signature
func TestIsRepositorySigned(t *testing.T) {
	tests := []struct {
		name        string
		packagePath string
		wantSigned  bool
		wantErr     bool
	}{
		{
			name:        "Author signed package (not repo signed)",
			packagePath: "testdata/TestPackage.AuthorSigned.1.0.0.nupkg",
			wantSigned:  false,
			wantErr:     false,
		},
		{
			name:        "Unsigned package",
			packagePath: "testdata/TestUpdatePackage.1.0.1.nupkg",
			wantSigned:  false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := OpenPackage(tt.packagePath)
			if err != nil {
				t.Skipf("Test package not found: %s", tt.packagePath)
			}
			defer func() { _ = pkg.Close() }()

			isSigned, err := pkg.IsRepositorySigned()

			if tt.wantErr {
				if err == nil {
					t.Errorf("IsRepositorySigned() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("IsRepositorySigned() error = %v", err)
			}

			if isSigned != tt.wantSigned {
				t.Errorf("IsRepositorySigned() = %v, want %v", isSigned, tt.wantSigned)
			}

			t.Logf("Is repository signed: %v", isSigned)
		})
	}
}
