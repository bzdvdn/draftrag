# Document Lifecycle — План

## Цель

Добавить bulk-delete по parent_id во все 4 vector stores и expose `DeleteDocument` / `UpdateDocument` в Pipeline.

## Scope

- `internal/domain/interfaces.go` — интерфейс `DocumentStore`
- `internal/infrastructure/vectorstore/memory.go` — `DeleteByParentID`
- `internal/infrastructure/vectorstore/pgvector.go` — `DeleteByParentID`
- `internal/infrastructure/vectorstore/qdrant.go` — `DeleteByParentID`
- `internal/infrastructure/vectorstore/chromadb.go` — `DeleteByParentID`
- `internal/application/pipeline.go` — `DeleteDocument`, `UpdateDocument`
- `pkg/draftrag/draftrag.go` — публичный API

## Acceptance Approach

- AC-001, AC-002 → `DeleteByParentID` в qdrant/chromadb + mock-HTTP тесты
- AC-003 → server error тесты
- AC-004 → context cancellation тесты
- AC-005 → capability check в `Pipeline.DeleteDocument`

## Стратегия реализации

- DEC-001 Capability check через type assertion
  Why: не все stores поддерживают delete; не хочется обязывать implement interface
  Tradeoff: runtime check вместо compile-time
  Affects: internal/application/pipeline.go
  Validation: `ErrDeleteNotSupported` возвращается для stores без интерфейса

- DEC-002 Compile-time assertions для Qdrant и ChromaDB
  Why: явная гарантия что реализация не сломается при рефакторинге
  Tradeoff: ошибка компиляции при несоответствии (это хорошо)
  Affects: qdrant.go, chromadb.go
  Validation: `var _ domain.DocumentStore = (*QdrantStore)(nil)`

## Порядок реализации

1. `domain.DocumentStore` интерфейс
2. `DeleteByParentID` в memory и pgvector
3. `DeleteByParentID` в qdrant и chromadb
4. `Pipeline.DeleteDocument` / `UpdateDocument`
5. Публичный API в pkg/draftrag
6. Тесты для qdrant и chromadb

## Риски

- Риск: UpdateDocument теряет документ при сбое re-index после delete
  Mitigation: задокументировано в godoc и docs/pipeline.md

## Rollout и compatibility

- Additive; нет breaking changes.
- Stores без DeleteByParentID возвращают `ErrDeleteNotSupported` (не panic).
