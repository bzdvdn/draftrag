# Prod Issues — Пакетное исправление production-проблем

## Scope Snapshot

- In scope: семь независимых улучшений production-готовности библиотеки draftRAG: token bucket rate limiter для LLM/Embedder, fallback-цепочка LLM, Health-интерфейс на store/LLM/embedder, coverage CI, фикс Go-версии в examples-smoke, примеры для semantic chunking и sub-query decomposition, реализация VectorStore для Pinecone.
- Out of scope: любые изменения domain-интерфейсов, ломающие обратную совместимость; встроенный HTTP-сервер; CLI-инструменты.

## Цель

Разработчики, использующие draftRAG в production-среде, получают:
- клиентский rate limiter для предотвращения 429 ошибок от LLM/Embedder API;
- fallback-цепочку LLM-провайдеров для graceful degradation при outage;
- Health-интерфейс и HTTP-handler'ы для K8s liveness/readiness/startup probes;
- coverage-отчёт в CI;
- корректную сборку examples на Go 1.23+;
- примеры использования semantic chunker и sub-query decomposition;
- готовый VectorStore для Pinecone.

Успех фичи измерим по отсутствию регрессий в существующих тестах, прохождению `go vet`/`golangci-lint` и наличию документации/тестов для каждого нового компонента.

## Основной сценарий

1. Стартовая точка: разработчик обновляет draftRAG до новой версии и настраивает Pipeline с production-требованиями.
2. Основное действие: разработчик оборачивает LLM-провайдер в `NewTokenBucketLLMProvider` с настройками rate limiter'а; настраивает fallback через `NewFallbackLLMProvider`; добавляет `HealthChecker` с компонентами store, LLM, embedder; использует готовые HTTP-handler'ы для K8s probes.
3. Результат: при превышении rate limit'а запросы ожидают токена вместо получения 429; при отказе primary LLM запрос автоматически направляется на secondary/local; K8s probes корректно отражают состояние компонентов.
4. Ошибка/fallback-путь: при отказе всех LLM-провайдеров в цепочке возвращается `ErrAllProvidersFailed`; при недоступности Pinecone — стандартная ошибка VectorStore.

## User Stories

- P1 Story: разработчик может обернуть LLM-провайдер rate limiter'ом, указав tokens/sec и burst — вызовы не превышают лимит API.
- P2 Story: разработчик может настроить цепочку `Primary → Secondary → Local` LLM — при outage primary запрос автоматически обрабатывается secondary.
- P3 Story: разработчик может получить Health-статус всех компонентов Pipeline через единый интерфейс и подключить к K8s probes.
- P4 Story: CI показывает coverage report после каждого push/PR.
- P5 Story: examples-smoke CI собирается на Go 1.23+.
- P6 Story: в README и examples/ есть работающие примеры semantic chunking и sub-query decomposition.
- P7 Story: приложение может использовать Pinecone как VectorStore через тот же интерфейс `domain.VectorStore`.

## MVP Slice

Наименьший независимый срез — rate-limiting-llm + graceful-degradation + health-check-interface: эти три изменения дают разработчику production-ready resilience. Первый implementation pass закрывает AC-001–AC-015.

## First Deployable Outcome

После первого implementation pass:
- проходят `go vet ./...`, `golangci-lint`, существующие тесты;
- `go test -race -count=1 ./...` зелёный;
- rate limiter, fallback, Health HTTP-handler'ы и Pinecone VectorStore имеют unit-тесты;
- CI workflow публикует coverage report artifact;
- examples-smoke CI билдится на Go 1.23.x.

## Scope

- Token bucket rate limiter для LLM-провайдера (обёртка `LLMProvider`) и embedder'а (обёртка `Embedder`) — публичный API `NewTokenBucketLLMProvider` / `NewTokenBucketEmbedder`.
- Fallback-цепочка для LLM-провайдера: `Primary → Secondary → Local` через `NewFallbackLLMProvider`.
- Health-интерфейс: `Health(ctx) error` на `VectorStore`, `LLMProvider`, `Embedder` (уже есть в domain); `HealthChecker` + `LivenessHandler`/`ReadinessHandler`/`StartupHandler` в `pkg/draftrag/health.go`.
- Coverage CI: `go test -race -coverprofile=coverage.out -covermode=atomic` + upload `coverage.out` artifact в `.github/workflows/ci.yml`.
- Фикс examples-smoke: `go-version: "1.23"` вместо `"1.21"` в `.github/workflows/examples-smoke.yml`.
- Примеры: `examples/semantic-chunking/` и `examples/sub-query-decomposition/` + секции в README (русская и английская версии).
- Pinecone VectorStore: реализация `VectorStore` интерфейса через Pinecone REST/gRPC API + конструктор `NewPineconeStore` в `pkg/draftrag/pinecone.go`.

