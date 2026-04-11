# Embedder OpenAI-compatible для draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/embedder-openai-compatible/spec.md`, `.draftspec/specs/embedder-openai-compatible/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно зафиксировать минимальный JSON контракт embeddings API без расплывчатых допущений.

## Цель

Добавить реализацию `Embedder`, работающую с OpenAI-compatible embeddings endpoint через стандартный `net/http`, доступную пользователю через публичную фабрику в `pkg/draftrag`. Реализация должна:
- уважать `context.Context` (отмена/таймаут),
- валидировать конфигурацию и возвращать детерминированную sentinel-ошибку через `errors.Is`,
- быть полностью тестируемой без внешней сети (через httptest.Server).

## Scope

- Infrastructure: HTTP клиент embeddings (`POST /v1/embeddings`) с минимальным контрактом запроса/ответа.
- Public API: options + `NewOpenAICompatibleEmbedder(opts) Embedder` в `pkg/draftrag`.
- Testing: unit-тесты на успех/ошибки/невалидный JSON/отмена контекста + тест full-cycle через `Pipeline` с in-memory store.
- Out: ретраи/backoff, batch embeddings как публичная API, streaming, LLM provider.

## Implementation Surfaces

- `pkg/draftrag/errors.go` — публичные sentinel-ошибки конфигурации (T1.1).
- `pkg/draftrag/openai_compatible_embedder.go` — публичная фабрика + options + метод Embed (T1.2, T2.1).
- `internal/infrastructure/embedder/openai_compatible.go` — HTTP реализация embedder (T2.2, T2.3).
- `internal/infrastructure/embedder/openai_compatible_test.go` — unit-тесты на httptest.Server (T3.1).
- `pkg/draftrag/openai_compatible_embedder_test.go` — тесты публичного API и e2e через Pipeline (T3.2).

## Влияние на архитектуру

- Clean Architecture сохраняется: интерфейс `Embedder` остаётся в domain; HTTP реализация — infrastructure; публичная точка входа — `pkg/draftrag`.
- Внешние зависимости: предпочтительно только стандартная библиотека (JSON/HTTP). Дополнительные SDK не добавляются.

## Acceptance Approach

- AC-001 -> фабрика и options в `pkg/draftrag/openai_compatible_embedder.go`; компиляция и `go doc` подтверждают доступность. Surfaces: pkg/draftrag/openai_compatible_embedder.go.
- AC-002 -> unit-тест на httptest.Server возвращает валидный JSON и Embed выдаёт `[]float64`. Surfaces: internal/infrastructure/embedder/openai_compatible.go, internal/infrastructure/embedder/openai_compatible_test.go.
- AC-003 -> unit-тест отмены `ctx` (deadline/cancel) возвращает `context.Canceled`/`context.DeadlineExceeded` в пределах 100мс. Surfaces: internal/infrastructure/embedder/openai_compatible_test.go.
- AC-004 -> тесты конфигурации: пустой BaseURL/APIKey/Model возвращают `errors.Is(err, draftrag.ErrInvalidEmbedderConfig)`. Surfaces: pkg/draftrag/openai_compatible_embedder.go, pkg/draftrag/openai_compatible_embedder_test.go.
- AC-005 -> e2e тест Pipeline Index + QueryTopK использует embedder на httptest.Server и возвращает результаты. Surfaces: pkg/draftrag/openai_compatible_embedder_test.go.

## Данные и контракты

### HTTP контракт (минимальный OpenAI-compatible)

- Endpoint: `POST {BaseURL}/v1/embeddings`
- Auth: `Authorization: Bearer {APIKey}`
- Request JSON:
  - `model`: string
  - `input`: string (v1 поддерживает только строковый input)
- Response JSON (минимум, который парсим):
  - `data`: массив, берём `data[0].embedding` как `[]float64`
- Ошибки: non-2xx -> ошибка, включающая status code и ограниченный (обрезанный) фрагмент body без секретов.

### Конфигурация / “data model”

- Конфигурация хранится в options struct (см. `data-model.md`); persisted state отсутствует.

## Стратегия реализации

- DEC-001 Реализация на стандартной библиотеке `net/http` + `encoding/json`
  Why: минимальные зависимости, хорошая тестируемость через httptest.Server.
  Tradeoff: меньше готовых features (ретраи/observability) чем в SDK.
  Affects: internal/infrastructure/embedder/openai_compatible.go
  Validation: unit-тесты и отсутствие внешних зависимостей в `go.mod`.

- DEC-002 Фабрика не возвращает error; ошибки конфигурации возвращаются из `Embed`
  Why: сохраняем сигнатуру из spec (`New...(...) Embedder`) и всё равно выполняем RQ-007 через sentinel-ошибку.
  Tradeoff: ошибки конфигурации проявляются при первом вызове `Embed`.
  Affects: pkg/draftrag/openai_compatible_embedder.go
  Validation: тесты AC-004 используют `errors.Is`.

- DEC-003 Sentinel-ошибка конфигурации: `draftrag.ErrInvalidEmbedderConfig`
  Why: детерминированная проверка ошибок клиентом и в тестах (RQ-007).
  Tradeoff: расширение набора ошибок публичного пакета.
  Affects: pkg/draftrag/errors.go
  Validation: `errors.Is(err, draftrag.ErrInvalidEmbedderConfig)` в тестах.

- DEC-004 Redaction секретов
  Why: APIKey не должен попадать в ошибки/логи (security-by-default).
  Tradeoff: меньше диагностической информации, но безопаснее.
  Affects: internal/infrastructure/embedder/openai_compatible.go
  Validation: тест проверяет, что ошибочный response не содержит API key.

- DEC-005 HTTP клиент/таймаут
  Why: пользователю нужен контроль над timeout и возможностью передать свой `*http.Client`.
  Tradeoff: усложнение options.
  Affects: pkg/draftrag/openai_compatible_embedder.go
  Validation: unit-тесты используют кастомный client/server.

## Incremental Delivery

### MVP (Первая ценность)

- Embed работает на httptest.Server, парсит JSON и возвращает embedding.
- Валидация конфигурации и sentinel-ошибка.
- Тесты AC-001..AC-004.

### Итеративное расширение

- Тест full-cycle с `Pipeline` (AC-005).
- Поддержка `input: []string` (batch) как внутренняя оптимизация (без изменения публичной API) — только если появится реальная потребность.

## Порядок реализации

1. Добавить публичные ошибки/опции и фабрику в `pkg/draftrag`.
2. Реализовать инфраструктурный embedder (HTTP + JSON + ctx).
3. Написать unit-тесты на httptest.Server.
4. Добавить e2e тест с `Pipeline`.

## Риски

- Риск 1: “OpenAI-compatible” вариативен у разных провайдеров.
  Mitigation: держать минимальный контракт (`/v1/embeddings`, `model`, `input`, `data[0].embedding`) и явно документировать, что парсим.
- Риск 2: таймауты/отмена могут вести себя по-разному на разных транспортных слоях.
  Mitigation: использовать `NewRequestWithContext`, тестировать cancel/deadline.

## Rollout и compatibility

- Нет rollout шагов (библиотека).
- Публичный API добавляется аддитивно (без breaking changes).

## Проверка

- `go test ./...` без внешней сети проходит.
- `go doc github.com/bzdvdn/draftrag/pkg/draftrag.NewOpenAICompatibleEmbedder` показывает русскую документацию.
- Тесты AC-002..AC-005 опираются на httptest.Server.

## Соответствие конституции

- нет конфликтов: чистая архитектура сохранена; контекстная безопасность обеспечена; конфигурация опциональна и задаётся через options.
