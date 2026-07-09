# api-resilience-fixes: план

## Phase Contract

Inputs: spec.md, inspect.md (pass), `internal/domain/models.go`, `internal/infrastructure/resilience/circuitbreaker.go`.
Outputs: plan.md, data-model.md.

## Цель

Две независимые точечные правки в рамках одного package:
1. **Rename**: глобальная замена `BMFinaKK` → `BMFinalK` в 5 Go-файлах через `replace_all`; поведение не меняется.
2. **Probe gate**: добавить поле `probeSent bool` в `CircuitBreaker`; защитить его через уже существующий `mu sync.RWMutex`; установить флаг при первом разрешении в `half-open`, сбросить при выходе из `half-open`.

## Scope

- Группа A (rename): `internal/domain/models.go`, `internal/domain/models_test.go`, `internal/infrastructure/vectorstore/hybrid.go`, `internal/infrastructure/vectorstore/hybrid_test.go`, `internal/infrastructure/vectorstore/hybrid_bench_test.go`
- Группа B (probe gate): `internal/infrastructure/resilience/circuitbreaker.go`, `internal/infrastructure/resilience/circuitbreaker_test.go`
- `pkg/draftrag/draftrag.go` — не трогается; `HybridConfig = domain.HybridConfig` — type alias, поле переименуется автоматически

## Implementation Surfaces

- **`internal/domain/models.go` (существующая)** — struct field rename + godoc + `Validate()` + `DefaultHybridConfig()`; 5 вхождений строки `BMFinaKK`.
- **`internal/domain/models_test.go` (существующая)** — 4 вхождения в тест-кейсах и имени тест-функции.
- **`internal/infrastructure/vectorstore/hybrid.go` (существующая)** — 1 вхождение: `config.BMFinaKK`.
- **`internal/infrastructure/vectorstore/hybrid_test.go` (существующая)** — 3 вхождения в struct literals и assert-строках.
- **`internal/infrastructure/vectorstore/hybrid_bench_test.go` (существующая)** — 3 вхождения в struct literals.
- **`internal/infrastructure/resilience/circuitbreaker.go` (существующая)** — добавить поле `probeSent bool`; изменить `CanExecute()` (case `CircuitOpen` и `CircuitHalfOpen`); изменить `RecordSuccess()` и `RecordFailure()` (сброс флага в `CircuitHalfOpen`).
- **`internal/infrastructure/resilience/circuitbreaker_test.go` (существующая)** — добавить тест на параллельный `half-open`.

## Влияние на архитектуру

- Группа A: нулевое влияние на семантику и внешние интерфейсы; type alias в `pkg/draftrag` распространяет переименование автоматически.
- Группа B: изменение только внутренней state machine; публичный API (`CanExecute`, `RecordSuccess`, `RecordFailure`, `State`) не меняется; `ResilienceEmbedder` и `ResilienceLLMProvider` не затрагиваются.
- Нет breaking changes; нет необходимости в migration или feature flag.

## Acceptance Approach

- **AC-001** → `replace_all` по строке `BMFinaKK` в каждом из 5 файлов группы A; затем `grep -r "BMFinaKK" --include="*.go"` → 0 совпадений; `go build ./...` → ok.
- **AC-002** → `go test ./internal/domain/... ./internal/infrastructure/vectorstore/...` после rename — тесты проходят без изменений в логике.
- **AC-003** → добавить `probeSent bool` в struct; в `CanExecute()` case `CircuitHalfOpen` проверять и устанавливать флаг под `mu.Lock()`; тест с 10 параллельными горутинами — счётчик разрешений == 1.
- **AC-004** → в `RecordSuccess()` и `RecordFailure()` в ветке `CircuitHalfOpen` сбрасывать `probeSent = false`; тест с двумя последовательными циклами `open → half-open` — оба раза счётчик == 1.

## Данные и контракты

Фича не вводит новых сущностей, не затрагивает API или event boundaries. `data-model.md` — placeholder.

## Стратегия реализации

- **DEC-001** `probeSent bool` под существующим `mu sync.RWMutex` (не `atomic.Bool`)
  Why: `CanExecute()` уже берёт `mu.Lock()` (не RLock) — добавление `bool` под тот же mutex не вводит нового примитива и сохраняет единую точку синхронизации.
  Tradeoff: `atomic.Bool` не потребовал бы lock, но `CanExecute()` в любом случае меняет `state` — lock уже обязателен.
  Affects: `internal/infrastructure/resilience/circuitbreaker.go`
  Validation: тест на параллельность (AC-003): 10 горутин, ровно 1 проходит.

