# Streaming ответов — Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для streaming-функциональности.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/infrastructure/llm/openai_compatible_responses.go` | T2.1, T2.2 |
| `internal/infrastructure/llm/mock_streaming.go` | T3.2 |
| `internal/application/pipeline.go` | T2.3, T2.4 |
| `pkg/draftrag/draftrag.go` | T2.5, T2.6, T2.7 |
| `pkg/draftrag/openai_compatible_llm.go` | T2.5, T2.6 |
| `internal/infrastructure/llm/openai_compatible_responses_test.go` | T3.1 |
| `internal/application/pipeline_answer_stream_test.go` | T3.1 |
| `pkg/draftrag/answer_stream_test.go` | T3.1 |

## Фаза 1: Интерфейс (Domain)

Цель: Определить capability-интерфейс `StreamingLLMProvider` как foundation для всех streaming-фич.

- [x] T1.1 Добавить `StreamingLLMProvider` интерфейс с методом `GenerateStream` — возвращает `(<-chan string, error)`, embeds `LLMProvider` для backward compatibility. Touches: `internal/domain/interfaces.go` — DEC-001, AC-004

## Фаза 2: Реализация

Цель: Реализовать SSE streaming, application use-cases и public API.

- [x] T2.1 Реализовать `GenerateStream` в `OpenAICompatibleResponsesLLM` — SSE парсинг, горутина-производитель, корректное закрытие канала при завершении или отмене контекста. Touches: `internal/infrastructure/llm/openai_compatible_responses.go` — AC-001, AC-003, AC-005, DEC-002

- [x] T2.2 Добавить обработку SSE edge cases — игнорирование `: ping`, пустых линий, многострочных `data:` чанков, ошибок в середине streaming'а. Touches: `internal/infrastructure/llm/openai_compatible_responses.go` — AC-005

- [x] T2.3 Реализовать `AnswerStream` в application Pipeline — retrieval (синхронно), затем streaming generation через type assertion на `StreamingLLMProvider`. Touches: `internal/application/pipeline.go` — AC-001, DEC-003

- [x] T2.4 Реализовать `AnswerStreamWithInlineCitations` в application Pipeline — retrieval с цитатами, затем streaming generation. Touches: `internal/application/pipeline.go` — AC-002

- [x] T2.5 Добавить `AnswerStream` в public API — метод `Pipeline.AnswerStream(ctx, question, topK) (<-chan string, error)`, валидация входных данных. Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/openai_compatible_llm.go` — AC-001

- [x] T2.6 Добавить `AnswerStreamWithInlineCitations` в public API — метод `Pipeline.AnswerStreamWithInlineCitations(ctx, question, topK) (<-chan string, []InlineCitation, error)`. Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/openai_compatible_llm.go` — AC-002

- [x] T2.7 Реализовать graceful degradation для non-streaming LLM — `AnswerStream*` возвращает `ErrStreamingNotSupported` если LLM не реализует `StreamingLLMProvider`. Touches: `pkg/draftrag/draftrag.go` — AC-004

## Фаза 3: Проверка

Цель: Доказать корректность через тесты и мок-реализацию.

- [x] T3.1 Добавить тесты на streaming-функциональность — `GenerateStream` с мок HTTP server (SSE), context cancellation без утечек, graceful degradation, `AnswerStream` end-to-end. Touches: `internal/infrastructure/llm/openai_compatible_responses_test.go`, `internal/application/pipeline_answer_stream_test.go`, `pkg/draftrag/answer_stream_test.go` — AC-001, AC-002, AC-003, AC-004, AC-005, RQ-005, RQ-006

- [x] T3.2 Создать мок-реализацию `StreamingLLMProvider` для тестирования — controlled token emission, поддержка таймаутов и ошибок. Touches: `internal/infrastructure/llm/mock_streaming.go` — RQ-007

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T2.3, T2.5, T3.1
- AC-002 -> T2.4, T2.6, T3.1
- AC-003 -> T2.1, T3.1
- AC-004 -> T1.1, T2.7, T3.1
- AC-005 -> T2.1, T2.2, T3.1

## Покрытие требований

| RQ | Покрытие задачами |
|----|-------------------|
| RQ-001 StreamingLLMProvider интерфейс | T1.1 |
| RQ-002 SSE streaming для OpenAI-compatible | T2.1, T2.2 |
| RQ-003 AnswerStream в public API | T2.3, T2.5 |
| RQ-004 AnswerStreamWithInlineCitations | T2.4, T2.6 |
| RQ-005 Отмена контекста без утечек | T2.1, T3.1 |
| RQ-006 Обработка ошибок streaming'а | T2.2, T3.1 |
| RQ-007 Мок-реализация для тестирования | T3.2 |

## Заметки

- Порядок задач соответствует плану: domain → infrastructure → application → public API → tests
- Фаза 1 блокирует Фазу 2 (нужен интерфейс для реализации)
- T2.1 блокирует T2.3/T2.4 (нужен `GenerateStream` для `AnswerStream`)
- T2.3/T2.4 блокируют T2.5/T2.6 (нужен application layer для public API)
- T3.1 может выполняться параллельно с T3.2 (независимые тестовые артефакты)
