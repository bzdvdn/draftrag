# OpenTelemetry observability (Hooks) — Задачи

## Phase Contract

Inputs: `.speckeep/specs/otel-observability/plan/plan.md`, текущие `draftrag.Hooks` и README секция `Observability hooks`.
Outputs: публичный OTel hooks (`pkg/draftrag/otel`) + README пример + тесты.
Stop if: требуется менять интерфейс `Hooks` или core pipeline (это вне scope).

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/otel/ | T1.1, T2.1, T2.2, T3.1 |
| go.mod | T1.2 |
| go.sum | T1.2 |
| README.md | T2.3 |

## Фаза 1: Основа

Цель: создать публичную поверхность и минимальные зависимости для OTel.

- [x] T1.1 Создать подпакет `pkg/draftrag/otel` с каркасом hooks. Touches: pkg/draftrag/otel/
  - Outcome: компилируемый пакет с публичным типом/конструктором, реализующим `draftrag.Hooks` (AC-001).
  - Links: AC-001, RQ-001, DEC-002

- [x] T1.2 Подключить минимальные зависимости OpenTelemetry (trace + metric). Touches: go.mod, go.sum
  - Outcome: `go test ./...` проходит с OTel зависимостями.
  - Links: AC-001, DEC-002

## Фаза 2: Основная реализация

Цель: реализовать spans+metrics контракт и документацию.

- [x] T2.1 Реализовать stage spans на `StageEnd` с атрибутами и ошибками. Touches: pkg/draftrag/otel/
  - Outcome: span создаётся с `draftrag.operation` и `draftrag.stage`, ошибки записываются при `Err != nil` (AC-002).
  - Links: AC-002, RQ-002, DEC-001, DEC-003

- [x] T2.2 Реализовать stage metrics (duration + errors) с labels. Touches: pkg/draftrag/otel/
  - Outcome: `draftrag.pipeline.stage.duration_ms` (histogram) и `draftrag.pipeline.stage.errors` (counter) пишутся с labels `operation`/`stage` (AC-003).
  - Links: AC-003, RQ-003, DEC-003

- [x] T2.3 Обновить README секцию `Observability hooks` примером OTel. Touches: README.md
  - Outcome: README содержит пример подключения OTel hooks и пояснение “минимальный код/без форка”, плюс контракт атрибутов/метрик (AC-004).
  - Links: AC-004, RQ-005

## Фаза 3: Проверка

Цель: доказать корректность и стабильность контракта.

- [x] T3.1 Добавить unit-тесты для tracing и metrics. Touches: pkg/draftrag/otel/
  - Outcome: тесты подтверждают атрибуты/error для spans и наличие метрик/лейблов (AC-002, AC-003).
  - Links: AC-002, AC-003

- [x] T3.2 Прогнать `go test ./...` и выровнять пакет. Touches: pkg/draftrag/otel/
  - Outcome: `go test ./...` зелёный; публичные имена/контракт фиксированы.
  - Links: AC-001

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T3.2
- AC-002 -> T2.1, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T2.3

## Заметки

- “Быстро” в README трактовать как “минимальный код/без форка”, без обещаний latency/SLO.
- Контракт имён: `draftrag.operation`, `draftrag.stage`, `draftrag.pipeline.stage.duration_ms`, `draftrag.pipeline.stage.errors` — считать стабильным (DEC-003).
