# Graceful Degradation: Chain Fallback для LLM-провайдеров

## Scope Snapshot

- In scope: механизм последовательного переключения между LLM-провайдерами при отказе основного (Primary → Secondary → Local), с логированием причины fallback и наблюдаемым состоянием цепи.
- Out of scope: автоматический health-check и динамическое переключение без on-flight ошибок; переключение для Embedder провайдеров; переключение для VectorStore.

## Цель

Разработчик RAG-системы на Go получает возможность сконфигурировать цепочку LLM-провайдеров, в которой при отказе основного провайдера (Primary) запрос автоматически направляется на Secondary, а при отказе Secondary — на Local (например, Ollama). Фича считается успешной, когда для любого PublicAPI метода LLMProvider (Generate, GenerateStream, GenerateWithUsage) при ошибке от текущего провайдера происходит fallback к следующему в цепи без потери запроса, и весь путь fallback наблюдаем через логи/hooks.

## Основной сценарий

1. Пользователь создаёт 2–3 LLM провайдера (Primary, Secondary, Local) и оборачивает их в `ChainLLMProvider`.
2. Вызов `Generate(ctx, system, user)` сначала пробует Primary.
3. Если Primary возвращает retryable-ошибку (в т.ч. после исчерпания retry) — запрос направляется к Secondary.
4. Если Secondary также возвращает ошибку — запрос направляется к Local.
5. Если все провайдеры в цепи отказали — возвращается aggregate-ошибка.
6. Каждый fallback логируется через `domain.Logger` и опционально через `domain.Hooks`.
7. Если провайдер вернул успешный ответ — он возвращается вызывающему коду, fallback на этом прерывается.

## User Stories

- P1 Story: конфигурация цепи из 2 провайдеров с fallback при retryable-ошибке, observable через hooks.
- P2 Story: конфигурация из 3 провайдеров (Primary → Secondary → Local) с наблюдаемым состоянием активного провайдера через публичный API.

## MVP Slice

P1: цепь из 2 провайдеров, fallback только для `Generate` и `Health`. Публичный API — `NewFallbackLLMProvider(primary, secondary, ...additional)`.

## First Deployable Outcome

Юнит-тест, в котором primary возвращает retryable-ошибку, secondary возвращает успешный ответ, и вызывающий код получает этот ответ. Вызовы и fallback логируются.

## Scope

- Новый тип `FallbackLLMProvider` (wrapper), реализующий `domain.LLMProvider`.
- Поддержка `Health(ctx)` — проверка доступности текущего провайдера в цепи (без fallback при Health).
- Логирование каждого акта fallback через Logger.
- Интеграция с Hooks (OnEnd/OnError для каждого шага fallback).
- Публичный конструктор `NewFallbackLLMProvider` в `pkg/draftrag/`.
- Покрытие юнит-тестами: happy path, fallback к secondary, fallback ко всем, non-retryable ошибка не вызывает fallback.
- Новый тип `ErrAllProvidersFailed` — публичный sentinel error.
- Публичный тип `FallbackStats` с полями `TotalCalls`, `PrimaryFailures`, `FallbackCount`, `LastError` и методом `Stats()` на каждом Fallback-типе.

## Контекст

- Существующий `RetryLLMProvider` покрывает retry внутри одного провайдера, но не fallback к другому.
- `RetryLLMProvider` не должен быть изменён — фича layer'уется поверх него.
- Все провайдеры реализуют `domain.LLMProvider`.
- `Health(ctx)` не вызывает fallback — это intentional: health-check должен отражать состояние только проверяемого провайдера.
- Контекст из вызывающего кода пробрасывается без изменений — fallback не отменяет оригинальный контекст.

## Зависимости

