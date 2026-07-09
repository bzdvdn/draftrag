# Metadata filtering

## Scope Snapshot

- In scope: произвольная фильтрация по полям метаданных документа при семантическом поиске; расширение публичного API pipeline-методами с поддержкой фильтра.
- Out of scope: гибридный поиск BM25+semantic, кэширование эмбеддингов, реализация Qdrant или ChromaDB.

## Цель

Разработчики, использующие draftRAG в production, получают возможность ограничивать семантический поиск подмножеством документов по произвольным метаданным (`author`, `date`, `category`, и т.д.). Сейчас единственный фильтр — `ParentIDFilter`, которого недостаточно для реальных сценариев. Фича расширяет интерфейс фильтрации и обеспечивает рабочую реализацию в pgvector-бэкенде, а также методы публичного API.

## Основной сценарий

1. Разработчик индексирует документы с произвольными метаданными (`doc.Metadata["category"] = "legal"`).
2. При запросе он передаёт `MetadataFilter{Fields: map[string]string{"category": "legal"}}` в метод `QueryWithMetadataFilter` или `AnswerWithMetadataFilter`.
3. Pipeline применяет фильтр на уровне векторного хранилища — pgvector возвращает только чанки, чьи метаданные соответствуют всем переданным полям.
4. Если ни один чанк не прошёл фильтр, метод возвращает пустой `RetrievalResult` без ошибки.

## Scope

- Новый domain-тип `MetadataFilter` с полем `Fields map[string]string` (точное совпадение по всем ключам).
- Расширение интерфейса `VectorStoreWithFilters`: новый метод `SearchWithMetadataFilter`.
- Реализация `SearchWithMetadataFilter` в pgvector-бэкенде с SQL WHERE по JSONB-колонке метаданных.
- Публичные методы `QueryWithMetadataFilter` и `AnswerWithMetadataFilter` в `pkg/draftrag/`.
- Unit-тесты для нового domain-типа и integration-тесты для pgvector-реализации.

## Контекст

- `Document.Metadata` уже существует как `map[string]string` в domain-модели — это основа для хранения фильтруемых полей.
- `Query.Filter map[string]string` уже есть в domain, но не используется ни одним бэкендом и публичным методом.
- `VectorStoreWithFilters` — уже существующий опциональный интерфейс, расширяющий `VectorStore` без ломки контракта.
- Pgvector-бэкенд хранит метаданные в JSONB-колонке; фильтрация реализуется через `@>` оператор JSONB.
- In-memory бэкенд используется только для тестов — реализация `SearchWithMetadataFilter` нужна для консистентности интерфейса, но не является production-требованием.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять тип `MetadataFilter` с полем `Fields map[string]string`, выражающий точное совпадение по всем переданным ключам.
- RQ-002 Интерфейс `VectorStoreWithFilters` ДОЛЖЕН быть расширен методом `SearchWithMetadataFilter(ctx, embedding, topK, filter MetadataFilter)`, не ломая существующий `SearchWithFilter(ParentIDFilter)`.
- RQ-003 Pgvector-реализация ДОЛЖНА транслировать `MetadataFilter.Fields` в SQL-условие на JSONB-колонке метаданных (оператор `@>`).
- RQ-004 In-memory реализация ДОЛЖНА реализовать `SearchWithMetadataFilter` с фильтрацией в памяти для поддержки тестового окружения.
- RQ-005 Публичный API ДОЛЖЕН предоставлять `QueryWithMetadataFilter(ctx, question, topK, filter MetadataFilter) (RetrievalResult, error)`.
- RQ-006 Публичный API ДОЛЖЕН предоставлять `AnswerWithMetadataFilter(ctx, question, topK, filter MetadataFilter) (Answer, error)`.
- RQ-007 Если `MetadataFilter.Fields` пуст или nil, метод ДОЛЖЕН вести себя идентично соответствующему методу без фильтра.
- RQ-008 Если бэкенд не реализует `VectorStoreWithFilters`, `QueryWithMetadataFilter` и `AnswerWithMetadataFilter` ДОЛЖНЫ возвращать `ErrFilterNotSupported`; молчаливая деградация запрещена.
- RQ-009 Каждый публичный тип и функция ДОЛЖНЫ иметь godoc-комментарий на русском языке.

## Вне scope

