# Slog и OTel Logger адаптеры

## Scope Snapshot

- In scope: `log/slog` adapter для `domain.Logger`, OTel log appender (bridge OTel logs → domain.Logger), wire-up в PipelineOptions.
- Out of scope: HTTP `/metrics` endpoint, OTel SDK exporter configuration, distributed context propagation (StageStart → context), переписывание существующих OTel hooks.

## Цель

Пользователь, который использует `log/slog` в своём приложении, сейчас не может передать его в draftRAG pipeline — нужно писать адаптер вручную. После внедрения `pipeline.WithLogger(slog.New(...))` работает из коробки. Успех измеряется: slog adapter покрывает все 4 LogLevel, OTel log bridge пишет structured logs в OTel pipeline.

## Основной сценарий

1. Стартовая точка: пользователь создаёт `slog.NewJSONHandler(os.Stdout, nil)`.
2. Основное действие: передаёт его в `NewPipelineWithOptions(store, llm, embedder, PipelineOptions{...})` через новое поле Logger или через `slogadapter.New()`.
3. Результат: все внутренние логи (retry, cache miss, pipeline stage) пишутся через slog с корректным уровнем и structured fields.
4. Дополнительно: в OTel bridge логи отправляются в OTel log pipeline.

## User Stories

- P1 (MVP): `pkg/draftrag/slogadapter/` — адаптер `slog.Logger` → `domain.Logger` с корректным маппингом LogLevel.
- P2: `pkg/draftrag/otel/logger.go` — bridge OTel log appender → `domain.Logger` (для пользователей, которые экспортируют логи через OTel).

## MVP Slice

- `pkg/draftrag/slogadapter/slog.go` — `New(logger *slog.Logger) domain.Logger`
- Маппинг: LogLevelDebug → slog.LevelDebug, LogLevelInfo → slog.LevelInfo, LogLevelWarn → slog.LevelWarn, LogLevelError → slog.LevelError.
- `context.Context` передаётся через `slog.WithContext` или через `slog.Record.AddAttrs` с trace ID.
- `SafeLog` совместимость: адаптер thread-safe.

## First Deployable Outcome

- `go test -v ./pkg/draftrag/slogadapter/` — тест, что все 4 LogLevel логируются с правильным уровнем и полями.
- Ручная проверка: `myPipeline := NewPipelineWithOptions(store, llm, embedder, PipelineOptions{Logger: slogadapter.New(slog.DefaultLogger())})` компилируется и работает.

## Scope

- `pkg/draftrag/slogadapter/slog.go` — новый пакет с адаптером
- `pkg/draftrag/slogadapter/slog_test.go` — тесты
- `pkg/draftrag/otel/logger.go` — OTel log bridge (P2)
- PipelineOptions.Logger уже существует (тип `domain.Logger`)

## Контекст

- `domain.Logger` интерфейс: `Log(ctx, level LogLevel, msg string, fields ...LogField)`
- `slog.Logger` имеет: `Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)`
- `LogField{Key, Value any}` нужно сконвертировать в `slog.Attr`
- PipelineOptions уже имеет поле `Logger Logger` (тип domain.Logger), но оно нигде не выставляется в публичный API явно
- Существующий `domain.NoopLogger()` — дефолт

## Требования

- RQ-001 Адаптер ДОЛЖЕН мапить LogLevel → slog.Level: debug→Debug, info→Info, warn→Warn, error→Error.
- RQ-002 Адаптер ДОЛЖЕН конвертировать `[]LogField` в `[]slog.Attr` с preserve порядка.
- RQ-003 Адаптер ДОЛЖЕН вызывать `logger.Log(ctx, level, msg, attrs...)` (корректный context).
- RQ-004 Адаптер ДОЛЖЕН быть thread-safe (делегирует slog.Logger, который thread-safe).
- RQ-005 OTel log bridge ДОЛЖЕН использовать `log/appender` или `slog`-совместимый API для OTel.
- RQ-006 Для trace correlation: если в context есть span, адаптер ДОЛЖЕН добавить trace_id и span_id как Attr.

## Вне scope

- Модификация существующего `pkg/draftrag/otel/hooks.go` — только новый файл.
- Distributed context propagation через StageStart — требует изменения domain.Hooks интерфейса (v2).
- OTel SDK конфигурация log exporter — пользователь настраивает сам.

## Критерии приемки

### AC-001 slog adapter экспортируется и компилируется

- **Given** `pkg/draftrag/slogadapter` пакет
- **When** `go build ./pkg/draftrag/slogadapter/`
- **Then** exit code 0
- Evidence: `go build` PASS

### AC-002 slog adapter корректно мапит все 4 LogLevel

- **Given** slog adapter с `slog.NewJSONHandler(buf, nil)`
- **When** вызываются все 4 Log метода (debug, info, warn, error)
- **Then** buf содержит записи с корректным уровнем и сообщением
- Evidence: `go test -run TestSlogAdapter_LevelMapping` PASS

### AC-003 slog adapter передаёт fields как Attr

- **Given** slog adapter с `slog.NewJSONHandler(buf, nil)`
- **When** Log вызывается с `[]LogField{{Key: "key1", Value: "val1"}, {Key: "key2", Value: 42}}`
- **Then** buf содержит `"key1":"val1"` и `"key2":42`
- Evidence: `go test -run TestSlogAdapter_Fields` PASS

### AC-004 slog adapter с trace correlation

- **Given** slog adapter
- **When** context содержит span
- **Then** log запись содержит trace_id и span_id атрибуты
- Evidence: `go test -run TestSlogAdapter_TraceContext` PASS

### AC-005 go vet + build без errors

- **Given** все изменения завершены
- **When** `go vet ./pkg/draftrag/...`
- **Then** exit code 0
- Evidence: `go vet` PASS

## Допущения

- Пользователь сам конфигурирует `slog.Handler` (JSON/text, уровень, output).
- OTel log bridge — для пользователей, которые уже используют OTel SDK и хотят логи в том же pipeline.
- Trace correlation требует `trace.SpanFromContext(ctx)` — OTel SDK dependency.

## Критерии успеха

- SC-001 `slogadapter.New(slogger)` реализует `domain.Logger`.
- SC-002 Все 4 LogLevel покрыты тестами.
- SC-003 Trace correlation тестирован с mock span.

## Краевые случаи

- nil *slog.Logger → panic (зеркалит поведение slog).
- ctx == nil → slog.Log паникует. Адаптер НЕ ДОЛЖЕН паниковать — SafeLog защищает.
- nil fields → пустой []slog.Attr.
- Trace correlation без span в context → без trace_id/span_id атрибутов.
- LogField.Value любого типа (string, int, error, struct) → корректно передаётся через slog.Any.

## Открытые вопросы

- Использовать `slog.New(slog.NewJSONHandler(w, nil))` или принимать готовый `*slog.Logger`? Принимаем готовый — пользователь уже настроил handler.
- Нужен ли отдельный `New` или можно встроить в PipelineOptions? Отдельный адаптер — композиция, не усложняет Pipeline.
