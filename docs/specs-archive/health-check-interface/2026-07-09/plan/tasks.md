# Health Check Interface Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/infrastructure/vectorstore/memory.go` | T1.2 |
| `internal/infrastructure/vectorstore/{pgvector,qdrant,chromadb,weaviate,milvus}.go` | T3.1 |
| `internal/infrastructure/embedder/{ollama,openai_compatible}.go` | T3.2 |
| `internal/infrastructure/llm/{openai_chat,anthropic,ollama}.go` | T3.3 |
| `internal/infrastructure/resilience/{embedder,llm}.go` | T3.4 |
| `pkg/draftrag/{mistral_llm,deepseek_llm,anthropic_llm,cached_embedder}.go` | T3.5 |
| `pkg/draftrag/health.go` (новый) | T2.1 |
| `pkg/draftrag/draftrag.go` | T2.1 |
| `internal/infrastructure/resilience/{embedder,llm}_test.go` | T4.1 |
| `pkg/draftrag/*_test.go` | T4.1 |
| `internal/infrastructure/vectorstore/*_test.go` | T4.2 |

## Implementation Context

- **Цель MVP:** `Health(ctx) error` в 3-х интерфейсах + InMemoryStore.Health + HealthChecker + HTTP-handler'ы — закрытый контур для K8s readiness/liveness
- **Границы приёмки:** AC-001..AC-009
- **Инварианты:**
  - `Health` на базовом интерфейсе, а не optional capability — все реализации обязаны реализовать (DEC-001)
  - Network-health: `(*sql.DB).PingContext` для pgvector, HEAD/GET baseURL для всех HTTP-бэкендов (DEC-002)
  - In-memory: всегда `nil` без проверок
  - Retry-обёртки: делегируют без retry (1 вызов), CB open → ошибка (DEC-003 в resilience)
  - Wrappers (mistral, deepseek, anthropic, cached): делегируют impl.Health без своей логики (DEC-005)
- **Контракты:**
  - `HealthChecker` — struct, не интерфейс (DEC-003)
  - `LivenessHandler` — всегда 200 (DEC-004)
  - `ReadinessHandler` — 200 (все healthy) / 503 + тело (хоть один unhealthy)
- **Ошибки:** `ErrCircuitOpen` уже существует (`internal/infrastructure/resilience/circuitbreaker.go:96`)
- **Proof signals:** `go build ./...`, `go vet ./...`, `go test ./...` без errors; `httptest` для handler'ов
- **Вне scope:** Chunker/Reranker/Hooks/Logger health, интеграционные тесты с реальными бэкендами

## Фаза 1: Основа

Цель: добавить `Health` в интерфейсы и реализовать первый Health на InMemoryStore — компиляция восстанавливается.

- [x] T1.1 Добавить `Health(ctx context.Context) error` в интерфейсы `VectorStore`, `Embedder`, `LLMProvider` в `internal/domain/interfaces.go`. Touches: `internal/domain/interfaces.go` (AC-001, AC-002, AC-003)
- [x] T1.2 Реализовать `Health(ctx context.Context) error` на `InMemoryStore` — возвращает `nil`. Touches: `internal/infrastructure/vectorstore/memory.go` (AC-004)

## Фаза 2: MVP Slice

Цель: HealthChecker + HTTP-handler'ы — минимальная product value (K8s probes с in-memory store).

- [x] T2.1 Создать `pkg/draftrag/health.go`: `HealthChecker` struct (слайс named компонентов), конструктор `NewHealthChecker`/`MustNewHealthChecker`, `Check(ctx) error` (агрегация ошибок, мульти-ошибка, уважает ctx cancellation) + `LivenessHandler`, `ReadinessHandler`, `StartupHandler` (http.HandlerFunc). Touches: `pkg/draftrag/health.go` (AC-005, AC-006, AC-007)
- [x] T2.2 Re-export `HealthChecker` через `pkg/draftrag/draftrag.go` (type alias). Touches: `pkg/draftrag/draftrag.go`

