# VectorStore pgvector (PostgreSQL) для draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/vectorstore-pgvector/spec.md`, `.draftspec/specs/vectorstore-pgvector/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: spec недостаточно конкретна для выбора публичной API поверхности и формы DDL/schema.

## Цель

Добавить production-ready реализацию `VectorStore` на PostgreSQL+pgvector, сохранив принципы конституции (Clean Architecture, интерфейсная абстракция, `context.Context` во всех операциях, минимальная конфигурация). Реализация должна быть доступна пользователю через `pkg/draftrag` и не требовать импорта `internal/...`, а тестовый контур должен проходить без поднятой БД по умолчанию (интеграционные тесты — opt-in).

## Scope

- Infrastructure: реализация `VectorStore` поверх `database/sql` (Upsert/Delete/Search) с использованием pgvector.
- Public API: фабрика `NewPGVectorStore(db, opts)` + helper `SetupPGVector(ctx, db, opts)` в `pkg/draftrag`.
- Data model: таблица чанков и индекс по embedding (IVFFlat/HNSW — выбрать в реализации; в v1 допустимо IVFFlat как более распространённый baseline).
- Testing: интеграционные тесты по DSN (env var) с `t.Skip()` по умолчанию.
- Out: автоматические миграции версий схемы, сложные фильтры, hybrid search.

## Implementation Surfaces

- internal/infrastructure/vectorstore/pgvector.go — новая реализация `VectorStore` на PostgreSQL+pgvector (T1.2, T2.1, T2.2).
- pkg/draftrag/pgvector.go — публичные API: `NewPGVectorStore`, `SetupPGVector`, `PGVectorOptions` (T1.1, T2.3).
- internal/infrastructure/vectorstore/pgvector_test.go — интеграционные тесты store (opt-in DSN) (T3.1).
- pkg/draftrag/pgvector_test.go — интеграционный тест совместимости с `Pipeline` (opt-in DSN) (T3.2).

## Влияние на архитектуру

- Clean Architecture сохраняется: интерфейс `VectorStore` в `internal/domain`, реализация — в `internal/infrastructure`.
- Публичный доступ — через `pkg/draftrag`, без экспонирования `internal/...`.
- Добавляется зависимость на PostgreSQL (в runtime) и на pgvector тип/кодек (в compile-time) — ограничивается только pgvector feature package.

## Acceptance Approach

- AC-001 -> `pkg/draftrag/pgvector.go` экспортирует фабрику, возвращающую `draftrag.VectorStore`; compile-time подтверждение через `go test ./...` и godoc. Surfaces: pkg/draftrag/pgvector.go, internal/infrastructure/vectorstore/pgvector.go.
- AC-002 -> `SetupPGVector(ctx, db, opts)` применяет идемпотентный DDL (таблица + индекс). Surfaces: `pkg/draftrag/pgvector.go`; evidence: интеграционный тест по DSN проверяет повторный вызов.
- AC-003 -> интеграционный тест по DSN: Upsert → Search находит → Delete → Search не находит. Surfaces: internal/infrastructure/vectorstore/pgvector_test.go.
- AC-004 -> интеграционный тест по DSN: `Search` возвращает `<= topK`, сортировка по score desc, score в [-1, 1]. Surfaces: `internal/infrastructure/vectorstore/pgvector_test.go`.
- AC-005 -> интеграционный тест по DSN: `draftrag.NewPipeline(pgvectorStore, llm, embedder)` + `Index` + `QueryTopK` возвращают результаты. Surfaces: `pkg/draftrag/pgvector_test.go`.

## Данные и контракты

- Data model фиксируется в `.draftspec/plans/vectorstore-pgvector/data-model.md` (таблица чанков).
- Публичный контракт (в `pkg/draftrag`):
  - `type PGVectorOptions struct { TableName string; EmbeddingDimension int; CreateExtension bool; IndexMethod string; Lists int; }` (точные поля финализируются в tasks, но intent фиксируется здесь).
  - `func SetupPGVector(ctx context.Context, db *sql.DB, opts PGVectorOptions) error`
  - `func NewPGVectorStore(db *sql.DB, opts PGVectorOptions) VectorStore`
- Контракты событий/HTTP отсутствуют.

## Стратегия реализации

