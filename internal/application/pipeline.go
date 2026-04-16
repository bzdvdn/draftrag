package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

const defaultSystemPromptV1 = "Ты — помощник. Отвечай на вопрос, используя предоставленный контекст. Если контекста недостаточно — честно скажи, что информации недостаточно."

var (
	// ErrFiltersNotSupported возвращается, если pipeline-метод с фильтрами вызван,
	// но underlying VectorStore не поддерживает filters capability.
	ErrFiltersNotSupported = errors.New("vector store does not support filters")

	// ErrStreamingNotSupported возвращается, если streaming-метод вызван,
	// но underlying LLMProvider не поддерживает StreamingLLMProvider capability.
	ErrStreamingNotSupported = errors.New("LLM provider does not support streaming")

	// ErrDeleteNotSupported возвращается, если DeleteDocument вызван,
	// но underlying VectorStore не реализует DocumentStore capability.
	ErrDeleteNotSupported = errors.New("vector store does not support DeleteByParentID")
)

// PipelineConfig задаёт опциональную конфигурацию application use-case Pipeline.
type PipelineConfig struct {
	// SystemPrompt переопределяет system prompt для Answer*. Пустая строка означает дефолт v1.
	SystemPrompt string
	// Chunker включает чанкинг при Index, если не nil.
	Chunker domain.Chunker
	// MaxContextChars ограничивает размер секции “Контекст:” в prompt для Answer* (в символах).
	// 0 означает “без лимита”.
	MaxContextChars int
	// MaxContextChunks ограничивает количество чанков в секции “Контекст:” в prompt для Answer*.
	// 0 означает “без лимита”.
	MaxContextChunks int

	// DedupByParentID включает дедупликацию retrieval результата по ParentID.
	// По умолчанию выключено.
	DedupByParentID bool

	// MMREnabled включает MMR selection поверх retrieval кандидатов.
	// По умолчанию выключено (backward compatibility).
	MMREnabled bool
	// MMRLambda задаёт баланс релевантность/разнообразие в диапазоне [0..1].
	// Если 0 и MMR включён — используется значение по умолчанию (0.5).
	MMRLambda float64
	// MMRCandidatePool задаёт сколько кандидатов запросить у VectorStore до MMR selection.
	// Если 0 — используется topK запроса.
	MMRCandidatePool int

	// Hooks — опциональные хуки наблюдаемости для стадий pipeline.
	// Если nil — no-op.
	Hooks domain.Hooks

	// IndexConcurrency задаёт количество workers для параллельной индексации в IndexBatch.
	// 0 или отрицательное значение означает "использовать default" (4).
	//
	// @ds-task T1.2: Добавить поле IndexConcurrency в PipelineConfig (AC-001, DEC-001)
	IndexConcurrency int

	// IndexBatchRateLimit задаёт максимальное количество вызовов Embed в секунду для IndexBatch.
	// 0 или отрицательное значение означает "использовать default" (10).
	//
	// @ds-task T1.2: Добавить поле IndexBatchRateLimit в PipelineConfig (AC-002, DEC-002)
	IndexBatchRateLimit int

	// Reranker — опциональный reranker, применяется после retrieval и dedup.
	// nil означает "без reranking".
	Reranker domain.Reranker
}

// Pipeline — application use-case, который композирует зависимости (VectorStore, Embedder, LLMProvider)
// для построения базового RAG-пайплайна.
type Pipeline struct {
	store               domain.VectorStore
	llm                 domain.LLMProvider
	embedder            domain.Embedder
	chunker             domain.Chunker
	systemPrompt        string
	maxContextChars     int
	maxContextChunks    int
	dedupByParentID     bool
	mmrEnabled          bool
	mmrLambda           float64
	mmrCandidatePool    int
	hooks               domain.Hooks
	indexConcurrency    int
	indexBatchRateLimit int
	reranker            domain.Reranker
}

// NewPipeline создаёт новый use-case Pipeline.
func NewPipeline(store domain.VectorStore, llm domain.LLMProvider, embedder domain.Embedder) *Pipeline {
	if store == nil {
		panic("nil store")
	}
	if llm == nil {
		panic("nil llm")
	}
	if embedder == nil {
		panic("nil embedder")
	}

	return &Pipeline{
		store:        store,
		llm:          llm,
		embedder:     embedder,
		chunker:      nil,
		systemPrompt: defaultSystemPromptV1,
	}
}

