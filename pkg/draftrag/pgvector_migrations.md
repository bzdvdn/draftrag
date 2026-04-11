# pgvector migrations (v1)

Этот файл описывает, где лежат миграции для pgvector store и как их применять.

## Где лежат миграции

SQL-миграции находятся в:

- `pkg/draftrag/migrations/pgvector/`

Файлы именуются так, чтобы порядок применения был очевиден (lexicographic).

## Как применять

Есть два поддерживаемых подхода:

### Внешний инструмент миграций (рекомендуется для production)

Применяйте SQL файлы из `pkg/draftrag/migrations/pgvector/` отдельным шагом деплоя (deploy job / init container).

Важно:

- `0000_pgvector_extension.sql` требует прав на `CREATE EXTENSION`. Если прав нет — применяйте extension отдельной процедурой DBA или пропустите этот шаг (при условии, что extension уже установлен).
- Файлы используют плейсхолдеры (`{{TABLE}}`, `{{DIM}}`, и т.д.). Их нужно заменить на ваши значения:
  - `{{TABLE}}` — имя таблицы чанков (quoted identifier, например `"draftrag_chunks"`),
  - `{{DIM}}` — размерность embedding (например `1536`).

### Через явный вызов runtime helper (подходит для “своим кодом” в deploy job)

Если вы хотите применять “те же самые” миграции программно, используйте:

- `draftrag.MigratePGVector(ctx, db, opts)` или `draftrag.SetupPGVector(ctx, db, opts)`

Этот путь:

- использует те же SQL файлы (через `go:embed`),
- подставляет `TableName/EmbeddingDimension` из `PGVectorOptions`,
- остаётся явным (не запускается автоматически при создании store).

## Индексы и метрика

Similarity search в коде использует cosine distance (`<=>`) и вычисляет `score = 1 - cosine_distance`.

Embedding-индекс создаётся/проверяется runtime helper’ом (`SetupPGVector/MigratePGVector`) с учётом:

- `PGVectorOptions.IndexMethod` (`ivfflat` по умолчанию, либо `hnsw`)
- `PGVectorOptions.Lists` (для `ivfflat`, по умолчанию `100`)

## Совместимость

Миграции добавлены аддитивно. Они не включаются автоматически: пользователь сам выбирает, где и когда выполнять DDL.

