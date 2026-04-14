# Batch indexing Data Model

## Phase Contract

Inputs: spec, plan.
Outputs: data model описание сущностей.

## Сущности

### IndexBatchResult

Тип результата batch-индексации, возвращаемый методом `IndexBatch`.

**Поля:**
- `Successful []Document` — документы, успешно проиндексированные (все чанки сохранены)
- `Errors []IndexBatchError` — ошибки по документам (partial failure)
- `ProcessedCount int` — общее количество обработанных документов (успешных + с ошибками)

**Жизненный цикл:** создаётся в начале `IndexBatch`, наполняется по мере работы workers, возвращается вызывающему. Не хранится persistently.

**Инвариант:** `ProcessedCount == len(Successful) + len(Errors)` (всегда true после завершения, даже при отмене контекста)

**AC покрытие:** AC-003 (частичные ошибки), AC-004 (отмена контекста — partial results)

### IndexBatchError

Тип ошибки для конкретного документа в batch.

**Поля:**
- `DocumentID string` — идентификатор документа, который не удалось проиндексировать
- `Error error` — оригинальная ошибка (embed, chunking или upsert)

**Жизненный цикл:** создаётся worker'ом при ошибке обработки документа, добавляется в `IndexBatchResult.Errors`.

**Инвариант:** `DocumentID` не пустой, `Error` не nil

**AC покрытие:** AC-003 (идентификация failed документов)

## Не вводит persisted state

Эта фича не добавляет новых persisted entities или storage. `IndexBatchResult` — transient тип для возврата результата операции. VectorStore остаётся единственным источником persisted state.

## Связь с существующими сущностями

- `IndexBatchResult.Successful` содержит `Document` (существующий тип)
- `IndexBatchResult` создаётся и используется `application.Pipeline`
- Результат операции (`Chunk` с `Embedding`) сохраняется в `VectorStore` (существующий интерфейс)

## Контракты

- API boundaries: не меняются — `IndexBatch` — новый метод на существующем типе `Pipeline`
- Event contracts: не меняются — hooks `StageStart`/`StageEnd` уже существуют
