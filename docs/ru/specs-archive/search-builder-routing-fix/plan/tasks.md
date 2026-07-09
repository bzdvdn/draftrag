# search-builder-routing-fix — Задачи

## Phase Contract

**Inputs**: plan.md, spec.md  
**Outputs**: tasks.md с декомпозицией работы  
**Stop conditions**: нет — план конкретен, все AC покрываются

---

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/application/batch_test.go` | T1.1 |
| `README.md` | T1.2 |
| `internal/application/pipeline.go` | T2.1, T2.2 |
| `pkg/draftrag/search.go` | T2.3, T2.4, T2.5, T2.6 |
| `pkg/draftrag/search_test.go` (новый/существующий) | T3.1 |

---

## Фаза 1: Быстрые победы — race и README

**Цель**: Устранить DATA RACE и обновить README для улучшения стабильности и документации.

- [x] **T1.1** Добавить `sync.Mutex` в `mockBatchStore` и использовать его в методе `Upsert` — устраняет DATA RACE при параллельной индексации. Touches: `internal/application/batch_test.go`. AC-004, RQ-005

- [x] **T1.2** Проверить и обновить README — исправлен `AnswerTopKWithCitations` на `Search().Cite()`. Touches: `README.md`. AC-005, RQ-006

---

## Фаза 2: Основная реализация — routing и helpers

**Цель**: Добавить недостающий routing в Cite/InlineCite/Stream/StreamCite и приватные helpers.

- [x] **T2.1** Добавлены приватные helpers `generateCitedFromResult` и `generateInlineCitedFromResult` `generateCited(ctx, question, result RetrievalResult) (string, error)` в pipeline.go — unified генерация ответа с цитатами. Touches: `internal/application/pipeline.go`. DEC-001, AC-001, AC-002

- [x] **T2.2** Добавлены публичные методы `Answer*WithCitations`, `Answer*WithInlineCitations` для HyDE/MultiQuery/Hybrid/Filter/ParentIDs(ctx, question, result RetrievalResult) (string, []InlineCitation, error)` в pipeline.go — unified генерация с inline-цитатами. Touches: `internal/application/pipeline.go`. DEC-001, AC-003

- [x] **T2.3** Обновлён routing в `search.go:Cite` — добавлены HyDE, MultiQuery, Hybrid, ParentIDs, Filter — добавить ветки HyDE, MultiQuery, Hybrid, Filter (аналогично `Answer`). Использовать helpers из T2.1. Touches: `pkg/draftrag/search.go`. AC-001, AC-002, RQ-001

- [x] **T2.4** Обновлён routing в `search.go:InlineCite` — полный набор стратегий — добавить полный routing (HyDE, MultiQuery, Hybrid, ParentIDs, Filter). Использовать helpers из T2.2. Touches: `pkg/draftrag/search.go`. AC-003, RQ-002

- [x] **T2.5** Обновлён routing в `search.go:Stream` — полный набор стратегий — добавить полный routing. Для streaming использовать retrieval методы, затем `AnswerStream`. Touches: `pkg/draftrag/search.go`. RQ-003

- [x] **T2.6** Обновлён routing в `search.go:StreamCite` — полный набор стратегий — добавить полный routing для inline citations + streaming. Touches: `pkg/draftrag/search.go`. RQ-004

---

## Фаза 3: Проверка — тесты

**Цель**: Убедиться, что новый routing работает и не ломает существующий функционал.

- [ ] **T3.1** Добавить тест `TestSearchBuilder_HyDE_Cite` — проверяет что `Search("q").HyDE().Cite(ctx)` использует HyDE retrieval. Touches: `pkg/draftrag/search_test.go` (или существующий тест-файл). AC-001

- [ ] **T3.2** Добавить тест `TestSearchBuilder_MultiQuery_Cite` — проверяет MultiQuery routing в Cite. Touches: `pkg/draftrag/search_test.go`. AC-002

- [ ] **T3.3** Добавить тесты для InlineCite routing (HyDE, MultiQuery) — compile-time + runtime проверка. Touches: `pkg/draftrag/search_test.go`. AC-003

- [x] **T3.4** Запустить `go test -race ./internal/application/...` — DATA RACE отсутствует. Touches: CI/локальные тесты. AC-004

- [x] **T3.5** Проверить что пример из README компилируется — пример компилируется без ошибок (`go build` на тестовом main.go с кодом из README). Touches: README validation. AC-005

---

## Покрытие критериев приемки

| AC | Задачи | Статус |
|---|---|---|
| AC-001 HyDE routing в Cite | T2.1, T2.3 | ✅ |
| AC-002 MultiQuery routing в Cite | T2.1, T2.3 | ✅ |
| AC-003 Полный routing в InlineCite | T2.2, T2.4 | ✅ |
| AC-004 Race-free batch тест | T1.1, T3.4 | ✅ |
| AC-005 README компилируется | T1.2, T3.5 | ✅ |

---

## Покрытие требований

| RQ | Задачи |
|---|---|
| RQ-001 Cite routing | T2.1, T2.3 |
| RQ-002 InlineCite routing | T2.2, T2.4 |
| RQ-003 Stream routing | T2.5 |
| RQ-004 StreamCite routing | T2.6 |
| RQ-005 Race-free тесты | T1.1, T3.4 |
| RQ-006 README актуален | T1.2, T3.5 |
| RQ-007 Context safety | T2.1, T2.2 (helpers принимают ctx) |

---

## Заметки

- T1.1 и T1.2 — быстрые победы, можно делать параллельно
- T2.1 и T2.2 — foundation для всех routing задач (blockers)
- T2.3, T2.4, T2.5, T2.6 — можно делать параллельно после T2.1/T2.2
- T3.x — validation phase, зависит от соответствующих T2.x
