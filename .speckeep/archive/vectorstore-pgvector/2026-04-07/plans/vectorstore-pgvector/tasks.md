# VectorStore pgvector (PostgreSQL) для draftRAG — Задачи

## Phase Contract

Inputs: `.draftspec/plans/vectorstore-pgvector/plan.md`, `.draftspec/plans/vectorstore-pgvector/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи становятся расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/vectorstore/pgvector.go | T1.2, T2.1, T2.2 |
| pkg/draftrag/pgvector.go | T1.1, T2.3 |
| internal/infrastructure/vectorstore/pgvector_test.go | T3.1 |
| pkg/draftrag/pgvector_test.go | T3.2 |

## Фаза 1: Публичный API и каркас реализации

Цель: зафиксировать стабильную API поверхность и минимальный каркас store до написания DDL/SQL.

- [x] T1.1 Создать `pkg/draftrag/pgvector.go` — добавить `PGVectorOptions`, `NewPGVectorStore(db, opts)` и `SetupPGVector(ctx, db, opts)`; публичные функции принимают `context.Context` первым параметром; `CreateExtension` — опционален (best-effort или явная ошибка при отсутствии прав). Touches: pkg/draftrag/pgvector.go
- [x] T1.2 Создать `internal/infrastructure/vectorstore/pgvector.go` — каркас `PGVectorStore` (хранит `*sql.DB` + options), compile-time соответствие `domain.VectorStore` (анонимное присваивание/var _ ...). Touches: internal/infrastructure/vectorstore/pgvector.go

## Фаза 2: Реализация store и schema helper

Цель: реализовать корректные SQL-операции и минимально идемпотентный DDL.

- [x] T2.1 Реализовать `Upsert(ctx, chunk)` и `Delete(ctx, id)` в `PGVectorStore` с использованием `ExecContext`; уважать отмену контекста; `Delete` для отсутствующего id — no-op без ошибки. Touches: internal/infrastructure/vectorstore/pgvector.go
- [x] T2.2 Реализовать `Search(ctx, embedding, topK)` в `PGVectorStore`: валидация `topK > 0`, сортировка по score desc, формула `score = 1 - cosine_distance`, score в [-1, 1], `TotalFound` отражает количество кандидатов до topK (если вычисляется) или количество найденных строк (если иначе). Touches: internal/infrastructure/vectorstore/pgvector.go
- [x] T2.3 Реализовать `SetupPGVector(ctx, db, opts)` идемпотентно: создание таблицы чанков (DM-001) и индекса по embedding; создание расширения — только при `CreateExtension=true`. Touches: pkg/draftrag/pgvector.go

## Фаза 3: Интеграционные тесты (opt-in по DSN)

Цель: дать доказательства AC при наличии реальной PostgreSQL, не ломая `go test ./...` без окружения.

- [x] T3.1 Создать `internal/infrastructure/vectorstore/pgvector_test.go` — интеграционные тесты Upsert/Delete/Search; по умолчанию `t.Skip()` если не задан `PGVECTOR_TEST_DSN` (или согласованная переменная); тесты проверяют порядок, `topK`, диапазон score и идемпотентность Setup. Touches: internal/infrastructure/vectorstore/pgvector_test.go
- [x] T3.2 Создать `pkg/draftrag/pgvector_test.go` — интеграционный тест совместимости с `Pipeline` (Index + QueryTopK) с тем же DSN/skip правилом. Touches: pkg/draftrag/pgvector_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2
- AC-002 -> T1.1, T2.3, T3.1
- AC-003 -> T2.1, T3.1
- AC-004 -> T2.2, T3.1
- AC-005 -> T3.2

## Заметки

- Интеграционные тесты всегда opt-in: без DSN — `t.Skip()`, чтобы сохранять требование RQ-005.
- Механизм индекса (IVFFlat/HNSW) финализируется в реализации; в tasks не фиксируем конкретный метод жёстко, но требуем индекс как минимум (см. AC-002).
