package draftrag

import (
	"context"
	"errors"
	"testing"
)

type countEmbedder struct {
	calls int
	vec   []float64
}

func (c *countEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	c.calls++
	return c.vec, nil
}

func TestCachedEmbedder_CachesResults(t *testing.T) {
	base := &countEmbedder{vec: []float64{1, 2, 3}}
	cached, err := NewCachedEmbedder(base, CacheOptions{MaxSize: 10})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	v1, _ := cached.Embed(ctx, "hello")
	v2, _ := cached.Embed(ctx, "hello")
	v3, _ := cached.Embed(ctx, "world")

	if base.calls != 2 {
		t.Fatalf("expected 2 calls (cache hit on second), got %d", base.calls)
	}
	if len(v1) != len(v2) || v1[0] != v2[0] {
		t.Fatal("cached result differs from original")
	}
	_ = v3
}

func TestCachedEmbedder_Stats(t *testing.T) {
	base := &countEmbedder{vec: []float64{0.5}}
	cached, _ := NewCachedEmbedder(base, CacheOptions{})
	ctx := context.Background()

	_, _ = cached.Embed(ctx, "a")
	_, _ = cached.Embed(ctx, "a") // hit
	_, _ = cached.Embed(ctx, "b")

	stats := cached.Stats()
	if stats.Hits != 1 {
		t.Fatalf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Fatalf("expected 2 misses, got %d", stats.Misses)
	}
	if stats.HitRate() < 0.3 || stats.HitRate() > 0.4 {
		t.Fatalf("expected hit rate ~0.33, got %.2f", stats.HitRate())
	}
}

func TestCachedEmbedder_ImplementsEmbedder(t *testing.T) {
	base := &countEmbedder{vec: []float64{1}}
	cached, _ := NewCachedEmbedder(base, CacheOptions{})
	var _ Embedder = cached // compile-time check
}

func TestCachedEmbedder_NilBaseError(t *testing.T) {
	_, err := NewCachedEmbedder(nil, CacheOptions{})
	if err == nil {
		t.Fatal("expected error for nil embedder")
	}
}

func TestCachedEmbedder_PropagatesError(t *testing.T) {
	wantErr := errors.New("embed failed")
	errBase := &errorEmbedder{err: wantErr}
	cached, _ := NewCachedEmbedder(errBase, CacheOptions{})

	_, err := cached.Embed(context.Background(), "text")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

type errorEmbedder struct{ err error }

func (e *errorEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return nil, e.err
}
