# Health Check Interface

## Scope Snapshot

- In scope: Добавление `Health(ctx context.Context) error` в интерфейсы `VectorStore`, `Embedder`, `LLMProvider` + утилита агрегации здоровья + HTTP-handler для K8s probes.
- Out of scope: Встроенный HTTP-сервер, startup probe delay, метрики здоровья, изменение контракта Chunker/Reranker/Hooks/Logger.

## Цель

Пользователь библиотеки (разработчик RAG-сервиса) получает единый стандартный способ проверить доступность каждого компонента (store, LLM, embedder) без изучения внутренних деталей провайдера. Появляется возможность подключить K8s liveness/readiness probes одной строкой кода в своём HTTP-сервере. Успех измеряется тем, что все штатные реализации в репозитории проходят `go vet ./...` и существующие тесты не ломаются.

## Основной сценарий

1. Разработчик конфигурирует Pipeline с конкретными реализациями VectorStore, Embedder, LLMProvider.
2. Для K8s-деплоймента разработчик создаёт `HealthChecker`, передаёт в него компоненты.
3. Разработчик монтирует готовые `http.HandlerFunc` (Liveness, Readiness, Startup) в свой HTTP-сервер.
4. K8s периодически дёргает эти endpoints; при падении зависимости (недоступен PostgreSQL, Qdrant, Anthropic API и т.д.) readiness возвращает 503, и K8s уводит трафик.
5. При старте (пока зависимости не прогрелись) startup probe может вернуть 503, откладывая проверки liveness.
6. Агрегатор опрашивает все зарегистрированные компоненты: если хотя бы один вернул ошибку — общая проверка считается failed.

## User Stories

- none (фича инфраструктурная, пользовательский сценарий один — мониторинг готовности)

## MVP Slice

Наименьший срез: `Health(ctx context.Context) error` как часть контракта `VectorStore`, `Embedder`, `LLMProvider` + реализация `func (m *InMemoryStore) Health(context.Context) error { return nil }`. Без этого не скомпилируется ни одна реализация, и невозможно продемонстрировать closed-loop.

MVP закрывает: AC-001, AC-002, AC-003, AC-004, AC-005.

## First Deployable Outcome

Собранный пакет (`go build ./...`) + одна in-memory проверка. Можно руками дёрнуть `Health()` на `InMemoryStore` и убедиться, что интерфейсы не сломаны.

## Scope

- `internal/domain/interfaces.go` — добавить `Health(ctx context.Context) error` в `VectorStore`, `Embedder`, `LLMProvider`
- `internal/infrastructure/vectorstore/*.go` — реализация Health для каждого backend
- `internal/infrastructure/embedder/*.go` — реализация Health для embedder'ов
- `internal/infrastructure/llm/*.go` — реализация Health для LLM провайдеров
- `internal/infrastructure/resilience/*.go` — реализация Health для retry/circuit-breaker обёрток
- `pkg/draftrag/draftrag.go` — экспорт HealthChecker
- `pkg/draftrag/health.go` (новый файл) — публичный `HealthChecker`, `LivenessHandler`, `ReadinessHandler`, `StartupHandler`
- `internal/domain/interfaces.go` — по желанию: `HealthChecker` интерфейс для агрегации (если требуется расширяемость)

## Контекст

- Репозиторий следует Clean Architecture: интерфейсы в `internal/domain/`, реализации в `internal/infrastructure/`, публичный API в `pkg/draftrag/`.
- Конституция запрещает встраивать HTTP-сервер в библиотеку — предоставляем http.HandlerFunc, пользователь сам монтирует.
- Все публичные операции обязаны принимать `context.Context` — Health не исключение.
- Существующие optional capability интерфейсы (`StreamingLLMProvider`, `VectorStoreWithFilters`) уже расширяют базовые; Health будет на уровне базового контракта.

## Зависимости

- `none` — внешние зависимости не вводятся. Health-реализации могут использовать библиотечные HTTP-клиенты (уже есть в каждом провайдере).

## Требования

