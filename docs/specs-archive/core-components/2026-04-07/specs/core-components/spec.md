# Core компоненты пакета draftRAG

## Scope Snapshot

- In scope: определение domain-интерфейсов (VectorStore, LLMProvider, Embedder), domain-моделей (Document, Chunk, Query, RetrievalResult) и публичного API пакета для композиции компонентов
- Out of scope: конкретные реализации инфраструктурных провайдеров (pgvector, Qdrant, OpenAI и т.д.), HTTP-сервер, CLI

## Цель

Разработчики, использующие draftRAG, получают набор абстракций для работы с RAG-системами без привязки к конкретным провайдерам. Фича предоставляет ядро пакета: интерфейсы для векторных хранилищ, LLM-провайдеров и эмбеддеров, а также базовые модели данных. Успех измеряется возможностью написать unit-тесты с мок-реализациями и собрать пакет без внешних зависимостей.

## Основной сценарий

1. Разработчик импортирует `pkg/draftrag` в свой Go-проект
2. Создаёт конкретные реализации интерфейсов (например, `NewQdrantStore()`, `NewOpenAIProvider()`) или использует фабрики пакета
3. Компонует RAG-пайплайн через `draftrag.NewPipeline(store, llm, embedder)`
4. Индексирует документы: `pipeline.Index(ctx, docs)`
5. Выполняет запрос: `result, err := pipeline.Query(ctx, "вопользователя")`

## Scope

- Domain-интерфейсы: `VectorStore`, `LLMProvider`, `Embedder`, `Chunker`
- Domain-модели: `Document`, `Chunk`, `Query`, `RetrievalResult`, `Embedding`
- Публичный API пакета в `pkg/draftrag/`: фабрики, конструкторы, Pipeline
- Application use-cases: `IndexDocuments`, `Retrieve`, `Generate`
- In-memory реализации для тестирования

## Контекст

- Go 1.21+, стандартная библиотека + минимальные внешние зависимости
- Clean Architecture: domain-слой не импортирует внешние пакеты
- Все публичные функции принимают `context.Context` первым параметром
- Пакет — библиотека, не приложение: нет своего HTTP-сервера или CLI
- Интерфейсы должны допускать замену реализаций без изменения клиентского кода

## Требования

- RQ-001 Интерфейс `VectorStore` ДОЛЖЕН поддерживать операции: Upsert, Delete, Search по embedding-вектору с метрикой similarity
- RQ-002 Интерфейс `LLMProvider` ДОЛЖЕН поддерживать синхронную генерацию текста с передачей system/user messages
- RQ-003 Интерфейс `Embedder` ДОЛЖЕН преобразовывать текст в вектор фиксированной размерности
- RQ-004 Модель `Document` ДОЛЖНА содержать ID, content, metadata (map[string]string), timestamps
- RQ-005 Модель `Chunk` ДОЛЖНА содержать ID, content, parent Document ID, embedding (опционально nil до вычисления)
- RQ-006 Все публичные методы ДОЛЖНЫ принимать `context.Context` первым параметром
- RQ-007 Пакет ДОЛЖЕН собираться без ошибок через `go build ./...`
- RQ-008 In-memory реализация `VectorStore` ДОЛЖНА существовать для тестирования

## Вне scope

- Конкретные реализации провайдеров: PostgreSQL+pgvector, Qdrant, ChromaDB, OpenAI, Anthropic, Ollama
- Стриминг ответов от LLM
- Асинхронная индексация через worker queue
- HTTP handlers или middleware
- CLI утилиты
- Миграции схем баз данных
- Retry logic и circuit breakers

## Критерии приемки

### AC-001 Интерфейсы определены и документированы

- Почему это важно: без чётких интерфейсов невозможна замена провайдеров и тестируемость
- **Given** разработчик открывает godoc для `pkg/draftrag`
- **When** он смотрит exported типы
- **Then** он видит интерфейсы `VectorStore`, `LLMProvider`, `Embedder` с godoc-комментариями на русском языке
- Evidence: `go doc pkg/draftrag.VectorStore` выводит описание методов на русском

### AC-002 Domain-модели позволяют описать типичный RAG-сценарий

- Почему это важно: модели — основа потока данных от документа до ответа
- **Given** разработчик создаёт `Document` с content и metadata
- **When** он передаёт его в `pipeline.Index(ctx, []Document{doc})`
- **Then** документ индексируется без ошибок, metadata сохраняется
- Evidence: unit-тест с in-memory store показывает Upsert и Search по созданному документу

### AC-003 Контекст поддерживается во всех операциях

- Почему это важно: отмена операций и таймауты критичны для production-систем
- **Given** pipeline с in-memory реализациями
- **When** вызывается `pipeline.Query(cancelledCtx, "вопрос")` с отменённым контекстом
- **Then** метод возвращает ошибку `context.Canceled` немедленно
- Evidence: unit-тест с `context.WithCancel()` и немедленным `cancel()`

### AC-004 In-memory VectorStore проходит базовые тесты

- Почему это важно: in-memory хранилище нужно для тестирования без внешних зависимостей
- **Given** in-memory реализация `VectorStore`
- **When** выполняются Upsert документа и Search по похожему запросу
- **Then** Search возвращает документ в результатах с корректным score
- Evidence: unit-тест `TestInMemoryStore_BasicSearch` проходит успешно

### AC-005 Публичный API позволяет скомпоновать pipeline

- Почему это важно: разработчики должны иметь простой способ собрать все компоненты вместе
- **Given** реализации `VectorStore`, `LLMProvider`, `Embedder`
- **When** разработчик вызывает `draftrag.NewPipeline(store, llm, embedder)`
- **Then** возвращается объект с методами `Index(ctx, docs)` и `Query(ctx, question)`
- Evidence: пример кода в godoc или integration-тест демонстрирует полный цикл

## Допущения

- Разработчики знакомы с Go и паттерном dependency injection
- Similarity-метрика по умолчанию — cosine similarity, но интерфейс позволяет передавать параметр
- Embedding вычисляется отдельно от индексации (клиент сам вызывает Embedder перед Upsert)
- Metadata в Document — плоская структура map[string]string, вложенность не требуется
- In-memory store хранит данные только в памяти процесса, без персистентности

## Критерии успеха

- SC-001 `go build ./...` завершается менее чем за 5 секунд на среднем ноутбуке
- SC-002 Тестовое покрытие domain-слоя ≥80% по `go test -cover`
- SC-003 godoc для всех exported типов содержит описание на русском языке

## Краевые случаи

- Пустой документ (content == "") — ДОЛЖЕН возвращать ошибку валидации
- Поиск по пустому хранилищу — ДОЛЖЕН возвращать пустой результат без ошибки
- Query с пустым текстом — ДОЛЖЕН возвращать ошибку валидации
- Nil context — ДОЛЖЕН вызывать panic (стандарт Go)

## Открытые вопросы

- none
