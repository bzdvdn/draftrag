# Eval harness: базовая оценка качества RAG (v1) — Задачи

## Phase Contract

Inputs: plan.  
Outputs: реализованный публичный eval harness + тесты.  
Stop if: невозможно написать детерминированные тесты без сети.

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/eval/models.go | T1.1 |
| pkg/draftrag/eval/metrics.go | T2.1 |
| pkg/draftrag/eval/harness.go | T2.2 |
| pkg/draftrag/eval/harness_test.go | T3.1 |

## Фаза 1: Основа

- [x] T1.1 Добавить модели данных eval-case и отчёта (`Case`, `CaseResult`, `Report`, `Metrics`). Touches: `pkg/draftrag/eval/models.go`. (RQ-001)

## Фаза 2: Основная реализация

- [x] T2.1 Реализовать вычисление retrieval-метрик hit@k и MRR по `ParentID`. Touches: `pkg/draftrag/eval/metrics.go`. (RQ-002, AC-001)
- [x] T2.2 Реализовать harness `Run` с интерфейсом retrieval runner и опциями (например `K`, `TopK`). Touches: `pkg/draftrag/eval/harness.go`. (RQ-003)

## Фаза 3: Проверка

- [x] T3.1 Добавить unit-тесты на синтетическом датасете: ожидаемые значения hit@k и MRR. Touches: `pkg/draftrag/eval/harness_test.go`. (AC-001, RQ-004)
- [x] T3.2 Прогнать `go test ./...` и убедиться, что всё аддитивно. Touches: repo. (AC-002)

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T2.2, T3.1
- AC-002 -> T3.2
