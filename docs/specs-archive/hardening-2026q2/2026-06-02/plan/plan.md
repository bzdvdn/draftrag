# Харденинг библиотеки — План

## Phase Contract

Inputs: spec (`docs/specs/hardening-2026q2/spec.md`), inspect (`pass`), constitution summary, репозиторий.
Outputs: `plan.md`, `data-model.md` (no-change).
Stop if: нет — spec конкретна, inspect pass.

## Цель

4 ортогональных workstream-а в одном плане. Каждый даёт независимую ценность и может быть выполнен параллельно, кроме связки AC-003 с AC-001 (prompt выносится при рефакторинге) и AC-010 с чтением surfaces pipeline.go/draftrag.go.

## MVP Slice

**Рефакторинг pipeline.go** (AC-001–004) — наименьший независимый срез. После него:
- 1915 строк → ≤400 в pipeline.go, модули в internal/application/
- Все тесты зелёные, ни один не изменён
- defaultSystemPromptV1 в prompts.go

Остальные 3 workstream-а добавляются итеративно и независимо.

## First Validation Path

```bash
git checkout feature/hardening-2026q2
go build ./...
go test ./internal/application/... -count=1
wc -l internal/application/pipeline.go                    # ≤400
grep 'defaultSystemPromptV1' internal/application/pipeline.go  # пусто (разрешено)
grep 'defaultSystemPromptV1' internal/application/prompts.go   # найдено
```

## Scope

1. **Рефакторинг pipeline.go** — только `internal/application/`. Без изменения поведения.
2. **Redis cache public** — только `pkg/draftrag/` (новый файл) + тесты.
3. **Покрытие тестами** — только `pkg/draftrag/*_test.go` (новые тесты).
4. **Унификация ошибок** — `internal/domain/errors.go` + `pkg/draftrag/errors.go` + `pkg/draftrag/draftrag.go` (mapValidationErr).

## Implementation Surfaces

| Surface | Тип | Для AC | Почему |
|---|---|---|---|
| `internal/application/pipeline.go` | существующий | AC-001–004 | god-object, рефакторинг |
| `internal/application/{query,answer,stream,batch,prompt,hooks,retrieval,rrf}.go` | новые | AC-001–004 | целевые модули после разбиения |
| `pkg/draftrag/cached_embedder_redis.go` | новый | AC-005–006 | публичный конструктор RedisCache |
| `pkg/draftrag/{search,errors,draftrag,resilience,pgvector_migrate}_test.go` | новые/существующие | AC-006–008 | тесты на непокрытые методы |
| `pkg/draftrag/errors.go` | существующий | AC-009 | переэкспорт sentinel-ошибок |
| `pkg/draftrag/draftrag.go` | существующий | AC-010 | упрощение mapValidationErr |

## Bootstrapping Surfaces

- `none` — все нужные пакеты и директории уже существуют.

## Влияние на архитектуру

- Domain-слой не затрагивается (только переэкспорт ошибок — type-aliases).
- Application-слой перестраивается внутренне: разбиение одного файла на несколько. SRP улучшается.
- `pkg/draftrag` расширяется без breaking changes: новый файл + тесты.
- CI/CD: те же команды `go test ./...`, `go vet`, `golangci-lint`.

## Acceptance Approach

- **AC-001**: `ls internal/application/*.go` ≥ 5 файлов; `wc -l internal/application/pipeline.go` ≤ 400.
- **AC-002**: `go test ./internal/application/... -count=1` success; `git diff --stat -- '*_test.go'` пуст.
- **AC-003**: `grep 'defaultSystemPromptV1' internal/application/prompts.go` найден; в pipeline.go — не найден.
- **AC-004**: `go vet ./internal/application/... && golangci-lint run ./internal/application/...` exit 0.
- **AC-005**: `go build ./pkg/draftrag/...` успешен; пример кода компилируется.
- **AC-006**: `go test -coverprofile ./pkg/draftrag/...` — покрытие новой функции > 0%.
- **AC-007**: `go test -covermode=atomic ./pkg/draftrag/...` → `go tool cover -func=coverage.out` ≥ 65%.
- **AC-008**: `go tool cover -func=coverage.out | grep 'pkg/draftrag/search.go'` — ни одна функция не 0.0%.
- **AC-009**: новый тест проверяет `errors.Is(err, draftrag.ErrXXX)` через цепочку `domain.ErrXXX`.
- **AC-010**: `grep -c 'errors.Is(err, domain' pkg/draftrag/draftrag.go` ≤ 1 (только для non-sentinel).

## Данные и контракты

- Ни одна domain-модель не меняется. Ни один публичный интерфейс не меняется. Все контракты сохраняются.
- API-контракты: `pkg/draftrag` расширяется новым экспортом `RedisCache` — это additive change, не breaking.
- Data model не меняется — см. `data-model.md`.

## Стратегия реализации

### DEC-001 Разбиение pipeline.go по доменам, а не по слоям

