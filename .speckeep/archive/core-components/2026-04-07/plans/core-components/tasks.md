# Core компоненты пакета draftRAG — Задачи

## Phase Contract

Inputs: `.draftspec/plans/core-components/plan.md`, `.draftspec/plans/core-components/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/interfaces.go | T1.1 |
| internal/domain/models.go | T1.2 |
| internal/infrastructure/vectorstore/memory.go | T2.1 |
| internal/application/pipeline.go | T2.2 |
| pkg/draftrag/draftrag.go | T3.1 |
| pkg/draftrag/errors.go | T3.2 |
| internal/domain/*_test.go | T4.1 |
| internal/application/*_test.go | T4.2 |
| internal/infrastructure/vectorstore/*_test.go | T4.3 |

## Фаза 1: Domain-слой

Цель: определить core-интерфейсы и domain-модели, от которых зависят все остальные слои

- [x] T1.1 Создать `internal/domain/interfaces.go` — определить интерфейсы VectorStore (Upsert, Delete, Search), LLMProvider (Generate), Embedder (Embed), Chunker (Chunk) с godoc-комментариями на русском. Touches: internal/domain/interfaces.go — DEC-001, DEC-002
- [x] T1.2 Создать `internal/domain/models.go` — определить структуры Document (ID, Content, Metadata, CreatedAt, UpdatedAt), Chunk (ID, Content, ParentID, Embedding, Position), Query (Text, TopK, Filter), RetrievalResult (Chunks, QueryText, TotalFound), RetrievedChunk (Chunk, Score), Embedding (Vector, Dimension, Model). Touches: internal/domain/models.go — AC-002, DM-001..DM-006

## Фаза 2: Infrastructure и Application-слои

Цель: реализовать in-memory VectorStore для тестирования и use-case Pipeline для композиции компонентов

- [x] T2.1 Создать `internal/infrastructure/vectorstore/memory.go` — реализовать in-memory VectorStore с cosine similarity; методы Upsert, Delete, Search возвращают RetrievalResult с RetrievedChunk.Score в диапазоне [-1, 1]. Touches: internal/infrastructure/vectorstore/memory.go — AC-004, DEC-003
- [x] T2.2 Создать `internal/application/pipeline.go` — реализовать Pipeline с методами Index(ctx, docs) и Query(ctx, question); все методы принимают context.Context первым параметром; Query возвращает ошибку при пустом тексте. Touches: internal/application/pipeline.go — AC-003, AC-005

## Фаза 3: Публичный API

Цель: экспортировать функциональность через pkg/draftrag для клиентов библиотеки

- [x] T3.1 Создать `pkg/draftrag/draftrag.go` — экспортировать функцию NewPipeline(store, llm, embedder) возвращающую Pipeline; экспортировать интерфейсы VectorStore, LLMProvider, Embedder через type aliases. Touches: pkg/draftrag/draftrag.go — AC-005
- [x] T3.2 Создать `pkg/draftrag/errors.go` — определить ошибки валидации ErrEmptyDocument, ErrEmptyQuery, ErrInvalidTopK; валидация в Pipeline.Index и Pipeline.Query возвращает эти ошибки при невалидных входных данных. Touches: pkg/draftrag/errors.go — AC-002, DEC-004

## Фаза 4: Тестирование

Цель: подтвердить корректность реализации через unit и integration тесты

- [x] T4.1 Создать `internal/domain/models_test.go` — тесты валидации: Document с пустым Content возвращает ошибку; Chunk с nil ParentID возвращает ошибку. Touches: internal/domain/models_test.go — AC-002
- [x] T4.2 Создать `internal/application/pipeline_test.go` — тест TestPipeline_ContextCancellation с отменённым контекстом возвращает context.Canceled; тест TestPipeline_FullCycle демонстрирует Index + Query цикл. Touches: internal/application/pipeline_test.go — AC-003, AC-005
- [x] T4.3 Создать `internal/infrastructure/vectorstore/memory_test.go` — тест TestInMemoryStore_BasicSearch: Upsert документа, Search по похожему тексту, проверка score > 0 и len(Chunks) > 0. Touches: internal/infrastructure/vectorstore/memory_test.go — AC-004

## Покрытие критериев приемки

- AC-001 -> T1.1 (интерфейсы с godoc на русском)
- AC-002 -> T1.2 (модели), T4.1 (валидация), T4.2 (полный цикл)
- AC-003 -> T2.2 (context в методах), T4.2 (тест отмены контекста)
- AC-004 -> T2.1 (in-memory store), T4.3 (тест BasicSearch)
- AC-005 -> T3.1 (NewPipeline), T4.2 (тест FullCycle)

## Заметки

- Порядок задач соответствует слоистой архитектуре: domain → infrastructure → application → public API → тесты
- Все задачи имеют явные Touches с конкретными файлами для batch-чтения implement-агентом
- Каждая задача ссылается на 1-2 стабильных ID (AC-*, DEC-*, DM-*)
- Фаза тестирования вынесена отдельно, но тесты пишутся параллельно с реализацией
