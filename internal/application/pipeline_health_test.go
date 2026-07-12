package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test arch-issues#T3.3: TestPipeline_HealthOK (AC-007)
func TestPipeline_HealthOK(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	err = p.Health(context.Background())
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

type unhealthyStore struct{}

func (unhealthyStore) Health(_ context.Context) error { return errors.New("store is down") }
func (unhealthyStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (unhealthyStore) Delete(_ context.Context, _ string) error       { return nil }
func (unhealthyStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

// @sk-test arch-issues#T3.3: TestPipeline_HealthUnhealthyStore (AC-007)
func TestPipeline_HealthUnhealthyStore(t *testing.T) {
	p, err := NewPipeline(unhealthyStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	err = p.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for unhealthy store")
	}
}

type slowStore struct{}

func (slowStore) Health(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
func (slowStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (slowStore) Delete(_ context.Context, _ string) error       { return nil }
func (slowStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

// @sk-test arch-issues#T3.3: TestPipeline_HealthTimeout (AC-007)
func TestPipeline_HealthTimeout(t *testing.T) {
	p, err := NewPipeline(slowStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	err = p.Health(context.Background())
	if err == nil {
		t.Fatal("expected timeout error for slow store")
	}
}
