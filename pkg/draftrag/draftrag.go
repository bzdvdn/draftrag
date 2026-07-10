package draftrag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bzdvdn/draftrag/internal/application"
	"github.com/bzdvdn/draftrag/internal/domain"
)

// VectorStore определяет интерфейс для работы с векторным хранилищем.
type VectorStore = domain.VectorStore

// LLMProvider определяет интерфейс для генерации текста через LLM.
type LLMProvider = domain.LLMProvider

// Embedder определяет интерфейс для преобразования текста в векторное представление.
type Embedder = domain.Embedder

// Chunker определяет интерфейс для разбиения документа на чанки.
type Chunker = domain.Chunker

// Hooks — опциональные хуки наблюдаемости для стадий pipeline.
type Hooks = domain.Hooks

// Logger — опциональный структурированный логгер для инфраструктурных событий (кэш, retry).
// nil означает no-op.
type Logger = domain.Logger

// LogLevel — уровень логирования.
type LogLevel = domain.LogLevel

// Уровни логирования.
const (
	LogLevelDebug = domain.LogLevelDebug
	LogLevelInfo  = domain.LogLevelInfo
	LogLevelWarn  = domain.LogLevelWarn
	LogLevelError = domain.LogLevelError
)

// LogField — структурированное поле лог-события.
type LogField = domain.LogField

// ParentIDFilter задаёт фильтрацию retrieval по ParentID.
type ParentIDFilter = domain.ParentIDFilter

// MetadataFilter задаёт условие точного совпадения по полям метаданных документа при поиске.
// Пустой Fields (nil или len==0) означает «без фильтра».
// Все условия применяются как AND: все пары ключ-значение из Fields должны совпасть.
//
// @ds-task T3.2: Переэкспортировать MetadataFilter из domain в публичный API (RQ-005, RQ-006, AC-003)
type MetadataFilter = domain.MetadataFilter

// VectorStoreWithFilters — опциональная capability интерфейса VectorStore, поддерживающая фильтры.
type VectorStoreWithFilters = domain.VectorStoreWithFilters

// HybridSearcher — опциональная capability интерфейса VectorStore, поддерживающая гибридный поиск (BM25 + semantic).
type HybridSearcher = domain.HybridSearcher

// HybridConfig задаёт параметры гибридного поиска (BM25 + semantic).
type HybridConfig = domain.HybridConfig

// DefaultHybridConfig возвращает конфигурацию гибридного поиска по умолчанию.
var DefaultHybridConfig = domain.DefaultHybridConfig

// Document представляет документ для индексации в RAG-системе.
type Document = domain.Document

// Chunk представляет фрагмент документа.
type Chunk = domain.Chunk

// RetrievalResult содержит результаты поиска.
type RetrievalResult = domain.RetrievalResult

// InlineCitation задаёт соответствие номера цитаты `[n]` и retrieval-источника (чанка).
type InlineCitation = domain.InlineCitation

// IndexBatchResult содержит результат batch-индексации документов.
type IndexBatchResult = domain.IndexBatchResult

// IndexBatchError представляет ошибку индексации конкретного документа при batch-индексации.
type IndexBatchError = domain.IndexBatchError

// StreamingLLMProvider — опциональная capability интерфейса LLMProvider, поддерживающая streaming.
type StreamingLLMProvider = domain.StreamingLLMProvider

// TokenUsage содержит количество токенов, использованных в одном LLM-вызове.
//
// @sk-task cost-tracking: re-export TokenUsage (AC-001, RQ-001)
type TokenUsage = domain.TokenUsage

// ModelPricing задаёт цены за 1K токенов для модели.
//
// @sk-task cost-tracking: re-export ModelPricing (AC-002, RQ-002)
type ModelPricing = domain.ModelPricing

// CostSnapshot — атомарный срез накопленной статистики cost tracker'а.
//
// @sk-task cost-tracking: re-export CostSnapshot (AC-003, RQ-003)
type CostSnapshot = domain.CostSnapshot

// UsageAwareLLMProvider — опциональная capability для LLMProvider,
// возвращающих token usage в API-ответе.
//
// @sk-task cost-tracking: re-export UsageAwareLLMProvider (AC-001, RQ-001)
type UsageAwareLLMProvider = domain.UsageAwareLLMProvider

// UsageAwareStreamingLLMProvider — опциональная capability для StreamingLLMProvider,
// возвращающих token usage из финального chunk SSE-потока.
//
// @sk-task cost-tracking: re-export UsageAwareStreamingLLMProvider (AC-005, RQ-006, T3.4)
type UsageAwareStreamingLLMProvider = domain.UsageAwareStreamingLLMProvider

