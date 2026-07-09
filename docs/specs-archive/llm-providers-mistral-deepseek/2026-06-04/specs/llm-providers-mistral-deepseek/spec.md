# LLM-провайдеры Mistral и DeepSeek

## Scope Snapshot

- In scope: публичные фабричные функции `NewMistralLLM` и `NewDeepSeekLLM`, реализующие `LLMProvider` и `StreamingLLMProvider` через OpenAI‑совместимый Chat Completions API; публичная фабрика `NewMistralEmbedder`, реализующая `Embedder` через OpenAI‑совместимый embeddings endpoint.
- Out of scope: поддержка Embedder для DeepSeek (нет API), CLI‑аргументы для этих провайдеров, модификация существующих провайдеров.

## Цель

Разработчик RAG‑системы на Go получает возможность использовать модели Mistral (mistral‑large, codestral, open‑mistral‑nemo) и DeepSeek (deepseek‑chat, deepseek‑reasoner) как `LLMProvider`. Оба провайдера используют один и тот же OpenAI‑совместимый Chat Completions API (`/v1/chat/completions`), поэтому реализация выносится в общий внутренний слой, а публичные конструкторы задают базовые URL и дефолтные модели по умолчанию. Streaming работает через SSE — тот же механизм, что у существующих провайдеров.

## Основной сценарий

1. Пользователь импортирует `github.com/bzdvdn/draftrag/pkg/draftrag` и вызывает `draftrag.NewMistralLLM(...)` или `draftrag.NewDeepSeekLLM(...)`.
2. Созданный объект реализует `draftrag.LLMProvider` и `draftrag.StreamingLLMProvider` — может быть передан в `draftrag.NewPipeline`.
3. Вызов `.Generate()` отправляет POST‑запрос на соответствующий API (`/v1/chat/completions`). Ответ парсится и возвращается как строка.
4. При наличии API‑ключа и сетевой доступности возвращается сгенерированный текст.
5. При невалидной конфигурации (пустой BaseURL, пустой APIKey, пустая модель) возвращается `ErrInvalidLLMConfig`.

## User Stories

- P1 (MVP): Разработчик может вызвать `NewMistralLLM` / `NewDeepSeekLLM` с валидными опциями, получить `LLMProvider` и вызвать `Generate` — ответ возвращается без ошибки.
- P2: Streaming support — `GenerateStream` отправляет SSE‑чанки через канал, аналогично существующим провайдерам.

## MVP Slice

P1 + P2 (один implementation pass). Оба провайдера полностью готовы к использованию: basic + streaming + примеры в `examples/`.

## First Deployable Outcome

После первого implementation pass пользователь может заменить в своём pipeline `draftrag.NewOpenAICompatibleLLM(...)` на `draftrag.NewMistralLLM(...)` или `draftrag.NewDeepSeekLLM(...)`, указав соответствующий API‑ключ, и получить рабочий RAG‑чат.

## Scope

- Внутренняя реализация в `internal/infrastructure/llm` — OpenAI‑совместимый Chat Completions API (эндпоинт `/v1/chat/completions`) с поддержкой streaming (SSE).
- Публичные фабрики в `pkg/draftrag/mistral_llm.go` и `pkg/draftrag/deepseek_llm.go`.
- Валидация опций через `validateMistralLLMOptions` / `validateDeepSeekLLMOptions` / `validateMistralEmbedderOptions`.
- Примеры в `examples/` для каждого LLM-провайдера (по аналогии с `examples/chat`).
- StreamingLLMProvider capability для обоих LLM-провайдеров.
- Embedder capability для Mistral Embedder.
- Обновление `examples/shared` при необходимости для поддержки новых провайдеров.

## Контекст