- **DEC-002** Флаг устанавливается при `CircuitOpen → CircuitHalfOpen` переходе (в `CanExecute`) и в ветке `CircuitHalfOpen` (для повторных вызовов)
  Why: переход в `half-open` происходит в `CanExecute()` — первая горутина уже получает `nil` как часть перехода. Без установки `probeSent = true` в этой ветке, вторая горутина попадёт в `CircuitHalfOpen` case и тоже пройдёт.
  Tradeoff: нет — это единственный корректный вариант.
  Affects: `circuitbreaker.go:CanExecute()`
  Validation: тест с двумя одновременными горутинами, из которых первая тригерит `Open → HalfOpen`.

  Конкретная схема изменений `CanExecute()`:
  ```
  case CircuitOpen:
      if time.Since(...) >= cb.timeout {
          cb.state = CircuitHalfOpen
          cb.probeSent = true   // ← первая probe уже выдана
          return nil
      }
      return ErrCircuitOpen

  case CircuitHalfOpen:
      if cb.probeSent {
          return ErrCircuitOpen // ← все последующие — отказ
      }
      cb.probeSent = true
      return nil
  ```

  Сброс `probeSent = false` — в `RecordSuccess()` и `RecordFailure()` в ветке `CircuitHalfOpen`.

## Incremental Delivery

### MVP (Первая ценность)

- Группа A (rename) — независима, может быть выполнена и проверена первой.
- Критерий: `go build ./...` + `go test ./internal/...` зелёный.

### Итеративное расширение

- Группа B (probe gate) — после группы A; добавляет тест на параллельность.
- Критерий: `go test ./internal/infrastructure/resilience/...` зелёный, включая новый параллельный тест.

## Порядок реализации

1. Rename `BMFinaKK` → `BMFinalK` во всех 5 файлах группы A.
2. `go build ./...` + `go test ./internal/domain/... ./internal/infrastructure/vectorstore/...` — убедиться, что rename чистый.
3. Добавить `probeSent bool` в `CircuitBreaker` + изменить `CanExecute`, `RecordSuccess`, `RecordFailure`.
4. Добавить тест на параллельный `half-open`.
5. `go test ./internal/infrastructure/resilience/...`.

Шаги 1-2 и 3-5 независимы, но выполнять последовательно для простоты отката.

## Риски

- **Риск:** пропущенное вхождение `BMFinaKK` в `.go` файле за пределами 5 известных.
  Mitigation: после rename — `grep -r "BMFinaKK" --include="*.go"` должен дать 0; если нет — найти и исправить до коммита.

- **Риск:** тест на параллельность даёт flaky результаты из-за планировщика.
  Mitigation: использовать `sync.WaitGroup` + `sync/atomic` счётчик разрешений; запускать N=10 горутин с `runtime.Gosched()` перед вызовом — детерминизм достаточен для unit-теста.

## Rollout и compatibility

Специальных rollout-действий не требуется. Обе правки — внутренние; ни одна не меняет публичный API-контракт.

## Проверка

- `grep -r "BMFinaKK" --include="*.go"` → 0 (AC-001).
- `go build ./...` → ok (AC-001).
- `go test ./internal/domain/... ./internal/infrastructure/vectorstore/...` → ok (AC-002).
- `go test ./internal/infrastructure/resilience/... -race` → ok (AC-003, AC-004); флаг `-race` важен для параллельного теста.

## Соответствие конституции

- **Чистая архитектура**: изменения в `internal/domain` и `internal/infrastructure` — слои не нарушены; `pkg/draftrag` не трогается ✓
- **Интерфейсная абстракция**: публичные интерфейсы `VectorStore`, `CircuitBreaker`-методы не меняются ✓
- **Тестируемость**: добавляется unit-тест с mock-горутинами ✓
- **`go vet`, `go fmt`**: rename и добавление `bool` тривиальны; vet-рисков нет ✓
- **Godoc на русском**: godoc поля `probeSent` и обновлённый `BMFinalK` — на русском ✓
- **Два изменения в одном спеке**: явное решение пользователя; конституция не запрещает явно, лишь рекомендует изоляцию ✓
