# VectorStore pgvector: миграции и стабильная схема (v1) — Задачи

## Phase Contract

Inputs: `.speckeep/plans/vectorstore-pgvector-migrations/plan.md`, `.speckeep/plans/vectorstore-pgvector-migrations/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи становятся расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/migrations/pgvector/ | T1.1, T2.2 |
| pkg/draftrag/pgvector_migrations.md | T1.2 |
| pkg/draftrag/pgvector_migrate.go | T2.1, T2.2 |
| pkg/draftrag/pgvector.go | T2.2 |
| pkg/draftrag/pgvector_migrations_assets_test.go | T3.1 |

## Фаза 1: Migration assets + документация

Цель: сделать миграции “находимыми” и однозначно применимыми, не трогая поведение runtime кода.

- [x] T1.1 Добавить директорию `pkg/draftrag/migrations/pgvector/` с версионированными SQL-миграциями, покрывающими базовую схему pgvector store (DM-001..DM-002): extension (опционально), таблица чанков и необходимые индексы (embedding + `parent_id`-фильтры). Имена файлов должны задавать очевидный порядок применения (lexicographic). Touches: pkg/draftrag/migrations/pgvector/
- [x] T1.2 Добавить краткую инструкцию применения миграций (порядок, обязательные/опциональные шаги, права на `CREATE EXTENSION`) в `pkg/draftrag/pgvector_migrations.md`. Touches: pkg/draftrag/pgvector_migrations.md

## Фаза 2: Согласование runtime migrator с SQL

Цель: избежать рассинхронизации между “кодом миграций” и “SQL файлами”, сохранив backward compatibility.

- [x] T2.1 Обновить `pkg/draftrag/pgvector_migrate.go`, чтобы источник истины DDL был общий с SQL-миграциями (например через `go:embed` и применение тех же файлов), сохраняя явный вызов `SetupPGVector` (без авто-миграций), текущую схему версионирования (если используется) и идемпотентность. Touches: pkg/draftrag/pgvector_migrate.go
- [x] T2.2 Обновить комментарии/документацию вокруг `SetupPGVector/MigratePGVector`: явно указать рекомендованный production-подход (применять SQL миграции отдельным шагом) и описать, как runtime helper соотносится с SQL файлами (эквивалентность или ограничения), включая дефолты по embedding-индексу/метрике и поддержку фильтров. Touches: pkg/draftrag/pgvector.go, pkg/draftrag/pgvector_migrate.go, pkg/draftrag/pgvector_migrations.md

## Фаза 3: Проверка (без внешней сети)

Цель: оставить автоматические доказательства AC-001/AC-003 без подключения к PostgreSQL.

- [x] T3.1 Добавить `pkg/draftrag/pgvector_migrations_assets_test.go`: unit-тест, который проверяет, что migration assets существуют, имеют детерминированный порядок (по именам) и содержат ожидаемые “якоря” DDL (например `CREATE TABLE`, `CREATE INDEX`, `CREATE EXTENSION` в отдельной миграции). Touches: pkg/draftrag/pgvector_migrations_assets_test.go
- [x] T3.2 Прогнать `go test ./...` и убедиться, что изменения аддитивны и ничего не сломано. Touches: (go test ./...)

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T3.1
- AC-002 -> T1.1, T2.1, T2.2
- AC-003 -> T3.2

## Заметки

- Интеграционные проверки “реально применить миграции к PostgreSQL” остаются вне требований v1 (в проекте уже есть opt-in DSN тесты), но DDL в SQL файлах должен быть согласован с текущим поведением store.
