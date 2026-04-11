# Metadata filtering — План

## Phase Contract

Inputs: `specs/metadata-filtering/spec.md`, `specs/metadata-filtering/inspect.md`, контекст репозитория.
Outputs: `plan.md`, `data-model.md`.
Stop if: spec слишком расплывчата для архитектурных решений.

## Цель

Расширить domain-интерфейс `VectorStoreWithFilters` новым методом `SearchWithMetadataFilter`, реализовать его в pgvector (JSONB `@>`) и in-memory store, провести фильтр через application-слой и опубликовать два новых метода в публичном API. Схема БД не меняется — JSONB-колонка `metadata` уже есть в migration 0002.

## Scope

- `internal/domain/` — новый тип `MetadataFilter` и расширение `VectorStoreWithFilters`
- `internal/infrastructure/vectorstore/` — реализации pgvector и in-memory
- `internal/application/pipeline.go` — два новых use-case метода
- `pkg/draftrag/draftrag.go` — два новых публичных метода `Pipeline`
- Миграции БД, схема хранилища, `Chunk`, `Document` — без изменений

## Implementation Surfaces

- **`internal/domain/interfaces.go`** (существующая) — добавить метод `SearchWithMetadataFilter` в `VectorStoreWithFilters`; контракт `VectorStore` и `SearchWithFilter` не трогать
- **`internal/domain/models.go`** (существующая) — добавить тип `MetadataFilter{Fields map[string]string}` и sentinel `ErrFilterNotSupported`; `Query.Filter` остается как есть без миграции
- **`internal/infrastructure/vectorstore/pgvector.go`** (существующая) — добавить `SearchWithMetadataFilter`; SQL через JSONB `@>` без изменения схемы
- **`internal/domain/models.go`** (существующая, дополнительно) — добавить поле `Metadata map[string]string` в `Chunk`; это нужно, чтобы in-memory store мог фильтровать по метаданным (и pgvector мог читать metadata обратно из БД в `Chunk`)
- **`internal/infrastructure/vectorstore/memory.go`** (существующая) — добавить `SearchWithMetadataFilter` с in-memory итерацией; фильтрует по `chunk.Metadata`
- **`internal/application/pipeline.go`** (существующая) — добавить `QueryWithMetadataFilter` и `AnswerWithMetadataFilter` по аналогии с существующими `QueryWithParentIDs`/`AnswerWithParentIDs`
- **`pkg/draftrag/draftrag.go`** (существующая) — добавить `QueryWithMetadataFilter` и `AnswerWithMetadataFilter` в `Pipeline`; переэкспортировать `MetadataFilter` из domain

## Влияние на архитектуру

- `VectorStoreWithFilters` расширяется новым методом — все существующие реализаторы (`PGVectorStore`) должны получить новый метод, иначе не скомпилируются. Список реализаторов: только `PGVectorStore` в infrastructure; `InMemoryStore` сейчас не реализует `VectorStoreWithFilters`, после этой фичи реализует.
- Публичный API пополняется двумя методами — non-breaking additive change, semver не требует major-bump.
- Нет изменений в схеме БД, wire-формате, migrations или конфигурации.

## Acceptance Approach

- **AC-001** (pgvector фильтрует по metadata) → `SearchWithMetadataFilter` в `PGVectorStore` строит SQL `WHERE metadata @> $N::jsonb`; integration-тест с двумя категориями документов проверяет, что возвращены только совпадающие.
- **AC-002** (пустой фильтр = нет фильтра) → при `len(filter.Fields) == 0` pgvector и in-memory делегируют в базовый `Search`; unit/integration-тест сравнивает результаты.
- **AC-003** (API передаёт фильтр сквозь pipeline) → `QueryWithMetadataFilter` в `pkg/draftrag` получает `MetadataFilter`, передаёт в `application.QueryWithMetadataFilter`, который делает type-assert на `VectorStoreWithFilters` и вызывает `SearchWithMetadataFilter`; unit-тест с mock-store проверяет, что метод был вызван с правильным фильтром.
- **AC-004** (нет совпадений → пустой результат без ошибки) → pgvector возвращает 0 строк, `TotalFound=0`, `err=nil`; тест проверяет `len(chunks)==0 && err==nil`.
- **AC-005** (in-memory компилируется) → `InMemoryStore` получает `SearchWithMetadataFilter`; `go build ./...` и `go test ./...` без ошибок; тест фильтрации in-memory покрывает базовые случаи.

## Данные и контракты

- Новый domain-тип `MetadataFilter` с полем `Fields map[string]string` — описан в `data-model.md` (DM-001).
- `ErrFilterNotSupported` уже существует в `internal/application` и `pkg/draftrag/errors.go` — новый sentinel в `domain/models.go` добавляется для типизации на domain-уровне; публичный `pkg` реэкспортирует как раньше.
- `domain.Chunk` пополняется полем `Metadata map[string]string` (optional, nil = нет метаданных) — см. DM-001 в data-model.md и DEC-005.
- Никаких изменений в SQL-схеме или migration-файлах.
- Публичный API изменяется аддитивно; `contracts/api.md` создан, так как добавляются два новых метода публичной границы `Pipeline`.

## Стратегия реализации

- **DEC-001** Расширить `VectorStoreWithFilters`, а не создавать новый интерфейс
  Why: отдельный интерфейс `VectorStoreWithMetadata` удвоил бы type-assertion логику в application-слое и усложнил будущую комбинацию фильтров. Существующий паттерн `VectorStoreWithFilters` уже принят в репозитории.
  Tradeoff: любой будущий реализатор `VectorStoreWithFilters` должен реализовать оба метода — `SearchWithFilter` и `SearchWithMetadataFilter`. Это приемлемо пока реализаторов мало.
  Affects: `internal/domain/interfaces.go`, `internal/infrastructure/vectorstore/pgvector.go`, `internal/infrastructure/vectorstore/memory.go`
  Validation: `go build ./...` не компилируется без реализации обоих методов; `var _ domain.VectorStoreWithFilters = (*PGVectorStore)(nil)` уже присутствует как compile-time assert.

