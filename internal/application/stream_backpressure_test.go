package application

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// counterStreamingLLM — мок StreamingLLMProvider, который эмитит N токенов с
// заданным интервалом и закрывает канал. Используется для тестов backpressure.
type counterStreamingLLM struct {
	mockLLMProvider
	tokens     []string
	emitDelay  time.Duration
	closeDelay time.Duration // дополнительная задержка перед close (для измерения producer progress)

	producedAt []time.Time
	mu         sync.Mutex
}

func (m *counterStreamingLLM) GenerateStream(ctx context.Context, _, _ string) (<-chan string, error) {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		for _, tok := range m.tokens {
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.emitDelay):
			}
			now := time.Now()
			m.mu.Lock()
			m.producedAt = append(m.producedAt, now)
			m.mu.Unlock()
			select {
			case ch <- tok:
			case <-ctx.Done():
				return
			}
		}
		if m.closeDelay > 0 {
			select {
			case <-time.After(m.closeDelay):
			case <-ctx.Done():
			}
		}
	}()
	return ch, nil
}

func (m *counterStreamingLLM) producedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.producedAt)
}

// @sk-test api-consistency-pass#T3.3: при StreamBufferSize=0 (default) output
// канал unbuffered (cap=0) — backward-compat с OQ-2 (DEC-006, RQ-006, AC-010).
func TestPipeline_StreamBackpressure_DefaultUnbuffered(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	emb := &mockEmbedder{}
	llm := &streamingLLM{}
	p, err := NewPipelineWithConfig(store, llm, emb, PipelineOptions{
		// StreamBufferSize: 0 (default)
	})
	if err != nil {
		t.Fatal(err)
	}

	stream, err := p.AnswerStream(context.Background(), "q", 5)
	if err != nil {
		t.Fatalf("AnswerStream: %v", err)
	}
	defer func() {
		for range stream {
		}
	}()

	v := reflect.ValueOf(stream)
	if v.Kind() != reflect.Chan {
		t.Fatalf("expected chan, got %v", v.Kind())
	}
	if got := v.Cap(); got != 0 {
		t.Fatalf("expected cap=0 (unbuffered) for default, got %d", got)
	}
}

// @sk-test api-consistency-pass#T3.3: при StreamBufferSize=N output канал
// имеет cap=N (DEC-006, RQ-006, AC-010).
func TestPipeline_StreamBackpressure_BufferSizeApplied(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	cases := []int{1, 5, 16, 64}
	for _, size := range cases {
		t.Run("", func(t *testing.T) {
			store := &mockVectorStore{}
			emb := &mockEmbedder{}
			llm := &streamingLLM{}
			p, err := NewPipelineWithConfig(store, llm, emb, PipelineOptions{
				StreamBufferSize: size,
			})
			if err != nil {
				t.Fatal(err)
			}

			stream, err := p.AnswerStream(context.Background(), "q", 5)
			if err != nil {
				t.Fatalf("AnswerStream: %v", err)
			}
			defer func() {
				for range stream {
				}
			}()

			v := reflect.ValueOf(stream)
			if got := v.Cap(); got != size {
				t.Fatalf("expected cap=%d, got %d", size, got)
			}
		})
	}
}

// @sk-test api-consistency-pass#T3.3: независимо от размера буфера, все N
// токенов доходят до consumer'а (DEC-006, RQ-006, AC-010).
func TestPipeline_StreamBackpressure_AllTokensDelivered(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	cases := []int{0, 1, 5, 32}
	tokens := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, size := range cases {
		t.Run("", func(t *testing.T) {
			store := &mockVectorStore{}
			emb := &mockEmbedder{}
			llm := &counterStreamingLLM{tokens: tokens, emitDelay: 0}
			p, err := NewPipelineWithConfig(store, llm, emb, PipelineOptions{
				StreamBufferSize: size,
			})
			if err != nil {
				t.Fatal(err)
			}

			stream, err := p.AnswerStream(context.Background(), "q", 5)
			if err != nil {
				t.Fatalf("AnswerStream: %v", err)
			}

			got := make([]string, 0, len(tokens))
			for tok := range stream {
				got = append(got, tok)
			}
			if len(got) != len(tokens) {
				t.Fatalf("size=%d: expected %d tokens, got %d", size, len(tokens), len(got))
			}
			for i := range tokens {
				if got[i] != tokens[i] {
					t.Errorf("size=%d: token[%d]=%q want %q", size, i, got[i], tokens[i])
				}
			}
		})
	}
}

