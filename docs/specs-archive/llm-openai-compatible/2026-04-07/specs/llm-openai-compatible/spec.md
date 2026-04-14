# LLMProvider OpenAI-compatible для draftRAG

## Scope Snapshot

- In scope: реализация `LLMProvider` через OpenAI-compatible HTTP API (Responses API) для синхронной генерации текста (`Generate(ctx, systemPrompt, userMessage)`) с публичной фабрикой в `pkg/draftrag`, поддержкой `context.Context`, таймаутов и базовой конфигурацией (base URL, API key, model, temperature, max_tokens).
- Out of scope: стриминг ответов, tool-calling/function calling, structured outputs, ретраи/backoff/circuit breaker, управление rate limits, хранение истории диалога и multi-turn chat API.

## Цель

Дать пользователю draftRAG готовую реализацию интерфейса `LLMProvider`, чтобы он мог сгенерировать ответ на основе системного промпта и пользовательского сообщения без написания собственного HTTP клиента. Успех измеряется тем, что библиотека корректно выполняет запрос к OpenAI-compatible endpoint, уважает отмену контекста/таймауты и полностью тестируется через `httptest.Server` без обращения к реальному внешнему сервису по умолчанию.

## Основной сценарий

1. Разработчик создаёт `llm := draftrag.NewOpenAICompatibleLLM(opts)` с `BaseURL`, `APIKey`, `Model`.
2. Вызывает `answer, err := llm.Generate(ctx, systemPrompt, userMessage)`.
3. Получает строковый ответ `answer`.
4. При отмене `ctx` вызов возвращает `context.Canceled`/`context.DeadlineExceeded` не позднее чем через 100мс (в тестовом сценарии).

## Scope

- Infrastructure-реализация интерфейса `internal/domain.LLMProvider` с HTTP клиентом на стандартной библиотеке (`net/http`).
- Публичный API в `pkg/draftrag`:
  - options struct (base URL, api key, model, temperature, max_tokens, http client/timeout)
  - фабрика `NewOpenAICompatibleLLM(opts) LLMProvider`
- Тестирование:
  - unit-тесты на `httptest.Server` (без внешней сети)
  - тест отмены контекста (ctx cancel/deadline)
- Валидация конфигурации: пустой `BaseURL`/`APIKey`/`Model` -> детерминированная sentinel-ошибка, проверяемая через `errors.Is`.

## Контекст

- Конституция требует интерфейсной абстракции: `LLMProvider` уже определён в domain; новая реализация должна жить в infrastructure и экспортироваться через `pkg/draftrag` без импорта `internal/...`.
- Пакет — библиотека: конфигурация через options; чтение env vars и управление секретами остаются на стороне пользователя.
- Все операции принимают `context.Context` первым параметром; `nil` context — panic.
- Зависимости должны быть минимальными; предпочтение стандартной библиотеке.

## Требования

- RQ-001 ДОЛЖНА существовать публичная фабрика `NewOpenAICompatibleLLM(opts)` в `pkg/draftrag`, возвращающая `draftrag.LLMProvider` без импорта `internal/...`.
- RQ-002 Реализация ДОЛЖНА выполнять HTTP запрос к OpenAI-compatible endpoint и возвращать текст ответа как `string`.
- RQ-003 Реализация ДОЛЖНА передавать `systemPrompt` и `userMessage` в запросе как отдельные сообщения (system/user).
- RQ-004 Все запросы ДОЛЖНЫ использовать `http.NewRequestWithContext(ctx, ...)` и уважать `ctx.Done()`; при отмене возвращать `context.Canceled`/`context.DeadlineExceeded`.
- RQ-005 По умолчанию `go test ./...` ДОЛЖЕН проходить без внешней сети: тесты используют `httptest.Server`.
- RQ-006 Документация (godoc) для публичных типов/функций LLM-провайдера ДОЛЖНА быть на русском языке.
- RQ-007 Ошибки конфигурации ДОЛЖНЫ быть детерминированными и сопоставимыми через `errors.Is` (например, `draftrag.ErrInvalidLLMConfig`).
- RQ-008 Ошибки не ДОЛЖНЫ включать `APIKey` (redaction по умолчанию).
- RQ-009 ДОЛЖНА быть возможность задать параметры генерации `temperature` и `max_tokens` через options; значения валидируются (temperature >= 0, max_tokens > 0).

