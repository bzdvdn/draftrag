# Hierarchical Indices — Parent Document Retrieval: План

## Phase Contract

Inputs: `docs/specs/hierarchical-indices/spec.md`, `docs/specs/hierarchical-indices/inspect.md`, `internal/domain/models.go`, `internal/domain/interfaces.go`, `internal/application/pipeline.go`, `internal/infrastructure/vectorstore/memory.go`, `pkg/draftrag/draftrag.go`
Outputs: `plan.md`, `data-model.md`
Stop if: spec has open questions that block safe sequencing — none, spec is ready.

## Цель

Реализовать двухуровневый retrieval: при индексации сохранять parent-документ (полный текст + embedding) как отдельную сущность в VectorStore; при retrieval для каждого найденного чанка загружать parent-документ и возвращать в `RetrievedChunk.ParentContent`. Изменения локализованы в domain (новая optional capability + поле), application (pipeline-helper) и InMemoryStore (reference implementation). Остальные VectorStore реализации не затрагиваются (graceful degradation).

## MVP Slice

Минимальный инкремент: `ParentDocumentStore` interface + `ParentContent` field → InMemoryStore implementation → pipeline integration для Index и Retrieve → AC-001, AC-002, AC-003.

## First Validation Path

`TestParentDocumentRetrieval` в `internal/infrastructure/vectorstore/memory_test.go`: индексировать документ через `Pipeline.Index` → `Pipeline.Retrieve` → проверить `RetrievedChunk.ParentContent == doc.Content`. Тест на graceful degradation: store без parent → `ParentContent == ""`.

## Scope

- `internal/domain/interfaces.go` — новый optional capability `ParentDocumentStore`
- `internal/domain/models.go` — новое поле `ParentContent string` в `RetrievedChunk`
- `internal/application/pipeline.go` — `processDocumentOp`: сохранение parent; новый helper `maybeAttachParentContent` для retrieval
- `internal/application/retrieval.go` / `query.go` — интеграция parent在每个 retrieval-пути (Query, QueryMulti, QueryHyDE, QueryHybrid и Answer-вариации)
- `internal/infrastructure/vectorstore/memory.go` — реализация `ParentDocumentStore`
- `pkg/draftrag/draftrag.go` — `ParentContextEnabled` в `PipelineOptions`, re-export `ParentDocumentStore`
- Bootstrapping surfaces: нет новых директорий/пакетов

## Performance Budget

- SC-001: parent retrieval не увеличивает latency retrieval более чем на 20% для сценария 1 doc / 10 chunks / 1 query. Для InMemoryStore это гарантировано (map lookup O(1)). Для production store — 1 дополнительный запрос на чанк (с группировкой по parentID).

## Implementation Surfaces

| Surface | Изменение | Почему |
|---|---|---|
| `internal/domain/interfaces.go` | Новый optional capability `ParentDocumentStore` | Расширяет VectorStore без ломки существующих реализаций |
| `internal/domain/models.go` | `ParentContent string` в `RetrievedChunk` | Носит parent-контекст до LLM-промпта |
| `internal/application/pipeline.go` | Parent save в `processDocumentOp`; helper `maybeAttachParentContent` | Центральная точка индексации и retrieval |
| `internal/application/query.go` | Вызов `maybeAttachParentContent` в Query, QueryMulti, QueryHyDE | Все retrieval-пути должны получать parent |
| `internal/application/answer.go` | Вызов `maybeAttachParentContent` в Answer* методах | Parent-контекст должен быть в ответе LLM |
| `internal/infrastructure/vectorstore/memory.go` | Реализация `ParentDocumentStore` | Reference implementation для тестов |
| `pkg/draftrag/draftrag.go` | `ParentContextEnabled` option; re-export `ParentDocumentStore` | Публичное API |
| `internal/application/pipeline.go` | Pipeline хранит `parentContextEnabled bool` | Runtime флаг |

## Bootstrapping Surfaces

`none` — существующая структура репозитория достаточна.

## Влияние на архитектуру

Локальное: добавление одного optional capability и одного поля в существующую модель. Никаких изменений в интеграциях, миграциях или rollout-последовательностях. Чанкеры, LLM-провайдеры, embedder'ы не затрагиваются.

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|---|---|---|---|
| AC-001 | Unit-тест: InMemoryStore.GetParentDocument после Index | models, interfaces, store, pipeline | `GetParentDocument` возвращает текст документа |
| AC-002 | Unit-тест: Pipeline.Retrieve после Index | pipeline, query | `RetrievedChunk.ParentContent` непуст |
| AC-003 | Unit-тест: store без parent (QdrantStore mock) → ошибки нет | pipeline (graceful degradation) | `ParentContent == ""`, `err == nil` |
| AC-004 | Unit-тест: `ParentContextEnabled=false` → parent не сохраняется и не возвращается | pipeline, models | parent отсутствует в store и в результате |

## Данные и контракты

- `RetrievedChunk.ParentContent string` — новое поле (data-model.md)
- `ParentDocumentStore` — новый optional interface (data-model.md)
- Compatibility: обратная совместимость для существующих VectorStore — none требует изменений

## Стратегия реализации

### DEC-001 ParentDocumentStore — отдельный optional capability

