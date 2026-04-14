# Retry / circuit breaker План

## Phase Contract

Inputs: spec, inspect report, контекст существующих интерфейсов `Embedder`, `LLMProvider`, `Hooks`.
Outputs: plan.md, data-model.md (runtime state только).

## Цель

Создать декораторы `RetryEmbedder` и `RetryLLMProvider` в `internal/infrastructure/resilience/` с retry-логикой (exponential backoff + jitter) и circuit breaker (closed/open/half-open). Обёртки прозрачны для существующего кода и интегрируются с `Hooks` для observability.

## Scope

- `internal/infrastructure/resilience/` — новый пакет для resilience-компонентов
- `RetryEmbedder` — обёртка для `Embedder`
- `RetryLLMProvider` — обёртка для `LLMProvider`
- `CircuitBreaker` — внутренний компонент state machine
- `Backoff` — стратегия задержек
- Интеграция с `domain.Hooks` для событий retry и CB transitions

## Implementation Surfaces

- **NEW** `internal/infrastructure/resilience/retry.go` — базовая retry-логика (Backoff, RetryConfig)
- **NEW** `internal/infrastructure/resilience/circuitbreaker.go` — state machine circuit breaker
- **NEW** `internal/infrastructure/resilience/embedder.go` — `RetryEmbedder` реализация
- **NEW** `internal/infrastructure/resilience/llm.go` — `RetryLLMProvider` реализация
- **NEW** `internal/infrastructure/resilience/errors.go` — классификация retryable ошибок
- **EXISTING** `internal/domain/interfaces.go` — читаем для соответствия контрактам
- **EXISTING** `internal/domain/hooks.go` — интеграция observability

## Влияние на архитектуру

- **Локальное**: новый пакет `resilience` в infrastructure-слое, не затрагивает domain
- **Интеграции**: декораторы оборачивают существующие реализации `Embedder`/`LLMProvider` без их изменения
- **Compatibility**: 100% backward compatible — использование обёрток опционально
- **Rollout**: нет breaking changes, нет миграций

## Acceptance Approach

- **AC-001** -> `RetryEmbedder.Embed()` реализует retry loop с backoff; unit-test с mock embedder возвращает ошибку→успех
- **AC-002** -> `RetryLLMProvider.Generate()` прекращает retry после maxRetries; unit-test проверяет количество вызовов
- **AC-003** -> `CircuitBreaker` считает ошибки в окне, переходит в open; unit-test проверяет переход и fast-fail
- **AC-004** -> `CircuitBreaker` отслеживает timeout, переходит в half-open, пробный запрос; unit-test с mock clock
- **AC-005** -> retry loop проверяет `ctx.Done()` после каждой попытки и между backoff; unit-test с cancelled context
- **AC-006** -> вызовы `Hooks.StageStart/StageEnd` с custom events (retry count, CB state); unit-test с mock hooks

## Данные и контракты

- Нет persisted данных, только runtime state в circuit breaker (счётчики ошибок, timestamps)
- Нет API contracts — внутренние декораторы
- Нет event contracts — hooks вызовы синхронны
- См. `data-model.md` для runtime state структуры

## Стратегия реализации

- **DEC-001** Отдельный пакет `resilience` в infrastructure
  - Why: retry и circuit breaker — cross-cutting concerns, не привязаны к конкретному провайдеру; отдельный пакет позволяет переиспользовать и тестировать изолированно
  - Tradeoff: небольшое дублирование конструкторов для `RetryEmbedder`/`RetryLLMProvider` vs generic-обёртка (Go не поддерживает generics для методов с разными сигнатурами)
  - Affects: `internal/infrastructure/resilience/*`
  - Validation: пакет компилируется, unit-tests проходят без зависимости от конкретных LLM/Embedder провайдеров

- **DEC-002** Circuit breaker state machine в памяти (без persistence)
  - Why: простота, достаточно для single-instance сценария; persistence требует распределённого хранилища и усложняет recovery
  - Tradeoff: при перезапуске приложения CB сбрасывается в closed — допустимо для transient failures
  - Affects: `circuitbreaker.go`
  - Validation: unit-test показывает переходы closed→open→half-open→closed

- **DEC-003** Jitter как random [0, 0.25] от задержки
  - Why: предотвращает thundering herd при массовом retry; 25% — консервативное значение из практики
  - Tradeoff: увеличивает максимальное время retry на 25%
  - Affects: `retry.go` Backoff.CalculateDelay()
  - Validation: unit-test проверяет jitter в диапазоне [delay, delay*1.25]

- **DEC-004** Идентификация retryable ошибок через типизацию
  - Why: Go не имеет стандартного способа определить retryable ошибку; создаём интерфейс `RetryableError` или функцию `IsRetryable(err)`
  - Tradeoff: базовые провайдеры должны возвращать typed errors для точной классификации; fallback на retry всех ошибок кроме context cancellation
  - Affects: `errors.go`, реализации обёрток
  - Validation: unit-test с разными типами ошибок показывает retry/not-retry behavior

## Incremental Delivery

### MVP (Первая ценность)

- `Backoff` с exponential + jitter
- `RetryEmbedder` с базовым retry loop
- Unit-tests для AC-001, AC-002, AC-005

Критерий готовности MVP: тесты проходят, можно обернуть любой Embedder и получить retry.

### Итеративное расширение

1. **Circuit breaker core** — state machine, переходы, окно ошибок
   - Покрывает AC-003, AC-004
2. **RetryLLMProvider** — аналогично Embedder
   - Покрывает AC-002 для LLM
3. **Hooks интеграция** — события retry attempts и CB transitions
   - Покрывает AC-006

## Порядок реализации

1. `errors.go` — классификация retryable errors (DEC-004)
2. `retry.go` — Backoff стратегия (DEC-003)
3. `circuitbreaker.go` — state machine (DEC-002)
4. `embedder.go` — RetryEmbedder с retry + CB
5. `llm.go` — RetryLLMProvider
6. Интеграция Hooks в обёртки (AC-006)

Параллелизм: пункты 4 и 5 можно делать параллельно после готовности 1-3.

## Риски

- **Риск**: Неточная классификация retryable ошибок приводит к retry 4xx или бесконечному retry
  - Mitigation: консервативный fallback — retry только при явно определённых transient errors; context cancellation никогда не retry

- **Риск**: Circuit breaker в open блокирует все запросы, включая healthy
  - Mitigation: настраиваемый timeout восстановления; metrics через hooks для наблюдения

- **Риск**: Накопление горутин при concurrent requests и half-open
  - Mitigation: один пробный запрос в half-open, остальные rejected или queued; mutex на state transition

## Rollout и compatibility

- Специальных rollout-действий не требуется
- Обёртки опциональны — существующий код не изменяется
- Feature не требует feature flags

## Проверка

- Unit-tests для каждого файла пакета `resilience`
- Coverage ≥80% для retry-логики и circuit breaker
- Интеграционный тест: обёртка + mock провайдер
- Проверка AC-006: mock hooks фиксирует вызовы

## Соответствие конституции

- **[Интерфейсная абстракция]** ✓ Обёртки реализуют `Embedder`/`LLMProvider`, не ломают контракты
- **[Чистая архитектура]** ✓ Новый пакет в infrastructure-слое, domain не затронут
- **[Контекстная безопасность]** ✓ Все операции принимают `context.Context`, проверяют cancellation
- **[Тестируемость]** ✓ Mock-реализации для базовых интерфейсов, изолированные unit-tests
- **[Минимальная конфигурация]** ✓ Разумные defaults (maxRetries=3, baseDelay=100ms)

Нет конфликтов с конституцией.
