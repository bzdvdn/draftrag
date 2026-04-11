-- Расширение схемы (v2): metadata/timestamps + индексы для фильтров.
--
-- Плейсхолдеры (заменяются runtime-migrator'ом или пользователем вручную):
-- - {{TABLE}}            — quoted identifier таблицы
-- - {{PARENT_ID_INDEX}}  — quoted identifier индекса по parent_id
-- - {{PARENT_POS_INDEX}} — quoted identifier композитного индекса (parent_id, position)
ALTER TABLE {{TABLE}} ADD COLUMN IF NOT EXISTS metadata jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE {{TABLE}} ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE {{TABLE}} ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS {{PARENT_ID_INDEX}} ON {{TABLE}} (parent_id);
CREATE INDEX IF NOT EXISTS {{PARENT_POS_INDEX}} ON {{TABLE}} (parent_id, position);

