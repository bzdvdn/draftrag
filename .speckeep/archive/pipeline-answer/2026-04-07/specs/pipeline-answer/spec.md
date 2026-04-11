# Pipeline.Answer для draftRAG (полный RAG-цикл)

## Scope Snapshot

- In scope: добавление публичного метода(ов) `Pipeline.Answer*`, которые выполняют полный RAG-цикл: embed вопроса → поиск контекста → сборка промпта → вызов `LLMProvider.Generate` → возврат текстового ответа.
- Out of scope: стриминг ответа, tool-calling, structured outputs, memory/chat history, reranking, компрессия/суммаризация контекста, лимитирование по токенам, кэширование, ретраи/backoff.

## Цель

Дать пользователю draftRAG один простой публичный вызов для получения ответа на вопрос по проиндексированным документам, не заставляя его вручную вызывать `Query*` и самостоятельно собирать prompt для LLM.

Успех измеряется тем, что:
- метод компилируется и доступен из `pkg/draftrag`,
- корректно использует `Embedder`, `VectorStore` и `LLMProvider`,
- уважает отмену контекста/таймауты,
- полностью тестируется без внешней сети.

## Основной сценарий

1. Разработчик собирает `p := draftrag.NewPipeline(store, llm, embedder)`.
2. Индексирует документы через `p.Index(ctx, docs)`.
3. Вызывает `answer, err := p.Answer(ctx, question)`.
4. Получает строковый `answer`.

## Scope

- Public API:
  - `(*Pipeline) Answer(ctx, question string) (string, error)` — использует `topK` по умолчанию (5).
  - `(*Pipeline) AnswerTopK(ctx, question string, topK int) (string, error)` — настраиваемый `topK`.
- Application слой:
  - добавление use-case метода в `internal/application.Pipeline`, который выполняет RAG-цикл поверх существующих зависимостей.
- Prompt contract (v1, минимальный):
  - фиксированный system prompt и детерминированный формат user message (контекст + вопрос).
- Тестирование:
  - unit-тесты без внешней сети (моки/заглушки зависимостей).

## Контекст

- Уже есть core-поток: `Index` и `Query` (retrieval) в `internal/application/pipeline.go`, а в `pkg/draftrag` есть публичный wrapper.
- `LLMProvider.Generate(ctx, systemPrompt, userMessage)` уже определён в domain и реализован OpenAI-compatible провайдером.
- Конституция: `context.Context` первым параметром, `nil` context — panic, минимальные зависимости, русские godoc для публичных символов.

## Требования

- RQ-001 ДОЛЖЕН существовать публичный метод `Pipeline.Answer(ctx, question)` в `pkg/draftrag`.
- RQ-002 ДОЛЖЕН существовать публичный метод `Pipeline.AnswerTopK(ctx, question, topK)` в `pkg/draftrag`.
- RQ-003 `Answer*` ДОЛЖНЫ использовать `Embedder` + `VectorStore.Search` для получения релевантных чанков (как retrieval-стадия).
- RQ-004 `Answer*` ДОЛЖНЫ вызывать `LLMProvider.Generate` и возвращать его текстовый ответ.
- RQ-005 `Answer*` ДОЛЖНЫ передавать контекст и вопрос в `Generate` как:
  - `systemPrompt` — фиксированная строка (см. Prompt Contract),
  - `userMessage` — детерминированная строка, содержащая “Контекст” и “Вопрос”.
- RQ-006 Все методы `Answer*` ДОЛЖНЫ уважать `ctx.Done()` и возвращать `context.Canceled`/`context.DeadlineExceeded` без оборачивания в несопоставимые ошибки.
- RQ-007 Валидация входных данных ДОЛЖНА соответствовать существующим публичным ошибкам:
  - пустой `question` -> `ErrEmptyQuery`
  - `topK <= 0` -> `ErrInvalidTopK`
- RQ-008 Тесты ДОЛЖНЫ проходить без внешней сети (`go test ./...` без реальных API).
- RQ-009 Публичные методы `Answer*` ДОЛЖНЫ иметь godoc на русском языке.

## Prompt Contract (v1, минимальный и детерминированный)

### System prompt (фиксированный)

```
Ты — помощник. Отвечай на вопрос, используя предоставленный контекст. Если контекста недостаточно — честно скажи, что информации недостаточно.
```

### User message (формат)

```
Контекст:
<chunk-1>
<chunk-2>
...

Вопрос:
<question>
```

Правила:
- чанки включаются в порядке убывания score (как их вернул `VectorStore.Search`);
- если чанков нет — секция “Контекст” остаётся пустой (после двоеточия перевод строки).

## Критерии приемки

### AC-001 Публичные методы Answer доступны и компилируются

- **Given** пользователь импортирует `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он вызывает `p.Answer(ctx, q)` и `p.AnswerTopK(ctx, q, k)`
- **Then** код компилируется
- Evidence: compile-time тест в `pkg/draftrag`.

### AC-002 Answer вызывает retrieval и LLM.Generate

- **Given** заглушки `VectorStore`, `Embedder`, `LLMProvider`, возвращающие детерминированные значения
- **When** вызывается `Answer(ctx, question)`
- **Then** выполняется `Embed(question)` → `Search(embedding, topK)` → `Generate(systemPrompt, userMessage)` и возвращается текст ответа
- Evidence: unit-тест, проверяющий вызовы и возвращаемое значение.

### AC-003 Prompt Contract соблюдается

- **Given** известный набор чанков в результате retrieval
- **When** вызывается `AnswerTopK(ctx, question, topK)`
- **Then** `LLMProvider.Generate` получает `systemPrompt` и `userMessage` в формате Prompt Contract (v1)
- Evidence: unit-тест, сравнивающий строки (или их ключевые части) детерминированно.

### AC-004 Валидация question/topK маппится в публичные ошибки

- **Given** пустой `question` или `topK <= 0`
- **When** вызывается `Answer*`
- **Then** возвращаются `ErrEmptyQuery` / `ErrInvalidTopK` соответственно
- Evidence: unit-тесты.

### AC-005 Контекстная отмена работает

- **Given** отменённый или просроченный `ctx`
- **When** вызывается `Answer*`
- **Then** метод возвращает `context.Canceled` / `context.DeadlineExceeded` и не выполняет лишнюю работу
- Evidence: unit-тест (в тестовом сценарии — не позже 100мс).

## Вне scope

- Реранкинг (re-rank) найденных чанков.
- Ограничение размера контекста по токенам/символам.
- Стриминг ответов.
- Расширенные промпт-шаблоны/плейсхолдеры/многоязычность.

## Открытые вопросы

- Нужен ли в v1 отдельный метод/опции для кастомизации system prompt, или оставляем фиксированный prompt до появления реальной потребности?