- **DEC-002** Фильтрация в pgvector через JSONB `@>` без нового индекса
  Why: колонка `metadata jsonb` уже есть (migration 0002); оператор `@>` работает как с GIN-индексом, так и без него (seq scan). На MVP-объёмах seq scan достаточен; GIN-индекс можно добавить позже отдельной migration.
  Tradeoff: без GIN-индекса поиск с фильтром деградирует на больших коллекциях. Это acceptable для MVP.
  Affects: `internal/infrastructure/vectorstore/pgvector.go`
  Validation: integration-тест проходит; explain-план не входит в scope.

- **DEC-003** `ErrFilterNotSupported` — переиспользовать существующий публичный sentinel
  Why: `pkg/draftrag/errors.go` уже экспортирует `ErrFiltersNotSupported`; дублировать или переименовывать не нужно. Application-слой уже имеет свой `ErrFiltersNotSupported` — паттерн маппинга через `errors.Is` уже реализован в `QueryTopKWithParentIDs`.
  Tradeoff: нет.
  Affects: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
  Validation: unit-тест: вызов `QueryWithMetadataFilter` с бэкендом без `VectorStoreWithFilters` возвращает `ErrFiltersNotSupported`.

- **DEC-005** Добавить `Metadata map[string]string` в `domain.Chunk`
  Why: `Chunk` не имеет поля метаданных, но in-memory `SearchWithMetadataFilter` должен по ним фильтровать. Без этого поля нет данных для фильтрации в памяти. pgvector также должен читать `metadata` из колонки обратно в `Chunk` при scan.
  Tradeoff: изменение `Chunk` затрагивает все места создания `Chunk` в тестах — добавление нового optional-поля (nil = нет метаданных) является non-breaking по семантике, но требует внимания в тестовых fixture'ах.
  Affects: `internal/domain/models.go`, `internal/infrastructure/vectorstore/pgvector.go` (scan), `internal/infrastructure/vectorstore/memory.go`
  Validation: `go build ./...` и существующие тесты проходят без изменений; тест `InMemoryStore.SearchWithMetadataFilter` создаёт `Chunk` с заполненным `Metadata`.

- **DEC-004** `Query.Filter` оставить без изменений
  Why: поле присутствует в domain, но нигде не используется в application или infrastructure-слое. Его удаление — breaking change domain-модели без реальной необходимости. Новый путь через `MetadataFilter` явно разделён.
  Tradeoff: небольшая двусмысленность для будущих контрибьюторов.
  Affects: только комментарий в `models.go` опционально.
  Validation: нет новых тестов на `Query.Filter`; старые тесты продолжают проходить.

## Порядок реализации

1. **Domain first** (блокирует всё остальное): добавить `MetadataFilter` в `models.go` и `SearchWithMetadataFilter` в `VectorStoreWithFilters` в `interfaces.go`.
2. **Infrastructure** (можно после шага 1, оба бэкенда параллельно):
   - pgvector: `SearchWithMetadataFilter` с JSONB `@>`, повторяя структуру `SearchWithFilter`
   - in-memory: `SearchWithMetadataFilter` с итерацией по chunks
3. **Application**: `QueryWithMetadataFilter` и `AnswerWithMetadataFilter` в `pipeline.go` (после шага 1)
4. **Public API**: два метода в `pkg/draftrag/draftrag.go` + переэкспорт `MetadataFilter` (после шага 3)
5. **Тесты**: unit (application + in-memory) и integration (pgvector) — можно писать параллельно с шагами 2–4

## Риски

- **Неполный список реализаторов `VectorStoreWithFilters`**: если в репозитории появятся новые реализаторы до мержа — компилятор поймает.
  Mitigation: compile-time assert `var _ domain.VectorStoreWithFilters = (*X)(nil)` обязателен для каждого реализатора.

- **pgvector runtime-тесты требуют реальной БД**: integration-тест для `SearchWithMetadataFilter` не пройдет без PostgreSQL+pgvector.
  Mitigation: следовать паттерну существующих `pgvector_runtime_test.go` (skip без DSN); unit-тест с mock покрывает application-слой независимо от БД.

## Rollout и compatibility

Специальных rollout-действий не требуется. Изменения аддитивны: новые методы, новый тип, расширение интерфейса. Схема БД не меняется. Пользователи пакета, не вызывающие новые методы, не затрагиваются.

## Проверка

- **AC-001, AC-004**: integration-тест pgvector с реальной БД (`pgvector_runtime_test.go` паттерн) — два класса документов, запрос с фильтром, проверка ID возвращённых чанков и случай нулевого результата.
- **AC-002**: unit-тест или integration-тест: пустой `MetadataFilter{}` даёт те же ID, что и `Search(...)`.
- **AC-003, DEC-003**: unit-тест с mock `VectorStoreWithFilters` — проверяет, что `SearchWithMetadataFilter` вызван с правильным фильтром; тест с non-filter store проверяет `ErrFiltersNotSupported`.
- **AC-005**: `go build ./...` и `go test ./...` без флагов — покрывает компиляцию in-memory store; unit-тест `InMemoryStore.SearchWithMetadataFilter` с несколькими чанками.

## Соответствие конституции

Нет конфликтов. Расширение идёт через существующий интерфейсный механизм (`VectorStoreWithFilters`), domain-слой не импортирует внешние пакеты, все публичные типы получат godoc-комментарии на русском языке, изменение аддитивно и не нарушает semver.