// Diff возвращает разницу между двумя снапшотами CostSnapshot.
//
// @sk-task cost-tracking: re-export Diff (AC-007, RQ-007)
var Diff = domain.Diff

// Reranker — опциональный интерфейс для переранжирования результатов retrieval.
type Reranker = domain.Reranker

// BatchReranker — опциональное расширение Reranker для batch-режима.
//
// @sk-task reranker-cross-encoder#T1.1: re-export BatchReranker (AC-008)
type BatchReranker = domain.BatchReranker

// @sk-task query-rewriting#T2.1: re-export QueryRewriter (AC-002)
// QueryRewriter — опциональный компонент для переписывания запроса перед retrieval.
type QueryRewriter = domain.QueryRewriter

// @sk-task query-rewriting#T2.1: re-export RewrittenQuery (AC-002)
// RewrittenQuery представляет результат переформулировки запроса.
type RewrittenQuery = domain.RewrittenQuery

// @sk-task query-rewriting#T2.1: re-export QueryHistory (AC-002)
// QueryHistory содержит историю предыдущих сообщений диалога.
type QueryHistory = domain.QueryHistory

// PipelineConfig — удалён. Используйте PipelineOptions.
//
// @sk-task arch-quality-pass#T1.1: re-export alias PipelineConfig → PipelineOptions (AC-004)
// @sk-task arch-quality-pass#T3.2: удалён (AC-004)

// DocumentStore — опциональная capability VectorStore для удаления по ParentID.
type DocumentStore = domain.DocumentStore

// TransactionalTx — транзакция в транзакционном vector store.
//
// @sk-task api-consistency-pass#T1.1: re-export для публичного API (RQ-005, AC-008)
type TransactionalTx = domain.TransactionalTx

// TransactionalDocumentStore — опциональная capability VectorStore, поддерживающая
// транзакционные операции для атомарного UpdateDocument.
//
// @sk-task api-consistency-pass#T1.1: re-export для публичного API (RQ-005, AC-008)
type TransactionalDocumentStore = domain.TransactionalDocumentStore

// Pipeline — публичный API для композиции core-компонентов draftRAG.
// Валидация входных данных выполняется здесь (см. errors.go).
// @sk-task query-rewriting#T2.1: добавлено поле queryRewriter (AC-002)
// @sk-task pii-guardrails#T2.1: добавлено поле piidetector (RQ-001, RQ-002)
type Pipeline struct {
	core          *application.Pipeline
	defaultTop    int
	queryRewriter QueryRewriter
	piidetector   domain.PIIDetector
}

