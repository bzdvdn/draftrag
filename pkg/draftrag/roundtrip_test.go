package draftrag

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

func randomChunk(rng *rand.Rand, id int) domain.Chunk {
	return domain.Chunk{
		ID:       fmt.Sprintf("roundtrip-%d", id),
		Content:  randomString(rng, 1+rng.Intn(100)),
		ParentID: fmt.Sprintf("parent-%d", rng.Intn(10)),
		Embedding: []float64{
			rng.Float64()*2 - 1,
			rng.Float64()*2 - 1,
			rng.Float64()*2 - 1,
		},
		Position: rng.Intn(100),
	}
}

func randomString(rng *rand.Rand, n int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 \t\n\x00\u00e9\u0430"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

// @sk-test fuzz-property-tests#T2.2: VectorStore roundtrip property — Upsert → Search → same ID
func TestVectorStoreRoundtrip(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		store := vectorstore.NewInMemoryStore()
		chunk := randomChunk(rng, i)

		if err := store.Upsert(ctx, chunk); err != nil {
			t.Fatalf("iter %d: upsert: %v", i, err)
		}

		result, err := store.Search(ctx, chunk.Embedding, 10)
		if err != nil {
			t.Fatalf("iter %d: search: %v", i, err)
		}

		if len(result.Chunks) == 0 {
			t.Fatalf("iter %d: expected at least 1 result, got 0 (chunk ID=%s)", i, chunk.ID)
		}

		found := false
		for _, rc := range result.Chunks {
			if rc.Chunk.ID == chunk.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("iter %d: chunk ID=%s not found in search results (%d results)", i, chunk.ID, len(result.Chunks))
		}
	}
}