- RQ-001 `VectorStore`, `Embedder`, `LLMProvider` ДОЛЖНЫ содержать метод `Health(ctx context.Context) error`. Возвращает `nil` если компонент работает, `error` с описанием проблемы если нет.
- RQ-002 Система ДОЛЖНА предоставлять `HealthChecker`, который принимает список компонентов с именами и проверяет их все за один вызов. Поведение: если хотя бы один компонент вернул ошибку — общий результат failed.
- RQ-003 Система ДОЛЖНА предоставлять конструкторы `http.HandlerFunc` для K8s probes: `LivenessHandler`, `ReadinessHandler`, `StartupHandler`. Все три используют `HealthChecker`: readiness/startup проверяют компоненты, liveness возвращает 200 всегда (библиотека жива, если процесс запущен).
- RQ-004 Каждая штатная реализация VectorStore (pgvector, qdrant, chromadb, weaviate, milvus, memory) ДОЛЖНА реализовать `Health()`. Для network-хранилищ — проверка доступности через ping/health API; для in-memory — всегда `nil`.
- RQ-005 Каждая штатная реализация Embedder (ollama, openai-compatible, cached, retry) ДОЛЖНА реализовать `Health()`. Для network-эмбеддеров — простой запрос к health-эндпоинту (или HEAD на base URL); для cached — делегирует внутреннему embedder; для retry — делегирует wrapped embedder.
- RQ-006 Каждая штатная реализация LLMProvider (anthropic, ollama, openai-chat, openai-compatible, mistral, deepseek, retry) ДОЛЖНА реализовать `Health()`. Для network-провайдеров — проверка доступности API (GET /health или HEAD на base URL).
- RQ-007 Retry-обёртки (`RetryEmbedder`, `RetryLLMProvider`) ДОЛЖНЫ делегировать `Health()` внутреннему компоненту без retry-логики (Health не должен маскировать недоступность).
- RQ-008 `RetryEmbedder`/`RetryLLMProvider` (содержат встроенный circuit breaker) ДОЛЖНЫ: если circuit closed — делегировать `Health()` внутреннему компоненту без retry-логики; если circuit open — возвращать ошибку «circuit breaker open» без обращения к внутреннему компоненту.
- RQ-009 `HealthChecker` ДОЛЖЕН принимать таймаут через контекст (один контекст на всю проверку). Если контекст истёк до ответа всех компонентов — возвращать `context.DeadlineExceeded` как общую ошибку.

## Вне scope

- Модификация интерфейсов `Chunker`, `Reranker`, `Hooks`, `Logger` — у них нет внешних зависимостей, Health для них не имеет смысла (Chunker чисто вычислительный).
- Интеграционные тесты, требующие реального PostgreSQL/Qdrant/etc — достаточно unit-mocks (наличие метода).
- Timeout per-component внутри HealthChecker — один контекст на всех.
- Health check metrics (latency histograms, success counters, gauge) — отдельная фича.
- Graceful shutdown / health-draining — ответственность пользователя и K8s.

## Критерии приемки

### AC-001 VectorStore.Health присутствует

- Почему это важно: базовый контракт, без него ни одна сторона не может положиться на наличие проверки.
- **Given** интерфейс `VectorStore` определён в `internal/domain/interfaces.go`
- **When** происходит компиляция любого кода, использующего `VectorStore`
- **Then** метод `Health(ctx context.Context) error` доступен в интерфейсе
- Evidence: `go vet ./...` проходит, в `interfaces.go` есть `Health(ctx context.Context) error` у `VectorStore`

### AC-002 Embedder.Health присутствует

- **Given** интерфейс `Embedder` определён в `internal/domain/interfaces.go`
- **When** происходит компиляция
- **Then** метод `Health(ctx context.Context) error` доступен у `Embedder`
- Evidence: `go vet ./...` проходит, `Embedder` содержит `Health`

### AC-003 LLMProvider.Health присутствует

- **Given** интерфейс `LLMProvider` определён в `internal/domain/interfaces.go`
- **When** происходит компиляция
- **Then** метод `Health(ctx context.Context) error` доступен у `LLMProvider`
- Evidence: `go vet ./...` проходит, `LLMProvider` содержит `Health`

### AC-004 InMemoryStore.Health возвращает nil

- Почему это важно: in-memory store всегда доступен (нет внешней зависимости), проверка здоровья не должна падать.
- **Given** создан `InMemoryStore`
- **When** вызывается `Health(ctx.Background())`
- **Then** возвращается `nil`
- Evidence: юнит-тест проверяет `err == nil`

