# Rate Limiting для LLM API (Token Bucket)

## Scope Snapshot

- In scope: клиентский token bucket rate limiter, оборачивающий LLMProvider и Embedder для предотвращения HTTP 429.
- Out of scope: rate limiting для VectorStore, распределённый (multi-instance) rate limiter, динамическая адаптация лимитов на основе 429-ответов.

## Цель

Разработчик, использующий draftRAG с LLM/embedder провайдерами (OpenAI, Anthropic, Mistral, DeepSeek и др.), получает декоратор, который сглаживает burst-нагрузку и предотвращает 429 Too Many Requests. Успех фичи: пользователь может сконфигурировать rate (запросов/сек) и burst для любого LLMProvider или Embedder, и драйвер не допускает превышения лимита, блокируя запрос до появления токенов.

## Основной сценарий

1. Пользователь создаёт LLMProvider (например, `draftrag.NewOpenAICompatibleLLM`).
2. Пользователь оборачивает его в `draftrag.NewTokenBucketLLMProvider(llm, opts)`, указав `TokensPerSecond` и `BurstSize`.
3. При вызове `Generate(ctx, ...)` декоратор проверяет token bucket: если есть токены — пропускает, если нет — блокирует до появления токена или отмены контекста.
4. При получении 429 от upstream (если лимит всё же превышен) — ошибка распознаётся и помечается как retryable.
5. Пользователь может дополнительно обернуть результат в `NewRetryLLMProvider` для retry при 429.

## User Stories

- P1 (MVP): обёртка `LLMProvider` с token bucket, блокирующая запрос при отсутствии токенов. Публичный конструктор в `pkg/draftrag/`.
- P2: обёртка `Embedder` с тем же механизмом — симметричный `NewTokenBucketEmbedder`.

## MVP Slice

P1 — обёртка `LLMProvider`. AC-001, AC-002, AC-003.

## First Deployable Outcome

После первого implementation pass: go-тест, создающий LLMProvider с rate limit 5 req/s, вызывающий Generate 20 раз и измеряющий время выполнения (не менее 3.8с). Тест доказывает корректность rate limiter'а.

## Scope

- Token bucket алгоритм с настраиваемыми `TokensPerSecond` и `BurstSize`
- Декоратор `TokenBucketLLMProvider`, реализующий `domain.LLMProvider`
- Декоратор `TokenBucketEmbedder`, реализующий `domain.Embedder`
- Блокировка при пустом bucket с учётом `context.Context`
- Публичные конструкторы в `pkg/draftrag/`
- Нулевые значения полей → rate limiting отключён (passthrough)

## Контекст

- Все LLM/embedder провайдеры имеют API-ratelimit, нарушение которого даёт 429
- Rate limit — свойство клиента, а не провайдера: разные API-ключи имеют разные лимиты
- В existing codebase уже есть `retry` и `circuit breaker` в `internal/infrastructure/resilience/`
- Token bucket — proactive-подход (не ждать 429, а не превышать лимит)
- `time.Ticker` не подходит для burst-семантики — нужен именно token bucket

## Зависимости

- Внешних зависимостей нет (чистый Go + `time` + `sync`)
- Композируется с `RetryLLMProvider` / `RetryEmbedder` (внешний слой — rate limit, внутренний — retry при 429)

## Требования

- RQ-001 Система ДОЛЖНА предоставлять `TokenBucketLLMProvider` — декоратор `LLMProvider` с конфигурируемыми rate (tokens/sec) и burst.
- RQ-002 При отсутствии токенов `Generate` ДОЛЖЕН блокироваться до появления токена, отмены контекста или закрытия bucket. Выбор: блокировка (wait), а не immediate error.
- RQ-003 `TokenBucketLLMProvider` ДОЛЖЕН корректно обрабатывать `context.Context`: при отмене контекста ожидание токена прерывается с возвратом `context.Canceled`/`context.DeadlineExceeded`.
- RQ-004 Система ДОЛЖНА предоставлять `TokenBucketEmbedder` — декоратор `Embedder` с теми же настройками и поведением.
- RQ-005 Token bucket ДОЛЖЕН быть goroutine-safe (sync.Mutex или sync/atomic).
- RQ-006 Нулевые значения полей (`Rate=0, Burst=0`) ДОЛЖНЫ отключать rate limiting (passthrough без блокировки).
- RQ-007 Token bucket ДОЛЖЕН логировать события ожидания и отказа через `domain.Hooks`: при блокировке запроса, при успешном прохождении после ожидания, при прерывании по контексту.

