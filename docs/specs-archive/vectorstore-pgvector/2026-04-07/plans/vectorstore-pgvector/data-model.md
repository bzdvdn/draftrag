# VectorStore pgvector (PostgreSQL) для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: AC-002, AC-003, AC-004, AC-005
- Связанные `DEC-*`: DEC-002, DEC-003, DEC-004
- Значимое изменение data model требуется: вводится таблица хранения чанков для pgvector-backed VectorStore.

## Сущности

### DM-001 ChunkRow (строка таблицы чанков)

- Назначение: persisted представление `domain.Chunk` для поиска по embedding в PostgreSQL.
- Источник истины: запись создаётся/обновляется через `VectorStore.Upsert`.
- Инварианты:
  - `id` уникален (primary key).
  - `embedding` имеет фиксированную размерность (EmbeddingDimension), значения конечны (не NaN/Inf).
  - `position >= 0`.
- Связанные `AC-*`: AC-003, AC-004, AC-005
- Связанные `DEC-*`: DEC-003, DEC-004
- Поля (минимальная схема v1):
  - `id` — text, required, primary key (соответствует `Chunk.ID`)
  - `parent_id` — text, required (соответствует `Chunk.ParentID`)
  - `content` — text, required (соответствует `Chunk.Content`)
  - `position` — integer, required (соответствует `Chunk.Position`)
  - `embedding` — vector(<dim>), required (соответствует `Chunk.Embedding`)
  - `created_at` — timestamptz, optional (по желанию; если добавляется — заполняется по умолчанию `now()`)
  - `updated_at` — timestamptz, optional (по желанию; если добавляется — обновляется триггером или на приложении)
- Жизненный цикл:
  - создаётся/обновляется при `Upsert`
  - удаляется при `Delete(id)` (v1: удаление по chunk id, каскад по parent_id — вне scope)
  - используется в `Search` (выборка topK по distance)
- Замечания по консистентности:
  - изменение `EmbeddingDimension` для существующей таблицы не поддерживается в v1 (вне scope); это должно приводить к явной ошибке/несовместимости.

## Связи

- Значимых межсущностных связей внутри этой фичи нет (одна таблица).
- `parent_id` — логическая связь на Document (вне БД-контрактов core, используется только для tracing/удаления в будущем).

## Производные правила

- Score вычисляется как `score = 1 - cosine_distance`, где `cosine_distance` берётся из оператора pgvector для cosine distance.

## Переходы состояний

- Upsert:
  - нет записи -> insert
  - есть запись -> update (по `id`)
- Delete:
  - запись существует -> delete
  - запись отсутствует -> no-op (или успешное удаление без ошибки; фиксируется в tasks)

## Вне scope

- Отдельная таблица документов.
- Поля metadata и фильтрация по ним.
- Версионирование схемы/миграции и backfill.

