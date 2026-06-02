package draftrag

import (
	"errors"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T3.1: Переэкспорт sentinel-ошибок в public API (AC-009)
var (
	// ErrEmptyDocument возвращается, если документ нельзя проиндексировать из-за пустого содержимого.
	ErrEmptyDocument = domain.ErrEmptyDocumentContent
	// ErrEmptyQuery возвращается, если Pipeline.Query* вызывается с пустым вопросом.
	ErrEmptyQuery = domain.ErrEmptyQueryText
	// ErrInvalidTopK возвращается, если topK <= 0.
	ErrInvalidTopK = domain.ErrInvalidQueryTopK
	// ErrInvalidEmbedderConfig возвращается при невалидной конфигурации Embedder.
	// Ошибка предназначена для проверок через errors.Is.
	ErrInvalidEmbedderConfig = errors.New("invalid embedder config")
	// ErrInvalidLLMConfig возвращается при невалидной конфигурации LLMProvider.
	// Ошибка предназначена для проверок через errors.Is.
	ErrInvalidLLMConfig = errors.New("invalid llm config")
	// ErrInvalidChunkerConfig возвращается при невалидной конфигурации Chunker.
	// Ошибка предназначена для проверок через errors.Is.
	ErrInvalidChunkerConfig = errors.New("invalid chunker config")

	// ErrEmbeddingDimensionMismatch возвращается, если размерность embedding-вектора не соответствует ожидаемой.
	//
	// Ошибка предназначена для проверок через errors.Is.
	ErrEmbeddingDimensionMismatch = domain.ErrEmbeddingDimensionMismatch

	// ErrUpdateNotAtomic возвращается, если UpdateDocument завершился частично:
	// старые чанки удалены, а новые не удалось проиндексировать.
	// Vector store не поддерживает транзакции — рекомендуется re-Index.
	//
	// Ошибка предназначена для проверок через errors.Is.
	ErrUpdateNotAtomic = domain.ErrUpdateNotAtomic

	// ErrFiltersNotSupported возвращается, если pipeline-метод с фильтрами вызван,
	// но используемый VectorStore не поддерживает filters capability.
	ErrFiltersNotSupported = errors.New("filters not supported")

	// ErrInvalidVectorStoreConfig возвращается при невалидной конфигурации VectorStore.
	// Ошибка предназначена для проверок через errors.Is.
	ErrInvalidVectorStoreConfig = errors.New("invalid vector store config")
)
