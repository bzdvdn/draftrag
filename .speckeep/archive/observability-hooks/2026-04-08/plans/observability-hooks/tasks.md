# Observability: хуки/метрики для pipeline стадий (v1) — Задачи

## Phase Contract

Inputs: plan.  
Outputs: hooks + тесты + verify.  
Stop if: hooks не удаётся внедрить без изменения поведения по умолчанию.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/hooks.go | T1.1 |
| internal/application/pipeline.go | T2.1 |
| pkg/draftrag/draftrag.go | T2.2 |
| internal/application/observability_hooks_test.go | T3.1 |

## Фаза 1: Основа

- [x] T1.1 Добавить domain-интерфейс hooks и типы стадий/событий (stdlib only). Touches: `internal/domain/hooks.go`. (RQ-001, RQ-002, RQ-003)

## Фаза 2: Основная реализация

- [x] T2.1 Инструментировать application pipeline: embed/search/generate/chunking с duration и error. Touches: `internal/application/pipeline.go`. (RQ-002, RQ-003, RQ-004, AC-001)
- [x] T2.2 Добавить опцию hooks в публичный API (`PipelineOptions`) и прокинуть в application config. Touches: `pkg/draftrag/draftrag.go`. (RQ-004)

## Фаза 3: Проверка

- [x] T3.1 Добавить unit-тесты: порядок/количество вызовов hooks на Answer (и chunking при наличии chunker). Touches: `internal/application/observability_hooks_test.go`. (RQ-005, AC-001)
- [x] T3.2 Прогнать `go test ./...` и убедиться, что nil hooks не влияет. Touches: repo. (AC-002)

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T3.1
- AC-002 -> T2.2, T3.2