// NewPipelineWithConfig создаёт новый use-case Pipeline с конфигурацией.
func NewPipelineWithConfig(
	store domain.VectorStore,
	llm domain.LLMProvider,
	embedder domain.Embedder,
	cfg PipelineConfig,
) *Pipeline {
	p := NewPipeline(store, llm, embedder)
	if strings.TrimSpace(cfg.SystemPrompt) != "" {
		p.systemPrompt = cfg.SystemPrompt
	}
	p.chunker = cfg.Chunker
	p.maxContextChars = cfg.MaxContextChars
	p.maxContextChunks = cfg.MaxContextChunks
	p.dedupByParentID = cfg.DedupByParentID
	p.mmrEnabled = cfg.MMREnabled
	p.mmrLambda = cfg.MMRLambda
	if p.mmrEnabled && p.mmrLambda == 0 {
		p.mmrLambda = 0.5
	}
	p.mmrCandidatePool = cfg.MMRCandidatePool
	p.hooks = cfg.Hooks
	p.indexConcurrency = cfg.IndexConcurrency
	if p.indexConcurrency <= 0 {
		p.indexConcurrency = 4 // default
	}
	p.indexBatchRateLimit = cfg.IndexBatchRateLimit
	if p.indexBatchRateLimit <= 0 {
		p.indexBatchRateLimit = 10 // default
	}
	p.reranker = cfg.Reranker
	return p
}

func (p *Pipeline) hookStart(ctx context.Context, op string, stage domain.HookStage) {
	if p.hooks == nil {
		return
	}
	p.hooks.StageStart(ctx, domain.StageStartEvent{
		Operation: op,
		Stage:     stage,
	})
}

func (p *Pipeline) hookEnd(ctx context.Context, op string, stage domain.HookStage, started time.Time, err error) {
	if p.hooks == nil {
		return
	}
	p.hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: op,
		Stage:     stage,
		Duration:  time.Since(started),
		Err:       err,
	})
}

func (p *Pipeline) maybeRerank(ctx context.Context, query string, result domain.RetrievalResult) (domain.RetrievalResult, error) {
	if p.reranker == nil {
		return result, nil
	}
	reranked, err := p.reranker.Rerank(ctx, query, result.Chunks)
	if err != nil {
		return result, fmt.Errorf("reranker: %w", err)
	}
	result.Chunks = reranked
	return result, nil
}

func (p *Pipeline) maybeDedup(result domain.RetrievalResult) domain.RetrievalResult {
	if !p.dedupByParentID {
		return result
	}
	result.Chunks = dedupRetrievedChunksByParentID(result.Chunks)
	return result
}

func dedupRetrievedChunksByParentID(chunks []domain.RetrievedChunk) []domain.RetrievedChunk {
	if len(chunks) == 0 {
		return chunks
	}

	type best struct {
		chunk domain.RetrievedChunk
		ix    int
	}

	bestByParent := make(map[string]best, len(chunks))
	for i, rc := range chunks {
		parentID := rc.Chunk.ParentID
		prev, ok := bestByParent[parentID]
		if !ok {
			bestByParent[parentID] = best{chunk: rc, ix: i}
			continue
		}

		// Выбираем лучший по score; при равенстве оставляем более ранний (детерминизм).
		if rc.Score > prev.chunk.Score {
			bestByParent[parentID] = best{chunk: rc, ix: i}
		}
	}

	out := make([]best, 0, len(bestByParent))
	for _, v := range bestByParent {
		out = append(out, v)
	}

	// Порядок по релевантности: score desc, tie-breaker — исходный индекс (stable).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].chunk.Score == out[j].chunk.Score {
			return out[i].ix < out[j].ix
		}
		return out[i].chunk.Score > out[j].chunk.Score
	})

	deduped := make([]domain.RetrievedChunk, 0, len(out))
	for _, v := range out {
		deduped = append(deduped, v.chunk)
	}
	return deduped
}

// NewPipelineWithChunker создаёт новый use-case Pipeline с Chunker для индексирования.
func NewPipelineWithChunker(
	store domain.VectorStore,
	llm domain.LLMProvider,
	embedder domain.Embedder,
	chunker domain.Chunker,
) *Pipeline {
	if chunker == nil {
		panic("nil chunker")
	}

	return NewPipelineWithConfig(store, llm, embedder, PipelineConfig{Chunker: chunker})
}

func (p *Pipeline) indexChunks(ctx context.Context, operation string, chunks []domain.Chunk) error {
	for _, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return err
		}

		embedStart := time.Now()
		p.hookStart(ctx, operation, domain.HookStageEmbed)
		embedding, err := p.embedder.Embed(ctx, chunk.Content)
		p.hookEnd(ctx, operation, domain.HookStageEmbed, embedStart, err)
		if err != nil {
			return err
		}
		chunk.Embedding = embedding

		if err := chunk.Validate(); err != nil {
			return err
		}
		if err := p.store.Upsert(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

// Index индексирует документы (v1: один чанк на документ).
func (p *Pipeline) Index(ctx context.Context, docs []domain.Document) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, doc := range docs {
		if err := doc.Validate(); err != nil {
			return err
		}

		if p.chunker != nil {
			chunkStart := time.Now()
			p.hookStart(ctx, "Index", domain.HookStageChunking)
			chunks, err := p.chunker.Chunk(ctx, doc)
			p.hookEnd(ctx, "Index", domain.HookStageChunking, chunkStart, err)
			if err != nil {
				return err
			}
			if err := p.indexChunks(ctx, "Index", chunks); err != nil {
				return err
			}
			continue
		}

		// Legacy path: один чанк на документ (backward compatibility).
		embedStart := time.Now()
		p.hookStart(ctx, "Index", domain.HookStageEmbed)
		embedding, err := p.embedder.Embed(ctx, doc.Content)
		p.hookEnd(ctx, "Index", domain.HookStageEmbed, embedStart, err)
		if err != nil {
			return err
		}

		chunk := domain.Chunk{
			ID:        fmt.Sprintf("%s#%d", doc.ID, 0),
			Content:   doc.Content,
			ParentID:  doc.ID,
			Embedding: embedding,
			Position:  0,
		}
		if err := chunk.Validate(); err != nil {
			return err
		}

		if err := p.store.Upsert(ctx, chunk); err != nil {
			return err
		}
	}

	return nil
}

