# Eval harness: только retrieval метрики (без качества генерации) План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи.
Outputs: plan, data model и contracts при необходимости.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Расширить существующий пакет `pkg/draftrag/eval/` дополнительными retrieval-метриками (NDCG@K, Precision@K, Recall@K), конфигурацией выбора метрик через Options, улучшенной валидацией входных данных и сериализацией Report в JSON. Реализация сосредоточена в существующем пакете eval без изменения публичного API RetrievalRunner.

## Scope

- Расширение структур `Metrics` и `CaseResult` в `pkg/draftrag/eval/models.go`
- Добавление функций вычисления NDCG, Precision, Recall в `pkg/draftrag/eval/metrics.go`
- Расширение структуры `Options` в `pkg/draftrag/eval/harness.go` для конфигурации метрик
- Обновление функции `Run` для поддержки новых метрик и улучшенной валидации
- Добавление JSON-сериализации для Report в `pkg/draftrag/eval/models.go`
- Обновление существующих unit-тестов и добавление тестов для новых метрик
- RetrievalRunner интерфейс остаётся без изменений
- Pipeline и другие компоненты draftRAG не затрагиваются

## Implementation Surfaces

- `pkg/draftrag/eval/models.go` - расширение структур Metrics и CaseResult, добавление JSON-сериализации (существующая поверхность)
- `pkg/draftrag/eval/metrics.go` - добавление функций computeNDCG, computePrecision, computeRecall (существующая поверхность)
- `pkg/draftrag/eval/harness.go` - расширение Options, обновление Run для условного вычисления метрик (существующая поверхность)
- `pkg/draftrag/eval/harness_test.go` - обновление и расширение тестов (существующая поверхность)

## Влияние на архитектуру

- Локальное влияние только на пакет eval, изменения не затрагивают domain/application слои
- RetrievalRunner интерфейс остаётся стабильным, совместимость с существующим кодом сохраняется
- Новые поля в Options имеют дефолтные значения для backward compatibility
- Migration не требуется, изменения additive-only

## Acceptance Approach

- AC-001 -> расширить Metrics struct полем NDCG, реализовать computeNDCG в metrics.go, вычислять в computeMetrics при включённом флаге в Options
- AC-002 -> расширить Metrics struct полями Precision и Recall (slice или map для @K), реализовать computePrecision/computeRecall, вычислять при включённых флагах
- AC-003 -> расширить Options флагами EnableNDCG, EnablePrecision, EnableRecall, условно вычислять метрики в computeMetrics
- AC-004 -> расширить CaseResult полями для per-case метрик (NDCG, Precision, Recall), заполнять в Run
- AC-005 -> расширить валидацию в Run: проверять ExpectedParentIDs на пустые строки после нормализации, проверять веса релевантности если используются
- AC-006 -> реализовать json.Marshaler для Report struct или добавить метод MarshalJSON, проверить через unit-тест

## Данные и контракты

- AC-001, AC-002, AC-004 требуют расширения data model: Metrics и CaseResult (подробности в data-model.md)
- AC-006 требует JSON-контракта для Report (подробности в data-model.md)
- API contracts не меняются, RetrievalRunner остаётся прежним
- Event contracts не используются

## Стратегия реализации

### DEC-001 Расширение Metrics и CaseResult в models.go
Why: необходимо хранить новые метрики и per-case результаты для AC-001, AC-002, AC-004
Tradeoff: увеличение размера структур в памяти, но это приемлемо для eval-сценариев
Affects: pkg/draftrag/eval/models.go, pkg/draftrag/eval/harness.go (заполнение полей)
Validation: unit-тесты проверяют заполнение полей корректными значениями

### DEC-002 Конфигурация метрик через Options
Why: AC-003 требует гибкости в выборе вычисляемых метрик для избежания лишних вычислений
Tradeoff: дополнительная сложность в Options, но даёт пользователю контроль над perf
Affects: pkg/draftrag/eval/harness.go (Options struct, Run функция)
Validation: unit-тесты проверяют условное вычисление метрик

