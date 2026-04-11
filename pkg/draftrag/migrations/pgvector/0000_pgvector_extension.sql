-- Создание extension pgvector.
--
-- Примечание: часто требует повышенных прав. Если прав нет, применяйте схему без этого шага,
-- либо включайте extension на уровне DBA/администратора кластера.
CREATE EXTENSION IF NOT EXISTS vector;

