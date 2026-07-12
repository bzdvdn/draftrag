# CJK Tokenization — Модель данных

## Scope

- Связанные `AC-*`: все
- Связанные `DEC-*`: DEC-003
- Статус: `no-change`
- Фича не добавляет и не меняет persisted entities, value objects, state transitions или contract-relevant payload shapes.

## No-Change Stub

- Статус: `no-change`
- Причина: фича модифицирует только внутреннюю логику `splitSentences` и `isSentenceBoundary`. Chunker interface, Document, Chunk — без изменений. Публичное API (pkg/draftrag) не затронуто.
- Revisit triggers:
  - появляется новый тип Chunker
  - меняется Chunk или Document структура
  - API/event payload shape нужно отслеживать
