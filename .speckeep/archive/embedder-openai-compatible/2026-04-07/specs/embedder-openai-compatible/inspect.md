---
report_type: inspect
slug: embedder-openai-compatible
status: pass
docs_language: ru
generated_at: 2026-04-07
---

# Inspect Report: embedder-openai-compatible

## Scope

- snapshot: проверка спецификации embedder-openai-compatible на соответствие конституции и качество acceptance criteria
- artifacts:
  - .draftspec/constitution.summary.md
  - .draftspec/specs/embedder-openai-compatible/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- В spec не зафиксирован точный формат авторизации и shape запроса/ответа (минимальный JSON контракт embeddings). В plan нужно выбрать поддерживаемый минимальный “OpenAI-compatible” формат (например, `POST /v1/embeddings` с полями `model`, `input`) и чётко описать mapping `data[0].embedding`.
- Требование RQ-007 предполагает стабильную sentinel-ошибку (`ErrInvalidEmbedderConfig`): в plan/tasks нужно явно определить, создаётся ли embedder с ошибкой (New... возвращает (Embedder, error)) или embedder создаётся всегда, а ошибка возвращается из `Embed`. Сейчас в spec допускается оба варианта — важно выбрать один и не оставлять двусмысленность.

## Questions

- Выбираем ли API-конструктор как `NewOpenAICompatibleEmbedder(opts) (Embedder, error)` (предпочтительно для ошибок конфигурации) или оставляем текущий паттерн без ошибки и валидируем на `Embed`?
- Нужно ли поддерживать custom `http.Client` (для прокси/трассировки) как часть options в v1, или достаточно timeout’а?

## Suggestions

- В `Контекст`/`Требования` явно добавить политику redaction: ошибки не должны включать `APIKey` и не должны логировать секреты.
- В `AC-002` добавить проверку, что embedding содержит только конечные значения (не NaN/Inf), чтобы не пускать мусор дальше в VectorStore.

## Traceability

- AC-001..AC-005: критерии приемки определены, Given/When/Then присутствуют, evidence наблюдаемы через unit-тесты на `httptest.Server`.

## Next Step

- safe to continue to plan
- Следующая команда: /draftspec.plan embedder-openai-compatible

