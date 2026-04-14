# API и concurrency: исправление опечатки BMFinaKK и race в circuit breaker half-open

## Scope Snapshot

- In scope: переименование `HybridConfig.BMFinaKK` → `BMFinalK` во всех Go-файлах; добавление probe-семафора в `CircuitBreaker.CanExecute()` для состояния `CircuitHalfOpen`.
- Out of scope: изменение семантики `HybridConfig`, добавление новых полей в `CircuitBreaker`, изменение публичного API resilience-обёрток.

## Цель

Устранить две проблемы корректности: опечатку в публичном API (`BMFinaKK` вместо `BMFinalK`), которая создаёт путаницу и должна быть исправлена до появления внешних пользователей, и concurrency-баг в circuit breaker — при параллельных вызовах `CanExecute()` в состоянии `half-open` несколько горутин проходят одновременно как "probe", что обесценивает защиту от каскадных отказов.

## Основной сценарий

**Сценарий A — переименование поля:**
1. Разработчик использует `HybridConfig{BMFinalK: 10}` — имя поля читаемо и понятно.
2. Компилятор принимает старое имя `BMFinaKK` как ошибку → миграция принудительна.
3. Все внутренние обращения к полю переименованы; поведение не меняется.

**Сценарий B — circuit breaker probe:**
1. Circuit breaker переходит в `half-open` после таймаута.
2. Несколько горутин одновременно вызывают `CanExecute()`.
3. **До фикса:** все проходят как разрешённый probe → несколько запросов летят к нестабильному upstream.
4. **После фикса:** ровно одна горутина получает разрешение; остальные немедленно получают `ErrCircuitOpen`.

## Scope

- `internal/domain/models.go` — переименование поля и обновление godoc/комментариев
- `internal/domain/models_test.go` — обновление тестов под новое имя
- `internal/infrastructure/vectorstore/hybrid.go` — обновление обращения к полю
- `internal/infrastructure/vectorstore/hybrid_test.go` — обновление тестов
- `internal/infrastructure/vectorstore/hybrid_bench_test.go` — обновление бенчмарков
- `internal/infrastructure/resilience/circuitbreaker.go` — добавление probe-семафора в `CanExecute()`
- `internal/infrastructure/resilience/circuitbreaker_test.go` — добавление теста на параллельный half-open

## Контекст

- `BMFinaKK` присутствует в 5 Go-файлах; в архивных `.speckeep/`-файлах менять не нужно.
- `HybridConfig` — тип-алиас в `pkg/draftrag/draftrag.go` (`type HybridConfig = domain.HybridConfig`); переименование поля в `domain` автоматически распространяется на публичный API без дополнительных изменений.
- `CircuitBreaker` используется через `ResilienceEmbedder` и `ResilienceLLMProvider` в `internal/infrastructure/resilience/`; интерфейс `CanExecute()` — internal, не публичный API.
- Конституция требует `go vet`, `go fmt` без ошибок и unit-тесты для новых/изменённых функций.

## Требования

- RQ-001 Поле `HybridConfig.BMFinaKK` ДОЛЖНО быть переименовано в `BMFinalK` во всех Go-файлах репозитория; поведение Validate и DefaultHybridConfig ДОЛЖНО остаться идентичным.
- RQ-002 После переименования `go build ./...` ДОЛЖЕН завершаться без ошибок; старое имя `BMFinaKK` НЕ ДОЛЖНО присутствовать ни в одном Go-файле (кроме архивных `.speckeep/`).
- RQ-003 `CircuitBreaker.CanExecute()` в состоянии `CircuitHalfOpen` ДОЛЖЕН разрешать прохождение ровно одной горутине; все последующие параллельные вызовы ДОЛЖНЫ получать `ErrCircuitOpen` до завершения probe.
- RQ-004 После успешного или неудачного probe флаг ДОЛЖЕН сбрасываться так, чтобы следующий цикл `open → half-open` мог снова пропустить ровно одну probe-горутину.

## Вне scope

- Изменение семантики `HybridConfig` (значения по умолчанию, диапазоны валидации).
- Добавление новых полей или методов в `CircuitBreaker`.
- Изменение публичных интерфейсов `resilience`-обёрток (`ResilienceEmbedder`, `ResilienceLLMProvider`).
- Изменение поведения `CircuitClosed` и `CircuitOpen` состояний.
- Обновление архивных `.speckeep/` файлов.

## Критерии приемки

### AC-001 Переименование BMFinaKK → BMFinalK

- Почему это важно: опечатка в публичном API должна быть исправлена до появления внешних пользователей; после релиза это будет breaking change.
- **Given** репозиторий с полем `HybridConfig.BMFinaKK` в Go-файлах
- **When** выполнено переименование во всех Go-файлах и запущен `go build ./...`
- **Then** сборка проходит без ошибок; `grep -r "BMFinaKK" --include="*.go"` возвращает 0 совпадений
- Evidence: `go build ./...` — ok; grep по `*.go` — пусто.

### AC-002 Поведение HybridConfig не изменилось

- Почему это важно: переименование не должно менять логику — только имя поля.
- **Given** тесты `TestHybridConfig_Validate_*` и бенчмарки используют новое имя `BMFinalK`
- **When** запускается `go test ./internal/domain/... ./internal/infrastructure/vectorstore/...`
- **Then** все тесты проходят; поведение Validate и DefaultHybridConfig идентично предыдущему
- Evidence: `go test` — ok, без изменений в подсчёте результатов или ошибок валидации.

### AC-003 CanExecute в half-open пропускает ровно одну горутину

- Почему это важно: несколько probe-запросов к нестабильному upstream нарушают смысл half-open состояния.
- **Given** `CircuitBreaker` в состоянии `CircuitHalfOpen`; N горутин одновременно вызывают `CanExecute()`
- **When** все N горутин получают ответ
- **Then** ровно одна горутина получает `nil` (разрешение); остальные N-1 получают `ErrCircuitOpen`
- Evidence: unit-тест с `sync.WaitGroup` по N=10 горутин; счётчик разрешений == 1.

### AC-004 Probe-флаг сбрасывается после завершения probe

- Почему это важно: circuit breaker должен уметь восстанавливаться в следующих циклах.
- **Given** `CircuitBreaker` завершил probe (через `RecordSuccess` или `RecordFailure`)
- **When** circuit breaker снова переходит в `CircuitHalfOpen` (после нового `open`-периода)
- **Then** `CanExecute()` снова разрешает ровно одну горутину
- Evidence: тест с двумя последовательными `open → half-open` циклами; оба раза счётчик == 1.

## Допущения

- `HybridConfig` нигде не сериализуется (JSON/msgpack) в продакшн-коде; переименование поля не затронет сохранённые данные.
- Архивные `.speckeep/` файлы не компилируются Go-компилятором — их обновлять не нужно.
- Probe-семафор реализуется через atomic bool (`probeSent bool` + `sync/atomic` или simple `bool` под уже существующим `mu sync.RWMutex`) — новых зависимостей не нужно.
- `RecordSuccess` и `RecordFailure` уже берут `mu.Lock()`, что является подходящим местом для сброса probe-флага.

## Открытые вопросы

- none