## Контекст

- Все семь под-фич являются независимыми изменениями в разных частях репозитория, но объединены в один feature slug для единого цикла spec → plan → implement → verify.
- Rate limiter, fallback и Health частично реализованы в `internal/infrastructure/resilience/` и `pkg/draftrag/`. Спека фиксирует текущее состояние и завершает недостающие части.
- Pinecone VectorStore — новый infrastructure-компонент, следующий паттерну существующих реализаций (qdrant, chromadb, weaviate).
- CI-изменения (coverage, go-version) не затрагивают библиотечный код.
- Примеры не являются частью библиотечного API и не требуют публичных экспортов.
- Существующий `domain.VectorStore` интерфейс не меняется.

## Зависимости

- Реализация Pinecone VectorStore требует HTTP-клиент (стандартный `net/http`) и JSON-сериализацию.
- Rate limiter использует только стандартную библиотеку (`sync`, `time`).
- Fallback не добавляет внешних зависимостей.
- Health HTTP-handler'ы используют `net/http` (стандартная библиотека).
- Pinecone API ключ и окружение конфигурируются через опции конструктора (аналогично Qdrant/ChromaDB).
- `none` меж-спековых зависимостей.

## Требования

### rate-limiting-llm

- RQ-001 Система ДОЛЖНА предоставлять `NewTokenBucketLLMProvider(llm, opts)` — декоратор `LLMProvider` с token bucket rate limiter'ом.
- RQ-002 Система ДОЛЖНА предоставлять `NewTokenBucketEmbedder(emb, opts)` — декоратор `Embedder` с token bucket rate limiter'ом.
- RQ-003 При `TokensPerSecond <= 0` декоратор ДОЛЖЕН работать как passthrough без rate limiting.
- RQ-004 При превышении rate limit запрос ДОЛЖЕН ожидать доступный токен (блокирующее ожидание с учётом context cancellation).
- RQ-005 `Health(ctx)` ДОЛЖЕН делегироваться внутреннему провайдеру.
- RQ-029 Система ДОЛЖНА предоставлять `NewTokenBucketStreamingLLMProvider(provider, opts)` — декоратор `StreamingLLMProvider` (реализующий также `LLMProvider`) с token bucket rate limiter'ом, работающим как для `GenerateStream`, так и для `Generate`.

### graceful-degradation

- RQ-006 Система ДОЛЖНА предоставлять `NewFallbackLLMProvider(providers, logger, hooks)` — `LLMProvider` с цепочкой fallback.
- RQ-007 При retryable-ошибке primary-провайдера ДОЛЖЕН автоматически вызываться следующий провайдер в цепочке.
- RQ-008 При не-retryable ошибке ДОЛЖЕН прекращать fallback и возвращать оригинальную ошибку.
- RQ-009 При отказе всех провайдеров ДОЛЖЕН возвращать `ErrAllProvidersFailed`.
- RQ-010 `Stats()` ДОЛЖНА возвращать статистику fallback'ов (сколько раз переключился, последняя ошибка).

### health-check-interface

- RQ-011 Система ДОЛЖНА предоставлять `NewHealthChecker(components...)` для агрегированной проверки здоровья.
- RQ-012 `HealthChecker.Check(ctx)` ДОЛЖЕН проверять все компоненты конкурентно и возвращать агрегированный результат.
- RQ-013 Система ДОЛЖНА предоставлять `LivenessHandler()` — HTTP handler, всегда отвечающий 200 OK.
- RQ-014 Система ДОЛЖНА предоставлять `ReadinessHandler(hc)` — HTTP handler, отвечающий 200/503 по результатам HealthChecker.
- RQ-015 Система ДОЛЖНА предоставлять `StartupHandler(hc)` — HTTP handler, идентичный ReadinessHandler.

### coverage-ci

- RQ-016 CI workflow `ci.yml` ДОЛЖЕН запускать тесты с флагом `-coverprofile=coverage.out -covermode=atomic`.
- RQ-017 CI workflow ДОЛЖЕН сохранять `coverage.out` как artifact для загрузки.
- RQ-018 CI workflow ДОЛЖЕН выводить coverage summary в лог через `go tool cover -func coverage.out`.

### ci-go-version-fix

- RQ-019 `examples-smoke.yml` ДОЛЖЕН использовать `go-version: "1.23"` (вместо `"1.21"`) во всех jobs.

### examples

- RQ-020 Система ДОЛЖНА содержать работающий пример `examples/semantic-chunking/`, демонстрирующий `NewSemanticChunker`.
- RQ-021 Система ДОЛЖНА содержать работающий пример `examples/sub-query-decomposition/`, демонстрирующий `QueryDecomposer` через `SearchBuilder.SubDecompose()`.
- RQ-022 README (русская и английская версии) ДОЛЖНЫ содержать секции или ссылки на новые примеры.

