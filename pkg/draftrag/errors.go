package draftrag

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Sentinel errors returned by the public API.
//
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

	// ErrNilContext возвращается, если публичный метод вызван с nil context.
	//
	// @sk-task arch-generics#T1.1: sentinel для nil context guard (AC-002)
	ErrNilContext = errors.New("nil context")

	// @sk-task config-management#T1.2: sentinel для неизвестного YAML-ключа (RQ-004, AC-003)
	ErrUnknownConfigKey = errors.New("unknown config key")

	// @sk-task config-management#T1.2: sentinel для отсутствующего обязательного поля (RQ-005, AC-004)
	ErrMissingRequiredField = errors.New("missing required config field")

	// @sk-task sub-query-decomposition#T1.1: sentinel для SubDecompose без decomposer (AC-001, AC-006)
	ErrSubDecomposeNotSupported = errors.New("sub-query decomposition not supported: no QueryDecomposer configured")

	// @sk-task arch-issues#T1.2: sentinel для streaming c tools (AC-004)
	ErrToolsNotSupportedInStream = errors.New("tool calling is not supported in streaming mode")
)

// @sk-task arch-generics#T1.1: nil context guard helper (AC-002)
func checkCtx(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}
	return nil
}

func validateOptions(baseURL, apiKey, model string, timeout time.Duration, configErr error) error {
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("%w: APIKey is empty", configErr)
	}
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("%w: BaseURL is empty", configErr)
	}
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("%w: Model is empty", configErr)
	}
	if timeout < 0 {
		return fmt.Errorf("%w: Timeout must be >= 0", configErr)
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: BaseURL must include scheme and host", configErr)
	}
	return nil
}

func validateEmbedderOptions(baseURL, apiKey, model string, timeout time.Duration) error {
	return validateOptions(baseURL, apiKey, model, timeout, ErrInvalidEmbedderConfig)
}

func validateLLMOptions(baseURL, apiKey, model string, timeout time.Duration) error {
	return validateOptions(baseURL, apiKey, model, timeout, ErrInvalidLLMConfig)
}
