package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// timestampRecorder потокобезопасно собирает временные метки вызовов processor'а.
type timestampRecorder struct {
	mu sync.Mutex
	ts []time.Time
}

func (r *timestampRecorder) record() {
	r.mu.Lock()
	r.ts = append(r.ts, time.Now())
	r.mu.Unlock()
}

func (r *timestampRecorder) countInWindow(start time.Time, dur time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := start.Add(dur)
	n := 0
	for _, t := range r.ts {
		if t.After(start) && !t.After(cutoff) {
			n++
		}
	}
	return n
}

func makeRateLimitDocs(n int) []domain.Document {
	docs := make([]domain.Document, n)
	for i := range docs {
		docs[i] = domain.Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("text %d", i),
		}
	}
	return docs
}

// @sk-test api-consistency-pass#T3.4: per-worker rate-limiter — каждый worker
// имеет свой ticker (DEC-007, RQ-007, AC-011, AC-012).
//
// Сценарий: Concurrency=4, RateLimit=10, PerWorker=true. Ожидаем ~40 вызовов
// в 1-секундном окне (4 worker'а × 10 calls/sec каждый).
func TestProcessDocsConcurrently_PerWorker_RateIsPerWorker(t *testing.T) {
	rec := &timestampRecorder{}
	docs := makeRateLimitDocs(60) // 60 / 4 workers / 10 per sec per worker ≈ 1.5s

	start := time.Now()
	successful, failed, ctxErr := processDocsConcurrently(
		context.Background(),
		docs,
		4,    // concurrency
		10,   // rateLimit
		true, // perWorker
		func(ctx context.Context, doc domain.Document) error {
			rec.record()
			return nil
		},
	)
	elapsed := time.Since(start)

	if ctxErr != nil {
		t.Fatalf("unexpected ctxErr: %v", ctxErr)
	}
	if len(failed) != 0 {
		t.Fatalf("expected 0 failed, got %d", len(failed))
	}
	if len(successful) != 60 {
		t.Fatalf("expected 60 successful, got %d", len(successful))
	}

	n := rec.countInWindow(start, time.Second)
	// Expected ~40, ±30% tolerance: [28..52].
	if n < 28 || n > 52 {
		t.Errorf("perWorker=true: expected ~40 calls in [start, start+1s], got %d in %v", n, elapsed)
	}
}

// @sk-test api-consistency-pass#T3.4: shared (default) rate-limiter — общий ticker
// на пул (DEC-007, RQ-007, AC-011).
//
// Сценарий: Concurrency=4, RateLimit=10, PerWorker=false. Ожидаем ~10 вызовов
// в 1-секундном окне (общий ticker, 4 worker'а конкурируют за тики).
func TestProcessDocsConcurrently_Shared_RateIsPoolWide(t *testing.T) {
	rec := &timestampRecorder{}
	docs := makeRateLimitDocs(30) // 30 calls / 10 per sec ≈ 3s total

	start := time.Now()
	successful, failed, ctxErr := processDocsConcurrently(
		context.Background(),
		docs,
		4,     // concurrency
		10,    // rateLimit
		false, // perWorker (shared)
		func(ctx context.Context, doc domain.Document) error {
			rec.record()
			return nil
		},
	)
	elapsed := time.Since(start)

	if ctxErr != nil {
		t.Fatalf("unexpected ctxErr: %v", ctxErr)
	}
	if len(failed) != 0 {
		t.Fatalf("expected 0 failed, got %d", len(failed))
	}
	if len(successful) != 30 {
		t.Fatalf("expected 30 successful, got %d", len(successful))
	}

	n := rec.countInWindow(start, time.Second)
	// Expected ~10, ±30% tolerance: [7..13].
	if n < 7 || n > 13 {
		t.Errorf("perWorker=false: expected ~10 calls in [start, start+1s], got %d in %v", n, elapsed)
	}
}

// @sk-test api-consistency-pass#T3.4: perWorker=true с rateLimit=0 — rate limiting
// отключён, все docs обрабатываются параллельно (DEC-007, RQ-007).
//
// Сценарий: Concurrency=4, RateLimit=0, PerWorker=true. Никакого ожидания
// на тики — все 10 документов обрабатываются практически мгновенно.
func TestProcessDocsConcurrently_ZeroRateLimit_NoThrottling(t *testing.T) {
	rec := &timestampRecorder{}
	docs := makeRateLimitDocs(10)

	start := time.Now()
	successful, failed, ctxErr := processDocsConcurrently(
		context.Background(),
		docs,
		4,    // concurrency
		0,    // rateLimit (disabled)
		true, // perWorker (irrelevant when rateLimit=0)
		func(ctx context.Context, doc domain.Document) error {
			rec.record()
			return nil
		},
	)
	elapsed := time.Since(start)

	if ctxErr != nil {
		t.Fatalf("unexpected ctxErr: %v", ctxErr)
	}
	if len(failed) != 0 {
		t.Fatalf("expected 0 failed, got %d", len(failed))
	}
	if len(successful) != 10 {
		t.Fatalf("expected 10 successful, got %d", len(successful))
	}

	// Все 10 вызовов должны произойти в первые ~100ms (concurrency=4 + zero throttle).
	n := rec.countInWindow(start, time.Millisecond*100)
	if n != 10 {
		t.Errorf("rateLimit=0: expected 10 calls in 100ms, got %d in %v", n, elapsed)
	}
}
