package restore

import (
	"testing"
)

func TestFnvHash64_BasicFunctionality(t *testing.T) {
	// Test empty string - hash should be the initial offset
	h1 := NewFnvHash64()
	h1.Update([]byte(""))
	if h1.hash != fnvOffset {
		t.Errorf("Empty string hash = %d, want %d (fnvOffset)", h1.hash, fnvOffset)
	}

	// Test single byte - verify GetHash returns valid base64
	h2 := NewFnvHash64()
	h2.Update([]byte("a"))
	hash := h2.GetHash()
	if len(hash) == 0 {
		t.Error("FnvHash64.GetHash() returned empty string")
	}

	// Test that hash changes with different input
	h3 := NewFnvHash64()
	h3.Update([]byte("b"))
	hash3 := h3.GetHash()
	if hash == hash3 {
		t.Error("Hash of 'a' should differ from hash of 'b'")
	}
}

func TestFnvHash64_IncrementalUpdate(t *testing.T) {
	// Test that multiple Update calls produce same result as single call
	data1 := []byte("hello ")
	data2 := []byte("world")
	combined := []byte("hello world")

	// Incremental update
	h1 := NewFnvHash64()
	h1.Update(data1)
	h1.Update(data2)
	hash1 := h1.GetHash()

	// Single update
	h2 := NewFnvHash64()
	h2.Update(combined)
	hash2 := h2.GetHash()

	if hash1 != hash2 {
		t.Errorf("Incremental hash %s != single hash %s", hash1, hash2)
	}
}

func TestFnvHash64_Constants(t *testing.T) {
	// Verify constants match NuGet.Client
	if fnvOffset != 14695981039346656037 {
		t.Errorf("fnvOffset = %d, want 14695981039346656037", fnvOffset)
	}
	if fnvPrime != 1099511628211 {
		t.Errorf("fnvPrime = %d, want 1099511628211", fnvPrime)
	}
}

func TestHash_Function(t *testing.T) {
	// Test standalone Hash function
	data := []byte("test data")
	hash1 := Hash(data)

	// Verify it matches the incremental approach
	h := NewFnvHash64()
	h.Update(data)
	hash2 := h.hash

	if hash1 != hash2 {
		t.Errorf("Hash() = %d, NewFnvHash64().Update() = %d", hash1, hash2)
	}
}
