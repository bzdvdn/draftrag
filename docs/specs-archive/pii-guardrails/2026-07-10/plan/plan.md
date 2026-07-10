# PII Guardrails — План

## Phase Contract

Inputs: `docs/specs/pii-guardrails/spec.md`, `.speckeep/constitution.summary.md`.
Outputs: `plan.md`, `data-model.md`.
Stop if: spec расплывчата или требует выдумывать AC — нет.

## Цель

Добавить в Pipeline конфигурируемый PII-детектор (domain-интерфейс + встроенные pattern-детекторы), который цензурирует содержимое на входе Index и на выходе Query/Retrieve/Answer/RewrittenQuery. Feature не меняет существующие контракты и не требует внешних зависимостей.

## MVP Slice

Встроенные детекторы email + телефон + SSN, интеграция с Index и Query (AC-001, AC-002, AC-004) + пример в `examples/`.

## First Validation Path

```go
p, _ := NewPipelineWithOptions(store, llm, embedder, PipelineOptions{
    PIIDetector: NewDefaultPIIDetector(PIICategories{Email: true, Phone: true, SSN: true}),
})
p.Index(ctx, []Document{{ID: "1", Content: "contact: user@example.com, phone: +1-555-123-4567"}})
res, _ := p.Query(ctx, "contact")
// res[0].Content contains "<redacted>" instead of PII
```

## Scope

- Новый domain-интерфейс `PIIDetector` с методом `Detect(text string) string`
- Встроенные реализации через `regexp`: email, телефон (E.164), SSN, номер карты
- Комбинированный `CompositePIIDetector` с независимым включением категорий
- PipelineOption `PIIDetector` в `pkg/draftrag.PipelineOptions` и `application.PipelineOptions`
- PII-redaction в `pkg/draftrag.Index` (перед делегированием в `core.Index`)
- PII-redaction в retrieval-результатах перед возвратом из Query/Retrieve/Answer
- PII-redaction для RewrittenQuery (в SearchBuilder или routeRewriter)
- Re-export `PIIDetector` + конструкторы в `pkg/draftrag/`
- Не меняется: domain-модели (Document, RetrievalResult), интерфейсы VectorStore/LLMProvider/Embedder

## Performance Budget

- `none` — pattern-матчинг через regexp для типичных документов (<10 KB) не создаёт заметной задержки. SC-001 (latency < 5%) проверяется benchmark.

## Implementation Surfaces

1. **`internal/domain/pii.go`** — новый файл: интерфейс `PIIDetector` (new)
2. **`internal/infrastructure/piidetector/`** — пакет встроенных детекторов (new)
3. **`internal/application/pipeline.go`** — `PipelineOptions.PIIDetector`, поле `piidetector` в `Pipeline`, передача в конструктор (existing, modify)
4. **`pkg/draftrag/draftrag.go`** — `PipelineOptions.PIIDetector`, re-export, конструктор, wire в Index/Query/Answer/Retrieve (existing, modify)
5. **`pkg/draftrag/pii.go`** — публичные конструкторы `NewDefaultPIIDetector`, `NewCompositePIIDetector`, re-export (new)
6. **`internal/application/query.go` / `answer.go` / `retrieval.go`** — точки redaction после retrieval (existing, modify)
7. **`pkg/draftrag/search.go` / `search_routing.go`** — redaction RewrittenQuery (existing, modify)

## Bootstrapping Surfaces

- `internal/infrastructure/piidetector/` — новая директория для реализации

## Влияние на архитектуру

- Локальное: добавление опционального PIIDetector в PipelineOptions
- Без влияния на существующие интерфейсы и контракты (nil-безопасность)
- Никаких migration/rollout-последствий

## Acceptance Approach

- AC-001 → PII-redaction перед `core.Index`; проверка через извлечение документа из store
- AC-002 → PII-redaction в retrieval-результатах перед возвратом из `pkg.Query`; сравнение вывода
- AC-003 → PII-redaction в RetrievalResults AnswerResponse; проверка ответа
- AC-004 → CompositePIIDetector с одной категорией; проверка что другая не затронута
- AC-005 → Кастомный `PIIDetector` через интерфейс; проверка в Index
- AC-006 → Nil PIIDetector; содержимое не меняется
- AC-007 → PII-redaction в RewrittenQuery; проверка через SearchBuilder с перехватом

## Данные и контракты

- Новый контракт: `domain.PIIDetector` (интерфейс)
- `pkg/draftrag.PipelineOptions` — новое поле `PIIDetector`
- `application.PipelineOptions` — новое поле `PIIDetector`
- Core domain model не меняется (Document, RetrievalResult, RewrittenQuery без изменений)
- См. `data-model.md`

