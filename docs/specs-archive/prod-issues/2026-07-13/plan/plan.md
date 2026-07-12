# Prod Issues — План

## Phase Contract

Inputs: spec.md, REPOSITORY_MAP.md, существующий код в `pkg/draftrag/`, `internal/infrastructure/resilience/`, `.github/workflows/`.
Outputs: план реализации, задачи с покрытием AC-001–AC-011.
Stop if: spec расплывчата — нет, spec детальна и стабильна.

## Цель

Восемь независимых улучшений production-готовности: (1) coverage CI, (2) Go-version fix в examples-smoke, (3) примеры semantic-chunking/sub-query-decomposition, (4) Pinecone VectorStore, (5) верификация существующих тестов rate limiter, fallback, health checker, (6) документация, (7) streaming rate limiter для GenerateStream.

Rate limiter, fallback, health checker уже реализованы в `pkg/draftrag/` и `internal/infrastructure/resilience/`. План закрывает пробелы: тесты HealthChecker, Pinecone VectorStore, CI-инфраструктура, примеры, документация.

## MVP Slice

Минимальный независимый инкремент — **CI + Go-version + Health tests**:
- AC-008 (coverage CI генерирует отчёт)
- AC-009 (examples-smoke билдится на Go 1.23)
- AC-005 (HealthChecker unhealthy при ошибке)
- AC-006 (LivenessHandler всегда 200)

Эти четыре критерия не требуют нового кода библиотеки и дают немедленный наблюдаемый результат в CI.

## First Validation Path

1. Проверить `go test -race -count=1 ./...` — зелёный.
2. Проверить `go vet ./...` — без ошибок.
3. Запустить `go test -coverprofile=coverage.out -covermode=atomic ./... && go tool cover -func coverage.out` — coverage summary в stdout.
4. Убедиться, что `examples/memory/main.go` билдится: `go build ./examples/memory/`.

## Scope

1. **Coverage CI** — `ci.yml`: замена строки `go test -race -count=1 ./...` на `go test -race -coverprofile=coverage.out -covermode=atomic ./...`, добавление upload-artifact и `go tool cover -func coverage.out`.
2. **Go version fix** — `examples-smoke.yml`: `"1.21"` → `"1.23"` в двух местах.
3. **Примеры** — создание `examples/semantic-chunking/main.go` и `examples/sub-query-decomposition/main.go` по шаблону `examples/memory/main.go`.
4. **Pinecone VectorStore** — новый `internal/infrastructure/vectorstore/pinecone.go` + `pkg/draftrag/pinecone.go` + `internal/infrastructure/vectorstore/pinecone_test.go`.
5. **Health check tests** — создание `pkg/draftrag/health_test.go`.
6. **Документация** — обновление README.md и README.ru.md.
7. **Streaming rate limiter** — создание `TokenBucketStreamingLLMProvider` в `internal/infrastructure/resilience/` + публичный конструктор `NewTokenBucketStreamingLLMProvider` в `pkg/draftrag/ratelimit.go`.

**Вне scope**: изменение `domain/interfaces.go`, встроенный HTTP-сервер, gRPC Pinecone, интеграционные тесты Pinecone в CI.

## Performance Budget

`none` — все изменения не вводят новых критических путей с performance-требованиями. Pinecone REST-клиент — стандартный `net/http` без особых требований.

## Implementation Surfaces

| Surface | Статус | Почему участвует |
|---------|--------|------------------|
| `.github/workflows/ci.yml` | существующий | Добавление coverage флагов, upload-artifact, go tool cover |
| `.github/workflows/examples-smoke.yml` | существующий | Go version `"1.21"` → `"1.23"` |
| `examples/semantic-chunking/main.go` | новый | Пример использования NewSemanticChunker |
| `examples/sub-query-decomposition/main.go` | новый | Пример использования QueryDecomposer через SearchBuilder.SubDecompose |
| `internal/infrastructure/vectorstore/pinecone.go` | новый | Реализация VectorStore для Pinecone REST API |
| `internal/infrastructure/vectorstore/pinecone_test.go` | новый | Unit-тесты Pinecone (mock HTTP) |
| `pkg/draftrag/pinecone.go` | новый | Публичный конструктор NewPineconeStore |
| `pkg/draftrag/health_test.go` | новый | Тесты HealthChecker + HTTP handlers |
| `README.md` | существующий | Секции примеров semantic-chunking, sub-query-decomposition |
| `README.ru.md` | существующий | Русская версия секций примеров |
| `internal/infrastructure/resilience/ratelimit_streaming.go` | новый | Реализация TokenBucketStreamingLLMProvider |
| `pkg/draftrag/ratelimit.go` | существующий | Публичный конструктор NewTokenBucketStreamingLLMProvider |

## Bootstrapping Surfaces

`none` — все необходимые структуры (директории пакетов, CI workflow, shared mock) уже существуют.

## Влияние на архитектуру

