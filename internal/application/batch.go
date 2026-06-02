package application

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T1.2: рефактор — IndexBatch как тонкая обёртка над processDocsConcurrently (DEC-004, RQ-004)
// @sk-task api-consistency-pass#T3.1: shared processDocumentOp между Index и IndexBatch (DEC-004, RQ-004, AC-006)
//
// IndexBatch индексирует набор документов параллельно и возвращает aggregate
// результат (успешные + ошибки по документам). batchSize интерпретируется как
// желаемая concurrency: при batchSize <= 0 используется p.indexConcurrency.
//
// Семантика: best-effort — не отменяет siblings при ошибке отдельного документа.
// Это отличие от Index, который fail-fast.
func (p *Pipeline) IndexBatch(ctx context.Context, docs []domain.Document, batchSize int) (*domain.IndexBatchResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	concurrency := p.indexConcurrency
	if batchSize > 0 {
		concurrency = batchSize
	}

	processor := func(procCtx context.Context, doc domain.Document) error {
		return p.processDocumentOp(procCtx, "IndexBatch", doc)
	}

	successful, failed, ctxErr := processDocsConcurrently(
		ctx,
		docs,
		concurrency,
		p.indexBatchRateLimit,
		p.indexBatchRateLimitPerWorker,
		processor,
	)

	result := &domain.IndexBatchResult{
		Successful:    successful,
		Errors:        failed,
		ProcessedCount: len(successful) + len(failed),
	}

	if ctxErr != nil {
		return result, ctxErr
	}

	return result, nil
}
