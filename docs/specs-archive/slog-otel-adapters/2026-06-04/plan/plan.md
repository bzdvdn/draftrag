# Slog и OTel Logger адаптеры План

## Phase Contract

Inputs: spec.md, domain/logger.go, pkg/draftrag/draftrag.go (PipelineOptions)
Outputs: plan.md, data-model.md
Stop if: нет — spec детальна.

## MVP Slice

- `pkg/draftrag/slogadapter/slog.go` — `New(*slog.Logger) domain.Logger`
- `pkg/draftrag/slogadapter/slog_test.go` — тесты на уровни, fields, trace correlation
- AC-001, AC-002, AC-003, AC-004, AC-005

## First Validation Path

`go test -v ./pkg/draftrag/slogadapter/` — 3 теста PASS. Ручная проверка: `NewPipelineWithOptions(store, llm, emb, PipelineOptions{Logger: slogadapter.New(slog.DefaultLogger())})` компилируется.

## Scope

- Новый пакет `pkg/draftrag/slogadapter/` с одним файлом адаптера + тесты
- OTel log bridge (`otel/logger.go`) — P2, только если MVP сделан быстро
- Trace correlation через `trace.SpanFromContext` — опционально, зависит от OTel

## Implementation Surfaces

| Surface | Почему участвует | Новая/сущ. |
|---------|-----------------|------------|
| `pkg/draftrag/slogadapter/slog.go` | `New()` — адаптер slog → domain.Logger | Новая |
| `pkg/draftrag/slogadapter/slog_test.go` | Тесты маппинга уровней, fields, trace | Новая |
| `internal/domain/logger.go` | Logger interface — целевой контракт | Сущ., без изменений |
| `pkg/draftrag/draftrag.go` | PipelineOptions.Logger (тип domain.Logger) — потребитель | Сущ., без изменений |

## Bootstrapping Surfaces

`none` — все нужные типы в репозитории уже есть.

## Влияние на архитектуру

- Локальное: только новый пакет slogadapter
- На интеграции: не влияет
- Migration/compatibility: не требуется

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|----|--------|----------|------------|
| AC-001 | `go build ./pkg/draftrag/slogadapter/` | `slog.go` | exit 0 |
| AC-002 | JSON handler buffer → проверить уровень и msg | `slog_test.go` | `go test -run TestSlogAdapter_LevelMapping` PASS |
| AC-003 | JSON handler buffer → проверить поля | `slog_test.go` | `go test -run TestSlogAdapter_Fields` PASS |
| AC-004 | Context с mock span → проверить trace_id | `slog_test.go` | `go test -run TestSlogAdapter_TraceContext` PASS |
| AC-005 | `go vet ./pkg/draftrag/...` | — | exit 0 |

## Данные и контракты

- Data model не меняется — см. `data-model.md`
- API/event контракты не меняются
- Единственный новый публичный символ: `slogadapter.New`

## Стратегия реализации

### DEC-001 Отдельный пакет `slogadapter`

Why: чёткое разделение — пользователь импортирует только если нужен slog. Не добавляет зависимость `log/slog` в основной пакет.
Tradeoff: дополнительный import для пользователя, но это стандартная Go-практика (см. `otel/`).
Affects: `pkg/draftrag/slogadapter/`
Validation: `go build ./pkg/draftrag/slogadapter/` — OK.

### DEC-002 Конвертация LogField → slog.Attr через `slog.Any`

Why: `LogField.Value any` → `slog.Any(key, value)` покрывает все типы (string, int, error, struct).
Tradeoff: для строк можно было бы использовать `slog.String`, но `slog.Any` универсальнее.
Affects: `slog.go`
Validation: `go test -run TestSlogAdapter_Fields` — поля всех типов корректны.

### DEC-003 Trace correlation через `trace.SpanFromContext`

Why: если context содержит span, добавляем trace_id и span_id как Attr. Используем `go.opentelemetry.io/otel/trace` — уже есть в зависимостях проекта (otel пакет).
Tradeoff: дополнительная зависимость от OTel в slogadapter. Принято: OTel уже в `go.mod`.
Affects: `slog.go`
Validation: `go test -run TestSlogAdapter_TraceContext` — trace_id присутствует.

### DEC-004 OTel log bridge — deferred (P2)

Why: MVP покрывает 80% use case. OTel log bridge требует `log/slog` + OTel SDK интеграцию, что сложнее и нужнее меньшинству.
Tradeoff: не все OTel-пользователи получат unified logging. Компенсация: OTel hooks уже пишут tracing/metrics.
Affects: `pkg/draftrag/otel/logger.go` — не создаётся в MVP.
Validation: не требуется в MVP.

## Incremental Delivery

### MVP (Первая ценность)

1. `slogadapter/slog.go` — `New`, convertLevel, convertFields, traceAttrs
2. `slogadapter/slog_test.go` — LevelMapping, Fields, TraceContext
3. `go build + go vet`

### Итеративное расширение

4. OTel log bridge (`otel/logger.go`) — если остаётся время

## Порядок реализации

1. `slog.go` — структура адаптера, New, Log метод
2. `slog_test.go` — LevelMapping
3. `slog_test.go` — Fields
4. `slog_test.go` — TraceContext
5. `go build + go vet`

Всё последовательно (зависит от файла slog.go).

## Риски

- **Риск 1**: `trace.SpanFromContext(ctx)` — если ctx без span, вернёт noop span с пустым SpanID. TraceAttrs проверяет `span.SpanContext().HasTraceID()`.
  Mitigation: тест с пустым context и с mock span.
- **Риск 2**: `slog.Log` паникует при nil ctx. Адаптер вызывает `SafeLog` → recover. Но сам Log должен проверять ctx.
  Mitigation: добавить защиту `if ctx == nil { ctx = context.Background() }`.

## Rollout и compatibility

Специальных rollout-действий не требуется. Новый пакет не меняет существующий API.

## Проверка

| Шаг | Check | AC |
|-----|-------|----|
| Build | `go build ./pkg/draftrag/slogadapter/` | AC-001 |
| Level test | `go test -run TestSlogAdapter_LevelMapping ./pkg/draftrag/slogadapter/` | AC-002 |
| Fields test | `go test -run TestSlogAdapter_Fields ./pkg/draftrag/slogadapter/` | AC-003 |
| Trace test | `go test -run TestSlogAdapter_TraceContext ./pkg/draftrag/slogadapter/` | AC-004 |
| Vet | `go vet ./pkg/draftrag/...` | AC-005 |

## Соответствие конституции

Нет конфликтов.