- **Локальное**: Pinecone VectorStore добавляет новую реализацию `domain.VectorStore` без изменения интерфейса. Публичный конструктор `NewPineconeStore` следует паттерну Qdrant/ChromaDB.
- **Интеграции**: Pinecone использует REST API, не добавляет новых зависимостей (только `net/http`, `encoding/json`).
- **CI**: изменения не затрагивают библиотечный код; coverage-флаг не ломает существующие тесты.
- **Совместимость**: полная обратная совместимость — изменения только additive.

## Acceptance Approach

- **AC-001** (TokenBucketLLMProvider блокирует) → уже реализовано в `internal/infrastructure/resilience/ratelimit_llm_test.go` + `tokenbucket_test.go`. Проверка: `go test ./internal/infrastructure/resilience/...`.
- **AC-002** (passthrough при rate=0) → уже реализовано. Проверка: тест `TestTokenBucketPassthrough` (или аналог).
- **AC-003** (Fallback переключается на secondary) → уже реализовано в `internal/infrastructure/resilience/fallback_llm_test.go`. Проверка: тесты проходят.
- **AC-004** (Fallback возвращает ошибку при отказе всех) → уже реализовано. Проверка: `TestFallbackAllFailed` проходит.
- **AC-005** (HealthChecker unhealthy) → поверхность: `internal/infrastructure/resilience/` (если есть HealthChecker-тесты) + новый `pkg/draftrag/health_test.go`. Проверка: `HealthChecker` с компонентом, возвращающим ошибку → `Healthy == false`.
- **AC-006** (LivenessHandler 200) → поверхность: `pkg/draftrag/health_test.go`. Проверка: HTTP GET → 200 OK.
- **AC-007** (Pinecone Upsert/Search) → поверхность: `pkg/draftrag/pinecone.go`, `internal/infrastructure/vectorstore/pinecone.go`, `internal/infrastructure/vectorstore/pinecone_test.go`. Проверка: unit-тест с mock HTTP-сервером.
- **AC-008** (coverage CI) → поверхность: `.github/workflows/ci.yml`. Проверка: substrate workflow с `-coverprofile`.
- **AC-009** (examples-smoke Go 1.23) → поверхность: `.github/workflows/examples-smoke.yml`. Проверка: `go-version: "1.23"` в job.
- **AC-010** (пример semantic-chunking работает) → поверхность: `examples/semantic-chunking/main.go`. Проверка: `cd examples/semantic-chunking && LLM_PROVIDER=mock go run .` — stdout содержит результат чанкинга.
- **AC-011** (пример sub-query-decomposition работает) → поверхность: `examples/sub-query-decomposition/main.go`. Проверка: `cd examples/sub-query-decomposition && LLM_PROVIDER=mock go run .` — stdout показывает под-вопросы.
- **AC-012** (TokenBucketStreamingLLMProvider блокирует GenerateStream) → поверхность: `internal/infrastructure/resilience/ratelimit_streaming.go`, `pkg/draftrag/ratelimit.go`. Проверка: unit-тест с превышением rate — третий вызов блокируется.
- **AC-013** (TokenBucketStreamingLLMProvider делегирует Generate) → поверхность: `internal/infrastructure/resilience/ratelimit_streaming.go`, `pkg/draftrag/ratelimit.go`. Проверка: `Generate` проходит rate limiting и возвращает результат.

## Данные и контракты

`data-model.md` не требуется — ни одна под-фича не меняет data model:
- CI/Go-version: только workflow-файлы.
- Примеры: не часть API, используют существующие публичные конструкторы.
- Pinecone: следует существующему контракту `domain.VectorStore` (Upsert/Delete/Search/Health) + `CollectionManager` + `Closer`.
- Rate limiter/Fallback/Health: реализованы, интерфейсы не меняются.

## Стратегия реализации

### DEC-001: CI-first sequencing

**Why**: Coverage CI и Go-version fix — технически тривиальны, независимы от остального кода, дают немедленный observable proof и не требуют тестов.
**Tradeoff**: Отсутствует — изменения CI ортогональны библиотеке.
**Affects**: `.github/workflows/ci.yml`, `.github/workflows/examples-smoke.yml`.
**Validation**: PR содержит diff workflow, проверяемый review.

### DEC-002: Pinecone следует паттерну Qdrant

**Why**: Существующие VectorStore (Qdrant, ChromaDB, Weaviate) используют единый паттерн: REST HTTP-клиент в `internal/infrastructure/vectorstore/` + публичный конструктор в `pkg/draftrag/` + unit-тесты с mock HTTP-сервером.
**Tradeoff**: REST-only, без gRPC; без namespace support (spec: out of scope).
**Affects**: `internal/infrastructure/vectorstore/pinecone.go`, `pkg/draftrag/pinecone.go`, `internal/infrastructure/vectorstore/pinecone_test.go`.
**Validation**: Unit-тест с `httptest.NewServer` верифицирует Upsert → Search → Delete → Health.

### DEC-003: Health-тесты в `pkg/draftrag/` (не в `internal/`)

