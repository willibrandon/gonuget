package cache

import (
	"testing"
	"time"
)

func TestNewSourceCacheContext(t *testing.T) {
	ctx := NewSourceCacheContext()

	if ctx.MaxAge != 30*time.Minute {
		t.Errorf("MaxAge = %v, want 30m", ctx.MaxAge)
	}

	if ctx.SessionID == "" {
		t.Error("SessionID should be set")
	}

	if ctx.NoCache || ctx.DirectDownload || ctx.RefreshMemoryCache {
		t.Error("Flags should be false by default")
	}
}

func TestSourceCacheContext_Clone(t *testing.T) {
	original := &SourceCacheContext{
		MaxAge:             1 * time.Hour,
		NoCache:            true,
		DirectDownload:     true,
		RefreshMemoryCache: true,
		SessionID:          "test-session",
	}

	clone := original.Clone()

	if clone.MaxAge != original.MaxAge {
		t.Errorf("MaxAge not cloned correctly")
	}
	if clone.NoCache != original.NoCache {
		t.Errorf("NoCache not cloned correctly")
	}
	if clone.DirectDownload != original.DirectDownload {
		t.Errorf("DirectDownload not cloned correctly")
	}
	if clone.RefreshMemoryCache != original.RefreshMemoryCache {
		t.Errorf("RefreshMemoryCache not cloned correctly")
	}
	if clone.SessionID != original.SessionID {
		t.Errorf("SessionID not cloned correctly")
	}

	// Verify it's a copy, not same reference
	clone.MaxAge = 2 * time.Hour
	if original.MaxAge == clone.MaxAge {
		t.Error("Clone should be independent copy")
	}
}

func TestSourceCacheContext_SessionIDUniqueness(t *testing.T) {
	ctx1 := NewSourceCacheContext()
	ctx2 := NewSourceCacheContext()

	if ctx1.SessionID == ctx2.SessionID {
		t.Error("NewSourceCacheContext should generate unique SessionIDs")
	}

	if ctx1.SessionID == "" || ctx2.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
}
