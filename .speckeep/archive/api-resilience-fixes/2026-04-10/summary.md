---
slug: api-resilience-fixes
generated_at: 2026-04-10
---

## Goal

Исправить опечатку `BMFinaKK` → `BMFinalK` в публичном API `HybridConfig` и устранить concurrency-баг в `CircuitBreaker.CanExecute()`: в состоянии `half-open` ровно одна горутина должна получать probe-разрешение.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | BMFinaKK переименован во всех `.go` файлах | `go build ./...` ok; `grep "BMFinaKK" **/*.go` — 0 совпадений |
| AC-002 | Поведение HybridConfig не изменилось | `go test ./internal/domain/... ./internal/infrastructure/vectorstore/...` — pass |
| AC-003 | half-open пропускает ровно 1 горутину | тест с N=10 горутин: счётчик разрешений == 1 |
| AC-004 | probe-флаг сбрасывается для следующего цикла | тест с 2 циклами open→half-open: оба раза счётчик == 1 |

## Out of Scope

- Изменение семантики или дефолтов `HybridConfig`
- Новые поля или методы `CircuitBreaker`
- Публичные интерфейсы resilience-обёрток
- Поведение `CircuitClosed` и `CircuitOpen`
- Архивные `.draftspec/` файлы