**Why**: Публичные handler'ы (`LivenessHandler`, `ReadinessHandler`, `StartupHandler`) и `HealthChecker` экспортируются из `pkg/draftrag/`. Тестировать их в том же пакете изоморфно — можно использовать `httptest.NewRecorder`.
**Tradeoff**: Тесты не покрывают внутреннюю конвертацию domain-Health в ComponentHealth, но spec не требует этого.
**Affects**: `pkg/draftrag/health_test.go`.
**Validation**: `go test ./pkg/draftrag/... -run TestHealth`.

### DEC-004: Примеры без `.env.example` (только mock)

**Why**: Примеры должны работать без внешних API. По spec, отсутствие .env → использование mock/default. Shared пакет предоставляет `NewMockLLM()` и `NewMockEmbedder(dim)`.
**Tradeoff**: Примеры не демонстрируют реальный semantic chunker (требует LLM) — используют shared mock для детерминированного вывода.
**Affects**: `examples/semantic-chunking/main.go`, `examples/sub-query-decomposition/main.go`.
**Validation**: `LLM_PROVIDER=mock EMBEDDING_DIM=1536 go run .` завершается успешно.

### DEC-005: Streaming rate limiter следует паттерну TokenBucketLLMProvider

**Why**: Существующий `TokenBucketLLMProvider` уже реализован и протестирован. Streaming-вариант использует ту же `TokenBucket` и тот же механизм блокирующего ожидания.
**Tradeoff**: Небольшая дупликация кода (две обёртки вместо одной с проверкой типа), но следует существующему паттерну раздельных имплементаций в кодовой базе.
**Affects**: `internal/infrastructure/resilience/ratelimit_streaming.go`, `pkg/draftrag/ratelimit.go`.
**Validation**: Unit-тесты для `GenerateStream` (блокировка при превышении rate) и `Generate` (делегирование).

## Incremental Delivery

### MVP (Первая ценность)

- CI: coverage flag + upload artifact + go tool cover summary
- Go-version: `"1.21"` → `"1.23"` в examples-smoke
- Health tests: `health_test.go` (AC-005, AC-006)
- Streaming rate limiter: `TokenBucketStreamingLLMProvider` (AC-012, AC-013)
- Проверка: `go test ./pkg/draftrag/...` + CI workflow diff

### Итеративное расширение

1. **Pinecone VectorStore** — после MVP. Зависит от понимания паттерна Qdrant. AC-007.
2. **Примеры** — после Pinecone. Независимы от MVP. AC-010, AC-011.
3. **Документация** — последней, после готовых примеров. AC-010, AC-011 (через ссылки в README).

## Порядок реализации

1. **Фаза 1 (Основа)**: CI + Go-version + Streaming rate limiter (T1.1–T1.3). Независимо, параллельно.
2. **Фаза 2 (MVP)**: Health tests (T2.1). Верификация существующих тестов rate limiter/fallback (T2.2).
3. **Фаза 3 (Основная)**: Pinecone VectorStore (T3.1–T3.2). Примеры (T3.3–T3.4).
4. **Фаза 4 (Проверка)**: Документация README (T4.1). Полный `go test ./...` + `go vet ./...` (T4.2).

Параллельно можно:
- CI + Go-version + Streaming rate limiter + Health tests (фазы 1–2)
- Pinecone после Health tests
- Примеры после Pinecone или параллельно (независимы)

## Риски

- **Риск 1**: Pinecone REST API изменится. **Mitigation**: используем только стабильные эндпоинты (describe index stats, upsert, query, delete). Unit-тесты изолированы mock-сервером.
- **Риск 2**: Примеры не скомпилируются из-за отсутствия экспортов. **Mitigation**: примеры используют только `pkg/draftrag` и `examples/shared` — оба пакета существуют.
- **Риск 3**: HealthChecker уже косвенно тестируется через resilience-тесты, но прямых тестов HTTP-handler'ов нет. **Mitigation**: явное создание `health_test.go` в фазе MVP.

## Rollout и compatibility

Специальных rollout-действий не требуется. Все изменения additive:
- CI-workflow меняется только для новых коммитов.
- Pinecone — новый компонент, существующий код не затрагивает.
- Go-version в examples-smoke — только CI-джобы, не влияет на библиотечный код.
- Примеры — новая директория, существующие примеры не меняются.

## Проверка

| Шаг проверки | Команда | Подтверждает |
|-------------|---------|--------------|
| Unit-тесты | `go test -race -count=1 ./...` | SC-001, AC-001–AC-007 |
| Vet | `go vet ./...` | SC-004 |
| Build examples | `go build ./examples/...` | AC-010, AC-011 |
| Coverage | `go test -coverprofile=coverage.out -covermode=atomic ./... && go tool cover -func coverage.out` | AC-008 |
| Go version | проверить `grep "1.23" .github/workflows/examples-smoke.yml` | AC-009 |
| Health test | `go test ./pkg/draftrag/... -run TestHealth` | AC-005, AC-006 |
| Memory example | `cd examples/memory && LLM_PROVIDER=mock EMBEDDING_DIM=1536 go run .` | SC-001 (no regression) |

## Соответствие конституции

нет конфликтов. Go 1.23+ — минимальная поддерживаемая версия (CONSTITUTION.md). Все изменения не ломают обратную совместимость.
