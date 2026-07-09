# Graceful Degradation План

## Phase Contract

Inputs: spec, inspect (pass), resilience layer surface.
Outputs: plan, data-model (no-change).
Stop if: spec расплывчата — нет, inspect pass.

## Цель

Реализовать цепочку fallback-обёрток над LLMProvider в `internal/infrastructure/resilience/` с публичным re-export в `pkg/draftrag/`. Без изменений существующих `RetryLLMProvider`/`RetryEmbedder` и domain-интерфейсов.

## MVP Slice

`FallbackLLMProvider` (Generate + Health) для 2+ провайдеров. AC-001, AC-002, AC-003, AC-004, AC-005, AC-009.

## First Validation Path

Юнит-тест: mock провайдер — primary всегда retryable-ошибка, secondary — успех. Вызов Generate возвращает ответ secondary. `Stats().PrimaryFailures == 1`.

## Scope

- Новый файл `internal/infrastructure/resilience/fallback_llm.go` — `FallbackLLMProvider`.
- Новый файл `internal/infrastructure/resilience/fallback_llm_test.go` — тесты.
- Новый файл `internal/infrastructure/resilience/fallback_streaming.go` — `FallbackStreamingLLMProvider`.
- Новый файл `internal/infrastructure/resilience/fallback_usage.go` — `FallbackUsageAwareLLMProvider`.
- Новый файл `internal/infrastructure/resilience/fallback.go` — общие типы (`FallbackStats`, `ErrAllProvidersFailed`).
- Расширение `pkg/draftrag/fallback.go` — публичный re-export: `NewFallbackLLMProvider`, `NewFallbackStreamingLLMProvider`, `NewFallbackUsageAwareLLMProvider`, `FallbackStats`, `ErrAllProvidersFailed`.
- `internal/infrastructure/resilience/doc.go` — может быть обновлён.
- `RetryLLMProvider`, `RetryEmbedder`, `internal/domain/interfaces.go` — **не меняются**.

## Performance Budget

- `none` — фича добавляет только последовательный вызов ~N провайдеров без аллокаций на hot path (кроме логирования при fallback).

## Implementation Surfaces

- `internal/infrastructure/resilience/fallback.go` (new) — общие типы: `FallbackStats`, `ErrAllProvidersFailed`, `IsRetryable` re-use.
- `internal/infrastructure/resilience/fallback_llm.go` (new) — `FallbackLLMProvider`.
- `internal/infrastructure/resilience/fallback_streaming.go` (new) — `FallbackStreamingLLMProvider`.
- `internal/infrastructure/resilience/fallback_usage.go` (new) — `FallbackUsageAwareLLMProvider`.
- `internal/infrastructure/resilience/fallback_llm_test.go` (new) — тесты всех трёх Fallback-типов.
- `pkg/draftrag/fallback.go` (new) — публичные конструкторы и re-export.

Почему не в одном файле: три разных контракта (LLMProvider, StreamingLLMProvider, UsageAwareLLMProvider) — чище разделить.

## Bootstrapping Surfaces

`internal/infrastructure/resilience/fallback.go` — должен существовать до fallback_*.go (общий тип FallbackStats).

## Влияние на архитектуру

- Новый паттерн обёртки, layer'ующийся поверх retry.
- Никаких изменений domain-интерфейсов и application-слоя.
- Pipeline не требует изменений — пользователь оборачивает провайдеры до передачи в `NewPipeline`.

## Acceptance Approach

- AC-001: testify/suite с mock провайдерами. Validate: ответ == secondary; `Stats().FallbackCount == 1`.
- AC-002: primary возвращает `WrapNonRetryable`. Validate: ответ содержит ошибку primary; `Stats().FallbackCount == 0`.
- AC-003: оба провайдера возвращают retryable-ошибку. Validate: `errors.Is(err, ErrAllProvidersFailed)`.
- AC-004: primary в CB open. Validate: `Health()` возвращает `ErrCircuitOpen`; secondary не вызван.
- AC-005: mock Hooks. Validate: `OnError` вызван ровно 1 раз.
- AC-006: streaming mock — канал primary закрывается с ошибкой. Validate: вторичный канал отдаёт токен.
- AC-007: usage-aware mock. Validate: TokenUsage от secondary.
- AC-008: 3 вызова (2 неудачи primary). Validate: `Stats() = {TotalCalls:3, PrimaryFailures:2, FallbackCount:2}`.
- AC-009: `NewFallbackLLMProvider()` с nil/empty. Validate: err != nil.

