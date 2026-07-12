# Архитектурное hardening: PII, tool calling, роутинг, lifecycle — План

## Phase Contract

Inputs: spec arch-issues, inspect pass.
Outputs: plan.md, data-model.md.
Stop if: spec слишком расплывчата — все 4 workstream имеют явные AC и RQ.

## Цель

4 независимых изменения в draftRAG, каждое реализуется как отдельный инкремент (workstream) без ломки существующего API (кроме расширения LLMProvider опциональной capability). Изменения минимально пересекаются по surfaces, могут реализовываться в любом порядке кроме указанных зависимостей.

## MVP Slice

**MVP = Workstream 1 (PII)**: PII-детекция в application слое. Только этот workstream не требует новых интерфейсов и не ломает backward compat. Закрывает AC-001, AC-002.

## First Validation Path

```bash
go test -race -count=1 -run TestPII ./internal/application/... ./pkg/draftrag/...
```
Mock PIIDetector с counter — проверить что `Detect` вызывается на application уровне и не дублируется в public API.

## Scope

1. **PII application layer** (`internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`)
2. **LLM tool calling** (`internal/domain/interfaces.go`, `internal/application/pipeline.go`, `pkg/draftrag/search.go`, `pkg/draftrag/search_routing.go`)
3. **Router code generation** (`pkg/draftrag/search_routing.go`, новый `pkg/draftrag/routergen/` или `pkg/draftrag/search_routes_gen.go`, Makefile)
4. **Health/Shutdown** (`internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`, `pkg/draftrag/search.go`)

Не меняется: VectorStore реализации, embedder/LLM конкретные реализации (кроме нового опционального интерфейса), chunker, config.go, repo-map.

## Performance Budget

- PII: < 1µs overhead на документ (pattern-based, без HTTP).
- Health: < 100ms p99 (fan-out с таймаутом 1s).
- Router gen: < 500ms `go generate`.
- Tool calling: overhead только при наличии tools — один дополнительный LLM round-trip.
- `none` для остальных — изменения не затрагивают горячие пути индексации/поиска.

## Implementation Surfaces

| Surface | Что меняется | Workstream |
|---------|-------------|------------|
| `internal/application/pipeline.go` | Добавить вызов `p.piidetector.Detect()` в Index/Query/Answer/Stream методы; добавить `Health()` и `Close()` с `closed` флагом | PII, Health/Shutdown |
| `internal/application/pipeline.go: PipelineOptions` | (уже есть `PIIDetector`) — без изменений | PII |
| `pkg/draftrag/draftrag.go` | Убрать `piidetector.Detect()` из Index/Query/Answer — теперь только в application; пробросить `Health()`/`Close()` | PII, Health/Shutdown |
| `internal/domain/interfaces.go` | Новый `ToolCallingLLMProvider` optional interface | Tool calling |
| `pkg/draftrag/search.go` | Добавить `Tools()` на SearchBuilder; передавать tools через routing | Tool calling |
| `pkg/draftrag/search_routing.go` | Вынести handler-маппинги в generated файл; добавить `routeTools` | Router gen, Tool calling |
| `pkg/draftrag/routergen/` (новый) | Пакет-генератор: описание маршрутов + `go generate` | Router gen |
| `Makefile` | Добавить `generate-router` target | Router gen |
| `internal/infrastructure/llm/` | Обновить существующие LLM типы (no-op для тех, кто не поддерживает tools — ошибка "not supported", но для mock-тестов нужна реализация) | Tool calling |

## Bootstrapping Surfaces

- `pkg/draftrag/routergen/` — новая директория для генератора (только для Router gen workstream). Для остальных workstream'ов существующая структура репозитория достаточна.

## Влияние на архитектуру

- **PII**: Перенос логики → application слой получает новую ответственность. Public API слой теряет PII-логику (чище архитектура).
- **Tool calling**: Новый optional interface не ломает существующие реализации LLMProvider. SearchBuilder получает новый метод `.Tools()`. Routing расширяется новым route.
- **Router gen**: Из `search_routing.go` уходят map-литералы — они заменяются на `// Code generated` файл. `pickRoute()` и handler-функции остаются рукописными.
- **Health/Shutdown**: Pipeline получает lifecycle. Это требует `sync.Once` и sentinel-ошибки. `Close()` не освобождает store/llm/embedder (пользователь управляет ими), только внутренние ресурсы pipeline.

