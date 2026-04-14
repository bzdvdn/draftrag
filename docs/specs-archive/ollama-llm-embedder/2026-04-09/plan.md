# Ollama LLM + Embedder План

## Phase Contract

Inputs: spec, inspect report и минимальный контекст репозитория.
Outputs: plan.md, data-model.md.

## Цель

Создать две новых реализации интерфейсов `domain.LLMProvider` и `domain.Embedder` для интеграции с локальным Ollama API. Реализации следуют паттернам существующих OpenAI-совместимых клиентов с адаптацией под специфику Ollama API (другие endpoint'ы и формат запросов/ответов).

## Scope

- `internal/infrastructure/llm/ollama.go` — реализация LLMProvider через `/api/chat`
- `internal/infrastructure/embedder/ollama.go` — реализация Embedder через `/api/embeddings`
- `internal/infrastructure/llm/ollama_test.go` — unit-тесты с мок-сервером
- `internal/infrastructure/embedder/ollama_test.go` — unit-тесты с мок-сервером

## Implementation Surfaces

| Surface | Type | Status | Rationale |
|---------|------|--------|-----------|
| `internal/infrastructure/llm/ollama.go` | New | Required | Реализация `LLMProvider` для Ollama `/api/chat` |
| `internal/infrastructure/llm/ollama_test.go` | New | Required | Тесты с мок-сервером для AC-001, AC-003, AC-004, AC-005 |
| `internal/infrastructure/embedder/ollama.go` | New | Required | Реализация `Embedder` для Ollama `/api/embeddings` |
| `internal/infrastructure/embedder/ollama_test.go` | New | Required | Тесты с мок-сервером для AC-002, AC-003, AC-004, AC-005 |

## Влияние на архитектуру

- Локальное влияние: новые файлы в инфраструктурных пакетах, нет изменений существующих компонентов
- Интерфейсы `LLMProvider` и `Embedder` остаются неизменными
- Ollama не требует API key, но конструкторы сохраняют signature для консистентности (apiKey можно игнорировать или передавать пустым)
- Нет breaking changes, нет миграции

## Acceptance Approach

| AC | Реализация | Surfaces | Observable Proof |
|----|------------|----------|------------------|
| AC-001 | Структуры `ollamaChatRequest`, `ollamaChatResponse`; метод `Generate()` | `ollama.go` | Мок-сервер получает POST на `/api/chat` с JSON `{"model": "...", "messages": [...], "stream": false}`; тест проверяет парсинг `message.content` |
| AC-002 | Структуры `ollamaEmbedRequest`, `ollamaEmbedResponse`; метод `Embed()` | `embedder/ollama.go` | Мок-сервер получает POST на `/api/embeddings` с JSON `{"model": "...", "prompt": "..."}`; тест проверяет парсинг `embedding` |
| AC-003 | Проверка `resp.StatusCode` в обоих методах | `ollama.go`, `embedder/ollama.go` | Мок возвращает 404/500; тест проверяет содержимое ошибки |
| AC-004 | `http.NewRequestWithContext(ctx, ...)` и проверка `ctx.Err()` | `ollama.go`, `embedder/ollama.go` | Тест с `context.WithTimeout` или `cancel()`; проверка что вернулся `context.DeadlineExceeded` или `context.Canceled` |
| AC-005 | Проверка `strings.TrimSpace(text) == ""` и `ctx == nil` в начале методов | `ollama.go`, `embedder/ollama.go` | Тест с пустой строкой — метод возвращает ошибку до HTTP-запроса |

## Данные и контракты

Данные: см. `data-model.md` — новые сущности для запросов/ответов Ollama API.

Контракты:
- **Request contract Ollama Chat**: `POST /api/chat`
  - Body: `{"model": string, "messages": [{"role": string, "content": string}], "stream": false, "temperature": number, "max_tokens": number}`
  - Response: `{"message": {"role": string, "content": string}}`

- **Request contract Ollama Embeddings**: `POST /api/embeddings`
  - Body: `{"model": string, "prompt": string}`
  - Response: `{"embedding": []float64}`

## Стратегия реализации

### DEC-001 Структуры данных как internal types
Why: Формат Ollama API отличается от OpenAI-compatible (`messages` вместо `input`, `prompt` вместо `input`, прямые поля вместо `data[]`). Отдельные структуры дают чистый код без условной логики.
Tradeoff: Дублирование pattern'ов (request/response types), но читаемость лучше generic map[string]interface{}.
Affects: `ollama.go`, `embedder/ollama.go`
Validation: Тесты проверяют сериализацию/десериализацию JSON

### DEC-002 Конструктор без API key validation
Why: Ollama работает локально без аутентификации; API key не нужен но сохраняем параметр для консистентности signature с другими реализациями.
Tradeoff: Незначительная несогласованность (параметр есть но не используется) vs breaking change в конструкторе.
Affects: `NewOllamaLLM()`, `NewOllamaEmbedder()`
Validation: Тесты с пустым apiKey проходят успешно

### DEC-003 Default base URL в конструкторе
Why: Стандартный порт Ollama — 11434, пользователи ожидают работу "из коробки" без явного указания URL.
Tradeoff: Магия вместо явности, но соответствует конституции "минимальная конфигурация".
Affects: Конструкторы — если baseURL == "", используется `http://localhost:11434`
Validation: Тест с пустым baseURL использует дефолтный URL

## Incremental Delivery

### MVP

- `ollama.go` с конструктором и `Generate()`
- `ollama_test.go` с мок-сервером для AC-001, AC-005
- `embedder/ollama.go` с конструктором и `Embed()`
- `embedder/ollama_test.go` с мок-сервером для AC-002, AC-005

Критерий готовности MVP: все unit-тесты проходят, coverage ≥60% для новых файлов.

### Итеративное расширение

- AC-003: тесты на обработку ошибок (можно добавить отдельными тест-функциями)
- AC-004: тесты на таймаут контекста (можно добавить отдельными тест-функциями)

## Порядок реализации

1. Создать `internal/infrastructure/llm/ollama.go` с конструктором и структурами
2. Создать `internal/infrastructure/embedder/ollama.go` с конструктором и структурами
3. Реализовать `Generate()` и `Embed()`
4. Создать тесты с мок-сервером
5. Добавить тесты на ошибки и контекст

Шаги 1-2 можно параллелить (независимые пакеты).

## Риски

| Риск | Mitigation |
|------|------------|
| Ollama API изменяется | Фиксируем минимальный контракт; если API breaking change — это внешняя зависимость вне scope библиотеки |
| Ollama не всегда доступен в CI | Интеграционные тесты делаем опциональными (skip if OLLAMA_URL not set); полное покрытие через моки |
| Пользователи путают форматы запросов | Godoc комментарии с примерами JSON; clear error messages при 404 |

## Rollout и compatibility

- Нет breaking changes (новые файлы)
- Нет миграции
- Нет feature flags
- Специальных rollout-действий не требуется

## Проверка

| Что проверить | Как | AC/DEC |
|---------------|-----|--------|
| Формат запроса к `/api/chat` | Мок-сервер + `httptest` | AC-001 |
| Формат запроса к `/api/embeddings` | Мок-сервер + `httptest` | AC-002 |
| Валидация пустых строк | Тест с `""` и `"   "` | AC-005 |
| Валидация nil context | Тест с `nil` context | AC-005 |
| HTTP ошибки 4xx/5xx | Мок возвращает 404/500 | AC-003 |
| Таймаут контекста | `context.WithTimeout(1ns)` | AC-004 |
| Проверка NaN/Inf в embedding | Мок возвращает вектор с NaN | RQ-006 |
| Конструктор с default URL | Тест `NewOllamaLLM(nil, "", "", "model", nil, nil)` | DEC-003 |

## Соответствие конституции

- **Интерфейсная абстракция**: `OllamaLLM` реализует `domain.LLMProvider`, `OllamaEmbedder` реализует `domain.Embedder` — соответствует.
- **Чистая архитектура**: Реализации в инфраструктурном слое, domain-интерфейсы неизменны — соответствует.
- **Контекстная безопасность**: Все операции принимают `context.Context` — соответствует.
- **Тестируемость**: Мок-реализации через `httptest.Server` — соответствует.
- **Минимальная конфигурация**: Default base URL `http://localhost:11434` — соответствует.

**Вердикт**: нет конфликтов с конституцией.
