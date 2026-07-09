---
slug: metadata-filtering
status: completed
archived_at: 2026-04-08
---

# Архив: Metadata filtering

## Статус

Завершена полностью. Все 9 задач выполнены, verify: pass.

## Причина архивации

Фича реализована и верифицирована — произвольная фильтрация по метаданным документа при семантическом поиске.

## Реализованный scope

- `domain.MetadataFilter{Fields map[string]string}` — новый тип фильтра
- `domain.Chunk.Metadata map[string]string` — новое поле чанка
- `VectorStoreWithFilters.SearchWithMetadataFilter` — расширение интерфейса
- `PGVectorStore.SearchWithMetadataFilter` — SQL `WHERE metadata @> $N::jsonb` + Upsert пишет JSONB-колонку
- `InMemoryStore.SearchWithFilter` + `SearchWithMetadataFilter` — in-memory реализация (store теперь реализует `VectorStoreWithFilters`)
- `application.Pipeline.QueryWithMetadataFilter`, `AnswerWithMetadataFilter`
- `pkg/draftrag.Pipeline.QueryWithMetadataFilter`, `AnswerWithMetadataFilter` + `MetadataFilter` type alias
- Unit-тесты: 4 в `memory_test.go`, 4 в `pipeline_test.go`
- Integration-тест: `pgvector_runtime_test.go` (skip без `PGVECTOR_TEST_DSN`)

## Заметные отклонения от плана

- `InMemoryStore` получила и `SearchWithFilter` (не только `SearchWithMetadataFilter`) — потребовалось для полного соответствия расширенному `VectorStoreWithFilters`.
- Существующий тест `TestPipeline_QueryWithParentIDs_FiltersNotSupported` переключён на `noFilterStore` — следствие того, что `InMemoryStore` теперь реализует `VectorStoreWithFilters`.

## Ветка

`feature/metadata-filtering`
