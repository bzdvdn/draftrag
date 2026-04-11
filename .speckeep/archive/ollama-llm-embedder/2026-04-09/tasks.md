# Ollama LLM + Embedder Задачи

## Phase Contract

Inputs: plan.md, data-model.md, spec.md.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/llm/ollama.go | T1.1, T2.1 |
| internal/infrastructure/llm/ollama_test.go | T3.1, T3.2 |
| internal/infrastructure/embedder/ollama.go | T1.2, T2.2 |
| internal/infrastructure/embedder/ollama_test.go | T3.1, T3.3 |

## Фаза 1: Структуры и конструкторы

Цель: Создать каркас обеих реализаций — структуры данных и конструкторы с default values.

- [x] T1.1 Создать `internal/infrastructure/llm/ollama.go` — структуры `ollamaChatRequest`, `ollamaChatResponse`, `OllamaLLM`, конструктор `NewOllamaLLM()` с default URL `http://localhost:11434` — AC-001, DEC-003. Touches: internal/infrastructure/llm/ollama.go

- [x] T1.2 Создать `internal/infrastructure/embedder/ollama.go` — структуры `ollamaEmbedRequest`, `ollamaEmbedResponse`, `OllamaEmbedder`, конструктор `NewOllamaEmbedder()` с default URL `http://localhost:11434` — AC-002, DEC-003. Touches: internal/infrastructure/embedder/ollama.go

## Фаза 2: Основная реализация

Цель: Реализовать методы `Generate()` и `Embed()` с обработкой ошибок и валидацией.

- [x] T2.1 Реализовать `OllamaLLM.Generate()` — POST на `/api/chat`, парсинг `message.content`, валидация nil context и пустых строк, обработка HTTP ошибок — AC-001, AC-003, AC-004, AC-005, RQ-001, RQ-002, RQ-003, RQ-006, RQ-007. Touches: internal/infrastructure/llm/ollama.go

- [x] T2.2 Реализовать `OllamaEmbedder.Embed()` — POST на `/api/embeddings` с полем `prompt`, парсинг `embedding`, проверка NaN/Inf, валидация nil context и пустых строк, обработка HTTP ошибок — AC-002, AC-003, AC-004, AC-005, RQ-004, RQ-005, RQ-006, RQ-007. Touches: internal/infrastructure/embedder/ollama.go

## Фаза 3: Тесты и проверка

Цель: Написать unit-тесты с мок-сервером, покрывающие все AC.

- [x] T3.1 Создать `internal/infrastructure/llm/ollama_test.go` — тест `Generate()` с мок-сервером: проверка формата запроса, парсинга ответа, ошибок 4xx/5xx, таймаута контекста, валидации пустых строк — AC-001, AC-003, AC-004, AC-005. Touches: internal/infrastructure/llm/ollama_test.go

- [x] T3.2 Добавить тест конструктора `NewOllamaLLM()` — проверка default base URL, корректной инициализации полей — DEC-003. Touches: internal/infrastructure/llm/ollama_test.go

- [x] T3.3 Создать `internal/infrastructure/embedder/ollama_test.go` — тест `Embed()` с мок-сервером: проверка поля `prompt` вместо `input`, парсинга `embedding`, проверки NaN/Inf, ошибок 4xx/5xx, таймаута контекста, валидации пустых строк — AC-002, AC-003, AC-004, AC-005. Touches: internal/infrastructure/embedder/ollama_test.go

- [x] T3.4 Добавить тест конструктора `NewOllamaEmbedder()` — проверка default base URL, корректной инициализации полей — DEC-003. Touches: internal/infrastructure/embedder/ollama_test.go

- [x] T3.5 Проверить `go build ./...`, `go vet ./...`, `go test ./internal/infrastructure/llm/... ./internal/infrastructure/embedder/...` — все тесты проходят, нет ошибок линтера. Touches: —

## Покрытие критериев приемки

| AC | Покрытие задачами |
|----|-------------------|
| AC-001 LLM-генерация через Ollama | T1.1, T2.1, T3.1 |
| AC-002 Эмбеддинги через Ollama | T1.2, T2.2, T3.3 |
| AC-003 Обработка ошибок Ollama | T2.1, T2.2, T3.1, T3.3 |
| AC-004 Контекстная безопасность | T2.1, T2.2, T3.1, T3.3 |
| AC-005 Валидация входных данных | T2.1, T2.2, T3.1, T3.3 |

## Покрытие требований и решений

| ID | Покрытие задачами |
|----|-------------------|
| RQ-001 OllamaLLM implements LLMProvider | T1.1, T2.1 |
| RQ-002 POST /api/chat | T2.1 |
| RQ-003 Параметры temperature, max_tokens | T2.1 |
| RQ-004 OllamaEmbedder implements Embedder | T1.2, T2.2 |
| RQ-005 POST /api/embeddings | T2.2 |
| RQ-006 HTTP error handling | T2.1, T2.2 |
| RQ-007 Валидация входных параметров | T2.1, T2.2, T3.1, T3.3 |
| DEC-001 Структуры как internal types | T1.1, T1.2 |
| DEC-002 Конструктор без API key validation | T1.1, T1.2, T3.2, T3.4 |
| DEC-003 Default base URL | T1.1, T1.2, T3.2, T3.4 |

## Заметки

- Фаза 1 и Фаза 2 можно выполнять параллельно для LLM и Embedder (независимые пакеты)
- Тесты используют `httptest.Server` для мокирования Ollama API (как в существующих тестах `anthropic_test.go`, `openai_compatible_responses_test.go`)
- Для проверки контекста используем `context.WithTimeout(1*time.Nanosecond)` или `cancel()`
- Проверка NaN/Inf: `math.IsNaN(v) || math.IsInf(v, 0)`
