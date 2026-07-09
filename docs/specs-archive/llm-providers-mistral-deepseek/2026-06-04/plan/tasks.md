# LLM-провайдеры Mistral и DeepSeek — Задачи

## Phase Contract

Inputs: plan, data-model, spec.
Outputs: исполнимые задачи с покрытием AC.
Stop if: coverage не удаётся сопоставить — нет, все AC имеют ≥ 1 задачи.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/llm/openai_chat.go` | T1.1 |
| `internal/infrastructure/llm/openai_chat_test.go` | T1.2 |
| `pkg/draftrag/mistral_llm.go` | T2.1 |
| `pkg/draftrag/mistral_llm_test.go` | T2.1 |
| `pkg/draftrag/deepseek_llm.go` | T2.2 |
| `pkg/draftrag/deepseek_llm_test.go` | T2.2 |
| `pkg/draftrag/mistral_embedder.go` | T2.3 |
| `pkg/draftrag/mistral_embedder_test.go` | T2.3 |
| `examples/mistral/main.go` | T3.1 |
| `examples/deepseek/main.go` | T3.2 |
| `examples/shared/` | T3.1, T3.2 |

## Implementation Context

- **Цель MVP**: добавить `NewMistralLLM` и `NewDeepSeekLLM` через общий слой Chat Completions API.
- **Инварианты/семантика**:
  - Оба провайдера используют `/v1/chat/completions` (НЕ `/v1/responses`).
  - Тело запроса: `model`, `messages[{role, content}]`, `temperature`, `max_tokens`, `stream`.
  - Ответ non-streaming: `choices[0].message.content`.
  - Ответ streaming: SSE, `choices[0].delta.content`, терминатор `data: [DONE]`.
- **Ошибки/коды**:
  - Пустой userMessage: `"userMessage is empty"`.
  - Невалидная конфигурация: `errors.Is(err, ErrInvalidLLMConfig)`.
  - HTTP 4xx/5xx: ошибка со статус-кодом.
- **Контракты/протокол**:
  - `Authorization: Bearer <key>`.
  - Content-Type: `application/json`.
  - Streaming: `Accept: text/event-stream`.
- **Границы scope**:
  - Добавляем Mistral Embedder (переиспользует `internal/infrastructure/embedder.OpenAICompatibleEmbedder`).
  - Не добавляем DeepSeek Embedder (нет API).
  - Не меняем `domain/interfaces.go` — `LLMProvider`/`StreamingLLMProvider`/`Embedder` уже есть.
  - Не меняем `OpenAICompatibleLLM` (Responses API).
- **Proof signals**:
  - `go test ./internal/infrastructure/llm/... -run "Chat"`
  - `go test ./pkg/draftrag/... -run "Mistral|DeepSeek"`
  - `cd examples/mistral && LLM_PROVIDER=mock go run .` → exit 0
  - `cd examples/deepseek && LLM_PROVIDER=mock go run .` → exit 0
- **References**: DEC-001 (единый слой), DEC-002 (wrapper pattern), DEC-003 (latest-теги).

## Фаза 1: Chat Completions internal layer

Цель: реализовать общий внутренний слой для OpenAI‑совместимого `/v1/chat/completions` с поддержкой streaming.

- [x] T1.1 Реализовать `internal/infrastructure/llm/openai_chat.go`:
  - `OpenAIChatLLM` структура с полями `client`, `baseURL`, `apiKey`, `model`, `temperature`, `maxTokens`.
  - `NewOpenAIChatLLM(...)` конструктор.
  - `Generate(ctx, systemPrompt, userMessage)` — POST `/v1/chat/completions`, парсит `choices[0].message.content`.
  - `GenerateStream(ctx, systemPrompt, userMessage)` — SSE streaming, чанки из `choices[0].delta.content`, терминатор `[DONE]`.
  - Валидация входов: пустой userMessage → ошибка; пустой systemPrompt допускается.
  - Touches: `internal/infrastructure/llm/openai_chat.go`

- [x] T1.2 Добавить `internal/infrastructure/llm/openai_chat_test.go`:
  - Тест Generate: httptest-сервер, проверка тела запроса (`model`, `messages`, `stream: false`), подмена ответа.
  - Тест GenerateStream: SSE-поток с 3 чанками + `[DONE]`, проверка канала.
  - Тест пустого userMessage → ошибка.
  - Touches: `internal/infrastructure/llm/openai_chat_test.go`

## Фаза 2: Публичные фабрики

Цель: реализовать `NewMistralLLM` и `NewDeepSeekLLM` как тонкие обёртки над `OpenAIChatLLM`.

- [x] T2.1 Реализовать `pkg/draftrag/mistral_llm.go` + `pkg/draftrag/mistral_llm_test.go`:
  - `MistralLLMOptions` struct (BaseURL, APIKey, Model, Temperature, MaxTokens, HTTPClient, Timeout).
  - `NewMistralLLM(opts) LLMProvider` — по умолчанию BaseURL=`https://api.mistral.ai`, Model=`mistral-large-latest`.
  - `validateMistralLLMOptions` — пустые поля → `ErrInvalidLLMConfig`.
  - Type assertion: `StreamingLLMProvider` успешна.
  - Тесты: создание + type assertion, дефолтные значения (AC-006), невалидная конфигурация (AC-005).
  - Touches: `pkg/draftrag/mistral_llm.go`, `pkg/draftrag/mistral_llm_test.go`

