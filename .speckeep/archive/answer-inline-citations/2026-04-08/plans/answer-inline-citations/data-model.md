# Answer: inline citations в тексте ответа (v1) — Модель данных

## Scope

- Связанные `AC-*`: `AC-001`
- Связанные `DEC-*`: `DEC-001`
- Изменение модели данных ограничено добавлением value-type для результата API; персистентная модель не меняется.

## Сущности

### DM-001 InlineCitation

- Назначение: детерминированный маппинг номера цитаты `n` (используется в тексте как `[n]`) на конкретный retrieval-источник.
- Источник истины: backend (pipeline) нумерует retrieval-чанки в том же порядке, в котором они попадают в prompt.
- Инварианты:
  - `Number` начинается с 1 и монотонно возрастает на 1 внутри одного вызова.
  - `Chunk` соответствует одному элементу из retrieval результата, реально доступного для цитирования.
- Связанные `AC-*`: `AC-001`
- Связанные `DEC-*`: `DEC-001`
- Поля:
  - `Number` - `int`, required, номер цитаты для текста ответа.
  - `Chunk` - `RetrievedChunk`, required, retrieval evidence (chunk + score).

## Связи

- `DM-001 InlineCitation -> RetrievedChunk`: 1:1 (value embedding), ownership у результата метода Answer*WithInlineCitations.

## Вне scope

- Хранение citations в базе данных.
- Автоматический parse/normalization фактов или “span-level” attribution.

