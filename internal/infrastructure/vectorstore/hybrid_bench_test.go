package vectorstore

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// BenchmarkFuseResults_RRF бенчмарк RRF fusion с разным количеством результатов.
func BenchmarkFuseResults_RRF(b *testing.B) {
	// Генерируем тестовые данные
	semantic := make([]domain.RetrievedChunk, 100)
	bm25 := make([]domain.RetrievedChunk, 100)
	for i := 0; i < 100; i++ {
		semantic[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('a' + i%26))},
			Score: float64(100-i) / 100.0,
		}
		bm25[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('z' - i%26))},
			Score: float64(100-i) / 100.0,
		}
	}

	config := domain.HybridConfig{
		UseRRF:   true,
		RRFK:     60,
		BMFinalK: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fuseResults(semantic, bm25, config)
	}
}

// BenchmarkFuseResults_RRF_Small бенчмарк RRF с малым набором (как в реальном использовании).
func BenchmarkFuseResults_RRF_Small(b *testing.B) {
	semantic := make([]domain.RetrievedChunk, 20)
	bm25 := make([]domain.RetrievedChunk, 20)
	for i := 0; i < 20; i++ {
		semantic[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('a' + i%26))},
			Score: float64(20-i) / 20.0,
		}
		bm25[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('z' - i%26))},
			Score: float64(20-i) / 20.0,
		}
	}

	config := domain.HybridConfig{
		UseRRF:   true,
		RRFK:     60,
		BMFinalK: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fuseResults(semantic, bm25, config)
	}
}

// BenchmarkFuseResults_Weighted бенчмарк weighted fusion.
func BenchmarkFuseResults_Weighted(b *testing.B) {
	semantic := make([]domain.RetrievedChunk, 100)
	bm25 := make([]domain.RetrievedChunk, 100)
	for i := 0; i < 100; i++ {
		semantic[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('a' + i%26))},
			Score: float64(100-i) / 100.0,
		}
		bm25[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('z' - i%26))},
			Score: float64(100-i) / 100.0,
		}
	}

	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 0.7,
		BMFinalK:       10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fuseResults(semantic, bm25, config)
	}
}

// BenchmarkCalculateRRF бенчмарк чистого RRF.
func BenchmarkCalculateRRF(b *testing.B) {
	semantic := make([]domain.RetrievedChunk, 50)
	bm25 := make([]domain.RetrievedChunk, 50)
	for i := 0; i < 50; i++ {
		semantic[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('a' + i%26))},
			Score: float64(50-i) / 50.0,
		}
		bm25[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('z' - i%26))},
			Score: float64(50-i) / 50.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateRRF(semantic, bm25, 60)
	}
}

// BenchmarkCalculateWeightedScore бенчмарк чистого weighted score.
func BenchmarkCalculateWeightedScore(b *testing.B) {
	semantic := make([]domain.RetrievedChunk, 50)
	bm25 := make([]domain.RetrievedChunk, 50)
	for i := 0; i < 50; i++ {
		semantic[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('a' + i%26))},
			Score: float64(50-i) / 50.0,
		}
		bm25[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: string(rune('z' - i%26))},
			Score: float64(50-i) / 50.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateWeightedScore(semantic, bm25, 0.7)
	}
}
