# VectorStore pgvector: production-ready (migrations, индексы, фильтры, лимиты/таймауты) Задачи

## Phase Contract

Inputs: `.draftspec/plans/vectorstore-pgvector-production/plan.md`, `.draftspec/plans/vectorstore-pgvector-production/data-model.md`, `.draftspec/specs/vectorstore-pgvector-production/spec.md`.
Outputs: упорядоченные исполнимые задачи с явным покрытием `AC-*`.
Stop if: хотя бы один `AC-*` не удаётся покрыть конкретной задачей без догадок по реализации.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/interfaces.go | T1.1 |
| internal/infrastructure/vectorstore/pgvector.go | T2.1, T2.4 |
| internal/application/pipeline.go | T2.2 |
| pkg/draftrag/pgvector.go | T2.3, T3.3 |
| pkg/draftrag/pgvector_migrate.go (new) | T2.3 |
| internal/infrastructure/vectorstore/pgvector_test.go | T3.2 |
| pkg/draftrag/pgvector_test.go | T3.2 |
| internal/application/pipeline_options_test.go | T3.1 |
| internal/application/pipeline_test.go (new) | T3.1 |

## Фаза 1: Основа

Цель: зафиксировать новые capability-контракты и runtime ограничения так, чтобы дальнейшая реализация не потребовала ломки существующего API.

- [x] T1.1 Добавить capability интерфейс фильтров `VectorStoreWithFilters` и `ParentIDFilter` (DEC-001, RQ-009). Touches: internal/domain/interfaces.go
- [x] T1.2 Ввести runtime options и валидацию лимитов/таймаутов (MaxTopK/MaxParentIDs/MaxContentBytes + дефолтные timeouts) (RQ-010, RQ-011). Touches: pkg/draftrag/pgvector.go

## Фаза 2: Основная реализация

Цель: реализовать production-функциональность (фильтр ParentID, миграции, индексы, таймауты) и сохранить совместимость.

- [x] T2.1 Реализовать `SearchWithFilter` в pgvector store с `WHERE parent_id = ANY($2)` и валидацией списка ParentIDs (AC-003). Touches: internal/infrastructure/vectorstore/pgvector.go
- [x] T2.2 Добавить pipeline-методы `QueryWithParentIDs`/`AnswerWithParentIDs`/`AnswerWithCitationsWithParentIDs` (или эквивалент) с обработкой отсутствия capability (`ErrFiltersNotSupported`) (AC-003, AC-005, DEC-001). Touches: internal/application/pipeline.go
- [x] T2.3 Реализовать версионированный migrator `MigratePGVector` (schema_migrations + V1/V2) и сохранить `SetupPGVector` как thin wrapper или deprecated alias (AC-001, AC-002, DEC-002, DEC-003). Touches: pkg/draftrag/pgvector.go, pkg/draftrag/pgvector_migrate.go
- [x] T2.4 Обновить `Upsert` для поддержки production-колонок (`updated_at = now()` на update; insert полагается на defaults) и убедиться, что runtime время выполнения ограничивается контекстом (AC-004, DM-001). Touches: internal/infrastructure/vectorstore/pgvector.go

## Фаза 3: Проверка

Цель: доказать корректность по `AC-*` и оставить изменение в reviewable/tested состоянии.

- [x] T3.1 Добавить unit tests: (1) pipeline возвращает явную ошибку при store без filters capability, (2) валидация лимитов `topK`/`parentIDs`/`content` (AC-004, AC-005). Touches: internal/application/pipeline_options_test.go, internal/application/pipeline_test.go
- [x] T3.2 Добавить/обновить integration tests (skip если нет pgvector/прав): (1) миграции идемпотентны и создают объекты, (2) индекс нужного типа, (3) фильтр ParentID работает (AC-001, AC-002, AC-003). Touches: internal/infrastructure/vectorstore/pgvector_test.go, pkg/draftrag/pgvector_test.go
- [x] T3.3 Обновить godoc/документацию `pkg/draftrag/pgvector.go`: новый migrator, стратегия смены IndexMethod (drop+create), и рекомендации по запуску миграций в deploy job (AC-001, AC-002, DEC-003). Touches: pkg/draftrag/pgvector.go

## Покрытие критериев приемки

- AC-001 -> T2.3, T3.2, T3.3
- AC-002 -> T2.3, T3.2, T3.3
- AC-003 -> T1.1, T2.1, T2.2, T3.2
- AC-004 -> T1.2, T2.4, T3.1
- AC-005 -> T1.1, T2.2, T3.1