- Поддержка операторов сравнения (`>=`, `<=`, `!=`, `LIKE`) — только точное совпадение в этой фиче.
- Комбинация `ParentIDFilter` и `MetadataFilter` в одном вызове — отдельный запрос не принимался.
- Реализация в Qdrant, ChromaDB или других бэкендах.
- Изменения в схеме таблиц pgvector (метаданные уже хранятся в JSONB).
- Streaming-версии методов с фильтром.

## Критерии приемки

### AC-001 MetadataFilter точно фильтрует результаты поиска в pgvector

- Почему это важно: разработчик должен получать только релевантные документы без постобработки на стороне приложения.
- **Given** в pgvector проиндексированы документы с разными значениями `Metadata["category"]` («legal» и «finance»)
- **When** вызывается `SearchWithMetadataFilter` с `MetadataFilter{Fields: {"category": "legal"}}`
- **Then** возвращаются только чанки документов с `category=legal`; чанки с `category=finance` отсутствуют в результате
- Evidence: integration-тест сравнивает ID возвращённых чанков с ожидаемым множеством.

### AC-002 Пустой фильтр не меняет поведение поиска

- Почему это важно: обратная совместимость — существующие вызовы через новые методы не должны ломаться.
- **Given** pgvector содержит несколько проиндексированных документов
- **When** вызывается `SearchWithMetadataFilter` с `MetadataFilter{Fields: nil}` или `MetadataFilter{}`
- **Then** результат идентичен вызову базового `Search` с теми же параметрами
- Evidence: тест сравнивает возвращённые наборы ID при пустом фильтре и без фильтра.

### AC-003 Публичный API передаёт фильтр сквозь pipeline

- Почему это важно: пользователь пакета работает через `pkg/draftrag/`, а не напрямую с domain-интерфейсами.
- **Given** pipeline сконфигурирован с pgvector-бэкендом и ненулевыми документами с метаданными
- **When** вызывается `QueryWithMetadataFilter` или `AnswerWithMetadataFilter` с непустым фильтром
- **Then** в `RetrievalResult` / `Answer` присутствуют только чанки, соответствующие фильтру
- Evidence: integration-тест или unit-тест с моком проверяет, что переданный фильтр дошёл до `SearchWithMetadataFilter`.

### AC-004 Фильтр по несуществующему значению возвращает пустой результат без ошибки

- Почему это важно: корректное пустое состояние — необходимое поведение для production-сценариев.
- **Given** в хранилище нет документов с `Metadata["category"]="nonexistent"`
- **When** вызывается `SearchWithMetadataFilter` с `MetadataFilter{Fields: {"category": "nonexistent"}}`
- **Then** возвращается пустой `RetrievalResult` и `error == nil`
- Evidence: assertion `len(result.Chunks) == 0 && err == nil` в тесте.

### AC-005 In-memory реализация удовлетворяет интерфейсу

- Почему это важно: тестовое окружение должно быть способно компилироваться и работать с новым интерфейсом.
- **Given** in-memory store используется в unit-тестах как `VectorStoreWithFilters`
- **When** компилируется и запускается `go test ./...`
- **Then** компиляция проходит без ошибок; in-memory `SearchWithMetadataFilter` фильтрует по переданным полям
- Evidence: `go build ./...` и `go test ./...` завершаются с exit code 0.

## Допущения

- Метаданные документов хранятся в JSONB-колонке в pgvector-таблице (существующая схема не меняется).
- Фильтрация выполняется только по точному совпадению строк — без числовых сравнений или regex.
- `MetadataFilter` применяется как AND-условие по всем переданным полям (все должны совпасть).
- Существующий `Query.Filter map[string]string` в domain будет заменён или вытеснен явным `MetadataFilter`-типом без изменения wire-формата хранения данных.
- Pgvector-бэкенд уже имеет JSONB-индекс или запросы без него достаточно быстры для MVP-объёмов.

## Краевые случаи

- Фильтр с несколькими ключами: все условия применяются как AND через JSONB-оператор `@>`.
- Документ без метаданных (пустой map): не проходит фильтр с непустыми полями.
- `MetadataFilter.Fields` содержит ключ с пустым значением `""`: трактуется как точное совпадение с пустой строкой, а не как wildcard.
- Бэкенд не реализует `VectorStoreWithFilters`: публичный API возвращает `ErrFilterNotSupported`; молчаливая деградация запрещена, так как может привести к утечке данных между тенантами или категориями.

## Открытые вопросы

- none