## Вне scope

- Распределённый rate limiter (Redis/etcd-бэкенд) — только in-process
- Адаптивный rate limiter (изменение лимитов на основе наблюдаемых 429)
- Dashboard/метрики использования rate limiter'а — только hooks-события при ожидании/отказе
- Rate limiting для Chunker, VectorStore и других компонентов

## Критерии приемки

### AC-001 TokenBucketLLMProvider блокирует до появления токена

- Почему это важно: предотвращает 429 в upstream API при burst-нагрузке
- **Given** TokenBucketLLMProvider с `TokensPerSecond=1, BurstSize=1` и `LLMProvider`, у которого Generate мгновенный
- **When** два последовательных вызова Generate выполняются без паузы
- **Then** второй вызов длится ≥ 950ms (ожидание пополнения bucket)
- Evidence: замер времени выполнения в тесте

### AC-002 TokenBucketLLMProvider уважает отмену контекста во время ожидания

- Почему это важно: пользователь может прервать долгое ожидание
- **Given** TokenBucketLLMProvider с `TokensPerSecond=1, BurstSize=1`
- **When** первый вызов израсходовал токен, и второй вызов начинает ожидание; контекст отменяется во время ожидания
- **Then** Generate возвращает ошибку, равную `context.Canceled`
- Evidence: `errors.Is(result, context.Canceled)`

### AC-003 TokenBucketLLMProvider passthrough при нулевых настройках

- Почему это важно: безопасный default — ноль = отключено
- **Given** TokenBucketLLMProvider с `TokensPerSecond=0, BurstSize=0`
- **When** вызывается Generate
- **Then** вызов не блокируется и проходит напрямую к внутреннему LLMProvider
- Evidence: время выполнения эквивалентно прямому вызову без rate limiter'а

### AC-004 TokenBucketEmbedder работает симметрично

- Почему это важно: embedder'ы также подвержены 429
- **Given** TokenBucketEmbedder с `TokensPerSecond=5, BurstSize=5`
- **When** 10 вызовов Embed выполняются параллельно
- **Then** общее время выполнения ≥ 1.8с
- Evidence: замер времени в тесте

### AC-005 TokenBucketLLMProvider логирует события блокировки через Hooks

- Почему это важно: observability — пользователь видит, что запросы задерживаются rate limiter'ом
- **Given** TokenBucketLLMProvider с `TokensPerSecond=1, BurstSize=1` и подключённым `Hooks`
- **When** два последовательных вызова Generate; второй вызов ожидает токен
- **Then** Hooks получает событие с типом `rate_limit_wait`, длительностью ожидания и именем операции
- Evidence: mock Hooks с подсчётом вызовов `OnEvent` проверяет наличие события `rate_limit_wait`

### AC-006 TokenBucketLLMProvider + RetryLLMProvider: retry при 429

- Почему это важно: если burst всё же превышен и upstream вернул 429, retry-слой повторяет запрос
- **Given** TokenBucketLLMProvider с `TokensPerSecond=10, BurstSize=5` обёрнут в RetryLLMProvider с `MaxRetries=1`, и внутренний LLMProvider возвращает 429-ошибку (retryable)
- **When** вызывается Generate
- **Then** первый вызов получает 429 → RetryLLMProvider выполняет retry → второй вызов проходит через TokenBucketLLMProvider (ожидает токен при необходимости) → внутренний LLMProvider возвращает успешный ответ
- Evidence: Generate возвращает успешный ответ; тест подсчитывает, что внутренний провайдер вызван дважды (первый раз с 429, второй — успешно)

## Допущения

- Rate limit — константа на всё время жизни декоратора (не меняется динамически)
- Провайдеры не сообщают Retry-After заголовок (упор на proactive предотвращение, а не реактивный backoff)
- Задержка refill токена вычисляется как `time.Second / TokensPerSecond` (равномерное пополнение)

## Критерии успеха

- SC-001 Тест с 20 вызовами при rate=10, burst=5 выполняется за ≥ 1.5с
- SC-002 Нет false-positive 429 при rate=10 и вызовах ≤10/сек

## Краевые случаи

- `TokensPerSecond=0, Burst=0` → passthrough (rate limiting отключён)
- `Burst=0` при ненулевом Rate → Burst=Rate (burst должен быть ≥ 1)
- `TokensPerSecond` или `Burst` < 0 — валидация с возвратом ошибки при конструкторе
- Огромный Burst (1M+) — проверка на integer overflow
- Контекст с deadline во время ожидания заполнения bucket

## Открытые вопросы

- none
