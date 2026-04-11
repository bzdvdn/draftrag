package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLRUCacheBasic операции Get/Set/Len.
// @sk-task T2.6: Базовые операции LRU (DEC-001)
func TestLRUCacheBasic(t *testing.T) {
	stats := &statsCollector{}
	cache := newLRUCache(3, stats)

	// Set и Get
	cache.Set("key1", []float64{1.0, 2.0})
	val, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, []float64{1.0, 2.0}, val)
	assert.Equal(t, 1, cache.Len())

	// Обновление существующего ключа
	cache.Set("key1", []float64{3.0, 4.0})
	val, ok = cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, []float64{3.0, 4.0}, val)
	assert.Equal(t, 1, cache.Len()) // размер не изменился
}

// TestLRUCacheMiss проверяет промах.
// @sk-task T2.6: Промах в LRU (DEC-001)
func TestLRUCacheMiss(t *testing.T) {
	stats := &statsCollector{}
	cache := newLRUCache(3, stats)

	val, ok := cache.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

// TestLRUCacheEviction проверяет вытеснение.
// @sk-task T2.6: Вытеснение LRU (AC-002)
func TestLRUCacheEviction(t *testing.T) {
	stats := &statsCollector{}
	cache := newLRUCache(2, stats)

	// Добавляем 2 элемента
	cache.Set("a", []float64{1.0})
	cache.Set("b", []float64{2.0})
	assert.Equal(t, 2, cache.Len())

	// Добавляем 3-й — должен вытеснить "a" (LRU)
	cache.Set("c", []float64{3.0})
	assert.Equal(t, 2, cache.Len())

	// "a" должен быть вытеснен
	_, ok := cache.Get("a")
	assert.False(t, ok)

	// "b" и "c" должны быть в кэше
	_, ok = cache.Get("b")
	assert.True(t, ok)
	_, ok = cache.Get("c")
	assert.True(t, ok)
}

// TestLRUCachePromotion проверяет promotion при доступе.
// @sk-task T2.6: Promotion LRU (DEC-001)
func TestLRUCachePromotion(t *testing.T) {
	stats := &statsCollector{}
	cache := newLRUCache(2, stats)

	// Добавляем a, b
	cache.Set("a", []float64{1.0}) // order: [a]
	cache.Set("b", []float64{2.0}) // order: [b, a]

	// Обращаемся к "a" — promotion, order: [a, b]
	_, ok := cache.Get("a")
	assert.True(t, ok)

	// Добавляем c — должен вытеснить "b", не "a"
	cache.Set("c", []float64{3.0}) // order: [c, a]

	// "a" должен быть в кэше
	_, ok = cache.Get("a")
	assert.True(t, ok)

	// "b" должен быть вытеснен
	_, ok = cache.Get("b")
	assert.False(t, ok)
}

// TestLRUCacheStatsEviction проверяет счётчик вытеснений.
// @sk-task T2.6: Статистика вытеснений (AC-007)
func TestLRUCacheStatsEviction(t *testing.T) {
	stats := &statsCollector{}
	cache := newLRUCache(2, stats)

	cache.Set("a", []float64{1.0})
	cache.Set("b", []float64{2.0})
	cache.Set("c", []float64{3.0}) // eviction

	s := stats.Stats()
	assert.Equal(t, uint64(1), s.Evictions)
}

// TestLRUCacheThreadSafety запускает параллельные операции.
// @sk-task T2.6: Thread-safety базовая проверка (AC-003)
func TestLRUCacheThreadSafety(t *testing.T) {
	stats := &statsCollector{}
	cache := newLRUCache(100, stats)

	// Запускаем несколько горутин
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Set(key, []float64{float64(j)})
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	// Ждём завершения
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем, что кэш в валидном состоянии
	assert.LessOrEqual(t, cache.Len(), 100)
}