// IndexBatch индексирует документы параллельно с ограничением concurrency и rate limiting.
//
// @ds-task T2.1: Реализовать IndexBatch с worker pool (AC-001, AC-003, AC-004, AC-005)
// @ds-task T2.2: Добавить rate limiter в IndexBatch (AC-002)
func (p *Pipeline) IndexBatch(ctx context.Context, docs []domain.Document, batchSize int) (*domain.IndexBatchResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// batchSize в публичном API трактуем как “сколько документов обрабатывать параллельно”.
	// Если не задан — используем дефолт pipeline.
	concurrency := p.indexConcurrency
	if batchSize > 0 {
		concurrency = batchSize
	}

	result := &domain.IndexBatchResult{
		Successful: make([]domain.Document, 0, len(docs)),
		Errors:     make([]domain.IndexBatchError, 0),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Семафор для ограничения concurrency
	semaphore := make(chan struct{}, concurrency)

	// Rate limiter: token bucket
	// @ds-task T2.2: Token bucket rate limiter для вызовов Embed (AC-002, DEC-002)
	rateLimitInterval := time.Second / time.Duration(p.indexBatchRateLimit)
	rateLimiter := time.NewTicker(rateLimitInterval)
	defer rateLimiter.Stop()

	// Канал для graceful shutdown при ошибке контекста
	done := make(chan struct{})
	defer close(done)

	// Отслеживаем была ли отмена контекста
	var ctxErr error
	var ctxMu sync.Mutex

	for _, doc := range docs {
		// Проверка отмены контекста перед обработкой каждого документа
		if err := ctx.Err(); err != nil {
			mu.Lock()
			result.ProcessedCount = len(result.Successful) + len(result.Errors)
			mu.Unlock()
			return result, err
		}

		// Проверка валидности документа
		if err := doc.Validate(); err != nil {
			mu.Lock()
			result.Errors = append(result.Errors, domain.IndexBatchError{
				DocumentID: doc.ID,
				Error:      err,
			})
			result.ProcessedCount++
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(d domain.Document) {
			defer wg.Done()

			// Ожидание семафора для ограничения concurrency
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Проверка отмены контекста внутри worker'а
			if err := ctx.Err(); err != nil {
				ctxMu.Lock()
				if ctxErr == nil {
					ctxErr = err
				}
				ctxMu.Unlock()
				mu.Lock()
				result.Errors = append(result.Errors, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      err,
				})
				result.ProcessedCount++
				mu.Unlock()
				return
			}

			// Rate limiting: ожидание токена
			select {
			case <-rateLimiter.C:
				// продолжаем
			case <-ctx.Done():
				err := ctx.Err()
				ctxMu.Lock()
				if ctxErr == nil {
					ctxErr = err
				}
				ctxMu.Unlock()
				mu.Lock()
				result.Errors = append(result.Errors, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      err,
				})
				result.ProcessedCount++
				mu.Unlock()
				return
			}

			// Обработка документа с chunking и embedding
			docErr := p.processDocumentForBatch(ctx, d)

			mu.Lock()
			if docErr != nil {
				result.Errors = append(result.Errors, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      docErr,
				})
			} else {
				result.Successful = append(result.Successful, d)
			}
			result.ProcessedCount++
			mu.Unlock()
		}(doc)
	}

	wg.Wait()

	// Если был отменён контекст, возвращаем ошибку с partial results
	if ctxErr != nil {
		return result, ctxErr
	}

	return result, nil
}

// DeleteDocument удаляет документ и все его чанки по ParentID.
// Требует, чтобы VectorStore реализовывал domain.DocumentStore.
// Если store не поддерживает — возвращает ErrDeleteNotSupported.
func (p *Pipeline) DeleteDocument(ctx context.Context, docID string) error {
	ds, ok := p.store.(domain.DocumentStore)
	if !ok {
		return ErrDeleteNotSupported
	}
	return ds.DeleteByParentID(ctx, docID)
}

// UpdateDocument удаляет старые чанки документа и переиндексирует его.
// Требует DocumentStore capability (аналогично DeleteDocument).
func (p *Pipeline) UpdateDocument(ctx context.Context, doc domain.Document) error {
	if err := p.DeleteDocument(ctx, doc.ID); err != nil {
		return err
	}
	return p.Index(ctx, []domain.Document{doc})
}