### pinecone-vectorstore

- RQ-023 Система ДОЛЖНА предоставлять `NewPineconeStore(opts)` — конструктор `VectorStore` для Pinecone.
- RQ-024 Pinecone VectorStore ДОЛЖЕН поддерживать `Upsert`, `Delete`, `Search` с метрикой cosine similarity.
- RQ-025 Pinecone VectorStore ДОЛЖЕН поддерживать `CollectionManager` (Create/Delete/Exists коллекции).
- RQ-026 Pinecone VectorStore ДОЛЖЕН поддерживать `Closer` (закрытие HTTP-клиента).
- RQ-027 Pinecone VectorStore ДОЛЖЕН поддерживать `Health(ctx)` — проверка доступности индекса через describe index stats.
- RQ-028 Конфигурация ДОЛЖНА приниматься через `PineconeOptions` struct (APIKey, Environment, ProjectID, IndexName, Cloud, Region, Dimension).

## Вне scope

- Встроенный HTTP-сервер или фреймворк — пользователь сам регистрирует handler'ы в своём HTTP-роутере.
- Rate limiter на основе скользящего окна или leaky bucket — только token bucket.
- Fallback для Embedder'а — только для LLMProvider.
- Поддержка Pinecone gRPC API — только REST.
- Pinecone Namespace support (все операции в namespace по умолчанию).
- Интеграционные тесты Pinecone в CI (требуют реального индекса).
- Покрытие CI для нового кода (требования RQ-016–018 относятся к общему CI workflow, не к coverage отдельных под-фич).
- Изменение интерфейсов `domain.VectorStore`, `domain.LLMProvider`, `domain.Embedder`.

## Критерии приемки

### AC-001 TokenBucketLLMProvider блокирует при превышении rate

- Почему это важно: защита от 429 ошибок без изменения клиентского кода.
- **Given** LLM-провайдер с rate limiter'ом 10 tokens/sec, burst=1
- **When** 2 запроса отправлены с интервалом < 100ms
- **Then** второй запрос ожидает токен и завершается успешно (без ошибки rate limit)
- Evidence: вызовы выполняются последовательно, оба возвращают успешный результат.

### AC-002 TokenBucketLLMProvider passthrough при rate=0

- Почему это важно: нулевая конфигурация не должна ломать существующий код.
- **Given** `TokensPerSecond=0`
- **When** создан `NewTokenBucketLLMProvider`
- **Then** провайдер работает как passthrough без блокировок
- Evidence: вызовы проходят немедленно, `TokensPerSecond()` возвращает 0.

### AC-003 FallbackLLMProvider переключается на secondary

- Почему это важно: автоматическая обработка outage LLM.
- **Given** два LLM-провайдера: primary выкидывает retryable-ошибку, secondary работает
- **When** вызван `Generate`
- **Then** результат возвращается от secondary
- Evidence: `Stats().FallbackCount > 0`, ответ от secondary.

### AC-004 FallbackLLMProvider возвращает ошибку при отказе всех

- Почему это важно: caller должен знать, что ни один провайдер не сработал.
- **Given** все провайдеры в цепочке выкидывают retryable-ошибки
- **When** вызван `Generate`
- **Then** возвращается `ErrAllProvidersFailed`
- Evidence: `errors.Is(err, ErrAllProvidersFailed) == true`.

### AC-005 HealthChecker возвращает unhealthy при ошибке компонента

- Почему это важно: K8s readiness probe должен корректно отражать состояние.
- **Given** HealthChecker с компонентом, чей `Health()` возвращает ошибку
- **When** вызван `Check(ctx)`
- **Then** `Healthy == false`, `Error` содержит имя проблемного компонента
- Evidence: `result.Healthy == false`, `result.Error` не пуст.

### AC-006 LivenessHandler всегда отвечает 200

- Почему это важно: K8s liveness probe не должен зависеть от состояния зависимостей.
- **Given** любой экземпляр `LivenessHandler()`
- **When** выполнен HTTP GET запрос
- **Then** status code 200, тело "OK"
- Evidence: HTTP response 200 OK.

### AC-007 Pinecone VectorStore: Upsert и Search

- Почему это важно: базовый контракт VectorStore для Pinecone.
- **Given** PineconeStore с настроенным индексом
- **When** выполнен Upsert чанка с embedding'ом, затем Search по вектору
- **Then** результат поиска содержит проиндексированный чанк
- Evidence: `Search` возвращает `RetrievalResult` с непустым `Chunks`.

### AC-008 coverage CI генерирует отчёт