// PipelineOptions задаёт конфигурацию Pipeline.
type PipelineOptions struct {
	// DefaultTopK — значение topK по умолчанию для Query/Answer.
	// Если 0, используется значение по умолчанию (5).
	// Если < 0, это считается ошибкой конфигурации (panic).
	DefaultTopK int
	// SystemPrompt — переопределение system prompt для Answer*. Пустая строка означает дефолт v1.
	SystemPrompt string
	// Chunker — опциональный чанкер; если не nil, Index индексирует чанки (Chunk→Embed→Upsert).
	Chunker Chunker
	// Hooks — опциональные хуки наблюдаемости для стадий pipeline (chunking/embed/search/generate).
	// nil означает no-op.
	Hooks Hooks

	// MaxContextChars — лимит размера секции “Контекст:” в prompt для Answer* (в символах).
	// 0 означает “без лимита”.
	MaxContextChars int
	// MaxContextChunks — лимит количества чанков, попадающих в секцию “Контекст:” в prompt для Answer*.
	// 0 означает “без лимита”.
	MaxContextChunks int

	// DedupByParentID включает дедупликацию retrieval sources по ParentID.
	// По умолчанию выключено (backward compatibility).
	DedupByParentID bool

	// MMREnabled включает MMR rerank/selection для retrieval sources (диверсификация контекста).
	// По умолчанию выключено (backward compatibility).
	MMREnabled bool
	// MMRLambda задаёт баланс релевантность/разнообразие в диапазоне [0..1].
	// Если 0 и MMR включён — используется значение по умолчанию (0.5).
	MMRLambda float64
	// MMRCandidatePool задаёт сколько кандидатов запросить у VectorStore до отбора.
	// Если 0 — используется topK запроса.
	MMRCandidatePool int

	// IndexConcurrency задаёт количество workers для параллельной индексации в IndexBatch.
	// Если 0 — используется значение по умолчанию (4).
	IndexConcurrency int
	// IndexBatchRateLimit задаёт максимальное количество вызовов Embed в секунду в IndexBatch.
	// Если 0 — без ограничений.
	IndexBatchRateLimit int
	// IndexBatchRateLimitPerWorker включает per-worker rate-limiter: каждый worker
	// индексирует документы с частотой IndexBatchRateLimit per second (а не весь
	// пул суммарно). Используется, когда у каждого worker'а свой внешний квотный
	// лимит (например, несколько подов с независимыми rate-limits в API-ключе).
	//
	// По умолчанию false (backward-compat): один общий ticker на пул.
	// При IndexBatchRateLimit=10 и IndexConcurrency=4:
	//   - false: 10 embed/sec суммарно на пул.
	//   - true:  10 embed/sec на каждого worker'а, т.е. 40 embed/sec суммарно.
	IndexBatchRateLimitPerWorker bool

	// StreamBufferSize задаёт ёмкость буфера канала streaming-вывода в Answer*.
	// 0 — unbuffered (backward-compat, OQ-2: токен передаётся синхронно).
	// N > 0 — producer (LLM-стрим) может обгонять consumer на N токенов
	// без блокировки; при заполнении буфера producer блокируется на send.
	// Используется для bounded backpressure между LLM и consuming кодом.
	StreamBufferSize int

	// Reranker — опциональный reranker, применяется после retrieval.
	// nil означает "без reranking".
	Reranker Reranker

	// QueryRewriter — опциональный rewriter для переписывания запроса перед retrieval.
	// Поддерживает 1:1 и 1:N режимы. nil означает "без переписывания".
	// При установке per-request Rewriter через SearchBuilder.Rewriter имеет приоритет.
	QueryRewriter QueryRewriter

	// @sk-task pii-guardrails#T2.1: PIIDetector опция (RQ-001, RQ-002)
	// PIIDetector — опциональный детектор PII.
	// Если установлен, содержимое документов и результатов retrieval
	// проходит через детектор для цензурирования PII.
	// nil означает "без обработки" (backward compatible).
	PIIDetector PIIDetector
}

// NewPipeline создаёт pipeline из зависимостей: VectorStore, LLMProvider и Embedder.
//
// @sk-task arch-quality-pass#T2.2: error return вместо panic (AC-002)
func NewPipeline(store VectorStore, llm LLMProvider, embedder Embedder) (*Pipeline, error) {
	return NewPipelineWithOptions(store, llm, embedder, PipelineOptions{})
}

// NewPipelineWithChunker создаёт pipeline из зависимостей: VectorStore, LLMProvider, Embedder и Chunker.
//
// При наличии Chunker метод Index будет индексировать чанки (Chunker.Chunk → Embed → Upsert).
//
// @sk-task arch-quality-pass#T2.2: error return вместо panic (AC-002)
func NewPipelineWithChunker(store VectorStore, llm LLMProvider, embedder Embedder, chunker Chunker) (*Pipeline, error) {
	core, err := application.NewPipelineWithChunker(store, llm, embedder, chunker)
	if err != nil {
		return nil, err
	}
	return &Pipeline{
		core:       core,
		defaultTop: 5,
	}, nil
}

