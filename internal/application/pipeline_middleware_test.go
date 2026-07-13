package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// --- AC-001: Middleware chain executes in declaration order ---

// recordPre appends label on pre (entry) only.
func recordPre(label string, order *[]string) domain.Middleware {
	return func(next domain.Handler) domain.Handler {
		return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
			*order = append(*order, label)
			return next(ctx, data)
		}
	}
}

// recordBoth appends label on pre and post.
func recordBoth(label string, pre, post *[]string) domain.Middleware {
	return func(next domain.Handler) domain.Handler {
		return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
			*pre = append(*pre, label)
			result, err := next(ctx, data)
			*post = append(*post, label)
			return result, err
		}
	}
}

func identityHandler(_ context.Context, data domain.StageData) (domain.StageData, error) {
	return data, nil
}

// @sk-test middleware-chain#T4.1: AC-001 order via runMiddleware (AC-001)
func TestMiddleware_RunMiddleware_Order(t *testing.T) {
	var order []string
	mw := []domain.Middleware{
		recordPre("A", &order),
		recordPre("B", &order),
		recordPre("C", &order),
	}

	_, err := runMiddleware(context.Background(), mw, domain.StageData{Stage: domain.HookStageGenerate}, identityHandler)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"A", "B", "C"}
	if !equalSlices(order, want) {
		t.Errorf("order=%v, want=%v", order, want)
	}
}

// @sk-test middleware-chain#T4.1: AC-001 order via Index (AC-001)
func TestMiddleware_Order_Index(t *testing.T) {
	var order []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				recordPre("A", &order),
				recordPre("B", &order),
				recordPre("C", &order),
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Index(context.Background(), []domain.Document{{ID: "d1", Content: "test"}}); err != nil {
		t.Fatal(err)
	}
	// Index: chunking stage (pre A B C) + embed stage (pre A B C)
	if len(order) == 0 {
		t.Fatal("middleware was never called")
	}
	// Verify first three calls are A, B, C
	firstThree := order[:3]
	want := []string{"A", "B", "C"}
	if !equalSlices(firstThree, want) {
		t.Errorf("first three order=%v, want=%v", firstThree, want)
	}
}

// @sk-test middleware-chain#T4.1: AC-001 order via Answer (AC-001)
func TestMiddleware_Order_Answer(t *testing.T) {
	var order []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				recordPre("A", &order),
				recordPre("B", &order),
				recordPre("C", &order),
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "test question", 5)
	if err != nil {
		t.Fatal(err)
	}
	// Answer: embed (pre A B C) + search (pre A B C) + generate (pre A B C)
	if len(order) == 0 {
		t.Fatal("middleware was never called")
	}
	firstThree := order[:3]
	want := []string{"A", "B", "C"}
	if !equalSlices(firstThree, want) {
		t.Errorf("first three order=%v, want=%v", firstThree, want)
	}
}

// @sk-test middleware-chain#T4.1: AC-001 pre + post order via runMiddleware (AC-001)
func TestMiddleware_RunMiddleware_PrePostOrder(t *testing.T) {
	var pre, post []string
	mw := []domain.Middleware{
		recordBoth("A", &pre, &post),
		recordBoth("B", &pre, &post),
		recordBoth("C", &pre, &post),
	}

	_, err := runMiddleware(context.Background(), mw, domain.StageData{Stage: domain.HookStageGenerate}, identityHandler)
	if err != nil {
		t.Fatal(err)
	}

	wantPre := []string{"A", "B", "C"}
	wantPost := []string{"C", "B", "A"}
	if !equalSlices(pre, wantPre) {
		t.Errorf("pre order=%v, want=%v", pre, wantPre)
	}
	if !equalSlices(post, wantPost) {
		t.Errorf("post order=%v, want=%v", post, wantPost)
	}
}

// --- AC-002: Error in middleware aborts pipeline ---

var errTestGuardrail = errors.New("guardrail rejected")

