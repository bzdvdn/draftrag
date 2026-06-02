package draftrag

import (
	"context"
	"errors"
	"testing"
)

// @sk-test hardening-2026q2#T3.3: NewRetryEmbedder
func TestNewRetryEmbedder_CreatesWrapper(t *testing.T) {
	base := &countEmbedder{vec: []float64{1, 2, 3}}
	re := NewRetryEmbedder(base, RetryOptions{})
	if re == nil {
		t.Fatal("expected non-nil RetryEmbedder")
	}

	var _ Embedder = re
}

func TestNewRetryEmbedder_DelegatesEmbed(t *testing.T) {
	base := &countEmbedder{vec: []float64{0.5, 0.25}}
	re := NewRetryEmbedder(base, RetryOptions{})

	ctx := context.Background()
	v, err := re.Embed(ctx, "test")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(v) != 2 || v[0] != 0.5 || v[1] != 0.25 {
		t.Fatalf("unexpected vector: %v", v)
	}
	if base.calls != 1 {
		t.Fatalf("expected 1 call to base embedder, got %d", base.calls)
	}
}

func TestNewRetryEmbedder_PropagatesError(t *testing.T) {
	want := errors.New("embed failed")
	base := &errorEmbedder{err: want}
	re := NewRetryEmbedder(base, RetryOptions{})

	_, err := re.Embed(context.Background(), "x")
	if !errors.Is(err, want) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestNewRetryEmbedder_WithOptions(t *testing.T) {
	base := &countEmbedder{vec: []float64{1}}
	re := NewRetryEmbedder(base, RetryOptions{
		MaxRetries:  5,
		CBThreshold: 10,
	})
	if re == nil {
		t.Fatal("expected non-nil RetryEmbedder")
	}

	_, err := re.Embed(context.Background(), "test")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
}
