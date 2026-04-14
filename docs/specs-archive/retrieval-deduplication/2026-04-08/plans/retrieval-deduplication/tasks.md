# Retrieval: дедупликация источников (v1) — Задачи

## Phase Contract

Inputs: `.speckeep/plans/retrieval-deduplication/plan.md`, `.speckeep/plans/retrieval-deduplication/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи становятся расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/draftrag.go | T1.1 |
| internal/application/pipeline.go | T2.1 |
| internal/application/retrieval_deduplication_test.go | T3.1, T3.2 |

## Фаза 1: Конфигурация (opt-in)

Цель: добавить механизм включения дедупликации без изменения поведения по умолчанию.

- [x] T1.1 Добавить опцию в `pkg/draftrag.PipelineOptions` и прокинуть её в `internal/application.PipelineConfig`, сохранив default “выключено”. Touches: pkg/draftrag/draftrag.go, internal/application/pipeline.go

## Фаза 2: Реализация дедупликации (ParentID)

Цель: реализовать дедуп по `ParentID` и применить её в retrieval path перед построением prompt/возвратом evidence.

- [x] T2.1 Реализовать функцию дедупликации (v1: по `ParentID`) и применить её в `internal/application/pipeline.go` после получения `domain.RetrievalResult` из `Search/SearchWithFilter` и до использования результата в prompt/возврате. Алгоритм: выбрать лучший чанк по score на `ParentID`, порядок результата — по релевантности (stable). Touches: internal/application/pipeline.go

## Фаза 3: Тесты (без внешней сети)

Цель: подтвердить AC и backward compatibility.

- [x] T3.1 Добавить unit-тест `internal/application/retrieval_deduplication_test.go`: при включённой дедупликации и наличии дубликатов по `ParentID` в retrieval result, итоговый result содержит не более 1 чанка на `ParentID` и выбран лучший по score. Touches: internal/application/retrieval_deduplication_test.go
- [x] T3.2 Добавить unit-тест `internal/application/retrieval_deduplication_test.go`: при выключенной дедупликации retrieval result не меняется (тот же набор чанков в том же порядке). Touches: internal/application/retrieval_deduplication_test.go
- [x] T3.3 Прогнать `go test ./...`. Touches: (go test ./...)

## Покрытие критериев приемки

- AC-001 -> T2.1, T3.1
- AC-002 -> T1.1, T3.2, T3.3

## Заметки

- В v1 дедупликация должна быть полностью opt-in: без включения опции поведение Answer/Query не меняется.
