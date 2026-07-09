# Resilience Public API (Retry + Circuit Breaker)

## Scope Snapshot

- In scope: экспорт `RetryEmbedder`, `RetryLLMProvider`, `CircuitBreakerStats` и сопутствующих типов из `internal/infrastructure/resilience` в `pkg/draftrag`.
- Out of scope: новая resilience-логика, изменение внутренних алгоритмов retry/CB.

## Цель

Разработчики, использующие draftRAG как библиотеку, получают защиту от transient failures и cascade failures через retry с exponential backoff и circuit breaker — без импорта `internal/` пакетов.

## Основной сценарий

1. Пользователь создаёт `draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{MaxRetries: 5})`.
2. При ошибках базового embedder'а — автоматические retry с jitter.
3. После `CBThreshold` последовательных ошибок — circuit breaker открывается, запросы быстро фейлятся с `ErrCircuitOpen`.
4. Через `CBTimeout` — half-open → probe → recovery.

## Scope

- `pkg/draftrag/resilience.go` — новый файл с публичными обёртками
- Нетронутым остаётся `internal/infrastructure/resilience/`

## Контекст

- `internal/infrastructure/resilience` уже реализует retry + CB; нужен только публичный фасад.
- Go запрещает прямой импорт `internal/` из вне модуля; `pkg/draftrag` — правильное место для экспорта.
- Embedder и LLM — два основных IO-bound компонента, требующих resilience.

## Требования

- **RQ-001** `RetryOptions` struct с полями: MaxRetries, BaseDelay, MaxDelay, Multiplier, JitterFactor, CBThreshold, CBTimeout; разумные дефолты при zero value.
- **RQ-002** `RetryEmbedder` — обёртка над `Embedder`, реализует `Embedder` интерфейс.
- **RQ-003** `RetryLLMProvider` — обёртка над LLM-провайдером.
- **RQ-004** `CircuitBreakerStats` — re-export статистики CB (FailureCount, State).
- **RQ-005** `ErrCircuitOpen` — sentinel error.
- **RQ-006** `IsRetryable`, `WrapRetryable`, `WrapNonRetryable` — утилиты классификации ошибок.
- **RQ-007** `CircuitState`, `CircuitClosed`, `CircuitOpen`, `CircuitHalfOpen` — re-export констант.

## Вне scope

- Новая resilience-логика.
- RetryVectorStore (не IO-heavy в той же степени).
- Метрики resilience через Hooks.

## Критерии приемки

### AC-001 Compile-time interface check

- **Given** `RetryEmbedder` создан через `NewRetryEmbedder`
- **When** присваивается переменной типа `Embedder`
- **Then** компилируется без ошибок
- **Evidence**: `var _ Embedder = re` в тесте или production коде

### AC-002 Zero-value RetryOptions дают разумные дефолты

- **Given** `RetryOptions{}` (zero value)
- **When** создаётся `RetryEmbedder`
- **Then** MaxRetries=3, BaseDelay=100ms, CBThreshold=5

### AC-003 ErrCircuitOpen accessible

- **Given** `draftrag.ErrCircuitOpen`
- **When** `errors.Is(err, draftrag.ErrCircuitOpen)`
- **Then** совпадает с ошибкой от открытого CB

## Допущения

- `pkg/draftrag` может импортировать `internal/` в том же модуле.
- Пользователи библиотеки знают о retry-паттерне и осознанно выбирают параметры.

## Открытые вопросы

- none
