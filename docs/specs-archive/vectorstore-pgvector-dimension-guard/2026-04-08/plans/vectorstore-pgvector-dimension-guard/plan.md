# VectorStore pgvector: guardrails для размерности эмбеддингов (v1) — План

## Phase Contract

Inputs: `.speckeep/specs/vectorstore-pgvector-dimension-guard/spec.md`, `.speckeep/specs/vectorstore-pgvector-dimension-guard/inspect.md`, `.speckeep/constitution.md`.
Outputs: `.speckeep/plans/vectorstore-pgvector-dimension-guard/plan.md`, `.speckeep/plans/vectorstore-pgvector-dimension-guard/data-model.md`.
Stop if: невозможно сделать ошибку типизированной и доступной пользователю без нарушения Clean Architecture.

## Цель

Сделать несоответствие размерности embedding-векторов явной, типизированной ошибкой, которую удобно проверять через `errors.Is`, и гарантировать, что `Upsert` и `Search` отрабатывают ранней валидацией до SQL-запросов.

## Scope

- В scope:
  - классификатор ошибки “embedding dimension mismatch” (`errors.Is`).
  - обёртка этой ошибки в pgvector store (`Upsert`, `Search`, `SearchWithFilter`) при `len(vec) != EmbeddingDimension`.
  - unit-тесты без БД/сети, проверяющие `errors.Is` и отсутствие обращения к БД на mismatch.
- Вне scope:
  - хранение dimension в БД и валидация “на старте”.
  - миграции данных при смене dimension.

## Implementation Surfaces

- `internal/domain/models.go`:
  - добавить sentinel-ошибку `ErrEmbeddingDimensionMismatch` как классификатор.
- `pkg/draftrag/errors.go`:
  - экспортировать `ErrEmbeddingDimensionMismatch` (как re-export на доменную sentinel), чтобы пользователь мог `errors.Is(err, draftrag.ErrEmbeddingDimensionMismatch)`.
- `internal/infrastructure/vectorstore/pgvector.go`:
  - обновить `validateEmbedding` так, чтобы dimension mismatch возвращался как wrap на sentinel (`%w`) с деталями `got/want`.
- `pkg/draftrag/pgvector.go` (если нужно):
  - уточнить в комментариях, что `EmbeddingDimension` — это “Dimension” из требований, и что mismatch классифицируется через `ErrEmbeddingDimensionMismatch`.
- Тесты:
  - `pkg/draftrag/pgvector_dimension_guard_test.go` — unit-тесты без реальной БД (через test driver), покрывающие mismatch на `Upsert` и `Search`.

## Влияние на архитектуру

- Domain расширяется только sentinel-ошибкой (без внешних зависимостей).
- Публичный API не ломается: новый экспортируемый `ErrEmbeddingDimensionMismatch` аддитивен.
- Infrastructure остаётся зависимой только от domain + stdlib.

## Acceptance Approach

- AC-001 -> unit-тесты: mismatch в `Upsert`/`Search` возвращает ошибку, сравнимую через `errors.Is(err, draftrag.ErrEmbeddingDimensionMismatch)`.
- AC-002 -> базовые существующие тесты `go test ./...` проходят; плюс отдельный тест, что на корректной размерности не происходит раннего отказа (happy path не ломается).

## Данные и контракты

- Конфигурационный контракт:
  - `PGVectorOptions.EmbeddingDimension` является “Dimension” в v1.
- Ошибочный контракт:
  - при mismatch возвращается error, который удовлетворяет `errors.Is(err, draftrag.ErrEmbeddingDimensionMismatch) == true`;
  - строковое сообщение может включать `got/want`, но это не является стабильным API.

## Стратегия реализации

- DEC-001 “Sentinel error в domain + re-export в pkg”
  Why: infrastructure не может зависеть от `pkg/`, а пользователю нужен стабильный классификатор ошибки.
  Tradeoff: добавляется ещё одна публичная ошибка в `pkg/draftrag`.
  Affects: `internal/domain/models.go`, `pkg/draftrag/errors.go`.
  Validation: unit-тесты `errors.Is` (AC-001).

- DEC-002 “Wrap mismatch через `%w` с деталями”
  Why: `errors.Is` работает, при этом сохраняются диагностические детали `got/want`.
  Tradeoff: текст ошибки не фиксируется как контракт.
  Affects: `internal/infrastructure/vectorstore/pgvector.go`.
  Validation: unit-тесты проверяют `errors.Is` и отсутствие обращения к БД при mismatch (AC-001).

## Incremental Delivery

### MVP (Первая ценность)

- Sentinel error в domain + re-export в pkg.
- Wrap mismatch в pgvector store.
- Unit-тесты mismatch для `Upsert` и `Search`.

Критерий готовности: AC-001.

### Итеративное расширение

- Расширить тесты на `SearchWithFilter`.
- (Опционально) добавить отдельные классификаторы для `nil embedding`/`non-finite` значений, если появится потребность.

## Порядок реализации

1. Ввести sentinel в domain и экспорт в pkg.
2. Перевести `validateEmbedding` на wrap `%w`.
3. Добавить unit-тесты (без БД/сети) на `Upsert` и `Search`.
4. Прогнать `go test ./...`.

## Риски

- Риск: “ложная совместимость” — если пользователи сравнивали текст ошибки.
  Mitigation: документировать, что стабильным является только `errors.Is` по sentinel.

## Rollout и compatibility

- Rollout не требуется.
- Compatibility: аддитивные изменения; поведение “ошибка при mismatch” уже было, меняется только классификация.

## Проверка

- `go test ./...`
- Проверка `errors.Is` для mismatch на `Upsert` и `Search` в unit-тестах.

## Соответствие конституции

- Нет конфликтов: интерфейсы остаются чистыми, зависимости не утяжеляются, тестируемость повышается.