- `internal/domain` — интерфейсы LLMProvider, Hooks, Logger.
- `internal/infrastructure/resilience` — классификация ошибок (IsRetryable).
- `pkg/draftrag` — публичный re-export.
- `NewRetryLLMProvider` может быть использован как один из элементов цепи, но цепь не требует обязательного retry.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять `FallbackLLMProvider`, реализующий `domain.LLMProvider` и принимающий Ordered-список провайдеров.
- RQ-002 Система ДОЛЖНА при retryable-ошибке текущего провайдера автоматически пробовать следующий в цепи.
- RQ-003 Система НЕ ДОЛЖНА вызывать fallback для non-retryable ошибок (возвращать ошибку немедленно).
- RQ-004 Система ДОЛЖНА логировать каждый акт fallback (причину, номер провайдера, последнюю ошибку).
- RQ-005 Система ДОЛЖНА возвращать aggregate-ошибку, если все провайдеры в цепи отказали.
- RQ-006 Система ДОЛЖНА поддерживать `Health(ctx)` без fallback — health запрашивается только у первого провайдера.
- RQ-007 Система ДОЛЖНА вызывать `domain.Hooks.OnError` для каждого fallback-события.
- RQ-008 Публичный API ДОЛЖЕН быть в `pkg/draftrag/` как `NewFallbackLLMProvider`.
- RQ-009 Система ДОЛЖНА предоставлять `FallbackStreamingLLMProvider`, реализующий `domain.StreamingLLMProvider` с fallback по тому же принципу — при ошибке/преждевременном закрытии канала переход к следующему провайдеру.
- RQ-010 Система ДОЛЖНА предоставлять `FallbackUsageAwareLLMProvider`, реализующий `domain.UsageAwareLLMProvider` с fallback при retryable-ошибке в `GenerateWithUsage`.
- RQ-011 Система ДОЛЖНА предоставлять публичный метод `Stats()` на каждом Fallback-типе, возвращающий `FallbackStats` со счётчиками: TotalCalls, PrimaryFailures, FallbackCount, LastError.
- RQ-012 Система ДОЛЖНА предоставлять публичную ошибку-сентинел `ErrAllProvidersFailed`.

## Вне scope

- Fallback для Embedder или VectorStore.
- Динамическое изменение цепи (add/remove провайдеров после создания).
- Автоматический health-check на фоне.
- Fallback для Embedder-аналогов Streaming/UsageAware (нет соответствующих интерфейсов в domain).
- Конфигурация через YAML/JSON (чисто программный API).

## Критерии приемки

### AC-001 Fallback при retryable-ошибке

- Почему это важно: при временной недоступности Primary запрос не теряется, а уходит на Secondary.
- **Given** FallbackLLMProvider с 2 провайдерами: Primary и Secondary
- **When** Primary возвращает retryable-ошибку (IsRetryable == true)
- **Then** Secondary получает запрос и возвращает ответ, и вызывающий код получает этот ответ
- **Evidence** результат вызова Generate содержит ответ Secondary; лог содержит запись о fallback с Primary → Secondary

### AC-002 Non-retryable ошибка не вызывает fallback

- Почему это важно: неверный запрос (bad request, invalid auth) не должен перенаправляться — он ошибка пользователя.
- **Given** FallbackLLMProvider с 2 провайдерами
- **When** Primary возвращает non-retryable-ошибку
- **Then** Secondary не вызывается, пользователю возвращается ошибка Primary
- **Evidence** возвращённая ошибка равна ошибке Primary; лог не содержит fallback-записей

### AC-003 Aggregate-ошибка при отказе всех провайдеров

- Почему это важно: вызывающий код должен знать, что все пути исчерпаны, и видеть последнюю ошибку.
- **Given** FallbackLLMProvider с 2 провайдерами, оба возвращают retryable-ошибки
- **When** Generate вызывается
- **Then** возвращается aggregate-ошибка, содержащая ошибку последнего провайдера
- **Evidence** `errors.Is(result, ErrAllProvidersFailed)` или errors.Unwrap даёт последнюю ошибку; в логе 2 fallback-записи

### AC-004 Health без fallback

- Почему это важно: Health должен показывать реальную доступность первого провайдера, а не эхо от последнего.
- **Given** FallbackLLMProvider с 2 провайдерами, Primary в состоянии circuit breaker open
- **When** Health(ctx) вызывается
- **Then** Health возвращает ошибку Primary (circuit breaker open), Secondary не вызывается
- **Evidence** результат Health содержит ErrCircuitOpen, лог не содержит fallback-записей

