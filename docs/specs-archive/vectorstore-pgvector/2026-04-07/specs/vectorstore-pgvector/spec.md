# VectorStore pgvector (PostgreSQL) для draftRAG

## Scope Snapshot

- In scope: публичная фабрика и внутренняя реализация `VectorStore` на PostgreSQL+pgvector для production-использования (Upsert/Delete/Search по embedding).
- Out of scope: любые LLM/Embedder провайдеры, HTTP/CLI, гибридный поиск (BM25), сложная фильтрация по metadata и управление миграциями вне предоставленного helper’а.

## Цель

Разработчик может подключить PostgreSQL с расширением pgvector и использовать draftRAG с реальным persisted векторным хранилищем через тот же интерфейс `VectorStore`, что и в тестовой in-memory реализации. Успех измеряется тем, что `Pipeline` (из `pkg/draftrag`) способен индексировать и искать документы, а `go test ./...` проходит без необходимости поднимать внешние сервисы по умолчанию.

## Основной сценарий

1. Разработчик поднимает PostgreSQL и включает расширение pgvector (один раз).
2. Создаёт подключение `*sql.DB` (или эквивалентный пул через `database/sql`).
3. Вызывает `draftrag.SetupPGVector(ctx, db, opts)` (или аналогичный helper) для создания таблицы/индекса, если их нет.
4. Создаёт `store := draftrag.NewPGVectorStore(db, opts)` и собирает `pipeline := draftrag.NewPipeline(store, llm, embedder)`.
5. Вызывает `pipeline.Index(ctx, docs)` и затем `pipeline.QueryTopK(ctx, question, topK)`; получает `RetrievalResult` с релевантными чанками и score.

## Scope

- Реализация pgvector-backed `VectorStore` с методами Upsert/Delete/Search (соответствие `internal/domain.VectorStore`).
- Публичный API в `pkg/draftrag` для создания store и (опционально) создания схемы (таблица + индекс).
- Модель хранения чанков: ID, ParentID, Content, Position, Embedding.
- Базовая метрика: cosine distance в БД + преобразование в similarity score в диапазоне [-1, 1] на выходе.
- Интеграционный тест (опционально) под реальную БД, с пропуском по умолчанию без DSN.

## Контекст

- Проект — библиотека, без собственного сервера/CLI; подключение к БД и lifecycle пула остаются на пользователе.
- Clean Architecture: интерфейс `VectorStore` остаётся в domain; pgvector — инфраструктурная реализация, доступная пользователю через публичную фабрику (без импорта `internal/...`).
- Все публичные функции принимают `context.Context` первым параметром; `nil` context — panic (стандарт Go).

## Требования

- RQ-001 ДОЛЖНА существовать публичная функция/фабрика в `pkg/draftrag`, позволяющая создать pgvector-backed `VectorStore`, не импортируя `internal/...`.
- RQ-002 Реализация ДОЛЖНА поддерживать `Upsert(ctx, chunk)`, `Delete(ctx, id)` и `Search(ctx, embedding, topK)` в соответствии с интерфейсом `VectorStore`.
- RQ-003 `Search` ДОЛЖЕН возвращать результаты, отсортированные по убыванию `Score`, а `Score` ДОЛЖЕН быть в диапазоне [-1, 1] (cosine similarity).
- RQ-004 Реализация ДОЛЖНА поддерживать создание схемы хранения через helper (например, `SetupPGVector`): создание расширения/таблицы/индекса, если отсутствуют.
- RQ-005 По умолчанию `go test ./...` ДОЛЖЕН проходить без поднятой PostgreSQL: интеграционные тесты ДОЛЖНЫ быть пропускаемыми (skip), если отсутствует DSN.
- RQ-006 Все операции ДОЛЖНЫ уважать `ctx.Done()` (передавать `ctx` в SQL-запросы); отмена контекста возвращает `context.Canceled`/`context.DeadlineExceeded`.

## Вне scope

