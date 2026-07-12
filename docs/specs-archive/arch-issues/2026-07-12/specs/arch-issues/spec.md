# Архитектурное hardening: PII, tool calling, роутинг, lifecycle

## Scope Snapshot

- In scope: четыре архитектурных дефекта draftRAG — PII redaction bypass, отсутствие LLM tool calling, дублирование handler-маппингов роутинга, отсутствие Health/Shutdown у Pipeline.
- Out of scope: новые VectorStore бэкенды, новые embedder/LLM провайдеры, CLI/HTTP-сервер, metrics export, изменение семантики индексации или retrieval.

## Цель

Разработчик, использующий draftRAG в production, получает:
1. PII-цензурирование, гарантированно работающее на всех entry points (не только через public API);
2. LLM-инструменты (tool/function calling) для построения Agentic RAG-пайплайнов;
3. Генерируемые handler-маппинги роутинга — без копипасты при добавлении нового типа запроса;
4. Pipeline с Health Check и Graceful Shutdown для production readiness.

Фича считается успешной, когда четыре перечисленных capability можно продемонстрировать тестами и встроенными примерами.

## Основной сценарий

1. Существующий Pipeline продолжает работать без изменений API (кроме расширения LLMProvider).
2. Разработчик создаёт Pipeline, конфигурирует PIIDetector — PII-фильтрация применяется на уровне application, независимо от того, через какой entry point вызван pipeline.
3. Разработчик создаёт LLMProvider с поддержкой tool calling — SearchBuilder может использовать инструменты для multi-step retrieval.
4. Разработчик добавляет новый тип роута (например, `HyDE + Filter`) — правка одного source-файла генерирует все handler-маппинги.
5. Разработчик вызывает `pipeline.Health(ctx)` — получает агрегированный статус store/llm/embedder.
6. Разработчик вызывает `pipeline.Close()` — pipeline освобождает ресурсы (HTTP-клиенты, кэш, горутины).

## User Stories

- P1 Story (PII): разработчик, использующий `internal/application.Pipeline` напрямую (Go-пакет может быть импортирован из internal), получает PII-цензурирование без дополнительных действий.
- P1 Story (Tool Calling): разработчик подключает LLM с tool calling и получает возможность использовать функции/инструменты в RAG-пайплайне.
- P2 Story (Router Gen): разработчик добавляет новый retrieval-режим — handler-маппинги генерируются автоматически.
- P2 Story (Health/Shutdown): разработчик может программно проверить доступность pipeline и корректно завершить его.

## MVP Slice

PII protection в application слое + Health() на Pipeline. Эти два изменения не требуют новых интерфейсов и ложатся на существующую архитектуру минимальными правками. Закрывает AC-001, AC-002, AC-007.

## First Deployable Outcome

Go-тест, демонстрирующий:
- PII-детектор, переданный в `PipelineOptions`, применяется в `internal/application.Pipeline.Index/Query/Answer` — не только в `pkg/draftrag`.
- Вызов `Pipeline.Health()` возвращает агрегированный статус трёх компонентов.
- `Pipeline.Close()` не паникует и не вызывает data race.

После первого pass — unit-тесты и `go test ./internal/application/... -run TestPII` проходят.

## Scope

1. **PII в application layer**: перенос вызова PIIDetector из `pkg/draftrag/draftrag.go` в `internal/application/pipeline.go` (методы `Index`, `Query`, `Answer`, `AnswerStream` и их варианты). Гарантия, что любой entry point применяет PII-фильтрацию.
2. **LLM tool calling**: расширение интерфейса `LLMProvider` опциональной capability `ToolCallingLLMProvider` с методом `GenerateWithTools`. Интеграция в SearchBuilder через `.Tools(tools)`. Поддержка streaming c tools.
3. **Router code generation**: замена ручных handler-маппингов (7 маршрутов × 7 output-режимов = 49 записей) на кодогенерацию из единой таблицы описания маршрутов. Генератор — `go generate` или внешний инструмент, запускаемый через `make generate-router`.
4. **Health и Shutdown**: добавление `Health(ctx) error` и `Close() error` на `Pipeline` (и `application.Pipeline`). Health агрегирует статус store/llm/embedder через их `Health(ctx)`. Close закрывает HTTP-клиенты embedder/LLM, останавливает фоновые горутины (rate-limiter, кэш).

## Контекст

- repository map: `internal/application/pipeline.go` (core orchestration), `pkg/draftrag/search_routing.go` (handler-маппинги), `internal/domain/interfaces.go` (LLMProvider), `pkg/draftrag/draftrag.go` (public Pipeline).
- В `pii.go` уже есть `PIIDetector`; `draftrag.go` применяет его, но `internal/application` нет.
- В `search_routing.go` уже 6 `nolint:dupl` блоков — дублирование признано техническим долгом.
- `LLMProvider` не расширялся с v0.1.0; tool calling — естественное следующее расширение.
- Pipeline не имеет lifecycle-методов; ресурсы (HTTP clients) управляются пользователем вне библиотеки.
- `Health()` уже есть на всех трёх core-интерфейсах, но не вызывается на уровне Pipeline.

