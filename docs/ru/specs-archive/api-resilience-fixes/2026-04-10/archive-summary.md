---
slug: api-resilience-fixes
status: completed
archived_at: 2026-04-10
---

# Archive Summary: api-resilience-fixes

## Status

completed

## Reason

Обе корректности-правки реализованы, проверены и подтверждены тестами:
1. Переименование `BMFinaKK` → `BMFinalK` — опечатка в публичном API устранена до появления внешних пользователей.
2. Probe-семафор в `CircuitBreaker.half-open` — concurrency-баг устранён; ровно одна горутина получает probe-разрешение.

## Completed Scope

- `internal/domain/models.go` — поле переименовано, godoc обновлён
- `internal/domain/models_test.go` — тесты обновлены
- `internal/infrastructure/vectorstore/hybrid.go` — обращение к полю обновлено
- `internal/infrastructure/vectorstore/hybrid_test.go` — тесты обновлены
- `internal/infrastructure/vectorstore/hybrid_bench_test.go` — бенчмарки обновлены
- `internal/infrastructure/resilience/circuitbreaker.go` — добавлен `probeSent bool`, обновлены `CanExecute`, `RecordSuccess`, `RecordFailure`
- `internal/infrastructure/resilience/circuitbreaker_test.go` — добавлены `TestCircuitBreaker_HalfOpen_ParallelProbe` и `TestCircuitBreaker_HalfOpen_ProbeReset`

## Acceptance

- AC-001: `grep "BMFinaKK" **/*.go` → 0; `go build ./...` → ok
- AC-002: `go test ./internal/domain/... ./internal/infrastructure/vectorstore/...` → ok
- AC-003: тест с 10 горутинами — счётчик == 1; `-race` чист
- AC-004: тест с 2 циклами open→half-open — оба раза счётчик == 1; `-race` чист

## Notable Deviations

none
