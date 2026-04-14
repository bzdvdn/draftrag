# api-resilience-fixes: задачи

## Phase Contract

Inputs: plan.md, summary.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех AC.
Stop if: задачи расплывчаты или coverage нельзя сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/models.go | T1.1 |
| internal/domain/models_test.go | T1.1 |
| internal/infrastructure/vectorstore/hybrid.go | T1.1 |
| internal/infrastructure/vectorstore/hybrid_test.go | T1.1 |
| internal/infrastructure/vectorstore/hybrid_bench_test.go | T1.1 |
| internal/infrastructure/resilience/circuitbreaker.go | T2.1 |
| internal/infrastructure/resilience/circuitbreaker_test.go | T2.2 |

## Фаза 1: Переименование BMFinaKK -> BMFinalK

Цель: устранить опечатку в публичном API через глобальный replace_all по 5 файлам; убедиться что сборка и тесты чистые.

- [x] T1.1 Переименовать BMFinaKK -> BMFinalK во всех 5 Go-файлах группы A — grep по *.go возвращает 0 совпадений; go build ./... ok (AC-001, AC-002). Touches: internal/domain/models.go, internal/domain/models_test.go, internal/infrastructure/vectorstore/hybrid.go, internal/infrastructure/vectorstore/hybrid_test.go, internal/infrastructure/vectorstore/hybrid_bench_test.go
- [x] T1.2 Прогнать go build ./... и go test ./internal/domain/... ./internal/infrastructure/vectorstore/... — зелёный (AC-001, AC-002). Touches: internal/domain/models.go

## Фаза 2: Probe-семафор в CircuitBreaker half-open

Цель: добавить флаг probeSent под существующим mutex; ровно одна горутина проходит как probe в состоянии half-open.

- [x] T2.1 Добавить поле probeSent bool в CircuitBreaker и обновить CanExecute, RecordSuccess, RecordFailure — half-open разрешает ровно 1 горутину; флаг сбрасывается при выходе из half-open (AC-003, AC-004, DEC-001, DEC-002). Touches: internal/infrastructure/resilience/circuitbreaker.go
- [x] T2.2 Добавить TestCircuitBreaker_HalfOpen_ParallelProbe — 10 горутин одновременно, счётчик разрешений == 1 (AC-003); и TestCircuitBreaker_HalfOpen_ProbeReset — два цикла open->half-open, оба раза счётчик == 1 (AC-004). Touches: internal/infrastructure/resilience/circuitbreaker_test.go
- [x] T2.3 Прогнать go test ./internal/infrastructure/resilience/... -race — все тесты зелёные, race detector чистый (AC-003, AC-004). Touches: internal/infrastructure/resilience/circuitbreaker_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2
- AC-002 -> T1.1, T1.2
- AC-003 -> T2.1, T2.2, T2.3
- AC-004 -> T2.1, T2.2, T2.3

## Заметки

- T1.1 использует replace_all для каждого файла — не ручной поиск-замену.
- T1.2 Touches указывает один файл для сигнала implement-агенту что этот файл уже прочитан; фактически команды go build/test не привязаны к конкретному файлу.
- T2.3 запускается с -race: параллельный тест должен быть чистым под race detector.
- Фазы независимы по коду, но выполнять последовательно для простоты отката.
