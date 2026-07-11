# Hierarchical Indices — Parent Document Retrieval

## Scope Snapshot

- In scope: добавление поддержки иерархических индексов, при которой родительский документ хранится как отдельная сущность в VectorStore, а retrieval возвращает как релевантные чанки, так и контекст их родительского документа.
- Out of scope: многоуровневая вложенность (3+ уровня), древовидные структуры с произвольной глубиной, cross-document parent-child связи.

## Цель

Пользователь RAG-библиотеки, работающий с длинными документами, получает более полный контекст при retrieval: вместо отдельных фрагментов (чанков) pipeline возвращает также содержимое родительского документа. Это улучшает качество ответа LLM за счёт доступа к исходному контексту, не требуя от пользователя увеличивать `topK` или писать кастомную логику стыковки чанков с документом. Успех фичи измеряется появлением нового поля `ParentContent` в `RetrievedChunk`.

## Основной сценарий

1. Пользователь индексирует документ через `Pipeline.Index` — pipeline вычисляет embedding родительского документа (весь текст) и сохраняет его как отдельную сущность в VectorStore, а также вычисляет embedding каждого чанка и сохраняет их со ссылкой на родителя.
2. При поиске (`Pipeline.Query` / `Pipeline.Retrieve`) pipeline находит релевантные чанки и для каждого чанка извлекает родительский документ из VectorStore.
3. Результат содержит как сами чанки, так и контекст их родительского документа (поле `ParentContent` в каждом `RetrievedChunk`).
4. Если VectorStore не поддерживает parent-сущности — pipeline возвращает оригинальные чанки без parent-контекста (graceful degradation).

## User Stories

- P1 Story: Как пользователь библиотеки, я хочу при retrieval получать содержимое родительского документа для каждого релевантного чанка, чтобы LLM имела больше контекста для генерации ответа.
- P2 Story: Как пользователь, я хочу иметь возможность отключить parent retrieval (использовать оригинальное поведение), если мне не нужен дополнительный контекст.

## MVP Slice

Минимальная реализация: при индексации сохранять parent-документ (полный текст + embedding) в VectorStore; при retrieval для каждого найденного чанка загружать parent-документ и возвращать его в `RetrievedChunk.ParentContent`. AC-001, AC-002, AC-003 должны быть покрыты.

## First Deployable Outcome

После первого implementation pass можно проиндексировать документ через `Index`, выполнить `Retrieve` и увидеть в результате поле `ParentContent` с полным текстом родительского документа. Проверяется через существующий интеграционный тест с InMemoryStore.

## Scope

- Новое поле `ParentContent` в `RetrievedChunk` для передачи текста родительского документа.
- Механизм хранения parent-сущности: новый метод `UpsertParent` или расширение существующего `VectorStore` optional capability `ParentDocumentStore`.
- При индексации: если chunker не настроен (весь документ — один чанк), parent-сущность равна самому документу (без дублирования).
- Graceful degradation: если store не поддерживает parent-операции, parent-контекст не возвращается.
- PipelineOptions: флаг `ParentContextEnabled` (по умолчанию `true`). При `false` pipeline пропускает как сохранение parent-сущности при индексации, так и загрузку parent-контекста при retrieval.

## Контекст

- `VectorStore` оперирует только `Chunk` — нет контракта для хранения произвольных документов как сущностей.
- `ParentID` уже существует в `Chunk` и указывает на ID документа.
- `DocumentStore` / `TransactionalDocumentStore` уже умеют удалять по `ParentID`, но не умеют загружать родительский документ.
- InMemoryStore используется как reference implementation для тестов.
- Chunker не гарантирует, что сохраняет оригинальный текст документа целиком — при chunking'е документ разбивается на фрагменты и исходный текст может быть недоступен.

## Зависимости

- Зависит от `VectorStore` (интерфейс): новый optional capability для хранения parent-сущности.
- Зависит от `Chunker`: если chunker используется, pipeline должен сохранить оригинальный `doc.Content` как parent до вызова chunker'а.
- `none` внешних сервисных зависимостей.

## Требования

- RQ-001 Pipeline ДОЛЖЕН при индексации сохранять parent-документ (ID + полный текст + embedding) в VectorStore, если store поддерживает parent-операции.
- RQ-002 Pipeline ДОЛЖЕН при retrieval загружать parent-документ для каждого найденного чанка и возвращать его содержимое в `RetrievedChunk.ParentContent`.
- RQ-003 Система ДОЛЖНА корректно работать без parent-контекста, если store не поддерживает parent-операции (graceful degradation, без ошибки).
- RQ-004 Пользователь ДОЛЖЕН иметь возможность отключить parent context через `PipelineOptions.ParentContextEnabled`. При `false` pipeline не сохраняет parent-сущность при индексации и не загружает parent-контекст при retrieval.

