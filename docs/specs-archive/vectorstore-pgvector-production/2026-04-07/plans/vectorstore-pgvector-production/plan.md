# VectorStore pgvector: production-ready (migrations, индексы, фильтры, лимиты/таймауты) План

## Phase Contract

Inputs: `.speckeep/specs/vectorstore-pgvector-production/spec.md` и узкий контекст текущей реализации pgvector.
Outputs: `.speckeep/plans/vectorstore-pgvector-production/plan.md`, `.speckeep/plans/vectorstore-pgvector-production/data-model.md`.
Stop if: потребуется breaking change в `domain.VectorStore` или “скрытые” продуктовые решения, не отражённые в spec.

## Цель

Добавить production-ориентированный контур для PostgreSQL+pgvector: версионированные миграции схемы (вместо единого Setup helper), детерминированные индексы под retrieval и фильтр `ParentID`, а также явные лимиты/таймауты на DDL и runtime операции. При этом сохранить обратную совместимость для текущих пользователей `domain.VectorStore` (вариант A: новый интерфейс `VectorStoreWithFilters` + новые методы Pipeline).

## Scope

- Версионированный migrator для pgvector DDL (V1/V2 из spec).
- Расширение pgvector store: поиск с фильтром по `ParentID` (SQL `WHERE parent_id = ANY(...)`).
- Runtime options: дефолтные таймауты и лимиты (topK, parentIDs, content bytes).

## Implementation Surfaces

- `pkg/draftrag/pgvector.go` (существующая surface): эволюция `SetupPGVector` → `MigratePGVector`/миграции; расширение options для production.
- `internal/infrastructure/vectorstore/pgvector.go` (существующая surface): добавить `SearchWithFilter` (и, при необходимости, хранение runtime options/timeout wrapper).
- `internal/domain/interfaces.go` (существующая surface): добавить новый интерфейс `VectorStoreWithFilters` и тип фильтра `ParentIDFilter`.
- `internal/application/pipeline.go` (существующая surface): добавить новые методы `QueryWithParentIDs` / `AnswerWithParentIDs` / `AnswerWithCitationsWithParentIDs` (или аналогичное), которые используют `VectorStoreWithFilters`.
- `*_test.go` (существующая surface): обновить/добавить unit+integration тесты под AC.

## Влияние на архитектуру

- Domain-граница расширяется не через изменение существующего интерфейса, а через новый “capability interface” (`VectorStoreWithFilters`), что сохраняет Clean Architecture и минимизирует breaking changes.
- Infrastructure pgvector реализация получает новую capability поверх существующего store.
- Application Pipeline получает новые методы (не меняя существующие), чтобы потребитель мог включить фильтр `ParentID` без прямого доступа к SQL.

## Acceptance Approach

- AC-001 (миграции идемпотентны): реализовать версионированный migrator + `schema_migrations` (или эквивалент) + тест на повторный запуск (в integration тестах с реальной БД; при отсутствии pgvector — skip как сейчас).
- AC-002 (IVFFLAT/HNSW): миграция/DDL-генератор должен создавать embedding-индекс согласно `IndexMethod`; тест — проверка `pg_indexes`/`pg_class` (integration).
- AC-003 (фильтр ParentID): `SearchWithFilter` использует `WHERE parent_id = ANY($2)` и валидацию списка; unit тест на формирование результатов + integration тест на отсутствие “чужих” parent_id.
- AC-004 (таймауты/лимиты): добавить wrapper, который если у ctx нет deadline — применяет дефолтный timeout per-op; тест — контекст с коротким timeout и ожидаемая ошибка.
- AC-005 (совместимость): существующие вызовы `Search(ctx, emb, topK)` и pipeline-методы должны оставаться без изменений и продолжать работать; новые методы доступны только если store поддерживает filters capability.

## Данные и контракты

- Data model меняется: таблица чанков расширяется полями `metadata`, `created_at`, `updated_at` (V2) и добавляются индексы на `parent_id` и `(parent_id, position)`. См. `.speckeep/plans/vectorstore-pgvector-production/data-model.md`.
- Публичные API контракты: добавляются новые функции/options в `pkg/draftrag` и новые методы в `Pipeline` (Go API). HTTP/event contracts не требуются.

## Стратегия реализации

