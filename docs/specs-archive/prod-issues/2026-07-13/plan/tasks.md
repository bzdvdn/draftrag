# Prod Issues — Задачи

## Phase Contract

Inputs: plan.md, spec.md, REPOSITORY_MAP.md, существующий код.
Outputs: упорядоченные задачи с покрытием AC-001–AC-011.
Stop if: задачи расплывчаты — нет, spec и plan детальны.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `.github/workflows/ci.yml` | T1.1 |
| `.github/workflows/examples-smoke.yml` | T1.2 |
| `pkg/draftrag/health_test.go` | T2.1 |
| `internal/infrastructure/resilience/*.go` (existing tests) | T2.2 |
| `pkg/draftrag/pinecone.go` | T3.1 |
| `internal/infrastructure/vectorstore/pinecone.go` | T3.1 |
| `internal/infrastructure/vectorstore/pinecone_test.go` | T3.2 |
| `examples/semantic-chunking/main.go` | T3.3 |
| `examples/sub-query-decomposition/main.go` | T3.4 |
| `README.md` | T4.1 |
| `README.ru.md` | T4.1 |
| `internal/infrastructure/resilience/ratelimit_streaming.go` | T1.3 |
| `pkg/draftrag/ratelimit.go` | T1.3 |

## Implementation Context

- **Цель MVP**: coverage CI + Go-version fix + Health tests — три изменения, дающие немедленный observable proof.
- **Границы приемки**: AC-005, AC-006, AC-008, AC-009 (MVP); AC-001–AC-004, AC-007, AC-010, AC-011 (полный scope).
- **Ключевые правила**:
  - Не менять domain-интерфейсы (`internal/domain/interfaces.go`).
  - Следовать паттерну Qdrant/ChromaDB для Pinecone.
  - Примеры используют только `pkg/draftrag` + `examples/shared`.
  - CI-workflow не фейлится по % покрытия.
- **Инварианты данных/домена**: VectorStore контракт не меняется; все конструкторы принимают opts struct.
- **Контракты/протокол**: Pinecone REST API (v2024-10); token bucket — `sync`, `time`; fallback — последовательный вызов.
- **Proof signals**: `go test ./...` зелёный; coverage artifact создаётся; pinecone unit-test с mock HTTP; примеры запускаются с LLM_PROVIDER=mock.
- **Вне scope**: gRPC Pinecone, namespace support, интеграционные тесты Pinecone, fallback для embedder.

## Фаза 1: Основа

Цель: подготовить CI-инфраструктуру и исправить Go-version для стабильного билда examples.

- [x] T1.1 **Добавить coverage в CI workflow** (AC-008)
  - Изменить `go test -race -count=1 ./...` на `go test -race -coverprofile=coverage.out -covermode=atomic ./...` в `.github/workflows/ci.yml`.
  - Добавить шаг `Upload coverage artifact` (actions/upload-artifact@v4, path: coverage.out).
  - Добавить шаг `Coverage summary`: `go tool cover -func coverage.out`.
  - Touches: `.github/workflows/ci.yml`
  - AC: AC-008 (RQ-016, RQ-017, RQ-018)

- [x] T1.2 **Исправить Go-version в examples-smoke** (AC-009)
  - Изменить `go-version: "1.21"` на `go-version: "1.23"` в двух местах `.github/workflows/examples-smoke.yml` (jobs: examples-build и examples-smoke).
  - Touches: `.github/workflows/examples-smoke.yml`
  - AC: AC-009 (RQ-019)

- [x] T1.3 **Добавить streaming rate limiter для LLM** (AC-012, AC-013)
  - Создать `internal/infrastructure/resilience/ratelimit_streaming.go`:
    - Структура `tokenBucketStreamingLLMProvider` с полями: inner (domain.StreamingLLMProvider), bucket (*TokenBucket)
    - `Generate(ctx, prompt, opts)` — делегирует inner.Generate через token bucket
    - `GenerateStream(ctx, prompt, opts)` — делегирует inner.GenerateStream через token bucket
    - `Health(ctx)` — делегирует inner.Health
  - Добавить публичный конструктор в `pkg/draftrag/ratelimit.go`:
    - `type TokenBucketStreamingLLMOptions struct` (публичная версия, extends аналог из NewTokenBucketLLMProvider)
    - `NewTokenBucketStreamingLLMProvider(provider StreamingLLMProvider, opts TokenBucketStreamingLLMOptions) (*TokenBucketStreamingLLMProvider, error)`
  - Touches: `internal/infrastructure/resilience/ratelimit_streaming.go`, `pkg/draftrag/ratelimit.go`
  - AC: AC-012 (RQ-029), AC-013