## Acceptance Approach

| AC | Workstream | Подход | Surfaces |
|----|-----------|--------|---------|
| AC-001 | PII | Тест: `application.Pipeline` с mock PIIDetector — `Detect` вызван | `internal/application/pipeline.go` |
| AC-002 | PII | Тест: public Pipeline с counter PIIDetector — `Detect` вызван 1 раз | `pkg/draftrag/draftrag.go`, `internal/application/pipeline.go` |
| AC-003 | Tool calling | Mock ToolCallingLLMProvider возвращает tool call; pipeline вызывает execution и повторный LLM | `internal/domain/interfaces.go`, `internal/application/pipeline.go` |
| AC-004 | Tool calling | SearchBuilder c `.Tools()` → pipeline вызывает `GenerateWithTools` | `pkg/draftrag/search.go`, `search_routing.go` |
| AC-005 | Router gen | `go generate` + проверка `// Code generated` маркера | `pkg/draftrag/routergen/`, `search_routing.go` |
| AC-006 | Router gen | Тест: добавить route, запустить `go generate`, проверить все 7 map | `pkg/draftrag/routergen/` |
| AC-007 | Health/Shutdown | Mock store с error → `Health()` возвращает error | `internal/application/pipeline.go` |
| AC-008 | Health/Shutdown | Тест: `Close()` → `Health()` возвращает sentinel; `-race` не находит проблем | `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go` |

## Данные и контракты

### Data model changes

- Поле `Tools` в `SearchBuilder` — не data model, а per-request параметр.
- `ToolCallingLLMProvider` — новый интерфейс, не тип данных.
- `ToolDefinition` / `ToolCall` — новые типы в `internal/domain/models.go` (JSON Schema-based).
- `ErrPipelineClosed` — новый sentinel в `internal/application/`.
- `ErrToolsNotSupportedInStream` — новый sentinel.

### Contract compatibility

- `ToolCallingLLMProvider` — optional capability: существующие LLMProvider не меняются.
- `search_routing.go` — old handlers удаляются из рукописного файла, но генерируются эквивалентные. Внешнего контракта нет.
- Pipeline API (Index/Query/Answer/Stream) — без изменений сигнатур (кроме нового метода `Health`/`Close`).

## Стратегия реализации

### DEC-001 PII перенос: удаление из public API, добавление в application

- **Why**: Только application слой гарантирует, что PII-фильтрация применяется на всех entry points. Public API переиспользует application и не должен дублировать.
- **Tradeoff**: application.Pipeline получает новую ответственность; при `piidetector == nil` — zero overhead (nil check).
- **Affects**: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
- **Validation**: AC-001, AC-002

### DEC-002 ToolCallingLLMProvider: optional interface с GenerateWithTools

- **Why**: Optional capability pattern уже используется в коде (StreamingLLMProvider, UsageAwareLLMProvider). Это консистентно. Не ломает существующие LLMProvider.
- **Tradeoff**: Pipeline должен делать type assertion на каждый вызов Generate. Это один type assertion — overhead < 1ns.
- **Affects**: `internal/domain/interfaces.go`, `internal/application/pipeline.go`
- **Validation**: AC-003, AC-004

### DEC-003 Tool model: `json.RawMessage` для аргументов

- **Why**: OpenAI, Anthropic и Mistral используют разные форматы tool definition. `json.RawMessage` позволяет передавать сырой JSON без парсинга со стороны библиотеки.
- **Tradeoff**: Пользователь должен валидировать JSON самостоятельно. Нет typed safe API. В спецификации ToolDefinition можно использовать `any` для аргументов, но на wire — `json.RawMessage`.
- **Affects**: `internal/domain/models.go`
- **Validation**: AC-003 (mock тест)

### DEC-004 Router generator: Go-таблица + text/template + go generate

- **Why**: Вариант (A) из открытых вопросов — не требует внешних зависимостей, не требует YAML-схемы, lint-friendly (`// Code Generated`).
- **Tradeoff**: Генератор — отдельный `main.go` в `routergen/`. Нужен `go run` на этапе генерации. Не самый быстрый подход, но < 500ms.
- **Affects**: `pkg/draftrag/routergen/`, `pkg/draftrag/search_routing.go`, Makefile
- **Validation**: AC-005, AC-006

### DEC-005 Health fan-out с таймаутом

