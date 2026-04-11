-- Migration: 0003_add_bm25
-- Description: Добавляет поддержку полнотекстового поиска (BM25) через tsvector

-- Up migration

-- 1. Добавляем tsvector колонку для полнотекстового поиска
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS content_tsv tsvector;

-- 2. Создаём GIN-индекс для быстрого полнотекстового поиска
CREATE INDEX IF NOT EXISTS idx_chunks_content_tsv ON chunks USING GIN (content_tsv);

-- 3. Создаём функцию для автоматического обновления tsvector при insert/update
CREATE OR REPLACE FUNCTION chunks_content_tsv_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.content_tsv := to_tsvector('english', NEW.content);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 4. Создаём триггер для автоматического обновления tsvector
DROP TRIGGER IF EXISTS trigger_chunks_content_tsv ON chunks;
CREATE TRIGGER trigger_chunks_content_tsv
    BEFORE INSERT OR UPDATE ON chunks
    FOR EACH ROW
    EXECUTE FUNCTION chunks_content_tsv_update();

-- 5. Backfill: обновляем существующие записи
UPDATE chunks SET content_tsv = to_tsvector('english', content) WHERE content_tsv IS NULL;

-- Down migration (раскомментировать при необходимости отката)
-- DROP TRIGGER IF EXISTS trigger_chunks_content_tsv ON chunks;
-- DROP FUNCTION IF EXISTS chunks_content_tsv_update();
-- DROP INDEX IF EXISTS idx_chunks_content_tsv;
-- ALTER TABLE chunks DROP COLUMN IF EXISTS content_tsv;