## Фаза 2: MVP Slice

Цель: проверить работоспособность существующих компонентов resilience + добавить тесты HealthChecker.

- [x] T2.1 **Добавить unit-тесты для HealthChecker и HTTP-handler'ов** (AC-005, AC-006)
  - Создать `pkg/draftrag/health_test.go`.
  - Тесты:
    - `TestLivenessHandler` — GET → 200, тело "OK"
    - `TestReadinessHandlerHealthy` — все компоненты здоровы → 200, `healthy: true`
    - `TestReadinessHandlerUnhealthy` — компонент с ошибкой → 503, `healthy: false`
    - `TestStartupHandler` — идентично ReadinessHandler
    - `TestHealthCheckerCheck` — проверка агрегации ошибок, пустой список, nil panic
    - `TestHealthCheckerEmptyComponents` — всегда healthy
    - `TestHealthCheckerContextCancellation` — при отменённом контексте возвращает ошибку
  - Использовать `net/http/httptest` для HTTP-тестов.
  - Touches: `pkg/draftrag/health_test.go`
  - AC: AC-005, AC-006 (RQ-011, RQ-012, RQ-013, RQ-014, RQ-015)

- [x] T2.2 **Верифицировать существующие тесты rate limiter и fallback** (AC-001, AC-002, AC-003, AC-004)
  - Запустить `go test -race -count=1 ./internal/infrastructure/resilience/...`.
  - Подтвердить прохождение:
    - `ratelimit_llm_test.go` — AC-001 (блокировка при превышении rate), AC-002 (passthrough)
    - `ratelimit_embedder_test.go` — RQ-002, RQ-004
    - `tokenbucket_test.go` — core token bucket
    - `fallback_llm_test.go` — AC-003 (переключение на secondary), AC-004 (все отказали)
    - `fallback_streaming_test.go` — если существует
  - Если тесты не проходят — зафиксировать regression и исправить.
  - Touches: `internal/infrastructure/resilience/*_test.go` (read-only verify, или fix)
  - AC: AC-001, AC-002, AC-003, AC-004 (RQ-001–RQ-010)

## Фаза 3: Основная реализация

Цель: реализовать Pinecone VectorStore и примеры использования.

- [x] T3.1 **Реализовать Pinecone VectorStore (infrastructure) + публичный конструктор** (AC-007)
  - Создать `internal/infrastructure/vectorstore/pinecone.go`:
    - Структура `pineconeStore` с полями: apiKey, environment, projectID, indexName, cloud, region, dimension, httpClient
    - `PineconeOptions` struct (APIKey, Environment, ProjectID, IndexName, Cloud, Region, Dimension)
    - `NewPineconeStore(ctx, opts)` — конструктор, проверяет обязательные поля
    - `Upsert(ctx, chunk)` — POST `https://{IndexName}-{ProjectID}.svc.{Environment}.pinecone.io/vectors/upsert`
    - `Delete(ctx, id)` — POST `https://.../vectors/delete`
    - `Search(ctx, embedding, topK)` — POST `https://.../query` с включением метаданных
    - `Health(ctx)` — GET `https://.../describe_index_stats`
    - `CreateCollection(ctx)` — POST `https://api.pinecone.io/collections` (если поддерживается)
    - `DeleteCollection(ctx)` — DELETE `https://api.pinecone.io/collections/{name}`
    - `CollectionExists(ctx)` — GET `https://api.pinecone.io/collections/{name}`
    - `Close()` — закрытие `http.Client.Transport` (CloseIdleConnections)
  - Создать `pkg/draftrag/pinecone.go`:
    - `type PineconeOptions` struct (публичная версия)
    - `NewPineconeStore(ctx, opts PineconeOptions) (*PineconeStore, error)` — обёртка над internal-конструктором
    - `type PineconeStore struct` с встроенным internal-типом
  - Touches: `internal/infrastructure/vectorstore/pinecone.go`, `pkg/draftrag/pinecone.go`
  - AC: AC-007 (RQ-023, RQ-024, RQ-025, RQ-026, RQ-027, RQ-028)

