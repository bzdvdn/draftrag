# ChromaDB: управление коллекциями

## Scope Snapshot

- In scope: публичный API для создания, удаления и проверки существования коллекции ChromaDB через новый domain-интерфейс `CollectionManager`.
- Out of scope: гибридный поиск для ChromaDB, управление коллекциями других хранилищ (Qdrant, Milvus, pgvector).

## Цель

Разработчики, использующие draftRAG с ChromaDB, сейчас не могут программно создавать или удалять коллекцию через библиотеку — нужно делать это вручную или через отдельный HTTP-клиент. Фича добавляет domain-интерфейс `CollectionManager` и реализует его в `ChromaStore`, чтобы жизненный цикл коллекции был полностью управляем из кода на Go.

## Основной сценарий

1. Разработчик создаёт `ChromaStore` для нового проекта.
2. Перед индексацией вызывает `store.CreateCollection(ctx)` — коллекция создаётся в ChromaDB (или уже существует — ошибки нет).
3. После завершения работы вызывает `store.DeleteCollection(ctx)` — коллекция удаляется.
4. Для условной логики вызывает `store.CollectionExists(ctx)` — получает `true`/`false` без ошибки.
5. Если ChromaDB недоступен — все три метода возвращают ошибку с понятным сообщением.

## Scope

- Новый интерфейс `CollectionManager` в `internal/domain/interfaces.go`
- Реализация `CreateCollection`, `DeleteCollection`, `CollectionExists` в `ChromaStore`
- Unit-тесты через `httptest.NewServer` для всех трёх методов и их edge cases
- Compile-time assertion: `var _ domain.CollectionManager = (*ChromaStore)(nil)`

## Контекст

- `ChromaStore` уже имеет приватный метод `createCollection` (autocreate при Upsert) — публичный `CreateCollection` должен переиспользовать ту же логику или заменить её.
- ChromaDB REST API v1: `POST /api/v1/collections` (create), `DELETE /api/v1/collections/{name}` (delete), `GET /api/v1/collections/{name}` (exists check).
- Другие хранилища (Qdrant, pgvector) управляют коллекциями/таблицами через отдельные методы — паттерн известен в репозитории.
- Domain-слой не должен импортировать внешние пакеты (конституция).

## Требования

- RQ-001 `CreateCollection(ctx)` создаёт коллекцию в ChromaDB; повторный вызов при уже существующей коллекции не возвращает ошибку (idempotent).
- RQ-002 `DeleteCollection(ctx)` удаляет коллекцию; вызов для несуществующей коллекции возвращает ошибку.
- RQ-003 `CollectionExists(ctx)` возвращает `(bool, error)`: `true` если коллекция существует, `false` если нет, `error` только при сетевой или серверной ошибке (не при отсутствии коллекции).
- RQ-004 Все методы принимают `context.Context` первым аргументом и уважают отмену/таймаут.
- RQ-005 Интерфейс `CollectionManager` объявлен в domain-слое и не содержит ChromaDB-специфичных типов.

## Вне scope

- Гибридный поиск (BM25) для ChromaDB — отдельная фича с более сложным scope.
- Управление коллекциями в Qdrant, Milvus, pgvector, in-memory — каждое хранилище отдельно.
- Настройка параметров коллекции (hnsw, размерность) через `CollectionManager` — ChromaStore уже хранит dimension, менять API не нужно.
- Список всех коллекций (`ListCollections`) — не требуется для заявленного сценария.

## Критерии приемки

### AC-001 CreateCollection создаёт коллекцию idempotently

- Почему это важно: разработчик может вызвать `CreateCollection` при старте сервиса не зная, существует ли коллекция.
- **Given** ChromaDB запущен, коллекция с данным именем отсутствует
- **When** вызван `store.CreateCollection(ctx)`
- **Then** метод возвращает `nil`; повторный вызов тоже возвращает `nil`
- Evidence: unit-тест с mock-сервером, возвращающим 200 на `POST /api/v1/collections`; повторный вызов — тест с `get_or_create: true` в теле запроса.

### AC-002 DeleteCollection удаляет существующую коллекцию

- Почему это важно: автоматизированные тесты и teardown-сценарии требуют программного удаления.
- **Given** ChromaDB доступен
- **When** вызван `store.DeleteCollection(ctx)`
- **Then** метод отправляет `DELETE /api/v1/collections/{name}` и возвращает `nil` при HTTP 200
- Evidence: unit-тест с mock-сервером, захватывающим метод и путь запроса.

### AC-003 DeleteCollection idempotent при 404, ошибка при других HTTP 4xx/5xx

- Почему это важно: 404 означает «коллекции нет» — удаление уже достигнуто; это согласовано с поведением `DeleteChromaDBCollection` в `pkg/draftrag/chromadb.go`.
- **Given** mock-сервер возвращает HTTP 404
- **When** вызван `store.DeleteCollection(ctx)`
- **Then** метод возвращает `nil` (idempotent)
- **Given** mock-сервер возвращает HTTP 500
- **When** вызван `store.DeleteCollection(ctx)`
- **Then** метод возвращает ненулевую ошибку, содержащую статус-код
- Evidence: два unit-теста — mock 404 → `nil`; mock 500 → `err != nil` со статусом в строке.

### AC-004 CollectionExists возвращает true для существующей коллекции

- Почему это важно: условная логика инициализации не должна создавать коллекцию дважды.
- **Given** ChromaDB доступен, коллекция существует (mock возвращает 200 на GET)
- **When** вызван `store.CollectionExists(ctx)`
- **Then** возвращает `(true, nil)`
- Evidence: unit-тест с mock-сервером, возвращающим 200 на `GET /api/v1/collections/{name}`.

### AC-005 CollectionExists возвращает false при отсутствии коллекции

- Почему это важно: отличает "коллекция не существует" от "произошла ошибка" — разные пути в клиентском коде.
- **Given** mock-сервер возвращает HTTP 404 на GET коллекции
- **When** вызван `store.CollectionExists(ctx)`
- **Then** возвращает `(false, nil)` — не ошибку
- Evidence: unit-тест с mock-сервером, возвращающим 404; проверка `exists == false && err == nil`.

### AC-006 Compile-time assertion для CollectionManager

- Почему это важно: гарантирует, что `ChromaStore` не отстаёт от интерфейса при рефакторинге.
- **Given** код компилируется
- **When** в `chromadb.go` присутствует `var _ domain.CollectionManager = (*ChromaStore)(nil)`
- **Then** `go build ./...` завершается без ошибок
- Evidence: `go build ./...` и `go vet ./...` без ошибок.

## Допущения

- ChromaDB REST API v1 стабилен (используется в текущей реализации `chromadb.go`).
- `GET /api/v1/collections/{name}` возвращает 404 при отсутствии коллекции и 200 при наличии — стандартное поведение ChromaDB 0.4.x+.
- `DELETE /api/v1/collections/{name}` возвращает 200 при успехе — стандартное поведение ChromaDB 0.4.x+.
- Существующий приватный `createCollection` в `ChromaStore` переиспользуется или поглощается публичным `CreateCollection` без изменения поведения Upsert.

## Краевые случаи

- `CollectionExists` при сетевой ошибке (таймаут, connection refused) — возвращает `(false, error)`.
- `CreateCollection` при HTTP 5xx от ChromaDB — возвращает ненулевую ошибку.
- Пустое имя коллекции в `ChromaStore` — поведение определяется ChromaDB (не валидируется на стороне клиента, ChromaDB вернёт 4xx).

## Открытые вопросы

- none
