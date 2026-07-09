# Рефакторинг SearchBuilder: Generics + единый routing — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/search_router.go` | T1.1 |
| `pkg/draftrag/search_routing.go` | T2.1 |
| `pkg/draftrag/search.go` | T3.1 |
| `pkg/draftrag/search_builder_test.go` | T3.2, T4.1 |
| `pkg/draftrag/ (benchmark)` | T4.2 |

## Implementation Context

- **Цель MVP:** заменить 7 switch-функций в `search_routing.go` на generic `router[T]` с result-structs; все существующие тесты проходят без изменений.
- **Инварианты/семантика:**
  - Публичные сигнатуры SearchBuilder НЕ меняются (RQ-004)
  - `pickRoute()` остаётся на SearchBuilder (DEC-003)
  - `mapAppError` вызывается внутри `router.execute`, не в handler (DEC-004)
  - handler сигнатура: `func(ctx, q string, topK int, b *SearchBuilder) (T, error)` — билдер целиком (DEC-005)
- **Новые типы (internal, не экспортируемые):**
  - `router[T any]` с полем `handlers [7]func(ctx, q, topK, *SearchBuilder) (T, error)`, методом `execute`
  - 7 result-structs: `rRetrieve`, `rAnswer`, `rCite`, `rInlineCite`, `rStream`, `rStreamSources`, `rStreamCite`
- **Ошибки/коды:** `mapAppError` уже существует; handler-ы возвращают application-level ошибки, execute маппит
- **Proof signals:**
  - `go test ./pkg/draftrag/ -run TestSearchBuilder -count=1` pass
  - `go vet ./pkg/draftrag/... && golangci-lint run ./pkg/draftrag/...` clean
  - `go test -race ./pkg/draftrag/ -count=1` pass
- **References:** DEC-001, DEC-002, DEC-003, DEC-004, DEC-005; AC-001, AC-002, AC-003, AC-004

## Фаза 1: Основа

Цель: подготовить generic-инфраструктуру, на которую будут опираться все output-методы.

- [x] T1.1 Создать `search_router.go` с `router[T any]`, 7-ю result-structs и методом `execute`.  
  `execute` принимает `(ctx, question, topK, route, b *SearchBuilder)`, выполняет валидацию (nil ctx / cancelled ctx через `b.validate()`), dispatch через `handlers[route]`, и `mapAppError`.  
  Touches: `pkg/draftrag/search_router.go`

## Фаза 2: MVP Slice

Цель: output-методы работают через `router[T]`, старые тесты проходят.

- [x] T2.1 Переписать `search_routing.go`: удалить 7 функций `runRetrieve`/`runAnswer`/`runCite`/`runInlineCite`/`runStream`/`runStreamSources`/`runStreamInline`. Вместо них определить 7 var-блоков handler registration (по одному на output-метод), инициализируемых через `sync.OnceValue`. Каждый handler — одна лямбда, вызывающая `pipeline.core.Query*`/`Answer*` и оборачивающая результат в result-struct.  
  Touches: `pkg/draftrag/search_routing.go`

- [x] T2.2 Обновить `search.go`: каждый output-метод вместо `b.pickRoute() → b.run*(...)` вызывает `b.pickRoute() → router.execute(..., r, b) → unpack result-struct`.  
  Touches: `pkg/draftrag/search.go`

- [x] T2.3 Подтвердить MVP: `go test ./pkg/draftrag/ -run TestSearchBuilder -count=1` pass, `go vet`, `golangci-lint`.  
  Touches: `pkg/draftrag/search_builder_test.go` (существующие тесты — верификация без изменений)

## Фаза 3: Основная реализация

Цель: добавить верификацию всех комбинаций и демонстрацию расширяемости.

- [x] T3.1 Добавить `TestSearchBuilder_RouteMatrix` — table-driven test с 42 subtests (6 routes × 7 methods). Каждый subtest конфигурирует SearchBuilder под конкретный маршрут (HyDE/MultiQuery/Hybrid/ParentIDs/Filter/basic) и вызывает конкретный output-метод (Retrieve/Answer/Cite/InlineCite/Stream/StreamSources/StreamCite), проверяя отсутствие `ErrEmptyQuery`, `ErrInvalidTopK`, panic.  
  Touches: `pkg/draftrag/search_builder_test.go`

- [x] T3.2 Добавить prototype `Analyze(ctx) (Analysis, error)` в `search_builder_test.go` как временный тест. Определить result-struct `rAnalyze`, handler-ы, output-метод через `router[rAnalyze]`. Измерить количество строк, добавляемых в тело output-метода (должно быть ≤ 5).  
  Touches: `pkg/draftrag/search_builder_test.go`

## Фаза 4: Проверка

Цель: доказать отсутствие регрессов и оставить пакет в reviewable состоянии.

- [x] T4.1 Запустить `go test -race ./pkg/draftrag/ -count=1` — убедиться в race-free работе handler registration (`sync.OnceValue`) и параллельных вызовов SearchBuilder.  
  Touches: `pkg/draftrag/search_builder_test.go`

- [x] T4.2 Сравнить производительность: `go test -benchmem -bench=. -count=10 > old.txt` до рефакторинга и `> new.txt` после. `benchstat old.txt new.txt` — p > 0.05 для всех бенчмарков. Если есть значимый регресс — проанализировать escape analysis (флаг `-gcflags=-m`).  
  Touches: `pkg/draftrag/search_routing.go`, `pkg/draftrag/search_router.go`, `pkg/draftrag/search.go`

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T2.2, T2.3
- AC-002 -> T3.1
- AC-003 -> T3.2
- AC-004 -> T2.3, T4.1, T4.2

## Заметки

- T1.1 не зависит ни от чего — можно начинать первым.
- T2.1 и T2.2 независимы от T3.1/T3.2 (можно параллелить в рамках implement).
- T4.2 требует baseline `old.txt` ДО начала рефакторинга — сохранить на текущей ветке main перед merge.
- Фаза 4 валидирует SC-001 (LoC reduction) и SC-002 (no perf regression) — измеряются вручную, не automated.
