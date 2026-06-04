---
report_type: inspect
slug: searchbuilder-generics
status: concerns
docs_language: ru
generated_at: 2026-06-04
---

# Inspect Report: searchbuilder-generics

## Scope

- snapshot: проверка spec рефакторинга SearchBuilder — замена 42 switch-функций на generic-router
- artifacts:
  - .speckeep/constitution.summary.md
  - docs/specs/searchbuilder-generics/spec.md
  - docs/specs/searchbuilder-generics/spec.md (read)

## Verdict

- status: concerns
- Основание: spec корректна по структуре и scope, но содержит техническую неоднозначность в generic-модели и риски в измеримых критериях.

## Errors

- none

## Warnings

### W-001 Generic-тип `router[T]` не покрывает разную арность возврата

Spec описывает `router[T]` как единый generic-тип, но output-методы имеют разную арность:
- `Retrieve` → `(RetrievalResult, error)`
- `Answer` → `(string, error)`
- `Cite` → `(string, RetrievalResult, error)`
- `InlineCite` → `(string, RetrievalResult, []InlineCitation, error)`
- `Stream` → `(<-chan string, error)`
- `StreamSources` → `(<-chan string, RetrievalResult, error)`
- `StreamCite` → `(<-chan string, RetrievalResult, []InlineCitation, error)`

Go generics не позволяют параметризовать разную арность одним `T`.  
**Решение (для plan):** использовать именованные result-structs (например, `citeResult{Answer string; Sources RetrievalResult}`), а `router[T any]` возвращает `(T, error)`.  
**Влияние на spec:** не требует изменений — spec описывает intent, не implementation. Но plan должен явно учесть этот паттерн.

### W-002 SC-001 (≤115 строк) — амбициозная оценка

Сейчас `search_routing.go` — 225 строк. После рефакторинга добавятся: определения result-structs (1 struct на каждый multi-return output — ~5 строк × 6 = 30 строк), регистрация handler-ов в `init()` или конструкторе.  
**Риск:** целевые 115 строк могут быть недостижимы; реальный объём ~130-150 строк.  
**Рекомендация:** смягчить до "≥ 40% сокращения" или явно указать "без учёта result-structs".

### W-003 AC-003 (≤5 строк) — не учтены result-structs

AC-003 требует ≤5 строк для нового output-метода. Если нужен новый result-struct (а для `Analyze` он скорее всего нужен), то:
- Определение struct: ~3-4 строки
- Регистрация handler-ов: ~6 строк (по числу маршрутов)
- Вызов execute: ~2 строки

**Итого:** ~11-12 строк вместо 5.  
**Рекомендация:** уточнить в AC-003 "≤5 строк в теле output-метода (без учёта определения struct и handler-ов)" или переформулировать.

## Questions

- Q-001: Result-structs размещать в `search_routing.go` или вынести в отдельный файл `search_types.go`? В spec это не определено — решить на plan.

## Suggestions

### S-001 Result-structs + `mapRoute` pattern

Для plan предлагаю конкретную реализацию:

```go
type router[T any] struct {
    handlers [7]func(ctx, q, topK) (T, error) // index = route
}

func (r *router[T]) execute(m route, ctx, q, topK) (T, error) {
    // validate → call handler[m] → mapAppError
}
```

Каждый output-метод создаёт `router[ResultType]` с pre-filled handlers и вызывает `execute`.  
Это даёт: единая валидация, один `mapAppError`, type safety.

### S-002 Вынести общий `search_router.go`

Создать отдельный файл для `router[T]` и shared-типов, оставив в `search_routing.go` только конфигурацию handler-ов для каждого output-метода.

## Traceability

- AC-001 → покрывается QA: существующие `TestSearchBuilder_*` должны pass
- AC-002 → новый table-driven test `TestSearchBuilder_AllRoutes × Methods`
- AC-003 → prototype в PR, измеряется code review
- AC-004 → CI gate (vet + linter)
- RQ-001..RQ-005 → архитектурные требования, верифицируются code review

## Next Step

- addressed warnings before planning (особенно W-001 — result-struct pattern)
- safe to continue to plan with noted caveats

Готово к: /speckeep.plan searchbuilder-generics