### AC-005 Hooks вызываются для каждого fallback

- Почему это важно: observability — владелец системы видит, сколько раз и почему происходил fallback.
- **Given** FallbackLLMProvider с Hooks, Primary возвращает retryable-ошибку, Secondary успешен
- **When** Generate вызывается
- **Then** Hooks.OnError вызывается ровно 1 раз с ошибкой Primary
- **Evidence** mock Hooks фиксирует 1 вызов OnError с правильной ошибкой

### AC-006 FallbackStreamingLLMProvider переключает провайдера при ошибке канала

- Почему это важно: стриминг не должен обрываться полностью при отказе Primary.
- **Given** FallbackStreamingLLMProvider с 2 провайдерами, Primary возвращает канал, который закрывается с ошибкой
- **When** `GenerateStream` вызывается и канал из Primary читается
- **Then** Secondary получает вызов, и последующие токены приходят из Secondary
- **Evidence** читатель получает последовательные токены от Secondary после ошибки Primary; лог содержит fallback-запись

### AC-007 FallbackUsageAwareLLMProvider возвращает корректный TokenUsage

- Почему это важно: при fallback на usage-aware провайдере пользователь не теряет учёт токенов.
- **Given** FallbackUsageAwareLLMProvider с 2 провайдерами (оба UsageAwareLLMProvider)
- **When** Primary возвращает retryable-ошибку, Secondary успешно отвечает с TokenUsage
- **Then** результат GenerateWithUsage содержит TokenUsage от Secondary
- **Evidence** TokenUsage.Secondary не нулевой; вызовы распределены: Primary — 1 неудача, Secondary — 1 успех

### AC-008 FallbackStats отражает падения Primary

- Почему это важно: разработчик может программно отслеживать деградацию Primary.
- **Given** FallbackLLMProvider с 2 провайдерами, Primary возвращает retryable-ошибку, Secondary успешен
- **When** Generate вызывается 3 раза, из них 2 — неудача Primary, 1 — успех Primary
- **Then** `Stats()` возвращает: TotalCalls=3, PrimaryFailures=2, FallbackCount=2, LastError=nil
- **Evidence** значения полей соответствуют ожидаемым

### AC-009 Пустая цепь отклоняется конструктором

- Почему это важно: цепь без провайдеров — ошибка конфигурации, её нужно поймать при сборке/ините.
- **Given** список провайдеров длины 0
- **When** `NewFallbackLLMProvider` вызывается
- **Then** возвращается ошибка (nil-провайдер)
- **Evidence** err != nil

## Допущения

- Провайдеры в цепи уже обёрнуты в RetryLLMProvider (если нужен retry) — FallbackLLMProvider не добавляет retry, только fallback.
- Non-retryable ошибки не вызывают fallback — это защита от багов в конфигурации.
- Fallback для `Generate`, `GenerateStream`, `GenerateWithUsage` живут в отдельных обёртках — каждая реализует свой контракт.
- Health во всех Fallback-типах проверяет только первый провайдер, без fallback.

## Критерии успеха

- SC-001 Время выполнения при успехе Secondary не превышает время Primary + одноразовая задержка логирования (<1ms overhead).
- SC-002 100% юнит-тестов проходят для всех сценариев fallback.

## Краевые случаи

- Пустая цепь (0 провайдеров) — конструктор возвращает error.
- Один провайдер в цепи — поведение эквивалентно прямому вызову (no fallback).
- Провайдер возвращает успех после retry — fallback не активируется.
- Провайдер возвращает ошибку, но context отменён между вызовами — fallback не должен продолжаться (проверка ctx.Err() перед каждым следующим провайдером).

## Открытые вопросы

1. ~~Что делать с `StreamingLLMProvider` и `UsageAwareLLMProvider` в цепи?~~ — Разделить на `FallbackStreamingLLMProvider` и `FallbackUsageAwareLLMProvider`.
2. ~~Нужна ли публичная метрика "сколько раз упал primary"?~~ — Да, добавлен `FallbackStats`.
3. Должен ли FallbackLLMProvider сам решать "ошибка против health" или полагаться на IsRetryable из resilience? — Полагается на IsRetryable.