const hydeSystemPrompt = "You are a helpful assistant. Write a short factual passage that would directly answer the question. Write only the passage."

// QueryHyDE выполняет поиск с использованием Hypothetical Document Embeddings.
// Сначала LLM генерирует гипотетический ответ на вопрос, затем ищем по его embedding.
func (p *Pipeline) QueryHyDE(ctx context.Context, question string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	genStart := time.Now()
	p.hookStart(ctx, "QueryHyDE:generate", domain.HookStageGenerate)
	hypothetical, err := p.llm.Generate(ctx, hydeSystemPrompt, question)
	p.hookEnd(ctx, "QueryHyDE:generate", domain.HookStageGenerate, genStart, err)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("hyde generate: %w", err)
	}

	embedStart := time.Now()
	p.hookStart(ctx, "QueryHyDE:embed", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, hypothetical)
	p.hookEnd(ctx, "QueryHyDE:embed", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("hyde embed: %w", err)
	}

	searchStart := time.Now()
	p.hookStart(ctx, "QueryHyDE:search", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, topK)
	p.hookEnd(ctx, "QueryHyDE:search", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question
	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	return result, nil
}

// AnswerHyDE генерирует ответ, используя HyDE для retrieval.
func (p *Pipeline) AnswerHyDE(ctx context.Context, question string, topK int) (string, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return "", err
	}
	return p.generateAnswer(ctx, question, result)
}

const multiQuerySystemPrompt = "You are a helpful assistant. Generate alternative phrasings of the given question to improve document retrieval. Output only the questions, one per line, no numbering, no extra text."

// QueryMulti выполняет multi-query retrieval: генерирует n перефразировок вопроса,
// выполняет поиск по каждой, объединяет результаты через Reciprocal Rank Fusion.
func (p *Pipeline) QueryMulti(ctx context.Context, question string, n, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if n <= 0 {
		n = 3
	}

	userMsg := fmt.Sprintf("Generate %d alternative phrasings of this question:\n\n%s", n, question)
	genStart := time.Now()
	p.hookStart(ctx, "QueryMulti:generate", domain.HookStageGenerate)
	raw, err := p.llm.Generate(ctx, multiQuerySystemPrompt, userMsg)
	p.hookEnd(ctx, "QueryMulti:generate", domain.HookStageGenerate, genStart, err)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("multi-query generate: %w", err)
	}

	queries := parseMultiQueryLines(raw)
	queries = append([]string{question}, queries...)

	allResults := make([]domain.RetrievalResult, 0, len(queries))
	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return domain.RetrievalResult{}, err
		}
		embedStart := time.Now()
		p.hookStart(ctx, "QueryMulti:embed", domain.HookStageEmbed)
		emb, err := p.embedder.Embed(ctx, q)
		p.hookEnd(ctx, "QueryMulti:embed", domain.HookStageEmbed, embedStart, err)
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("multi-query embed: %w", err)
		}
		searchStart := time.Now()
		p.hookStart(ctx, "QueryMulti:search", domain.HookStageSearch)
		res, err := p.store.Search(ctx, emb, topK)
		p.hookEnd(ctx, "QueryMulti:search", domain.HookStageSearch, searchStart, err)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		allResults = append(allResults, res)
	}

	merged := rrfMergeMultiple(allResults, topK)
	merged = p.maybeDedup(merged)
	merged.QueryText = question
	merged, err = p.maybeRerank(ctx, question, merged)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	return merged, nil
}

// AnswerMulti генерирует ответ используя multi-query retrieval.
func (p *Pipeline) AnswerMulti(ctx context.Context, question string, n, topK int) (string, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return "", err
	}
	return p.generateAnswer(ctx, question, result)
}

func (p *Pipeline) generateAnswer(ctx context.Context, question string, result domain.RetrievalResult) (string, error) {
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)
	genStart := time.Now()
	p.hookStart(ctx, "Answer:generate", domain.HookStageGenerate)
	answer, err := p.llm.Generate(ctx, p.systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer:generate", domain.HookStageGenerate, genStart, err)
	return answer, err
}

func parseMultiQueryLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func rrfMergeMultiple(lists []domain.RetrievalResult, topK int) domain.RetrievalResult {
	const k = 60
	scores := make(map[string]float64)
	byID := make(map[string]domain.RetrievedChunk)
	for _, res := range lists {
		for rank, rc := range res.Chunks {
			id := rc.Chunk.ID
			scores[id] += 1.0 / float64(k+rank+1)
			byID[id] = rc
		}
	}
	merged := make([]domain.RetrievedChunk, 0, len(scores))
	for id, rc := range byID {
		rc.Score = scores[id]
		merged = append(merged, rc)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	if topK > 0 && len(merged) > topK {
		merged = merged[:topK]
	}
	return domain.RetrievalResult{Chunks: merged}
}

// processDocumentForBatch обрабатывает один документ для batch-индексации.
// Выполняет chunking (если настроен), embedding и upsert всех чанков.
//
// @ds-task T2.1: Вспомогательный метод для обработки документа в worker (AC-001, AC-005)
func (p *Pipeline) processDocumentForBatch(ctx context.Context, doc domain.Document) error {
	if p.chunker != nil {
		chunkStart := time.Now()
		p.hookStart(ctx, "IndexBatch", domain.HookStageChunking)
		chunks, err := p.chunker.Chunk(ctx, doc)
		p.hookEnd(ctx, "IndexBatch", domain.HookStageChunking, chunkStart, err)
		if err != nil {
			return err
		}
		return p.indexChunks(ctx, "IndexBatch", chunks)
	}

	// Legacy path: один чанк на документ
	embedStart := time.Now()
	p.hookStart(ctx, "IndexBatch", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, doc.Content)
	p.hookEnd(ctx, "IndexBatch", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return err
	}

	chunk := domain.Chunk{
		ID:        fmt.Sprintf("%s#%d", doc.ID, 0),
		Content:   doc.Content,
		ParentID:  doc.ID,
		Embedding: embedding,
		Position:  0,
	}
	if err := chunk.Validate(); err != nil {
		return err
	}

	return p.store.Upsert(ctx, chunk)
}

// Query выполняет поиск по вопросу и возвращает RetrievalResult.
func (p *Pipeline) Query(ctx context.Context, question string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, topK)
	p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// QueryWithParentIDs выполняет поиск по вопросу с фильтром по ParentIDs.
//
// Если parentIDs пустой — эквивалентно Query.
func (p *Pipeline) QueryWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (domain.RetrievalResult, error) {
	if len(parentIDs) == 0 {
		return p.Query(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageSearch)
	result, err := vs.SearchWithFilter(ctx, embedding, topK, domain.ParentIDFilter{ParentIDs: parentIDs})
	p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// Answer выполняет полный RAG-цикл: retrieval (Embed+Search) → prompt → LLM.Generate.
func (p *Pipeline) Answer(ctx context.Context, question string, topK int) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", errors.New("question is empty")
	}
	if topK <= 0 {
		return "", errors.New("topK must be > 0")
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", err
	}
	result = p.maybeDedup(result)

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, err := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, err)
	return answer, err
}

// AnswerWithParentIDs выполняет retrieval с фильтром по ParentIDs и генерирует ответ.
//
// Если parentIDs пустой — эквивалентно Answer.
func (p *Pipeline) AnswerWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (string, error) {
	answer, _, err := p.AnswerWithCitationsWithParentIDs(ctx, question, topK, parentIDs)
	return answer, err
}

// QueryWithMetadataFilter выполняет поиск по вопросу с фильтром по метаданным документа.
//
// Если filter.Fields пустой — эквивалентно Query.
// Если store не реализует VectorStoreWithFilters — возвращает ErrFiltersNotSupported.
//
// @ds-task T3.1: Добавить QueryWithMetadataFilter в application.Pipeline (RQ-005, AC-003, DEC-003)
func (p *Pipeline) QueryWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return p.Query(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageSearch)
	result, err := vs.SearchWithMetadataFilter(ctx, embedding, topK, filter)
	p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// AnswerWithMetadataFilter выполняет retrieval с фильтром по метаданным и генерирует ответ.
//
// Если filter.Fields пустой — эквивалентно Answer.
// Если store не реализует VectorStoreWithFilters — возвращает ErrFiltersNotSupported.
//
// @ds-task T3.1: Добавить AnswerWithMetadataFilter в application.Pipeline (RQ-006, AC-003, DEC-003)
func (p *Pipeline) AnswerWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (string, error) {
	if len(filter.Fields) == 0 {
		return p.Answer(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", errors.New("question is empty")
	}
	if topK <= 0 {
		return "", errors.New("topK must be > 0")
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return "", ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := vs.SearchWithMetadataFilter(ctx, embedding, candidateTopK, filter)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", err
	}
	result = p.maybeDedup(result)

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	return answer, genErr
}

// ErrHybridNotSupported возвращается, если pipeline-метод гибридного поиска вызван,
// но underlying VectorStore не поддерживает HybridSearcher capability.
var ErrHybridNotSupported = errors.New("vector store does not support hybrid search")

// QueryHybrid выполняет гибридный поиск (BM25 + semantic) по вопросу.
//
// Если store не реализует HybridSearcher — возвращает ErrHybridNotSupported.
func (p *Pipeline) QueryHybrid(ctx context.Context, question string, topK int, config domain.HybridConfig) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}

	hs, ok := p.store.(domain.HybridSearcher)
	if !ok {
		return domain.RetrievalResult{}, ErrHybridNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "QueryHybrid", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "QueryHybrid", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "QueryHybrid", domain.HookStageSearch)
	result, err := hs.SearchHybrid(ctx, question, embedding, topK, config)
	p.hookEnd(ctx, "QueryHybrid", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// AnswerHybrid выполняет гибридный поиск (BM25 + semantic) и генерирует ответ.
//
// Если store не реализует HybridSearcher — возвращает ErrHybridNotSupported.
func (p *Pipeline) AnswerHybrid(ctx context.Context, question string, topK int, config domain.HybridConfig) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", errors.New("question is empty")
	}
	if topK <= 0 {
		return "", errors.New("topK must be > 0")
	}
	if err := config.Validate(); err != nil {
		return "", err
	}

	result, err := p.QueryHybrid(ctx, question, topK, config)
	if err != nil {
		return "", err
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerHybrid", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "AnswerHybrid", domain.HookStageGenerate, genStart, genErr)
	return answer, genErr
}

