# Health Check Interface План

## Цель

Добавить `Health(ctx context.Context) error` в интерфейсы `VectorStore`, `Embedder`, `LLMProvider` и реализовать его во всех штатных реализациях. Предоставить публичный `HealthChecker` (агрегатор) + `http.HandlerFunc` для K8s probes. Меняется только несущая конструкция интерфейсов — ни одна существующая сигнатура не ломается.

## MVP Slice

Интерфейсный контракт + одна реализация (InMemoryStore). Покрывает AC-001..AC-004.

## First Validation Path

`go build ./...` проходит, `go vet ./...` проходит. Затем `go test -run TestInMemoryStore_Health ./internal/infrastructure/vectorstore/`.

## Scope

- `internal/domain/interfaces.go` — добавить `Health(ctx) error` в `VectorStore`, `Embedder`, `LLMProvider`
- `internal/infrastructure/vectorstore/*.go` — реализация Health для 6 бэкендов
- `internal/infrastructure/embedder/*.go` — реализация Health для 2 embedder'ов
- `internal/infrastructure/llm/*.go` — реализация Health для 3 LLM (OpenAIChatLLM, ClaudeLLM, OllamaLLM)
- `pkg/draftrag/mistral_llm.go`, `deepseek_llm.go`, `anthropic_llm.go` — делегирующие Health
- `pkg/draftrag/cached_embedder.go` — делегирующая Health
- `internal/infrastructure/resilience/embedder.go` — Health с CB-логикой
- `internal/infrastructure/resilience/llm.go` — Health с CB-логикой
- `pkg/draftrag/health.go` (новый) — `HealthChecker`, `LivenessHandler`, `ReadinessHandler`, `StartupHandler`

Не меняются: `Chunker`, `Reranker`, `Hooks`, `Logger`, `Pipeline` (кроме экспорта HealthChecker), `search*.go`, `eval/`, `otel/`.

## Performance Budget

`none` — Health не на критическом пути запроса.

## Implementation Surfaces

| Surface | Почему меняется |
|---|---|
| `internal/domain/interfaces.go` | Base contract — добавляется `Health` |
| `internal/infrastructure/vectorstore/{pgvector,qdrant,chromadb,weaviate,milvus,memory}.go` | Каждый бэкенд реализует Health |
| `internal/infrastructure/embedder/{ollama,openai_compatible}.go` | Каждый embedder реализует Health |
| `internal/infrastructure/llm/{openai_chat,anthropic,ollama}.go` | Каждый LLM реализует Health |
| `pkg/draftrag/{mistral_llm,deepseek_llm,anthropic_llm}.go` | Делегируют impl.Health |
| `pkg/draftrag/cached_embedder.go` | Делегирует внутреннему embedder |
| `internal/infrastructure/resilience/{embedder,llm}.go` | Health с CB: closed→делегировать, open→ошибка |
| `pkg/draftrag/health.go` (новый) | Публичный `HealthChecker` + HTTP-handler'ы |
| `pkg/draftrag/draftrag.go` | Re-export `HealthChecker` |

## Bootstrapping Surfaces

`pkg/draftrag/health.go` — новый файл.

## Влияние на архитектуру

- Минимальное: добавляется метод к трём базовым интерфейсам. Все существующие имплементации обновляются в той же фиче — backward compat не страдает.
- Никаких новых зависимостей. HTTP-handler'ы используют только `net/http` (уже в stdlib).
- HealthChecker — простая структура, не затрагивает Pipeline.

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|---|---|---|---|
| AC-001 | Добавить `Health` в `VectorStore` | `interfaces.go` | `go vet ./...` |
| AC-002 | Добавить `Health` в `Embedder` | `interfaces.go` | `go vet ./...` |
| AC-003 | Добавить `Health` в `LLMProvider` | `interfaces.go` | `go vet ./...` |
| AC-004 | `InMemoryStore.Health() -> nil` | `memory.go` | unit-test |
| AC-005 | `HealthChecker` агрегирует ошибки | `health.go` | unit-test |
| AC-006 | `LivenessHandler -> 200` | `health.go` | `httptest` |
| AC-007 | `ReadinessHandler -> 200/503` | `health.go` | `httptest` ×2 |
| AC-008 | `RetryEmbedder.Health` делегирует без retry | `resilience/embedder.go` | mock + callCount |
| AC-009 | `RetryEmbedder.Health` при open CB → ошибка | `resilience/embedder.go` | mock + open state |

## Данные и контракты

data model не меняется — Health не добавляет полей в `Document`, `Chunk` и т.д. См. `data-model.md` (no-change).

## Стратегия реализации

- **DEC-001: `Health()` как часть базового интерфейса, а не optional capability**
  - Why: Health должна быть гарантированно доступна для любого компонента — пользователь не должен проверять type assertion. Все штатные реализации обновляются в той же фиче.
  - Tradeoff: breaking change для внешних пользовательских имплементаций интерфейсов (должны добавить `Health`). Принимаем — библиотека в pre-1.0.
  - Affects: `internal/domain/interfaces.go`
  - Validation: `go build ./...` + `go vet ./...`

