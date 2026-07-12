package middleware

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// PIIDetectorMiddleware создаёт middleware, которая применяет PII redaction
// к содержимому документов (stage=chunking) и запросов (stage=embed/search/generate).
func PIIDetectorMiddleware(detector domain.PIIDetector) domain.Middleware {
	return func(next domain.Handler) domain.Handler {
		return func(ctx context.Context, data domain.StageData) (domain.StageData, error) {
			switch data.Stage {
			case domain.HookStageChunking:
				data.Document.Content = detector.Detect(data.Document.Content)
			case domain.HookStageEmbed, domain.HookStageSearch, domain.HookStageGenerate:
				if data.Query != "" {
					data.Query = detector.Detect(data.Query)
				}
			}
			return next(ctx, data)
		}
	}
}
