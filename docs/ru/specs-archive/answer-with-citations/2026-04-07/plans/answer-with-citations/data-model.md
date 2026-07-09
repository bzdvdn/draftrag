# AnswerWithCitations для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: `AC-002`, `AC-003`
- Связанные `DEC-*`: `DEC-001`
- Persisted data model не меняется.

## Сущности

### DM-001 AnswerWithCitationsResult (результат вызова)

- Назначение: вернуть пользователю и ответ, и источники (retrieval evidence).
- Источник истины: возвращаемые значения методов `Answer*WithCitations`.
- Инварианты:
  - `answer` — строка (может быть пустой при ошибке `Generate`).
  - `retrieval` — `domain.RetrievalResult`, полученный из `VectorStore.Search`.
  - `err` — ошибка, если возникла на любом этапе.
- Связанные `AC-*`: `AC-002`, `AC-003`
- Связанные `DEC-*`: `DEC-001`
- Поля:
  - `answer` — `string`
  - `retrieval` — `domain.RetrievalResult` (экспортируется как `draftrag.RetrievalResult`)

## Связи

- `retrieval` используется для построения prompt (как и в обычном Answer), но также возвращается наружу как evidence.

## Производные правила

- При ошибке `Generate` retrieval возвращается (partial result), чтобы пользователь мог отобразить источники/диагностику.

## Вне scope

- Нумерация/разметка цитат внутри текста ответа.

