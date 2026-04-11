package vectorstore

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// InMemoryStore реализует VectorStore в памяти для тестирования.
type InMemoryStore struct {
	mu     sync.RWMutex
	chunks map[string]domain.Chunk
}

// NewInMemoryStore создаёт новое in-memory хранилище.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		chunks: make(map[string]domain.Chunk),
	}
}

// Upsert сохраняет или обновляет чанк в хранилище.
func (s *InMemoryStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.chunks[chunk.ID] = chunk
	return nil
}

// Delete удаляет чанк по ID из хранилища.
func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.chunks, id)
	return nil
}

// DeleteByParentID удаляет все чанки с указанным ParentID.
func (s *InMemoryStore) DeleteByParentID(ctx context.Context, parentID string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, chunk := range s.chunks {
		if chunk.ParentID == parentID {
			delete(s.chunks, id)
		}
	}
	return nil
}

// Search выполняет поиск похожих чанков по embedding-вектору с использованием cosine similarity.
func (s *InMemoryStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []domain.RetrievedChunk

	for _, chunk := range s.chunks {
		if err := ctx.Err(); err != nil {
			return domain.RetrievalResult{}, err
		}
		if chunk.Embedding == nil {
			continue
		}

		score := cosineSimilarity(embedding, chunk.Embedding)
		results = append(results, domain.RetrievedChunk{
			Chunk: chunk,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	totalFound := len(results)
	if len(results) > topK {
		results = results[:topK]
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  "",
		TotalFound: totalFound,
	}, nil
}

// SearchWithFilter выполняет поиск похожих чанков с фильтрацией по ParentID.
// Пустой filter.ParentIDs (nil или len==0) делегирует в базовый Search.
//
// @ds-task T2.2: Реализовать SearchWithFilter в InMemoryStore для полного соответствия VectorStoreWithFilters (AC-005)
func (s *InMemoryStore) SearchWithFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.ParentIDFilter,
) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.Search(ctx, embedding, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	allowed := make(map[string]struct{}, len(filter.ParentIDs))
	for _, id := range filter.ParentIDs {
		allowed[id] = struct{}{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []domain.RetrievedChunk

	for _, chunk := range s.chunks {
		if err := ctx.Err(); err != nil {
			return domain.RetrievalResult{}, err
		}
		if _, ok := allowed[chunk.ParentID]; !ok {
			continue
		}
		if chunk.Embedding == nil {
			continue
		}

		score := cosineSimilarity(embedding, chunk.Embedding)
		results = append(results, domain.RetrievedChunk{
			Chunk: chunk,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	totalFound := len(results)
	if len(results) > topK {
		results = results[:topK]
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  "",
		TotalFound: totalFound,
	}, nil
}

// SearchWithMetadataFilter выполняет поиск похожих чанков с фильтрацией по полям метаданных.
// Пустой filter.Fields (nil или len==0) делегирует в базовый Search без фильтра.
// Фильтрация выполняется в памяти: все пары (ключ, значение) из Fields должны присутствовать в chunk.Metadata (AND).
//
// @ds-task T2.2: Реализовать SearchWithMetadataFilter в InMemoryStore (RQ-004, AC-002, AC-005)
func (s *InMemoryStore) SearchWithMetadataFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.MetadataFilter,
) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return s.Search(ctx, embedding, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []domain.RetrievedChunk

	for _, chunk := range s.chunks {
		if err := ctx.Err(); err != nil {
			return domain.RetrievalResult{}, err
		}
		if chunk.Embedding == nil {
			continue
		}
		if !matchesMetadataFilter(chunk.Metadata, filter.Fields) {
			continue
		}

		score := cosineSimilarity(embedding, chunk.Embedding)
		results = append(results, domain.RetrievedChunk{
			Chunk: chunk,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	totalFound := len(results)
	if len(results) > topK {
		results = results[:topK]
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  "",
		TotalFound: totalFound,
	}, nil
}

// matchesMetadataFilter проверяет, что все пары (ключ, значение) из fields присутствуют в metadata.
// Пустой fields означает «совпадает всегда».
func matchesMetadataFilter(metadata, fields map[string]string) bool {
	for k, v := range fields {
		if metadata[k] != v {
			return false
		}
	}
	return true
}

// cosineSimilarity вычисляет cosine similarity между двумя векторами.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))

	if similarity > 1 {
		return 1
	}
	if similarity < -1 {
		return -1
	}

	return similarity
}
