# LLM-провайдеры Mistral и DeepSeek — План

## Phase Contract

Inputs: spec + inspect (pass), минимальный repo-контекст.
Outputs: plan, data-model stub.
Stop if: spec расплывчата — нет, inspect pass.

## Цель

Добавить публичные фабрики `NewMistralLLM` и `NewDeepSeekLLM` в `pkg/draftrag/`, реализованные через общий внутренний слой OpenAI Chat Completions API (`internal/infrastructure/llm/`). Ни новая data model, ни изменения в domain-интерфейсах не требуются — оба провайдера реализуют существующие `LLMProvider` и `StreamingLLMProvider`.

## MVP Slice

Один implementation pass: внутренний слой + оба провайдера + streaming + примеры + все AC (1–7). Разделение на два PR не даёт независимой ценности — пользователю нужны обе фабрики сразу или ни одна.

## First Validation Path

```
go test ./internal/infrastructure/llm/... -run "Chat" -v
go test ./pkg/draftrag/... -run "Mistral|DeepSeek" -v
```

После: `cd examples/mistral && LLM_PROVIDER=mock go run .` — проверяет, что пример компилируется и завершается exit 0.

## Scope

- Внутренний слой: `internal/infrastructure/llm/openai_chat.go` + `internal/infrastructure/llm/openai_chat_test.go`
- Публичные фабрики LLM: `pkg/draftrag/mistral_llm.go` + `pkg/draftrag/deepseek_llm.go`
- Публичная фабрика Embedder: `pkg/draftrag/mistral_embedder.go`
- Тесты фабрик: `pkg/draftrag/mistral_llm_test.go` + `pkg/draftrag/deepseek_llm_test.go` + `pkg/draftrag/mistral_embedder_test.go`
- Примеры: `examples/mistral/main.go` + `examples/deepseek/main.go`
- `pkg/draftrag/doc.go` — без изменений (пакет уже задокументирован)
- `internal/domain/interfaces.go` — без изменений (LLMProvider/StreamingLLMProvider/Embedder уже существуют)

## Implementation Surfaces

| Surface | Статус | Почему участвует |
|---------|--------|-----------------|
| `internal/infrastructure/llm/openai_chat.go` | **новая** | Chat Completions API — другой эндпоинт и формат ответа, чем Responses API |
| `pkg/draftrag/mistral_llm.go` | **новая** | Публичная фабрика + опции + валидация |
| `pkg/draftrag/deepseek_llm.go` | **новая** | Публичная фабрика + опции + валидация |
| `pkg/draftrag/mistral_embedder.go` | **новая** | Публичная фабрика Mistral Embedder + опции + валидация |
| `examples/mistral/main.go` | **новая** | Демо-пример с mock-режимом |
| `examples/deepseek/main.go` | **новая** | Демо-пример с mock-режимом |

## Bootstrapping Surfaces

- `internal/infrastructure/llm/` — существует (есть ollama, anthropic, openai_compatible_responses)
- `pkg/draftrag/` — существует
- `examples/` — существует

Никаких новых директорий до реализации не требуется; структура уже готова.

## Влияние на архитектуру

- Локальное: добавляется новая реализация `LLMProvider`/`StreamingLLMProvider` на инфраструктурном слое.
- Интеграции: не меняются — новые провайдеры подключаются через существующие Go-интерфейсы.
- Migration/rollout: не требуется — библиотека, новая функциональность опциональна.

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдаемый результат |
|----|--------|----------|----------------------|
| AC-001 | Unit-тест: `NewMistralLLM` + type assertion | `mistral_llm.go` | Не-nil + StreamingLLMProvider success |
| AC-002 | Unit-тест: `NewDeepSeekLLM` + type assertion | `deepseek_llm.go` | Не-nil + StreamingLLMProvider success |
| AC-003 | Unit-тест с httptest: захват тела запроса, подмена ответа | `openai_chat.go` | Проверка `model`, `messages`, `stream: false`; ответ `choices[0].message.content` |
| AC-004 | Unit-тест с httptest: SSE-поток | `openai_chat.go` | Чанки из `choices[0].delta.content`; канал закрыт после `[DONE]` |
| AC-005 | Table-driven test: пустые поля | `mistral_llm.go`, `deepseek_llm.go` | `errors.Is(err, ErrInvalidLLMConfig)` |
| AC-006 | Unit-тест с httptest: пустые опции, проверка URL и model | `mistral_llm.go`, `deepseek_llm.go` | Дефолтные значения в запросе |
| AC-007 | E2E: `go run .` с mock LLM | `examples/mistral/`, `examples/deepseek/` | exit 0 |
| AC-008 | Unit-тест: `NewMistralEmbedder` + Pipeline full cycle | `mistral_embedder.go` | Не-nil Embedder + pipeline index/retrieve |
| AC-009 | Unit-тест с httptest: захват тела запроса, подмена ответа | `mistral_embedder.go` (impl в embedder слое) | Проверка model/input; ответ `data[0].embedding` |
| AC-010 | Table-driven test: пустые поля | `mistral_embedder.go` | `errors.Is(err, ErrInvalidEmbedderConfig)` |
| AC-011 | Unit-тест с httptest: пустые опции, проверка model | `mistral_embedder.go` | Дефолтная model=`mistral-embed` в запросе |

## Данные и контракты

