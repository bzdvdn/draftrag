# Eval harness: только retrieval метрики (без качества генерации) Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для этой фичи.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/eval/models.go | T1.1, T1.2, T2.4 |
| pkg/draftrag/eval/metrics.go | T2.1, T2.2 |
| pkg/draftrag/eval/harness.go | T2.3, T2.5 |
| pkg/draftrag/eval/harness_test.go | T3.1, T3.2 |

## Фаза 1: Основа

Цель: расширить data model для поддержки новых retrieval-метрик.

- [x] T1.1 Расширить Metrics struct полями NDCG, Precision, Recall — struct содержит новые поля для метрик с дефолтными значениями 0. Touches: pkg/draftrag/eval/models.go — DEC-001, AC-001, AC-002
- [x] T1.2 Расширить CaseResult полями для per-case метрик — struct содержит NDCG, Precision, Recall для каждого кейса. Touches: pkg/draftrag/eval/models.go — DEC-001, AC-004

## Фаза 2: Основная реализация

Цель: реализовать вычисление новых метрик, конфигурацию и интеграцию в harness.

- [x] T2.1 Реализовать computeNDCG функцию — функция вычисляет NDCG@K с опциональными весами релевантности по стандартной формуле. Touches: pkg/draftrag/eval/metrics.go — DEC-003, AC-001
- [x] T2.2 Реализовать computePrecision и computeRecall функции — функции вычисляют Precision@K и Recall@K для множества релевантных документов. Touches: pkg/draftrag/eval/metrics.go — AC-002
- [x] T2.3 Расширить Options флагами EnableNDCG, EnablePrecision, EnableRecall — struct содержит булевы флаги для включения вычисления метрик с дефолтом false для backward compatibility. Touches: pkg/draftrag/eval/harness.go — DEC-002, AC-003
- [x] T2.4 Реализовать MarshalJSON для Report struct — Report сериализуется в валидный JSON со всеми метриками и деталями кейсов. Touches: pkg/draftrag/eval/models.go — DEC-004, AC-006
- [x] T2.5 Обновить computeMetrics и Run для условного вычисления метрик — computeMetrics вычисляет новые метрики при включённых флагах, Run заполняет per-case метрики и улучшает валидацию входных данных. Touches: pkg/draftrag/eval/metrics.go, pkg/draftrag/eval/harness.go — AC-003, AC-004, AC-005

## Фаза 3: Проверка

Цель: доказать корректность реализации через unit-тесты и performance-тесты.

- [x] T3.1 Добавить unit-тесты для новых метрик и валидации — тесты покрывают computeNDCG, computePrecision, computeRecall, условное вычисление через Options, валидацию входных данных и MarshalJSON round-trip. Touches: pkg/draftrag/eval/harness_test.go — SC-003
- [x] T3.2 Добавить performance-тесты для SC-001 и SC-002 — тесты подтверждают что 1000 кейсов обрабатываются <10с и 10000 кейсов потребляют <500MB памяти. Touches: pkg/draftrag/eval/harness_test.go — SC-001, SC-002

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T2.5
- AC-002 -> T1.1, T2.2, T2.5
- AC-003 -> T2.3, T2.5
- AC-004 -> T1.2, T2.5
- AC-005 -> T2.5
- AC-006 -> T2.4, T3.1

## Заметки

- Задачи следуют порядку реализации из plan.md
- Все задачи имеют конкретные Touches: для batch-чтения на фазе implement
- Performance-тесты вынесены в отдельную задачу T3.2 для ясности scope
