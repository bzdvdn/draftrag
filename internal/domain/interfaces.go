package domain

import (
	"context"
)

// @sk-task health-check-interface#T1.1: Добавлен Health(ctx) в VectorStore (AC-001, RQ-001)
// VectorStore определяет интерфейс для работы с векторным хранилищем.
// Реализации должны поддерживать операции upsert, delete и поиска по embedding-вектору.
type VectorStore interface {
	// Upsert сохраняет или обновляет чанк в хранилище.
	Upsert(ctx context.Context, chunk Chunk) error

	// Delete удаляет чанк по ID из хранилища.
	Delete(ctx context.Context, id string) error

	// Search выполняет поиск похожих чанков по embedding-вектору.
	// Возвращает результат с релевантными чанками, отсортированными по score (по убыванию).
	Search(ctx context.Context, embedding []float64, topK int) (RetrievalResult, error)

	// Health проверяет доступность хранилища.
	// Возвращает nil если компонент работает, error с описанием проблемы если нет.
	Health(ctx context.Context) error
}

// ParentIDFilter задаёт фильтрацию retrieval по ParentID (например, в пределах одного документа).
type ParentIDFilter struct {
	// ParentIDs — список допустимых parent_id. Пустой список означает “без фильтра”.
	ParentIDs []string
}

// VectorStoreWithFilters — опциональная capability интерфейса VectorStore.
//
// Реализации, которые поддерживают фильтры, должны реализовать этот интерфейс дополнительно
// (без ломки существующего контракта VectorStore).
//
// @ds-task T1.3: Расширить VectorStoreWithFilters методом SearchWithMetadataFilter (RQ-002, DEC-001)
type VectorStoreWithFilters interface {
	VectorStore

	// SearchWithFilter выполняет поиск похожих чанков по embedding-вектору с дополнительным фильтром.
	SearchWithFilter(ctx context.Context, embedding []float64, topK int, filter ParentIDFilter) (RetrievalResult, error)

	// SearchWithMetadataFilter выполняет поиск похожих чанков с фильтрацией по полям метаданных.
	// Пустой filter.Fields (nil или len==0) эквивалентно вызову Search без фильтра.
	SearchWithMetadataFilter(ctx context.Context, embedding []float64, topK int, filter MetadataFilter) (RetrievalResult, error)
}

// @sk-task cost-tracking: UsageAwareLLMProvider — optional capability для LLMProvider (AC-001, RQ-001)
// UsageAwareLLMProvider — опциональная capability интерфейса LLMProvider.
//
// Реализации, которые могут возвращать token usage из API-ответа, должны
// реализовать этот интерфейс дополнительно (без ломки существующего контракта LLMProvider).
//
// @sk-task cost-tracking: GenerateWithUsage возвращает token usage (AC-001, RQ-001)
type UsageAwareLLMProvider interface {
	LLMProvider

	// GenerateWithUsage генерирует ответ и возвращает token usage.
	GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, TokenUsage, error)

	// ModelName возвращает имя модели (например, "gpt-4o", "claude-3-haiku-20240307").
	ModelName() string
}

// @sk-task health-check-interface#T1.1: Добавлен Health(ctx) в LLMProvider (AC-003, RQ-001)
// LLMProvider определяет интерфейс для генерации текста через LLM.
type LLMProvider interface {
	// Generate генерирует ответ на основе system и user сообщений.
	Generate(ctx context.Context, systemPrompt, userMessage string) (string, error)

	// Health проверяет доступность LLM провайдера.
	// Возвращает nil если компонент работает, error с описанием проблемы если нет.
	Health(ctx context.Context) error
}

