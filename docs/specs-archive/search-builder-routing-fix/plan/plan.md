# search-builder-routing-fix — План

## Phase Contract

**Inputs**: spec.md, inspect.md, кодовая база (search.go, pipeline.go, batch_test.go, README.md)  
**Outputs**: plan.md, data-model.md (placeholder), tasks.md  
**Stop conditions**: нет — spec одобрен, implementation surface ясен

---

## Цель

Устранить три конкретных дефекта:
1. Тихое игнорирование retrieval-стратегий (HyDE/MultiQuery/Hybrid/ParentIDs/Filter) в `Cite`/`InlineCite`/`Stream`/`StreamCite`
2. DATA RACE в `mockBatchStore` при `go test -race`
3. Устаревшие ссылки на удалённые методы в README

---

## Scope

### Входит
- `pkg/draftrag/search.go` — routing в `Cite`, `InlineCite`, `Stream`, `StreamCite` (HyDE, MultiQuery, Hybrid, ParentIDs, Filter)
- `internal/application/pipeline.go` — приватные helpers для citation/stream + публичные методы при необходимости
- `internal/application/batch_test.go` — `sync.Mutex` в `mockBatchStore`
- `README.md` — обновление feature list и code example

### Не входит
- Новые retrieval стратегии
- Рефакторинг application layer
- Новые провайдеры

---

## Implementation Surfaces

| Surface | Тип | Описание |
|---|---|---|
| `pkg/draftrag/search.go` | Существующий | Добавить routing во все методы: `Cite`, `InlineCite`, `Stream`, `StreamCite` — аналогично `Answer` и `Retrieve` |
| `internal/application/pipeline.go` | Существующий | Приватные helpers: `generateCited`, `generateInlineCited`; возможно публичные методы `AnswerWithCitations`, `AnswerWithInlineCitations` для HyDE/MultiQuery/Hybrid |
| `internal/application/batch_test.go` | Существующий | Добавить `sync.Mutex` в `mockBatchStore` для защиты `chunks` |
| `README.md` | Существующий | Удалить/обновить устаревшие методы, обновить примеры |

---

## Architectural Impact

- **Локальное**: Изменения только в `search.go` (routing), `pipeline.go` (helpers), тесте и README
- **Compatibility**: Сохраняется backward compatibility — новый routing добавляет функциональность, не ломает существующую
- **Consistency**: `Cite`/`InlineCite`/`Stream`/`StreamCite` будут вести себя аналогично `Answer` и `Retrieve`

---

## Acceptance Approach

### AC-001 HyDE routing в Cite
- **Реализация**: В `search.go:Cite` добавить ветку `if b.hyDE { ... }` вызывающую `AnswerWithCitations` или `generateCited` поверх `QueryHyDE`
- **Surface**: `pkg/draftrag/search.go`
- **Evidence**: Тест `TestSearchBuilder_HyDE_Cite` pass

### AC-002 MultiQuery routing в Cite
- **Реализация**: В `search.go:Cite` добавить ветку `if b.multiQuery > 0 { ... }`
- **Surface**: `pkg/draftrag/search.go`
- **Evidence**: Тест `TestSearchBuilder_MultiQuery_Cite` pass

### AC-003 Полный routing в InlineCite
- **Реализация**: В `search.go:InlineCite` добавить все routing ветки (HyDE, MultiQuery, Hybrid, ParentIDs, Filter)
- **Surface**: `pkg/draftrag/search.go`
- **Evidence**: Тесты pass

### AC-004 Race-free batch тест
- **Реализация**: Добавить `sync.Mutex` в `mockBatchStore` и использовать его в `Upsert`
- **Surface**: `internal/application/batch_test.go`
- **Evidence**: `go test -race ./internal/application/...` exit 0

### AC-005 README компилируется
- **Реализация**: Проверить README на устаревшие методы (grep), обновить примеры
- **Surface**: `README.md`
- **Evidence**: Пример из README компилируется

---

## Data и Contracts

**Data model**: Нет новых сущностей — используются существующие `RetrievalResult`, `InlineCitation`.

**API Contracts**: Нет изменений в публичных API — только добавление логики внутри существующих методов.

---

## Стратегия реализации

### DEC-001 Приватные helpers в pipeline.go
- **Why**: Избежать дублирования логики генерации с цитатами в 4+ методах
- **Tradeoff**: Небольшое увеличение pipeline.go (уже большой файл)
- **Affects**: `internal/application/pipeline.go`
- **Validation**: Все `Cite`/`InlineCite` методы используют одни helpers

### DEC-002 Рефакторинг routing в SearchBuilder
- **Why**: Пользователь ожидает идентичного поведения `.HyDE().Cite()` и `.HyDE().Answer()`
- **Tradeoff**: Нет — консистентность улучшает UX
- **Affects**: `pkg/draftrag/search.go`

### DEC-003 Порядок routing priority (как в Answer/Retrieve)
```
HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic
```

---

## Incremental Delivery

### MVP (первая ценность)
1. **Mutex в mockBatchStore** (AC-004) — быстрый win, делает CI stable
2. **README fix** (AC-005) — улучшает onboarding

### Итеративное расширение
3. **Routing в Cite** — HyDE + MultiQuery (AC-001, AC-002)
4. **Routing в InlineCite** — полный набор (AC-003)
5. **Routing в Stream/StreamCite** — аналогично

---

## Порядок реализации

1. **T1.1** Добавить `sync.Mutex` в `mockBatchStore` — блокер для стабильного CI
2. **T1.2** Проверить и обновить README — быстрый win
3. **T2.1** Добавить `generateCited` helper в pipeline.go
4. **T2.2** Добавить `generateInlineCited` helper в pipeline.go
5. **T2.3** Обновить routing в `search.go:Cite`
6. **T2.4** Обновить routing в `search.go:InlineCite`
7. **T2.5** Обновить routing в `search.go:Stream`
8. **T2.6** Обновить routing в `search.go:StreamCite`
9. **T3.1** Написать тесты для новых routing путей

---

## Риски

| Риск | Mitigation |
|---|---|
| pipeline.go становится слишком большим | Приватные helpers компактны; при необходимости вынести в отдельный файл |
| Сложность routing логики | Точное следование порядку в Answer/Retrieve; unit-тесты |
| Regression в существующих тестах | Все существующие тесты должны продолжать проходить |

---

## Rollout и compatibility

- **Backward compatible**: Да — добавляется функциональность, не ломается существующая
- **Breaking changes**: Нет
- **Migration**: Не требуется

---

## Проверка

| Проверка | AC | Метод |
|---|---|---|
| `go test -race ./internal/application/...` | AC-004 | Автоматический |
| `TestSearchBuilder_HyDE_Cite` | AC-001 | Автоматический |
| `TestSearchBuilder_MultiQuery_Cite` | AC-002 | Автоматический |
| Тесты InlineCite | AC-003 | Автоматический |
| README пример компилируется | AC-005 | Ручной / CI |

---

## Соответствие конституции

| Принцип | Применение | Статус |
|---|---|---|
| Интерфейсная абстракция | Используются существующие интерфейсы | ✅ |
| Чистая архитектура | Изменения в application и pkg слоях | ✅ |
| Контекстная безопасность | Все новые методы принимают `context.Context` | ✅ (RQ-007) |
| Тестируемость | Все AC имеют проверяемые evidence | ✅ |
| Языковая политика | Комментарии на русском | ✅ |

**Конфликтов нет.**

---

**Slug**: `search-builder-routing-fix`  
**Следующая команда**: `/speckeep.tasks search-builder-routing-fix`