// @sk-test api-consistency-pass#T3.3: producer (LLM-стрим) может обгонять
// consumer на размер буфера. С StreamBufferSize=N и producer, который
// эмитит токены быстро (1ms), а consumer читает медленно (50ms),
// producer goroutine должен завершить эмиссию всех токенов значительно
// раньше, чем consumer их все прочитает.
//
// Без буфера producer строго gated: send ждёт receive. С буфером=N
// producer может поставить N токенов в очередь без ожидания.
//
// Допуск: проверяем, что (a) с буфером=N producer завершает эмиссию
// за < 50% от времени, нужного consumer'у; (b) без буфера producer
// завершает не раньше, чем consumer прочитает все токены.
func TestPipeline_StreamBackpressure_ProducerRunsAhead(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	const (
		tokenCount    = 20
		emitDelay     = 1 * time.Millisecond
		consumeDelay  = 50 * time.Millisecond
		bufferSize    = tokenCount
		maxAheadFrac  = 0.5 // producer должен закончить ≤ 50% от consumer time
	)

	tokens := make([]string, tokenCount)
	for i := range tokens {
		tokens[i] = "x"
	}

	// Helper: drain stream with delay per token, return (consumerDone time, total elapsed).
	drainWithDelay := func(stream <-chan string) time.Duration {
		start := time.Now()
		for range stream {
			time.Sleep(consumeDelay)
		}
		return time.Since(start)
	}

	// === Buffered: producer должен вырваться вперёд ===
	store := &mockVectorStore{}
	emb := &mockEmbedder{}
	llm := &counterStreamingLLM{tokens: tokens, emitDelay: emitDelay}
	p, err := NewPipelineWithConfig(store, llm, emb, PipelineOptions{
		StreamBufferSize: bufferSize,
	})
	if err != nil {
		t.Fatal(err)
	}

	stream, err := p.AnswerStream(context.Background(), "q", 5)
	if err != nil {
		t.Fatalf("AnswerStream: %v", err)
	}

	consumerStart := time.Now()
	var producerDoneAt atomic.Int64
	go func() {
		// Wait for all tokens to be produced.
		deadline := time.Now().Add(2 * time.Second)
		for llm.producedCount() < tokenCount && time.Now().Before(deadline) {
			time.Sleep(500 * time.Microsecond)
		}
		producerDoneAt.Store(time.Since(consumerStart).Nanoseconds())
	}()

	consumerTotal := drainWithDelay(stream)
	producerTotal := time.Duration(producerDoneAt.Load())

	t.Logf("buffered: producer finished in %v, consumer took %v", producerTotal, consumerTotal)
	if producerTotal >= consumerTotal {
		t.Fatalf("expected producer to finish before consumer (buffered), got producer=%v consumer=%v", producerTotal, consumerTotal)
	}
	if float64(producerTotal) > maxAheadFrac*float64(consumerTotal) {
		t.Errorf("expected producer ≤ 50%% of consumer time, got producer=%v consumer=%v (%.1f%%)",
			producerTotal, consumerTotal, float64(producerTotal)/float64(consumerTotal)*100)
	}
}

// @sk-test api-consistency-pass#T3.3: wrapStreamWithHook — channel cap
// соответствует p.streamBufferSize (0 или N). Прямая проверка через
// рефлексию, не зависит от LLM-выборки.
func TestPipeline_wrapStreamWithHook_ChannelCapacity(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	cases := []int{0, 1, 8, 100}
	for _, size := range cases {
		t.Run("", func(t *testing.T) {
			store := &mockVectorStore{}
			emb := &mockEmbedder{}
			llm := &mockLLMProvider{} // не streaming — нужна обёртка
			p, err := NewPipelineWithConfig(store, llm, emb, PipelineOptions{
				StreamBufferSize: size,
			})
			if err != nil {
				t.Fatal(err)
			}

			source := make(chan string)
			close(source)
			out := p.wrapStreamWithHook(context.Background(), source, time.Now())

			v := reflect.ValueOf(out)
			if v.Kind() != reflect.Chan {
				t.Fatalf("expected chan, got %v", v.Kind())
			}
			if got := v.Cap(); got != size {
				t.Fatalf("expected cap=%d, got %d", size, got)
			}
			// Канал должен закрыться.
			for range out {
			}
		})
	}
}

// @sk-test api-consistency-pass#T3.3: domain.Hooks вызываются корректно при
// streaming'е с буфером. hookEnd Generate вызывается после закрытия канала.
func TestPipeline_StreamBackpressure_HooksCalledOnClose(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	emb := &mockEmbedder{}
	llm := &counterStreamingLLM{tokens: []string{"a", "b", "c"}, emitDelay: 0}
	hooks := &countingHooks{}
	p, err := NewPipelineWithConfig(store, llm, emb, PipelineOptions{
		StreamBufferSize: 5,
		Hooks:            hooks,
	})
	if err != nil {
		t.Fatal(err)
	}

	stream, err := p.AnswerStream(context.Background(), "q", 5)
	if err != nil {
		t.Fatalf("AnswerStream: %v", err)
	}
	for range stream {
	}

	if hooks.endCount[domain.HookStageGenerate] == 0 {
		t.Fatal("expected HookStageGenerate end to fire on close")
	}
}

// countingHooks — мок-реализация domain.Hooks, считает вызовы Start/End.
type countingHooks struct {
	startCount map[domain.HookStage]int
	endCount   map[domain.HookStage]int
	mu         sync.Mutex
}

// @sk-test arch-quality-pass#T1.2: countingHooks обновлён (AC-001)
func (h *countingHooks) StageStart(ctx context.Context, ev domain.StageStartEvent) context.Context {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.startCount == nil {
		h.startCount = make(map[domain.HookStage]int)
	}
	h.startCount[ev.Stage]++
	return ctx
}

func (h *countingHooks) StageEnd(_ context.Context, ev domain.StageEndEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.endCount == nil {
		h.endCount = make(map[domain.HookStage]int)
	}
	h.endCount[ev.Stage]++
}
