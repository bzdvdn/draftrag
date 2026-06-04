package draftrag

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-task pipeline-e2e-benchmarks#T1.1: bench helpers — benchEmbedder, benchLLM, genDocs, setupBenchPipeline
type benchEmbedder struct{}

func (benchEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{1, 0, 0}, nil
}

type benchLLM struct{}

func (benchLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "answer", nil
}

func genDocs(count, contentSize int) []domain.Document {
	docs := make([]domain.Document, count)
	for i := range docs {
		docs[i] = domain.Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: strings.Repeat("a", contentSize),
		}
	}
	return docs
}

func setupBenchPipeline(b *testing.B) (*Pipeline, *vectorstore.InMemoryStore) {
	b.Helper()
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, benchLLM{}, benchEmbedder{})
	return p, store
}

func setupBenchPipelineWithChunker(b *testing.B) *Pipeline {
	b.Helper()
	store := vectorstore.NewInMemoryStore()
	chunker := NewBasicChunker(BasicChunkerOptions{
		ChunkSize: 100,
		Overlap:   20,
	})
	p := NewPipelineWithChunker(store, benchLLM{}, benchEmbedder{}, chunker)
	return p
}

func prepopulateStore(b *testing.B, store *vectorstore.InMemoryStore, docs []domain.Document) {
	b.Helper()
	ctx := context.Background()
	for _, doc := range docs {
		chunk := domain.Chunk{
			ID:        doc.ID + "#0",
			Content:   doc.Content,
			ParentID:  doc.ID,
			Embedding: []float64{1, 0, 0},
		}
		if err := store.Upsert(ctx, chunk); err != nil {
			b.Fatalf("prepopulate upsert: %v", err)
		}
	}
}

// @sk-test pipeline-e2e-benchmarks#T2.1: Index benchmark with sub-benchmarks docs10/docs100/docs1000
func BenchmarkPipelineE2E_Index(b *testing.B) {
	sizes := []struct {
		name       string
		docCount   int
		contentLen int
	}{
		{"docs10", 10, 500},
		{"docs100", 100, 500},
		{"docs1000", 1000, 500},
	}

	for _, sz := range sizes {
		n := sz.docCount
		if testing.Short() && n > 10 {
			continue
		}

		b.Run(sz.name, func(b *testing.B) {
			p := setupBenchPipelineWithChunker(b)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				docs := genDocs(n, sz.contentLen)
				if err := p.Index(ctx, docs); err != nil {
					b.Fatalf("index: %v", err)
				}
			}
		})
	}
}

// @sk-test pipeline-e2e-benchmarks#T2.2: Query benchmark with pre-populated store
func BenchmarkPipelineE2E_Query(b *testing.B) {
	sizes := []struct {
		name       string
		docCount   int
		contentLen int
	}{
		{"docs10", 10, 500},
		{"docs100", 100, 500},
		{"docs1000", 1000, 500},
	}

	for _, sz := range sizes {
		n := sz.docCount
		if testing.Short() && n > 10 {
			continue
		}

		b.Run(sz.name, func(b *testing.B) {
			p, store := setupBenchPipeline(b)
			ctx := context.Background()

			docs := genDocs(n, sz.contentLen)
			prepopulateStore(b, store, docs)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := p.Query(ctx, "test query")
				if err != nil {
					b.Fatalf("query: %v", err)
				}
			}
		})
	}
}

// @sk-test pipeline-e2e-benchmarks#T3.1: Full pipeline Index+Query benchmark
func BenchmarkPipelineE2E_Full(b *testing.B) {
	sizes := []struct {
		name       string
		docCount   int
		contentLen int
	}{
		{"docs10", 10, 500},
		{"docs100", 100, 500},
		{"docs1000", 1000, 500},
	}

	for _, sz := range sizes {
		n := sz.docCount
		if testing.Short() && n > 10 {
			continue
		}

		b.Run(sz.name, func(b *testing.B) {
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				p := setupBenchPipelineWithChunker(b)

				docs := genDocs(n, sz.contentLen)
				if err := p.Index(ctx, docs); err != nil {
					b.Fatalf("index: %v", err)
				}

				_, err := p.Query(ctx, "test query")
				if err != nil {
					b.Fatalf("query: %v", err)
				}
			}
		})
	}
}