## Вне scope

- Произвольная вложенность (3+ уровня): только parent → chunks.
- Массовая загрузка parent-документов отдельно от индексации (batch parent upsert).
- Parent-контент в streaming-пути (`Stream` / `StreamSources`).
- Изменение API чанкеров: chunker продолжает возвращать `[]Chunk`, pipeline берёт на себя сохранение parent.
- Ручное управление версионированием parent-документа.

## Критерии приемки

### AC-001 Parent сохраняется при индексации

- Почему это важно: без сохранения parent-документа невозможна двухуровневая выборка.
- **Given** пустое VectorStore, поддерживающее parent-операции
- **When** pipeline индексирует документ с ID `doc-1` и текстом `"Hello world. This is a test."` через `Index`
- **Then** в VectorStore сохранены: (1) parent-сущность с ID `doc-1` и текстом `"Hello world. This is a test."`, (2) один или несколько чанков с `ParentID = doc-1`
- Evidence: прямой вызов store.GetParentDocument(ctx, "doc-1") возвращает parent-документ с корректным текстом.

### AC-002 Parent-контекст возвращается при retrieval

- Почему это важно: пользователь получает полный контекст, а не фрагменты.
- **Given** VectorStore, поддерживающее parent-операции, с проиндексированным документом `doc-1` и его чанками
- **When** вызывается `Retrieve` или `Query` с текстом запроса, релевантным одному из чанков
- **Then** каждый `RetrievedChunk` содержит непустое `ParentContent` с полным текстом `doc-1`
- Evidence: принт результатов retrieval показывает `ParentContent` для каждого чанка, содержимое соответствует `doc.Content`.

### AC-003 Graceful degradation для store без parent

- Почему это важно: обратная совместимость с существующими store-реализациями.
- **Given** VectorStore без поддержки parent-операций (например, QdrantStore по умолчанию)
- **When** вызывается `Retrieve` после индексации
- **Then** `RetrievedChunk.ParentContent` пуст (zero value), ошибка не возвращается
- Evidence: вызов `Retrieve` возвращает успешный результат, `ParentContent == ""` во всех чанках, `errors.Is(err, nil) == true`.

### AC-004 Parent-контекст отключается через PipelineOptions

- Почему это важно: пользователь должен иметь контроль над поведением.
- **Given** Pipeline, созданный с `PipelineOptions.ParentContextEnabled = false`, и VectorStore с поддержкой parent-операций
- **When** вызывается `Retrieve` после индексации
- **Then** `RetrievedChunk.ParentContent` пуст, даже если store поддерживает parent
- Evidence: при `ParentContextEnabled = false` результаты идентичны поведению AC-003.

## Допущения

- Parent-документ сохраняется один раз при индексации и не обновляется при partial update чанков.
- `doc.ID` уникален и используется как идентификатор parent-сущности.
- Для документа без chunker'а (весь документ — один чанк) parent-сущность сохраняется, но ParentContent при retrieval будет равен Content этого единственного чанка (избыточно, но консистентно).
- Существующие VectorStore-реализации (Qdrant, ChromaDB, Weaviate, Milvus) не реализуют parent-операции в MVP — только InMemoryStore.

## Критерии успеха

- SC-001 Parent retrieval не увеличивает latency retrieval более чем на 20% для типового сценария (1 документ, 10 чанков, 1 запрос).

## Краевые случаи

- Пустой документ (Content == "") — parent не сохраняется, ошибка валидации.
- Документ с одним чанком — parent-сущность сохраняется отдельно, при retrieval ParentContent == Content чанка.
- Удаление документа через `DeleteDocument` — parent-сущность и все чанки удаляются.
- Обновление документа через `UpdateDocument` — parent-сущность перезаписывается новым текстом и embedding'ом.

## Открытые вопросы

- Стоит ли вводить отдельный тип `ParentDocument` в domain или достаточно хранить parent через `Chunk` с флагом `IsParent`?
- Какой метод использовать для загрузки parent: `GetParentDocument(ctx, parentID string) (*Document, error)` в новом `ParentDocumentStore`, или переиспользовать `Search` с фильтром?
- Нужна ли отдельная колонка/поле для parent-документа в SQL-схеме pgvector?
