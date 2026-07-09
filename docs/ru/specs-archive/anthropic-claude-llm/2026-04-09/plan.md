# Anthropic Claude LLM План

## Phase Contract

Inputs: spec, inspect report, существующая реализация `OpenAICompatibleResponsesLLM` как референс.
Outputs: plan, data-model.

## Цель

Создать нативный клиент для Anthropic Messages API в `internal/infrastructure/llm/`, реализующий интерфейсы `LLMProvider` и `StreamingLLMProvider`. Клиент использует endpoint `https://api.anthropic.com/v1/messages` с обязательным заголовком `anthropic-version`.

## Scope

- Создание нового файла `internal/infrastructure/llm/anthropic.go`
- Структуры запроса/ответа для Anthropic Messages API
- Реализация `Generate()` для непотоковой генерации
- Реализация `GenerateStream()` для SSE streaming
- Unit-тесты с мок-сервером
- Без изменений в существующем `OpenAICompatibleResponsesLLM`

## Implementation Surfaces

| Surface | Status | Почему участвует |
|---------|--------|------------------|
| `internal/infrastructure/llm/anthropic.go` | новый | Основная реализация клиента Anthropic |
| `internal/infrastructure/llm/anthropic_test.go` | новый | Unit-тесты с мок-сервером |
| `internal/domain/interfaces.go` | без изменений | Интерфейсы `LLMProvider` и `StreamingLLMProvider` уже определены |
| `internal/infrastructure/llm/openai_compatible_responses.go` | референс | Паттерны для HTTP клиента, SSE парсинга, обработки ошибок |

## Влияние на архитектуру

- **Локальное**: Новый файл в `internal/infrastructure/llm/` — чистое добавление, никаких breaking changes
- **Интеграция**: Реализация существующих интерфейсов domain-слоя
- **Compatibility**: Новый провайдер, не затрагивает существующие реализации

## Acceptance Approach

| AC | Реализация | Observable Proof |
|----|------------|------------------|
| AC-001 | `ClaudeLLM.Generate()` отправляет запрос и парсит ответ | Тест `TestClaudeLLM_Generate_Success` — мок-сервер возвращает валидный JSON, клиент возвращает текст |
| AC-002 | Структуры запроса соответствуют Anthropic-формату | Тест перехватывает request body, проверяет поля `model`, `max_tokens`, `system`, `messages` |
| AC-003 | Заголовок `anthropic-version` добавляется в каждый запрос | Тест проверяет `req.Header.Get("anthropic-version")` |
| AC-004 | `ClaudeLLM.GenerateStream()` возвращает канал с чанками | Тест с SSE-ответом, проверяет что канал получает >1 чанка |
| AC-005 | Ошибки API возвращаются с деталями, ключ редатирован | Тесты на 401/429, проверка `!strings.Contains(err.Error(), apiKey)` |

## Данные и контракты

- **Data model изменений не требуется** — нет persisted state
- **API contracts**: Anthropic Messages API v1 (stable версия `2023-06-01`)
- **Request format**:
  ```json
  {
    "model": "claude-3-haiku-20240307",
    "max_tokens": 1024,
    "system": "System prompt (optional)",
    "messages": [{"role": "user", "content": "User message"}]
  }
  ```
- **Response format**:
  ```json
  {
    "content": [{"type": "text", "text": "Response"}],
    "role": "assistant"
  }
  ```

## Стратегия реализации

### DEC-001 Структура клиента аналогична OpenAI клиенту
- **Why**: Сохраняем консистентность кодовой базы, облегчаем поддержку
- **Tradeoff**: Меньше гибкости для специфичных фич Anthropic в будущем
- **Affects**: `internal/infrastructure/llm/anthropic.go`
- **Validation**: Структура `ClaudeLLM` повторяет поля `OpenAICompatibleResponsesLLM` (httpClient, apiKey, model, и т.д.)

### DEC-002 Отдельное поле `system` в запросе (не в messages массиве)
- **Why**: Anthropic API требует `system` как top-level поле, не как сообщение с `role: system`
- **Tradeoff**: Небольшое усложнение маппинга из `Generate(systemPrompt, userMessage)`
- **Affects**: Формирование `messagesRequest` в `Generate()` и `GenerateStream()`
- **Validation**: Тест проверяет JSON-структуру запроса

### DEC-003 SSE формат Anthropic совместим с базовым парсингом
- **Why**: Anthropic использует стандартный SSE: `data: {...}\n\n` и `data: [DONE]`
- **Tradeoff**: Нет поддержки специфичных Anthropic SSE events (message_start, message_delta и т.д.) — достаточно для базового streaming
- **Affects**: `GenerateStream()` — парсинг аналогичен OpenAI клиенту
- **Validation**: Тест с mock SSE-ответом

## Incremental Delivery

### MVP (Первая ценность)
- `ClaudeLLM` структура и конструктор
- `Generate()` с базовым HTTP запросом
- Тест `TestClaudeLLM_Generate_Success`
- Покрывает: AC-001, AC-002, AC-003

### Итеративное расширение
- `GenerateStream()` с SSE парсингом
- Тесты на streaming и ошибки
- Покрывает: AC-004, AC-005

## Порядок реализации

1. **Сначала**: Структуры данных (`messagesRequest`, `messagesResponse`, `streamEvent`)
2. **Затем**: `ClaudeLLM` структура и конструктор `NewClaudeLLM()`
3. **Параллельно**: `Generate()` + тесты (MVP)
4. **После MVP**: `GenerateStream()` + тесты

## Риски

- **Разница в SSE формате** Anthropic vs OpenAI
  - Mitigation: Проверка через mock-тест перед интеграцией
- **Изменения в Anthropic API**
  - Mitigation: `anthropic-version` заголовок фиксирует версию

## Rollout и compatibility

Специальных rollout-действий не требуется — новый провайдер, не затрагивает существующий код.

## Проверка

| Что проверяем | Как | AC/DEC |
|---------------|-----|--------|
| Генерация работает | `TestClaudeLLM_Generate_Success` | AC-001 |
| Формат запроса корректен | Проверка JSON-структуры в тесте | AC-002 |
| Заголовок версии присутствует | `req.Header.Get()` в тесте | AC-003 |
| Streaming работает | `TestClaudeLLM_GenerateStream_Success` | AC-004 |
| Ошибки обрабатываются | Тесты на 401/429 | AC-005 |

## Соответствие конституции

- **Интерфейсная абстракция**: ✓ Реализация `LLMProvider` и `StreamingLLMProvider`
- **Чистая архитектура**: ✓ Infrastructure слой, нет импортов domain→infrastructure
- **Минимальная конфигурация**: ✓ Разумные defaults для всех параметров
- **Контекстная безопасность**: ✓ `context.Context` первый параметр
- **Тестируемость**: ✓ Unit-тесты с мок-сервером
- **Языковая политика**: ✓ Godoc на русском

нет конфликтов