- **Why**: Вместо хранения parent как `Chunk` с флагом `IsParent`. Parent — концептуально другой тип (нет Position, нет чанк-метаданных, embedding текста целиком). Отдельный интерфейс соблюдает single responsibility и не ломает существующие store.
- **Tradeoff**: Один дополнительный type assertion в pipeline. Для store без parent — graceful degradation.
- **Affects**: `internal/domain/interfaces.go`, `internal/infrastructure/vectorstore/memory.go`, `pkg/draftrag/draftrag.go`
- **Validation**: InMemoryStore реализует interface; store без parent не требует изменений.

### DEC-002 GetParentDocument — прямой lookup по parentID

- **Why**: Вместо `Search` с фильтром по ID. Parent-документ — не векторный поиск, а точное совпадение по ключу. Search по embedding для exact match — концептуально неверно и дорого.
- **Tradeoff**: Необходимость хранить parent-embedding в store (нельзя вычислить на лету). Принимаем: embedding вычисляется один раз при Index.
- **Affects**: `internal/domain/interfaces.go` (сигнатура метода), `internal/application/pipeline.go` (вызов)
- **Validation**: `GetParentDocument(ctx, docID)` возвращает `(*Document, error)`.

### DEC-003 Parent embedding вычисляется из doc.Content до чанкинга

- **Why**: Chunker может модифицировать или не сохранять оригинальный текст документа. Parent embedding должен быть по полному тексту.
- **Tradeoff**: Дополнительный вызов embedder на документ. При `ParentContextEnabled=false` не вызывается.
- **Affects**: `internal/application/pipeline.go` — `produceChunks` или новый helper
- **Validation**: Embedding parent-документа не зависит от выбранного chunker'а.

## Incremental Delivery

### MVP (Первая ценность)

- `ParentDocumentStore` interface + `ParentContent` field
- InMemoryStore реализация
- Pipeline: parent save в `processDocumentOp`
- Pipeline: `maybeAttachParentContent` helper, встроен в все Query/Answer методы
- `ParentContextEnabled` option
- Покрытие: AC-001, AC-002, AC-003, AC-004

### Итеративное расширение

- `none` — фича поставляется одним инкрементом.

## Порядок реализации

1. `internal/domain/interfaces.go` + `internal/domain/models.go` — domain-изменения (без них ничего не компилируется)
2. `internal/infrastructure/vectorstore/memory.go` — InMemoryStore реализация (позволяет писать тесты)
3. `internal/application/pipeline.go` — parent save в processDocumentOp, maybeAttachParentContent helper (core логика)
4. `internal/application/query.go` + `answer.go` — интеграция helper'а во все retrieval-пути
5. `pkg/draftrag/draftrag.go` — PipelineOptions + re-export
6. Тесты на AC-001—AC-004

Шаги 1-2 можно параллелить (разные файлы, нет циклических зависимостей). Шаги 3-4 зависят от 1-2. Шаг 5 зависит от 3-4. Шаг 6 зависит от 5.

## Риски

- **Риск 1: Parent embedding удваивает количество embedder-вызовов при индексации**
  Mitigation: `ParentContextEnabled=false` полностью отключает parent embedding. Для MVP это приемлемо. При необходимости — batch embedding parent-ов как отдельная оптимизация.
- **Риск 2: Parent-контекст может превысить лимит контекста LLM при Answer**
  Mitigation: `maxContextChars` уже ограничивает размер контекста. ParentContent — часть `RetrievedChunk`, формирование промпта уже учитывает `maxContextChars`. Дополнительных мер не требуется.
- **Риск 3: N+1 запросов к VectorStore при retrieval (каждый parent — отдельный GetParentDocument)**
  Mitigation: group by parentID → один GetParentDocument на уникальный parent. Для InMemoryStore и pgvector это O(1). Для remote store — 1 round-trip на parent (обычно 1-3 parent на retrieval).

## Rollout и compatibility

Специальных rollout-действий не требуется. Новое поле `ParentContent` добавляется к `RetrievedChunk` — zero value для кода, который не обновлён. Новый optional capability — реализации VectorStore без него продолжают работать без изменений. `ParentContextEnabled` по умолчанию `true`.

## Проверка

| Шаг | Тест / проверка | AC/DEC |
|---|---|---|
| Domain model | `TestRetrievedChunkParentContent` — zero value по умолчанию | AC-001 |
| InMemoryStore parent | `TestInMemoryStoreParentDocumentStore` — UpsertParent + GetParentDocument + DeleteParent | AC-001, DEC-001, DEC-002 |
| Pipeline Index + Retrieve | `TestPipelineParentDocumentRetrieval` — полный цикл | AC-001, AC-002 |
| Graceful degradation | `TestPipelineParentDocumentGracefulDegradation` — store без parent | AC-003 |
| Opt-out | `TestPipelineParentContextDisabled` — `ParentContextEnabled=false` | AC-004 |
| Parent embedding до чанкинга | `TestParentEmbeddingFromFullContent` — chunker меняет текст, parent embedding от оригинала | DEC-003 |
| Performance budget | `BenchmarkParentRetrievalLatency` — 1 doc, 10 chunks | SC-001 |

## Соответствие конституции

нет конфликтов
