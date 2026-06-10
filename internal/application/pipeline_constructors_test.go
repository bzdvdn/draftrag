package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Mock implementations для тестирования конструкторов

type mockVectorStore struct{}

func (m *mockVectorStore) Upsert(_ context.Context, _ domain.Chunk) error {
	return nil
}

func (m *mockVectorStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockVectorStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

type mockLLMProvider struct{}

func (m *mockLLMProvider) Generate(_ context.Context, _, _ string) (string, error) {
	return "response", nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{0.1, 0.2}, nil
}

type testChunker struct{} // используем другое имя чтобы избежать конфликта с batch_test.go

func (m *testChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return []domain.Chunk{
		{
			ID:       "c1",
			Content:  doc.Content,
			ParentID: doc.ID,
			Position: 0,
		},
	}, nil
}

func TestNewPipeline_NilStore(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	_, err := NewPipeline(nil, &mockLLMProvider{}, &mockEmbedder{})
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestNewPipeline_NilLLM(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	_, err := NewPipeline(&mockVectorStore{}, nil, &mockEmbedder{})
	if err == nil {
		t.Error("expected error for nil llm")
	}
}

func TestNewPipeline_NilEmbedder(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	_, err := NewPipeline(&mockVectorStore{}, &mockLLMProvider{}, nil)
	if err == nil {
		t.Error("expected error for nil embedder")
	}
}

func TestNewPipeline_Success(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	if p.store != store {
		t.Error("store not set correctly")
	}
	if p.llm != llm {
		t.Error("llm not set correctly")
	}
	if p.embedder != embedder {
		t.Error("embedder not set correctly")
	}
	if p.systemPrompt == "" {
		t.Error("system prompt should have default value")
	}
	if p.chunker != nil {
		t.Error("chunker should be nil by default")
	}
}

func TestNewPipelineWithConfig_CustomSystemPrompt(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	customPrompt := "You are a helpful assistant."
	cfg := PipelineOptions{
		SystemPrompt: customPrompt,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.systemPrompt != customPrompt {
		t.Errorf("expected system prompt %q, got %q", customPrompt, p.systemPrompt)
	}
}

func TestNewPipelineWithConfig_EmptySystemPrompt(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		SystemPrompt: "   ", // только пробелы
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Должен остаться дефолтный промпт
	if p.systemPrompt == "" {
		t.Error("system prompt should have default value when config has whitespace only")
	}
}

func TestNewPipelineWithConfig_WithChunker(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}
	chunker := &testChunker{}

	cfg := PipelineOptions{
		Chunker: chunker,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.chunker != chunker {
		t.Error("chunker not set correctly")
	}
}

func TestNewPipelineWithConfig_MaxContextChars(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		MaxContextChars: 5000,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.maxContextChars != 5000 {
		t.Errorf("expected maxContextChars 5000, got %d", p.maxContextChars)
	}
}

func TestNewPipelineWithConfig_MaxContextChunks(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		MaxContextChunks: 10,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.maxContextChunks != 10 {
		t.Errorf("expected maxContextChunks 10, got %d", p.maxContextChunks)
	}
}

func TestNewPipelineWithConfig_DedupByParentID(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		DedupByParentID: true,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !p.dedupByParentID {
		t.Error("dedupByParentID not set correctly")
	}
}

func TestNewPipelineWithConfig_MMREnabled(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		MMREnabled: true,
		MMRLambda:  0.7,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !p.mmrEnabled {
		t.Error("mmrEnabled not set correctly")
	}
	if p.mmrLambda != 0.7 {
		t.Errorf("expected mmrLambda 0.7, got %f", p.mmrLambda)
	}
}

func TestNewPipelineWithConfig_MMREnabled_DefaultLambda(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		MMREnabled: true,
		// MMRLambda = 0, должен стать 0.5
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.mmrLambda != 0.5 {
		t.Errorf("expected default mmrLambda 0.5, got %f", p.mmrLambda)
	}
}

func TestNewPipelineWithConfig_MMRCandidatePool(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		MMRCandidatePool: 20,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.mmrCandidatePool != 20 {
		t.Errorf("expected mmrCandidatePool 20, got %d", p.mmrCandidatePool)
	}
}

func TestNewPipelineWithConfig_WithHooks(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	hooks := &mockHooks{}
	cfg := PipelineOptions{
		Hooks: hooks,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.hooks != hooks {
		t.Error("hooks not set correctly")
	}
}

func TestNewPipelineWithConfig_IndexConcurrency(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		IndexConcurrency: 5,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.indexConcurrency != 5 {
		t.Errorf("expected indexConcurrency 5, got %d", p.indexConcurrency)
	}
}

func TestNewPipelineWithConfig_IndexConcurrency_Default(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		IndexConcurrency: 0, // должно стать дефолтным
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.indexConcurrency == 0 {
		t.Error("indexConcurrency should have default value when set to 0")
	}
}

func TestNewPipelineWithConfig_IndexBatchRateLimit(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineOptions{
		IndexBatchRateLimit: 100,
	}

	p, err := NewPipelineWithConfig(store, llm, embedder, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if p.indexBatchRateLimit != 100 {
		t.Errorf("expected indexBatchRateLimit 100, got %d", p.indexBatchRateLimit)
	}
}

// @sk-test arch-quality-pass#T1.2: mockHooks обновлён под новый контракт (AC-001)
type mockHooks struct{}

func (m *mockHooks) StageStart(ctx context.Context, _ domain.StageStartEvent) context.Context {
	return ctx
}

func (m *mockHooks) StageEnd(_ context.Context, _ domain.StageEndEvent) {
	// no-op
}