## Фаза 3: Основная реализация

Цель: реализовать Health для всех network-бэкендов, resilience-обёрток и публичных wrapper-типов.

- [x] T3.1 Реализовать `Health(ctx) error` для network VectorStore бэкендов: pgvector (`PingContext`), qdrant (GET /health), chromadb (GET /api/v1/heartbeat), weaviate (GET /v1/.well-known/ready), milvus (GET /v1/health). Все используют существующий http.Client / *sql.DB. Touches: `internal/infrastructure/vectorstore/pgvector.go`, `internal/infrastructure/vectorstore/qdrant.go`, `internal/infrastructure/vectorstore/chromadb.go`, `internal/infrastructure/vectorstore/weaviate.go`, `internal/infrastructure/vectorstore/milvus.go`
- [x] T3.2 Реализовать `Health(ctx) error` для embedder'ов: OllamaEmbedder (HEAD /api/tags или GET base URL), OpenAICompatibleEmbedder (HEAD base URL). Touches: `internal/infrastructure/embedder/ollama.go`, `internal/infrastructure/embedder/openai_compatible.go`
- [x] T3.3 Реализовать `Health(ctx) error` для LLM: OpenAIChatLLM (HEAD base URL), ClaudeLLM (HEAD base URL), OllamaLLM (HEAD /api/tags). Touches: `internal/infrastructure/llm/openai_chat.go`, `internal/infrastructure/llm/anthropic.go`, `internal/infrastructure/llm/ollama.go`
- [x] T3.4 Реализовать `Health(ctx) error` на `RetryEmbedder` и `RetryLLMProvider`: если circuit closed — делегировать `Health()` внутреннему компоненту (без retry, ровно 1 вызов); если circuit open — возвращать `ErrCircuitOpen`. Touches: `internal/infrastructure/resilience/embedder.go`, `internal/infrastructure/resilience/llm.go` (AC-008, AC-009, RQ-007, RQ-008)
- [x] T3.5 Реализовать `Health(ctx) error` на публичных wrapper-типах: `mistralLLM` (→ impl.Health), `deepseekLLM` (→ impl.Health), `anthropicLLM` (→ impl.Health), `CachedEmbedder` (→ inner embedder.Health). Touches: `pkg/draftrag/mistral_llm.go`, `pkg/draftrag/deepseek_llm.go`, `pkg/draftrag/anthropic_llm.go`, `pkg/draftrag/cached_embedder.go`

## Фаза 4: Проверка

Цель: unit-тесты для всех критических AC + финальный verify.

- [x] T4.1 Добавить unit-тесты для HealthChecker, LivenessHandler, ReadinessHandler через `httptest`: агрегация ошибок, liveness 200, readiness 200/503. Добавить тесты для RetryEmbedder.Health и RetryLLMProvider.Health (делегация без retry, CB open). Touches: `pkg/draftrag/health_test.go` (новый), `internal/infrastructure/resilience/embedder_test.go`, `internal/infrastructure/resilience/llm_test.go` (AC-005, AC-006, AC-007, AC-008, AC-009)
- [x] T4.2 Выполнить `go build ./...`, `go vet ./...`, `go test ./...` — все проходят. Убедиться, что существующие тесты не сломаны.

## Покрытие критериев приемки

- AC-001 → T1.1
- AC-002 → T1.1
- AC-003 → T1.1
- AC-004 → T1.2
- AC-005 → T2.1, T4.1
- AC-006 → T2.1, T4.1
- AC-007 → T2.1, T4.1
- AC-008 → T3.4, T4.1
- AC-009 → T3.4, T4.1

## Заметки

- T1.1 — compilation gate: без него ни одна реализация не компилируется, поэтому строго первая
- T1.2..T3.3 можно выполнять параллельно после T1.1
- T3.4 (resilience) — следует после T1.1, но независим от T3.1..T3.3
- T3.5 (wrappers) — зависит от T3.2/T3.3 (нужны Health на внутренних типах)
- T2.1/T2.2 независимы от T3.x, можно параллелить