- DEC-001 Используем `database/sql` как основной boundary
  Why: стандартная абстракция Go, не привязывает к драйверу; соответствует «минимальной конфигурации» для библиотечного кода.
  Tradeoff: ограничения типизации для pgvector и необходимости аккуратного encoding/decoding.
  Affects: `internal/infrastructure/vectorstore/pgvector.go`, `pkg/draftrag/pgvector.go`
  Validation: интеграционные тесты с любым `database/sql` драйвером (ожидаемо pgx) проходят по DSN.

- DEC-002 Helper `SetupPGVector` создаёт таблицу и индекс идемпотентно; `CREATE EXTENSION` — опционально
  Why: права на `CREATE EXTENSION` часто отсутствуют; это отмечено в inspect warnings.
  Tradeoff: пользователю может потребоваться отдельный manual step для расширения.
  Affects: `pkg/draftrag/pgvector.go`
  Validation: интеграционный тест подтверждает, что повторный Setup не падает; при отсутствии прав — возвращает понятную ошибку.

- DEC-003 Метрика в v1 фиксирована: cosine distance в SQL + score как similarity
  Why: держим v1 простой и совпадающей со spec; расширяемость возможна в v2.
  Tradeoff: нельзя переключить метрику без расширения API/DDL.
  Affects: `internal/infrastructure/vectorstore/pgvector.go`
  Validation: тест проверяет, что `Score` в [-1, 1] и порядок результатов согласован с ожидаемой близостью.

- DEC-004 Формула score: `score = 1 - cosine_distance`
  Why: cosine distance в pgvector соответствует `1 - cosine_similarity`; преобразование возвращает диапазон [-1, 1] (как требуется в core).
  Tradeoff: потребуются проверки на NaN/Inf (если входные embedding невалидны).
  Affects: `internal/infrastructure/vectorstore/pgvector.go`
  Validation: интеграционный тест проверяет диапазон score; unit-проверки на некорректный `topK`.

- DEC-005 Интеграционные тесты запускаются только при наличии DSN
  Why: требование RQ-005 — `go test ./...` без внешних сервисов.
  Tradeoff: часть поведения проверяется только в opt-in контуре.
  Affects: `internal/infrastructure/vectorstore/pgvector_test.go`, `pkg/draftrag/pgvector_test.go`
  Validation: без DSN тесты пропускаются; с DSN — проходят.

## Incremental Delivery

### MVP (Первая ценность)

- Реализация `NewPGVectorStore` + `PGVectorStore.Upsert/Delete/Search` с базовой таблицей и simple Setup (таблица + индекс).
- Интеграционный тест Search/Upsert/Delete по DSN.
- Критерий готовности MVP: AC-001, AC-003, AC-004.

### Итеративное расширение

- `SetupPGVector` с управлением `CreateExtension` и более явными options.
- Интеграционный тест совместимости с `Pipeline`.
- Критерий готовности: AC-002, AC-005.

## Порядок реализации

1. Определить публичные options и сигнатуры API в `pkg/draftrag/pgvector.go`.
2. Реализовать `internal/infrastructure/vectorstore/pgvector.go` (CRUD + Search).
3. Добавить `SetupPGVector` (DDL + idempotency).
4. Добавить интеграционные тесты (store, затем pipeline).

## Риски

- Риск 1: Pgvector DDL/операторы зависят от версии расширения и типа индекса.
  Mitigation: ограничиться одной метрикой и одним индексом в v1; держать DDL простым и идемпотентным.
- Риск 2: Права на `CREATE EXTENSION` отсутствуют.
  Mitigation: сделать `CreateExtension` опцией и возвращать понятную ошибку; в docs зафиксировать предпосылки.
- Риск 3: Слишком сильная зависимость API от схемы таблицы.
  Mitigation: скрыть детали в options + оставить возможность смены TableName.

## Rollout и compatibility

- Специальных rollout шагов нет (библиотека).
- Схема создаётся по запросу через helper; existing deployments контролируются пользователем.

## Проверка

- `go test ./...` без DSN проходит (интеграционные тесты skip).
- С DSN:
  - `go test ./...` выполняет pgvector integration tests и подтверждает AC-002..AC-005.
- `go vet ./...` без предупреждений.

## Соответствие конституции

- нет конфликтов: чистая архитектура сохранена; внешняя зависимость (PostgreSQL) инкапсулирована в infrastructure; публичные методы принимают `context.Context`.
