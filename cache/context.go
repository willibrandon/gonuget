package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const cacheContextKey contextKey = "nuget.cache.context"

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

// WithCacheContext adds the source cache context to the Go context.
// This allows protocol layer code to respect cache control flags without
// passing SourceCacheContext through every function.
func WithCacheContext(ctx context.Context, cacheCtx *SourceCacheContext) context.Context {
	if cacheCtx == nil {
		return ctx
	}
	return context.WithValue(ctx, cacheContextKey, cacheCtx)
}

// FromContext retrieves the source cache context from the Go context.
// Returns nil if no cache context was set.
func FromContext(ctx context.Context) *SourceCacheContext {
	if ctx == nil {
		return nil
	}
	if cacheCtx, ok := ctx.Value(cacheContextKey).(*SourceCacheContext); ok {
		return cacheCtx
	}
	return nil
}