// AnswerWithInlineCitations выполняет полный RAG-цикл и возвращает ответ с inline-цитатами `[n]`,
// а также retrieval evidence и детерминированный маппинг `n -> chunk`.
//
// Если retrieval уже выполнен, а Generate вернул ошибку, метод возвращает retrieval результат (partial),
// массив citations и ошибку.
func (p *Pipeline) AnswerWithInlineCitations(
	ctx context.Context,
	question string,
	topK int,
) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", domain.RetrievalResult{}, nil, errors.New("question is empty")
	}
	if topK <= 0 {
		return "", domain.RetrievalResult{}, nil, errors.New("topK must be > 0")
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", domain.RetrievalResult{}, nil, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, citations, genErr
	}
	return answer, result, citations, nil
}

// AnswerStream выполняет RAG-цикл с streaming генерацией ответа.
// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
// Retrieval выполняется синхронно перед началом streaming'а.
//
// @ds-task T2.3: Реализовать AnswerStream в application Pipeline (AC-001, DEC-003)
func (p *Pipeline) AnswerStream(
	ctx context.Context,
	question string,
	topK int,
) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("question is empty")
	}
	if topK <= 0 {
		return nil, errors.New("topK must be > 0")
	}

	// Type assertion для проверки поддержки streaming
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return nil, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return nil, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, genErr
	}

	// Обёртка для отслеживания завершения генерации
	return p.wrapStreamWithHook(ctx, tokenChan, genStart), nil
}

// wrapStreamWithHook оборачивает канал токенов для вызова hook по завершении.
func (p *Pipeline) wrapStreamWithHook(ctx context.Context, source <-chan string, genStart time.Time) <-chan string {
	output := make(chan string)

	go func() {
		defer close(output)
		defer p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, nil)

		for {
			select {
			case <-ctx.Done():
				return
			case token, ok := <-source:
				if !ok {
					return
				}
				select {
				case output <- token:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return output
}

// AnswerStreamWithInlineCitations выполняет RAG-цикл с streaming генерацией и inline-цитатами.
// Возвращает канал для чтения текстовых чанков и слайс цитат (заполняется синхронно перед streaming'ом).
//
// @ds-task T2.4: Реализовать AnswerStreamWithInlineCitations в application Pipeline (AC-002)
func (p *Pipeline) AnswerStreamWithInlineCitations(
	ctx context.Context,
	question string,
	topK int,
) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return nil, domain.RetrievalResult{}, nil, errors.New("question is empty")
	}
	if topK <= 0 {
		return nil, domain.RetrievalResult{}, nil, errors.New("topK must be > 0")
	}

	// Type assertion для проверки поддержки streaming
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, domain.RetrievalResult{}, nil, ErrStreamingNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return nil, domain.RetrievalResult{}, nil, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, result, citations, genErr
	}

	return p.wrapStreamWithHook(ctx, tokenChan, genStart), result, citations, nil
}

