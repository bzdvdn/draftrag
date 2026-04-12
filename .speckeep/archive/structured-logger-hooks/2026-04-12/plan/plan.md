# Structured logger hooks (замена log.Printf) План

## Phase Contract

Inputs: `.speckeep/specs/structured-logger-hooks/spec.md`, `.speckeep/specs/structured-logger-hooks/inspect.md` и минимальный контекст кода вокруг кэша/ретраев.
Outputs: `plan.md`, `data-model.md`. Contracts не требуются.
Stop if: решение требует добавления внешних зависимостей на логгер (zap/logrus/slog) или требует breaking changes в существующих публичных конструкторах.

## Цель

Ввести опциональный структурированный логгер (интерфейс без внешних зависимостей) и заменить прямые `log.Printf` в инфраструктурных компонентах на best-effort вызовы этого логгера. По умолчанию (nil) библиотека остаётся “тихой”. Логгер должен быть безопасен: паника внутри логгера не должна ломать основной execution path.

## Scope

- Добавить интерфейс логгера и типы уровней/полей в `internal/domain` и переэкспортировать в `pkg/draftrag`.
- Добавить логгер в конфигурацию `CachedEmbedder` (и внутренние опции `EmbedderCache`) и логировать деградацию Redis L2 без `log.Printf`.
- Добавить логгер в конфигурацию `RetryEmbedder` и `RetryLLMProvider` и логировать retry/CB-события.
- Добавить unit-тесты на вызовы логгера в ключевых ветках (Redis fail/decode; retry attempt; CB rejection; logger panic safety).
- Добавить минимальный usage пример в документацию.

## Implementation Surfaces

- `internal/domain/logger.go` (новый): интерфейс `Logger`, `LogLevel`, `LogField`, no-op и/или helper для безопасного вызова.
- `pkg/draftrag/draftrag.go`: `type Logger = domain.Logger` и связанные типы (переэкспорт).
- `internal/infrastructure/embedder/cache/cache.go`: заменить `log.Printf` на `logger.Log(...)` через safe wrapper.
- `internal/infrastructure/embedder/cache/options.go`: добавить опцию `WithLogger(...)` и хранение логгера в `EmbedderCache`.
- `pkg/draftrag/cached_embedder.go`: расширить `CacheOptions` (или `RedisCacheOptions`) логгером и пробросить в internal.
- `internal/infrastructure/resilience/embedder.go`, `internal/infrastructure/resilience/llm.go`: добавить логгер в `RetryEmbedder`/`RetryLLMProvider`, логировать retry/CB события.
- `pkg/draftrag/resilience.go`: расширить `RetryOptions` логгером и пробросить его в internal resilience.
- Tests:
  - `internal/infrastructure/embedder/cache/redis_test.go`: подтверждение логов на Redis деградации/битых данных
  - `internal/infrastructure/resilience/embedder_test.go`, `internal/infrastructure/resilience/llm_test.go`: подтверждение логов на retry/CB + safety при panic логгера
- Docs:
  - `README.md` и/или `docs/embedders.md` / `docs/*.md`: короткий пример адаптации под `log/slog` или кастомный логгер.

## Влияние на архитектуру

- Добавляется новый “observability” интерфейс в domain-слое (аналогично существующим `domain.Hooks`), но без изменения `domain.Hooks`.
- В инфраструктурных пакетах устраняется прямое использование глобального `log` — повышается управляемость в приложениях и тестируемость.
- Публичный API расширяется опциональными полями/опциями (без breaking changes).

## Acceptance Approach

- AC-001 -> убедиться, что при nil логгере код ведёт себя как раньше; и прямых `log.Printf` в изменённых местах нет. Evidence: unit-тесты + `rg`/CI check по строкам `log.Printf`.
- AC-002 -> кэш: при Redis Get/Set/decode ошибках логгер получает structured event. Evidence: unit-тесты с fake logger, проверка полей `component=embedder_cache`, `operation=redis_get|redis_set|redis_decode`, `err`.
- AC-003 -> retry: при retry attempt и CB rejection логгер получает structured event. Evidence: unit-тесты с контролируемыми ошибками и fake logger; проверки полей `component=resilience_retry`, `operation=embed|generate`, `attempt`, `rejected`.
- AC-004 -> safety: fake logger паникует, но `Embed`/`Generate` не падают из-за логирования. Evidence: unit-тест, где логгер всегда `panic`, а метод возвращает ожидаемую ошибку/результат.
- AC-005 -> docs: пример подключения логгера присутствует и компилируем (как snippet). Evidence: обновлённый раздел в `README.md`/`docs/*`.

