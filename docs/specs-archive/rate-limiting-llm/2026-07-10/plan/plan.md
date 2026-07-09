# Rate Limiting для LLM API (Token Bucket) — План

## Phase Contract

Inputs: spec, inspect report (pass), минимальный repo-контекст.
Outputs: plan, data-model.md.
Stop if: нет.

## Цель

Два декоратора (`TokenBucketLLMProvider`, `TokenBucketEmbedder`) в пакете `internal/infrastructure/resilience/` с публичными конструкторами в `pkg/draftrag/`. Внутренний token bucket — разделяемый тип, hooks — через новое значение `HookStageRateLimit` в существующем `domain.Hooks`.

## MVP Slice

`TokenBucketLLMProvider` + внутренний token bucket. AC-001, AC-002, AC-003.

## First Validation Path

`go test -run TestTokenBucketLLMProvider_Blocks -v` — замер времени 2 последовательных вызовов при rate=1. Второй вызов ≥ 900ms.

## Scope

- `internal/infrastructure/resilience/tokenbucket.go` — token bucket (internal struct)
- `internal/infrastructure/resilience/ratelimit_llm.go` — `TokenBucketLLMProvider`
- `internal/infrastructure/resilience/ratelimit_embedder.go` — `TokenBucketEmbedder`
- `internal/domain/hooks.go` — добавление `HookStageRateLimit`
- `pkg/draftrag/ratelimit.go` — публичные конструкторы + `TokenBucketOptions`
- `*_test.go` — юнит-тесты на каждый декоратор

## Performance Budget

- `none` — накладные расходы token bucket: 1 mutex lock + 1 time calculation per call, negligible

## Implementation Surfaces

- `internal/infrastructure/resilience/tokenbucket.go` (new) — core struct `tokenBucket` с `Take(ctx, n int64) error`; goroutine-safe через `sync.Mutex`; refill через ticker
- `internal/infrastructure/resilience/ratelimit_llm.go` (new) — `TokenBucketLLMProvider`, хранит bucket + inner `LLMProvider` + `Hooks`; `Generate` вызывает `bucket.Take()` перед вызовом inner
- `internal/infrastructure/resilience/ratelimit_embedder.go` (new) — `TokenBucketEmbedder`, симметрично
- `internal/domain/hooks.go` (change) — добавить `HookStageRateLimit HookStage = "rate_limit"`
- `pkg/draftrag/ratelimit.go` (new) — `TokenBucketOptions` struct, `NewTokenBucketLLMProvider`, `NewTokenBucketEmbedder`

## Bootstrapping Surfaces

- `pkg/draftrag/ratelimit.go` — новый файл в существующем пакете
- `internal/infrastructure/resilience/tokenbucket.go` — новый файл в существующем пакете

## Влияние на архитектуру

- Локальное: добавление `HookStageRateLimit` в domain — не breaking change (только новый const)
- Нет migration/rollout-последствий — библиотека, не сервис
- Слой rate limiter прозрачен для pipeline: пользователь оборачивает провайдер до передачи в `NewPipeline`

## Acceptance Approach

- AC-001 → `ratelimit_llm_test.go`: два последовательных `Generate` при rate=1, замер времени второго
- AC-002 → `ratelimit_llm_test.go`: отмена контекста во время ожидания, проверка `errors.Is(result, context.Canceled)`
- AC-003 → `ratelimit_llm_test.go`: нулевые настройки, вызов проходит без задержки
- AC-004 → `ratelimit_embedder_test.go`: 10 параллельных `Embed` при rate=5, замер ≥ 1.8s
- AC-005 → `ratelimit_llm_test.go`: mock Hooks, проверка события `rate_limit_wait` с длительностью
- AC-006 → `ratelimit_llm_test.go`: inner возвращает 429, RetryLLMProvider на верхнем слое делает retry, подсчёт вызовов inner

## Данные и контракты

- `domain.HooksStage` — новое значение `HookStageRateLimit`, не breaking change
- `domain.TokenUsage`, `domain.ModelPricing` и другие модели данных не меняются
- Публичные конструкторы — новый файл, новая экспортируемая поверхность
- `data-model.md`: no-change (кроме HookStage const)

## Стратегия реализации

