# Retry / circuit breaker Задачи

## Phase Contract

Inputs: plan.md, data-model.md, summary.md.
Outputs: исполнимые задачи с покрытием всех AC.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/resilience/errors.go | T1.1 |
| internal/infrastructure/resilience/retry.go | T1.2, T3.1 |
| internal/infrastructure/resilience/circuitbreaker.go | T2.1, T3.2 |
| internal/infrastructure/resilience/embedder.go | T2.2, T3.3 |
| internal/infrastructure/resilience/llm.go | T2.3, T3.4 |
| internal/infrastructure/resilience/hooks.go | T2.4 |

## Фаза 1: Базовые компоненты

Цель: подготовить классификацию ошибок и backoff-стратегию для retry-логики.

- [x] T1.1 Реализовать классификацию retryable ошибок — `errors.go` с интерфейсом `RetryableError` и функцией `IsRetryable(err)` — DEC-004
  - Touches: internal/infrastructure/resilience/errors.go

- [x] T1.2 Реализовать exponential backoff с jitter — `retry.go` с типом `Backoff` и методом `CalculateDelay(attempt int) time.Duration` — DEC-003
  - Touches: internal/infrastructure/resilience/retry.go

## Фаза 2: Основная реализация

Цель: реализовать circuit breaker state machine и обёртки для Embedder/LLM с интеграцией hooks.

- [x] T2.1 Реализовать circuit breaker state machine — `circuitbreaker.go` с состояниями closed/open/half-open, переходами и thread-safe доступом — AC-003, AC-004, DEC-002
  - Touches: internal/infrastructure/resilience/circuitbreaker.go

- [x] T2.2 Реализовать `RetryEmbedder` — `embedder.go` с retry loop, backoff, CB integration и context cancellation — AC-001, AC-005
  - Touches: internal/infrastructure/resilience/embedder.go

- [x] T2.3 Реализовать `RetryLLMProvider` — `llm.go` с retry loop, backoff, CB integration и context cancellation — AC-002, AC-005
  - Touches: internal/infrastructure/resilience/llm.go

- [x] T2.4 Реализовать интеграцию с `Hooks` — события retry attempts и CB transitions — AC-006, RQ-008
  - Touches: internal/infrastructure/resilience/embedder.go, internal/infrastructure/resilience/llm.go

## Фаза 3: Тестирование и покрытие

Цель: доказать корректность через unit-tests с mock провайдерами и hooks.

- [x] T3.1 Добавить unit-tests для `Backoff` — проверка exponential + jitter в диапазоне [delay, delay*1.25] — DEC-003
  - Touches: internal/infrastructure/resilience/retry_test.go

- [x] T3.2 Добавить unit-tests для `CircuitBreaker` — проверка переходов closed→open→half-open→closed — AC-003, AC-004
  - Touches: internal/infrastructure/resilience/circuitbreaker_test.go

- [x] T3.3 Добавить unit-tests для `RetryEmbedder` — retry успех, исчерпание попыток, context cancellation — AC-001, AC-005
  - Touches: internal/infrastructure/resilience/embedder_test.go

- [x] T3.4 Добавить unit-tests для `RetryLLMProvider` — retry исчерпание, context cancellation — AC-002, AC-005
  - Touches: internal/infrastructure/resilience/llm_test.go

- [x] T3.5 Добавить unit-tests для hooks интеграции — mock hooks фиксируют retry count и CB state transitions — AC-006
  - Touches: internal/infrastructure/resilience/embedder_test.go, internal/infrastructure/resilience/llm_test.go

- [x] T3.6 Проверить coverage ≥80% для пакета `resilience` — go test -cover выводит 90.8%
  - Touches: internal/infrastructure/resilience/

## Покрытие критериев приемки

| AC | Покрытие задачами |
|----|-------------------|
| AC-001 | T2.2, T3.3 |
| AC-002 | T2.3, T3.4 |
| AC-003 | T2.1, T3.2 |
| AC-004 | T2.1, T3.2 |
| AC-005 | T2.2, T2.3, T3.3, T3.4 |
| AC-006 | T2.4, T3.5 |

## Заметки

- Фаза 1 и Фаза 2 можно распараллелить после готовности T1.1, T1.2
- Все задачи покрыты AC, каждый AC покрыт хотя бы одной задачей
- T3.6 — валидационная задача, не блокирует предыдущие
