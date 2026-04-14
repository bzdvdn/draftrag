# Поддержка Milvus как векторного хранилища — Задачи

## Phase Contract

Inputs: plan.md, data-model.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех 8 acceptance criteria.
Stop if: задачи расплывчаты или coverage не удаётся сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/vectorstore/milvus.go | T1.1, T2.1, T2.2, T2.3, T3.1, T3.2 |
| internal/infrastructure/vectorstore/milvus_test.go | T4.1 |

## Фаза 1: Каркас

Цель: создать минимальную основу файла, чтобы все последующие методы можно было добавлять без структурных изменений.

- [x] T1.1 Создать `milvus.go` с типом `MilvusStore`, конструктором `NewMilvusStore`, хелпером `doRequest` и compile-time assertions — файл компилируется, `go build ./...` проходит, `var _ domain.X = (*MilvusStore)(nil)` присутствуют (AC-007, AC-008, DEC-001, DEC-002, DEC-004). Touches: internal/infrastructure/vectorstore/milvus.go

## Фаза 2: Базовые операции

Цель: реализовать Upsert, Delete и Search — базовый `domain.VectorStore` и `domain.DocumentStore`.

- [x] T2.1 Реализовать `Upsert` — POST `/v2/vectordb/entities/upsert` с DM-002 Upsert body; `metadata` сериализуется через `json.Marshal`; возвращает ошибку при `code != 0` или HTTP 4xx/5xx (AC-001, DEC-003). Touches: internal/infrastructure/vectorstore/milvus.go

- [x] T2.2 Реализовать `Delete` и `DeleteByParentID` — POST `/v2/vectordb/entities/delete`; фильтр `id == "<id>"` и `parent_id == "<id>"` соответственно (AC-002, AC-006). Touches: internal/infrastructure/vectorstore/milvus.go

- [x] T2.3 Реализовать `Search` — POST `/v2/vectordb/entities/search` с DM-002 Search body; десериализовать DM-003 в `[]domain.RetrievedChunk`; пустой `data` → пустой слайс, не ошибка (AC-003, DEC-003). Touches: internal/infrastructure/vectorstore/milvus.go

## Фаза 3: Фильтры

Цель: реализовать методы с фильтрами — полный `domain.VectorStoreWithFilters`.

- [x] T3.1 Реализовать `SearchWithFilter` — добавляет `"filter": "parent_id in [\"a\",\"b\"]"` при непустых `ParentIDs`; при пустом срезе поле `filter` опускается (AC-004). Touches: internal/infrastructure/vectorstore/milvus.go

- [x] T3.2 Реализовать `SearchWithMetadataFilter` — строит AND-выражение `metadata["k"] == "v" && ...` из `Fields`; при пустом `Fields` поле `filter` опускается (AC-005, DEC-003). Touches: internal/infrastructure/vectorstore/milvus.go

## Фаза 4: Тесты

Цель: доказать корректность всех методов и error-path через unit-тесты с мок-сервером.

- [x] T4.1 Создать `milvus_test.go` с `httptest.NewServer` — тесты для всех 8 AC: тело запроса Upsert, фильтры Delete/DeleteByParentID/SearchWithFilter/SearchWithMetadataFilter, количество чанков в Search-ответе, error-path (`code != 0`, HTTP 5xx); `go test -run TestMilvus` зелёный, coverage ≥ 60% (AC-001..AC-008). Touches: internal/infrastructure/vectorstore/milvus_test.go

## Покрытие критериев приемки

- AC-001 -> T2.1, T4.1
- AC-002 -> T2.2, T4.1
- AC-003 -> T2.3, T4.1
- AC-004 -> T3.1, T4.1
- AC-005 -> T3.2, T4.1
- AC-006 -> T2.2, T4.1
- AC-007 -> T1.1
- AC-008 -> T1.1, T4.1
