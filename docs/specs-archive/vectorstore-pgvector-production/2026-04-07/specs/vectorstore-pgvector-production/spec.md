# VectorStore pgvector: production-ready (migrations, индексы, фильтры, лимиты/таймауты)

## Scope Snapshot

- In scope: сделать PostgreSQL+pgvector реализацию VectorStore пригодной для production за счёт версионированных миграций, выбора индекса (HNSW/IVFFLAT), фильтрации Search по `ParentID`, а также явных лимитов и таймаутов.
- Out of scope: добавление новых типов векторных БД/провайдеров, многоарендность (tenant/namespace), полнотекстовый поиск, кластеризация/шардинг, персистентные фоновые джобы по rebuild индексов.

## Цель

Разработчик Go-приложения получает предсказуемый и безопасный способ поднять и обновлять схему pgvector-хранилища (DDL и индексы) в production, а также выполнять retrieval с ограничениями (timeouts/limits) и с фильтрами по `ParentID`, не прибегая к ручному SQL и не рискуя деградацией производительности/стабильности.

## Основной сценарий

1. Приложение стартует и вызывает `draftrag.MigratePGVector(...)` (или эквивалентный public helper) один раз при деплое/старте.
2. Мигратор идемпотентно применяет недостающие версии схемы (extension/table/indexes/constraints).
3. При обработке запроса retrieval код вызывает поиск с фильтром по `ParentID` (например, ограничить поиск чанков рамками одного документа или набора документов).
4. Поиск и запись соблюдают таймауты и лимиты; при превышении — возвращают явную ошибку (или предсказуемый отказ), не уводя БД в долгие запросы.

## Scope

- Версионированные миграции схемы pgvector-хранилища (минимальный встроенный migrator без тяжёлых зависимостей).
- Поддержка индекса `ivfflat` и `hnsw` через явную конфигурацию, с параметрами, проверяемыми на валидность.
- Фильтрация поиска по `ParentID` на уровне SQL (не пост-фильтрацией в памяти).
- Защитные лимиты и таймауты для DDL, upsert/delete/search и для параметров запросов.

## Контекст

- В проекте уже есть pgvector store (`internal/infrastructure/vectorstore/pgvector.go`) и helper для DDL (`pkg/draftrag/pgvector.go`), но DDL не версионирован и Search не поддерживает фильтры.
- По конституции все операции принимают `context.Context`; поведение по таймаутам должно быть контекстно-безопасным и предсказуемым.
- Текущий `domain.VectorStore` имеет `Search(ctx, embedding, topK)` без фильтров; для `ParentID` потребуется эволюция контракта без ломки существующих потребителей, либо контролируемый breaking change.

## Дизайн (конкретизация для реализации)

### Public API (минимальный набросок)

- `func MigratePGVector(ctx context.Context, db *sql.DB, opts PGVectorMigrateOptions) error`
- `type PGVectorMigrateOptions struct { PGVectorOptions; DDLTimeout time.Duration; CreateExtension bool }`
- `type PGVectorRuntimeOptions struct { SearchTimeout, UpsertTimeout, DeleteTimeout time.Duration; MaxTopK int; MaxParentIDs int; MaxContentBytes int }`
- `type domain.ParentIDFilter struct { ParentIDs []string }`
- `type domain.VectorStoreWithFilters interface { domain.VectorStore; SearchWithFilter(ctx context.Context, embedding []float64, topK int, filter ParentIDFilter) (domain.RetrievalResult, error) }`
- `Pipeline` получает новые методы (без ломки существующих): например `QueryWithParentIDs(ctx, question string, topK int, parentIDs []string)` и `AnswerWithParentIDs(...)`, которые используют `VectorStoreWithFilters`, а если store не поддерживает — возвращают явную ошибку (например `ErrFiltersNotSupported`).

### Версии миграций (минимальный набор)

- V1 (совместимость с текущим `SetupPGVector`):
  - (опц.) `CREATE EXTENSION IF NOT EXISTS vector`
  - `CREATE TABLE IF NOT EXISTS <table>(id text pk, parent_id text not null, content text not null, position int not null, embedding vector(<dim>) not null)`
  - `CREATE INDEX IF NOT EXISTS <table>_embedding_idx ...` (hnsw/ivfflat)
- V2 (production расширение без ломки данных):
  - `ALTER TABLE <table> ADD COLUMN IF NOT EXISTS metadata jsonb NOT NULL DEFAULT '{}'::jsonb`
  - `ALTER TABLE <table> ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now()`
  - `ALTER TABLE <table> ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now()`
  - `CREATE INDEX IF NOT EXISTS <table>_parent_id_idx ON <table>(parent_id)`
  - (опц.) `CREATE INDEX IF NOT EXISTS <table>_parent_pos_idx ON <table>(parent_id, position)`

