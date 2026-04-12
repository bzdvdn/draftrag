package draftrag

import (
	"context"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/embedder/cache"
)

// EmbedCacheStats — статистика LRU-кэша embedder'а.
type EmbedCacheStats = cache.CacheStats

// CacheOptions задаёт параметры кэширующего embedder'а.
type CacheOptions struct {
	// MaxSize — максимальное количество записей в LRU-кэше.
	// 0 → 1000.
	MaxSize int

	// Redis — опциональный Redis L2 кэш.
	Redis RedisCacheOptions

	// Logger — опциональный структурированный логгер для событий кэша.
	// nil означает no-op.
	Logger Logger
}

// RedisCacheClient — адаптер-интерфейс Redis клиента для кэша эмбеддингов.
//
// Контракт:
// - GetBytes: если ключ отсутствует, должен возвращать (nil, nil).
// - ttl == 0 означает запись без TTL.
type RedisCacheClient interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// RedisCacheOptions задаёт параметры Redis second-level cache.
type RedisCacheOptions struct {
	// Client — Redis клиент; nil отключает Redis кэш.
	Client RedisCacheClient

	// TTL — время жизни записей в Redis (0 → без TTL).
	TTL time.Duration

	// KeyPrefix — префикс ключей в Redis ("" → дефолт `draftrag:embedder:`).
	KeyPrefix string
}

// CachedEmbedder оборачивает Embedder двухуровневым LRU-кэшем.
// Повторные запросы для одного текста не идут в API.
// Реализует Embedder.
type CachedEmbedder struct {
	impl *cache.EmbedderCache
}

type redisClientAdapter struct {
	client RedisCacheClient
}

func (a *redisClientAdapter) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return a.client.GetBytes(ctx, key)
}

func (a *redisClientAdapter) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return a.client.SetBytes(ctx, key, value, ttl)
}

// NewCachedEmbedder создаёт кэширующий embedder с in-memory LRU.
//
//	embedder := draftrag.NewCachedEmbedder(
//	    draftrag.NewOpenAICompatibleEmbedder(...),
//	    draftrag.CacheOptions{MaxSize: 5000},
//	)
//	pipeline := draftrag.NewPipeline(store, llm, embedder)
func NewCachedEmbedder(e Embedder, opts CacheOptions) (*CachedEmbedder, error) {
	var cacheOpts []cache.Option
	if opts.MaxSize > 0 {
		cacheOpts = append(cacheOpts, cache.WithCacheSize(opts.MaxSize))
	}
	if opts.Logger != nil {
		cacheOpts = append(cacheOpts, cache.WithLogger(opts.Logger))
	}
	if opts.Redis.Client != nil {
		cacheOpts = append(cacheOpts, cache.WithRedis(
			&redisClientAdapter{client: opts.Redis.Client},
			opts.Redis.TTL,
			opts.Redis.KeyPrefix,
		))
	}
	impl, err := cache.NewEmbedderCache(e, cacheOpts...)
	if err != nil {
		return nil, err
	}
	return &CachedEmbedder{impl: impl}, nil
}

// Embed реализует Embedder.
func (c *CachedEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return c.impl.Embed(ctx, text)
}

// Stats возвращает текущую статистику кэша (попадания, промахи, вытеснения).
func (c *CachedEmbedder) Stats() EmbedCacheStats {
	return c.impl.Stats()
}
