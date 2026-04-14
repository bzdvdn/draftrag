# VectorStore pgvector: production-ready (migrations, индексы, фильтры, лимиты/таймауты) Модель данных

## Scope

- Связанные `AC-*`: `AC-001`, `AC-002`, `AC-003`
- Связанные `DEC-*`: `DEC-002`, `DEC-003`

## Сущности

### DM-001 PGVectorChunk (строка таблицы чанков)

- Назначение: хранение нормализованных retrieval-чанков для семантического поиска.
- Источник истины: PostgreSQL таблица `<table>` (по умолчанию `draftrag_chunks`, задаётся `TableName`).
- Инварианты:
  - `id` уникален и стабильный.
  - `parent_id` не пустой.
  - `embedding` имеет фиксированную размерность `EmbeddingDimension`.
  - `position >= 0`.
  - `created_at` и `updated_at` выставлены и не NULL.
- Связанные `AC-*`: `AC-001`, `AC-003`
- Связанные `DEC-*`: `DEC-002`
- Поля:
  - `id` - `text`, required, PK, идентификатор чанка (например `doc#0` или `doc:0`).
  - `parent_id` - `text`, required, ID родителя (обычно `Document.ID`), используется для фильтра `ParentID`.
  - `content` - `text`, required, текст чанка.
  - `position` - `integer`, required, позиция чанка внутри родителя.
  - `embedding` - `vector(<EmbeddingDimension>)`, required, embedding-вектор.
  - `metadata` - `jsonb`, required, default `{}`, зарезервировано под будущие фильтры (в этой фиче не используется).
  - `created_at` - `timestamptz`, required, default `now()`, момент создания строки.
  - `updated_at` - `timestamptz`, required, default `now()`, момент последнего обновления строки.
- Жизненный цикл:
  - создаётся: `Upsert` для нового `id`.
  - обновляется: `Upsert` для существующего `id` (обновляет контент/position/embedding и выставляет `updated_at`).
  - удаляется: `Delete(id)` удаляет строку.
- Замечания по консистентности:
  - недопустимо хранить embedding другой размерности — это должно отлавливаться валидацией до SQL.
  - недопустимо возвращать чанки с `parent_id` вне заданного фильтра.

### DM-002 SchemaMigration (версия схемы)

- Назначение: фиксировать прогресс применения миграций и обеспечить идемпотентный upgrade.
- Источник истины: таблица `schema_migrations` (в той же БД; нейминг фиксируется в migrator).
- Инварианты:
  - версия монотонно возрастает.
  - миграции применяются строго по порядку.
- Связанные `AC-*`: `AC-001`, `AC-002`
- Связанные `DEC-*`: `DEC-002`
- Поля (минимально):
  - `version` - `integer`, required, PK или unique, номер миграции.
  - `applied_at` - `timestamptz`, required, default `now()`.

## Связи

- `DM-001 -> DM-002`: прямой связи нет; `schema_migrations` управляет формой `DM-001`, но не ссылается на неё.

## Производные правила

- `updated_at` обновляется на каждом `Upsert` (через SQL `SET updated_at = now()` или через trigger — в рамках этой фичи предпочтительно без trigger, чтобы не усложнять схему).

## Переходы состояний

- `Upsert(new id)` -> отсутствует -> существует
- `Upsert(existing id)` -> существует -> существует (обновлён)
- `Delete(id)` -> существует -> отсутствует

## Вне scope

- Многоарендность: `tenant_id`, `namespace`, RLS.
- Мягкое удаление (soft delete), TTL/архивация.
- Фильтры по `metadata` (кроме резерва поля).
