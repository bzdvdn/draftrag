package application

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// --- Mock store: implements domain.VectorStore + domain.TransactionalDocumentStore ---

// txMockStore — in-memory store, поддерживающий TransactionalDocumentStore capability.
// Покрывает T3.2 transactional path в unit-тестах (раньше — только integration test).
type txMockStore struct {
	tx *txMockTx
}

func (s *txMockStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (s *txMockStore) Delete(_ context.Context, _ string) error       { return nil }
func (s *txMockStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

func (s *txMockStore) BeginTx(_ context.Context) (domain.TransactionalTx, error) {
	if s.tx == nil {
		s.tx = &txMockTx{}
	}
	return s.tx, nil
}

// txMockTx — TransactionalTx-реализация с счётчиками операций.
type txMockTx struct {
	upserts   int
	deletes   int
	commits   int
	rollbacks int

	// injectErr — если не nil, возвращается на следующей операции.
	injectErr error
}

func (t *txMockTx) Upsert(_ context.Context, _ domain.Chunk) error {
	if t.injectErr != nil {
		err := t.injectErr
		t.injectErr = nil
		return err
	}
	t.upserts++
	return nil
}

func (t *txMockTx) DeleteByParentID(_ context.Context, _ string) error {
	if t.injectErr != nil {
		err := t.injectErr
		t.injectErr = nil
		return err
	}
	t.deletes++
	return nil
}

func (t *txMockTx) Commit() error {
	if t.injectErr != nil {
		err := t.injectErr
		t.injectErr = nil
		return err
	}
	t.commits++
	return nil
}

func (t *txMockTx) Rollback() error {
	t.rollbacks++
	return nil
}

// @sk-test api-consistency-pass#T4.1-coverage: покрывает updateDocumentAtomicTransactional
// (0% → covered) на happy path. Mock store реализует TransactionalDocumentStore →
// dispatcher идёт в transactional ветку: produceChunks → BeginTx → tx.DeleteByParentID
// → tx.Upsert × N → tx.Commit. Rollback не вызывается.
func TestPipeline_UpdateDocument_Transactional_HappyPath(t *testing.T) {
	store := &txMockStore{tx: &txMockTx{}}
	p := NewPipeline(store, testLLM{}, &testEmbedder{})

	if err := p.UpdateDocument(context.Background(), domain.Document{ID: "d-1", Content: "hello"}); err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}

	tx := store.tx
	if tx.commits != 1 {
		t.Errorf("expected 1 Commit, got %d", tx.commits)
	}
	if tx.rollbacks != 0 {
		t.Errorf("expected 0 Rollback, got %d", tx.rollbacks)
	}
	if tx.deletes != 1 {
		t.Errorf("expected 1 tx.DeleteByParentID, got %d", tx.deletes)
	}
	if tx.upserts < 1 {
		t.Errorf("expected >=1 tx.Upsert, got %d", tx.upserts)
	}
}

// @sk-test api-consistency-pass#T4.1-coverage: покрывает updateDocumentAtomicTransactional
// на Rollback path при ошибке tx.Commit. В этом случае tx.Rollback должен быть вызван
// через deferred safety net.
func TestPipeline_UpdateDocument_Transactional_CommitErrorTriggersRollback(t *testing.T) {
	store := &txMockStore{tx: &txMockTx{injectErr: errors.New("commit failed")}}
	p := NewPipeline(store, testLLM{}, &testEmbedder{})

	err := p.UpdateDocument(context.Background(), domain.Document{ID: "d-1", Content: "hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "commit failed") {
		t.Errorf("expected underlying 'commit failed' in error, got %v", err)
	}

	tx := store.tx
	if tx.commits != 0 {
		t.Errorf("expected 0 successful Commit, got %d", tx.commits)
	}
	if tx.rollbacks != 1 {
		t.Errorf("expected 1 Rollback via deferred safety net, got %d", tx.rollbacks)
	}
}

// @sk-test api-consistency-pass#T4.1-coverage: покрывает updateDocumentAtomicTransactional
// на Rollback path при ошибке tx.Upsert.
func TestPipeline_UpdateDocument_Transactional_UpsertErrorTriggersRollback(t *testing.T) {
	store := &txMockStore{tx: &txMockTx{injectErr: errors.New("upsert failed")}}
	p := NewPipeline(store, testLLM{}, &testEmbedder{})

	err := p.UpdateDocument(context.Background(), domain.Document{ID: "d-1", Content: "hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "upsert failed") {
		t.Errorf("expected underlying 'upsert failed' in error, got %v", err)
	}

	tx := store.tx
	if tx.commits != 0 {
		t.Errorf("expected 0 Commit, got %d", tx.commits)
	}
	if tx.rollbacks != 1 {
		t.Errorf("expected 1 Rollback, got %d", tx.rollbacks)
	}
}

// @sk-test api-consistency-pass#T4.1-coverage: Index использует новую T3.4 perWorker
// signature — вызов должен пройти happy path.
func TestPipeline_Index_PerWorker_HappyPath(t *testing.T) {
	p := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		testLLM{},
		&testEmbedder{},
		PipelineConfig{IndexConcurrency: 2, IndexBatchRateLimit: 1000},
	)
	if err := p.Index(context.Background(), []domain.Document{{ID: "a", Content: "x"}}); err != nil {
		t.Fatalf("Index: %v", err)
	}
}

// @sk-test api-consistency-pass#T4.1-coverage: processDocsConcurrently с cancelled ctx
// на входе — worker pool обнаруживает отмену до запуска горутин, ctxErr возвращается,
// docs не обрабатываются.
func TestProcessDocsConcurrently_CancelledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	docs := []domain.Document{{ID: "d1", Content: "hello"}}
	successful, failed, ctxErr := processDocsConcurrently(
		ctx, docs, 1, 0, false,
		func(_ context.Context, _ domain.Document) error { return nil },
	)

	if ctxErr == nil {
		t.Errorf("expected non-nil ctxErr, got nil")
	}
	if !errors.Is(ctxErr, context.Canceled) {
		t.Errorf("expected ctxErr=context.Canceled, got %v", ctxErr)
	}
	if len(successful) != 0 {
		t.Errorf("expected 0 successful, got %d", len(successful))
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed (no goroutine started), got %d", len(failed))
	}
}

// contains — простая substring проверка.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// fmt guard to keep import alive.
var _ = fmt.Sprintf
