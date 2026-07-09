# Rate Limiting для LLM API (Token Bucket) — Задачи

## Phase Contract

Inputs: spec, plan, data-model (no-change).
Outputs: исполнимые задачи с покрытием 6 AC.
Stop if: нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/hooks.go` | T0.1 |
| `internal/infrastructure/resilience/tokenbucket.go` | T0.2 |
| `internal/infrastructure/resilience/ratelimit_llm.go` | T1.1 |
| `pkg/draftrag/ratelimit.go` | T1.2 |
| `internal/infrastructure/resilience/ratelimit_llm_test.go` | T1.3, T3.2, T4.1 |
| `internal/infrastructure/resilience/ratelimit_embedder.go` | T2.1 |
| `internal/infrastructure/resilience/ratelimit_embedder_test.go` | T2.2 |
| `internal/infrastructure/resilience/tokenbucket_test.go` | T0.3 |

## Implementation Context

- Цель MVP: `TokenBucketLLMProvider` + public `NewTokenBucketLLMProvider` в `pkg/draftrag/` + тесты AC-001–003
- Инварианты: token bucket refill — 1 токен per `time.Second / TokensPerSecond`; zero options = passthrough; burst ≥ 1
- Контракты: `HookStageRateLimit` — новый const, не breaking; `TokenBucketOptions` — zero = off
- Ошибки: при отмене контекста — возвращать `ctx.Err()`; при `Rate<0 || Burst<0` — ошибка конструктора
- Proof signals: timing-sensitive тесты с slack 0.9×; mock Hooks для AC-005; подсчёт вызовов inner для AC-006
- Вне scope: adaptive rate limit, distributed rate limiter, VectorStore rate limit, Chunker rate limit
- References: DEC-001 (internal tokenBucket), DEC-002 (HookStage), DEC-003 (options zero=passthrough), DEC-004 (429 delegation)

## Фаза 0: Bootstrapping

Цель: подготовить константу HookStage и core token bucket, на который опираются все декораторы.

- [x] T0.1 Добавить `HookStageRateLimit HookStage = "rate_limit"` в `internal/domain/hooks.go`. Touches: `internal/domain/hooks.go`
- [x] T0.2 Реализовать internal struct `tokenBucket` с методами `Take(ctx, n int64) error` и `Stop()` в `internal/infrastructure/resilience/tokenbucket.go`. refill через time.Ticker, goroutine-safe через sync.Mutex. При отмене контекста во время ожидания — возвращать `ctx.Err()`. Touches: `internal/infrastructure/resilience/tokenbucket.go`
- [x] T0.3 Добавить unit-тесты на token bucket: `TestTokenBucket_Take_Blocks`, `TestTokenBucket_Take_ContextCancel`, `TestTokenBucket_Take_RefillRate`. Touches: `internal/infrastructure/resilience/tokenbucket_test.go`

## Фаза 1: MVP (TokenBucketLLMProvider)

Цель: TokenBucketLLMProvider оборачивает LLMProvider, блокирует при исчерпании токенов, passthrough при нулевых настройках.

- [x] T1.1 Реализовать `TokenBucketLLMProvider` — декоратор `domain.LLMProvider`. Хранит `tokenBucket`, inner `LLMProvider`, опциональные `Hooks`. `Generate()` вызывает `bucket.Take()` перед вызовом inner. При нулевых настройках — passthrough без bucket. Touches: `internal/infrastructure/resilience/ratelimit_llm.go`
- [x] T1.2 Реализовать `TokenBucketOptions` struct (`TokensPerSecond`, `BurstSize`) и публичный конструктор `NewTokenBucketLLMProvider(llm, opts)` в `pkg/draftrag/ratelimit.go`. Валидация: `Rate<0 || Burst<0` → error. `Rate==0` → passthrough. Touches: `pkg/draftrag/ratelimit.go`
- [x] T1.3 Добавить тесты для AC-001 (два последовательных вызова, замер времени), AC-002 (context cancel во время ожидания), AC-003 (passthrough при нулевых настройках). `@sk-test` маркеры над каждой тест-функцией. Touches: `internal/infrastructure/resilience/ratelimit_llm_test.go`

## Фаза 2: Embedder

Цель: симметричный декоратор для Embedder с тем же token bucket механизмом.

- [x] T2.1 Реализовать `TokenBucketEmbedder` — декоратор `domain.Embedder`. Симметричен TokenBucketLLMProvider: `Embed()` вызывает `bucket.Take()`. Переиспользует тот же internal `tokenBucket`. Touches: `internal/infrastructure/resilience/ratelimit_embedder.go`
- [x] T2.2 Добавить публичный конструктор `NewTokenBucketEmbedder(emb, opts)` в `pkg/draftrag/ratelimit.go`. Тест для AC-004: 10 параллельных Embed при rate=5, замер ≥ 1.8s. `@sk-test` маркеры. Touches: `pkg/draftrag/ratelimit.go`, `internal/infrastructure/resilience/ratelimit_embedder_test.go`

## Фаза 3: Observability (Hooks)

Цель: rate limiter логирует события ожидания через Hooks.

- [x] T3.1 Добавить hooks-логирование в `TokenBucketLLMProvider.Generate()`: при начале ожидания — `StageStart` с `HookStageRateLimit` и `Operation="rate_limit_wait"`, при завершении — `StageEnd` с длительностью. При отмене контекста — `StageEnd` с ошибкой. Touches: `internal/infrastructure/resilience/ratelimit_llm.go`
- [x] T3.2 Добавить тест для AC-005: mock Hooks, проверка события `rate_limit_wait` с длительностью и Operation. Touches: `internal/infrastructure/resilience/ratelimit_llm_test.go`
- [x] T4.1 Добавить тест для AC-006: TokenBucketLLMProvider обёрнут в RetryLLMProvider; inner LLMProvider возвращает 429 (retryable) на первый вызов; проверить, что Generate возвращает успех, inner вызван дважды. Touches: `internal/infrastructure/resilience/ratelimit_llm_test.go`

## Покрытие критериев приемки

- AC-001 -> T0.2, T1.1, T1.3
- AC-002 -> T0.2, T1.1, T1.3
- AC-003 -> T1.1, T1.2, T1.3
- AC-004 -> T0.2, T2.1, T2.2
- AC-005 -> T3.1, T3.2
- AC-006 -> T4.1

## Заметки

- Фазы 0–4 выполняются строго последовательно (каждая опирается на предыдущую)
- `@sk-task` маркеры — над owning function/method/type declaration, не на уровне package/import
- После implement: `go test ./internal/infrastructure/resilience/...` + `go test ./pkg/draftrag/...` + `go vet ./...`
