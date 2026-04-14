# Pipeline.Answer для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: `AC-002`, `AC-003`
- Связанные `DEC-*`: `DEC-002`, `DEC-003`
- Persisted data model не меняется: `Answer*` вычисляет prompt и возвращает `string`.

## Сущности

### DM-001 PromptContractV1 (вычисляемый prompt)

- Назначение: детерминированно формировать вход для `LLMProvider.Generate` из найденного контекста и вопроса.
- Источник истины: application слой (`internal/application`), где формируется `systemPrompt` и `userMessage`.
- Инварианты:
  - `systemPrompt` — фиксированная строка v1.
  - `userMessage` содержит секции “Контекст:” и “Вопрос:” в фиксированном формате.
  - чанки включаются в порядке, в котором они пришли из `VectorStore.Search` (score-desc как контракт хранилища).
  - при отсутствии чанков “Контекст:” присутствует, но список пустой.
- Связанные `AC-*`: `AC-003`
- Связанные `DEC-*`: `DEC-002`
- Поля:
  - `systemPrompt` — `string`, fixed.
  - `userMessage` — `string`, derived from `RetrievalResult.Chunks` and `question`.
- Жизненный цикл:
  - создаётся при каждом вызове `Answer*`.
  - не хранится.
- Замечания по консистентности:
  - формат должен оставаться стабильным для тестов; изменения требуют обновления spec/plan и тестов.

### DM-002 AnswerFlowInputs (входы use-case)

- Назначение: минимальный контракт данных, необходимых для выполнения RAG-цикла.
- Источник истины: параметры `Answer*` и возвращаемые значения зависимостей.
- Инварианты:
  - `question` — непустая строка (после TrimSpace).
  - `topK > 0`.
- Связанные `AC-*`: `AC-002`, `AC-004`, `AC-005`
- Связанные `DEC-*`: `DEC-003`
- Поля:
  - `question` — `string`, required.
  - `topK` — `int`, required.
  - `embedding` — `[]float64`, derived (результат `Embed`).
  - `retrieval` — `domain.RetrievalResult`, derived (результат `Search`).
  - `answer` — `string`, derived (результат `Generate`).
- Жизненный цикл:
  - создаются/вычисляются на время одного вызова.

## Связи

- `DM-002 -> DM-001`: результаты retrieval и `question` используются для построения prompt.

## Производные правила

- `Answer(ctx, question)` эквивалентен `AnswerTopK(ctx, question, defaultTopK=5)`.
- Если `ctx.Err() != nil`, метод возвращает соответствующую ошибку и не вызывает зависимости.

## Переходы состояний

- Отдельных переходов состояний нет: метод stateless.

## Вне scope

- История диалога, память, трейс/метрики, кэширование результатов.

