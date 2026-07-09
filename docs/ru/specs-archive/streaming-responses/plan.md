# Streaming ответов — План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для streaming-функциональности.
Outputs: plan.md, data-model.md.

## Цель

Добавить streaming-генерацию ответов через Go-каналы, сохранив backward compatibility. Реализация через capability-интерфейс `StreamingLLMProvider` (как `VectorStoreWithFilters`), SSE-парсинг для OpenAI-compatible API, и новые методы `AnswerStream*` в application и public API.

## Scope

- Domain: новый capability-интерфейс `StreamingLLMProvider`
- Infrastructure: SSE streaming в `OpenAICompatibleResponsesLLM`
- Application: `AnswerStream`, `AnswerStreamWithInlineCitations` use-cases
- Public API: `Pipeline.AnswerStream*`, `OpenAICompatibleLLMOptions` с streaming-флагом
- Мок-реализация для тестирования

## Implementation Surfaces

| Surface | Type | Location | Why |
|---------|------|----------|-----|
| `StreamingLLMProvider` | New interface | `internal/domain/interfaces.go` | Capability-интерфейс для streaming, аналогичный `VectorStoreWithFilters` |
| `GenerateStream()` | New method | `internal/infrastructure/llm/openai_compatible_responses.go` | SSE-реализация для OpenAI API |
| `AnswerStream()` | New method | `internal/application/pipeline.go` | Application use-case для streaming ответов |
| `AnswerStreamWithInlineCitations()` | New method | `internal/application/pipeline.go` | Streaming с цитатами |
| `AnswerStream()` | New method | `pkg/draftrag/draftrag.go` | Public API exposure |
| `AnswerStreamWithInlineCitations()` | New method | `pkg/draftrag/draftrag.go` | Public API exposure |
| Streaming mock | New file | `internal/infrastructure/llm/mock_streaming.go` | Тестовая реализация `StreamingLLMProvider` |

## Влияние на архитектуру

- **Local impact**: Новый capability-интерфейс не ломает существующий `LLMProvider`
- **Integration**: Pipeline проверяет `llm.(StreamingLLMProvider)` для streaming-методов
- **Compatibility**: Существующие реализации `LLMProvider` продолжают работать; streaming методы возвращают `ErrStreamingNotSupported`

## Acceptance Approach

- **AC-001**: Реализовать `GenerateStream` → создать канал, запустить горутину с SSE-чтением, возвращать канал
- **AC-002**: Реализовать `AnswerStreamWithInlineCitations` → retrieval (синхронно) + streaming генерация, цитаты собираются заранее
- **AC-003**: Context cancellation → `select` на `<-ctx.Done()` в горутине, закрытие канала при отмене
- **AC-004**: Backward compatibility → type assertion на `StreamingLLMProvider`, возврат ошибки если не реализован
- **AC-005**: SSE parsing → линейное сканирование `data:` префиксов, игнорирование `: ping` и пустых линий

## Данные и контракты

- **Нет persisted state**: Streaming — runtime-only концепция (каналы, горутины)
- **API Contract**: `GenerateStream(ctx, systemPrompt, userMessage) (<-chan string, error)` — канал закрывается при завершении или ошибке
- **Error handling**: Ошибка возвращается через error return value, канал закрывается

## Стратегия реализации

### DEC-001 Capability-интерфейс вместо расширения LLMProvider

- **Why**: Сохранить backward compatibility. Существующие реализации `LLMProvider` не должны меняться.
- **Tradeoff**: Дополнительный type assertion в application layer
- **Affects**: `internal/domain/interfaces.go`, `internal/application/pipeline.go`
- **Validation**: Тест на graceful degradation для non-streaming LLM

### DEC-002 SSE streaming с горутиной-производителем

- **Why**: Идиоматичный Go-подход для streaming; позволяет начать получать данные до завершения HTTP-запроса
- **Tradeoff**: Нужна аккуратная обработка утечек горутин
- **Affects**: `internal/infrastructure/llm/openai_compatible_responses.go`
- **Validation**: Тест на context cancellation без утечек

### DEC-003 Синхронный retrieval перед streaming generation

- **Why**: Retrieval (Embed + Search) не поддерживает streaming; нужен полный контекст перед генерацией
- **Tradeoff**: Пользователь ждёт retrieval перед началом streaming'а
- **Affects**: `internal/application/pipeline.go` — `AnswerStream*` методы
- **Validation**: AC-002 — цитаты собираются заранее, streaming только для генерации

## Incremental Delivery

### MVP (Первая ценность)

- `StreamingLLMProvider` интерфейс
- `GenerateStream` для OpenAI-compatible
- `AnswerStream` в application и public API
- Тесты на AC-001, AC-003, AC-004

### Итеративное расширение

- `AnswerStreamWithInlineCitations` — добавляет цитаты (AC-002)
- SSE edge cases: пустые линии, ping, многострочные чанки (AC-005)
- Мок-реализация для тестирования (RQ-007)

## Порядок реализации

1. **First**: `StreamingLLMProvider` интерфейс (domain) — foundation для всех streaming-фич
2. **Then**: `GenerateStream` в OpenAI-compatible (infrastructure) — можно тестировать независимо
3. **Then**: `AnswerStream` в application + public API — интеграция
4. **Parallel**: `AnswerStreamWithInlineCitations` и мок-реализация

## Риски

- **Горутинные утечки** при отмене контекста
  Mitigation: Использовать `defer close(ch)`, `select` на `ctx.Done()`, тесты с race detector
- **Несовместимость SSE форматов** у разных провайдеров
  Mitigation: Начать с OpenAI официального API, документировать ограничение (только OpenAI-compatible)
- **Сложность тестирования streaming**
  Mitigation: Мок-реализация с controlled token emission, таймауты в тестах

## Rollout и compatibility

- Не требуется migration или feature flags
- Backward compatibility: streaming методы возвращают `ErrStreamingNotSupported` для legacy LLM
- Zero breaking changes

## Проверка

- Unit: `GenerateStream` с мок HTTP server (SSE response)
- Unit: Context cancellation не приводит к утечкам (go leak detector или manual verification)
- Integration: `AnswerStream` end-to-end с мок LLM
- Unit: Graceful degradation для non-streaming LLM

## Соответствие конституции

- **Интерфейсная абстракция**: ✅ `StreamingLLMProvider` — capability-интерфейс
- **Чистая архитектура**: ✅ Domain interface → Infrastructure impl → Application use-case → Public API
- **Контекстная безопасность**: ✅ Все методы принимают `context.Context`
- **Тестируемость**: ✅ Мок-реализация в плане
- **Backward compatibility**: ✅ Capability pattern сохраняет существующий `LLMProvider`
