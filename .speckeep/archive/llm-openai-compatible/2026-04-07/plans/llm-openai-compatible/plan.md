# LLMProvider OpenAI-compatible (Responses API) для draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/llm-openai-compatible/spec.md`, `.draftspec/specs/llm-openai-compatible/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно зафиксировать стабильный минимальный parsing contract Responses API без расплывчатости.

## Цель

Добавить реализацию `LLMProvider`, использующую OpenAI-compatible Responses API (`POST /v1/responses`) для синхронной генерации текста. Реализация должна быть:
- доступна пользователю через `pkg/draftrag` (без импорта `internal/...`),
- контекстно-безопасна (ctx cancel/deadline),
- конфигурируема через options (model/temperature/max_tokens/base URL/API key),
- полностью тестируема без внешней сети (через `httptest.Server`),
- безопасна по умолчанию (redaction API key в ошибках).

## Scope

- Infrastructure: HTTP клиент Responses API (request/response JSON) на стандартной библиотеке.
- Public API: options struct + фабрика `NewOpenAICompatibleLLM(opts) LLMProvider`.
- Testing: unit-тесты на `httptest.Server` (success / non-2xx / invalid JSON / ctx cancel / config validation / redaction).

## Implementation Surfaces

- `pkg/draftrag/errors.go` — добавить `ErrInvalidLLMConfig` (sentinel) (T1.1).
- `pkg/draftrag/openai_compatible_llm.go` — публичная фабрика, options, валидация конфигурации и `Generate` (T1.2, T2.1).
- `internal/infrastructure/llm/openai_compatible_responses.go` — реализация `domain.LLMProvider` (HTTP + parsing + ошибки) (T2.2, T2.3).
- `internal/infrastructure/llm/openai_compatible_responses_test.go` — unit-тесты Responses клиента на `httptest.Server` (T3.1).
- `pkg/draftrag/openai_compatible_llm_test.go` — тесты публичного API (errors.Is, фабрика, redaction) (T3.2).

## Влияние на архитектуру

- Clean Architecture сохраняется: интерфейс `LLMProvider` в domain; реализация — infrastructure; публичный доступ — `pkg/draftrag`.
- Зависимости: только стандартная библиотека (http/json); никаких внешних SDK.

## Acceptance Approach

- AC-001 -> фабрика и options в `pkg/draftrag/openai_compatible_llm.go`; compile-time подтверждение + `go doc`. Surfaces: pkg/draftrag/openai_compatible_llm.go.
- AC-002 -> unit-тест на `httptest.Server` возвращает валидный JSON, `Generate` возвращает строку. Surfaces: internal/infrastructure/llm/openai_compatible_responses.go, internal/infrastructure/llm/openai_compatible_responses_test.go.
- AC-003 -> unit-тест отмены ctx возвращает `context.Canceled`/`context.DeadlineExceeded` в пределах 100мс. Surfaces: internal/infrastructure/llm/openai_compatible_responses_test.go.
- AC-004 -> config validation возвращает `errors.Is(err, draftrag.ErrInvalidLLMConfig)`. Surfaces: pkg/draftrag/openai_compatible_llm.go, pkg/draftrag/openai_compatible_llm_test.go, pkg/draftrag/errors.go.
- AC-005 -> redaction: unit-тест проверяет, что `err.Error()` не содержит APIKey даже если body “эхом” содержит ключ. Surfaces: internal/infrastructure/llm/openai_compatible_responses_test.go, pkg/draftrag/openai_compatible_llm_test.go.

## Данные и контракты

### HTTP контракт (Responses API, минимальный v1)

- Endpoint: `POST {BaseURL}/v1/responses`
- Auth: `Authorization: Bearer {APIKey}`
- Request JSON (минимум):
  - `model`: string
  - `input`: массив сообщений, содержащий system и user в одном запросе
  - `temperature`: number (optional)
  - `max_output_tokens`: integer (optional)

### Parsing contract (минимальный и стабильный для тестов)

В v1 реализуем извлечение текста в следующем порядке:
1. если в ответе есть top-level строковое поле `output_text` — используем его;
2. иначе ищем в массиве `output` первый элемент с `type == "message"`, затем в его `content` ищем первый объект с `type == "output_text"` и берём поле `text`.

Если ни одно правило не сработало — ошибка `invalid response`.

### Конфигурация

Конфигурация описана в `data-model.md` (options struct). Persisted состояние отсутствует.

## Стратегия реализации

- DEC-001 Реализация на `net/http` + `encoding/json`
  Why: минимальные зависимости, тестируемость через `httptest.Server`.
  Tradeoff: меньше готовых возможностей (ретраи/observability).
  Affects: internal/infrastructure/llm/openai_compatible_responses.go
  Validation: unit-тесты + отсутствие новых внешних зависимостей.

- DEC-002 Фабрика не возвращает error; ошибки конфигурации возвращаются из `Generate`
  Why: сохраняем симметрию с embedder-openai-compatible (у нас фабрика тоже без error).
  Tradeoff: конфиг-ошибки проявляются при первом вызове.
  Affects: pkg/draftrag/openai_compatible_llm.go
  Validation: тесты AC-004 используют `errors.Is`.

- DEC-003 Sentinel-ошибка конфигурации: `draftrag.ErrInvalidLLMConfig`
  Why: детерминированные проверки в клиентском коде (RQ-007).
  Tradeoff: расширение набора публичных ошибок.
  Affects: pkg/draftrag/errors.go
  Validation: unit-тесты AC-004.

- DEC-004 Redaction секретов
  Why: APIKey не должен попадать в ошибки (RQ-008).
  Tradeoff: чуть меньше диагностической информации.
  Affects: internal/infrastructure/llm/openai_compatible_responses.go
  Validation: unit-тест AC-005.

- DEC-005 Default parameters
  Why: минимальная конфигурация, но контроль параметров генерации нужен пользователю.
  Tradeoff: больше options полей и валидации.
  Affects: pkg/draftrag/openai_compatible_llm.go
  Validation: unit-тесты на валидацию `temperature`/`max_output_tokens`.

## Incremental Delivery

### MVP (Первая ценность)

- `Generate` отправляет request и парсит `output_text`.
- Валидация конфигурации и sentinel-ошибка.
- Тесты AC-001..AC-004.

### Итеративное расширение

- Поддержка fallback-парсинга через `output[].content[]` (второе правило parsing contract) + тесты.
- Дополнительные поля options (например, `TopP`) только если появится реальная необходимость.

## Порядок реализации

1. Публичные ошибки/опции и фабрика в `pkg/draftrag`.
2. Infrastructure-клиент Responses API + parsing contract.
3. Unit-тесты на `httptest.Server` (success/error/cancel/redaction).

## Риски

- Риск 1: “OpenAI-compatible” вариативен по Responses payload.
  Mitigation: фиксируем parsing contract и тестируем оба пути (`output_text` и `output[].content[]`).
- Риск 2: Некорректные параметры генерации могут приводить к трудно-диагностируемым ошибкам API.
  Mitigation: строгая локальная валидация `temperature`/`max_output_tokens`.

## Rollout и compatibility

- Нет rollout шагов (библиотека).
- Публичный API добавляется аддитивно.

## Проверка

- `go test ./...` проходит без внешней сети.
- `go doc github.com/bzdvdn/draftrag/pkg/draftrag.NewOpenAICompatibleLLM` показывает русскую документацию.

## Соответствие конституции

- нет конфликтов: интерфейсы сохраняются, зависимости минимальны, контекст соблюдается, тестируемость обеспечена.
