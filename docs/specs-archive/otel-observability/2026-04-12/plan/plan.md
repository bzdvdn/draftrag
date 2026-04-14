# OpenTelemetry observability (Hooks) — План

## Phase Contract

Inputs: `.speckeep/specs/otel-observability/spec.md`, `.speckeep/specs/otel-observability/inspect.md`, текущий публичный интерфейс `draftrag.Hooks` и README секция `Observability hooks`.
Outputs: `.speckeep/specs/otel-observability/plan/plan.md`, `.speckeep/specs/otel-observability/plan/data-model.md`.
Stop if: реализация OTel spans/metrics требует изменения `Hooks` интерфейса (это вне scope).

## Цель

Добавить опциональный OTel-hooks адаптер (реализация `draftrag.Hooks`) на публичной поверхности, который создаёт spans и метрики по стадиям pipeline (chunking/embed/search/generate) и документировать подключение в README как “минимальный код, без форка”, без обещаний SLO/производительности.

## Scope

- Новый публичный подпакет для OTel hooks (внутри `pkg/draftrag`).
- Обновление README: добавить/расширить секцию `Observability hooks` примером для OpenTelemetry и описанием stable атрибутов/метрик и ограничений синхронных hooks.
- Добавить минимальные unit-тесты, подтверждающие: корректные span attributes/error и запись метрик (без требований к конкретному exporter/collector).

## Implementation Surfaces

- `pkg/draftrag/otel/` (новая поверхность): OTel hooks, опции, стабильные имена/атрибуты/метрики.
- `README.md` (существующая поверхность): секция `Observability hooks` — добавить OTel пример и пояснения.
- `go.mod`/`go.sum` (существующая поверхность): добавить зависимости `go.opentelemetry.io/otel/...` (trace + metric) как compile-time dependency, при этом использование остаётся opt-in через импорт подпакета.

## Влияние на архитектуру

- Core pipeline и доменные интерфейсы не меняются: интеграция реализуется как внешний адаптер на публичной поверхности, подключаемый через `PipelineOptions.Hooks`.
- Clean Architecture сохраняется: observability — внешний слой, не протекает в `internal/domain` и не требует changes там.

## Acceptance Approach

- AC-001 -> создать публичный тип/конструктор в `pkg/draftrag/otel`, который реализует `draftrag.Hooks` и подключается в `PipelineOptions.Hooks`; evidence: godoc + компиляция примера.
- AC-002 -> на `StageEnd` создавать span как child текущего контекстного span (если он есть), выставлять атрибуты `operation`/`stage`, и отмечать ошибку; evidence: unit-тест с in-memory span recorder.
- AC-003 -> на `StageEnd` писать duration и error metrics с labels `operation`/`stage`; evidence: unit-тест с manual meter reader.
- AC-004 -> расширить README секцию: пример подключения hooks + список атрибутов/метрик + disclaimer про “быстро” (минимальный код/без форка) и синхронность hooks; evidence: README содержит новую подсекцию/код-блок.

## Данные и контракты

- Data model: не требуется.
- API contracts: добавляется новый публичный подпакет `pkg/draftrag/otel` (новая API-поверхность), но существующий `draftrag` API не меняется.
- Стабильные контракты observability:
  - Атрибуты span: `draftrag.operation` (string), `draftrag.stage` (string).
  - Метрики:
    - `draftrag.pipeline.stage.duration_ms` — histogram (ms), labels: `operation`, `stage`.
    - `draftrag.pipeline.stage.errors` — counter, labels: `operation`, `stage`.

## Стратегия реализации

- DEC-001 Реализовывать spans/metrics на `StageEnd`, не полагаясь на хранение state между start/end
  Why: `Hooks` интерфейс не возвращает обновлённый `context`, поэтому безопаснее и проще создавать stage span по `StageEndEvent.Duration` (с `WithTimestamp`) без глобальных мап/стеков.
  Tradeoff: span не “оборачивает” реальные подпроцессы внутри стадии (он ретроспективный), но даёт точную длительность и привязку к `operation`/`stage`.
  Affects: `pkg/draftrag/otel/hooks.go`.
  Validation: unit-тест проверяет, что span start/end timestamps соответствуют `Duration` и что error записан.

- DEC-002 Подпакет `pkg/draftrag/otel` как opt-in интеграция
  Why: не нагружает базовый импорт `pkg/draftrag` OTel-символами и сохраняет явное подключение через импорт подпакета.
  Tradeoff: в `go.mod` появится OTel dependency в целом модуле; это приемлемо для библиотечного проекта, но нужно держать минимальный набор пакетов.
  Affects: `pkg/draftrag/otel/*`, `go.mod`, `README.md`.
  Validation: `go test ./...` проходит; пример в README использует только публичные символы.

- DEC-003 Стабильные имена метрик/атрибутов фиксируем как контракт v1
  Why: enterprise команды строят дашборды/алерты; переименование ломает наблюдаемость сильнее, чем смена реализации.
  Tradeoff: сложнее менять формат в будущем; при изменениях потребуется migration note.
  Affects: `pkg/draftrag/otel/constants.go` (или эквивалент), `README.md`.
  Validation: unit-тесты используют эти имена; README перечисляет их явно.

## Incremental Delivery

### MVP (Первая ценность)

- Реализовать OTel hooks (tracing + 2 метрики) и минимальные тесты (AC-001..AC-003).
- Добавить README пример подключения и описание контракта (AC-004).

### Итеративное расширение

- Добавить опции: кастомные tracer/meter, префикс имён метрик, включение/выключение отдельных сигналов (trace/metrics).
- Добавить защиту от лишней аллокации (переиспользование attribute sets), если будет доказанная нагрузка.

## Порядок реализации

- Сначала: реализовать `pkg/draftrag/otel` с минимальными опциями и тестами (фиксирует контракт).
- Затем: обновить README секцию `Observability hooks` (минимальный код, без обещаний SLO).
- В конце: прогнать `go test ./...` и выровнять линтер/форматирование (если включено в CI).

## Риски

- Риск: кардинальность labels по `operation` может вырасти при добавлении новых операций.
  Mitigation: `operation` сейчас контролируется библиотекой; задокументировать это и держать список операций стабильным/ограниченным.
- Риск: hooks синхронные → возможный overhead OTel SDK/exporter.
  Mitigation: в README явно указать, что exporters должны быть неблокирующими/батчевыми; по умолчанию OTel no-op минимизирует overhead.

## Rollout и compatibility

- Breaking changes не требуется; новый подпакет добавляется как opt-in.
- При изменении имён метрик/атрибутов в будущем — документировать в changelog и сохранять backward compatibility по возможности.

## Проверка

- Unit tests:
  - spans: `operation`/`stage` атрибуты + error-status/recorded error (AC-002).
  - metrics: наличие `duration_ms` histogram и `errors` counter с labels (AC-003).
- Документация:
  - README содержит пример подключения `otel` hooks + список контрактных имён/лейблов + пояснение “быстро = минимальный код/без форка” (AC-004).
- Репозиторий:
  - `go test ./...` проходит (AC-001).

## Соответствие конституции

- Контекстная безопасность: hooks используют `context.Context` как parent для span (propagation).
- Минимальная конфигурация: OTel hooks подключаются опционально; при отсутствии hooks поведение pipeline не меняется.
- Интерфейсная абстракция: интеграция реализует существующий интерфейс `Hooks`, без изменений core API.
- Тестируемость: OTel hooks покрывается unit-тестами с in-memory OTel test providers/recorders.