// @sk-test middleware-chain#T4.1: AC-002 error abort via Answer (AC-002)
func TestMiddleware_ErrorAbort_Answer(t *testing.T) {
	store := &spyStore{}
	llm := &spyLLM{}
	p, err := NewPipelineWithConfig(
		store, llm, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				func(_ domain.Handler) domain.Handler {
					return func(_ context.Context, data domain.StageData) (domain.StageData, error) {
						return data, errTestGuardrail
					}
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "test question", 5)
	if !errors.Is(err, errTestGuardrail) {
		t.Fatalf("expected errTestGuardrail, got %v", err)
	}
	if llm.called {
		t.Error("LLM should not be called after middleware error")
	}
}

// spyStore records whether Search/Upsert were called.
type spyStore struct {
	mockVectorStore
	searched bool
}

func (s *spyStore) Search(_ context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	s.searched = true
	return domain.RetrievalResult{}, nil
}

// spyLLM records the last userMessage and whether Generate was called.
type spyLLM struct {
	called      bool
	userMessage string
}

func (m *spyLLM) Health(_ context.Context) error { return nil }
func (m *spyLLM) Generate(_ context.Context, _, userMessage string) (string, error) {
	m.called = true
	m.userMessage = userMessage
	return "response", nil
}

// spyStreamingLLM records the last userMessage.
type spyStreamingLLM struct {
	spyLLM
}

func (m *spyStreamingLLM) GenerateStream(_ context.Context, _, userMessage string) (<-chan string, error) {
	m.called = true
	m.userMessage = userMessage
	ch := make(chan string, 1)
	ch <- "response"
	close(ch)
	return ch, nil
}

// @sk-test middleware-chain#T4.1: AC-002 error abort — downstream stage not called (AC-002)
func TestMiddleware_ErrorAbort_DownstreamStageSkipped(t *testing.T) {
	store := &spyStore{}
	llm := &spyLLM{}
	// Middleware that aborts on pre-embed — search should not execute
	abortOnEmbed := func(next domain.Handler) domain.Handler {
		return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
			if data.Stage == domain.HookStageEmbed {
				return data, errTestGuardrail
			}
			return next(ctx, data)
		}
	}

	p, err := NewPipelineWithConfig(
		store, llm, &mockEmbedder{},
		PipelineOptions{Middleware: []domain.Middleware{abortOnEmbed}},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "test question", 5)
	if !errors.Is(err, errTestGuardrail) {
		t.Fatalf("expected errTestGuardrail, got %v", err)
	}
	if store.searched {
		t.Error("Search should not be called after embed middleware abort")
	}
	if llm.called {
		t.Error("LLM should not be called after embed middleware abort")
	}
}

// --- AC-003: Middleware called on all HookStage stages ---

// stageRecorder records each HookStage value it observes on pre.
func stageRecorder(stages *[]string) domain.Middleware {
	return func(next domain.Handler) domain.Handler {
		return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
			*stages = append(*stages, string(data.Stage))
			return next(ctx, data)
		}
	}
}

// @sk-test middleware-chain#T4.2: AC-003 all stages via Answer (AC-003)
func TestMiddleware_Stages_Answer(t *testing.T) {
	var stages []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{stageRecorder(&stages)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "test question", 5)
	if err != nil {
		t.Fatal(err)
	}

	// Answer: embed, search, generate
	expected := []string{"embed", "search", "generate"}
	for _, s := range expected {
		found := false
		for _, seen := range stages {
			if seen == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stage %q not seen in middleware stages=%v", s, stages)
		}
	}
}