## Вне scope

- Стриминг (SSE/WS) и incremental generation.
- Tool calling / function calling / structured JSON outputs.
- Логирование, метрики, трассировка (кроме возможности передать свой `http.Client`).
- Поддержка нескольких альтернативных ответов (n>1).

## Критерии приемки

### AC-001 Публичная фабрика LLMProvider доступна из pkg/draftrag

- Почему это важно: пользователю нужен готовый LLMProvider без импорта `internal/...`.
- **Given** пользователь импортирует только `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он создаёт провайдера через `draftrag.NewOpenAICompatibleLLM(opts)`
- **Then** код компилируется, а возвращаемое значение удовлетворяет интерфейсу `draftrag.LLMProvider`
- Evidence: unit-тест/пример компиляции в `pkg/draftrag` и `go doc` показывают фабрику.

### AC-002 Generate возвращает ответ на валидный OpenAI-compatible JSON

- Почему это важно: базовая функциональность LLMProvider — вернуть текст ответа.
- **Given** настроенный провайдер и `httptest.Server`, возвращающий валидный JSON ответ
- **When** вызывается `Generate(ctx, systemPrompt, userMessage)`
- **Then** возвращается строка ответа и `err == nil`
- Evidence: unit-тест `TestOpenAICompatibleLLM_Generate_Success` проходит.

### AC-003 Контекстная отмена работает

- Почему это важно: отмена/таймауты критичны для production.
- **Given** контекст `ctx` отменён до запроса или во время запроса
- **When** вызывается `Generate(ctx, systemPrompt, userMessage)`
- **Then** метод возвращает `context.Canceled` (или `context.DeadlineExceeded`) не позднее чем через 100мс
- Evidence: unit-тест с `context.WithCancel()`/`cancel()` и таймаутом 100мс проходит.

### AC-004 Конфигурация валидируется через errors.Is

- Почему это важно: клиентский код должен различать ошибки конфигурации и сетевые ошибки.
- **Given** options с пустым `APIKey` или `BaseURL` или `Model`
- **When** создаётся провайдер или вызывается `Generate`
- **Then** возвращается ошибка, совместимая с `errors.Is(err, draftrag.ErrInvalidLLMConfig)`
- Evidence: unit-тест `TestOpenAICompatibleLLM_ConfigValidation` проходит.

### AC-005 Redaction: APIKey не попадает в текст ошибки

- Почему это важно: секреты не должны утекать через error strings.
- **Given** провайдер с `APIKey`, а сервер возвращает ошибку, содержащую этот ключ в body
- **When** вызывается `Generate`
- **Then** `err.Error()` не содержит `APIKey`
- Evidence: unit-тест `TestOpenAICompatibleLLM_RedactsAPIKey` проходит.

## Допущения

- Минимальный “OpenAI-compatible” контракт реализуем через Responses API: `POST /v1/responses`.
- Ответ содержит итоговый текст в одном из стандартных полей ответа Responses API (в v1 фиксируем минимальный parsing contract в plan/tasks).
- Пользователь сам выбирает модель и отвечает за корректность ключа/провайдера.

## Критерии успеха

- SC-001 `go test ./...` проходит без внешней сети и без реальных API ключей.
- SC-002 Реализация не требует внешних SDK (только стандартная библиотека).

## Краевые случаи

- Пустой `userMessage` -> ошибка валидации.
- non-2xx от API -> ошибка со status code и обрезанным body (с redaction).
- Невалидный JSON или отсутствует `choices[0].message.content` -> ошибка “invalid response”.
- Некорректные `temperature`/`max_tokens` -> ошибка конфигурации (`errors.Is(err, draftrag.ErrInvalidLLMConfig)`).

## Открытые вопросы

- none
