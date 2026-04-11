-- Базовая таблица чанков для pgvector store.
--
-- Плейсхолдеры (заменяются runtime-migrator'ом или пользователем вручную):
-- - {{TABLE}} — quoted identifier таблицы (например "draftrag_chunks")
-- - {{DIM}}   — размерность embedding (например 1536)
CREATE TABLE IF NOT EXISTS {{TABLE}} (
    id TEXT PRIMARY KEY,
    parent_id TEXT NOT NULL,
    content TEXT NOT NULL,
    position INTEGER NOT NULL,
    embedding VECTOR({{DIM}}) NOT NULL
);

