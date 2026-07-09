# Embedder OpenAI-compatible для draftRAG

## Scope Snapshot

- In scope: реализация `Embedder` через OpenAI-compatible HTTP API (embeddings endpoint) с публичной фабрикой в `pkg/draftrag`, поддержкой `context.Context`, таймаутов и базовой конфигурацией (base URL, API key, model).
- Out of scope: LLM-провайдер, стриминг, batch/async индексация, продвинутая ретрай-логика/циркут-брейкеры, управление rate limits и multi-tenant ключи.

## Цель

Дать пользователю draftRAG готовую реализацию интерфейса `Embedder`, чтобы он мог получить embedding-векторы для `Pipeline.Index` и `Pipeline.Query*` без написания собственного клиента. Успех измеряется тем, что библиотека может вычислять embeddings через OpenAI-compatible API, корректно обрабатывает отмену контекста/таймауты и покрыта unit-тестами без обращения к реальному внешнему сервису по умолчанию.

## Основной сценарий

1. Разработчик создаёт `embedder := draftrag.NewOpenAICompatibleEmbedder(opts)` с `BaseURL`, `APIKey`, `Model`.
2. Собирает pipeline: `pipeline := draftrag.NewPipeline(store, llm, embedder)`.
3. Вызывает `pipeline.Index(ctx, docs)` — внутри используется `Embedder.Embed(ctx, text)` для вычисления embedding.
4. Вызывает `pipeline.QueryTopK(ctx, question, topK)` — embedding вопроса вычисляется тем же embedder’ом.
5. При отмене `ctx` вызовы возвращают `context.Canceled`/`context.DeadlineExceeded` без зависаний.

## Scope

- Infrastructure-реализация интерфейса `internal/domain.Embedder` с HTTP клиентом на стандартной библиотеке (`net/http`).
- Публичный API в `pkg/draftrag`:
  - options struct (base URL, api key, model, http client/timeout)
  - фабрика `NewOpenAICompatibleEmbedder(opts) Embedder`
- Тестирование:
  - unit-тесты на `httptest.Server` (без внешней сети)
  - тест отмены контекста (ctx cancel) для HTTP запроса
- Валидация входов: пустой текст -> ошибка; пустой API key/base URL/model -> ошибка конфигурации.

## Контекст

- Конституция требует интерфейсной абстракции: `Embedder` уже определён в domain, новая реализация — в infrastructure.
- Пакет — библиотека: конфигурация через options, без чтения env var внутри ядра (env — ответственность пользователя/обвязки).
- Все операции принимают `context.Context` первым параметром; `nil` context — panic.
- Зависимости должны быть минимальными; предпочтение стандартной библиотеке.

## Требования

- RQ-001 ДОЛЖНА существовать публичная фабрика `NewOpenAICompatibleEmbedder(...)` в `pkg/draftrag`, возвращающая `draftrag.Embedder` без импорта `internal/...`.
- RQ-002 Реализация ДОЛЖНА выполнять HTTP запрос к OpenAI-compatible embeddings endpoint и возвращать `[]float64` embedding (вектор фиксированной размерности, определяемой моделью).
- RQ-003 Все запросы ДОЛЖНЫ использовать `http.NewRequestWithContext(ctx, ...)` и уважать `ctx.Done()`; при отмене возвращать соответствующую ошибку контекста.
- RQ-004 ДОЛЖНА быть возможность указать `BaseURL` (для OpenAI и совместимых провайдеров) и имя модели embeddings.
- RQ-005 По умолчанию `go test ./...` ДОЛЖЕН проходить без внешней сети: тесты используют `httptest.Server`, интеграционные тесты (если будут) — opt-in.
- RQ-006 Документация (godoc) для публичных типов/функций embedder’а ДОЛЖНА быть на русском языке.
- RQ-007 Ошибки конфигурации ДОЛЖНЫ быть детерминированными и сопоставимыми через `errors.Is` (например, `draftrag.ErrInvalidEmbedderConfig`).

## Вне scope

