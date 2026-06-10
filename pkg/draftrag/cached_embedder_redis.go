package draftrag

import (
	"context"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/embedder/cache"
)

type RedisClient = cache.RedisClient

// NewRedisCache creates a CachedEmbedder backed by Redis for embedding cache.
//
// @sk-task hardening-2026q2#T2.1: Публичный wrapper Redis cache с type-alias (AC-005)
func NewRedisCache(ctx context.Context, e Embedder, client RedisClient, ttl time.Duration) (*CachedEmbedder, error) {
	_ = ctx
	return NewCachedEmbedder(e, CacheOptions{
		Redis: RedisCacheOptions{
			Client: client,
			TTL:    ttl,
		},
	})
}