## Зависимости

- **PII**: `internal/domain/pii.go` (интерфейс), `internal/infrastructure/piidetector/` (реализация). Никаких новых зависимостей.
- **Tool calling**: может потребоваться структура для описания tools (JSON Schema для параметров). Никаких внешних библиотек — только стандартная `encoding/json`.
- **Router gen**: Go 1.23+ `go generate` + скрипт или embedded генератор. Внешних зависимостей не вносить.
- **Health/Shutdown**: io.Closer интерфейс. Никаких новых внешних зависимостей.

## Требования

- RQ-001 Pipeline ДОЛЖЕН применять PIIDetector на уровне application (internal/application), а не только в public API (pkg/draftrag).
- RQ-002 Application-слой ДОЛЖЕН проверять наличие PIIDetector и применять его ко всем текстовым полям документов (Content), запросов и результатов retrieval.
- RQ-003 LLMProvider ДОЛЖЕН иметь опциональную capability ToolCallingLLMProvider с методом GenerateWithTools.
- RQ-004 SearchBuilder ДОЛЖЕН поддерживать per-request tools через метод .Tools(tools) — tools передаются в LLM при генерации.
- RQ-005 Handler-маппинги роутинга (search_routing.go) ДОЛЖНЫ генерироваться из единого описания маршрутов, а не поддерживаться вручную.
- RQ-006 Добавление нового route (например, `HyDE + Filter`) ДОЛЖНО требовать правки только в файле описания маршрутов и в реализации handler для нового route — без правки 7× map-маппингов для разных output-режимов.
- RQ-007 Pipeline.Health(ctx) error ДОЛЖЕН проверять доступность store, llm и embedder через их Health(ctx).
- RQ-008 Pipeline.Close() error ДОЛЖЕН освобождать явно управляемые ресурсы и останавливать фоновые горутины.

## Вне scope

- Инструменты для observability (metrics, tracing) — не добавляются, только исправляется существующий OTel hooks механизм.
- PII-детекция изображений/аудио — только текст.
- Tool calling для streaming LLM — P1 `.Stream(ctx)` с tools возвращает ErrToolsNotSupportedInStream; `.Answer(ctx)` и `.Cite(ctx)` с tools — основной P1 сценарий.
- Автоматическая генерация доки/комментариев для handler-маппингов.
- Health-эндпоинт HTTP — Pipeline остаётся библиотекой.
- Pipeline autoscaling, connection pooling — управляется пользователем.

## Критерии приемки

### AC-001 PII redaction bypass устранён

- Почему это важно: пользователь, импортирующий `internal/application.Pipeline`, не должен терять PII-защиту.
- **Given** Pipeline сконфигурирован с PIIDetector
- **When** пользователь вызывает `pipeline.Index(ctx, docs)` или `pipeline.Query(ctx, question)` напрямую через `internal/application.Pipeline`
- **Then** текстовые поля документов и результатов содержат отцензурированный PII
- Evidence: unit-тест создаёт `application.Pipeline` с mock PIIDetector и проверяет, что `Index` и `Query` вызывают `Detect()`.

### AC-002 Public API PII не дублирует application PII

- Почему это важно: чтобы не было двойного применения PII-детекции.
- **Given** Pipeline в `pkg/draftrag` сконфигурирован с PIIDetector
- **When** вызывается `pipeline.Index(ctx, docs)` из public API
- **Then** PII-детекция применяется ровно один раз (на уровне application)
- Evidence: тест с counter-обёрткой PIIDetector проверяет, что `Detect` вызван 1 раз на документ.

### AC-003 ToolCallingLLMProvider capability

- Почему это важно: LLM с tool calling — базовый паттерн Agentic RAG.
- **Given** LLMProvider, реализующий `ToolCallingLLMProvider`
- **When** вызывается `GenerateWithTools(ctx, systemPrompt, userMessage, tools)`
- **Then** возвращается структура, содержащая либо финальный текст ответа, либо список tool calls (имя + аргументы). Pipeline выполняет инструменты и повторно вызывает LLM с результатами для получения финального ответа.
- Evidence: mock-реализация ToolCallingLLMProvider в тестах возвращает предопределённый tool call; pipeline вызывает execution callback и возвращает финальный ответ от LLM после второго вызова.

### AC-004 Tools integration в SearchBuilder