### DEC-003 Реализация NDCG с опциональными весами релевантности
Why: AC-001 требует NDCG с учётом градаций релевантности, что важнее бинарного Hit@K
Tradeoff: сложность алгоритма выше, но это стандартная IR-метрика
Affects: pkg/draftrag/eval/metrics.go (новая функция computeNDCG), pkg/draftrag/eval/models.go (расширение Case для весов)
Validation: unit-тесты с известными значениями NDCG

### DEC-004 JSON-сериализация через encoding/json
Why: AC-006 требует сериализации Report для CI/CD интеграции, стандартный пакет достаточен
Tradeoff: зависимость от encoding/json (стандартная библиотека), никаких внешних зависимостей
Affects: pkg/draftrag/eval/models.go (MarshalJSON для Report)
Validation: unit-тест проверяет Marshal/Unmarshal round-trip

## Incremental Delivery

### MVP (Первая ценность)

- Расширение Metrics struct полями NDCG, Precision, Recall
- Реализация computeNDCG, computePrecision, computeRecall
- Расширение Options флагами для включения метрик
- Критерий готовности MVP: AC-001, AC-002, AC-003 покрыты базовыми тестами

### Итеративное расширение

- Расширение CaseResult для per-case метрик (AC-004)
- Улучшенная валидация входных данных (AC-005)
- JSON-сериализация Report (AC-006)
- Критерий готовности: все AC покрыты тестами

## Порядок реализации

1. Расширить models.go: добавить поля в Metrics и CaseResult, добавить веса релевантности в Case
2. Реализовать computeNDCG, computePrecision, computeRecall в metrics.go
3. Расширить Options флагами EnableNDCG, EnablePrecision, EnableRecall
4. Обновить computeMetrics для условного вычисления новых метрик
5. Обновить Run для заполнения per-case метрик и улучшенной валидации
6. Реализовать MarshalJSON для Report
7. Обновить и расширить unit-тесты
8. Проверить performance по SC-001, SC-002

## Риски

- Риск 1: NDCG алгоритм может быть сложнее чем ожидается, особенно с весами релевантности
  Mitigation: использовать стандартную формулу DCG = sum(relevance_i / log2(i+1)), протестировать на известных примерах
- Риск 2: Performance может пострадать при вычислении всех метрик для больших датасетов
  Mitigation: условное вычисление через Options флаги (AC-003), тесты по SC-001, SC-002
- Риск 3: JSON-сериализация больших Report может быть медленной
  Mitigation: использовать стандартный encoding/json, который оптимизирован, тесты по SC-001

## Rollout и compatibility

- Специальных rollout-действий не требуется, изменения additive-only
- Backward compatibility сохраняется: новые поля в Options имеют дефолтные значения, старый код продолжает работать
- Monitoring не требуется, это библиотека без runtime dependencies

## Проверка

- Unit-тесты для computeNDCG, computePrecision, computeRecall с известными значениями (AC-001, AC-002)
- Unit-тесты для условного вычисления метрик через Options (AC-003)
- Unit-тесты для per-case метрик в CaseResult (AC-004)
- Unit-тесты для валидации входных данных (AC-005)
- Unit-тест для MarshalJSON round-trip (AC-006)
- Performance-тесты для 1000 и 10000 кейсов (SC-001, SC-002)
- Unit-тесты покрывают все функции вычисления метрик (SC-003)

## Соответствие конституции

- Интерфейсная абстракция: RetrievalRunner интерфейс остаётся без изменений, соблюдается
- Чистая архитектура: изменения только в infrastructure слое (pkg/draftrag/eval/), domain и application не затрагиваются, соблюдается
- Минимальная конфигурация: Options имеет дефолтные значения, пользователь может переопределить, соблюдается
- Контекстная безопасность: Run уже принимает context.Context, изменения сохраняют это, соблюдается
- Тестируемость: все новые функции имеют unit-тесты, соблюдается
- Язык реализации: Go 1.23+, соблюдается
- Язык документации: русский для комментариев и godoc, соблюдается
