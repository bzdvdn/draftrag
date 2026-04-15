package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Mock implementations для тестирования конструкторов

type mockVectorStore struct{}

func (m *mockVectorStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	return nil
}

func (m *mockVectorStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

type mockLLMProvider struct{}

func (m *mockLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "response", nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2}, nil
}

type testChunker struct{} // используем другое имя чтобы избежать конфликта с batch_test.go

func (m *testChunker) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
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
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil store")
		}
	}()
	NewPipeline(nil, &mockLLMProvider{}, &mockEmbedder{})
}

func TestNewPipeline_NilLLM(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil llm")
		}
	}()
	NewPipeline(&mockVectorStore{}, nil, &mockEmbedder{})
}

func TestNewPipeline_NilEmbedder(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil embedder")
		}
	}()
	NewPipeline(&mockVectorStore{}, &mockLLMProvider{}, nil)
}

func TestNewPipeline_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	if p == nil {
		t.Fatal("expected non-nil pipeline")
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
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	customPrompt := "You are a helpful assistant."
	cfg := PipelineConfig{
		SystemPrompt: customPrompt,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.systemPrompt != customPrompt {
		t.Errorf("expected system prompt %q, got %q", customPrompt, p.systemPrompt)
	}
}

func TestNewPipelineWithConfig_EmptySystemPrompt(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		SystemPrompt: "   ", // только пробелы
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	// Должен остаться дефолтный промпт
	if p.systemPrompt == "" {
		t.Error("system prompt should have default value when config has whitespace only")
	}
}

func TestNewPipelineWithConfig_WithChunker(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}
	chunker := &testChunker{}

	cfg := PipelineConfig{
		Chunker: chunker,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.chunker != chunker {
		t.Error("chunker not set correctly")
	}
}

func TestNewPipelineWithConfig_MaxContextChars(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		MaxContextChars: 5000,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.maxContextChars != 5000 {
		t.Errorf("expected maxContextChars 5000, got %d", p.maxContextChars)
	}
}

func TestNewPipelineWithConfig_MaxContextChunks(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		MaxContextChunks: 10,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.maxContextChunks != 10 {
		t.Errorf("expected maxContextChunks 10, got %d", p.maxContextChunks)
	}
}

func TestNewPipelineWithConfig_DedupByParentID(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		DedupByParentID: true,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if !p.dedupByParentID {
		t.Error("dedupByParentID not set correctly")
	}
}

func TestNewPipelineWithConfig_MMREnabled(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		MMREnabled: true,
		MMRLambda:  0.7,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if !p.mmrEnabled {
		t.Error("mmrEnabled not set correctly")
	}
	if p.mmrLambda != 0.7 {
		t.Errorf("expected mmrLambda 0.7, got %f", p.mmrLambda)
	}
}

func TestNewPipelineWithConfig_MMREnabled_DefaultLambda(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		MMREnabled: true,
		// MMRLambda = 0, должен стать 0.5
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.mmrLambda != 0.5 {
		t.Errorf("expected default mmrLambda 0.5, got %f", p.mmrLambda)
	}
}

func TestNewPipelineWithConfig_MMRCandidatePool(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		MMRCandidatePool: 20,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.mmrCandidatePool != 20 {
		t.Errorf("expected mmrCandidatePool 20, got %d", p.mmrCandidatePool)
	}
}

func TestNewPipelineWithConfig_WithHooks(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	hooks := &mockHooks{}
	cfg := PipelineConfig{
		Hooks: hooks,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.hooks != hooks {
		t.Error("hooks not set correctly")
	}
}

func TestNewPipelineWithConfig_IndexConcurrency(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		IndexConcurrency: 5,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.indexConcurrency != 5 {
		t.Errorf("expected indexConcurrency 5, got %d", p.indexConcurrency)
	}
}

func TestNewPipelineWithConfig_IndexConcurrency_Default(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		IndexConcurrency: 0, // должно стать дефолтным
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.indexConcurrency == 0 {
		t.Error("indexConcurrency should have default value when set to 0")
	}
}

func TestNewPipelineWithConfig_IndexBatchRateLimit(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	cfg := PipelineConfig{
		IndexBatchRateLimit: 100,
	}

	p := NewPipelineWithConfig(store, llm, embedder, cfg)

	if p.indexBatchRateLimit != 100 {
		t.Errorf("expected indexBatchRateLimit 100, got %d", p.indexBatchRateLimit)
	}
}

type mockHooks struct{}

func (m *mockHooks) StageStart(ctx context.Context, ev domain.StageStartEvent) {
	// no-op
}

func (m *mockHooks) StageEnd(ctx context.Context, ev domain.StageEndEvent) {
	// no-op
}