### DDL индекса embedding (пример)

- `ivfflat` (cosine): `CREATE INDEX ... USING ivfflat (embedding vector_cosine_ops) WITH (lists = <N>)`
- `hnsw` (cosine): `CREATE INDEX ... USING hnsw (embedding vector_cosine_ops)` (+ параметры `m`, `ef_construction`, если добавляем)

### SQL-паттерн поиска с фильтром по ParentID

- Без фильтра: `... FROM <table> ORDER BY (embedding <=> $1) ASC LIMIT $2`
- С фильтром: `... FROM <table> WHERE parent_id = ANY($2) ORDER BY (embedding <=> $1) ASC LIMIT $3`

## Требования

- RQ-001 Миграции: система ДОЛЖНА предоставлять public API для версионированного апгрейда схемы pgvector-хранилища (идемпотентно, без ручного SQL).
- RQ-002 Миграции: система ДОЛЖНА хранить версию схемы в БД (например, таблица `schema_migrations` в пределах той же схемы) и применять миграции строго по порядку.
- RQ-003 Миграции: мигратор ДОЛЖЕН поддерживать безопасный повторный запуск (re-entrant): повторный вызов после успешного выполнения не меняет состояние.
- RQ-004 Схема данных: таблица чанков ДОЛЖНА поддерживать минимум поля `id`, `parent_id`, `content`, `position`, `embedding`, совместимые с текущими моделями domain (`Chunk`) и поиском cosine distance.
- RQ-005 Схема данных (production): таблица ДОЛЖНА включать дополнительные поля, необходимые для эксплуатации:
  - `created_at` / `updated_at` (timestamptz) для диагностики и lifecycle;
  - `metadata` (jsonb) как задел под будущие фильтры (но без обязательного использования в этой фиче).
- RQ-006 Индексы: система ДОЛЖНА уметь создавать и использовать индекс по embedding через `ivfflat` или `hnsw` по конфигурации.
- RQ-006a Индексы (ivfflat): дефолты ДОЛЖНЫ быть заданы и валидированы: `lists=100`, а также runtime-настройка `probes=10` (если добавляем управление probes на уровне запросов).
- RQ-006b Индексы (hnsw): дефолты ДОЛЖНЫ быть заданы и валидированы: `m=16`, `ef_construction=64`, `ef_search=40` (если добавляем параметры; иначе фиксируем, что используется DDL без параметров).
- RQ-007 Индексы: система ДОЛЖНА создавать btree-индексы для эффективной фильтрации по `parent_id` и типовым запросам:
  - `<table>_parent_id_idx` на `(parent_id)`
  - `<table>_parent_pos_idx` на `(parent_id, position)`
- RQ-008 Фильтры: система ДОЛЖНА поддерживать поиск с фильтром по `ParentID` на уровне SQL:
  - фильтр по одному `ParentID`;
  - фильтр по списку `ParentID` (ограниченный по размеру).
- RQ-009 API фильтров: система ДОЛЖНА предоставить API, позволяющий использовать фильтр `ParentID` без обязательного breaking change для существующих пользователей:
-  - вариант A (выбран): добавить отдельный интерфейс `domain.VectorStoreWithFilters` + новые методы/опции в Pipeline; старые методы остаются.
- RQ-010 Таймауты: операции DDL и runtime (upsert/delete/search) ДОЛЖНЫ завершаться по deadline контекста; если deadline не задан, система ДОЛЖНА использовать дефолтные таймауты (настраиваемые через options):
  - `DDLTimeout`: 30s
  - `SearchTimeout`: 2s
  - `UpsertTimeout`: 5s
  - `DeleteTimeout`: 5s
- RQ-011 Лимиты: система ДОЛЖНА валидировать и ограничивать параметры:
  - `topK`: `1 <= topK <= MaxTopK`, где `MaxTopK` дефолт = 50;
  - размер списка `ParentID`: `len(parentIDs) <= MaxParentIDs`, где `MaxParentIDs` дефолт = 128;
  - размерность embedding (строго равна `EmbeddingDimension`);
  - максимальную длину `content` при записи (опционально): если `MaxContentBytes > 0`, то `len(content)` ДОЛЖЕН быть `<= MaxContentBytes`, иначе лимит не применяется.