// @sk-task cost-tracking: UsageAwareStreamingLLMProvider — optional capability для streaming (AC-005, RQ-006, T3.4)
// UsageAwareStreamingLLMProvider — опциональная capability интерфейса StreamingLLMProvider.
//
// Реализации, которые поддерживают streaming и могут возвращать token usage
// из финального chunk SSE-потока, должны реализовать этот интерфейс дополнительно
// (без ломки существующего контракта StreamingLLMProvider).
//
// StreamUsage ДОЛЖЕН вызываться только после полного потребления канала из GenerateStream.
// Возвращает TokenUsage и true, если usage доступен.
type UsageAwareStreamingLLMProvider interface {
	StreamingLLMProvider

	// StreamUsage возвращает token usage последнего streaming-вызова.
	// Должен вызываться после полного чтения канала GenerateStream.
	// Возвращает (TokenUsage{}, false) если usage недоступен.
	StreamUsage() (TokenUsage, bool)
}

// StreamingLLMProvider — опциональная capability интерфейса LLMProvider.
//
// Реализации, которые поддерживают streaming, должны реализовать этот интерфейс дополнительно
// (без ломки существующего контракта LLMProvider).
//
// @ds-task T1.1: Добавить StreamingLLMProvider интерфейс (DEC-001, AC-004)
type StreamingLLMProvider interface {
	LLMProvider

	// GenerateStream генерирует ответ токен за токеном через канал.
	// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
	GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error)
}

// @sk-task health-check-interface#T1.1: Добавлен Health(ctx) в Embedder (AC-002, RQ-001)
// Embedder определяет интерфейс для преобразования текста в векторное представление.
type Embedder interface {
	// Embed преобразует текст в embedding-вектор фиксированной размерности.
	Embed(ctx context.Context, text string) ([]float64, error)

	// Health проверяет доступность embedder'а.
	// Возвращает nil если компонент работает, error с описанием проблемы если нет.
	Health(ctx context.Context) error
}

// Chunker определяет интерфейс для разбиения документа на чанки.
type Chunker interface {
	// Chunk разбивает документ на фрагменты для индексации.
	Chunk(ctx context.Context, doc Document) ([]Chunk, error)
}

// HybridSearcher определяет capability для хранилищ, поддерживающих гибридный поиск (BM25 + semantic).
type HybridSearcher interface {
	// SearchHybrid выполняет гибридный поиск: семантический + BM25.
	// Возвращает объединённые результаты с скором от fusion-стратегии.
	SearchHybrid(ctx context.Context, query string, embedding []float64, topK int, config HybridConfig) (RetrievalResult, error)
}

// Reranker — опциональная capability для переранжирования результатов retrieval.
// Принимает исходный вопрос и список чанков, возвращает переупорядоченный список.
// Типичные реализации: cross-encoder, Cohere Rerank, LLM-based scoring.
type Reranker interface {
	Rerank(ctx context.Context, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)
}

// BatchReranker — опциональное расширение Reranker для batch-режима.
//
// @sk-task reranker-cross-encoder#T1.1: BatchReranker interface (AC-008)
// Позволяет переранжировать один набор чанков по нескольким query одновременно.
// Pipeline проверяет реализацию через type assertion в multi-query режиме.
type BatchReranker interface {
	Reranker
	// RerankBatch принимает несколько query и один набор чанков.
	// Возвращает список результатов той же длины, что и queries.
	// Каждый результат — переранжированная версия chunks для соответствующего query.
	RerankBatch(ctx context.Context, queries []string, chunks []RetrievedChunk) ([][]RetrievedChunk, error)
}

// @sk-task query-rewriting#T1.1: QueryRewriter interface (AC-001)
// QueryRewriter — опциональный компонент для переписывания запроса перед retrieval.
//
// Реализации могут быть LLM-based (через LLMProvider), rule-based или гибридными.
// Поддерживает два режима: 1:1 (одна переформулировка) и 1:N (несколько переформулировок).
type QueryRewriter interface {
	// Rewrite переписывает запрос с учётом истории диалога.
	// Возвращает одну или несколько переформулировок.
	// При пустом результате (nil или len==0) pipeline использует исходный запрос.
	// Ошибка не фатальна — pipeline логирует и использует исходный запрос.
	Rewrite(ctx context.Context, query string, history QueryHistory) ([]RewrittenQuery, error)
}

