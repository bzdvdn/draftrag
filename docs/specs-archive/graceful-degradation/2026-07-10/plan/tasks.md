# Graceful Degradation Задачи

## Phase Contract

Inputs: plan, data-model (no-change), spec.
Outputs: упорядоченные исполнимые задачи с покрытием всех 9 AC.
Stop if: нет — tasks дедуцируются из plan.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/resilience/fallback.go (new) | T1.1 |
| internal/infrastructure/resilience/fallback_llm.go (new) | T2.1 |
| internal/infrastructure/resilience/fallback_streaming.go (new) | T3.1 |
| internal/infrastructure/resilience/fallback_usage.go (new) | T3.2 |
| internal/infrastructure/resilience/fallback_llm_test.go (new) | T2.2, T3.3, T3.4, T3.5, T5.1 |
| pkg/draftrag/fallback.go (new) | T4.1 |

## Implementation Context

- Цель MVP: `FallbackLLMProvider` с Generate + Health для 2+ провайдеров.
- Инварианты/семантика:
  - Fallback только на retryable-ошибках (AC-002). Non-retryable → немедленный возврат.
  - Health без fallback — только первый провайдер (AC-004).
  - Все провайдеры отказали → aggregate-ошибка `ErrAllProvidersFailed` (AC-003).
  - `FallbackStats` через `sync/atomic.Int64` — thread-safe (DEC-001).
- Ошибки/коды:
  - `ErrAllProvidersFailed` — sentinel для aggregate-ошибки (wrap последней ошибки).
  - Контекстные ошибки (ctx.Err()) всегда non-retryable.
- Контракты/протокол:
  - `FallbackLLMProvider` реализует `domain.LLMProvider`.
  - `FallbackStreamingLLMProvider` реализует `domain.StreamingLLMProvider`.
  - `FallbackUsageAwareLLMProvider` реализует `domain.UsageAwareLLMProvider`.
  - Публичные конструкторы в `pkg/draftrag/fallback.go`.
- Границы scope:
  - Не меняем `RetryLLMProvider`/`RetryEmbedder`.
  - Не трогаем `internal/domain/interfaces.go`.
- Proof signals:
  - Юнит-тест: primary возвращает retryable-ошибку, secondary отвечает успехом.
  - `go test -race ./internal/infrastructure/resilience/...` без ошибок.
  - `go vet ./pkg/draftrag/...` без предупреждений.
- References: DEC-001, DEC-002, DEC-003, DEC-004, DEC-005; RQ-001–RQ-012.

## Фаза 1: Основа

Цель: общие типы (FallbackStats, ErrAllProvidersFailed) — prerequisite для всех fallback-обёрток.

- [x] T1.1 Создать `internal/infrastructure/resilience/fallback.go` с `FallbackStats` (TotalCalls, PrimaryFailures, FallbackCount, LastError — через `sync/atomic.Int64`), `ErrAllProvidersFailed` (sentinel с Unwrap к последней ошибке), и helper `aggregateError` для chain fallback. Touches: internal/infrastructure/resilience/fallback.go

## Фаза 2: MVP Slice

Цель: FallbackLLMProvider (Generate + Health + Stats + Hooks) — покрывает AC-001–005, AC-009.

- [x] T2.1 Реализовать `FallbackLLMProvider` в `internal/infrastructure/resilience/fallback_llm.go`. Конструктор `NewFallbackLLM(providers []domain.LLMProvider, logger domain.Logger, hooks domain.Hooks)`. Генератор: цикл по провайдерам, при retryable-ошибке — fallback к следующему, логирование, обновление Stats. Non-retryable → immediate return. Все исчерпаны → aggregate-ошибка. `Health(ctx)` — только на providers[0] без fallback. `Stats()` возвращает FallbackStats. Touches: internal/infrastructure/resilience/fallback_llm.go
- [x] T2.2 Написать юнит-тесты для FallbackLLMProvider:
- [x] T3.1 Реализовать `FallbackStreamingLLMProvider` в `internal/infrastructure/resilience/fallback_streaming.go`. `GenerateStream` пробует провайдеры по очереди. При ошибке в потоке (канал закрывается после ошибки) — пытается следующий провайдер. Гарантированный drain канала в defer. Touches: internal/infrastructure/resilience/fallback_streaming.go
- [x] T3.2 Реализовать `FallbackUsageAwareLLMProvider` в `internal/infrastructure/resilience/fallback_usage.go`. `GenerateWithUsage` — тот же цикл fallback, но каждый провайдер — `UsageAwareLLMProvider`. Touches: internal/infrastructure/resilience/fallback_usage.go
- [x] T3.3 Тесты FallbackStreamingLLMProvider: AC-006 — primary канал с ошибкой, secondary отдаёт токен. Touches: internal/infrastructure/resilience/fallback_llm_test.go
- [x] T3.4 Тесты FallbackUsageAwareLLMProvider: AC-007 — primary retryable ошибка, secondary возвращает успех с TokenUsage. Touches: internal/infrastructure/resilience/fallback_llm_test.go
- [x] T3.5 Тест AC-008 на FallbackStreamingLLMProvider и FallbackUsageAwareLLMProvider: Stats корректно считает по всем обёрткам. Touches: internal/infrastructure/resilience/fallback_llm_test.go
- [x] T4.1 Создать `pkg/draftrag/fallback.go` с публичными конструкторами `NewFallbackLLMProvider`, `NewFallbackStreamingLLMProvider`, `NewFallbackUsageAwareLLMProvider`, re-export `FallbackStats`, `ErrAllProvidersFailed`. Touches: pkg/draftrag/fallback.go
- [x] T5.1 Запустить `go vet ./...`, `golangci-lint run`, `go test -race ./...` — все проходят. Touches: (все файлы фазы)
- [x] T5.2 Проставить `@sk-task` trace-маркеры над owning declarations во всех созданных файлах. Touches: internal/infrastructure/resilience/fallback.go, fallback_llm.go, fallback_streaming.go, fallback_usage.go, pkg/draftrag/fallback.go

## Покрытие критериев приемки

- AC-001 → T2.1, T2.2
- AC-002 → T2.1, T2.2
- AC-003 → T2.1, T2.2
- AC-004 → T2.1, T2.2
- AC-005 → T2.1, T2.2
- AC-006 → T3.1, T3.3
- AC-007 → T3.2, T3.4
- AC-008 → T2.1, T2.2, T3.5
- AC-009 → T2.1, T2.2