---
report_type: inspect
slug: retrieval-reranker-mmr
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Inspect Report: retrieval-reranker-mmr

## Scope

- snapshot: проверена спецификация MMR rerank на аддитивность, детерминизм и отсутствие сетевых зависимостей
- artifacts:
  - .draftspec/constitution.md
  - .draftspec/specs/retrieval-reranker-mmr/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- Важно явно зафиксировать ограничение v1: MMR применим только если embeddings доступны в retrieval результате (в `Chunk.Embedding`). Если нет — MMR должен быть выключен или должен возвращать явную ошибку конфигурации.

## Questions

- Какой surface выбрать для конфигурации: расширяем `PipelineOptions` (public) или только application config? (предложение: public опция в `PipelineOptions`, по аналогии с другими настройками pipeline)

## Suggestions

- Делать MMR как selection поверх `topKCandidates` (candidate pool), выбирая `topK` для prompt.
- Тестировать на синтетических embeddings, чтобы детерминированно получить “кластеры”.

## Traceability

- AC-001: unit-тест фиксирует, что при включении MMR выбранные чанки покрывают разные “кластеры” (не только самые похожие друг на друга).
- AC-002: при выключенном MMR порядок и выбор совпадают с текущим поведением (score desc).

## Next Step

- safe to continue to plan

