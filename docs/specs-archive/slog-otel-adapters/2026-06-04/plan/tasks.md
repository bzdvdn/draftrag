# Slog и OTel Logger адаптеры — Задачи

## Phase Contract

Inputs: plan.md, spec.md, domain/logger.go, pkg/draftrag/draftrag.go
Outputs: tasks.md
Stop if: нет — plan детален, AC привязаны.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/slogadapter/slog.go` | T1.1, T2.1, T2.2 |
| `pkg/draftrag/slogadapter/slog_test.go` | T2.1, T2.2, T2.3 |

## Implementation Context

- **Цель MVP**: пакет `slogadapter` с `New(*slog.Logger) domain.Logger`, конвертацией LogLevel и LogField, trace correlation.
- **Инварианты/семантика**:
  - LogLevel debug/info/warn/error → slog.Level Debug/Info/Warn/Error
  - `[]LogField` → `[]slog.Attr` через `slog.Any`, порядок сохраняется
  - trace_id/span_id извлекаются из context через `trace.SpanFromContext`
  - nil ctx → ctx = context.Background()
  - nil fields → пустой `[]slog.Attr`
- **Ошибки/коды**: slog паникует при nil *slog.Logger — адаптер не защищает (документировано)
- **Контракты/протокол**: `slogadapter.New(slogger)` — единственная публичная функция
- **Границы scope**: OTel log bridge deferred (P2), не меняем PipelineOptions
- **Proof signals**: `go test -v ./pkg/draftrag/slogadapter/` — 3 теста PASS; `go vet ./pkg/draftrag/...` — exit 0
- **References**: DEC-001 (отдельный пакет), DEC-002 (slog.Any), DEC-003 (trace.SpanFromContext), DEC-004 (OTel deferred)

## Фаза 1: Основа

Цель: пакет slogadapter с базовым адаптером.

- [x] T1.1 Создать `pkg/draftrag/slogadapter/slog.go`: функция `New`, структура `adapter`, метод `Log` с конвертацией level, fields, trace attrs. Пакет `slogadapter`, импортирует `log/slog`, `context`, `go.opentelemetry.io/otel/trace`. Touches: `pkg/draftrag/slogadapter/slog.go`

## Фаза 2: Тесты

Цель: 3 теста покрывают маппинг уровней, fields и trace correlation.

- [x] T2.1 Добавить тест `TestSlogAdapter_LevelMapping` — JSON handler buffer, проверить что все 4 уровня логируются с правильным slog.Level и msg. Touches: `pkg/draftrag/slogadapter/slog_test.go`
- [x] T2.2 Добавить тест `TestSlogAdapter_Fields` — LogField.Key/Value → slog.Attr, проверить строки, числа, error, struct. Touches: `pkg/draftrag/slogadapter/slog_test.go`
- [x] T2.3 Добавить тест `TestSlogAdapter_TraceContext` — context с mock span → JSON содержит trace_id и span_id. Touches: `pkg/draftrag/slogadapter/slog_test.go`

## Фаза 3: Проверка

Цель: доказать, что адаптер работает и lint-free.

- [x] T3.1 Финальная проверка: `go build ./pkg/draftrag/slogadapter/`, `go test -v ./pkg/draftrag/slogadapter/`, `go vet ./pkg/draftrag/...` — все PASS. Touches: `pkg/draftrag/slogadapter/`

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1
- AC-002 -> T2.1, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T2.3, T3.1
- AC-005 -> T3.1

## Заметки

- T3.1 — единственная verification задача (build + test + vet).
- OTel log bridge (P2) не входит в задачи — deferred по плану.