### DEC-001 Token bucket — internal struct с методами Take и Stop

Why: единый механизм для LLM и Embedder обёрток; не нужно дублировать refill-логику.
Tradeoff: `sync.Mutex` вместо атомиков — простота важнее наносекунд (rate limit ~1–1000 req/s, contention нерелевантен).
Affects: `internal/infrastructure/resilience/tokenbucket.go`.
Validation: `TestTokenBucket_Take_Blocks`, `TestTokenBucket_Take_ContextCancel`.

### DEC-002 HookStageRateLimit — новое значение HookStage

Why: не breaking change; существующие реализации Hooks получают no-op для неизвестного stage (игнорируют).
Tradeoff: rate limit не является pipeline stage, но переиспользование Hooks проще, чем новый интерфейс.
Affects: `internal/domain/hooks.go`.
Validation: AC-005.

### DEC-003 Options struct — нулевое значение = passthrough

Why: следует паттерну `RetryOptions` в том же пакете; безопасный default.
Tradeoff: пользователь может случайно не задать лимит и не получить rate limiting — это intentional (opt-in).
Affects: `pkg/draftrag/ratelimit.go`.
Validation: AC-003.

### DEC-004 429 detection — делегируется существующему IsRetryable / RetryLLMProvider

Why: spec не требует встроенного распознавания 429 в rate limiter'е; retry-слой уже умеет это.
Tradeoff: если пользователь не обернёт в RetryLLMProvider, 429 дойдёт до него как обычная ошибка.
Affects: `internal/infrastructure/resilience/ratelimit_llm.go`, `pkg/draftrag/resilience.go`.
Validation: AC-006.

## Incremental Delivery

### MVP (Первая ценность)

- `tokenbucket.go` — core struct
- `ratelimit_llm.go` — `TokenBucketLLMProvider`
- `pkg/draftrag/ratelimit.go` — конструктор + options
- `ratelimit_llm_test.go` — AC-001, AC-002, AC-003
- `internal/domain/hooks.go` — `HookStageRateLimit`
Критерий: `TestTokenBucketLLMProvider_Blocks` проходит.

### Итеративное расширение

- Шаг 2: `ratelimit_embedder.go` + тесты → AC-004
- Шаг 3: hooks-логирование в `ratelimit_llm.go` + тест → AC-005
- Шаг 4: композиционный тест `TokenBucketLLMProvider` + `RetryLLMProvider` → AC-006

## Порядок реализации

1. `internal/domain/hooks.go` — добавить `HookStageRateLimit` (1 строка, не блокирует остальное)
2. `internal/infrastructure/resilience/tokenbucket.go` — core token bucket (MVP foundation)
3. `internal/infrastructure/resilience/ratelimit_llm.go` — LLM обёртка (MVP)
4. `pkg/draftrag/ratelimit.go` — публичный API (MVP)
5. Тесты AC-001, AC-002, AC-003 (MVP validation)
6. `internal/infrastructure/resilience/ratelimit_embedder.go` + тест AC-004
7. hooks-логирование в ratelimit_llm.go + тест AC-005
8. Композиционный тест AC-006
Параллельно: тесты на internal token bucket.

## Риски

- Flakiness в timing-чувствительных тестах (AC-001, AC-004) — Mitigation: slack 0.9× от теоретического минимума; `sleep` padding не используется, только time.Now
- Hooks-событие `rate_limit_wait` требует конвенции об именовании событий — Mitigation: Operation = `"rate_limit_wait"`, Stage = `HookStageRateLimit`; документировано в комментарии
- При `TokensPerSecond=0, Burst=0` (passthrough) hooks не вызываются — intentional (нет ожидания = нет события)

## Rollout и compatibility

- Специальных rollout-действий не требуется — это библиотека; пользователь явно выбирает использование
- `HookStageRateLimit` не breaking change (новый const)

## Проверка

- Юнит-тесты на каждый AC (см. Acceptance Approach)
- `go test ./internal/infrastructure/resilience/...` + `go test ./pkg/draftrag/...`
- `go vet ./...`, `golangci-lint run ./...`
- `@sk-test` маркеры над каждым тестом, `@sk-task` над каждым конструктором/методом

## Соответствие конституции

- нет конфликтов
