package application

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task api-consistency-pass#T1.1: стаб для atomic UpdateDocument (RQ-005, AC-008, AC-009)
// @sk-task api-consistency-pass#T3.2: реализация transactional + best-effort веток (DEC-005, RQ-005, AC-008, AC-009)
//
// updateDocumentAtomic выполняет атомарное обновление документа: delete старых чанков
// + переиндексация. Использует transactional path, если underlying store реализует
// domain.TransactionalDocumentStore; иначе — best-effort path с возвратом
// domain.ErrUpdateNotAtomic при сбое после успешного delete.
//
// Контракт:
// - ctx обязателен; первый параметр; nil panic (как в других методах Pipeline).
// - doc валидируется ДО открытия транзакции (избегаем пустого tx).
// - Если store реализует TransactionalDocumentStore: BeginTx → DeleteByParentID +
//   Upsert всех чанков в tx → Commit; при любой ошибке — Rollback (через deferred
//   safety net) + wrapped error. Транзакция откатывается полностью: ни старые,
//   ни новые чанки не сохраняются частично.
// - Если store НЕ реализует TransactionalDocumentStore: DeleteByParentID + Index;
//   при ошибке Index после успешного delete — return ErrUpdateNotAtomic с wrapped
//   underlying error. Чанки, которые успели проиндексироваться до ошибки, остаются
//   в store; чанки, которые не успели — отсутствуют. Консистентность best-effort.
func (p *Pipeline) updateDocumentAtomic(ctx context.Context, doc domain.Document) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Валидируем doc ДО delete: для best-effort path иначе старые чанки
	// удалились бы, а новые не записались (ErrUpdateNotAtomic с пустым
	// content-ом не имеет смысла — лучше сразу вернуть ErrEmptyDocumentContent
	// и не трогать store).
	if err := doc.Validate(); err != nil {
		return err
	}

	if txStore, ok := p.store.(domain.TransactionalDocumentStore); ok {
		return p.updateDocumentAtomicTransactional(ctx, txStore, doc)
	}
	return p.updateDocumentAtomicBestEffort(ctx, doc)
}

// @sk-task api-consistency-pass#T3.2: transactional ветка — produceChunks → BeginTx → atomic upsert (DEC-005, AC-008)
//
// updateDocumentAtomicTransactional выполняет атомарное обновление через tx:
// produceChunks (chunk+embed) → BeginTx → DeleteByParentID (в tx) → Upsert всех
// чанков (в tx) → Commit. Любая ошибка между BeginTx и Commit приводит к
// Rollback через deferred safety net.
//
// produceChunks выполняется ДО BeginTx, чтобы embed-failures не открывали
// пустую транзакцию и не приводили к лишним Rollback'ам. Это гарантирует
// "in-store остаются старые чанки" при любой ошибке.
func (p *Pipeline) updateDocumentAtomicTransactional(
	ctx context.Context,
	txStore domain.TransactionalDocumentStore,
	doc domain.Document,
) error {
	chunks, err := p.produceChunks(ctx, "Update", doc)
	if err != nil {
		return fmt.Errorf("produce chunks: %w", err)
	}

	tx, err := txStore.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := tx.DeleteByParentID(ctx, doc.ID); err != nil {
		return fmt.Errorf("delete parent in tx: %w", err)
	}

	for _, ch := range chunks {
		if err := tx.Upsert(ctx, ch); err != nil {
			return fmt.Errorf("upsert chunk in tx: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true
	return nil
}

// @sk-task api-consistency-pass#T3.2: best-effort ветка — delete + index, ErrUpdateNotAtomic при сбое (DEC-005, AC-009)
//
// updateDocumentAtomicBestEffort — degraded path для store без TransactionalDocumentStore:
// DeleteByParentID → processDocumentOp (chunk+embed+upsert). При ошибке upsert
// (после успешного delete) возвращает ErrUpdateNotAtomic с wrapped underlying
// error, чтобы вызывающий код мог корректно классифицировать сбой через
// errors.Is.
func (p *Pipeline) updateDocumentAtomicBestEffort(
	ctx context.Context,
	doc domain.Document,
) error {
	ds, ok := p.store.(domain.DocumentStore)
	if !ok {
		return ErrDeleteNotSupported
	}
	if err := ds.DeleteByParentID(ctx, doc.ID); err != nil {
		return err
	}
	if err := p.processDocumentOp(ctx, "Update", doc); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrUpdateNotAtomic, err)
	}
	return nil
}
