# Hierarchical Indices — Parent Document Retrieval: Задачи

## Phase Contract

Inputs: `docs/specs/hierarchical-indices/plan.md`, `docs/specs/hierarchical-indices/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием всех AC
Stop if: AC-* нельзя привязать к задачам — все привязаны.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/domain/models.go` | T1.2 |
| `internal/infrastructure/vectorstore/memory.go` | T2.1 |
| `internal/application/pipeline.go` | T3.1, T3.2 |
| `internal/application/query.go` | T3.3 |
| `internal/application/answer.go` | T3.3 |
| `pkg/draftrag/draftrag.go` | T4.1 |
| `internal/domain/models.go` (новые тесты) | T5.1 |
| `internal/infrastructure/vectorstore/memory_test.go` | T5.2 |
| `internal/application/pipeline_test.go` | T5.3 |

## Implementation Context

- **Цель MVP:** parent-документ сохраняется при Index, возвращается в `RetrievedChunk.ParentContent` при Query/Retrieve/Answer
- **Инварианты/семантика:**
  - `ParentDocumentStore` — optional capability (не ломает существующие реализации)
  - `ParentContent` — zero value (`""`) если store не поддерживает или отключено флагом
  - Parent embedding вычисляется из `doc.Content` до вызова Chunker (DEC-003)
  - Group by parentID: один `GetParentDocument` на уникальный parent при retrieval
- **Ошибки/коды:**
  - Отсутствие capability → graceful degradation (без ошибки)
  - nil store → паника (существующий контракт не меняется)
- **Контракты/протокол:**
  - `ParentDocumentStore`:
    - `UpsertParent(ctx, doc Document, embedding []float64) error`
    - `GetParentDocument(ctx, parentID string) (*Document, error)` — возвращает `(*Document, nil)` или `(nil, nil)` если не найден
    - `DeleteParent(ctx, parentID string) error` — идемпотентно
  - `PipelineOptions.ParentContextEnabled bool` — default `true`
- **Границы scope:**
  - Не меняем chunker, LLM provider, embedder
  - Не добавляем parent-поддержку в production store (Qdrant, ChromaDB и т.д.)
  - Не трогаем streaming-пути (`Stream`, `StreamSources`)
- **Proof signals:** `RetrievedChunk.ParentContent` содержит полный текст документа; `ParentContextEnabled=false` → пустая строка; store без capability → пустая строка
- **References:** DEC-001, DEC-002, DEC-003, DM-001, DM-002, RQ-001—RQ-004

## Фаза 1: Domain + Store Foundation

Цель: подготовить domain-модель и reference implementation для родительских документов.

- [x] T1.1 Добавить `ParentDocumentStore` optional capability в `internal/domain/interfaces.go`
  Touches: `internal/domain/interfaces.go`
  Контракты из DM-002: `UpsertParent`, `GetParentDocument`, `DeleteParent`
  Подтверждение: `go vet ./internal/domain/...` проходит; type assertion `s.(domain.ParentDocumentStore)` компилируется

- [x] T1.2 Добавить поле `ParentContent string` в `RetrievedChunk` в `internal/domain/models.go`
  Touches: `internal/domain/models.go`
  Подтверждение: `RetrievedChunk{}` компилируется; `chunk.ParentContent == ""` для zero value

- [x] T2.1 Реализовать `ParentDocumentStore` в `InMemoryStore` (`internal/infrastructure/vectorstore/memory.go`)
  Touches: `internal/infrastructure/vectorstore/memory.go`
  Хранилище: `parents map[string]parentEntry` (parentEntry: `{doc domain.Document, embedding []float64}`)
  Подтверждение: `InMemoryStore` можно привести к `domain.ParentDocumentStore`; `go vet ./internal/infrastructure/vectorstore/...` проходит

## Фаза 2: Pipeline Core

Цель: интегрировать parent-документ в pipeline — сохранение при индексации и загрузка при retrieval.

- [x] T3.1 Реализовать сохранение parent-документа в `Pipeline.processDocumentOp`/`produceChunks`
  Touches: `internal/application/pipeline.go`
  Семантика:
  - Если `p.parentContextEnabled && p.store` реализует `ParentDocumentStore` → вычислить embedding `doc.Content` и вызвать `UpsertParent`
  - Parent embedding вычисляется **до** chunker'а (DEC-003)
  - Если chunker отсутствует (весь doc — один chunk) → parent embedding = embedding единственного chunk (избегаем дублирования embedder-вызова)
  Подтверждение: после `Index(doc)` parent существует в InMemoryStore

