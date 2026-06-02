package application

import (
	"context"
	"sync"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// docProcessor обрабатывает один документ. Возвращает nil при успехе, error при сбое.
// Реализация не должна делать assumptions о concurrency — она вызывается в одной
// горутине на документ.
type docProcessor func(ctx context.Context, doc domain.Document) error

// processDocsConcurrently обрабатывает документы параллельно с ограничением
// concurrency и rate limiting.
//
// Семантика:
// - rateLimit — максимальное количество вызовов processor в секунду.
//   - perWorker=false (default): один общий ticker с интервалом
//     time.Second / rateLimit. Все worker'ы ждут тик перед вызовом processor.
//     При rateLimit=10 и concurrency=4 общий rate = 10/sec (rateLimit на пул).
//   - perWorker=true: каждый worker создаёт свой собственный ticker с тем же
//     интервалом. При rateLimit=10 и concurrency=4 суммарный rate = 40/sec
//     (rateLimit на каждого worker'а).
//   При rateLimit <= 0 rate limiting отключён в обоих режимах.
// - concurrency — максимальное количество одновременно работающих worker'ов.
//   При concurrency <= 0 используется 1 (sequential).
// - processor вызывается после успешного получения семафора и тика rate limiter'а.
// - Каждый документ валидируется ДО запуска горутины; невалидные документы сразу
//   попадают в failed без конкурентного доступа.
// - При отмене контекста уже запущенные worker'ы корректно завершаются (select
//   на ctx.Done), новые не запускаются.
//
// Возвращает:
// - successful — документы, для которых processor вернул nil.
// - failed — ошибки по документам (включая ошибки валидации и ctx-cancellation).
// - ctxErr — первая зафиксированная ошибка контекста (если была); nil если ctx не отменялся.
//
// Потокобезопасность: concurrent appends к successful/failed защищены mu.
//
// @sk-task api-consistency-pass#T1.2: выделение worker pool из IndexBatch (DEC-004, RQ-004)
// @sk-task api-consistency-pass#T3.4: per-worker rate-limiter toggle (DEC-007, RQ-007, AC-011, AC-012)
func processDocsConcurrently(
	ctx context.Context,
	docs []domain.Document,
	concurrency int,
	rateLimit int,
	perWorker bool,
	processor docProcessor,
) ([]domain.Document, []domain.IndexBatchError, error) {
	if concurrency <= 0 {
		concurrency = 1
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	successful := make([]domain.Document, 0, len(docs))
	failed := make([]domain.IndexBatchError, 0)

	var ctxErr error
	var ctxMu sync.Mutex

	semaphore := make(chan struct{}, concurrency)

	// Shared rate-limiter (perWorker=false). Создаётся один раз и Stop'ается
	// при выходе из функции. Все worker'ы ждут один и тот же тик.
	var sharedLimiter *time.Ticker
	if rateLimit > 0 && !perWorker {
		interval := time.Second / time.Duration(rateLimit)
		if interval <= 0 {
			interval = time.Millisecond
		}
		sharedLimiter = time.NewTicker(interval)
		defer sharedLimiter.Stop()
	}

	// Per-worker rate-limiter (perWorker=true): каждый worker создаёт свой ticker
	// внутри своей горутины. sharedInterval вычисляется один раз и используется
	// всеми worker'ами.
	var sharedInterval time.Duration
	if rateLimit > 0 && perWorker {
		sharedInterval = time.Second / time.Duration(rateLimit)
		if sharedInterval <= 0 {
			sharedInterval = time.Millisecond
		}
	}

	for _, doc := range docs {
		if err := ctx.Err(); err != nil {
			ctxMu.Lock()
			if ctxErr == nil {
				ctxErr = err
			}
			ctxMu.Unlock()
			break
		}

		if err := doc.Validate(); err != nil {
			mu.Lock()
			failed = append(failed, domain.IndexBatchError{
				DocumentID: doc.ID,
				Error:      err,
			})
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(d domain.Document) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				err := ctx.Err()
				ctxMu.Lock()
				if ctxErr == nil {
					ctxErr = err
				}
				ctxMu.Unlock()
				mu.Lock()
				failed = append(failed, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      err,
				})
				mu.Unlock()
				return
			}
			defer func() { <-semaphore }()

			// Rate-limiter wait: shared или per-worker.
			if sharedLimiter != nil {
				select {
				case <-sharedLimiter.C:
				case <-ctx.Done():
					err := ctx.Err()
					ctxMu.Lock()
					if ctxErr == nil {
						ctxErr = err
					}
					ctxMu.Unlock()
					mu.Lock()
					failed = append(failed, domain.IndexBatchError{
						DocumentID: d.ID,
						Error:      err,
					})
					mu.Unlock()
					return
				}
			} else if sharedInterval > 0 {
				// perWorker=true: создаём локальный ticker для этого worker'а.
				localLimiter := time.NewTicker(sharedInterval)
				defer localLimiter.Stop()
				select {
				case <-localLimiter.C:
				case <-ctx.Done():
					err := ctx.Err()
					ctxMu.Lock()
					if ctxErr == nil {
						ctxErr = err
					}
					ctxMu.Unlock()
					mu.Lock()
					failed = append(failed, domain.IndexBatchError{
						DocumentID: d.ID,
						Error:      err,
					})
					mu.Unlock()
					return
				}
			}

			if procErr := processor(ctx, d); procErr != nil {
				mu.Lock()
				failed = append(failed, domain.IndexBatchError{
					DocumentID: d.ID,
					Error:      procErr,
				})
				mu.Unlock()
			} else {
				mu.Lock()
				successful = append(successful, d)
				mu.Unlock()
			}
		}(doc)
	}

	wg.Wait()

	return successful, failed, ctxErr
}
