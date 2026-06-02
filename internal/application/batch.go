package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// IndexBatch индексирует документы параллельно с ограничением concurrency и rate limiting.
//
// @ds-task T2.1: Реализовать IndexBatch с worker pool (AC-001, AC-003, AC-004, AC-005)
// @ds-task T2.2: Добавить rate limiter в IndexBatch (AC-002)
func (p *Pipeline) IndexBatch(ctx context.Context, docs []domain.Document, batchSize int) (*domain.IndexBatchResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// batchSize в публичном API трактуем как “сколько документов обрабатывать параллельно”.
	// Если не задан — используем дефолт pipeline.
	concurrency := p.indexConcurrency
	if batchSize > 0 {
		concurrency = batchSize
	}

	result := &domain.IndexBatchResult{
		Successful: make([]domain.Document, 0, len(docs)),
		Errors:     make([]domain.IndexBatchError, 0),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Семафор для ограничения concurrency
	semaphore := make(chan struct{}, concurrency)

	// Rate limiter: token bucket
	// @ds-task T2.2: Token bucket rate limiter для вызовов Embed (AC-002, DEC-002)
	rateLimitInterval := time.Second / time.Duration(p.indexBatchRateLimit)
	rateLimiter := time.NewTicker(rateLimitInterval)
	defer rateLimiter.Stop()

	// Канал для graceful shutdown при ошибке контекста
	done := make(chan struct{})
	defer close(done)

	// Отслеживаем была ли отмена контекста
	var ctxErr error
	var ctxMu sync.Mutex

	for _, doc := range docs {
		// Проверка отмены контекста перед обработкой каждого документа
		if err := ctx.Err(); err != nil {
			mu.Lock()
			result.ProcessedCount = len(result.Successful) + len(result.Errors)
			mu.Unlock()
			return result, err
		}

		// Проверка валидности документа
		if err := doc.Validate(); err != nil {
			mu.Lock()
			result.Errors = append(result.Errors, domain.IndexBatchError{
				DocumentID: doc.ID,
				Error:      err,
			})
			result.ProcessedCount++
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(d domain.Document) {
			defer wg.Done()

			// Ожидание семафора для ограничения concurrency
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Проверка отмены контекста внутри worker'а
			if err := ctx.Err(); err != nil {
				ctxMu.Lock()
				if ctxErr == nil {
					ctxErr = err
				}
				ctxMu.Unlock()
				mu.Lock()
				result.Errors = append(result.Errors, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      err,
				})
				result.ProcessedCount++
				mu.Unlock()
				return
			}

			// Rate limiting: ожидание токена
			select {
			case <-rateLimiter.C:
				// продолжаем
			case <-ctx.Done():
				err := ctx.Err()
				ctxMu.Lock()
				if ctxErr == nil {
					ctxErr = err
				}
				ctxMu.Unlock()
				mu.Lock()
				result.Errors = append(result.Errors, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      err,
				})
				result.ProcessedCount++
				mu.Unlock()
				return
			}

			// Обработка документа с chunking и embedding
			docErr := p.processDocumentForBatch(ctx, d)

			mu.Lock()
			if docErr != nil {
				result.Errors = append(result.Errors, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      docErr,
				})
			} else {
				result.Successful = append(result.Successful, d)
			}
			result.ProcessedCount++
			mu.Unlock()
		}(doc)
	}

	wg.Wait()

	// Если был отменён контекст, возвращаем ошибку с partial results
	if ctxErr != nil {
		return result, ctxErr
	}

	return result, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// processDocumentForBatch обрабатывает один документ для batch-индексации.
// Выполняет chunking (если настроен), embedding и upsert всех чанков.
//
// @ds-task T2.1: Вспомогательный метод для обработки документа в worker (AC-001, AC-005)
func (p *Pipeline) processDocumentForBatch(ctx context.Context, doc domain.Document) error {
	if p.chunker != nil {
		chunkStart := time.Now()
		p.hookStart(ctx, "IndexBatch", domain.HookStageChunking)
		chunks, err := p.chunker.Chunk(ctx, doc)
		p.hookEnd(ctx, "IndexBatch", domain.HookStageChunking, chunkStart, err)
		if err != nil {
			return err
		}
		return p.indexChunks(ctx, "IndexBatch", chunks)
	}

	// Legacy path: один чанк на документ
	embedStart := time.Now()
	p.hookStart(ctx, "IndexBatch", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, doc.Content)
	p.hookEnd(ctx, "IndexBatch", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return err
	}

	chunk := domain.Chunk{
		ID:        fmt.Sprintf("%s#%d", doc.ID, 0),
		Content:   doc.Content,
		ParentID:  doc.ID,
		Embedding: embedding,
		Position:  0,
	}
	if err := chunk.Validate(); err != nil {
		return err
	}

	return p.store.Upsert(ctx, chunk)
}

