package draftrag

import (
	"fmt"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// PineconeOptions задаёт опции для подключения к Pinecone.
type PineconeOptions struct {
	// APIKey — API ключ Pinecone (обязательно).
	APIKey string
	// Environment — окружение Pinecone (например, "us-east-1-aws").
	Environment string
	// ProjectID — ID проекта в Pinecone.
	ProjectID string
	// IndexName — имя индекса (обязательно).
	IndexName string
	// Dimension — размерность векторов (обязательно, > 0).
	Dimension int
	// Cloud — облачный провайдер (по умолчанию: "aws").
	Cloud string
	// Region — регион облака (по умолчанию: "us-west-2").
	Region string
	// Timeout — HTTP таймаут (по умолчанию: 30s).
	Timeout time.Duration
}

// Validate проверяет корректность опций.
func (o PineconeOptions) Validate() error {
	if o.APIKey == "" {
		return fmt.Errorf("APIKey is required")
	}
	if o.IndexName == "" {
		return fmt.Errorf("IndexName is required")
	}
	if o.Dimension <= 0 {
		return fmt.Errorf("Dimension must be > 0")
	}
	return nil
}

// NewPineconeStore создаёт новый VectorStore на базе Pinecone.
func NewPineconeStore(opts PineconeOptions) (VectorStore, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	return vectorstore.NewPineconeStore(vectorstore.PineconeOptions{
		APIKey:      opts.APIKey,
		Environment: opts.Environment,
		ProjectID:   opts.ProjectID,
		IndexName:   opts.IndexName,
		Dimension:   opts.Dimension,
		Cloud:       opts.Cloud,
		Region:      opts.Region,
		Timeout:     opts.Timeout,
	})
}
