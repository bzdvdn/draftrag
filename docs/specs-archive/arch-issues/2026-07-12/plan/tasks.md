# Архитектурное hardening — Задачи

## Phase Contract

Inputs: plan arch-issues, data-model.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех 8 AC.
Stop if: задачи расплывчаты — нет, все AC привязаны к конкретным поверхностям.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/models.go` | T1.1, T4.1 |
| `internal/domain/interfaces.go` | T4.2 |
| `internal/application/pipeline.go` | T2.1, T3.1, T4.3 |
| `internal/application/errors.go` (новый для sentinels) | T1.2 |
| `pkg/draftrag/draftrag.go` | T2.2, T3.2 |
| `pkg/draftrag/search.go` | T4.4 |
| `pkg/draftrag/search_routing.go` | T4.5, T5.1, T5.2 |
| `pkg/draftrag/errors.go` | T1.2 |
| `pkg/draftrag/routergen/` (новый) | T5.1 |
| `Makefile` | T5.3 |
| `internal/application/*_test.go` | T2.3, T3.3, T4.6, T6.1 |
| `pkg/draftrag/*_test.go` | T2.3, T3.3, T4.6, T6.1 |

## Implementation Context

- **Цель MVP**: Workstream 1 — PII redaction в application слое. AC-001, AC-002.
- **Инварианты/семантика**:
  - `application.Pipeline.piidetector` уже есть в `PipelineOptions` — только не используется в методах.
  - После переноса PII → application, `pkg/draftrag` НЕ вызывает `Detect()` — только application.
  - `Health()` fan-out: store → llm → embedder. `errors.Join` для агрегации.
  - `Close()`: `sync.Once`. Sentinel `ErrPipelineClosed`. После Close все методы (кроме Close) возвращают sentinel.
  - `ToolCallingLLMProvider` — optional capability (type assertion как StreamingLLMProvider).
  - Router gen: Go-таблица + `text/template` + `go generate`. Никаких внешних зависимостей.
  - `.Stream(ctx)` с tools → `ErrToolsNotSupportedInStream`.
- **Ошибки/коды**:
  - `ErrPipelineClosed` — `internal/application/`
  - `ErrToolsNotSupportedInStream` — `pkg/draftrag/errors.go`
- **Контракты/протокол**:
  - `ToolDefinition`, `ToolCall`, `ToolResult` в `internal/domain/models.go`
  - ToolCallingLLMProvider: `GenerateWithTools(ctx, systemPrompt, userMessage, tools []ToolDefinition) (string, []ToolCall, error)`
  - Execution callback выполняется в pipeline: ToolCall → ToolResult → повторный LLM с ToolResult.
- **Границы scope**:
  - Не меняем конкретные реализации LLM (ollama, openai, anthropic) — только интерфейс.
  - Не добавляем PII в streaming path — только Index/Query/Answer/Retrieve.
  - Router gen не меняет `pickRoute()` и handler-функции — только map-литералы.
- **References**: DEC-001 (PII), DEC-002 (ToolCallingLLMProvider), DEC-003 (json.RawMessage), DEC-004 (router gen), DEC-005 (Health), DEC-006 (Close), DM (ToolDefinition, ToolCall, ToolResult).

## Фаза 1: Основа (Data model)

Цель: подготовить типы и sentinel'ы, от которых зависят все workstream'ы.

- [x] T1.1 Добавить `ToolDefinition`, `ToolCall`, `ToolResult` в `internal/domain/models.go`.
  Touches: `internal/domain/models.go` (добавить импорт `encoding/json`, три новых типа)
- [x] T1.2 Добавить sentinel'ы: `ErrPipelineClosed` в `internal/application/`, `ErrToolsNotSupportedInStream` в `pkg/draftrag/errors.go`.
  Touches: `internal/application/pipeline.go` (var block), `pkg/draftrag/errors.go` (добавить sentinel)

## Фаза 2: MVP — PII в application слое

Цель: перенести вызов PIIDetector из public API в application слой. AC-001, AC-002.

- [x] T2.1 Добавить вызов `p.piidetector.Detect()` во все методы `application.Pipeline`, принимающие пользовательский текст: `Index`, `Query`, `Answer`, `Retrieve`, `UpdateDocument`. При `piidetector == nil` — no-op.
  Touches: `internal/application/pipeline.go` (Index, Query, processDocumentOp, Answer), `internal/application/retrieval.go` (Retrieve), `internal/application/query.go` (QueryWithQueries, QueryHyDE, и т.д.)
  - Импорт `strings` уже есть. `p.piidetector` уже есть. Нужно добавить вызов в начале каждого метода.
- [x] T2.2 Удалить дублирующие вызовы `p.piidetector.Detect()` из `pkg/draftrag/draftrag.go` (Index, Query, Answer, redactRetrievalResult). PII теперь только в application.
  Touches: `pkg/draftrag/draftrag.go` (Index, Query, Answer, UpdateDocument, redactRetrievalResult)
- [x] T2.3 Добавить unit-тесты: counter-обёртка PIIDetector проверяет ровно 1 вызов на метод через public API; application.Pipeline с mock PIIDetector проверяет вызов.
  Touches: `internal/application/pipeline_pii_test.go` (новый), `pkg/draftrag/draftrag_pii_test.go` (новый)

## Фаза 3: Health + Shutdown

Цель: Pipeline.Health() и Pipeline.Close(). AC-007, AC-008.

- [x] T3.1 Реализовать `Health(ctx) error` на `application.Pipeline`: fan-out к store, llm, embedder через их `Health(ctx)` с таймаутом 1s. Использовать `errors.Join` для агрегации. Реализовать `Close() error` с `sync.Once`, `closed` atomic флагом, sentinel `ErrPipelineClosed`.
  Touches: `internal/application/pipeline.go` (Pipeline struct — добавить `closeOnce`, `closed`; методы Health, Close; guard-проверка `p.closed` в начале Index/Query/Answer/UpdateDocument)
- [x] T3.2 Пробросить `Health()` и `Close()` в `pkg/draftrag.Pipeline` (фасадные методы). `Close()` освобождает только ресурсы pipeline (не store/llm/embedder — пользователь управляет ими).
  Touches: `pkg/draftrag/draftrag.go` (Pipeline struct — добавить `closeOnce`, `closed`; методы Health, Close)
- [x] T3.3 Добавить тесты: Health возвращает error при нездоровом store; Close → Health возвращает sentinel; `-race` не находит data race; двойной Close — no-op.
  Touches: `internal/application/pipeline_health_test.go` (новый), `internal/application/pipeline_close_test.go` (новый)

## Фаза 4: Tool calling

Цель: ToolCallingLLMProvider, SearchBuilder.Tools(), pipeline tool execution loop. AC-003, AC-004.

- [x] T4.1 Добавить `ToolCallingLLMProvider` optional interface в `internal/domain/interfaces.go`. Метод: `GenerateWithTools(ctx, systemPrompt, userMessage string, tools []ToolDefinition) (string, []ToolCall, error)`.
  Touches: `internal/domain/interfaces.go`, `internal/domain/models.go` (ToolDefinition, ToolCall уже в T1.1)
- [x] T4.2 Реализовать tool execution pipeline в `internal/application/pipeline.go`: type assertion `llm.(domain.ToolCallingLLMProvider)` в методе `Answer` / `Retrieve`, при наличии tools вызывать `GenerateWithTools`, выполнять `ToolCall` → `ToolResult`, повторно вызывать LLM с результатами. [`@sk-task arch-issues#T4.2: tool execution loop`]
  Touches: `internal/application/pipeline.go` (Answer / Retrieve — добавить tool execution loop)
- [x] T4.3 Добавить `routeTools` в `pickRoute()` в `search_routing.go`. Приоритет: после `routeHyDE` и `routeMultiQuery`, перед `routeHybrid`.
  Touches: `pkg/draftrag/search_routing.go` (константа `routeTools`, `pickRoute` case)
- [x] T4.4 Добавить `.Tools(tools []ToolDefinition)` на SearchBuilder + handler-функции для нового route (retrieve, answer, cite, inlineCite, streamSources). Stream возвращает `ErrToolsNotSupportedInStream`.
  Touches: `pkg/draftrag/search.go` (SearchBuilder — поле `tools`, метод `Tools`), `pkg/draftrag/search_routing.go` (toolsRetrieve, toolsAnswer, toolsCite, toolsInlineCite, toolsStreamSources, обновление map)
- [x] T4.5 Пока router gen не реализован (Фаза 5), вписать `routeTools` вручную во все 7 map-маппингов.
  Touches: `pkg/draftrag/search_routing.go` (7 map: retrieveHandlers, answerHandlers, citeHandlers, inlineCiteHandlers, streamHandlers, streamSourcesHandlers, streamCiteHandlers)
- [x] T4.6 Добавить тесты: mock ToolCallingLLMProvider возвращает tool call → pipeline выполняет execution → второй вызов LLM; SearchBuilder c `.Tools()` → GenerateWithTools вызван.
  Touches: `internal/application/pipeline_tool_test.go` (новый), `pkg/draftrag/search_tool_test.go` (новый)

## Фаза 5: Router code generation

Цель: заменить рукописные handler-маппинги на сгенерированные. AC-005, AC-006.

- [x] T5.1 Создать `pkg/draftrag/routergen/main.go` — генератор handler-маппингов. Читает Go-таблицу описания маршрутов, генерирует `search_routes_gen.go` с 7 map-литералами (retrieve, answer, cite, inlineCite, stream, streamSources, streamCite) с маркером `// Code generated`.
  Touches: `pkg/draftrag/routergen/main.go` (новый), `pkg/draftrag/routergen/routes.go` (новый — таблица описания маршрутов)
- [x] T5.2 Удалить рукописные map-литералы из `search_routing.go`. Оставить `pickRoute()` и handler-функции. Добавить `//go:generate go run ./routergen/` в `pkg/draftrag/doc.go` (или отдельный `generate.go`).
  Touches: `pkg/draftrag/search_routing.go` (удалить var retrieveHandlers, answerHandlers, citeHandlers, inlineCiteHandlers, streamHandlers, streamSourcesHandlers, streamCiteHandlers; удалить `nolint:dupl`), `pkg/draftrag/gen.go` (новый — `//go:generate` directive)
- [x] T5.3 Добавить `generate-router` target в Makefile: `go generate ./pkg/draftrag/...`.
  Touches: `Makefile`
- [x] T5.4 Добавить тест: генерация создаёт map с 8 route-ключами (включая newRoute); добавление нового route в таблицу + `go generate` = все 7 map обновлены.
  Touches: `pkg/draftrag/routergen/routergen_test.go` (новый), `pkg/draftrag/search_routes_gen_test.go` (новый)

## Фаза 6: Проверка

Цель: финальный прогон, lint, race detector, кодогенерация. Завершение.

- [x] T6.1 Запустить `go test -race -count=1 ./internal/application/... ./pkg/draftrag/...`, `go vet ./...`, `golangci-lint run`, `go generate ./pkg/draftrag/... && go build ./...`. Исправить все ошибки.
  Touches: все поверхности из Surface Map
- [x] T6.2 Обновить `REPOSITORY_MAP.md` (добавить `routergen/`), проверить README на актуальность.
  Touches: `REPOSITORY_MAP.md`

## Покрытие критериев приемки

- AC-001 -> T2.1, T2.3
- AC-002 -> T2.2, T2.3
- AC-003 -> T4.1, T4.2, T4.6
- AC-004 -> T4.3, T4.4, T4.5, T4.6
- AC-005 -> T5.1, T5.2, T5.4
- AC-006 -> T5.1, T5.4
- AC-007 -> T3.1, T3.3
- AC-008 -> T3.1, T3.2, T3.3

## Заметки

- T2.1 и T2.2 должны быть в одном коммите — удаление дублирования без переноса сломает PII.
- T4.5 (ручное добавление routeTools в 7 map) — временный шаг, будет заменён в Фазе 5. Без него AC-004 не работает до Фазы 5.
- T5.1 (routergen) можно делать параллельно с Фазой 4.
- T6.1 — последний шаг, перед ним все предыдущие фазы должны быть завершены.
- Все тесты должны проходить с `-race` — это обязательное условие AC-008.
