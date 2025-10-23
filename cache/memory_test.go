package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Set a value
	key := "test-key"
	value := []byte("test-value")
	mc.Set(key, value, 1*time.Hour)

	// Get the value
	got, ok := mc.Get(key)
	if !ok {
		t.Fatal("expected key to be found")
	}
	if string(got) != string(value) {
		t.Errorf("got %s, want %s", got, value)
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Set with short TTL
	key := "expiring-key"
	value := []byte("expiring-value")
	mc.Set(key, value, 50*time.Millisecond)

	// Should exist immediately
	_, ok := mc.Get(key)
	if !ok {
		t.Fatal("expected key to exist")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = mc.Get(key)
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	mc := NewMemoryCache(3, 1024*1024) // Max 3 entries

	// Add 3 entries
	mc.Set("key1", []byte("value1"), 1*time.Hour)
	mc.Set("key2", []byte("value2"), 1*time.Hour)
	mc.Set("key3", []byte("value3"), 1*time.Hour)

	// Access key1 to make it recently used
	mc.Get("key1")

	// Add 4th entry, should evict key2 (least recently used)
	mc.Set("key4", []byte("value4"), 1*time.Hour)

	// key2 should be evicted
	_, ok := mc.Get("key2")
	if ok {
		t.Fatal("expected key2 to be evicted")
	}

	// key1, key3, key4 should exist
	if _, ok := mc.Get("key1"); !ok {
		t.Fatal("expected key1 to exist")
	}
	if _, ok := mc.Get("key3"); !ok {
		t.Fatal("expected key3 to exist")
	}
	if _, ok := mc.Get("key4"); !ok {
		t.Fatal("expected key4 to exist")
	}
}

func TestMemoryCache_SizeEviction(t *testing.T) {
	mc := NewMemoryCache(100, 100) // Max 100 bytes

	// Add entry that's 60 bytes
	mc.Set("key1", make([]byte, 60), 1*time.Hour)

	// Add entry that's 50 bytes (total 110, exceeds limit)
	mc.Set("key2", make([]byte, 50), 1*time.Hour)

	// key1 should be evicted
	_, ok := mc.Get("key1")
	if ok {
		t.Fatal("expected key1 to be evicted")
	}

	// key2 should exist
	_, ok = mc.Get("key2")
	if !ok {
		t.Fatal("expected key2 to exist")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Add entries
	mc.Set("key1", []byte("value1"), 1*time.Hour)
	mc.Set("key2", []byte("value2"), 1*time.Hour)

	// Clear cache
	mc.Clear()

	// All entries should be gone
	_, ok := mc.Get("key1")
	if ok {
		t.Fatal("expected key1 to be cleared")
	}
	_, ok = mc.Get("key2")
	if ok {
		t.Fatal("expected key2 to be cleared")
	}

	// Stats should be zero
	stats := mc.Stats()
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.Entries)
	}
	if stats.SizeBytes != 0 {
		t.Errorf("expected 0 bytes, got %d", stats.SizeBytes)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Add entries
	mc.Set("key1", []byte("value1"), 1*time.Hour)
	mc.Set("key2", []byte("value2"), 1*time.Hour)

	// Delete key1
	mc.Delete("key1")

	// key1 should be gone
	_, ok := mc.Get("key1")
	if ok {
		t.Fatal("expected key1 to be deleted")
	}

	// key2 should still exist
	_, ok = mc.Get("key2")
	if !ok {
		t.Fatal("expected key2 to exist")
	}

	// Delete non-existent key (should not panic)
	mc.Delete("nonexistent")
}

func TestMemoryCache_Update(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Set initial value
	mc.Set("key1", []byte("value1"), 1*time.Hour)

	// Update with new value
	mc.Set("key1", []byte("updated-value"), 1*time.Hour)

	// Should get updated value
	got, ok := mc.Get("key1")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if string(got) != "updated-value" {
		t.Errorf("got %s, want updated-value", got)
	}

	// Update with different size
	mc.Set("key1", []byte("x"), 1*time.Hour)
	got, ok = mc.Get("key1")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if string(got) != "x" {
		t.Errorf("got %s, want x", got)
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	mc := NewMemoryCache(100, 1024*1024)

	// Initially empty
	stats := mc.Stats()
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.Entries)
	}
	if stats.SizeBytes != 0 {
		t.Errorf("expected 0 bytes, got %d", stats.SizeBytes)
	}

	// Add entries
	mc.Set("key1", []byte("12345"), 1*time.Hour) // 5 bytes
	mc.Set("key2", []byte("678"), 1*time.Hour)   // 3 bytes

	stats = mc.Stats()
	if stats.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.Entries)
	}
	if stats.SizeBytes != 8 {
		t.Errorf("expected 8 bytes, got %d", stats.SizeBytes)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	mc := NewMemoryCache(10000, 10*1024*1024)
	mc.Set("benchmark-key", []byte("benchmark-value"), 1*time.Hour)

	b.ResetTimer()
	for b.Loop() {
		mc.Get("benchmark-key")
	}
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	mc := NewMemoryCache(10000, 10*1024*1024)
	value := []byte("benchmark-value")

	b.ResetTimer()
	for b.Loop() {
		mc.Set("benchmark-key", value, 1*time.Hour)
	}
}