- RQ-012 Безопасность DDL: создание расширения `vector` ДОЛЖНО быть опциональным (как сейчас), а отсутствие прав — возвращать читаемую ошибку с контекстом.

## Вне scope

- Шифрование на уровне столбцов, RLS/ACL, tenant_id/namespace.
- Полноценный менеджмент жизненного цикла индексов (онлайновый rebuild, фоновые анализ/вакуум, автотюнинг параметров).
- Фильтры по произвольным метаданным (кроме `ParentID`).
- Поддержка нескольких таблиц/коллекций в одном store объекте (кроме текущего `TableName`).

## Критерии приемки

### AC-001 Версионированные миграции применяются идемпотентно

- Почему это важно: production деплой не должен зависеть от ручных SQL-операций и должен быть повторяемым.
- **Given** пустая PostgreSQL БД с установленным pgvector (или возможностью `CREATE EXTENSION`)
- **When** приложение вызывает `MigratePGVector(ctx, db, opts)` дважды подряд
- **Then** схема создана, версия сохранена, второй запуск не меняет схему и не падает
- Evidence: наличие таблицы чанков, btree-индекса по `parent_id`, индекса по embedding и записи о версии в `schema_migrations`

### AC-002 Поддержаны индексы IVFFLAT и HNSW

- Почему это важно: разные профили нагрузки требуют разных индексов и trade-offs.
- **Given** конфигурация `IndexMethod=ivfflat` и валидный `lists`
- **When** миграции применены
- **Then** создан ivfflat-индекс с `vector_cosine_ops` и указанными параметрами
- Evidence: `pg_indexes`/`pg_class` отражают метод `ivfflat` и параметры

### AC-003 Поиск с фильтром по ParentID ограничивает результаты

- Почему это важно: retrieval часто должен работать в рамках конкретного документа/набора документов.
- **Given** в таблице есть чанки с разными `parent_id`
- **When** выполняется поиск с фильтром `ParentID in [A, B]`
- **Then** в выдаче отсутствуют чанки с `parent_id` вне `[A, B]`
- Evidence: unit/integration тест, который проверяет `ParentID` каждого returned chunk

### AC-004 Таймауты и лимиты предотвращают “долгие” запросы

- Почему это важно: предотвращает деградацию БД и зависания приложения.
- **Given** контекст без deadline
- **When** выполняется `Search/Upsert/Delete`
- **Then** применяются дефолтные operation timeouts (конфигурируемые), и при превышении возвращается ошибка `context deadline exceeded` (или явно обёрнутая)
- Evidence: тест/пример с коротким timeout, который стабильно фейлится по времени

### AC-005 Совместимость интерфейсов сохранена (если выбран вариант A)

- Почему это важно: снижает стоимость апгрейда и риск breaking changes.
- **Given** существующий код использует `domain.VectorStore.Search(ctx, emb, topK)` без фильтров
- **When** библиотека обновлена
- **Then** сборка проходит без изменений в пользовательском коде, а фильтры доступны через новый API
- Evidence: `go test ./...` проходит, новые методы покрыты тестами

## Допущения

- PostgreSQL версия и pgvector-расширение совместимы с операторами дистанции и индексами `ivfflat`/`hnsw`.
- В production миграции запускаются в контролируемом контуре (deploy job / init container) и могут требовать отдельного уровня прав.
- Фильтр по `ParentID` на данном этапе — единственный обязательный фильтр для retrieval.

## Критерии успеха

- SC-001 Миграции на чистой БД завершаются за <30с при дефолтных настройках (без rebuild больших индексов).
- SC-002 Search p95 <200ms при topK=10 на датасете порядка 100k чанков при корректно настроенном индексе (метод зависит от профиля).
- SC-003 Ошибки, связанные с правами/DDL/таймаутами, возвращаются с контекстом операции (какая миграция/какой этап).

## Краевые случаи

- БД без прав на `CREATE EXTENSION vector`: миграции должны фейлиться предсказуемо и объяснимо (без частично применённой схемы).
- Смена `EmbeddingDimension` для существующей таблицы: НЕ поддерживается в рамках миграций; мигратор должен вернуть ошибку и потребовать новую таблицу (новый `TableName`).
- Смена `IndexMethod` на существующей схеме: мигратор пересоздаёт embedding-индекс (drop + create) под тем же детерминированным именем; online rebuild (`CONCURRENTLY`) вне scope.
- Пустой фильтр `ParentID` (nil/empty): трактуется как “без фильтра”.
- Слишком длинный список `ParentID`: ошибка валидации до выполнения SQL.

## Открытые вопросы

- none
