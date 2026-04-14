# Resilience Public API — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/resilience.go | T1.1, T2.1 |
| docs/advanced.md | T3.1 |

## Фаза 1: Основа

- [x] T1.1 Создать `pkg/draftrag/resilience.go` с `RetryOptions`, `RetryEmbedder`, `RetryLLMProvider`. Touches: pkg/draftrag/resilience.go

## Фаза 2: Основная реализация

- [x] T2.1 Re-export `CircuitState`, `CircuitClosed`, `CircuitOpen`, `CircuitHalfOpen`, `ErrCircuitOpen`, `CircuitBreakerStats`, `IsRetryable`, `WrapRetryable`, `WrapNonRetryable`. Touches: pkg/draftrag/resilience.go

## Фаза 3: Проверка

- [x] T3.1 Задокументировать в `docs/advanced.md`: секция Resilience с RetryOptions таблицей, примерами CB stats и классификации ошибок. Touches: docs/advanced.md
- [x] T3.2 Убедиться что `go build ./...` проходит без ошибок.

## Покрытие критериев приемки

- AC-001 → T1.1
- AC-002 → T1.1
- AC-003 → T2.1
