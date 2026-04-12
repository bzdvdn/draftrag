# ChromaDB: миграции коллекций

## Scope Snapshot

- **In scope**: Добавить функции управления коллекциями ChromaDB (`CreateCollection`, `DeleteCollection`, `CollectionExists`) в публичный API, аналогично Qdrant.
- **Out of scope**: Гибридный поиск (BM25) для ChromaDB — требует отдельного исследования возможностей ChromaDB.

## Цель

Разработчики, использующие ChromaDB как векторное хранилище, получают возможность программно создавать и удалять коллекции без прямого обращения к ChromaDB HTTP API. Это упрощает onboarding и делает код более переносимым между разными векторными хранилищами (pgvector, Qdrant, ChromaDB имеют единый API миграций).

## Основной сценарий

1. Разработчик импортирует `draftrag` и создаёт `ChromaDBOptions` с параметрами подключения
2. Вызывает `draftrag.CreateCollection(ctx, opts)` перед созданием `VectorStore`
3. Коллекция создаётся в ChromaDB с указанной размерностью векторов
4. Разработчик создаёт `VectorStore` через `NewChromaDBStore(opts)` и использует его
5. При необходимости удаления — вызывает `draftrag.DeleteCollection(ctx, opts)`

## Scope

- Функция `CreateCollection(ctx, opts ChromaDBOptions) error`
- Функция `DeleteCollection(ctx, opts ChromaDBOptions) error`
- Функция `CollectionExists(ctx, opts ChromaDBOptions) (bool, error)`
- Метод `Validate()` для `ChromaDBOptions` (как у `QdrantOptions`)
- Тесты для всех трёх функций с HTTP mock-сервером
- Обновление `NewChromaDBStore` для использования `Validate()`

## Контекст

- ChromaDB использует REST API v1 (`/api/v1/collections`)
- Существующая реализация `ChromaStore` в `internal/infrastructure/vectorstore/chromadb.go` уже поддерживает базовые операции (Upsert, Search, Delete)
- Аналогичные функции для Qdrant уже реализованы в `pkg/draftrag/qdrant.go` — можно использовать как reference
- ChromaDB API возвращает 200 на успешное создание, 409 на "already exists", 404 на "not found"

## Требования

- **RQ-001**: `CreateCollection` создаёт коллекцию в ChromaDB с указанной размерностью векторов через POST `/api/v1/collections`
- **RQ-002**: `DeleteCollection` удаляет коллекцию через DELETE `/api/v1/collections/{name}`; 404 не считается ошибкой (идемпотентность)
- **RQ-003**: `CollectionExists` проверяет существование коллекции через GET `/api/v1/collections/{name}`; возвращает `true` если статус 200, `false` если 404
- **RQ-004**: `ChromaDBOptions.Validate()` проверяет что `Collection` не пустой и `Dimension > 0`
- **RQ-005**: Все функции принимают `context.Context` и поддерживают cancellation/timeout
- **RQ-006**: `NewChromaDBStore` использует `opts.Validate()` для валидации входных параметров

## Вне scope

- Гибридный поиск (BM25 + semantic) — ChromaDB не поддерживает BM25 нативно
- Управление индексами внутри коллекции — используются дефолтные настройки ChromaDB
- Batch-операции для миграций — создаётся/удаляется одна коллекция за вызов
- Миграции схемы (schema versioning) — как и у Qdrant, управление коллекциями = create/delete

## Критерии приемки

### AC-001 Создание коллекции

- **Почему важно**: Базовая операция для начала работы с ChromaDB
- **Given**: ChromaDB сервер доступен, коллекция "test" не существует
- **When**: Вызывается `CreateCollection(ctx, ChromaDBOptions{Collection: "test", Dimension: 1536})`
- **Then**: В ChromaDB создаётся коллекция "test" с размерностью 1536, функция возвращает `nil`
- **Evidence**: HTTP 200 от ChromaDB API, последующий вызов `CollectionExists` возвращает `true`

### AC-002 Удаление коллекции

- **Почему важно**: Нужно для cleanup в тестах и при удалении данных
- **Given**: Коллекция "test" существует в ChromaDB
- **When**: Вызывается `DeleteCollection(ctx, ChromaDBOptions{Collection: "test"})`
- **Then**: Коллекция удалена, функция возвращает `nil`
- **Evidence**: HTTP 200 или 404 от ChromaDB, `CollectionExists` возвращает `false`

### AC-003 Проверка существования (коллекция есть)

- **Почему важно**: Idempotent operations — перед созданием можно проверить
- **Given**: Коллекция "test" существует
- **When**: Вызывается `CollectionExists(ctx, ChromaDBOptions{Collection: "test"})`
- **Then**: Функция возвращает `(true, nil)`
- **Evidence**: HTTP 200 от ChromaDB API

### AC-004 Проверка существования (коллекции нет)

- **Почему важно**: Корректная обработка отсутствующих ресурсов
- **Given**: Коллекция "missing" не существует
- **When**: Вызывается `CollectionExists(ctx, ChromaDBOptions{Collection: "missing"})`
- **Then**: Функция возвращает `(false, nil)`
- **Evidence**: HTTP 404 от ChromaDB API не считается ошибкой

### AC-005 Валидация опций

- **Почему важно**: Fail fast — ошибки конфигурации ловятся до HTTP запроса
- **Given**: `ChromaDBOptions{Collection: "", Dimension: 0}`
- **When**: Вызывается `Validate()` или любая миграционная функция
- **Then**: Возвращается ошибка валидации (collection required, dimension must be > 0)
- **Evidence**: Ошибка с понятным сообщением, HTTP запрос не выполняется

### AC-006 Контекстная отмена

- **Почему важно**: Production-ready — поддержка timeout и cancellation
- **Given**: Создан `context.WithTimeout(ctx, 1ms)` который быстро истекает
- **When**: Вызывается `CreateCollection` с этим контекстом на медленном соединении
- **Then**: Функция возвращает ошибку `context.DeadlineExceeded`
- **Evidence**: Тест с timeout показывает корректную обработку cancellation

## Допущения

- ChromaDB сервер версии 0.4.x+ с HTTP API v1 (как указано в существующем коде)
- Дефолтный `BaseURL`: `http://localhost:8000` (как в существующем `NewChromaStore`)
- Все операции синхронные (HTTP request-response), нет long-polling или streaming
- Аутентификация не требуется (ChromaDB по умолчанию без auth в локальном режиме)
- Коллекция создаётся с дефолтными настройками ChromaDB (distance metric = Cosine, как у Qdrant)

## Критерии успеха

- **SC-001**: Все тесты проходят с покрытием >80% для новых функций
- **SC-002**: API консистентен с Qdrant: те же имена функций, похожая сигнатура `Options`
- **SC-003**: Пример использования в `examples/chromadb/` или обновление существующего примера

## Краевые случаи

- Создание коллекции которая уже есть: ChromaDB возвращает 409 — обработать как ошибку или success (решить при планировании)
- Удаление несуществующей коллекции: 404 — не ошибка (идемпотентность как у Qdrant)
- Невалидный JSON от ChromaDB: обернуть в понятную ошибку
- Пустой `BaseURL` в опциях: использовать дефолт `http://localhost:8000`

## Открытые вопросы

- none

---

**Slug**: `chromadb-migrations`

**Следующая команда**: `/speckeep.inspect chromadb-migrations`