// DocumentStore — опциональная capability VectorStore для удаления документа целиком по ParentID.
type DocumentStore interface {
	VectorStore
	// DeleteByParentID удаляет все чанки с указанным ParentID.
	DeleteByParentID(ctx context.Context, parentID string) error
}

// TransactionalTx — транзакция в транзакционном vector store.
//
// Контракт:
//   - DeleteByParentID и Upsert работают в контексте открытой транзакции;
//     изменения видимы только после Commit.
//   - При ошибке любого метода (или явном Rollback) все изменения откатываются.
//   - Методы НЕ ДОЛЖНЫ вызываться после Commit/Rollback — поведение зависит от реализации.
//
// @sk-task api-consistency-pass#T1.1: интерфейс для атомарного UpdateDocument (RQ-005, AC-008)
type TransactionalTx interface {
	// DeleteByParentID удаляет все чанки с указанным ParentID в транзакции.
	DeleteByParentID(ctx context.Context, parentID string) error
	// Upsert сохраняет или обновляет чанк в транзакции.
	Upsert(ctx context.Context, chunk Chunk) error
	// Commit фиксирует все изменения, сделанные в транзакции.
	Commit() error
	// Rollback откатывает все изменения, сделанные в транзакции.
	Rollback() error
}

// TransactionalDocumentStore — опциональная capability VectorStore, поддерживающая
// транзакционные операции для атомарного UpdateDocument.
//
// Реализации, поддерживающие транзакции (например, pgvector через *sql.Tx),
// реализуют этот интерфейс дополнительно к DocumentStore. Pipeline при наличии
// capability использует транзакционный путь; иначе — best-effort path с возвратом
// ErrUpdateNotAtomic при сбое после успешного delete.
//
// @sk-task api-consistency-pass#T1.1: новый optional capability для atomic UpdateDocument (RQ-005, AC-008)
type TransactionalDocumentStore interface {
	// BeginTx открывает новую транзакцию.
	// Возвращает ошибку, если транзакция не может быть начата.
	BeginTx(ctx context.Context) (TransactionalTx, error)
}

// CollectionManager — опциональная capability VectorStore для управления жизненным циклом коллекции.
// Реализации, поддерживающие управление коллекциями, должны реализовывать этот интерфейс дополнительно
// (без ломки существующего контракта VectorStore).
//
// @ds-task T1.1: Добавить интерфейс CollectionManager в domain (AC-006, DEC-001)
type CollectionManager interface {
	// CreateCollection создаёт коллекцию в хранилище.
	// Idempotent: повторный вызов при уже существующей коллекции возвращает nil.
	CreateCollection(ctx context.Context) error

	// DeleteCollection удаляет коллекцию из хранилища.
	// Idempotent: возвращает nil если коллекция не существует (404).
	DeleteCollection(ctx context.Context) error

	// CollectionExists проверяет существование коллекции.
	// Возвращает (true, nil) если коллекция существует, (false, nil) если нет,
	// (false, error) при сетевой или серверной ошибке.
	CollectionExists(ctx context.Context) (bool, error)
}

// HybridSearcherWithFilters расширяет HybridSearcher фильтрами для гибридного поиска.
type HybridSearcherWithFilters interface {
	HybridSearcher

	// SearchHybridWithParentIDFilter выполняет гибридный поиск с фильтрацией по ParentID.
	SearchHybridWithParentIDFilter(ctx context.Context, query string, embedding []float64, topK int, config HybridConfig, filter ParentIDFilter) (RetrievalResult, error)

	// SearchHybridWithMetadataFilter выполняет гибридный поиск с фильтрацией по метаданным.
	SearchHybridWithMetadataFilter(ctx context.Context, query string, embedding []float64, topK int, config HybridConfig, filter MetadataFilter) (RetrievalResult, error)
}
