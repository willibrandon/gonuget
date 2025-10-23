package cache

import (
	"time"

	"github.com/google/uuid"
)

// SourceCacheContext provides cache control settings matching NuGet.Client behavior.
type SourceCacheContext struct {
	// MaxAge is the maximum age for cached entries (default: 30 minutes)
	MaxAge time.Duration

	// NoCache bypasses the global disk cache if true
	NoCache bool

	// DirectDownload skips cache writes (read-only mode)
	DirectDownload bool

	// RefreshMemoryCache forces in-memory cache reload
	RefreshMemoryCache bool

	// SessionID is a unique identifier for the session (X-NuGet-Session-Id header)
	SessionID string
}

// NewSourceCacheContext creates a new cache context with defaults.
func NewSourceCacheContext() *SourceCacheContext {
	return &SourceCacheContext{
		MaxAge:    30 * time.Minute, // Default from NuGet.Client
		SessionID: uuid.New().String(),
	}
}

// Clone creates a copy of the cache context.
func (ctx *SourceCacheContext) Clone() *SourceCacheContext {
	return &SourceCacheContext{
		MaxAge:             ctx.MaxAge,
		NoCache:            ctx.NoCache,
		DirectDownload:     ctx.DirectDownload,
		RefreshMemoryCache: ctx.RefreshMemoryCache,
		SessionID:          ctx.SessionID,
	}
}
