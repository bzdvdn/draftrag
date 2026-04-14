# VectorStore pgvector: guardrails для размерности эмбеддингов (v1) — Модель данных

## Scope

- Связанные `AC-*`: `AC-001`, `AC-002`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`
- Persisted data model не меняется. Эта фича добавляет только классификатор ошибки и поведение ранней валидации.

## Сущности

### DM-001 EmbeddingDimensionConstraint (ограничение размерности)

- Назначение: гарантировать, что `Upsert`/`Search` получают embedding длины `== EmbeddingDimension` и возвращают классифицируемую ошибку при нарушении.
- Источник истины:
  - конфигурация: `PGVectorOptions.EmbeddingDimension` и поле `embeddingDim` внутри `PGVectorStore`;
  - ошибка: `ErrEmbeddingDimensionMismatch`.
- Инварианты:
  - `len(embedding) == embeddingDim` для успешного выполнения операции.
  - при нарушении возвращается ошибка, сравнимая через `errors.Is`.
- Связанные `AC-*`: `AC-001`, `AC-002`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`
- Поля:
  - `embeddingDim` — `int`, required, > 0
  - `embedding` — `[]float64`, required
- Жизненный цикл:
  - проверяется на каждом вызове `Upsert`/`Search`/`SearchWithFilter` перед обращением к БД.

## Связи

- Нет значимых межсущностных связей: это локальная валидация для операций store.

## Производные правила

- “Dimension mismatch” классифицируется через sentinel error + wrap, а диагностические детали (`got/want`) остаются в тексте ошибки.

## Переходы состояний

- Не применимо.

## Вне scope

- Хранение dimension в БД и валидация “на старте”.
- Поддержка нескольких размерностей embeddings в одной таблице.

