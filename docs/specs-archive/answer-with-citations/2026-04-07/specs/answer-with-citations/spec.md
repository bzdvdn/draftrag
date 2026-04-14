# AnswerWithCitations: ответ + источники (retrieval evidence) для draftRAG

## Scope Snapshot

- In scope: добавить публичные методы pipeline, которые возвращают не только строковый ответ LLM, но и “evidence” — retrieval результат (чанки + score), чтобы пользователь мог показывать источники/цитаты.
- Out of scope: автоматическая разметка цитат внутри ответа, корректная нумерация ссылок в тексте ответа, reranking, дедупликация источников, стриминг, structured outputs.

## Цель

Сделать `Answer*` более полезным для RAG-приложений: помимо ответа возвращать контекст, на основе которого ответ был сформирован (retrieval result). Это повышает доверие, позволяет строить UI “Sources” и облегчает отладку.

## Основной сценарий

1. Пользователь вызывает `answer, sources, err := p.AnswerWithCitations(ctx, question)`.
2. Получает:
   - `answer` — текст LLM,
   - `sources` — `RetrievalResult` (чанки с score и query text),
   - `err` — ошибку (если есть).

## Scope

- Public API (`pkg/draftrag`):
  - `(*Pipeline) AnswerWithCitations(ctx, question string) (string, RetrievalResult, error)`
  - `(*Pipeline) AnswerTopKWithCitations(ctx, question string, topK int) (string, RetrievalResult, error)`
- Application (`internal/application`):
  - use-case метод, который делает retrieval (Embed+Search), строит prompt и вызывает `Generate`, возвращая и `answer`, и retrieval результат.
- Testing:
  - unit-тесты без внешней сети: корректное возвращение retrieval результата, корректный порядок вызовов и маппинг ошибок.

## Контекст

- Уже есть `Pipeline.Answer`/`AnswerTopK`, но они возвращают только `string`.
- Retrieval результат уже моделируется как `domain.RetrievalResult` и экспортирован как `draftrag.RetrievalResult`.
- Prompt уже формируется в application слое; лимиты контекста могут быть включены через options (не требуется для этой фичи, но не должны ломать поведение).

## Требования

- RQ-001 ДОЛЖНЫ существовать публичные методы `AnswerWithCitations` и `AnswerTopKWithCitations` в `pkg/draftrag`.
- RQ-002 `AnswerWithCitations` ДОЛЖЕН использовать `DefaultTopK` pipeline (как `Answer`).
- RQ-003 Методы ДОЛЖНЫ возвращать `RetrievalResult`, содержащий чанки, которые были получены на этапе retrieval.
- RQ-004 Методы ДОЛЖНЫ вызывать `LLMProvider.Generate` и возвращать его текстовый ответ.
- RQ-005 Валидация входных данных должна соответствовать существующим публичным ошибкам:
  - пустой `question` -> `ErrEmptyQuery`
  - `topK <= 0` -> `ErrInvalidTopK`
- RQ-006 Контекстная отмена ДОЛЖНА работать: при отмене возвращать `context.Canceled`/`context.DeadlineExceeded`.
- RQ-007 Тесты ДОЛЖНЫ проходить без внешней сети.
- RQ-008 Публичные методы ДОЛЖНЫ иметь godoc на русском.

## Критерии приемки

### AC-001 Методы Answer*WithCitations доступны и компилируются

- **Given** пользователь импортирует `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он вызывает `AnswerWithCitations` и `AnswerTopKWithCitations`
- **Then** код компилируется
- Evidence: compile-time тест в `pkg/draftrag`.

### AC-002 Возвращается retrieval evidence

- **Given** заглушки `VectorStore`, `Embedder`, `LLMProvider`, и search возвращает известный `RetrievalResult`
- **When** вызывается `AnswerTopKWithCitations`
- **Then** возвращаемый `RetrievalResult` совпадает с тем, что вернул `VectorStore.Search` (включая `Chunks` и `QueryText`)
- Evidence: unit-тест.

### AC-003 Ответ LLM возвращается как answer string

- **Given** `LLMProvider.Generate` возвращает `"ok"`
- **When** вызывается `AnswerTopKWithCitations`
- **Then** `answer == "ok"`, `err == nil`
- Evidence: unit-тест.

### AC-004 Backward compatibility

- **Given** существующие методы `Answer`/`AnswerTopK`
- **When** проект собирается и запускаются тесты
- **Then** новые методы добавлены аддитивно и ничего не сломано
- Evidence: `go test ./...` проходит.

## Вне scope

- Встраивание “цитат” внутрь ответа (например, [1], [2]).
- Автоматическая очистка/дедупликация источников.

## Открытые вопросы

- Возвращаем ли `RetrievalResult` даже если `Generate` вернул ошибку (partial result), или возвращаем пустой результат? (предпочтение: возвращать retrieval result, если retrieval успел выполниться, чтобы облегчить диагностику).