- Mistral API: `https://api.mistral.ai/v1/chat/completions`, заголовок `Authorization: Bearer <key>`.
- DeepSeek API: `https://api.deepseek.com/v1/chat/completions`, заголовок `Authorization: Bearer <key>`.
- Оба API поддерживают OpenAI‑совместимый формат `/v1/chat/completions`. Этот формат отличается от `/v1/responses` (Responses API), уже реализованного в `OpenAICompatibleResponsesLLM`.
- Принимаемые параметры: `model`, `messages` (system + user), `temperature`, `max_tokens`, `stream`.
- Streaming: SSE, токен в поле `choices[0].delta.content`, сигнал `[DONE]`.
- Существующий `examples/chat` и `examples/milvus` используют switch по `provider` — туда добавляются `mistral`/`deepseek`.
- Mistral предоставляет embeddings endpoint (`/v1/embeddings`) — OpenAI‑совместимый, модель `mistral-embed`.
- DeepSeek не предоставляет embeddings API.

## Требования

- RQ-001 Публичная функция `NewMistralLLM(opts MistralLLMOptions) LLMProvider` в пакете `draftrag`.
- RQ-002 Публичная функция `NewDeepSeekLLM(opts DeepSeekLLMOptions) LLMProvider` в пакете `draftrag`.
- RQ-003 Оба провайдера ДОЛЖНЫ реализовывать `StreamingLLMProvider` (capability через type assertion).
- RQ-004 Опции обоих провайдеров ДОЛЖНЫ содержать поля: `BaseURL`, `APIKey`, `Model`, `Temperature *float64`, `MaxTokens *int`, `HTTPClient *http.Client`, `Timeout time.Duration`.
- RQ-005 MistralLLMOptions.Model по умолчанию: `"mistral-large-latest"`, DeepSeekLLMOptions.Model по умолчанию: `"deepseek-chat"`.
- RQ-006 MistralLLMOptions.BaseURL по умолчанию: `"https://api.mistral.ai"`, DeepSeekLLMOptions.BaseURL по умолчанию: `"https://api.deepseek.com"`.
- RQ-007 Невалидная конфигурация (пустой BaseURL/APIKey/Model, неверный URL, отрицательный Temperature) возвращает ошибку, сопоставимую с `errors.Is(err, ErrInvalidLLMConfig)`.
- RQ-008 Внутренняя реализация `/v1/chat/completions` вынесена в `internal/infrastructure/llm` и переиспользуется обоими LLM-провайдерами.
- RQ-009 Публичная функция `NewMistralEmbedder(opts MistralEmbedderOptions) Embedder` в пакете `draftrag`.
- RQ-010 Mistral Embedder использует существующий `internal/infrastructure/embedder.OpenAICompatibleEmbedder` для HTTP-взаимодействия.
- RQ-011 `MistralEmbedderOptions` ДОЛЖНА содержать поля: `BaseURL`, `APIKey`, `Model`, `HTTPClient *http.Client`, `Timeout time.Duration`.
- RQ-012 По умолчанию BaseURL=`"https://api.mistral.ai"`, Model=`"mistral-embed"`.
- RQ-013 Невалидная конфигурация Mistral Embedder (пустой APIKey/BaseURL/Model, неверный URL, отрицательный Timeout) возвращает ошибку, сопоставимую с `errors.Is(err, ErrInvalidEmbedderConfig)`.

## Вне scope

- DeepSeek Embedder — DeepSeek не предоставляет embeddings API.
- CLI‑аргументы для выбора провайдера — только Go API.
- Модификация существующего `OpenAICompatibleLLM` / Responses API.
- Поддержка tool calling / function calling.
- Provider-specific features (напр., FIM для codestral, deepseek‑reasoner с reasoning_content).
- Rate limiting / retry logic — используется существующий механизм `resilience`.

## Критерии приемки

### AC-001 Создание MistralLLM

- Почему это важно: разработчик может подключить Mistral как LLM в RAG-пайплайн.
- **Given** валидные MistralLLMOptions (непустой APIKey, BaseURL, Model)
- **When** вызывается `NewMistralLLM(opts)`
- **Then** возвращается не‑nil `LLMProvider`
- **Then** type assertion `StreamingLLMProvider` успешна
- Evidence: тест с httptest-сервером проверяет создание + реализацию обоих интерфейсов

