# VectorStore pgvector: guardrails для размерности эмбеддингов (v1) — Задачи

## Phase Contract

Inputs: `.speckeep/plans/vectorstore-pgvector-dimension-guard/plan.md`, `.speckeep/plans/vectorstore-pgvector-dimension-guard/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи становятся расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/models.go | T1.1 |
| pkg/draftrag/errors.go | T1.2 |
| internal/infrastructure/vectorstore/pgvector.go | T2.1 |
| pkg/draftrag/pgvector.go | T2.2 |
| pkg/draftrag/pgvector_dimension_guard_test.go | T3.1 |

## Фаза 1: Классификатор ошибки

Цель: ввести стабильный классификатор для `errors.Is` без нарушения зависимостей.

- [x] T1.1 Добавить sentinel `ErrEmbeddingDimensionMismatch` в `internal/domain/models.go` (как `var` рядом с другими `Err*`). Touches: internal/domain/models.go
- [x] T1.2 Экспортировать ошибку в публичный слой как re-export: добавить `ErrEmbeddingDimensionMismatch = domain.ErrEmbeddingDimensionMismatch` в `pkg/draftrag/errors.go`, чтобы пользователи могли делать `errors.Is(err, draftrag.ErrEmbeddingDimensionMismatch)`. Touches: pkg/draftrag/errors.go

## Фаза 2: Инфраструктурная валидация

Цель: возвращать типизированную ошибку при mismatch и не трогать happy-path.

- [x] T2.1 Обновить `internal/infrastructure/vectorstore/pgvector.go`: в `validateEmbedding` при `len(embedding) != expectedDim` возвращать wrap на sentinel (`fmt.Errorf(\"%w: got=%d want=%d\", domain.ErrEmbeddingDimensionMismatch, ...)`), сохранив остальную валидацию (`nil`, non-finite) без изменения контракта. Touches: internal/infrastructure/vectorstore/pgvector.go
- [x] T2.2 (Если нужно) уточнить godoc/комментарии в `pkg/draftrag/pgvector.go`, что `EmbeddingDimension` — это “Dimension” и mismatch классифицируется `ErrEmbeddingDimensionMismatch`. Touches: pkg/draftrag/pgvector.go

## Фаза 3: Тесты (без внешней сети)

Цель: доказать AC без Postgres.

- [x] T3.1 Добавить `pkg/draftrag/pgvector_dimension_guard_test.go`: unit-тесты, которые создают store через `NewPGVectorStore` с `EmbeddingDimension=N` и проверяют. Touches: pkg/draftrag/pgvector_dimension_guard_test.go
  - `Upsert` с embedding длины `!= N` возвращает `errors.Is(err, ErrEmbeddingDimensionMismatch) == true`;
  - `Search` с embedding длины `!= N` возвращает `errors.Is(err, ErrEmbeddingDimensionMismatch) == true`;
  - happy-path не ломается: при корректной длине ошибка dimension mismatch не возникает (не проверяя доступ к БД).
- [x] T3.2 Прогнать `go test ./...`. Touches: (go test ./...)

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T2.1, T3.1
- AC-002 -> T2.1, T2.2, T3.1, T3.2

## Заметки

- Тесты должны быть без сети/БД: для happy-path достаточно проверить, что `errors.Is(err, ErrEmbeddingDimensionMismatch) == false` (а не что SQL реально выполняется).