- [x] T3.2 **Добавить unit-тесты Pinecone VectorStore** (AC-007)
  - Создать `internal/infrastructure/vectorstore/pinecone_test.go`.
  - Использовать `net/http/httptest.NewServer` для mock Pinecone REST API.
  - Тесты:
    - `TestPineconeUpsertAndSearch` — upsert вектора → search возвращает результат
    - `TestPineconeDelete` — upsert → delete → search не находит
    - `TestPineconeHealth` — describe_index_stats возвращает OK
    - `TestPineconeHealthFail` — сервер недоступен → Health error
    - `TestPineconeInvalidAPIKey` — 401 → возвращается ошибка
    - `TestPineconeCollectionCreateDeleteExists` — CollectionManager (опционально, через mock)
    - `TestPineconeEmptyEmbedding` — пустой вектор → ошибка валидации
  - Touches: `internal/infrastructure/vectorstore/pinecone_test.go`
  - AC: AC-007 (RQ-024, RQ-025, RQ-026, RQ-027)

- [x] T3.3 **Создать пример semantic-chunking** (AC-010)
  - Создать `examples/semantic-chunking/main.go` по шаблону `examples/memory/main.go`.
  - Демонстрировать `NewSemanticChunker` с `SemanticChunkerOptions`.
  - Использовать `shared.NewMockLLM()` и `shared.NewMockEmbedder(dim)`.
  - Индексировать 3-4 документа, затем показать результат семантического чанкинга.
  - При запуске с `LLM_PROVIDER=mock` выводить детерминированный результат в stdout.
  - Touches: `examples/semantic-chunking/main.go`
  - AC: AC-010 (RQ-020)

- [x] T3.4 **Создать пример sub-query-decomposition** (AC-011)
  - Создать `examples/sub-query-decomposition/main.go` по шаблону `examples/memory/main.go`.
  - Демонстрировать `SearchBuilder.SubDecompose()` с mock-LLM.
  - Показывать под-вопросы и результаты retrieval по каждому под-вопросу.
  - При запуске с `LLM_PROVIDER=mock` — детерминированный вывод.
  - Touches: `examples/sub-query-decomposition/main.go`
  - AC: AC-011 (RQ-021)

## Фаза 4: Проверка

Цель: доказать, что фича работает, обновить документацию.

- [x] T4.1 **Обновить README с секциями примеров** (AC-010, AC-011)
  - В `README.md` добавить секцию «Examples» (или расширить существующую) со ссылками на `examples/semantic-chunking/` и `examples/sub-query-decomposition/`.
  - Краткое описание каждого примера и команда запуска.
  - То же самое для `README.ru.md` (русская версия).
  - Touches: `README.md`, `README.ru.md`
  - AC: AC-010, AC-011 (RQ-022)

- [x] T4.2 **Финальная верификация** (SC-001, SC-002, SC-003, SC-004)
  - `go vet ./...` — без ошибок.
  - `go test -race -count=1 ./...` — зелёный.
  - `go build ./examples/...` — успешно.
  - `golangci-lint` — без новых предупреждений.
  - Touches: весь код фичи
  - AC: SC-001, SC-002, SC-003, SC-004

## Покрытие критериев приемки

- AC-001 -> T2.2
- AC-002 -> T2.2
- AC-003 -> T2.2
- AC-004 -> T2.2
- AC-005 -> T2.1
- AC-006 -> T2.1
- AC-007 -> T3.1, T3.2
- AC-008 -> T1.1
- AC-009 -> T1.2
- AC-010 -> T3.3, T4.1
- AC-011 -> T3.4, T4.1
- AC-012 -> T1.3
- AC-013 -> T1.3

## Заметки

- T1.1, T1.2 и T1.3 можно выполнять параллельно — они независимы.
- T2.2 — read-only verify; если тесты не проходят, завести баг и исправить.
- T3.1 и T3.2 — реализация + тесты Pinecone, рекомендуется выполнять вместе.
- T3.3 и T3.4 — независимы друг от друга и от Pinecone; можно параллелить.
- T4.2 выполняется последней, после всех остальных задач.