- [x] T2.2 Реализовать `pkg/draftrag/deepseek_llm.go` + `pkg/draftrag/deepseek_llm_test.go`:
  - `DeepSeekLLMOptions` struct (те же поля).
  - `NewDeepSeekLLM(opts) LLMProvider` — по умолчанию BaseURL=`https://api.deepseek.com`, Model=`deepseek-chat`.
  - `validateDeepSeekLLMOptions` — пустые поля → `ErrInvalidLLMConfig`.
  - Type assertion: `StreamingLLMProvider` успешна.
  - Тесты: создание + type assertion, дефолтные значения (AC-006), невалидная конфигурация (AC-005).
  - Touches: `pkg/draftrag/deepseek_llm.go`, `pkg/draftrag/deepseek_llm_test.go`

- [x] T2.3 Реализовать `pkg/draftrag/mistral_embedder.go` + `pkg/draftrag/mistral_embedder_test.go`:
  - `MistralEmbedderOptions` struct (BaseURL, APIKey, Model, HTTPClient, Timeout).
  - `NewMistralEmbedder(opts) Embedder` — переиспользует `internal/infrastructure/embedder.OpenAICompatibleEmbedder`.
  - По умолчанию BaseURL=`https://api.mistral.ai`, Model=`mistral-embed`.
  - `validateMistralEmbedderOptions` — пустые поля (APIKey, BaseURL, Model) → `ErrInvalidEmbedderConfig`.
  - Тесты: создание + type assertion (AC-008), дефолтные значения (AC-011), невалидная конфигурация (AC-010), PipelineFullCycle (AC-008), redaction.
  - Touches: `pkg/draftrag/mistral_embedder.go`, `pkg/draftrag/mistral_embedder_test.go`

## Фаза 3: Примеры

Цель: добавить работающие примеры с mock-режимом для CI.

- [x] T3.1 Добавить `examples/mistral/main.go`:
  - По аналогии с `examples/chat/`. Поддержка `LLM_PROVIDER=mock` (+ ollama/openai/anthropic в switch).
  - `go run .` с mock → exit 0.
  - При необходимости: обновить `examples/shared` (добавить провайдер в switch, если shared использует enum).
  - Touches: `examples/mistral/main.go`, `examples/shared/` (если нужно)

- [x] T3.2 Добавить `examples/deepseek/main.go`:
  - Аналогично T3.1.
  - `go run .` с mock → exit 0.
  - Touches: `examples/deepseek/main.go`, `examples/shared/` (если нужно)

## Фаза 4: Проверка

Цель: итоговая верификация.

- [x] T4.1 Финальная проверка:
  - `go vet ./...`
  - `go test ./...`
  - `go build ./...`
  - Touches: none (запуск команд)

## Покрытие критериев приемки

- AC-001 (Создание MistralLLM) → T2.1
- AC-002 (Создание DeepSeekLLM) → T2.2
- AC-003 (Generate запрос/ответ) → T1.1, T1.2
- AC-004 (Streaming) → T1.1, T1.2
- AC-005 (ErrInvalidLLMConfig) → T2.1, T2.2
- AC-006 (Default-значения) → T2.1, T2.2
- AC-007 (Пример с mock) → T3.1, T3.2, T4.1
- AC-008 (Создание Mistral Embedder) → T2.3
- AC-009 (Embedder запрос/ответ) → T1.1 (переиспользует OpenAICompatibleEmbedder), T2.3
- AC-010 (ErrInvalidEmbedderConfig) → T2.3
- AC-011 (Default-значения Embedder) → T2.3

## Заметки

- T1.1 и T1.2 — единственные с hardware dependency: требуют httptest, внешних сетевых вызовов нет.
- T2.1 и T2.2 независимы и могут выполняться параллельно.
- T2.3 зависит только от существующего `internal/infrastructure/embedder.OpenAICompatibleEmbedder` (уже реализован).
- T3.1 и T3.2 независимы и могут выполняться параллельно.
- Все задачи могут быть выполнены без реального API-ключа (mock/httptest).
