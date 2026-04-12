package cache

import (
	"context"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

const defaultRedisKeyPrefix = "draftrag:embedder:"

// Option функциональная опция для конфигурации EmbedderCache.
// @ds-task T1.4: Тип функциональной опции (AC-001)
type Option func(*EmbedderCache)

// WithCacheSize устанавливает размер in-memory LRU кэша.
// Минимальное значение — 1. По умолчанию 1000.
// @ds-task T1.4, T2.4: Опция размера кэша с валидацией (AC-001, RQ-004)
func WithCacheSize(size int) Option {
	return func(c *EmbedderCache) {
		if size < 1 {
			size = 1
		}
		c.cacheSize = size
	}
}

// WithLogger настраивает опциональный структурированный логгер.
// nil означает no-op.
func WithLogger(logger domain.Logger) Option {
	return func(c *EmbedderCache) {
		c.logger = logger
	}
}

// RedisClient — минимальный адаптер-интерфейс для Redis.
//
// Контракт:
// - GetBytes: если ключ отсутствует, должен возвращать (nil, nil).
// - Любая ошибка должна означать проблему с доступом/исполнением операции.
//
// @ds-task T1.1: Адаптер-интерфейс Redis клиента (RQ-002.1)
type RedisClient interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// WithRedis настраивает Redis как second-level cache (L2).
// client может быть nil — в этом случае Redis не используется.
// ttl задаёт время жизни записей в Redis (0 — без TTL).
// keyPrefix задаёт префикс keyspace ("" → дефолт `draftrag:embedder:`).
// @ds-task T3.2: Опция для Redis second-level cache (AC-004, AC-005)
func WithRedis(client RedisClient, ttl time.Duration, keyPrefix string) Option {
	return func(c *EmbedderCache) {
		c.redis = &redisCache{
			client: client,
			ttl:    ttl,
			prefix: keyPrefix,
		}
	}
}