## Стратегия реализации

### DEC-001 PII-детектор на публичном слое, а не в application

Why: pipeline.go в `pkg/draftrag/` уже является валидационным/guard-слоем (nil context, empty content). PII-redaction логически относится к этой же зоне ответственности. Размещение в application-слое потребовало бы передачи PIIDetector сквозь весь application-слой, хотя application.Pipeline его не использует — это внешний cross-cutting concern.

Tradeoff: application.Pipeline остаётся "чистым" (не знает о PII). Index/Query на публичном слое применяют redaction до/после вызова core. Недостаток: если появится прямой вызов `core.Pipeline` пользователем (через internal-импорт), PII не сработает — это acceptable, т.к. публичный API — через `pkg/draftrag`.

Affects: `pkg/draftrag/draftrag.go`, `pkg/draftrag/pii.go`

Validation: AC-001, AC-002

### DEC-002 CompositePIIDetector + отдельные детекторы

Why: RQ-006 требует независимого включения категорий. Роутер-детектор (composite), который применяет только включённые под-детекторы, проще и прозрачнее, чем единый детектор с флагами. Пользователь может также собрать свой набор из встроенных + кастомных.

Tradeoff: Дополнительный уровень композиции. Но это повторяет паттерн `VectorStoreWithFilters` — опциональные capability через композицию.

Affects: `internal/infrastructure/piidetector/composite.go`, `internal/infrastructure/piidetector/patterns.go`

Validation: AC-004, AC-005

### DEC-003 Redaction RewrittenQuery через SearchBuilder/Rewriter

Why: RewrittenQuery генерируется внутри SearchBuilder при наличии QueryRewriter. PII-redaction применяется после генерации rewritten query, перед использованием для retrieval. Это чище, чем модификация QueryRewriter (который может быть кастомным и не знать о PII).

Tradeoff: Если Rewriter сам генерирует PII (напр., через запрос к LLM с PII-контекстом), redaction происходит пост-фактум. Для MVP это приемлемо; если потребуется pre-redaction ввода для Rewriter — отдельная задача.

Affects: `pkg/draftrag/search_routing.go`

Validation: AC-007

## Incremental Delivery

### MVP (Первая ценность)

- `domain.PIIDetector` интерфейс
- Composite + email/phone/SSN детекторы
- PipelineOption + Index/Query redaction
- Тесты на детекторы + AC-001, AC-002, AC-004, AC-006
- Пример в `examples/`

Критерий: `go test ./...` проходит, пример выводит `<redacted>`.

### Итеративное расширение

- AC-003: Answer redaction
- AC-005: кастомный детектор (пример в документации)
- AC-007: RewrittenQuery redaction
- Credit card pattern-detector
- Benchmark SC-001

## Порядок реализации

1. `internal/domain/pii.go` — интерфейс (нет зависимостей)
2. `internal/infrastructure/piidetector/` — детекторы + composite
3. `pkg/draftrag/pii.go` — re-export + конструкторы
4. Модификация `PipelineOptions` + `NewPipelineWithOptions` (оба слоя)
5. PII-redaction в `pkg.Index` (перед `core.Index`)
6. PII-redaction в `pkg.Query` / `pkg.Answer` / `pkg.Retrieve` (после core)
7. PII-redaction в RewrittenQuery (SearchBuilder)
8. Тесты: unit (детекторы) + integration (pipeline)
9. Пример в `examples/`

Параллельно: 1+2, 3+4, 5+6, 7, 8, 9.

## Риски

- Regexp-детекторы дают false positives (напр., номер кредитки в номере заказа). Mitigation: спецификация паттернов с границами слов (\b); возможность кастомного детектора.
- Производительность на больших документах. Mitigation: SC-001 benchmark; при проблеме — кэширование или ограничение размера.
- PII в метаданных не обрабатывается. Mitigation: явно зафиксировано в spec как out of scope.

## Rollout и compatibility

- Nil PIIDetector → no-op, полная backward compatibility
- Feature flag не требуется
- Специальных rollout-действий нет

## Проверка

- Unit-тесты: минимум 3 кейса на каждый pattern-детектор (SC-002)
- Integration: Pipeline с детектором / без детектора (AC-001, AC-002, AC-006)
- Custom detector: AC-005
- Answer + RewrittenQuery: AC-003, AC-007
- Go vet + golangci-lint без ошибок
- `go test ./...` — все тесты проходят

## Соответствие конституции

- нет конфликтов
- Простота > расширяемость: CompositePIIDetector проще единого конфигуратора
- Интерфейсы > конкретные типы: PIIDetector — интерфейс в domain
