package packaging

import "errors"

var (
	// ErrPackageNotSigned indicates the package does not contain a signature
	ErrPackageNotSigned = errors.New("package is not signed")

	// ErrInvalidPackage indicates the package structure is invalid
	ErrInvalidPackage = errors.New("invalid package structure")

	// ErrNuspecNotFound indicates no .nuspec file was found
	ErrNuspecNotFound = errors.New("nuspec file not found")

	// ErrMultipleNuspecs indicates multiple .nuspec files were found
	ErrMultipleNuspecs = errors.New("multiple nuspec files found")

	// ErrInvalidPath indicates an invalid file path (e.g., path traversal)
	ErrInvalidPath = errors.New("invalid file path")
)
