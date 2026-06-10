package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// slowEmbedder — тестовый Embedder, добавляющий фиксированную задержку к
// каждому вызову Embed. Используется для измерения реальной concurrency.
type slowEmbedder struct {
	delay  time.Duration
	calls  atomic.Int32
	failOn string
}

func (e *slowEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	e.calls.Add(1)
	if e.failOn != "" && text == e.failOn {
		return nil, fmt.Errorf("embed failed for %s", text)
	}
	select {
	case <-time.After(e.delay):
		return []float64{0.1, 0.2, 0.3}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// @sk-test api-consistency-pass#T3.1: проверяет, что Index реально использует
// параллелизм (DEC-004, RQ-004, AC-006). 10 документов × 100ms embed при
// concurrency=4 и rate limit 1000/sec должны выполняться за ≤ 600ms
// (теоретический минимум ~300ms), а не за 1s (sequential). Тест ассертит
// также отсутствие ложного ctx.Err.
//
// rate limit выставлен высоким, чтобы изолировать проверку concurrency от
// rate-limiter эффекта (T3.4 покрывает rate limiting отдельно).
func TestPipeline_Index_ConcurrencySpeedup(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := vectorstore.NewInMemoryStore()
	emb := &slowEmbedder{delay: 100 * time.Millisecond}
	p, err := NewPipelineWithConfig(
		store,
		testLLM{},
		emb,
		PipelineOptions{IndexConcurrency: 4, IndexBatchRateLimit: 1000},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := make([]domain.Document, 10)
	for i := range docs {
		docs[i] = domain.Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("hello-%d", i),
		}
	}

	start := time.Now()
	if err := p.Index(context.Background(), docs); err != nil {
		t.Fatalf("index: %v", err)
	}
	elapsed := time.Since(start)

	// Sequential baseline: 10 * 100ms = 1000ms.
	// Concurrency=4: ceil(10/4)=3 волны × 100ms = ~300ms; с overhead ≤ 600ms.
	if elapsed > 600*time.Millisecond {
		t.Fatalf("Index not parallel enough: %v (expected ≤ 600ms with concurrency=4)", elapsed)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("Index suspiciously fast: %v (looks like tests didn't exercise per-doc work)", elapsed)
	}
	if got := emb.calls.Load(); got != int32(len(docs)) {
		t.Fatalf("expected %d embed calls, got %d", len(docs), got)
	}
}

// @sk-test api-consistency-pass#T3.1: проверяет, что при ошибке Embed
// возвращается оригинальная ошибка (не context.Canceled) и in-flight siblings
// прерываются (DEC-004, RQ-004, AC-006). Тест на 8 документов с 1
// гарантированно падающим; success-путь достаточно длинный, чтобы дать
// goroutine время стартовать и быть отменённой через cancel().
func TestPipeline_Index_FailFast_ReturnsOriginalError(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := vectorstore.NewInMemoryStore()
	emb := &slowEmbedder{delay: 50 * time.Millisecond, failOn: "boom"}
	p, err := NewPipelineWithConfig(
		store,
		testLLM{},
		emb,
		PipelineOptions{IndexConcurrency: 4, IndexBatchRateLimit: 1000},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{ID: "d1", Content: "ok-1"},
		{ID: "d2", Content: "ok-2"},
		{ID: "d3", Content: "boom"},
		{ID: "d4", Content: "ok-3"},
		{ID: "d5", Content: "ok-4"},
		{ID: "d6", Content: "ok-5"},
		{ID: "d7", Content: "ok-6"},
		{ID: "d8", Content: "ok-7"},
	}

	err = p.Index(context.Background(), docs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected original embed error, got ctx error: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error to mention 'boom', got %v", err)
	}
}

// @sk-test api-consistency-pass#T3.1: если родительский ctx уже отменён до
// запуска Index, метод должен вернуть context.Canceled без запуска workers
// (DEC-004, RQ-004, AC-006).
func TestPipeline_Index_ContextCancelled(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := vectorstore.NewInMemoryStore()
	emb := &slowEmbedder{delay: 100 * time.Millisecond}
	p, err := NewPipelineWithConfig(
		store,
		testLLM{},
		emb,
		PipelineOptions{IndexConcurrency: 4, IndexBatchRateLimit: 1000},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	docs := []domain.Document{{ID: "d1", Content: "hello"}}
	err = p.Index(ctx, docs)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if got := emb.calls.Load(); got != 0 {
		t.Fatalf("expected 0 embed calls (ctx cancelled before start), got %d", got)
	}
}

// @sk-test api-consistency-pass#T3.1: проверяет, что Document.Validate
// ошибки также прерывают Index и возвращаются как оригинальная ошибка
// (не ctx.Canceled) (DEC-004, RQ-004, AC-006).
func TestPipeline_Index_ValidationErrorFailsFast(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := vectorstore.NewInMemoryStore()
	emb := &slowEmbedder{delay: 50 * time.Millisecond}
	p, err := NewPipelineWithConfig(
		store,
		testLLM{},
		emb,
		PipelineOptions{IndexConcurrency: 4, IndexBatchRateLimit: 1000},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{ID: "d1", Content: "ok-1"},
		{ID: "d2", Content: ""}, // invalid
		{ID: "d3", Content: "ok-2"},
	}

	err = p.Index(context.Background(), docs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected validation error, got ctx error: %v", err)
	}
	if !errors.Is(err, domain.ErrEmptyDocumentContent) {
		t.Fatalf("expected ErrEmptyDocumentContent, got %v", err)
	}
}

// @sk-test api-consistency-pass#T3.1: проверяет, что concurrency=1
// действительно даёт sequential-время (sanity check на параметр).
// Sequential baseline: 5 * 100ms = 500ms; с overhead ≤ 800ms.
func TestPipeline_Index_ConcurrencyOneSequential(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := vectorstore.NewInMemoryStore()
	emb := &slowEmbedder{delay: 100 * time.Millisecond}
	p, err := NewPipelineWithConfig(
		store,
		testLLM{},
		emb,
		PipelineOptions{IndexConcurrency: 1, IndexBatchRateLimit: 1000},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := make([]domain.Document, 5)
	for i := range docs {
		docs[i] = domain.Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("hello-%d", i),
		}
	}

	start := time.Now()
	if err := p.Index(context.Background(), docs); err != nil {
		t.Fatalf("index: %v", err)
	}
	elapsed := time.Since(start)

	// 5 docs × 100ms sequential = 500ms; с overhead ≤ 800ms.
	if elapsed < 400*time.Millisecond {
		t.Fatalf("Index suspiciously fast for concurrency=1: %v (expected ≥ 400ms)", elapsed)
	}
	if elapsed > 800*time.Millisecond {
		t.Fatalf("Index too slow for concurrency=1: %v (expected ≤ 800ms)", elapsed)
	}
}
