package cache

import (
	"container/list"
	"sync"
)

// lruEntry представляет элемент кэша.
type lruEntry struct {
	key   string
	value []float64
}

// lruCache реализует thread-safe LRU кэш с ограниченным размером.
// @ds-task T1.2: Структура LRU кэша с RWMutex (DEC-001)
type lruCache struct {
	capacity int
	items    map[string]*list.Element // O(1) lookup
	order    *list.List               // LRU ordering: front = most recent
	mu       sync.RWMutex
	stats    *statsCollector
}

// newLRUCache создаёт новый LRU кэш с указанной ёмкостью.
// @ds-task T1.2: Конструктор LRU кэша (DEC-001)
func newLRUCache(capacity int, stats *statsCollector) *lruCache {
	if capacity < 1 {
		capacity = 1
	}
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
		stats:    stats,
	}
}

// Get возвращает значение из кэша и флаг наличия.
// При попадании элемент перемещается в начало списка (promotion).
// @ds-task T1.2, T2.3: Метод получения с promotion (DEC-001, AC-002)
// Используем Lock (не RLock) т.к. promotion модифицирует список.
func (c *lruCache) Get(key string) ([]float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// Promotion: перемещаем в front (most recent)
	c.order.MoveToFront(elem)

	entry := elem.Value.(*lruEntry)
	return entry.value, true
}

// Set добавляет или обновляет значение в кэше.
// При превышении capacity вытесняет least recently used элемент.
// @ds-task T1.2, T2.2: Метод добавления с eviction (DEC-001, AC-002)
func (c *lruCache) Set(key string, value []float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Если ключ уже есть — обновляем значение и перемещаем в front
	if elem, ok := c.items[key]; ok {
		elem.Value.(*lruEntry).value = value
		c.order.MoveToFront(elem)
		return
	}

	// Создаём новый элемент
	entry := &lruEntry{key: key, value: value}
	elem := c.order.PushFront(entry)
	c.items[key] = elem

	// Eviction: если превысили capacity, удаляем tail (LRU)
	if c.order.Len() > c.capacity {
		c.evictLRU()
	}
}

// evictLRU удаляет least recently used элемент (tail списка).
// @ds-task T2.2: Логика вытеснения LRU (AC-002)
func (c *lruCache) evictLRU() {
	tail := c.order.Back()
	if tail == nil {
		return
	}

	entry := tail.Value.(*lruEntry)
	delete(c.items, entry.key)
	c.order.Remove(tail)

	if c.stats != nil {
		c.stats.RecordEviction()
	}
}

// Len возвращает текущее количество элементов в кэше.
// @ds-task T1.2: Метод получения размера кэша (DEC-001)
func (c *lruCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}
