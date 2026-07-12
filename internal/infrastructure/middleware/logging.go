package middleware

import (
	"context"
	"log"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// LoggingMiddleware логирует каждую стадию pipeline с длительностью.
// Использует стандартный log пакет. Для structured logging используйте
// domain.Logger через кастомную middleware.
func LoggingMiddleware() domain.Middleware {
	return func(next domain.Handler) domain.Handler {
		return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
			start := time.Now()
			result, err := next(ctx, data)
			duration := time.Since(start)
			if err != nil {
				log.Printf("[middleware] stage=%s op=%s duration=%s error=%v", data.Stage, data.Operation, duration, err)
			} else {
				log.Printf("[middleware] stage=%s op=%s duration=%s", data.Stage, data.Operation, duration)
			}
			return result, err
		}
	}
}
