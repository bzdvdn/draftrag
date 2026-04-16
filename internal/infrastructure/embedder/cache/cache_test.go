package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbedder тестовая реализация Embedder с счётчиком вызовов.
type mockEmbedder struct {
	callCount int
	vectors   map[string][]float64
	embedFunc func(text string) ([]float64, error)
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	m.callCount++
	if m.embedFunc != nil {
		return m.embedFunc(text)
	}
	if vec, ok := m.vectors[text]; ok {
		return vec, nil
	}
	// Default: возвращаем вектор на основе длины текста
	return []float64{float64(len(text)), float64(len(text) * 2)}, nil
}

// TestBasicCaching проверяет базовое кэширование (AC-001).
// @sk-task T2.6: Базовое кэширование (AC-001)
func TestBasicCaching(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}

	cache, err := NewEmbedderCache(mock, WithCacheSize(100))
	require.NoError(t, err)

	text := "test text"

	// Первый вызов — должен обратиться к embedder
	vec1, err := cache.Embed(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.callCount, "первый вызов должен обратиться к embedder")

	// Второй вызов — должен взять из кэша
	vec2, err := cache.Embed(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.callCount, "второй вызов не должен обращаться к embedder")
	assert.Equal(t, vec1, vec2, "векторы должны совпадать")

	// Статистика
	stats := cache.Stats()
	assert.Equal(t, uint64(1), stats.Hits, "должен быть 1 hit")
	assert.Equal(t, uint64(1), stats.Misses, "должен быть 1 miss")
}

// TestLRUEviction проверяет вытеснение при переполнении (AC-002).
// @sk-task T2.6: LRU eviction (AC-002)
func TestLRUEviction(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}

	// Кэш размером 2
	cache, err := NewEmbedderCache(mock, WithCacheSize(2))
	require.NoError(t, err)

	// Добавляем 3 разных текста
	_, err = cache.Embed(ctx, "text1")
	require.NoError(t, err)
	_, err = cache.Embed(ctx, "text2")
	require.NoError(t, err)
	_, err = cache.Embed(ctx, "text3")
	require.NoError(t, err)

	// Должно быть 3 вызова embedder (все miss)
	assert.Equal(t, 3, mock.callCount)

	// Проверяем размер кэша
	assert.Equal(t, 2, cache.cache.Len(), "размер кэша не должен превышать capacity")

	// Проверяем eviction в статистике
	stats := cache.Stats()
	assert.Equal(t, uint64(1), stats.Evictions, "должно быть 1 вытеснение")
}

// TestLRUPromotion проверяет promotion (часто используемые остаются).
// @sk-task T2.6: LRU promotion (DEC-001)
func TestLRUPromotion(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}

	// Кэш размером 2
	cache, err := NewEmbedderCache(mock, WithCacheSize(2))
	require.NoError(t, err)

	// Добавляем text1, text2, потом снова обращаемся к text1 (promotion)
	_, err = cache.Embed(ctx, "text1") // miss, в кэше: [text1]
	require.NoError(t, err)
	_, err = cache.Embed(ctx, "text2") // miss, в кэше: [text2, text1]
	require.NoError(t, err)

	// Promotion: text1 становится most recent
	_, err = cache.Embed(ctx, "text1") // hit, в кэше: [text1, text2]
	require.NoError(t, err)

	// Добавляем text3 — должно вытеснить text2, не text1
	_, err = cache.Embed(ctx, "text3") // miss, в кэше: [text3, text1]
	require.NoError(t, err)

	// text1 должен быть в кэше (hit)
	mock.callCount = 0
	_, err = cache.Embed(ctx, "text1")
	require.NoError(t, err)
	assert.Equal(t, 0, mock.callCount, "text1 должен быть в кэше")

	// text2 должен быть вытеснен (miss)
	_, err = cache.Embed(ctx, "text2")
	require.NoError(t, err)
	assert.Equal(t, 1, mock.callCount, "text2 должен быть вытеснен")
}