- DEC-001 Capability interface для фильтров вместо breaking change
  Why: сохранить обратную совместимость и не ломать пользователей `domain.VectorStore`.
  Tradeoff: появляется второй путь поиска (`Search` и `SearchWithFilter`) и необходимость обработки “capability missing”.
  Affects: `internal/domain/interfaces.go`, `internal/application/pipeline.go`, `internal/infrastructure/vectorstore/pgvector.go`.
  Validation: AC-005; unit тест, что pipeline возвращает явную ошибку при отсутствии capability.

- DEC-002 Версионированные миграции без внешней зависимости
  Why: библиотеке нужен предсказуемый DDL без привязки к сторонним миграторам.
  Tradeoff: ограниченный функционал миграций (без `CONCURRENTLY`, без rollback).
  Affects: `pkg/draftrag/pgvector.go` (+ новые файлы в `pkg/draftrag` при необходимости).
  Validation: AC-001, AC-002; integration тесты.

- DEC-003 Детерминированные имена индексов и стратегия смены IndexMethod
  Why: чтобы миграции были идемпотентными и предсказуемыми.
  Tradeoff: смена IndexMethod требует drop+create (offline rebuild), что может быть дорогим.
  Affects: `pkg/draftrag/pgvector.go` (DDL builder), миграции.
  Validation: AC-002; проверка наличия ровно одного embedding-индекса нужного типа.

- DEC-004 Таймауты через контекст с дефолтами
  Why: соблюсти конституцию (контекстная безопасность) и избежать зависаний без deadline.
  Tradeoff: дефолты могут быть неидеальны для всех окружений, поэтому они конфигурируемы.
  Affects: `pkg/draftrag` options, `internal/infrastructure/vectorstore/pgvector.go`, `internal/application/pipeline.go` (если добавляем).
  Validation: AC-004; unit тесты на поведение при ctx без deadline.

## Incremental Delivery

### MVP (Первая ценность)

- Реализовать `VectorStoreWithFilters` + `SearchWithFilter` для pgvector + pipeline-методы с ParentIDs (AC-003, AC-005 частично).
- Добавить лимиты `MaxTopK`/`MaxParentIDs` (AC-004 частично: лимиты).
- Критерий готовности MVP: unit tests на валидацию и фильтрацию + сборка `go test ./...`.

### Итеративное расширение

- Добавить migrator V1/V2 и `schema_migrations` (AC-001, AC-002).
- Добавить дефолтные таймауты per-op (AC-004 полностью).
- Добавить integration тесты (условно-скипаемые) на DDL и индексы (AC-001/002/003).

## Порядок реализации

- Сначала: domain capability interface + pgvector `SearchWithFilter` + pipeline методы (самое “видимое” поведение, минимальные DDL изменения).
- Затем: миграции и расширение схемы (V1/V2) + детерминированные индексы.
- Параллельно можно: unit тесты на лимиты/валидации и ошибки capability.
- В конце: integration тесты (с skip при отсутствии pgvector/прав) и документирование upgrade path в godoc.

## Риски

- Риск: online rebuild индексов (CONCURRENTLY) вне scope может быть нужен production пользователям.
  Mitigation: явно задокументировать в godoc/README, что смена `IndexMethod` делает drop+create; рекомендовать выполнять миграции в maintenance window.
- Риск: дефолтные таймауты могут не подходить всем.
  Mitigation: сделать таймауты конфигурируемыми и не переопределять deadline, если он уже задан в ctx.
- Риск: integration тесты зависят от доступности Postgres+pgvector.
  Mitigation: использовать skip-паттерн, уже применяемый в `pgvector_test.go`.

## Rollout и compatibility

- Backward compatibility: сохраняем `SetupPGVector` (как thin wrapper) или предоставляем мигратор новым API, оставив старый как deprecated (без удаления).
- Миграции должны быть безопасны при повторном запуске; запускать их рекомендуется отдельным шагом деплоя.
- Специальных feature flags не требуется (библиотека), но новые методы pipeline — опциональны.

## Проверка

- Unit tests:
  - валидация лимитов `topK`, `parentIDs` и `content` (AC-004 частично).
  - поведение pipeline при store без `VectorStoreWithFilters` (DEC-001, AC-005).
- Integration tests (skip если нет pgvector/прав):
  - миграции применяются идемпотентно и создают ожидаемые объекты (AC-001).
  - создан индекс нужного типа (AC-002).
  - поиск с фильтром не возвращает “чужие” parent_id (AC-003).
  - таймауты соблюдаются (AC-004).

## Соответствие конституции

- нет конфликтов
