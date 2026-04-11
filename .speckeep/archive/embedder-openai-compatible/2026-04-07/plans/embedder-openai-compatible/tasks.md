# Embedder OpenAI-compatible для draftRAG — Задачи

## Phase Contract

Inputs: `.draftspec/plans/embedder-openai-compatible/plan.md`, `.draftspec/plans/embedder-openai-compatible/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/errors.go | T1.1 |
| pkg/draftrag/openai_compatible_embedder.go | T1.2, T2.1 |
| internal/infrastructure/embedder/openai_compatible.go | T2.2, T2.3 |
| internal/infrastructure/embedder/openai_compatible_test.go | T3.1 |
| pkg/draftrag/openai_compatible_embedder_test.go | T3.2 |
| httptest.Server | T3.1, T3.2 |

## Фаза 1: Публичные ошибки и API каркас

Цель: зафиксировать публичный контракт ошибок и фабрику embedder’а.

- [x] T1.1 Обновить `pkg/draftrag/errors.go` — добавить `ErrInvalidEmbedderConfig` (sentinel) и при необходимости `ErrEmbedderRequestFailed`/`ErrEmbedderInvalidResponse` (не обязательно), чтобы тесты могли использовать `errors.Is`. Touches: pkg/draftrag/errors.go
- [x] T1.2 Создать `pkg/draftrag/openai_compatible_embedder.go` — определить options struct (BaseURL, APIKey, Model, HTTPClient, Timeout) и фабрику `NewOpenAICompatibleEmbedder(opts) Embedder`; godoc на русском; фабрика возвращает объект, ошибки конфигурации возвращаются из `Embed` через `ErrInvalidEmbedderConfig`. Touches: pkg/draftrag/openai_compatible_embedder.go

## Фаза 2: Инфраструктурная реализация (HTTP + JSON)

Цель: реализовать корректный HTTP вызов `/v1/embeddings`, парсинг JSON и обработку ошибок/контекста.

- [x] T2.1 В публичном embedder’е реализовать метод `Embed(ctx, text)` (через внутреннюю реализацию) с валидацией: `ctx != nil` (panic), `text != ""`, валидная конфигурация; при ошибке конфигурации возвращать `ErrInvalidEmbedderConfig` (errors.Is). Touches: pkg/draftrag/openai_compatible_embedder.go
- [x] T2.2 Создать `internal/infrastructure/embedder/openai_compatible.go` — реализовать минимальный OpenAI-compatible контракт: `POST {BaseURL}/v1/embeddings` + `Authorization: Bearer {APIKey}` + request JSON `{model,input}` + response JSON `data[0].embedding`; non-2xx -> ошибка со status code + обрезанный body (без секретов); embedding только с конечными float значениями (не NaN/Inf). Touches: internal/infrastructure/embedder/openai_compatible.go
- [x] T2.3 Реализовать redaction: гарантировать, что `APIKey` не попадает в ошибки даже при “echo” сервера. Touches: internal/infrastructure/embedder/openai_compatible.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC через `httptest.Server` и e2e через `Pipeline` без внешних зависимостей.

- [x] T3.1 Создать `internal/infrastructure/embedder/openai_compatible_test.go` — unit-тесты на `httptest.Server`: success (валидный JSON -> embedding), non-200 (ошибка без утечек APIKey), invalid JSON/missing embedding (ошибка), ctx cancel/deadline (ошибка контекста не позже 100мс). Touches: internal/infrastructure/embedder/openai_compatible_test.go
  - success: валидный JSON -> embedding []float64
  - non-200: возвращает ошибку (без утечек APIKey)
  - invalid JSON / missing embedding -> ошибка
  - ctx cancel/deadline -> `context.Canceled`/`context.DeadlineExceeded` не позже 100мс
- [x] T3.2 Создать `pkg/draftrag/openai_compatible_embedder_test.go` — тесты публичного API: AC-001 (фабрика возвращает `Embedder`), AC-004 (errors.Is для `ErrInvalidEmbedderConfig`), AC-005 (e2e Pipeline.Index + Pipeline.QueryTopK с embedder’ом на `httptest.Server`). Touches: pkg/draftrag/openai_compatible_embedder_test.go
  - AC-001: фабрика возвращает `Embedder`
  - AC-004: errors.Is для `ErrInvalidEmbedderConfig`
  - AC-005: e2e `Pipeline.Index` + `Pipeline.QueryTopK` (in-memory store + мок LLM) с embedder’ом на `httptest.Server`

## Покрытие критериев приемки

- AC-001 -> T1.2, T3.2
- AC-002 -> T2.2, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T1.1, T1.2, T2.1, T3.2
- AC-005 -> T3.2

## Заметки

- Все тесты должны проходить без внешней сети и без реальных API ключей (используем `httptest.Server`).
- В v1 поддерживаем только `input: string`; batch может быть добавлен позже без публичного API изменения (как внутренняя оптимизация).
