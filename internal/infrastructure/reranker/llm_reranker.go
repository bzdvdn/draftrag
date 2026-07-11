package reranker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task reranker-llm-based#T1.1: defaultJudgePrompt — дефолтный системный промпт (AC-001, AC-002)
const defaultJudgePrompt = `You are a relevance judge. Evaluate each chunk's relevance to the given query.
For each chunk output an integer score from 0 (completely irrelevant) to 10 (perfectly relevant).
Respond with ONLY a JSON array of integers in the same order as the chunks.
Example: [8, 3, 9, 5, 2]`

const defaultBatchSize = 10
const defaultMaxRetries = 1

// @sk-task reranker-llm-based#T1.1: LLMReranker — публичный тип (AC-001, AC-002)
// LLMReranker предоставляет LLM-as-judge zero-shot переранжирование retrieval-результатов.
type LLMReranker struct {
	inner *llmReranker
}

// @sk-task reranker-llm-based#T1.1: llmReranker — внутренняя реализация domain.Reranker (AC-001, AC-002)
type llmReranker struct {
	llm            domain.LLMProvider
	promptTemplate string
	batchSize      int
	maxRetries     int
	usageLLM       domain.UsageAwareLLMProvider
}

// @sk-task reranker-llm-based#T1.2: NewLLMReranker — конструктор с параметрами (AC-001, AC-002)
// NewLLMReranker создаёт LLMReranker.
// llm — обязательный LLMProvider.
// promptTemplate — опциональный system prompt (пустая строка = дефолтный).
// batchSize — количество чанков в одном LLM-вызове (<= 0 = default 10).
// maxRetries — количество повторных попыток при ошибке LLM (< 0 = default 1).
func NewLLMReranker(llm domain.LLMProvider, promptTemplate string, batchSize, maxRetries int) (*LLMReranker, error) {
	if llm == nil {
		return nil, fmt.Errorf("reranker: nil llm")
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	if maxRetries < 0 {
		maxRetries = defaultMaxRetries
	}

	usageLLM, _ := llm.(domain.UsageAwareLLMProvider)
	return &LLMReranker{
		inner: &llmReranker{
			llm:            llm,
			promptTemplate: promptTemplate,
			batchSize:      batchSize,
			maxRetries:     maxRetries,
			usageLLM:       usageLLM,
		},
	}, nil
}

// @sk-task reranker-llm-based#T1.1: Rerank — переранжирование чанков через LLM (AC-001, AC-002, AC-004, AC-005)
func (r *LLMReranker) Rerank(ctx context.Context, query string, chunks []domain.RetrievedChunk) ([]domain.RetrievedChunk, error) {
	return r.inner.Rerank(ctx, query, chunks)
}

// @sk-task reranker-llm-based#T2.2: RerankBatch — batch rerank для multi-query режима (AC-006)
func (r *LLMReranker) RerankBatch(ctx context.Context, queries []string, chunks []domain.RetrievedChunk) ([][]domain.RetrievedChunk, error) {
	return r.inner.RerankBatch(ctx, queries, chunks)
}

// @sk-task reranker-llm-based#T1.1: Rerank — переранжирование чанков через LLM (AC-001, AC-002, AC-004, AC-005)
func (r *llmReranker) Rerank(ctx context.Context, query string, chunks []domain.RetrievedChunk) ([]domain.RetrievedChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return chunks, nil
	}

	scores, err := r.scoreChunks(ctx, query, chunks)
	if err != nil {
		return chunks, nil
	}

	for i := range chunks {
		chunks[i].Score = scores[i]
	}

	sort.SliceStable(chunks, func(i, j int) bool {
		return chunks[i].Score > chunks[j].Score
	})

	return chunks, nil
}

// @sk-task reranker-llm-based#T1.1,T2.3: scoreChunks — batch LLM-скоринг с retry (AC-001, AC-004, AC-005, AC-007)
func (r *llmReranker) scoreChunks(ctx context.Context, query string, chunks []domain.RetrievedChunk) ([]float64, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	type batch struct {
		start int
		end   int
	}
	var batches []batch
	for i := 0; i < len(chunks); i += r.batchSize {
		end := i + r.batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batches = append(batches, batch{start: i, end: end})
	}

	scores := make([]float64, len(chunks))

	for _, b := range batches {
		batchChunks := chunks[b.start:b.end]
		prompt := r.buildPrompt(query, batchChunks)

		var response string
		var lastErr error

		for attempt := 0; attempt <= r.maxRetries; attempt++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			response, lastErr = r.llm.Generate(ctx, prompt.system, prompt.user)
			if lastErr == nil {
				break
			}
		}

		if lastErr != nil {
			return nil, lastErr
		}

		parsed, err := parseScores(response, len(batchChunks))
		if err != nil {
			log.Printf("reranker: failed to parse LLM response: %v; response: %s", err, response)
			for i := range batchChunks {
				scores[b.start+i] = 0
			}
			continue
		}
		for i, s := range parsed {
			scores[b.start+i] = s
		}
	}

	return scores, nil
}

type judgePrompt struct {
	system string
	user   string
}

// @sk-task reranker-llm-based#T1.1,T2.1: buildPrompt — формирование системного и пользовательского промпта (AC-003)
func (r *llmReranker) buildPrompt(query string, chunks []domain.RetrievedChunk) judgePrompt {
	system := r.promptTemplate
	if system == "" {
		system = defaultJudgePrompt
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Query: %s\n\n", query))
	for i, ch := range chunks {
		b.WriteString(fmt.Sprintf("Chunk %d: %s\n", i+1, ch.Chunk.Content))
	}

	return judgePrompt{
		system: system,
		user:   b.String(),
	}
}

// @sk-task reranker-llm-based#T1.1: parseScores — парсинг JSON-массива [0..10] от LLM (AC-001)
func parseScores(response string, expected int) ([]float64, error) {
	response = strings.TrimSpace(response)

	var raw []int
	if err := json.Unmarshal([]byte(response), &raw); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	if len(raw) != expected {
		return nil, fmt.Errorf("expected %d scores, got %d", expected, len(raw))
	}

	scores := make([]float64, len(raw))
	for i, s := range raw {
		if s < 0 {
			s = 0
		}
		if s > 10 {
			s = 10
		}
		scores[i] = float64(s) / 10.0
	}
	return scores, nil
}

// @sk-task reranker-llm-based#T2.2: RerankBatch — batch rerank для multi-query режима (AC-006)
func (r *llmReranker) RerankBatch(ctx context.Context, queries []string, chunks []domain.RetrievedChunk) ([][]domain.RetrievedChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return nil, nil
	}

	results := make([][]domain.RetrievedChunk, len(queries))
	for i, q := range queries {
		chunkCopy := make([]domain.RetrievedChunk, len(chunks))
		copy(chunkCopy, chunks)
		reranked, err := r.Rerank(ctx, q, chunkCopy)
		if err != nil {
			return nil, fmt.Errorf("rerank batch query %d: %w", i, err)
		}
		results[i] = reranked
	}
	return results, nil
}

// Ensure llmReranker implements domain.Reranker and domain.BatchReranker at compile time.
var (
	_ domain.Reranker      = (*llmReranker)(nil)
	_ domain.BatchReranker = (*llmReranker)(nil)
)