// NewPipelineWithOptions создаёт pipeline из зависимостей: VectorStore, LLMProvider и Embedder,
// применяя конфигурацию из PipelineOptions.
//
// @sk-task arch-quality-pass#T2.2: error return вместо panic (AC-002)
func NewPipelineWithOptions(store VectorStore, llm LLMProvider, embedder Embedder, opts PipelineOptions) (*Pipeline, error) {
	defaultTop := 5
	if opts.DefaultTopK < 0 {
		return nil, fmt.Errorf("DefaultTopK must be >= 0, got %d", opts.DefaultTopK)
	}
	if opts.DefaultTopK > 0 {
		defaultTop = opts.DefaultTopK
	}
	if opts.MaxContextChars < 0 {
		return nil, fmt.Errorf("MaxContextChars must be >= 0, got %d", opts.MaxContextChars)
	}
	if opts.MaxContextChunks < 0 {
		return nil, fmt.Errorf("MaxContextChunks must be >= 0, got %d", opts.MaxContextChunks)
	}
	if opts.MMRCandidatePool < 0 {
		return nil, fmt.Errorf("MMRCandidatePool must be >= 0, got %d", opts.MMRCandidatePool)
	}
	if opts.MMRLambda < 0 || opts.MMRLambda > 1 {
		return nil, fmt.Errorf("MMRLambda must be in [0..1], got %f", opts.MMRLambda)
	}

	core, err := application.NewPipelineWithConfig(store, llm, embedder, application.PipelineOptions{
		SystemPrompt:                 opts.SystemPrompt,
		Chunker:                      opts.Chunker,
		Hooks:                        opts.Hooks,
		MaxContextChars:              opts.MaxContextChars,
		MaxContextChunks:             opts.MaxContextChunks,
		DedupByParentID:              opts.DedupByParentID,
		MMREnabled:                   opts.MMREnabled,
		MMRLambda:                    opts.MMRLambda,
		MMRCandidatePool:             opts.MMRCandidatePool,
		IndexConcurrency:             opts.IndexConcurrency,
		IndexBatchRateLimit:          opts.IndexBatchRateLimit,
		IndexBatchRateLimitPerWorker: opts.IndexBatchRateLimitPerWorker,
		StreamBufferSize:             opts.StreamBufferSize,
		Reranker:                     opts.Reranker,
		PIIDetector:                  opts.PIIDetector,
	})
	if err != nil {
		return nil, err
	}
	return &Pipeline{
		core:          core,
		defaultTop:    defaultTop,
		queryRewriter: opts.QueryRewriter,
		piidetector:   opts.PIIDetector,
	}, nil
}

// queryRewriter доступен на Pipeline для передачи в SearchBuilder.
// @sk-task query-rewriting#T2.1: pipeline-level queryRewriter (AC-002)

// Index индексирует документы.
// @sk-task arch-generics#T2.2: nil context guard + checkCtx вместо panic (AC-002)
// @sk-task pii-guardrails#T2.2: PII redaction в Index (AC-001, RQ-001)
func (p *Pipeline) Index(ctx context.Context, docs []Document) error {
	if err := checkCtx(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for i := range docs {
		content := strings.TrimSpace(docs[i].Content)
		if content == "" {
			return ErrEmptyDocument
		}
		if p.piidetector != nil {
			docs[i].Content = p.piidetector.Detect(docs[i].Content)
		}
	}

	err := p.core.Index(ctx, docs)
	return mapAppError(err)
}

// Query выполняет поиск с topK по умолчанию (PipelineOptions.DefaultTopK или 5).
// Для расширенных параметров используйте Search builder.
// @sk-task arch-generics#T2.2: nil context guard + checkCtx вместо panic (AC-002)
// @sk-task pii-guardrails#T2.3: PII redaction в Query (AC-002, RQ-002)
func (p *Pipeline) Query(ctx context.Context, question string) (RetrievalResult, error) {
	if err := checkCtx(ctx); err != nil {
		return RetrievalResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return RetrievalResult{}, err
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return RetrievalResult{}, ErrEmptyQuery
	}
	result, err := p.core.Query(ctx, question, p.defaultTop)
	if err != nil {
		return RetrievalResult{}, mapAppError(err)
	}
	if p.piidetector != nil {
		for i := range result.Chunks {
			result.Chunks[i].Chunk.Content = p.piidetector.Detect(result.Chunks[i].Chunk.Content)
		}
	}
	return result, nil
}

// Answer генерирует ответ с topK по умолчанию (PipelineOptions.DefaultTopK или 5).
// Для расширенных параметров используйте Search builder.
// @sk-task arch-generics#T2.2: nil context guard + checkCtx вместо panic (AC-002)
func (p *Pipeline) Answer(ctx context.Context, question string) (string, error) {
	if err := checkCtx(ctx); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return "", ErrEmptyQuery
	}
	return p.core.Answer(ctx, question, p.defaultTop)
}