- Почему это важно: пользователь может указать инструменты per-request через fluent API.
- **Given** SearchBuilder с `.Tools(tools)` и LLMProvider, поддерживающим tool calling
- **When** вызывается `.Answer(ctx)` или `.Cite(ctx)`
- **Then** LLM получает tools в запросе и может вернуть ответ с tool calls
- Evidence: integration test проверяет, что pipeline вызывает `GenerateWithTools` при наличии tools.

### AC-005 Handler-маппинги генерируются из единого описания

- Почему это важно: добавление нового route не должно требовать 7 правок в разных местах.
- **Given** единый файл описания маршрутов (например, `search_routes.yaml` или Go-таблица)
- **When** запускается `go generate ./pkg/draftrag/` (или `make generate-router`)
- **Then** все handler-маппинги (route → handler для каждого output-режима) помещены в `// Code generated` файл; рукописный `search_routing.go` не содержит handler-маппингов.
- Evidence: `go generate` + проверка, что `search_routing.go` не содержит map-литералов с handler-функциями; сгенерированный файл имеет маркер `// Code generated`.

### AC-006 Новый route добавляется без дублирования

- Почему это важно: developer experience при расширении роутинга.
- **Given** единое описание маршрутов
- **When** разработчик добавляет route `HyDE + Filter`
- **Then** после кодогенерации все 7 handler-маппингов содержат запись для нового route
- Evidence: тест проверяет, что после генерации map содержит ключ для `routeHyDEWithFilter` (или аналогичного) во всех router-переменных.

### AC-007 Pipeline.Health() возвращает агрегированный статус

- Почему это важно: production-системы должны проверять readiness.
- **Given** Pipeline с store, llm, embedder
- **When** вызывается `pipeline.Health(ctx)`
- **Then** возвращается nil, если все три компонента здоровы; error с деталями, если хотя бы один нездоров
- Evidence: unit-тест с mock store, возвращающим error, проверяет, что `Health()` возвращает не-nil ошибку.

### AC-008 Pipeline.Close() освобождает ресурсы

- Почему это важно: предотвращение утечки ресурсов при завершении приложения.
- **Given** Pipeline с запущенным rate-limiter'ом и HTTP-клиентами
- **When** вызывается `pipeline.Close()`
- **Then** все HTTP-клиенты помечены как closed; повторный вызов Health или операций возвращает ошибку (или no-op без паники)
- Evidence: тест проверяет, что после Close() вызов Health() возвращает ошибку "pipeline closed"; data race detector (-race) не находит проблем.

## Допущения

- PIIDetector stateless: не требует очистки/закрытия (если потребуется — добавит `io.Closer` в будущем).
- ToolCallingLLMProvider — optional capability, не ломает существующие LLMProvider.
- Router generator не требует внешних бинарных зависимостей; используется `go generate` + стандартная библиотека + `text/template`.
- Health(ctx) на store/llm/embedder — лёгкая операция (timeout < 1s).
- После Close() Pipeline переходит в терминальное состояние: повторные вызовы возвращают sentinel-ошибку, но не паникуют.

## Критерии успеха

- SC-001: `go test -race ./internal/application/... ./pkg/draftrag/...` без ошибок после всех изменений.
- SC-002: PII-тесты покрывают все entry points (Index, Query, Answer, Stream) на уровне application.
- SC-003: `go generate ./pkg/draftrag/` завершается за <1 секунды на среднем ноутбуке.

## Краевые случаи

- PIIDetector nil: Pipeline работает без PII-фильтрации (backward compat).
- LLMProvider без tool calling: `.Tools()` в SearchBuilder возвращает ошибку или no-op.
- Router generator: пустое описание маршрутов → пустые map'ы (не nil).
- `Health()` с nil store/llm/embedder: возвращает ошибку "component not initialized".
- `Close()` без Health: не паникует.
- Двойной `Close()`: второй вызов — no-op.
- Tool calling с streaming LLM: AC-004 не включает streaming tools (P2); `.Stream(ctx)` с tools возвращает ErrToolsNotSupportedInStream.
- Route `subDecompose + rewriter` вместе: приоритет остаётся за subDecompose (как сейчас).

## Открытые вопросы

1. Router generator: какой формат описания маршрутов? Варианты: (A) Go-таблица с рефлексией, (B) YAML-конфиг, (C) Go-annotations с кодогенерацией. Предпочтение — (A) как наименее ломающий и не требующий внешних зависимостей.
2. ToolCallingLLMProvider: модель данных для tools — `map[string]any` (JSON Schema) или строго типизированная структура? Предпочтение — свободная `json.RawMessage` для совместимости со всеми API.
3. После `Close()`: возвращать sentinel `ErrPipelineClosed` для всех операций или только документировать undefined behavior? Предпочтение — sentinel.
4. Нужен ли `io.Closer` на `VectorStore`/`Embedder`/`LLMProvider` для передачи управления ресурсами Pipeline'у? Если да — новый опциональный интерфейс `io.Closer`.