### AC-002 Создание DeepSeekLLM

- Почему это важно: разработчик может подключить DeepSeek как LLM в RAG-пайплайн.
- **Given** валидные DeepSeekLLMOptions (непустой APIKey, BaseURL, Model)
- **When** вызывается `NewDeepSeekLLM(opts)`
- **Then** возвращается не‑nil `LLMProvider`
- **Then** type assertion `StreamingLLMProvider` успешна
- Evidence: тест с httptest-сервером проверяет создание + реализацию обоих интерфейсов

### AC-003 Generate отправляет корректный запрос и парсит ответ

- Почему это важно: разработчик получает осмысленный сгенерированный текст.
- **Given** настроенный провайдер (Mistral или DeepSeek) с httptest-сервером
- **When** вызывается `Generate(ctx, systemPrompt, userMessage)`
- **Then** HTTP-запрос содержит `model`, `messages` с правильной структурой (system + user roles), `stream: false`
- **Then** возвращается текст из поля `choices[0].message.content`
- Evidence: тест перехватывает тело запроса и подменяет ответ, проверяет структуру запроса и результат

### AC-004 Streaming возвращает SSE-чанки

- Почему это важно: UI / real‑time приложения получают ответ по мере генерации.
- **Given** настроенный провайдер с httptest-сервером
- **When** вызывается `GenerateStream(ctx, systemPrompt, userMessage)`
- **Then** запрос содержит `stream: true`
- **Then** канал возвращает текстовые чанки из `choices[0].delta.content`
- **Then** канал закрывается после `data: [DONE]`
- Evidence: тест воспроизводит SSE-поток, читает канал до закрытия, проверяет конкатенированный результат

### AC-005 Невалидная конфигурация возвращает ErrInvalidLLMConfig

- Почему это важно: разработчик получает ошибку, классифицируемую через `errors.Is`, на этапе конфигурации, а не в рантайме.
- **Given** пустой BaseURL (или APIKey, или Model)
- **When** вызывается `Generate` или `GenerateStream`
- **Then** возвращается ошибка, где `errors.Is(err, ErrInvalidLLMConfig) == true`
- Evidence: тесты с пустыми полями проверяют возврат ErrInvalidLLMConfig через `errors.Is`

### AC-006 Default-значения в опциях

- Почему это важно: разработчик может указать минимум параметров (только APIKey) и получить рабочий провайдер.
- **Given** `MistralLLMOptions{APIKey: "sk-..."}` (без BaseURL и Model)
- **When** вызывается `Generate`
- **Then** BaseURL по умолчанию `"https://api.mistral.ai"`, Model по умолчанию `"mistral-large-latest"`
- **Given** `DeepSeekLLMOptions{APIKey: "sk-..."}` (без BaseURL и Model)
- **Then** BaseURL по умолчанию `"https://api.deepseek.com"`, Model по умолчанию `"deepseek-chat"`
- Evidence: тест с httptest-сервером проверяет, что в запросе используется ожидаемый URL и model

### AC-007 Пример в examples/ работает с mock LLM

- Почему это важно: в CI можно проверить, что примеры компилируются и выполняются без реального API-ключа.
- **Given** `LLM_PROVIDER=mock` (или аналогичный механизм)
- **When** запускается `go run .` в `examples/mistral` или `examples/deepseek`
- **Then** программа выводит ответ от mock LLM без ошибок
- Evidence: CI‑шаг `go run .` с mock-режимом завершается exit 0

### AC-008 Создание Mistral Embedder

- Почему это важно: разработчик может использовать Mistral для embeddings в RAG-пайплайне.
- **Given** валидные MistralEmbedderOptions (непустой APIKey)
- **When** вызывается `NewMistralEmbedder(opts)`
- **Then** возвращается не‑nil `Embedder`
- Evidence: тест проверяет создание + реализацию интерфейса Embedder

