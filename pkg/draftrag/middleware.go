package draftrag

import (
	"github.com/bzdvdn/draftrag/internal/infrastructure/middleware"
)

// NewLoggingMiddleware создаёт middleware, которая логирует каждую стадию pipeline.
// Использует стандартный log пакет.
func NewLoggingMiddleware() Middleware {
	return middleware.LoggingMiddleware()
}

// NewPIIDetectorMiddleware создаёт middleware, которая применяет PII redaction
// к содержимому документов и запросов на каждой стадии pipeline.
// detector — обязательный детектор PII (см. NewDefaultPIIDetector).
func NewPIIDetectorMiddleware(detector PIIDetector) Middleware {
	return middleware.PIIDetectorMiddleware(detector)
}