- **Why**: Health — fan-out к трём компонентам. Если pgvector/timeout — не ждать > 1s. Каждый компонент проверяется независимо.
- **Tradeoff**: Ошибка одного компонента не маскирует другие. Для агрегации ошибок — `errors.Join`.
- **Affects**: `internal/application/pipeline.go`
- **Validation**: AC-007

### DEC-006 Close sync.Once + closed флаг + sentinel

- **Why**: `sync.Once` гарантирует одноразовое закрытие. `closed` флаг с atomic проверкой во всех операциях. Sentinel `ErrPipelineClosed` возвращается из Health и операций после Close.
- **Tradeoff**: Каждая операция получает дополнительный `atomic.Load` чтение. Overhead < 1ns.
- **Affects**: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
- **Validation**: AC-008

## Incremental Delivery

### MVP — Workstream 1: PII application level

- Задачи: перенос вызова PIIDetector в application слой, удаление из public API, тесты.
- AC: AC-001, AC-002
- Проверка: `go test -run TestPII`

### Итеративное расширение (порядок независимый)

#### Workstream 2: Health + Shutdown
- Задачи: Health(), Close(), sentinel, тесты.
- AC: AC-007, AC-008
- Зависимость: желательно после PII (чтобы не мешать тесты).
- Проверка: `go test -run TestHealth -race`

#### Workstream 3: Tool calling
- Задачи: ToolCallingLLMProvider, tool model, SearchBuilder.Tools(), routing, тесты.
- AC: AC-003, AC-004
- Проверка: `go test -run TestTool`

#### Workstream 4: Router code generation
- Задачи: generator в routergen/, go generate, Makefile, тесты.
- AC: AC-005, AC-006
- Проверка: `go generate ./pkg/draftrag/ && go test ./pkg/draftrag/...`

## Порядок реализации

1. **Workstream 1 (PII)** — самый безопасный, без новых интерфейсов, без рисков. Можно начинать немедленно.
2. **Workstream 2 (Health/Shutdown)** — orthogonal, можно параллельно с PII.
3. **Workstream 3 (Tool calling)** — после PII, чтобы не создавать конфликты при merge. Можно параллельно с Workstream 2.
4. **Workstream 4 (Router gen)** — можно в любой момент; не зависит от других. Рекомендуется после PII.

Параллельно безопасно: Workstream 1 + Workstream 2; Workstream 3 + Workstream 4.

## Риски

- **Риск 1**: Tool calling интерфейс не совместим с форматом Anthropic/Mistral/OpenAI tools.
  *Mitigation*: `json.RawMessage` для аргументов — универсально. Разные провайдеры реализуют кастинг в своих конкретных LLM реализациях.
- **Риск 2**: Router generator добавляет сложность в сборку (go generate — ещё один шаг).
  *Mitigation*: `go generate ./...` уже используется в проекте (для migrations). Добавить в CI и Makefile.
- **Риск 3**: Close() может быть вызван во время Health() — data race.
  *Mitigation*: `sync.Mutex` на `closed` + проверка в начале Health(). Безопасно через `sync.Once`.
- **Риск 4**: PII в application добавляет вызов Detector на каждую операцию — пользователь мог рассчитывать на zero overhead.
  *Mitigation*: nil check — при piidetector == zero overhead. Документировано в spec.

## Rollout и compatibility

- **PII**: Backward compatible — пользователи без PIIDetector не видят изменений. С PIIDetector — поведение не меняется (только точка вызова).
- **Tool calling**: Новый optional interface. Существующий код без tools не меняется.
- **Router gen**: Генерация — новый шаг в сборке. Но `search_routing.go` остаётся (без handler-маппингов). `go vet`/`go build` не замечают разницы.
- **Health/Shutdown**: Новые методы. Backward compatible.
- Специальных rollout-действий не требуется.

## Проверка

- `go test -race -count=1 ./internal/application/... ./pkg/draftrag/...`
- `go vet ./...`
- `golangci-lint run`
- `go generate ./pkg/draftrag/... && go build ./...`
- Для AC-008: `go test -race -run TestPipelineClose`

## Соответствие конституции

- нет конфликтов. Все изменения следуют Clean Architecture (domain → application → infrastructure). ToolCallingLLMProvider — optional capability, совместим с принципом интерфейсной абстракции. PII перенос усиливает архитектурную чистоту (PII — ответственность application слоя).