- Почему это важно: отслеживание тестового покрытия в CI.
- **Given** CI workflow `ci.yml`
- **When** запущен `go test` с `-coverprofile`
- **Then** `coverage.out` сохранён как artifact, coverage summary в логе
- Evidence: artifact `coverage.out` доступен после завершения workflow.

### AC-009 examples-smoke билдится на Go 1.23

- Почему это важно: сборка examples должна работать на актуальной версии Go.
- **Given** `examples-smoke.yml`
- **When** workflow выполняет `go build ./examples/...`
- **Then** используется `go-version: "1.23"`, билд проходит без ошибок
- Evidence: CI job Examples Build завершается успешно с Go 1.23.x.

### AC-010 Пример semantic-chunking работает

- Почему это важно: пользователи могут изучить semantic chunker на практике.
- **Given** `examples/semantic-chunking/` с `main.go`
- **When** запущен `go run .` (или эквивалент через shared/mock)
- **Then** пример выводит результат чанкинга текста semantic chunker'ом
- Evidence: stdout содержит ожидаемый вывод (чанки с embedding'ами).

### AC-011 Пример sub-query-decomposition работает

- Почему это важно: пользователи могут изучить sub-query decomposition.
- **Given** `examples/sub-query-decomposition/` с `main.go`
- **When** запущен `go run .` (или эквивалент через shared/mock)
- **Then** пример демонстрирует разбиение запроса на под-вопросы и retrieval
- Evidence: stdout показывает под-вопросы и результаты retrieval.

### AC-012 TokenBucketStreamingLLMProvider ограничивает GenerateStream

- Почему это важно: streaming-вызовы также должны уважать rate limit для предотвращения 429 ошибок.
- **Given** LLM-провайдер с streaming rate limiter'ом 2 tokens/sec, burst=1
- **When** вызван `GenerateStream` 3 раза с интервалом < 500ms
- **Then** вызовы ограничиваются token bucket'ом — не более 2 успешных вызовов в секунду
- Evidence: третий вызов блокируется до появления токена.

### AC-013 TokenBucketStreamingLLMProvider делегирует Generate

- Почему это важно: декоратор должен корректно работать и для не-streaming вызовов.
- **Given** `TokenBucketStreamingLLMProvider` с rate=10, burst=5
- **When** вызван `Generate`
- **Then** вызов проходит rate limiting и делегируется внутреннему `LLMProvider.Generate`
- Evidence: `Generate` возвращает результат, соблюдая token bucket.

## Допущения

- Pinecone REST API стабилен (v2024-10 или актуальная).
- Все изменения сохраняют обратную совместимость публичного API.
- Rate limiter не гарантирует равномерное распределение между несколькими инстансами (per-process).
- Fallback цепочка вызывается последовательно (не параллельно).
- CI инфраструктура GitHub Actions доступна и использует ubuntu-latest.
- Go 1.23+ — минимальная поддерживаемая версия (CONSTITUTION.md: Go 1.23+).
- Pinecone VectorStore не требует транзакционной поддержки (не реализует `TransactionalDocumentStore`).

## Критерии успеха

- SC-001 Все существующие тесты проходят без изменений: `go test -race -count=1 ./...` зелёный.
- SC-002 Новый код имеет unit-тесты с покрытием ≥70%.
- SC-003 `golangci-lint` не выдаёт новых предупреждений.
- SC-004 `go vet ./...` не выдаёт ошибок.

## Краевые случаи

- Token bucket: burst=0 (интерпретируется как burst=rate), rate < 0 (возвращается ошибка конструктора).
- Fallback: пустой список провайдеров (ошибка конструктора), один провайдер в списке (passthrough).
- HealthChecker: пустой список компонентов (всегда healthy), nil context (panic).
- Pinecone: неверный API ключ (ошибка при Health/операции), несуществующий индекс (ошибка при Upsert/Search), пустой embedding (ошибка валидации).
- Examples: отсутствие .env файла, переменных окружения (использовать default/mock).
- CI: покрытие может упасть из-за добавления нового кода без тестов — пайплайн не фейлится по % покрытия, только генерирует отчёт.

## Открытые вопросы

- Нужен ли rate limiter для streaming? — Да, решено. `TokenBucketStreamingLLMProvider` реализован как декоратор `StreamingLLMProvider` с token bucket rate limiter'ом (см. RQ-029).
- Должен ли `FallbackLLMProvider` поддерживать `StreamingLLMProvider`? — Уже есть отдельный `FallbackStreamingLLMProvider`.
- Pinecone: использовать Rest API или gRPC? — REST (минимальные зависимости, gRPC требует дополнительных протобафов).
- Pinecone: поддержка sparse vectors для hybrid search? — Не входит в scope, только dense vectors.
- `none` — дополнительных уточнений не требуется.