// AnswerWithCitations выполняет полный RAG-цикл и возвращает retrieval evidence вместе с ответом.
//
// Если retrieval уже выполнен, а Generate вернул ошибку, метод возвращает retrieval результат (partial)
// и ошибку, чтобы упростить диагностику и отображение источников.
func (p *Pipeline) AnswerWithCitations(ctx context.Context, question string, topK int) (string, domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return "", domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", domain.RetrievalResult{}, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// AnswerWithCitationsWithParentIDs выполняет RAG-цикл с фильтром по ParentIDs и возвращает retrieval evidence.
//
// Если parentIDs пустой — эквивалентно AnswerWithCitations.
func (p *Pipeline) AnswerWithCitationsWithParentIDs(
	ctx context.Context,
	question string,
	topK int,
	parentIDs []string,
) (string, domain.RetrievalResult, error) {
	if len(parentIDs) == 0 {
		return p.AnswerWithCitations(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return "", domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return "", domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := vs.SearchWithFilter(ctx, embedding, candidateTopK, domain.ParentIDFilter{ParentIDs: parentIDs})
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", domain.RetrievalResult{}, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// generateCitedFromResult генерирует ответ с цитатами из готового RetrievalResult.
// Helper для унификации логики генерации в Answer*WithCitations методах.
func (p *Pipeline) generateCitedFromResult(
	ctx context.Context,
	question string,
	result domain.RetrievalResult,
) (string, domain.RetrievalResult, error) {
	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// generateInlineCitedFromResult генерирует ответ с inline-цитатами из готового RetrievalResult.
func (p *Pipeline) generateInlineCitedFromResult(
	ctx context.Context,
	question string,
	result domain.RetrievalResult,
) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, citations, genErr
	}
	return answer, result, citations, nil
}

// AnswerHyDEWithCitations выполняет HyDE retrieval и генерирует ответ с цитатами.
func (p *Pipeline) AnswerHyDEWithCitations(ctx context.Context, question string, topK int) (string, domain.RetrievalResult, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerMultiWithCitations выполняет MultiQuery retrieval и генерирует ответ с цитатами.
func (p *Pipeline) AnswerMultiWithCitations(ctx context.Context, question string, n, topK int) (string, domain.RetrievalResult, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerHybridWithCitations выполняет Hybrid retrieval и генерирует ответ с цитатами.
func (p *Pipeline) AnswerHybridWithCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (string, domain.RetrievalResult, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerWithCitationsWithMetadataFilter выполняет retrieval с фильтром по метаданным и генерирует ответ с цитатами.
func (p *Pipeline) AnswerWithCitationsWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (string, domain.RetrievalResult, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerHyDEWithInlineCitations выполняет HyDE retrieval и генерирует ответ с inline-цитатами.
func (p *Pipeline) AnswerHyDEWithInlineCitations(ctx context.Context, question string, topK int) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerMultiWithInlineCitations выполняет MultiQuery retrieval и генерирует ответ с inline-цитатами.
func (p *Pipeline) AnswerMultiWithInlineCitations(ctx context.Context, question string, n, topK int) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerHybridWithInlineCitations выполняет Hybrid retrieval и генерирует ответ с inline-цитатами.
func (p *Pipeline) AnswerHybridWithInlineCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerWithInlineCitationsWithMetadataFilter выполняет retrieval с фильтром по метаданным и генерирует ответ с inline-цитатами.
func (p *Pipeline) AnswerWithInlineCitationsWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerWithInlineCitationsWithParentIDs выполняет retrieval с фильтром по ParentIDs и генерирует ответ с inline-цитатами.
func (p *Pipeline) AnswerWithInlineCitationsWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// streamFromResult выполняет streaming генерацию из готового RetrievalResult.
func (p *Pipeline) streamFromResult(ctx context.Context, question string, result domain.RetrievalResult) (<-chan string, error) {
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, genErr
	}

	return p.wrapStreamWithHook(ctx, tokenChan, genStart), nil
}

// streamInlineFromResult выполняет streaming генерацию с inline citations из готового RetrievalResult.
func (p *Pipeline) streamInlineFromResult(ctx context.Context, question string, result domain.RetrievalResult) (<-chan string, []domain.InlineCitation, error) {
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, nil, ErrStreamingNotSupported
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, citations, genErr
	}

	return p.wrapStreamWithHook(ctx, tokenChan, genStart), citations, nil
}

// AnswerHyDEStream выполняет HyDE retrieval и streaming генерацию.
func (p *Pipeline) AnswerHyDEStream(ctx context.Context, question string, topK int) (<-chan string, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerMultiStream выполняет MultiQuery retrieval и streaming генерацию.
func (p *Pipeline) AnswerMultiStream(ctx context.Context, question string, n, topK int) (<-chan string, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerHybridStream выполняет Hybrid retrieval и streaming генерацию.
func (p *Pipeline) AnswerHybridStream(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerStreamWithParentIDs выполняет retrieval с фильтром по ParentIDs и streaming генерацию.
func (p *Pipeline) AnswerStreamWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerStreamWithMetadataFilter выполняет retrieval с фильтром по метаданным и streaming генерацию.
func (p *Pipeline) AnswerStreamWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @ds-task T1.1: Методы Answer*StreamWithSources — потоковый ответ с источниками (AC-001, AC-002, DEC-001)

// AnswerStreamWithSources выполняет базовый retrieval и streaming генерацию.
// Возвращает канал токенов и RetrievalResult синхронно.
func (p *Pipeline) AnswerStreamWithSources(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.Query(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerHyDEStreamWithSources выполняет HyDE retrieval и streaming генерацию.
// Возвращает канал токенов и RetrievalResult синхронно.
func (p *Pipeline) AnswerHyDEStreamWithSources(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerMultiStreamWithSources выполняет MultiQuery retrieval и streaming генерацию.
// Возвращает канал токенов и RetrievalResult синхронно.
func (p *Pipeline) AnswerMultiStreamWithSources(ctx context.Context, question string, n, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerHybridStreamWithSources выполняет Hybrid retrieval и streaming генерацию.
// Возвращает канал токенов и RetrievalResult синхронно.
func (p *Pipeline) AnswerHybridStreamWithSources(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerStreamWithParentIDsWithSources выполняет retrieval с фильтром по ParentIDs и streaming генерацию.
// Возвращает канал токенов и RetrievalResult синхронно.
func (p *Pipeline) AnswerStreamWithParentIDsWithSources(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerStreamWithMetadataFilterWithSources выполняет retrieval с фильтром по метаданным и streaming генерацию.
// Возвращает канал токенов и RetrievalResult синхронно.
func (p *Pipeline) AnswerStreamWithMetadataFilterWithSources(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerHyDEStreamWithInlineCitations выполняет HyDE retrieval и streaming генерацию с inline-цитатами.
func (p *Pipeline) AnswerHyDEStreamWithInlineCitations(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerMultiStreamWithInlineCitations выполняет MultiQuery retrieval и streaming генерацию с inline-цитатами.
func (p *Pipeline) AnswerMultiStreamWithInlineCitations(ctx context.Context, question string, n, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerHybridStreamWithInlineCitations выполняет Hybrid retrieval и streaming генерацию с inline-цитатами.
func (p *Pipeline) AnswerHybridStreamWithInlineCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerStreamWithParentIDsWithInlineCitations выполняет retrieval с фильтром по ParentIDs и streaming генерацию с inline-цитатами.
func (p *Pipeline) AnswerStreamWithParentIDsWithInlineCitations(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerStreamWithMetadataFilterWithInlineCitations выполняет retrieval с фильтром по метаданным и streaming генерацию с inline-цитатами.
func (p *Pipeline) AnswerStreamWithMetadataFilterWithInlineCitations(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

func buildUserMessageV1(result domain.RetrievalResult, question string, maxContextChars, maxContextChunks int) string {
	contextText := buildContextTextV1(result, maxContextChars, maxContextChunks)

	var b strings.Builder
	b.WriteString("Контекст:\n")
	b.WriteString(contextText)
	// Гарантируем одну пустую строку между контекстом и вопросом (как в Prompt Contract v1),
	// независимо от того, был ли контекст обрезан по символам.
	if contextText == "" || strings.HasSuffix(contextText, "\n") {
		b.WriteString("\nВопрос:\n")
	} else {
		b.WriteString("\n\nВопрос:\n")
	}
	b.WriteString(question)
	return b.String()
}

func buildUserMessageV1InlineCitations(
	result domain.RetrievalResult,
	question string,
	maxContextChars, maxContextChunks int,
) (string, []domain.InlineCitation) {
	contextText, citations := buildContextTextV1InlineCitations(result, maxContextChars, maxContextChunks)

	var b strings.Builder

	b.WriteString("Инструкция:\n")
	b.WriteString("- В тексте ответа добавляй ссылки на источники в формате [n].\n")
	b.WriteString("- Используй только номера, которые есть в списке источников.\n\n")

	b.WriteString("Источники:\n")
	b.WriteString(contextText)
	// Гарантируем одну пустую строку между источниками и вопросом.
	if contextText == "" || strings.HasSuffix(contextText, "\n") {
		b.WriteString("\nВопрос:\n")
	} else {
		b.WriteString("\n\nВопрос:\n")
	}
	b.WriteString(question)

	return b.String(), citations
}

func buildContextTextV1(result domain.RetrievalResult, maxContextChars, maxContextChunks int) string {
	var b strings.Builder

	wroteChunks := 0
	for _, rc := range result.Chunks {
		if maxContextChunks > 0 && wroteChunks >= maxContextChunks {
			break
		}
		b.WriteString(rc.Chunk.Content)
		b.WriteString("\n")
		wroteChunks++
	}

	context := b.String()
	if maxContextChars <= 0 {
		return context
	}

	runes := []rune(context)
	if len(runes) <= maxContextChars {
		return context
	}
	return string(runes[:maxContextChars])
}

func buildContextTextV1InlineCitations(
	result domain.RetrievalResult,
	maxContextChars, maxContextChunks int,
) (string, []domain.InlineCitation) {
	var b strings.Builder
	citations := make([]domain.InlineCitation, 0, len(result.Chunks))

	runesWritten := 0
	wroteChunks := 0

	for _, rc := range result.Chunks {
		if maxContextChunks > 0 && wroteChunks >= maxContextChunks {
			break
		}

		number := wroteChunks + 1
		marker := "[" + strconv.Itoa(number) + "]"
		line := marker + " " + rc.Chunk.Content + "\n"

		if maxContextChars > 0 {
			lineRunes := []rune(line)
			if runesWritten+len(lineRunes) > maxContextChars {
				remaining := maxContextChars - runesWritten
				// Если не влезает даже маркер — ничего не добавляем и завершаем.
				if remaining <= len([]rune(marker)) {
					break
				}
				b.WriteString(string(lineRunes[:remaining]))
				citations = append(citations, domain.InlineCitation{
					Number: number,
					Chunk:  rc,
				})
				break
			}
		}

		b.WriteString(line)
		if maxContextChars > 0 {
			runesWritten += len([]rune(line))
		}
		citations = append(citations, domain.InlineCitation{
			Number: number,
			Chunk:  rc,
		})
		wroteChunks++
	}

	return b.String(), citations
}