- Why: каждый файл соответствует use-case (query, answer, stream), а не техническому слою (валидация, hooks). Это ближе к архитектурно значимой структуре: разработчик ищет код по фиче.
- Tradeoff: некоторые функции (hooks, dedup, rrf) — утилиты, а не фичи, но их естественно выделить в hooks.go/retrieval.go/rrf.go.
- Affects: `internal/application/`
- Validation: `go test ./internal/application/...` зелёный, ни один тест не изменён.

### DEC-002 Redis cache — type-alias, а не обёртка

- Why: все остальные реализации в `pkg/draftrag/` используют type-aliases на internal-типы. Консистентность API.
- Tradeoff: пользователь видит internal-интерфейс `RedisClient` через alias. Если internal меняется — публичное API тоже меняется.
- Affects: `pkg/draftrag/cached_embedder_redis.go`, `internal/infrastructure/embedder/cache/redis.go`
- Validation: пример компилируется.

### DEC-003 mapValidationErr — урезать, а не удалять

- Why: ошибки вида `fmt.Errorf("reranker: %w", err)` не имеют sentinel и не могут быть переэкспортированы. mapValidationErr остаётся для таких случаев, но удаляются дублирующие `errors.Is(err, domain.ErrXXX)`.
- Tradeoff: частичное сохранение anti-corruption layer.
- Affects: `pkg/draftrag/draftrag.go`
- Validation: `grep -c 'errors.Is(err, domain' pkg/draftrag/draftrag.go` ≤ 1.

### DEC-004 defaultSystemPromptV1 — отдельный .go-файл (const), без //go:embed

- Why: одна константа не оправдывает embed-ресурс. const-файл проще тестировать и читать.
- Tradeoff: интернационализация (i18n) потребует рефакторинга, но это не в scope.
- Affects: `internal/application/prompts.go`
- Validation: grep подтверждает перенос.

## Incremental Delivery

### MVP (Первая ценность)

Рефакторинг pipeline.go (AC-001–004). После него `internal/application/` clean.

Проверка:
```bash
go test ./internal/application/... -count=1 && \
  wc -l internal/application/pipeline.go | grep -E '^\s*[0-9]+' | awk '$1 <= 400'
```

### Итеративное расширение

| Шаг | Что | AC | Независимость |
|---|---|---|---|
| 2 | Errors unification | AC-009–010 | Можно параллелить с шагом 1, но затрагивает draftrag.go |
| 3 | Redis cache public | AC-005–006 | Полностью независим |
| 4 | Coverage boost | AC-007–008 | Зависит от шага 2 (errors_test.go), остальное — независимо |

Шаги 2–4 можно безопасно параллелить (один разработчик — шаг 2, другой — шаг 3). Шаг 4 частично зависит от шага 2 (errors_test.go), может идти последовательно.

## Порядок реализации

1. **pipeline.go рефакторинг** (AC-001–004) — должен быть первым, т.к. меняет структуру основного application-слоя. Все остальные workstream-ы не затрагивают `internal/application/`.
2. **Остальные 3 workstream-а** — параллельно (ограничение: шаг 4 частично зависит от шага 2; см. выше).

## Риски

- **Риск 1**: случайный перенос кода меняет поведение (регрессия).
  Mitigation: `git diff --stat '*_test.go'` — если тесты изменены, рефакторинг НЕ safe. Полный `go test ./...`.
- **Риск 2**: `RedisClient` интерфейс в internal меняется после публикации.
  Mitigation: type-alias — если internal меняется, компилятор ловит несоответствие.
- **Риск 3**: покрытие 65% не достигается, если streaming-тесты сложны в мокинге.
  Mitigation: mocking через `MockStreamingLLMProvider` (уже есть в `internal/infrastructure/llm/mock_streaming.go`). Если не хватает — расширить мок.
- **Риск 4**: `golangci-lint` на подпакете `./internal/application/...` может не найти файлы, если конфигурация linter не поддерживает подпакеты.
  Mitigation: запускать `golangci-lint run` без пути (весь проект) как fallback.

## Rollout и compatibility

- Все изменения additive или refactoring-only. Никакого rollout, флагов, миграций не требуется.
- Redis cache — новый экспорт. Пользователи обновляют go.mod, получают новую функцию. Обратная совместимость полная.
- Coverage — только новые тесты. Никакого влияния на production-бинарник.

## Проверка

| Этап | Проверка |
|---|---|
| После рефакторинга | `go test ./...`, `go vet ./...`, `golangci-lint run`, `wc -l pipeline.go` |
| После Redis cache | пример компилируется, `go test ./pkg/draftrag/...` |
| После errors | `go test ./...`, новая errors_test.go проходит |
| После coverage | `go test -covermode=atomic ./pkg/draftrag/...` ≥ 65% |
| Финальная | `go test ./... && go vet ./... && golangci-lint run && go build ./...` |

## Соответствие конституции

- нет конфликтов. Clean Architecture сохраняется (domain не импортирует infrastructure). Интерфейсная абстракция сохраняется (Redis cache — type-alias на существующий internal-интерфейс). Покрытие повышается в сторону конституционной нормы.
