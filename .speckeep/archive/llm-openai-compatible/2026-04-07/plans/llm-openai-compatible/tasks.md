# LLMProvider OpenAI-compatible (Responses API) для draftRAG — Задачи

## Phase Contract

Inputs: `.draftspec/plans/llm-openai-compatible/plan.md`, `.draftspec/plans/llm-openai-compatible/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/errors.go | T1.1 |
| pkg/draftrag/openai_compatible_llm.go | T1.2, T2.1 |
| internal/infrastructure/llm/openai_compatible_responses.go | T2.2, T2.3 |
| internal/infrastructure/llm/openai_compatible_responses_test.go | T3.1 |
| pkg/draftrag/openai_compatible_llm_test.go | T3.2 |
| httptest.Server | T3.1, T3.2 |
| domain.LLMProvider | T2.2 |
| errors.Is | T1.1, T2.1, T3.2 |

## Фаза 1: Публичные ошибки и API каркас

Цель: зафиксировать публичный контракт ошибок и фабрику LLMProvider.

- [x] T1.1 Обновить `pkg/draftrag/errors.go` — добавить `ErrInvalidLLMConfig` (sentinel) для проверок через `errors.Is`. Touches: pkg/draftrag/errors.go
- [x] T1.2 Создать `pkg/draftrag/openai_compatible_llm.go` — options struct (BaseURL, APIKey, Model, Temperature, MaxOutputTokens, HTTPClient, Timeout) и фабрика `NewOpenAICompatibleLLM(opts) LLMProvider`; godoc на русском; ошибки конфигурации возвращаются из `Generate` через `ErrInvalidLLMConfig`. Touches: pkg/draftrag/openai_compatible_llm.go

## Фаза 2: Infrastructure реализация (Responses API)

Цель: реализовать HTTP вызов `/v1/responses`, parsing contract и безопасные ошибки.

- [x] T2.1 Реализовать `Generate(ctx, systemPrompt, userMessage)` в публичном провайдере: `ctx != nil` (panic), `userMessage != ""`, валидация конфигурации и параметров (`temperature >= 0`, `max_output_tokens > 0`), `errors.Is` для `ErrInvalidLLMConfig`, применение `Timeout` как `context.WithTimeout`. Touches: pkg/draftrag/openai_compatible_llm.go
- [x] T2.2 Создать `internal/infrastructure/llm/openai_compatible_responses.go` — реализация минимального контракта: `POST {BaseURL}/v1/responses` + `Authorization: Bearer {APIKey}` + request JSON (`model`, `input` system+user, `temperature`, `max_output_tokens`) + parsing (`output_text` или fallback через `output[].content[]`) + non-2xx ошибка (status + обрезанный body) с redaction APIKey. Touches: internal/infrastructure/llm/openai_compatible_responses.go
- [x] T2.3 Реализовать redaction: гарантировать, что `APIKey` не попадает в ошибки даже если сервер “эхом” возвращает ключ. Touches: internal/infrastructure/llm/openai_compatible_responses.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC через `httptest.Server`.

- [x] T3.1 Создать `internal/infrastructure/llm/openai_compatible_responses_test.go` — unit-тесты на `httptest.Server`: success через `output_text`, success через fallback `output[].content[]`, non-200 -> ошибка без утечек APIKey, invalid JSON/missing text -> ошибка, ctx cancel/deadline -> ошибка контекста не позже 100мс. Touches: internal/infrastructure/llm/openai_compatible_responses_test.go
- [x] T3.2 Создать `pkg/draftrag/openai_compatible_llm_test.go` — тесты публичного API: AC-001 (compile-time), AC-004 (errors.Is), AC-005 (redaction). Touches: pkg/draftrag/openai_compatible_llm_test.go

## Покрытие критериев приемки

- AC-001 -> T1.2, T3.2
- AC-002 -> T2.2, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T1.1, T1.2, T2.1, T3.2
- AC-005 -> T2.3, T3.1, T3.2

## Заметки

- Все тесты должны проходить без внешней сети и без реальных API ключей (используем `httptest.Server`).
- В v1 не добавляем стриминг/tool-calling; поддерживаем только синхронный `Generate`.