- Автоматическое управление миграциями версий схемы (в т.ч. downgrade/upgrade).
- Сложная фильтрация (metadata filters), multi-tenant изоляция, RLS.
- Конфигурируемые метрики similarity и hybrid search.
- Batch-операции (bulk upsert) и асинхронная индексация.

## Критерии приемки

### AC-001 Публичная фабрика pgvector VectorStore

- Почему это важно: пользователь библиотеки должен иметь доступ к реализации без импорта `internal/...`.
- **Given** пользователь пишет код, импортируя только `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он создаёт pgvector store через публичный API
- **Then** код компилируется без обращения к `internal/...`, а возвращаемое значение удовлетворяет интерфейсу `draftrag.VectorStore`
- Evidence: пример/тест компиляции в пакете `pkg/draftrag` (или `go doc`) демонстрирует фабрику.

### AC-002 Setup helper создаёт schema/table/index

- Почему это важно: без минимального helper’а старт требует ручного SQL и нарушает принцип «разумные дефолты».
- **Given** пустая база PostgreSQL с установленным расширением pgvector (или права на `CREATE EXTENSION`)
- **When** вызывается `draftrag.SetupPGVector(ctx, db, opts)` (или аналог)
- **Then** таблица хранения и индекс создаются, повторный вызов идемпотентен и не падает
- Evidence: интеграционный тест с реальной БД (условный, по DSN) или проверка DDL через запрос к `information_schema`.

### AC-003 Upsert/Delete работают для persisted чанков

- Почему это важно: базовый контракт `VectorStore` должен быть корректным на реальном хранилище.
- **Given** pgvector store и валидный `Chunk` с embedding
- **When** вызываются `Upsert`, затем `Delete`
- **Then** чанк доступен через поиск после Upsert и отсутствует после Delete
- Evidence: интеграционный тест (по DSN) подтверждает поведение.

### AC-004 Search возвращает topK и корректный score

- Почему это важно: retrieval качество и сортировка напрямую влияют на RAG.
- **Given** в хранилище есть несколько чанков с разной близостью к query-embedding
- **When** выполняется `Search(ctx, embedding, topK)`
- **Then** возвращается не более `topK` результатов, отсортированных по `Score` desc, и `Score` в [-1, 1]
- Evidence: интеграционный тест (по DSN) проверяет порядок и диапазон score.

### AC-005 Совместимость с Pipeline

- Почему это важно: pgvector store должен быть взаимозаменяемым с in-memory реализацией на уровне use-case.
- **Given** `pipeline := draftrag.NewPipeline(pgvectorStore, llm, embedder)`
- **When** выполняются `Index` и `QueryTopK`
- **Then** `QueryTopK` возвращает не пустой `RetrievalResult` при наличии релевантных данных
- Evidence: интеграционный тест (по DSN) или пример в godoc.

## Допущения

- У пользователя есть доступный PostgreSQL и возможность включить pgvector (manual или через роли).
- Размерность embedding фиксирована для конкретного хранилища; пользователь/Embedder обеспечивают согласованность размерности.
- Пользователь управляет `*sql.DB` (пулом соединений) и закрывает его вне draftRAG.

## Критерии успеха

- SC-001 Интеграционные тесты pgvector можно запустить локально одной переменной окружения (например, `PGVECTOR_TEST_DSN`) без правок кода.
- SC-002 `Search` на тестовом наборе (десятки/сотни чанков) завершает запрос <200мс на среднем ноутбуке (best-effort, без строгих гарантий).

## Краевые случаи

- Отсутствует расширение pgvector / недостаточно прав: `SetupPGVector` возвращает понятную ошибку.
- `topK <= 0`: `Search` возвращает ошибку валидации.
- Пустое хранилище: `Search` возвращает пустой результат без ошибки.
- Отмена контекста во время SQL-операции: методы возвращают `context.Canceled`/`context.DeadlineExceeded`.

## Открытые вопросы

- Нужно ли поддерживать несколько метрик (cosine/inner product/L2) в v1, или фиксируем только cosine?
- Нужен ли отдельный публичный пакет `pkg/draftrag/pgvector`, или достаточно функций в корневом `pkg/draftrag`?