- [x] T3.2 Реализовать helper `maybeAttachParentContent` в `Pipeline`
  Touches: `internal/application/pipeline.go`
  Семантика:
  - Type-assert `p.store` на `domain.ParentDocumentStore`
  - Если не реализует → return (graceful degradation)
  - Если `!p.parentContextEnabled` → return
  - Group by unique `ParentID` → batched `GetParentDocument`
  - Присвоить `ParentContent` каждому `RetrievedChunk`
  Подтверждение: helper возвращает chunks с заполненным `ParentContent`

- [x] T3.3 Интегрировать `maybeAttachParentContent` во все retrieval-пути
  Touches: `internal/application/query.go`, `internal/application/answer.go`
  Точки интеграции:
  - `Query` — после Search + Dedup + Rerank
  - `QueryWithParentIDs` — после SearchWithFilter
  - `QueryWithMetadataFilter` — после SearchWithMetadataFilter
  - `QueryHyDE` — после Search + Dedup + Rerank
  - `QueryMulti` — после RRF merge + Dedup
  - `QueryWithQueries` — после RRF merge + Dedup
  - `QueryHybrid` — после SearchHybrid + Dedup + Rerank
  - Все Answer-вариации — через единый `generateAnswer` или встроенный вызов (т.к. Answer вызывает Query* методы, parent уже прикреплён к `RetrievalResult`)
  - `maybeRerankBatch` — parent сохраняется до rerank
  Подтверждение: `Retrieve` / `Query` / `Answer` возвращают `ParentContent`

## Фаза 3: Public API

Цель: открыть новую возможность пользователям библиотеки через `PipelineOptions` и re-export.

- [x] T4.1 Добавить `ParentContextEnabled` в `PipelineOptions` и re-export `ParentDocumentStore`
  Touches: `pkg/draftrag/draftrag.go`
  Изменения:
  - Поле `ParentContextEnabled bool` в `PipelineOptions` (default `false` — zero value; plan: default `true`, значит явно устанавливаем `true` в `NewPipelineWithConfig`)
  - Type alias `ParentDocumentStore = domain.ParentDocumentStore`
  - Прокинуть флаг в `Pipeline.parentContextEnabled` в `NewPipelineWithConfig`
  Подтверждение: `go vet ./pkg/...` проходит; пользователь может задать `ParentContextEnabled: false`

## Фаза 4: Верификация

Цель: доказать, что все AC выполняются, и застраховать от регрессий.

- [x] T5.1 Добавить тест на zero value `RetrievedChunk.ParentContent`
  Touches: `internal/domain/models_test.go`
  Проверка: `RetrievedChunk{}.ParentContent == ""`
  Покрывает: AC-001 (косвенно — zero value invariant)

- [x] T5.2 Добавить unit-тесты `ParentDocumentStore` для InMemoryStore
  Touches: `internal/infrastructure/vectorstore/memory_test.go`
  Тесты:
  - `TestInMemoryStoreParentDocumentStore`: UpsertParent → GetParentDocument → DeleteParent → GetParentDocument (nil)
  - `TestInMemoryStoreParentDocumentStoreNil` — документ не найден → (nil, nil)
  Покрывает: AC-001, DEC-001, DEC-002

- [x] T5.3 Добавить интеграционные тесты Pipeline с parent-документом
  Touches: `internal/application/pipeline_test.go`
  Тесты:
  - `TestPipelineParentDocumentRetrieval`: Index → Retrieve → `ParentContent == doc.Content` (AC-001, AC-002)
  - `TestPipelineParentDocumentGracefulDegradation`: store без `ParentDocumentStore` → `ParentContent == ""`, err == nil (AC-003)
  - `TestPipelineParentContextDisabled`: `ParentContextEnabled=false` → parent не сохранён, `ParentContent == ""` (AC-004)
  - `TestParentEmbeddingFromFullContent`: chunker с изменением текста → parent embedding от оригинального `doc.Content` (DEC-003)
  Покрывает: AC-001, AC-002, AC-003, AC-004

## Покрытие критериев приемки

- AC-001 -> T1.2, T3.1, T5.2, T5.3
- AC-002 -> T3.2, T3.3, T5.3
- AC-003 -> T3.2, T5.3
- AC-004 -> T3.1, T4.1, T5.3

## Заметки

- T1.1 и T1.2 можно параллелить (разные файлы одного пакета)
- T2.1 зависит от T1.1 (интерфейс должен существовать)
- T3.1—T3.3 зависят от T1.1—T2.1
- T4.1 зависит от T3.3 (Pipeline с `parentContextEnabled`)
- T5.1—T5.3 зависят от T3.3 и T4.1
- AC-003 не требует отдельной задачи для graceful degradation в query/answer — это поведение встроено в `maybeAttachParentContent` (T3.2)