## Данные и контракты

- Data model (типы логгера, уровни и поля) фиксируется в `data-model.md`.
- Внешних API/event contracts не добавляется.

## Стратегия реализации

- DEC-001 Logger как domain интерфейс + публичный re-export
  Why: internal пакеты уже зависят от `internal/domain`; единый интерфейс без циклов импорта.
  Tradeoff: публичный API получает больше типов; нужно удержать их минимальными и стабильными.
  Affects: `internal/domain/logger.go`, `pkg/draftrag/draftrag.go`.
  Validation: отсутствие импорт-циклов; `go test ./...` проходит.

- DEC-002 Форма API: `Log(ctx, level, msg, fields...)` + `LogField{Key, Value}`
  Why: один метод проще адаптировать под любые backends и проще мокать в тестах.
  Tradeoff: нет compile-time строгой типизации значений полей (используем `any`).
  Affects: `internal/domain/logger.go`, call sites в cache/resilience.
  Validation: unit-тесты проверяют наличие ключевых полей; docs показывают адаптер.

- DEC-003 Safe logging: единый helper с `recover`
  Why: AC-004 требует, чтобы паника логгера не ломала основной поток.
  Tradeoff: скрывает панику (только best-effort); но это ожидаемо для observability.
  Affects: `internal/domain/logger.go` (или internal helper), все call sites.
  Validation: unit-тесты с panic logger.

- DEC-004 Поля событий и naming scheme
  Why: без базовой схемы поля будут несогласованны и трудно фильтруемы.
  Tradeoff: минимальная схема полей ограничивает свободу implement-части.
  Affects: cache/resilience call sites, tests, docs.
  Validation: unit-тесты проверяют `component`, `operation` и ключевые поля.

## Incremental Delivery

### MVP (Первая ценность)

- Добавить интерфейс `Logger` и safe wrapper.
- Прокинуть логгер в `CachedEmbedder`/`EmbedderCache` и заменить `log.Printf`.
- Прокинуть логгер в `RetryEmbedder`/`RetryLLMProvider` и залогировать CB rejection + retry attempts.
- Покрыть unit-тестами AC-001..AC-004 и добавить минимальную доку (AC-005).

Критерий готовности MVP: `go test ./...` проходит; прямых `log.Printf` в изменённых местах нет.

### Итеративное расширение

- При необходимости — расширить набор событий (например, отдельный event для “non-retryable error”), не меняя интерфейс.

## Порядок реализации

- Сначала: определить `Logger` интерфейс/типы и safe helper.
- Затем: интегрировать в кэш (самый простой источник событий) и добавить тесты.
- Затем: интегрировать в resilience (retry/CB) и добавить тесты.
- В конце: обновить документацию и провести финальный `go test ./...`.

## Риски

- Риск: слишком “богатый” интерфейс логгера усложнит поддержку.
  Mitigation: один метод `Log` + минимальный набор типов, без внешних зависимостей.
- Риск: добавление логирования меняет performance при высокой частоте событий.
  Mitigation: логгер опционален; call sites должны быть cheap и best-effort.

## Rollout и compatibility

- Backward compatible: новые поля/опции опциональны; nil = no-op.
- Не требуется миграций: поведение по умолчанию не меняется.

## Проверка

- Unit-тесты на cache/resilience call sites и panic safety.
- `go test ./...`
- Точечный grep/rg check: отсутствие `log.Printf` в затронутых местах.

## Соответствие конституции

- Интерфейсная абстракция: логгер подключается через Go-интерфейс, без привязки к конкретной библиотеке.
- Минимальная конфигурация: по умолчанию no-op, явная передача опций включает логирование.
- Контекстная безопасность: логгер принимает `context.Context`.
- Тестируемость: логгер мокается/fake’ится в unit-тестах без внешних сервисов.