- Поддержка нескольких входных строк в одном запросе (batch embeddings) как публичный API (можно добавить позже).
- Автоматические ретраи, backoff, circuit breaker.
- Строгая нормализация/квантование embeddings.
- Поддержка “Responses API” и других вариантов, кроме embeddings endpoint.

## Критерии приемки

### AC-001 Публичная фабрика Embedder доступна из pkg/draftrag

- Почему это важно: пользователю нужен готовый embedder без импорта `internal/...`.
- **Given** пользователь импортирует только `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он создаёт embedder через `draftrag.NewOpenAICompatibleEmbedder(opts)`
- **Then** код компилируется, а возвращаемое значение удовлетворяет интерфейсу `draftrag.Embedder`
- Evidence: unit-тест/пример компиляции в `pkg/draftrag` и `go doc` показывают фабрику.

### AC-002 Embedder.Embed возвращает embedding-вектор

- Почему это важно: embeddings — базовая зависимость для retrieval.
- **Given** настроенный embedder и `httptest.Server`, возвращающий валидный OpenAI-compatible JSON ответ
- **When** вызывается `Embed(ctx, "text")`
- **Then** возвращается `[]float64` ненулевой длины и без ошибок
- Evidence: unit-тест `TestOpenAICompatibleEmbedder_Embed_Success` проходит.

### AC-003 Контекстная отмена работает

- Почему это важно: отмена/таймауты критичны для production.
- **Given** контекст `ctx` отменён до запроса или во время запроса
- **When** вызывается `Embed(ctx, "text")`
- **Then** метод возвращает `context.Canceled` (или `context.DeadlineExceeded`) не позднее чем через 100мс
- Evidence: unit-тест с `context.WithCancel()`/`cancel()` и таймаутом 100мс проходит.

### AC-004 Конфигурация валидируется

- Почему это важно: ошибки конфигурации должны быть детерминированными и проверяемыми в клиентском коде.
- **Given** options с пустым `APIKey` или `BaseURL` или `Model`
- **When** создаётся embedder или вызывается `Embed`
- **Then** возвращается ошибка, совместимая с `errors.Is(err, draftrag.ErrInvalidEmbedderConfig)`
- Evidence: unit-тесты `TestOpenAICompatibleEmbedder_ConfigValidation` проходят.

### AC-005 Pipeline использует embedder end-to-end (через мок LLM/Store)

- Почему это важно: embedder должен работать в составе core pipeline.
- **Given** `Pipeline` с in-memory `VectorStore`, мок `LLMProvider` и OpenAI-compatible embedder на `httptest.Server`
- **When** выполняются `Index(ctx, docs)` и `QueryTopK(ctx, question, topK)`
- **Then** retrieval возвращает результаты (len > 0) без ошибок
- Evidence: unit-тест в `pkg/draftrag` или `internal/application` демонстрирует полный цикл.

## Допущения

- OpenAI-compatible embeddings endpoint принимает текст и возвращает числовой embedding (JSON) в стандартной форме “data[0].embedding”.
- Пользователь отвечает за безопасное хранение API key и за выбор модели embeddings, совместимой с его RAG.
- Сеть может быть нестабильной; корректность важнее агрессивных ретраев (в v1 ретраи не включаем).

## Критерии успеха

- SC-001 `go test ./...` проходит без внешней сети и без реальных API ключей.
- SC-002 В embedder’е нет внешних зависимостей, кроме стандартной библиотеки (если возможно).

## Краевые случаи

- Пустой входной текст -> ошибка валидации.
- API возвращает не-200 или невалидный JSON -> ошибка с контекстом (status code / body snippet, без утечек секретов).
- Ответ содержит пустой embedding -> ошибка (невалидный ответ).

## Открытые вопросы

- Нужно ли в v1 поддержать “input: []string” (batch) внутри реализации (без публичного API), чтобы `Pipeline.Index` мог ускориться на большом наборе docs?
- Где лучше разместить код: `internal/infrastructure/embedder/openai_compatible.go` + экспорт через `pkg/draftrag`, или сразу в `pkg/draftrag` с внутренним типом?