- Связанные AC: все — ни один не требует изменения data model.
- Data model не меняется: `MistralLLMOptions`/`DeepSeekLLMOptions` — новые types, но они не domain-сущности, а configuration objects публичного API. `LLMProvider`/`StreamingLLMProvider` уже существуют.
- API-контракты: не меняются.
- См. `data-model.md` (no-change).

## Стратегия реализации

### DEC-001 Единый внутренний слой для Chat Completions API

Why: Mistral и DeepSeek используют идентичный OpenAI-совместимый `/v1/chat/completions`. Дублирование кода для каждого провайдера не дает преимуществ и увеличивает surface для багов.
Tradeoff: Если один из провайдеров изменит API, фикс затронет обоих.
Affects: `internal/infrastructure/llm/openai_chat.go`
Validation: Тесты AC-003 и AC-004 доказывают, что слой работает для общей спецификации.

### DEC-002 Провайдеры как тонкие обёртки (wrapper pattern)

Why: Соответствует существующей архитектуре (`openAICompatibleLLM` → `OpenAICompatibleResponsesLLM`, `anthropicLLM` → `ClaudeLLM`). Публичный слой отвечает только за валидацию опций и defaults.
Tradeoff: Добавляет один уровень косвенности, но он консистентен с кодом.
Affects: `pkg/draftrag/mistral_llm.go`, `pkg/draftrag/deepseek_llm.go`
Validation: AC-001, AC-002, AC-005, AC-006.

### DEC-003 Дефолтная модель — latest-тег

Why: Пользователь, указавший только APIKey, получает самую свежую стабильную модель провайдера без дополнительной конфигурации.
Tradeoff: `mistral-large-latest` может неожиданно изменить поведение при апстрим-релизе. Аналогично для `deepseek-chat`. Пользователь всегда может указать конкретную модель.
Affects: `pkg/draftrag/mistral_llm.go`, `pkg/draftrag/deepseek_llm.go`
Validation: AC-006.

### DEC-004 Переиспользование OpenAICompatibleEmbedder для Mistral Embedder

Why: Mistral embeddings endpoint (`/v1/embeddings`) полностью совместим с OpenAI-форматом. Существующий `internal/infrastructure/embedder.OpenAICompatibleEmbedder` поддерживает POST-запрос, Bearer auth, парсинг `data[0].embedding`. Создавать отдельную HTTP-реализацию нет необходимости.
Tradeoff: Если Mistral изменит формат embeddings API, придётся либо расширять `OpenAICompatibleEmbedder`, либо создавать отдельную реализацию.
Affects: `internal/infrastructure/embedder/openai_compatible.go`, `pkg/draftrag/mistral_embedder.go`
Validation: AC-008, AC-009, AC-010, AC-011.

## Incremental Delivery

### MVP (Первая ценность)

Весь объём: внутренний слой Chat Completions + оба LLM-провайдера + Mistral Embedder + streaming + примеры.
AC: 1–11.
Validation: `go test ./...` + `go run ./examples/mistral` + `go run ./examples/deepseek` с mock.

## Порядок реализации

1. **Внутренний слой** (`openai_chat.go`) — без него публичные фабрики не имеют смысла. Можно и нужно тестировать изолированно через httptest.
2. **Публичные фабрики LLM** (`mistral_llm.go`, `deepseek_llm.go`) — можно параллельно друг другу, обе зависят только от шага 1.
3. **Публичная фабрика Embedder** (`mistral_embedder.go`) — зависит только от существующего `internal/infrastructure/embedder.OpenAICompatibleEmbedder` (уже реализован); может выполняться параллельно с шагом 2.
4. **Примеры** (`examples/mistral`, `examples/deepseek`) — можно параллельно друг другу, зависят от шага 2.
5. **Тесты** — пишутся одновременно с каждым файлом (TDD-стиль: сначала тест, потом реализация).

Параллельно: фабрики Mistral LLM, DeepSeek LLM и Mistral Embedder не зависят друг от друга и могут реализовываться одновременно.

## Риски

- **Изменение API провайдера**: Mistral или DeepSeek могут изменить формат `/v1/chat/completions`. 
  Mitigation: тесты AC-003/AC-004 используют httptest и не зависят от реального API. При изменении достаточно обновить единый внутренний слой.
- **Разные требования к streaming-формату**: хотя оба провайдера декларируют OpenAI-совместимый SSE, могут быть нюансы (отсутствие `[DONE]`, другое поле delta).
  Mitigation: AC-004 тестирует общий SSE-формат. Реальные провайдеры будут проверены в E2E при интеграции с реальным API-ключом.

## Rollout и compatibility

- Специальные rollout-действия не требуются. Новая функциональность опциональна: существующий код не меняется.
- Обратная совместимость: не нарушается.

## Проверка

1. `go test ./internal/infrastructure/llm/...` — покрытие Chat Completions слоя.
2. `go test ./pkg/draftrag/... -run "Mistral|DeepSeek"` — покрытие публичных фабрик LLM.
3. `go test ./pkg/draftrag/... -run "MistralEmbedder"` — покрытие публичной фабрики Embedder.
4. `cd examples/mistral && LLM_PROVIDER=mock go run .` — E2E mock.
5. `cd examples/deepseek && LLM_PROVIDER=mock go run .` — E2E mock.
6. `go vet ./...` — статический анализ.
7. Подтверждает: AC-001–011, DEC-001–004.

## Соответствие конституции

- нет конфликтов: Clean Architecture (domain → application → infrastructure) соблюдена; внешние зависимости через Go-интерфейсы; язык docs=ru.
