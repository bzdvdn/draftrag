# Metadata filtering — Задачи

## Phase Contract

Inputs: `plans/metadata-filtering/plan.md`, `data-model.md`, `contracts/api.md`, `specs/metadata-filtering/summary.md`.
Outputs: упорядоченные исполнимые задачи с покрытием всех 5 AC.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/models.go` | T1.1, T1.2 |
| `internal/domain/interfaces.go` | T1.3 |
| `internal/infrastructure/vectorstore/pgvector.go` | T1.3, T2.1 |
| `internal/infrastructure/vectorstore/memory.go` | T2.2 |
| `internal/application/pipeline.go` | T3.1 |
| `pkg/draftrag/draftrag.go` | T3.2 |
| `internal/infrastructure/vectorstore/memory_test.go` | T4.1 |
| `internal/application/pipeline_test.go` | T4.1 |
| `pkg/draftrag/pgvector_runtime_test.go` | T4.2 |

---

## Фаза 1: Domain

Цель: определить новые domain-типы и расширить интерфейс — всё остальное блокируется этой фазой.

- [x] T1.1 Добавить тип `MetadataFilter{Fields map[string]string}` и sentinel `ErrFilterNotSupported` в domain — тип экспортирован, godoc на русском. `RQ-001`, `RQ-008`, `DEC-001` Touches: internal/domain/models.go
- [x] T1.2 Добавить поле `Metadata map[string]string` в `domain.Chunk` — nil означает «нет метаданных», godoc на русском. `DEC-005` Touches: internal/domain/models.go
- [x] T1.3 Расширить `VectorStoreWithFilters` методом `SearchWithMetadataFilter(ctx, embedding, topK, MetadataFilter) (RetrievalResult, error)` и обновить compile-time assert в pgvector.go. `RQ-002`, `DEC-001` Touches: internal/domain/interfaces.go, internal/infrastructure/vectorstore/pgvector.go

---

## Фаза 2: Infrastructure

Цель: реализовать `SearchWithMetadataFilter` в обоих бэкендах — pgvector и in-memory (параллельно).

- [x] T2.1 Реализовать `PGVectorStore.SearchWithMetadataFilter` — SQL `WHERE metadata @> $N::jsonb`; пустой `Fields` делегирует в `Search`; scan заполняет `Chunk.Metadata` из колонки `metadata`. `RQ-003`, `AC-001`, `AC-002`, `DEC-002` Touches: internal/infrastructure/vectorstore/pgvector.go
- [x] T2.2 Реализовать `InMemoryStore.SearchWithMetadataFilter` (и `SearchWithFilter`) с итерацией по chunks; AND-фильтр по `chunk.Metadata`; пустой `Fields` делегирует в `Search`. `RQ-004`, `AC-002`, `AC-005` Touches: internal/infrastructure/vectorstore/memory.go

---

## Фаза 3: Application и публичный API

Цель: провести фильтр сквозь application-слой и опубликовать два метода `Pipeline`.

- [x] T3.1 Добавить `QueryWithMetadataFilter` и `AnswerWithMetadataFilter` в `application.Pipeline` — type-assert на `VectorStoreWithFilters`; `ErrFiltersNotSupported` если store не поддерживает. `RQ-005`, `RQ-006`, `AC-003`, `DEC-003` Touches: internal/application/pipeline.go
- [x] T3.2 Добавить `MetadataFilter = domain.MetadataFilter` type alias и методы `QueryWithMetadataFilter`, `AnswerWithMetadataFilter` в `pkg/draftrag.Pipeline`; маппинг `application.ErrFiltersNotSupported -> ErrFiltersNotSupported`; godoc на русском. `RQ-005`, `RQ-006`, `RQ-007`, `RQ-008`, `AC-003` Touches: pkg/draftrag/draftrag.go

---

## Фаза 4: Валидация

Цель: доказать корректность всех 5 AC тестами; обеспечить `go build ./...` и `go test ./...` без ошибок.

- [x] T4.1 Добавить unit-тесты: `InMemoryStore.SearchWithMetadataFilter` (непустой фильтр, пустой фильтр, нет совпадений); application-layer тест с mock-store — фильтр доходит до `SearchWithMetadataFilter`; non-filter store возвращает `ErrFiltersNotSupported`. `AC-002`, `AC-003`, `AC-004`, `AC-005`, `DEC-003` Touches: internal/infrastructure/vectorstore/memory_test.go, internal/application/pipeline_test.go
- [x] T4.2 Добавить integration-тест pgvector: два класса документов с разным `Metadata["category"]`; `SearchWithMetadataFilter` возвращает только совпадающие ID; несуществующий фильтр — `len(chunks)==0 && err==nil`. `AC-001`, `AC-004`, `DEC-002` Touches: pkg/draftrag/pgvector_runtime_test.go

---

## Покрытие критериев приемки

- AC-001 -> T2.1, T4.2
- AC-002 -> T2.1, T2.2, T4.1
- AC-003 -> T3.1, T3.2, T4.1
- AC-004 -> T2.1, T4.1, T4.2
- AC-005 -> T1.3, T2.2, T4.1