// @sk-test middleware-chain#T4.2: AC-003 all stages via Index (AC-003)
func TestMiddleware_Stages_Index(t *testing.T) {
	var stages []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Chunker:    &testChunker{},
			Middleware: []domain.Middleware{stageRecorder(&stages)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Index(context.Background(), []domain.Document{{ID: "d1", Content: "test"}}); err != nil {
		t.Fatal(err)
	}

	expected := []string{"chunking", "embed"}
	for _, s := range expected {
		found := false
		for _, seen := range stages {
			if seen == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stage %q not seen in middleware stages=%v", s, stages)
		}
	}
}

// @sk-test middleware-chain#T4.2: AC-003 all stages via AnswerStream (AC-003)
func TestMiddleware_Stages_AnswerStream(t *testing.T) {
	var stages []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &spyStreamingLLM{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{stageRecorder(&stages)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ch, err := p.AnswerStream(context.Background(), "test question", 5)
	if err != nil {
		t.Fatal(err)
	}
	for range ch { // drain
		_ = 1
	}

	// AnswerStream: embed, search, generate
	expected := []string{"embed", "search", "generate"}
	for _, s := range expected {
		found := false
		for _, seen := range stages {
			if seen == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stage %q not seen in middleware stages=%v", s, stages)
		}
	}
}

// @sk-test middleware-chain#T-concern: AC-003 all stages via AnswerStreamWithInlineCitations (AC-003)
func TestMiddleware_Stages_AnswerStreamWithInlineCitations(t *testing.T) {
	var stages []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &spyStreamingLLM{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{stageRecorder(&stages)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ch, _, _, err := p.AnswerStreamWithInlineCitations(context.Background(), "test question", 5)
	if err != nil {
		t.Fatal(err)
	}
	for range ch { // drain
		_ = 1
	}

	expected := []string{"embed", "search", "generate"}
	for _, s := range expected {
		found := false
		for _, seen := range stages {
			if seen == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stage %q not seen in middleware stages=%v", s, stages)
		}
	}
}

// @sk-test middleware-chain#T-concern: AC-003 all stages via AnswerStreamWithSources (streamFromResult) (AC-003)
func TestMiddleware_Stages_AnswerStreamWithSources(t *testing.T) {
	var stages []string
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &spyStreamingLLM{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{stageRecorder(&stages)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ch, _, err := p.AnswerStreamWithSources(context.Background(), "test question", 5)
	if err != nil {
		t.Fatal(err)
	}
	for range ch { // drain
		_ = 1
	}

	expected := []string{"embed", "search", "generate"}
	for _, s := range expected {
		found := false
		for _, seen := range stages {
			if seen == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stage %q not seen in middleware stages=%v", s, stages)
		}
	}
}

// --- AC-004: Middleware modifies in-flight data ---

// @sk-test middleware-chain#T4.2: AC-004 modify query on pre-generate via Answer (AC-004)
func TestMiddleware_ModifyQuery_Answer(t *testing.T) {
	llm := &spyLLM{}
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, llm, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						if data.Stage == domain.HookStageGenerate {
							data.Query = "redacted question"
						}
						return next(ctx, data)
					}
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "original question", 5)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(llm.userMessage, "redacted question") {
		t.Errorf("expected userMessage to contain 'redacted question', got %q", llm.userMessage)
	}
	if strings.Contains(llm.userMessage, "original question") {
		t.Errorf("expected userMessage NOT to contain 'original question', got %q", llm.userMessage)
	}
}

// @sk-test middleware-chain#T4.2: AC-004 modify query on pre-generate via AnswerStream (AC-004)
func TestMiddleware_ModifyQuery_AnswerStream(t *testing.T) {
	llm := &spyStreamingLLM{}
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, llm, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						if data.Stage == domain.HookStageGenerate {
							data.Query = "redacted stream question"
						}
						return next(ctx, data)
					}
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ch, err := p.AnswerStream(context.Background(), "original stream question", 5)
	if err != nil {
		t.Fatal(err)
	}
	for range ch { // drain
		_ = 1
	}

	if !strings.Contains(llm.userMessage, "redacted stream question") {
		t.Errorf("expected userMessage to contain 'redacted stream question', got %q", llm.userMessage)
	}
}

// @sk-test middleware-chain#T4.2: AC-004 modify answer post-generate (AC-004)
func TestMiddleware_ModifyAnswerPostGenerate(t *testing.T) {
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						result, err := next(ctx, data)
						if err != nil {
							return result, err
						}
						if data.Stage == domain.HookStageGenerate {
							result.Answer += " [post-processed]"
						}
						return result, nil
					}
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	answer, err := p.Answer(context.Background(), "test", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(answer, " [post-processed]") {
		t.Errorf("expected answer suffix [post-processed], got %q", answer)
	}
}

// --- AC-005: No-op with nil/empty middleware ---

// @sk-test middleware-chain#T4.1: AC-005 nil middleware slice (AC-005)
func TestMiddleware_NilSlice(t *testing.T) {
	p1, err := NewPipeline(&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	p2, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{Middleware: nil},
	)
	if err != nil {
		t.Fatal(err)
	}

	p3, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{Middleware: []domain.Middleware{}},
	)
	if err != nil {
		t.Fatal(err)
	}

	// All three should produce identical results
	ctx := context.Background()
	doc := []domain.Document{{ID: "d1", Content: "test"}}

	for _, p := range []*Pipeline{p1, p2, p3} {
		if err := p.Index(ctx, doc); err != nil {
			t.Fatal(err)
		}
		answer, err := p.Answer(ctx, "test", 5)
		if err != nil {
			t.Fatal(err)
		}
		if answer != "response" {
			t.Errorf("expected 'response', got %q", answer)
		}
	}
}

// @sk-test middleware-chain#T4.2: AC-005 empty middleware slice (AC-005)
func TestMiddleware_EmptySlice_Answer(t *testing.T) {
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{Middleware: []domain.Middleware{}},
	)
	if err != nil {
		t.Fatal(err)
	}

	answer, err := p.Answer(context.Background(), "test", 5)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "response" {
		t.Errorf("expected 'response', got %q", answer)
	}
}

// --- Edge cases ---

// @sk-test middleware-chain#T4.2: context cancel propagates through middleware
func TestMiddleware_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						return next(ctx, data)
					}
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(ctx, "test", 5)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// --- Benchmark SC-001: 3 no-op middleware < 5% latency overhead ---

// @sk-test middleware-chain#T4.3: SC-001 baseline benchmark (no middleware)
func BenchmarkAnswer_NoMiddleware(b *testing.B) {
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{},
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Answer(ctx, "test", 5)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// @sk-test middleware-chain#T4.3: SC-001 benchmark (3 no-op middleware)
func BenchmarkMiddleware_NoOp(b *testing.B) {
	p, err := NewPipelineWithConfig(
		&mockVectorStore{}, &mockLLMProvider{}, &mockEmbedder{},
		PipelineOptions{
			Middleware: []domain.Middleware{
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						return next(ctx, data)
					}
				},
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						return next(ctx, data)
					}
				},
				func(next domain.Handler) domain.Handler {
					return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
						return next(ctx, data)
					}
				},
			},
		},
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Answer(ctx, "test", 5)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- helpers ---

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