## Данные и контракты

- `data-model.md`: no-change (фича не добавляет persisted entities).
- Новые публичные типы: `FallbackStats` (struct), `ErrAllProvidersFailed` (sentinel error).
- Никаких API/event contract изменений.

## Стратегия реализации

### DEC-001 Один общий FallbackStats через sync/atomic

- Why: thread-safety без блокировок. FallbackLLMProvider может вызываться из нескольких goroutine (Pipeline использует worker pool).
- Tradeoff: `sync/atomic.Int64` требует Go 1.19+ (Go 1.21 ok). Счётчики только монотонные — нет сброса.
- Affects: `internal/infrastructure/resilience/fallback.go`
- Validation: `go vet` + race detector в тестах.

### DEC-002 FallbackStreamingLLMProvider — отдельная обёртка, не type assertion

- Why: контракт GenerateStream (канал) принципиально другой — логика ошибки в потоке сложнее. Type assertion сделал бы FallbackLLMProvider монолитным.
- Tradeoff: дублирование цикла fallback (3×). Но код тривиален — <30 строк на обёртку.
- Affects: `internal/infrastructure/resilience/fallback_llm.go`, `fallback_streaming.go`, `fallback_usage.go`
- Validation: каждый тип покрыт отдельным AC.

### DEC-003 Fallback только на retryable-ошибках

- Why: non-retryable (bad request, auth) — ошибка пользователя/конфигурации. Fallback на них маскирует баги.
- Tradeoff: пользователь должен явно помечать ошибки через `WrapNonRetryable`.
- Affects: все Fallback-типы.
- Validation: AC-002.

### DEC-004 Health без fallback, только первый провайдер

- Why: health-check должен отражать реальную доступность первого (целевого) провайдера. Fallback скрыл бы проблему.
- Tradeoff: при outage Primary health будет false даже если Secondary жив.
- Affects: все Fallback-типы.
- Validation: AC-004.

### DEC-005: Публичный re-export в `pkg/draftrag/fallback.go`, не в `resilience.go`

- Why: `resilience.go` уже содержит `RetryLLMProvider`/`RetryEmbedder`. Fallback — отдельная концепция. Чище в отдельном файле.
- Tradeoff: дополнительный файл.
- Validation: код компилируется, `go doc` видит конструкторы.

## Incremental Delivery

### MVP (Первая ценность)

- `FallbackLLMProvider` (Generate + Health + Stats + Hooks)
- Покрывает AC-001, AC-002, AC-003, AC-004, AC-005, AC-009.
- Проверка: юнит-тест с 2 mock провайдерами.

### Итеративное расширение

- Шаг 2: `FallbackStreamingLLMProvider` — AC-006.
- Шаг 3: `FallbackUsageAwareLLMProvider` — AC-007.
- Шаг 4: публичный re-export — AC-008 (Stats уже на месте).

## Порядок реализации

1. `fallback.go` (FallbackStats, ErrAllProvidersFailed) — без него не скомпилируются остальные.
2. `fallback_llm.go` + тесты — MVP, независимо валидируется.
3. `fallback_streaming.go` + тесты — может параллельно с #4.
4. `fallback_usage.go` + тесты.
5. `pkg/draftrag/fallback.go` — после всех internal-файлов.
6. `go vet`, `golangci-lint`, race detector.

## Риски

- Риск: FallbackStreamingLLMProvider — детекция "ошибки в канале" vs "закрытие без ошибки". Mitigation: проверять второй сигнал (ok == false + получена ошибка перед закрытием). Для первого прохода достаточно упрощённой модели: если канал закрывается, а ошибка была залогирована (провайдер сообщил об ошибке), то fallback.
- Риск: goroutine leak в FallbackStreamingLLMProvider при панике/досрочном выходе. Mitigation: гарантированное потребление канала через `for range` с drain в defer при необходимости.
- Риск: Race в FallbackStats при параллельных Generate. Mitigation: `sync/atomic.Int64`.

## Rollout and compatibility

- Полностью additive — не требует флагов или миграций.
- Пользователь выбирает, использовать ли Fallback-обёртки.

## Проверка

- Юнит-тесты: 9 AC × 1+ тест = 10+ тестов.
- `go vet ./...`
- `golangci-lint run`
- `go test -race ./...`

## Соответствие конституции

- нет конфликтов