### AC-005 HealthChecker агрегирует ошибки

- Почему это важно: пользователь должен иметь единую точку проверки всех компонентов.
- **Given** `HealthChecker` сконфигурирован с двумя компонентами (один healthy, один unhealthy)
- **When** вызывается `HealthChecker.Health(ctx)`
- **Then** возвращается ошибка, содержащая имя unhealthy компонента
- Evidence: юнит-тест проверяет `err != nil` и сообщение включает имя

### AC-006 LivenessHandler возвращает 200

- Почему это важно: Kubernetes liveness probe должна успешно проходить, если процесс жив.
- **Given** `LivenessHandler` создан из `HealthChecker`
- **When** HTTP-запрос к handler
- **Then** статус 200 OK (без проверки зависимостей)
- Evidence: `httptest.NewRecorder()` проверяет `code == 200`

### AC-007 ReadinessHandler возвращает 200/503

- **Given** `ReadinessHandler` создан из `HealthChecker`
- **When** все компоненты здоровы
- **Then** статус 200 OK
- **When** хотя бы один компонент не здоров (возвращает ошибку)
- **Then** статус 503 Service Unavailable + тело с описанием
- Evidence: два `httptest`-теста на оба исхода

### AC-008 RetryEmbedder.Health делегирует без retry

- Почему это важно: Health должен отражать реальное состояние зависимости, retry не должен это маскировать.
- **Given** `RetryEmbedder` оборачивает другой embedder
- **When** вызывается `Health()`
- **Then** вызывается `Health()` на внутреннем embedder ровно один раз (без retry)
- Evidence: mock-embedder с counter, тест проверяет `callCount == 1`

### AC-009 RetryEmbedder.Health пробрасывает ошибку при open circuit

- **Given** `RetryEmbedder` оборачивает другой embedder, circuit open
- **When** вызывается `Health()`
- **Then** возвращается ошибка «circuit breaker open» не доходя до внутреннего embedder
- Evidence: тест с circuit breaker в open state

## Допущения

- Каждый сетевой компонент (pgvector, qdrant, chromadb, weaviate, milvus, ollama, LLM API, embedder API) имеет health/ready endpoint, доступный по HTTP из библиотеки.
- In-memory реализации не имеют внешних зависимостей — `Health()` всегда `nil`.
- Пользователь библиотеки самостоятельно запускает HTTP-сервер и монтирует handler'ы — библиотека не стартует горутины.
- K8s kubelet не добавляет query-параметры к probe-запросам (стандартное поведение).

## Критерии успеха

- SC-001 Все штатные реализации проходят `go vet ./...` и `go build ./...` без ошибок после добавления `Health()`.
- SC-002 Все существующие тесты (`go test ./...`) продолжают проходить без изменений (новый метод не ломает обратную совместимость на уровне API — все имплементации обновлены).

## Краевые случаи

- **Контекст отменён**: если контекст истёк до ответа HealthChecker — возвращать `context.DeadlineExceeded`.
- **Компонент не зарегистрирован**: HealthChecker без компонентов (nil/empty slice) — всегда `nil` (нечего проверять).
- **Ошибка при Health() одного компонента**: не прерывает проверку остальных — собираются все ошибки (multierror).
- **Retry-обёртка не паникует при nil inner**: конструктор RetryEmbedder с nil inner — паника при Health (как и при Embed), это ожидаемо (programmer error).
- **HTTP handler без HealthChecker**: конструктор паникует при nil *HealthChecker (programmer error).

## Открытые вопросы

- none

## Self-Check

- [x] Нет `TODO`/`???`/`<placeholder>`/`TKTK`/`NEEDS CLARIFICATION`
- [x] Каждый AC-* содержит `Given`, `When`, `Then` с observable proof в Then
- [x] Секции Out of Scope (`Вне scope`), `Допущения`, `Открытые вопросы` существуют (или `none`)
- [x] Нет implementation steps или декомпозиции — spec только про intent
- [x] Технологии/версии не зафиксированы
- [x] Spec описывает ровно одну фичу — без multi-feature scope creep
- [x] Goal и RQ-* ID согласованы с AC-* критериями
- [x] Каждый AC-* ведёт к уникальному observable outcome
