# search-builder-routing-fix — Summary

**Статус**: ✅ Inspected (готова к плану)  
**Branch**: `feature/search-builder-routing-fix`  

---

## Scope

3 бага в SearchBuilder и тестах:
1. `Cite`/`InlineCite`/`Stream`/`StreamCite` игнорируют HyDE/MultiQuery/Hybrid/ParentIDs/Filter
2. Race condition в `mockBatchStore` при `go test -race`
3. Устаревшие ссылки на удалённые методы в README

---

## Требования (7)

| ID | Описание |
|---|---|
| RQ-001 | `Cite` поддерживает routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic |
| RQ-002 | `InlineCite` поддерживает такой же routing |
| RQ-003 | `Stream` поддерживает routing |
| RQ-004 | `StreamCite` поддерживает routing |
| RQ-005 | `go test -race ./internal/application/...` без DATA RACE |
| RQ-006 | README содержит только существующие методы |
| RQ-007 | Новые методы принимают `context.Context` первым параметром |

---

## Критерии приемки (5)

| ID | Описание | Evidence |
|---|---|---|
| AC-001 | HyDE routing в Cite | `TestSearchBuilder_HyDE_Cite` pass |
| AC-002 | MultiQuery routing в Cite | `TestSearchBuilder_MultiQuery_Cite` pass |
| AC-003 | Полный routing в InlineCite | тесты pass |
| AC-004 | Race-free batch тест | `go test -race` exit 0 |
| AC-005 | README компилируется | `go build` без ошибок |

---

## Файлы затронуты

- `internal/application/pipeline.go` — приватные helpers
- `pkg/draftrag/search.go` — routing
- `internal/application/batch_test.go` — mutex в mockBatchStore
- `README.md` — обновление примеров

---

## Результат инспекции

✅ **PASS** — нет конфликтов с конституцией, все AC проверяемы, scope чёткий.

---

**Следующая команда**: `/speckeep.plan search-builder-routing-fix`