// TestCacheStats проверяет корректность статистики (AC-007).
// @sk-task T2.6: Статистика кэша (AC-007)
func TestCacheStats(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}

	cache, err := NewEmbedderCache(mock, WithCacheSize(100))
	require.NoError(t, err)

	// 5 hit, 3 miss
	for i := 0; i < 3; i++ {
		_, err := cache.Embed(ctx, "unique_text_"+string(rune('a'+i)))
		require.NoError(t, err)
	}
	// Повторные вызовы (должны быть hits)
	for i := 0; i < 5; i++ {
		_, err := cache.Embed(ctx, "unique_text_a") // первый текст
		require.NoError(t, err)
	}

	stats := cache.Stats()
	assert.Equal(t, uint64(5), stats.Hits, "hits")
	assert.Equal(t, uint64(3), stats.Misses, "misses")
	assert.InDelta(t, 5.0/8.0, stats.HitRate(), 0.001, "hit rate")
}

// TestHashConsistency проверяет консистентность хэширования (AC-006).
// @sk-task T2.6: Хэш консистентности (AC-006)
func TestHashConsistency(t *testing.T) {
	cache, _ := NewEmbedderCache(&mockEmbedder{}, WithCacheSize(10))

	text1 := "identical text"
	text2 := "identical text"

	// Должны давать одинаковый хэш
	hash1 := cache.hashKey(text1)
	hash2 := cache.hashKey(text2)
	assert.Equal(t, hash1, hash2, "одинаковые тексты должны давать одинаковый хэш")

	// Разные тексты — разные хэши (с высокой вероятностью)
	hash3 := cache.hashKey("different text")
	assert.NotEqual(t, hash1, hash3, "разные тексты должны давать разные хэши")
}

// TestCacheSizeValidation проверяет валидацию размера кэша.
// @sk-task T2.6: Валидация размера (RQ-004)
func TestCacheSizeValidation(t *testing.T) {
	mock := &mockEmbedder{}

	// Размер 0 должен стать 1
	cache, err := NewEmbedderCache(mock, WithCacheSize(0))
	require.NoError(t, err)
	assert.Equal(t, 1, cache.cacheSize)

	// Размер -5 должен стать 1
	cache, err = NewEmbedderCache(mock, WithCacheSize(-5))
	require.NoError(t, err)
	assert.Equal(t, 1, cache.cacheSize)
}

// TestNilEmbedder проверяет валидацию базового embedder.
// @sk-task T2.6: Валидация embedder (RQ-002)
func TestNilEmbedder(t *testing.T) {
	_, err := NewEmbedderCache(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedder cannot be nil")
}

// TestEmbedError проверяет проброс ошибок от базового embedder.
func TestEmbedError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("embedder error")
	mock := &mockEmbedder{
		embedFunc: func(_ string) ([]float64, error) {
			return nil, expectedErr
		},
	}

	cache, err := NewEmbedderCache(mock)
	require.NoError(t, err)

	_, err = cache.Embed(ctx, "test")
	assert.ErrorIs(t, err, expectedErr)
}

// TestCacheImplementsEmbedder проверяет, что кэш реализует domain.Embedder.
// @sk-task T2.6: Проверка интерфейса (RQ-001)
func TestCacheImplementsEmbedder(t *testing.T) {
	mock := &mockEmbedder{}
	cache, err := NewEmbedderCache(mock)
	require.NoError(t, err)

	// Проверяем, что cache реализует domain.Embedder
	var _ domain.Embedder = cache
}

// BenchmarkCacheHit измеряет latency попадания в кэш (SC-002).
// @sk-task T2.6: Бенчмарк hit latency (SC-002)
func BenchmarkCacheHit(b *testing.B) {
	ctx := context.Background()
	mock := &mockEmbedder{}
	cache, _ := NewEmbedderCache(mock, WithCacheSize(1000))

	// Предзаполняем кэш
	_, _ = cache.Embed(ctx, "benchmark text")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Embed(ctx, "benchmark text")
	}
}
