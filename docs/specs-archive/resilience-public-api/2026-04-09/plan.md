# Resilience Public API — План

## Цель

Создать тонкий публичный фасад над существующим `internal/infrastructure/resilience` без дублирования логики.

## Scope

- `pkg/draftrag/resilience.go` — новый файл

## Стратегия реализации

- DEC-001 Embedding (не делегирование) для RetryEmbedder/RetryLLMProvider
  Why: `type RetryEmbedder struct { *resilience.RetryEmbedder }` автоматически наследует все методы
  Tradeoff: пользователь видит методы внутреннего типа при автодополнении; но это ок для library-level use
  Affects: pkg/draftrag/resilience.go
  Validation: компилируется; методы CircuitBreakerStats, CircuitBreakerState доступны

- DEC-002 Type aliases для re-export констант
  Why: `type CircuitState = resilience.CircuitState` сохраняет identity без wrapper
  Tradeoff: нет
  Affects: pkg/draftrag/resilience.go
  Validation: `errors.Is(err, draftrag.ErrCircuitOpen)` работает

## Порядок реализации

1. Создать `pkg/draftrag/resilience.go`
2. Добавить `RetryOptions` с дефолтами
3. Добавить `RetryEmbedder` и `RetryLLMProvider` через embedding
4. Re-export констант и error-утилит
5. Задокументировать в `docs/advanced.md`

## Риски

- Риск: внутренний API `resilience` изменится — фасад сломается
  Mitigation: `internal/` — часть того же модуля; изменения видны на compile

## Rollout и compatibility

- Additive; нет breaking changes.
