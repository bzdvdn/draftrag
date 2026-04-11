package domain

import (
	"context"
)

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

// LLMProvider определяет интерфейс для генерации текста через LLM.
type LLMProvider interface {
	// Generate генерирует ответ на основе system и user сообщений.
	Generate(ctx context.Context, systemPrompt, userMessage string) (string, error)
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

// Embedder определяет интерфейс для преобразования текста в векторное представление.
type Embedder interface {
	// Embed преобразует текст в embedding-вектор фиксированной размерности.
	Embed(ctx context.Context, text string) ([]float64, error)
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

// DocumentStore — опциональная capability VectorStore для удаления документа целиком по ParentID.
type DocumentStore interface {
	VectorStore
	// DeleteByParentID удаляет все чанки с указанным ParentID.
	DeleteByParentID(ctx context.Context, parentID string) error
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
