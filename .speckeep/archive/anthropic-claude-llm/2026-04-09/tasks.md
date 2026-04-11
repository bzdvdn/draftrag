# Tasks: Anthropic Claude LLM

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/llm/anthropic.go` | T1.1, T1.2, T2.1, T2.2 |
| `internal/infrastructure/llm/anthropic_test.go` | T1.3, T2.3, T2.4 |
| `internal/infrastructure/llm/openai_compatible_responses.go` | T1.1 (референс) |

## Покрытие критериев приемки

| AC | Покрывающие задачи |
|----|-------------------|
| AC-001 | T1.3 (Generate работает) |
| AC-002 | T1.3 (JSON-структура запроса) |
| AC-003 | T1.3 (заголовок anthropic-version) |
| AC-004 | T2.3 (streaming работает) |
| AC-005 | T2.4 (обработка ошибок, редатация) |

## Фаза 1: Структуры и Generate (MVP)

**Цель:** Создать клиент с базовой генерацией текста, покрывающий AC-001, AC-002, AC-003.

- [x] **T1.1** Добавить структуры данных — `messagesRequest`, `messagesResponse`, `messageContent`, `contentBlock`, константы `anthropicMessagesPath`, `defaultAnthropicVersion` — AC-002, DEC-001
  - Touches: `internal/infrastructure/llm/anthropic.go`

- [x] **T1.2** Реализовать `ClaudeLLM` структуру и конструктор `NewClaudeLLM()` — поля аналогичны OpenAI клиенту, разумные defaults — DEC-001
  - Touches: `internal/infrastructure/llm/anthropic.go`

- [x] **T1.3** Реализовать `Generate()` и тесты — HTTP POST с `anthropic-version` заголовком, парсинг ответа, тесты с мок-сервером — AC-001, AC-002, AC-003
  - Touches: `internal/infrastructure/llm/anthropic.go`, `internal/infrastructure/llm/anthropic_test.go`

## Фаза 2: Streaming и обработка ошибок

**Цель:** Добавить streaming поддержку и корректную обработку ошибок, покрывая AC-004, AC-005.

- [x] **T2.1** Добавить структуры для streaming — `messagesStreamRequest`, `streamEvent` — AC-004, DEC-003
  - Touches: `internal/infrastructure/llm/anthropic.go`

- [x] **T2.2** Реализовать `GenerateStream()` — SSE парсинг через горутину, канал для чанков — AC-004, DEC-003
  - Touches: `internal/infrastructure/llm/anthropic.go`

- [x] **T2.3** Добавить тест на streaming — мок-сервер возвращает SSE, проверка получения чанков — AC-004
  - Touches: `internal/infrastructure/llm/anthropic_test.go`

- [x] **T2.4** Добавить тесты на ошибки — 401, 429 статусы, проверка редатации API ключа — AC-005
  - Touches: `internal/infrastructure/llm/anthropic_test.go`