### AC-009 Mistral Embedder Embed отправляет корректный запрос и парсит ответ

- Почему это важно: разработчик получает осмысленный embedding-вектор.
- **Given** настроенный Mistral Embedder с httptest-сервером
- **When** вызывается `Embed(ctx, text)`
- **Then** HTTP-запрос содержит `model`, `input` с правильной структурой
- **Then** возвращается вектор из `data[0].embedding`
- Evidence: тест с httptest-сервером проверяет тело запроса и ответ

### AC-010 Невалидная конфигурация Mistral Embedder возвращает ErrInvalidEmbedderConfig

- Почему это важно: разработчик получает ошибку, классифицируемую через `errors.Is`, на этапе конфигурации.
- **Given** пустой APIKey (или BaseURL, или Model)
- **When** вызывается `Embed`
- **Then** возвращается ошибка, где `errors.Is(err, ErrInvalidEmbedderConfig) == true`
- Evidence: тесты с пустыми полями проверяют возврат ErrInvalidEmbedderConfig

### AC-011 Default-значения для Mistral Embedder

- Почему это важно: разработчик может указать минимум параметров (только APIKey) и получить рабочий Embedder.
- **Given** `MistralEmbedderOptions{APIKey: "sk-..."}` (без BaseURL и Model)
- **When** вызывается `Embed`
- **Then** BaseURL по умолчанию `"https://api.mistral.ai"`, Model по умолчанию `"mistral-embed"`
- Evidence: тест с httptest-сервером проверяет, что в запросе используется ожидаемая model

## Допущения

- Mistral и DeepSeek сохранят OpenAI‑совместимый формат `/v1/chat/completions` в обозримом будущем.
- Оба API поддерживают одинаковое тело запроса (`model`, `messages`, `temperature`, `max_tokens`, `stream`).
- Заголовок авторизации — `Authorization: Bearer <key>` (как у OpenAI).
- Поле ответа для non‑streaming: `choices[0].message.content`.
- Поле ответа для streaming: `choices[0].delta.content`, терминатор `data: [DONE]`.
- Mistral embeddings endpoint: `POST /v1/embeddings`, тело `{model, input}`, ответ `data[0].embedding`.
- Примеры используют mock‑режим в CI (не требуют реальных ключей).

## Краевые случаи

- Пустой userMessage в Generate/GenerateStream → ошибка `"userMessage is empty"`.
- Пустой systemPrompt — допускается (передаётся пустая строка в messages).
- HTTP 4xx/5xx от API → ошибка с статус‑кодом.
- HTTP‑таймаут → `context.DeadlineExceeded`.
- Streaming: разрыв соединения до `[DONE]` → ошибка в канале.
- Streaming: пустой SSE‑поток (только `[DONE]`) → пустой результат без ошибки.
- API‑ключ передан, но невалиден → HTTP 401 → ошибка с `status=401`.

## Открытые вопросы

- Нужен ли отдельный пример для каждого провайдера (mistral + deepseek), или достаточно одного общего `examples/chat` с поддержкой обоих в switch? — **Решение**: отдельные примеры (`examples/mistral`, `examples/deepseek`) по аналогии с `examples/chat`.
- Обновлять ли `buildComponents` в существующих примерах (milvus, pgvector, qdrant etc.)? — **Решение**: нет, только в новых примерах и в общем `examples/chat`.
- Нужен ли `InternalServerError`‑retry в базовой реализации? — **Решение**: нет, retry остаётся в слое `resilience`; базовая реализация просто возвращает HTTP‑ошибку.
- Нужен ли отдельный пример для Mistral Embedder? — **Решение**: нет, embedder не требует отдельного примера; тест `TestMistralEmbedder_PipelineFullCycle` проверяет интеграцию с Pipeline.

Готово к: /speckeep.inspect llm-providers-mistral-deepseek