// UpdateDocument удаляет все чанки документа и переиндексирует его.
// Атомарности нет: при ошибке переиндексации старые чанки уже удалены.
// Требует DocumentStore capability.
// @sk-task arch-generics#T2.2: nil context guard + checkCtx вместо panic (AC-002)
func (p *Pipeline) UpdateDocument(ctx context.Context, doc Document) error {
	if err := checkCtx(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(doc.Content) == "" {
		return ErrEmptyDocument
	}
	err := p.core.UpdateDocument(ctx, doc)
	if errors.Is(err, application.ErrDeleteNotSupported) {
		return ErrDeleteNotSupported
	}
	return err
}

// Retrieve выполняет поиск по вопросу с заданным topK и возвращает RetrievalResult.
// Удобен для прямой передачи в eval.Run (реализует eval.RetrievalRunner).
// Для цепочки с фильтрами, hybrid и другими параметрами используйте Search builder.
func (p *Pipeline) Retrieve(ctx context.Context, question string, topK int) (RetrievalResult, error) {
	return p.Search(question).TopK(topK).Retrieve(ctx)
}

// DeleteDocument удаляет документ и все его чанки по ID документа (ParentID).
// Требует, чтобы VectorStore реализовывал DocumentStore capability.
// Если store не поддерживает — возвращает ErrDeleteNotSupported.
// @sk-task arch-generics#T2.2: nil context guard + checkCtx вместо panic (AC-002)
func (p *Pipeline) DeleteDocument(ctx context.Context, docID string) error {
	if err := checkCtx(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(docID) == "" {
		return ErrEmptyDocumentID
	}
	err := p.core.DeleteDocument(ctx, docID)
	if errors.Is(err, application.ErrDeleteNotSupported) {
		return ErrDeleteNotSupported
	}
	return err
}

// ErrStreamingNotSupported возвращается, если streaming-метод вызван,
// но underlying LLMProvider не поддерживает StreamingLLMProvider capability.
var ErrStreamingNotSupported = errors.New("LLM provider does not support streaming")

// ErrHybridNotSupported возвращается, если метод гибридного поиска вызван,
// но underlying VectorStore не поддерживает HybridSearcher capability.
var ErrHybridNotSupported = errors.New("vector store does not support hybrid search")

// ErrDeleteNotSupported возвращается, если DeleteDocument вызван,
// но underlying VectorStore не реализует DocumentStore capability.
var ErrDeleteNotSupported = errors.New("vector store does not support DeleteByParentID")

// ErrEmptyDocumentID возвращается, если передан пустой ID документа.
var ErrEmptyDocumentID = errors.New("document ID must not be empty")

// IndexBatch индексирует документы параллельно с ограничением concurrency.
// @sk-task arch-generics#T2.2: nil context guard + checkCtx вместо panic (AC-002)
//
// В отличие от Index, IndexBatch обрабатывает документы конкурентно (batchSize workers)
// и возвращает partial results: успешно проиндексированные документы и ошибки отдельно.
// Ошибка одного документа не прерывает обработку остальных.
//
// batchSize — количество параллельных workers (0 → значение по умолчанию 4).
// Для управления concurrency и rate limiting используйте PipelineOptions.IndexConcurrency,
// PipelineOptions.IndexBatchRateLimit и PipelineOptions.IndexBatchRateLimitPerWorker
// при создании Pipeline.
func (p *Pipeline) IndexBatch(ctx context.Context, docs []Document, batchSize int) (*IndexBatchResult, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if strings.TrimSpace(doc.Content) == "" {
			return nil, ErrEmptyDocument
		}
	}

	result, err := p.core.IndexBatch(ctx, docs, batchSize)
	return result, mapAppError(err)
}

// @sk-task pii-guardrails#T2.3: redactRetrievalResult (RQ-002, AC-002)
func (p *Pipeline) redactRetrievalResult(rr *RetrievalResult) {
	if p.piidetector == nil {
		return
	}
	for i := range rr.Chunks {
		rr.Chunks[i].Chunk.Content = p.piidetector.Detect(rr.Chunks[i].Chunk.Content)
	}
}

// mapAppError — единая точка маппинга application/domain ошибок на публичные sentinel'ы.
//
// @sk-task api-consistency-pass#T2.2: rename mapValidationErr → mapAppError + расширение маппинга (RQ-003, AC-005, AC-006)
// @sk-task hardening-2026q2#T3.2: Упрощение mapValidationErr (AC-010)
func mapAppError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, application.ErrStreamingNotSupported):
		return ErrStreamingNotSupported
	case errors.Is(err, application.ErrHybridNotSupported):
		return ErrHybridNotSupported
	case errors.Is(err, application.ErrFiltersNotSupported):
		return ErrFiltersNotSupported
	case errors.Is(err, application.ErrDeleteNotSupported):
		return ErrDeleteNotSupported
	case errors.Is(err, domain.ErrEmptyQueryText):
		return ErrEmptyQuery
	case errors.Is(err, domain.ErrInvalidQueryTopK):
		return ErrInvalidTopK
	case errors.Is(err, domain.ErrEmptyDocumentContent):
		return ErrEmptyDocument
	case errors.Is(err, domain.ErrEmbeddingDimensionMismatch):
		return ErrEmbeddingDimensionMismatch
	case errors.Is(err, domain.ErrUpdateNotAtomic):
		return ErrUpdateNotAtomic
	}
	return err
}
