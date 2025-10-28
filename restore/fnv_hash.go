package restore

import (
	"encoding/base64"
	"encoding/binary"
)

// FnvHash64 implements the FNV-1a 64-bit hash algorithm used by NuGet.Client.
// Reference: NuGet.ProjectModel/FnvHash64Function.cs
type FnvHash64 struct {
	hash uint64
}

const (
	// FNV-1a 64-bit constants
	fnvOffset uint64 = 14695981039346656037
	fnvPrime  uint64 = 1099511628211
)

// NewFnvHash64 creates a new FNV-1a 64-bit hash.
func NewFnvHash64() *FnvHash64 {
	return &FnvHash64{
		hash: fnvOffset,
	}
}

// Update feeds data into the hash.
// Matches FnvHash64Function.Update() in NuGet.Client.
func (f *FnvHash64) Update(data []byte) {
	// Update hash with each byte: hash = (hash ^ byte) * Prime
	for _, b := range data {
		f.hash = (f.hash ^ uint64(b)) * fnvPrime
	}
}

// GetHash returns the base64-encoded hash string.
// Matches FnvHash64Function.GetHash() in NuGet.Client.
func (f *FnvHash64) GetHash() string {
	// Convert hash to bytes (little-endian, matching BitConverter.GetBytes on Windows)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, f.hash)

	// Return base64-encoded hash
	return base64.StdEncoding.EncodeToString(bytes)
}

// Hash computes FNV-1a hash of data in one call.
func Hash(data []byte) uint64 {
	hash := fnvOffset
	for _, b := range data {
		hash = (hash ^ uint64(b)) * fnvPrime
	}
	return hash
}