- **DEC-002: Network-Health через HEAD/PING, без нового HTTP-клиента**
  - Why: pgvector — `(*sql.DB).PingContext()`. Остальные network-бэкенды — HEAD на base URL с существующим `http.Client`. Не вводим новых зависимостей.
  - Tradeoff: HEAD может не поддерживаться всеми API — в этом случае GET с close body. Для Qdrant/ChromaDB/Weaviate используем их существующие health-эндпоинты.
  - Affects: все network-реализации

- **DEC-003: `HealthChecker` — простая структура, а не интерфейс**
  - Why: HealthChecker — композиция named компонентов, не требующая расширения. Если понадобится кастомная агрегация — пользователь напишет свою.
  - Tradeoff: нельзя подменить логику агрегации без копипасты.
  - Affects: `health.go`

- **DEC-004: `LivenessHandler` всегда 200 (не проверяет компоненты)**
  - Why: спецификация K8s: liveness = процесс жив, readiness = готов обслуживать. Разделяем ответственность.
  - Tradeoff: нет. Если процесс запущен — библиотека "жива".
  - Affects: `health.go`

- **DEC-005: Wrapper-типы (mistralLLM, deepseekLLM, anthropicLLM, CachedEmbedder) делегируют Health без добавления логики**
  - Why: Health отражает состояние внешней зависимости, а не конфигурации. Валидация опций не относится к Health.
  - Affects: `mistral_llm.go`, `deepseek_llm.go`, `anthropic_llm.go`, `cached_embedder.go`
  - Validation: unit-test через mock

## Incremental Delivery

### MVP (Первая ценность)

1. `Health` в интерфейсах + `InMemoryStore.Health` — AC-001..AC-004
2. `HealthChecker` + HTTP-handler'ы — AC-005..AC-007
3. Retry/CB обёртки — AC-008..AC-009

**Критерий готовности MVP:** `go build ./...`, `go vet ./...`, все новые unit-test'ы проходят.

### Итеративное расширение

После MVP — реализации Health для всех network-бэкендов (pgvector, qdrant, chromadb, weaviate, milvus, ollama embedder, openai-compatible embedder, OpenAIChatLLM, ClaudeLLM, OllamaLLM, mistral, deepseek, anthropic wrapper). Каждая — отдельный коммит/PR с unit-test.

## Порядок реализации

1. `internal/domain/interfaces.go` — добавить `Health` (иначе не компилируется ни одна имплементация)
2. `internal/infrastructure/vectorstore/memory.go` — первый Health
3. `internal/infrastructure/vectorstore/` — остальные бэкенды
4. `internal/infrastructure/embedder/` — embedder'ы
5. `internal/infrastructure/llm/` — LLM
6. `internal/infrastructure/resilience/` — retry/CB
7. `pkg/draftrag/` — wrapper'ы (mistral, deepseek, anthropic, cached)
8. `pkg/draftrag/health.go` — HealthChecker + handler'ы
9. `pkg/draftrag/draftrag.go` — re-export

Параллельно: шаги 2-5 можно выполнять в любом порядке после шага 1.

## Риски

| Риск | Mitigation |
|---|---|
| Network-Health вызов может сам заблокироваться | Используем контекст с таймаутом (пользовательский контекст в HealthChecker) |
| HEAD-запрос не поддерживается бэкендом | Падаем на GET с закрытием тела — O(1) по ресурсам |
| Новая имплементация VectorStore вне репозитория (у пользователя) ломается | pre-1.0 — принимаем. Документируем в CHANGELOG / release notes |

## Rollout и compatibility

- Все публичные интерфейсы расширяются новым методом — пользовательские реализации вне репозитория перестанут компилироваться. Это pre-1.0 breaking change, ожидаемый.
- Новые публичные типы (`HealthChecker`, `LivenessHandler`, `ReadinessHandler`, `StartupHandler`) — только добавление.
- Специальных rollout-действий не требуется.

## Проверка

- `go vet ./...` — AC-001..AC-003
- `go test ./internal/domain/...` — не ломает существующие
- `go test ./internal/infrastructure/vectorstore/...` — AC-004 + не ломает
- `go test ./internal/infrastructure/embedder/...` — не ломает
- `go test ./internal/infrastructure/llm/...` — не ломает
- `go test ./internal/infrastructure/resilience/...` — AC-008..AC-009
- `go test ./pkg/draftrag/...` — AC-005..AC-007
- Новые unit-test'ы: `TestInMemoryStore_Health`, `TestHealthChecker_AggregatesErrors`, `TestLivenessHandler_Returns200`, `TestReadinessHandler_Healthy_Returns200`, `TestReadinessHandler_Unhealthy_Returns503`, `TestRetryEmbedder_Health_Delegates`, `TestRetryEmbedder_Health_CBOpen`

## Соответствие конституции

нет конфликтов
