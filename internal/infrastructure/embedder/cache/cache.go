package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// EmbedderCache обёртка над Embedder с LRU in-memory кэшем и опциональным Redis.
// Реализует интерфейс domain.Embedder.
// @ds-task T1.1: Основная структура кэша (AC-001, RQ-001)
type EmbedderCache struct {
	embedder  domain.Embedder
	cache     *lruCache
	redis     *redisCache
	cacheSize int
	stats     statsCollector
}

// NewEmbedderCache создаёт новый кэширующий embedder.
// @ds-task T1.1, T1.4, T2.4: Конструктор с валидацией (AC-001, RQ-002)
func NewEmbedderCache(embedder domain.Embedder, opts ...Option) (*EmbedderCache, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}

	c := &EmbedderCache{
		embedder:  embedder,
		cacheSize: 1000, // default size
	}

	// Применяем опции
	for _, opt := range opts {
		opt(c)
	}

	// Инициализируем LRU кэш
	c.cache = newLRUCache(c.cacheSize, &c.stats)

	return c, nil
}

// Embed преобразует текст в векторное представление с использованием кэша.
// Сначала проверяет in-memory LRU (L1), затем Redis (L2, если настроен),
// при miss вызывает базовый embedder и сохраняет результат в кэш.
// @ds-task T1.1, T2.1, T3.2: Двухуровневый lookup с fallback (AC-001, RQ-003, RQ-005, RQ-009)
func (c *EmbedderCache) Embed(ctx context.Context, text string) ([]float64, error) {
	// Вычисляем хэш ключа
	key := c.hashKey(text)

	// L1: Проверяем in-memory LRU кэш
	if value, ok := c.cache.Get(key); ok {
		c.stats.RecordHit()
		return value, nil
	}

	// L2: Проверяем Redis (если настроен)
	if c.redis != nil && c.redis.client != nil {
		value, ok, err := c.redis.Get(ctx, key)
		if err == nil && ok {
			// Попадание в Redis — сохраняем в L1 и возвращаем
			c.cache.Set(key, value)
			c.stats.RecordHit()
			return value, nil
		}
		if err != nil {
			// treat-as-miss: любая ошибка Redis/декодирования не должна ломать Embed
			log.Printf("[EmbedderCache] Redis read failed (treat-as-miss), falling back to embedder: %v", err)
		}
	}

	// Miss: вызываем базовый embedder
	value, err := c.embedder.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	// Сохраняем в L1
	c.cache.Set(key, value)

	// Сохраняем в L2 (Redis), если настроен и нет ошибок
	if c.redis != nil && c.redis.client != nil {
		if err := c.redis.Set(ctx, key, value); err != nil {
			// treat-as-miss: запись best-effort
			log.Printf("[EmbedderCache] Redis write failed (best-effort): %v", err)
		}
	}

	c.stats.RecordMiss()
	return value, nil
}

// hashKey вычисляет SHA-256 хэш текста для ключа кэша.
// @ds-task T1.1: Хэширование ключа SHA-256 (DEC-003, AC-006)
func (c *EmbedderCache) hashKey(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// Stats возвращает текущие метрики кэша.
// @ds-task T2.5: Метод получения статистики (AC-007)
func (c *EmbedderCache) Stats() CacheStats {
	return c.stats.Stats()
}
