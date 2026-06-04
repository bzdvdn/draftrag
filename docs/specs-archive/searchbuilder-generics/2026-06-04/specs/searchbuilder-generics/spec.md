# Рефакторинг SearchBuilder: Generics + единый routing

## Scope Snapshot

- In scope: замена 42 дублирующих switch-функций в `search_routing.go` на один обобщённый generic-маршрутизатор с type-safe возвращаемыми типами.
- Out of scope: изменение публичного API SearchBuilder, добавление/удаление маршрутов или output-методов, рефакторинг `internal/application/pipeline.go`.

## Цель

Разработчик, добавляющий новый output-метод (например, `Analyze`) или новый retrieval-маршрут, сейчас вынужден писать 7 новых switch-функций × N кейсов = O(n*m) дублирования. После рефакторинга добавление output-метода потребует только описать тип результата и одну функцию-диспетчер; добавление маршрута — один case в central router. Успех измеряется: количество строк в `search_routing.go` сокращается ≥50% при полной сохранности тестов.

## Основной сценарий

1. Исходная точка: `SearchBuilder` имеет 7 output-методов, каждый с дублирующим switch по 6 маршрутам. Итого 42 кейса в 7 функциях с идентичной структурой.
2. Рефакторинг: вводится generic-тип `routeHandler[T]` с методом `addRoute(route, fn)` и единым `execute(ctx, q, topK) T`. Каждый output-метод создаёт свой `routeHandler[ReturnType]`, регистрирует лямбды и вызывает `execute`.
3. Результат: 7 регистраций + 1 execute вместо 7 switch + общая логика валидации/маппинга ошибок вынесена в base.
4. Fallback: если generic-решение ухудшает читаемость или производительность (escape analysis, allocs), откат к текущей структуре.

## User Stories

- P1 (MVP): generic `router[T any]` покрывает все существующие output-методы, тесты проходят без изменений.
- P2: пример добавления нового output-метода (`Analyze`) требует только 3 строки вместо 42 — демонстрация расширяемости.

## MVP Slice

Generic `routeHandler` с поддержкой возвращаемых типов:

| Тип | Output-методы | AC |
|-----|---------------|----|
| `RetrievalResult` | Retrieve | AC-001 |
| `string` | Answer | AC-001 |
| `(string, RetrievalResult)` | Cite | AC-001 |
| `(string, RetrievalResult, []InlineCitation)` | InlineCite | AC-001 |
| `(<-chan string)` | Stream | AC-001 |
| `(<-chan string, RetrievalResult)` | StreamSources | AC-001 |
| `(<-chan string, RetrievalResult, []InlineCitation)` | StreamCite | AC-001 |

## First Deployable Outcome

- `search_routing.go` переписан: один `router[T]` вместо 7 switch-функций.
- Все существующие тесты в `search_builder_test.go` проходят без изменений.
- `go vet ./...` и `golangci-lint run` чисты.

## Scope

- `pkg/draftrag/search_routing.go` — полная переработка.
- `pkg/draftrag/search.go` — минимальные изменения (если требуется новый internal-метод).
- `pkg/draftrag/search_builder_test.go` — только добавление теста на новый output-метод (P2).

## Контекст

- Go 1.23+ с полной поддержкой generics.
- `SearchBuilder` — единственный пользователь public API, ломать его нельзя.
- Все output-методы повторяют pattern: validate → pickRoute → switch → mapError. Повторение — единственная причина рефакторинга.
- Производительность: generic-функции в Go инлайнятся; ожидается zero alloc overhead сверх текущего.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять generic-тип `router[T]`, параметризованный возвращаемым типом output-метода.
- RQ-002 Каждый output-метод ДОЛЖЕН регистрировать обработчики маршрутов при создании `router[T]` (один раз, не в хот-пасе).
- RQ-003 `router.execute` ДОЛЖЕН принимать `(ctx, question, topK) → (T, error)` и выполнять единую валидацию + mapAppError.
- RQ-004 Сигнатуры публичных методов SearchBuilder НЕ ДОЛЖНЫ измениться.
- RQ-005 Добавление нового output-метода (P2) ДОЛЖНО требовать ≤5 строк в `search_routing.go`.

## Вне scope

- Рефакторинг `internal/application/pipeline.go` или методов `pipeline.core.Query*` / `Answer*`.
- Изменение `route` enum, добавление или удаление маршрутов.
- Изменение публичного API `pipeline.Search()`.
- Оптимизация производительности SearchBuilder (не является целью, хотя не должна ухудшиться).
- Переход на кодогенерацию для роутинга.

## Критерии приемки

### AC-001 Все output-методы работают через generic router

- Почему это важно: гарантирует, что рефакторинг не сломал существующее поведение.
- **Given** pipeline с InMemoryStore, mockLLM и fixedEmbedder
- **When** каждый output-метод вызывается с теми же параметрами, что в `search_builder_test.go`
- **Then** результат идентичен текущему: те же возвращаемые значения, те же ошибки
- Evidence: `go test ./pkg/draftrag/ -run TestSearchBuilder -count=1` — все тесты pass

### AC-002 Покрытие всех комбинаций маршрут × output-метод

- Почему это важно: generic router должен корректно диспатчить все 42 комбинации.
- **Given** pipeline с поддержкой всех маршрутов (HyDE, MultiQuery, Hybrid, ParentIDs, Filter, basic)
- **When** каждый output-метод вызывается с каждым маршрутом
- **Then** ни одна комбинация не возвращает `ErrInvalidTopK`, `ErrEmptyQuery` или panic
- Evidence: table-driven test с 42 subtest-ами (6 routes × 7 methods)

### AC-003 Добавление нового output-метода — 5 строк

- Почему это важно: метрика расширяемости после рефакторинга.
- **Given** generic `router[T]`
- **When** добавляется output-метод `Analyze(ctx) (Analysis, error)`
- **Then** требуется только: (1) определить `Analysis`, (2) создать `router[Analysis]`, (3) зарегистрировать 6 обработчиков, (4) вызвать `execute`
- Evidence: prototype-код добавлен в PR, lines of code ≤ 5 в `search_routing.go` (без учёта типов возврата)

### AC-004 `go vet` и `golangci-lint` без errors

- Почему это важно: код должен быть идиоматичным и безопасным.
- **Given** рефакторинг завершён
- **When** `go vet ./pkg/draftrag/... && golangci-lint run ./pkg/draftrag/...`
- **Then** exit code 0
- Evidence: CI-artefact

## Допущения

- Go 1.23+ — generics без ограничений.
- SearchBuilder не имеет exported-полей; изменения internal-структуры не ломают API.
- mockLLM и fixedEmbedder из тестов достаточны для проверки роутинга (не требуется интеграционных тестов).

## Критерии успеха

- SC-001 Строк кода в `search_routing.go` сокращено ≥ 50% (с ~225 до ≤ 115).
- SC-002 `go test -bench=BenchmarkSearchBuilder -benchmem` не показывает значимого регресса (p > 0.05) vs baseline.

## Краевые случаи

- nil context — panic в каждом output-методе (текущее поведение, сохранено).
- cancelled context — возврат `context.Canceled`.
- Маршрут, не поддерживаемый store (Filter без VectorStoreWithFilters) — `ErrFiltersNotSupported`.
- Маршрут, не поддерживаемый LLM (Stream без StreamingLLMProvider) — `ErrStreamingNotSupported`.

## Открытые вопросы

- none
